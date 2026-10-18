package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func installedSkillsRoot() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(filepath.Dir(executable))
}
func rejectRedirect(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
func fixedClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	copy := *client
	copy.CheckRedirect = rejectRedirect
	return &copy
}

// Public IDs are opaque: model-labelled IDs contain dots; legacy outputs contain
// colons. Keep them in one path segment and exclude leading dot segments.
var taskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._:-]{0,255}$`)

func validTaskID(id string) bool { return taskIDPattern.MatchString(id) }
func safePublicCode(code string) string {
	if !publicCodePattern.MatchString(code) {
		return ""
	}
	return safePublicString(code)
}
func taskReceipt(request taskRequest, id, status string) receipt {
	parameters := make(map[string]any)
	// Media URLs, prompts and file paths are deliberately excluded from receipts.
	for _, key := range []string{"n", "size", "image_size", "aspect_ratio", "width", "height", "duration", "resolution", "generate_audio", "instrumental", "strength", "quality", "output_format", "response_format"} {
		if value, exists := request.Parameters[key]; exists {
			switch v := value.(type) {
			case string:
				if safePublicString(v) == "" {
					continue
				}
			case float64, int, bool:
			default:
				continue
			}
			if key == "n" || key == "width" || key == "height" || key == "duration" {
				var number float64
				switch v := value.(type) {
				case float64:
					number = v
				case int:
					number = float64(v)
				default:
					continue
				}
				if math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
					continue
				}
				if key == "n" && (number > maxImageCount || math.Trunc(number) != number) {
					continue
				}
			}
			if key == "generate_audio" || key == "instrumental" {
				if _, ok := value.(bool); !ok {
					continue
				}
			}
			if key == "size" || key == "image_size" || key == "aspect_ratio" || key == "resolution" || key == "strength" || key == "quality" || key == "output_format" || key == "response_format" {
				if text, ok := value.(string); !ok || !safeParameterString(key, text) {
					continue
				}
			}
			parameters[key] = value
		}
	}
	original := request.OriginalOperation
	if original == "" && request.Operation != "continue" {
		original = request.Operation
		if request.MediaOperation != "" {
			original = request.MediaOperation
		}
	}
	if safePublicCode(original) == "" {
		original = ""
	}
	model := ""
	if validModelID(request.Model) {
		model = request.Model
	}
	count := request.RequestedCount
	if count < 0 || count > maxImageCount {
		count = 0
	}
	kind := request.Kind
	if kind != "image" && kind != "video" && kind != "music" {
		kind = ""
	}
	operation := safePublicCode(request.Operation)
	if status != "" {
		status = continuationStatus(status)
	}
	retryAt := request.RetryNotBefore
	if _, err := time.Parse(time.RFC3339, retryAt); err != nil {
		retryAt = ""
	}
	return receipt{OK: true, Kind: kind, Operation: operation, OriginalOperation: original, Model: model, TaskID: id, Status: status, RequestedCount: count, Parameters: parameters, RetryNotBefore: retryAt, ReconciliationRequired: request.ReconciliationRequired, WaitWindowsCompleted: min(2, max(0, request.WaitWindowsCompleted))}
}

// A safe field name alone does not make arbitrary request text safe to persist.
// Keep a conservative projection even for malformed requests and imported records.
func safeParameterString(key, text string) bool {
	patterns := map[string]string{
		"size":            `^(auto|[1-9][0-9]{0,4}x[1-9][0-9]{0,4})$`,
		"image_size":      `^[1-9][0-9]?[Kk]$`,
		"aspect_ratio":    `^(auto|adaptive|[1-9][0-9]{0,2}:[1-9][0-9]{0,2})$`,
		"resolution":      `^[1-9][0-9]{0,3}[pPkK]$`,
		"strength":        `^(LOW|MID|HIGH)$`,
		"quality":         `^(auto|low|medium|high|xhigh|max)$`,
		"output_format":   `^(png|jpeg|webp)$`,
		"response_format": `^(url|mp3|wav)$`,
	}
	pattern, ok := patterns[key]
	return ok && regexp.MustCompile(pattern).MatchString(text)
}
func mergeFailure(current, failure receipt) receipt {
	current.OK = false
	current.FailurePhase = failure.FailurePhase
	current.HTTPStatus = failure.HTTPStatus
	current.APIErrorCode = failure.APIErrorCode
	current.ErrorMessage = failure.ErrorMessage
	current.NextAction = failure.NextAction
	current.RetryAfterSecs = failure.RetryAfterSecs
	current.LocalErrorCode = failure.LocalErrorCode
	return current
}
func applyTaskStatus(result receipt, body []byte, httpStatus int) (receipt, error) {
	id, state, reconcile, code := parseTaskIdentity(body)
	if code != "" {
		return mergeFailure(result, taskIdentityFailure("status", code, httpStatus)), errors.New("task identity unreadable")
	}
	if id != result.TaskID {
		return mergeFailure(result, receipt{FailurePhase: "status", HTTPStatus: httpStatus, ErrorMessage: "The status response did not identify this task.", NextAction: "Keep this task ID; do not create another task."}), errors.New("task identity mismatch")
	}
	result.Status = state
	result.ReconciliationRequired = reconcile
	if reconcile {
		result.OK = false
		result.FailurePhase = "status"
		result.ErrorMessage = "The task requires server-side reconciliation."
		result.NextAction = "Keep this task ID and check the same task later; do not infer a refund or submit a replacement."
		return result, errors.New("reconciliation required")
	}
	if terminalFailure(state) {
		result.taskFailureConfirmed = true
		code, message := publicAPIError(body)
		if message == "" {
			message = "The API reported that this task failed."
		}
		return mergeFailure(result, apiFailure("status", httpStatus, 0, code, message, "Review this task's error. Do not retry or change models automatically.")), errors.New("task failed")
	}
	if !terminalSuccess(state) && state != "pending" && state != "queued" && state != "processing" && state != "running" && state != "in_progress" {
		result.Status = "unknown"
		return mergeFailure(result, receipt{FailurePhase: "status", ErrorMessage: "The API returned an unsupported task state.", NextAction: "Keep this task ID and check its status later; do not resubmit."}), errors.New("unknown task state")
	}
	return result, nil
}
func executeExistingTask(command string, input io.Reader, output io.Writer, svc service) error {
	request, err := decodeTaskRequest(input)
	if err != nil {
		writeReceipt(output, validationFailure("A complete same-task request JSON is required."))
		return err
	}
	request.Operation = "continue"
	result := taskReceipt(request, request.TaskID, continuationStatus(request.TaskStatus))
	if err = validateTaskRequest(request); err != nil || !validTaskID(request.TaskID) {
		if !validTaskID(request.TaskID) {
			result.TaskID = ""
		}
		writeReceipt(output, mergeFailure(result, validationFailure("Choose image, video or music, a valid existing task ID, and valid continuation metadata.")))
		return errors.New("invalid existing task")
	}
	if command == "wait" {
		return pollAndDeliver(output, svc, request, result)
	}
	if retryDelay(result.RetryNotBefore, svc.clock()) > 0 {
		result.OK = false
		result.FailurePhase = "status"
		result.LocalErrorCode = "retry_wait_required"
		result.ErrorMessage = "The next same-task read is not due yet."
		result.NextAction = "Wait until retry_not_before before reading this task again; do not resubmit."
		writeReceipt(output, result)
		return errors.New("retry wait required")
	}
	if command == "content" {
		// The caller must supply the completed state from the last same-task receipt.
		// The content endpoint itself is authoritative and rejects unfinished tasks.
		if !terminalSuccess(request.TaskStatus) || request.ReconciliationRequired {
			writeReceipt(output, mergeFailure(result, validationFailure("Read the same task status first; content requires task_status completed.")))
			return errors.New("completion not confirmed")
		}
		result.Status = "completed"
		return deliverTask(output, svc, request, result)
	}
	body, status, retry, code, message, err := svc.request(context.Background(), http.MethodGet, statusPath(request.Kind, result.TaskID), nil, "")
	if err != nil || status < 200 || status >= 300 {
		failure := apiFailure("status", status, retry, code, message, "Keep this task ID; continue the same task later.")
		if status >= 200 && status < 300 && errors.Is(err, errAPIResponseUnreadable) {
			failure = taskIdentityFailure("status", "task_response_unreadable", status)
		}
		writeReceipt(output, withRetry(mergeFailure(result, failure), retry, svc.clock()))
		return errors.New("status failed")
	}
	result, err = applyTaskStatus(result, body, status)
	result = withRetry(result, retry, svc.clock())
	if err == nil {
		result.NextAction = "Continue this same task if pending; use content if completed. Never submit it again."
	}
	writeReceipt(output, result)
	return err
}

func readAPIObject(body []byte) (map[string]any, error) {
	var result map[string]any
	if json.Unmarshal(body, &result) != nil || result == nil {
		return nil, errors.New("invalid response")
	}
	if result["error"] != nil || result["success"] == false {
		return nil, errors.New("API logical error")
	}
	return result, nil
}
func projectCatalog(value any) any {
	document := jsonObject(value)
	projected := []any{}
	for _, entry := range jsonArray(document["data"]) {
		model := jsonObject(entry)
		id := jsonString(model["id"])
		if !validModelID(id) {
			continue
		}
		result := map[string]any{"id": id}
		caps := []string{}
		for _, cap := range jsonArray(model["capabilities"]) {
			if cap == "image" || cap == "video" {
				caps = append(caps, cap.(string))
			}
		}
		result["capabilities"] = caps
		schema := reviewedMediaSchema(id, jsonObject(model["input_schema"]))
		selected := map[string]any{}
		for _, field := range []string{"properties", "constraints", "operations", "lifecycle", "outputSizes"} {
			if v, ok := schema[field]; ok {
				selected[field] = publicJSON(v, 0)
			}
		}
		result["input_schema"] = selected
		projected = append(projected, result)
	}
	return map[string]any{"data": projected}
}

// Catalog JSON is data, never instructions or a channel for raw error/config data.
func publicJSON(value any, depth int) any {
	if depth > 16 {
		return nil
	}
	switch v := value.(type) {
	case string:
		return safePublicString(v)
	case []any:
		result := make([]any, 0, len(v))
		for _, item := range v {
			result = append(result, publicJSON(item, depth+1))
		}
		return result
	case map[string]any:
		result := make(map[string]any)
		for key, item := range v {
			if !publicCodePattern.MatchString(key) {
				continue
			}
			switch strings.ToLower(key) {
			case "apikey", "api_key", "token", "secret", "authorization", "headers", "provider", "upstream":
				continue
			}
			result[key] = publicJSON(item, depth+1)
		}
		return result
	case float64, bool, nil:
		return v
	default:
		return nil
	}
}
