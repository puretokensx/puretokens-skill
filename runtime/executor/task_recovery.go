package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func (svc service) clock() time.Time {
	if svc.now != nil {
		return svc.now()
	}
	return time.Now()
}

func (svc service) waitFor(ctx context.Context, delay time.Duration) bool {
	finishTiming := svc.timePhase("wait")
	defer finishTiming()
	if svc.wait != nil {
		return svc.wait(ctx, delay)
	}
	return waitContext(ctx, delay)
}

func (svc service) contentTimeout() time.Duration {
	if svc.downloadTimeout > 0 {
		return svc.downloadTimeout
	}
	return 10 * time.Minute
}

func retryDelay(value string, now time.Time) time.Duration {
	at, err := time.Parse(time.RFC3339, value)
	if err != nil || !at.After(now) {
		return 0
	}
	return at.Sub(now)
}

func withRetry(result receipt, seconds int, now time.Time) receipt {
	result.RetryAfterSecs = seconds
	result.RetryNotBefore = ""
	if seconds > 0 {
		// Saturate at the largest RFC3339 year without duration overflow.
		unix := int64(253402300799)
		if int64(seconds) <= unix-now.Unix() {
			unix = now.Unix() + int64(seconds)
			if now.Nanosecond() != 0 && unix < 253402300799 {
				unix++
			}
		}
		result.RetryNotBefore = time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	return result
}

func executePreflight(output io.Writer, svc service, request taskRequest) error {
	if request.Operation == "continue" {
		writeReceipt(output, validationFailure("Use status or resume for an existing task. Preflight only validates a proposed new request."))
		return errors.New("preflight cannot continue")
	}
	err := validateTaskRequest(request)
	if err == nil {
		err = prepareProfileRequest(&request, svc)
	}
	result := taskReceipt(request, "", "")
	result.Operation = "preflight"
	result.SubmissionOutcome = "not_submitted"
	if err != nil {
		message := safePublicString(err.Error())
		if message == "" {
			message = "The proposed request is not supported by the selected model profile."
		}
		result = mergeFailure(result, validationFailure(message))
	} else {
		result.NextAction = "The declared parameters are valid; no task was submitted. This is not a price, balance, permission or generation-success guarantee."
	}
	writeReceipt(output, result)
	return err
}

func continuationStatus(status string) string {
	switch status {
	case "completed", "succeeded", "success", "failed", "cancelled", "canceled", "expired", "error", "pending", "queued", "processing", "running", "in_progress", "unknown":
		return status
	default:
		return "pending"
	}
}

func clearTaskFailure(result receipt) receipt {
	result.OK = true
	result.FailurePhase = ""
	result.HTTPStatus = 0
	result.LocalErrorCode = ""
	result.APIErrorCode = ""
	result.FailureCategory = ""
	result.FailureSource = ""
	result.recordedFailure = nil
	result.taskFailureConfirmed = false
	result.ErrorMessage = ""
	result.NextAction = ""
	return result
}

func finishKnownTask(output io.Writer, result receipt) error {
	if result.ReconciliationRequired {
		result.OK = false
		result.FailurePhase = "status"
		result.ErrorMessage = "The task requires server-side reconciliation before further automatic work."
		result.NextAction = "This task needs reconciliation. Keep its ID and explicitly check status later; do not submit a replacement."
	} else if terminalFailure(result.Status) {
		result.OK = false
		result.FailurePhase = "status"
		result.FailureCategory = "unknown"
		result.ErrorMessage = "The recorded task has ended unsuccessfully; this record does not identify a more specific cause."
		result.NextAction = "This task has ended unsuccessfully. Its cause is unknown; recover the original failure receipt or contact support before proposing a model or group change. Do not automatically submit a replacement."
		if failure := result.recordedFailure; failure != nil && validTaskFailure(failure, result.Status) {
			result.FailureCategory = failure.Category
			result.FailurePhase = failure.Phase
			result.APIErrorCode = failure.APIErrorCode
			if failure.Category != "unknown" {
				result.ErrorMessage = "The saved task result identifies the failure category. This is historical task evidence, not a new API check."
			}
		}
	} else {
		result.NextAction = "This task is complete. Retrieve only missing content indexes and attach downloaded files."
	}
	writeReceipt(output, result)
	if !result.OK {
		return errors.New("task cannot be resumed")
	}
	return nil
}

var (
	errInvalidMedia  = errors.New("invalid media content")
	errMediaTooLarge = errors.New("media exceeds size limit")
)

func contentFailure(err error, status, retry int, code, message string) receipt {
	result := apiFailure("content", status, retry, code, message, "Keep this task ID. Retrieve only the missing index; do not submit another task.")
	if status >= 300 {
		return result
	}
	result.LocalErrorCode = "media_download_incomplete"
	result.ErrorMessage = "The media download did not finish or could not be validated."
	result.NextAction = "Keep this task ID. Check network access and free disk space, then retrieve this same index again."
	var network net.Error
	switch {
	case errors.Is(err, errMediaTooLarge):
		result.LocalErrorCode = "media_size_limit_exceeded"
		result.ErrorMessage = "The content response exceeds the supported media size limit."
		result.NextAction = "Keep the task ID and contact Pure Tokens support about this output size; do not repeat the submission."
	case errors.Is(err, errInvalidMedia):
		result.LocalErrorCode = "invalid_media_content"
		result.ErrorMessage = "The content response is incomplete or is not a supported native media file."
		result.NextAction = "Keep the task ID. Retrieve this same index only when requested; if the content remains invalid, contact Pure Tokens support."
	case errors.Is(err, os.ErrPermission):
		result.LocalErrorCode = "output_permission_denied"
		result.ErrorMessage = "The selected output directory is not writable."
		result.NextAction = "Choose a writable output directory and retrieve this same task index."
	case errors.As(err, &network) && network.Timeout():
		result.LocalErrorCode = "media_download_timeout"
		result.ErrorMessage = "The media download exceeded its separate 10-minute transfer limit."
	case strings.Contains(err.Error(), "existing output"), strings.Contains(err.Error(), "finalize media"):
		result.LocalErrorCode = "output_file_conflict"
		result.ErrorMessage = "An existing or incomplete output file prevents safe delivery."
		result.NextAction = "Choose another output directory and retrieve this same task index. Existing files were preserved."
	case strings.Contains(err.Error(), "invalid media"):
		result.LocalErrorCode = "invalid_media_content"
		result.ErrorMessage = "The content response is not a supported native media file."
	}
	return result
}
