// Pure Tokens API executor is the small, self-contained process used by the
// installable Skills. It intentionally has no dependency on Node, Python,
// MCP, a proxy, or a background service.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	apiOrigin              = "https://api.puretokensx.com"
	maxConfigBytes         = 1 << 20
	maxResponseBytes       = 2 << 20
	maxImageBytes    int64 = 25 << 20
	maxVideoBytes    int64 = 512 << 20
)

var executorVersion = "0.18.7"
var executorSourceSHA256 = "unbuilt"

const imageInitialPollDelay = 5 * time.Second
const initTimeout = 20 * time.Second

type attachment struct {
	Field             string `json:"field"`
	Path              string `json:"path"`
	snapshot          os.FileInfo
	snapshotDigest    [sha256.Size]byte
	hasSnapshotDigest bool
}

type taskRequest struct {
	MediaOperation         string `json:"media_operation,omitempty"`
	Index                  int    `json:"index,omitempty"`
	TaskStatus             string `json:"task_status,omitempty"`
	OriginalOperation      string `json:"original_operation,omitempty"`
	RetryNotBefore         string `json:"retry_not_before,omitempty"`
	ReconciliationRequired bool   `json:"reconciliation_required,omitempty"`
	WaitWindowsCompleted   int    `json:"wait_windows_completed,omitempty"`
	route                  string

	Kind           string         `json:"kind"`
	Operation      string         `json:"operation"`
	Model          string         `json:"model"`
	Prompt         string         `json:"prompt"`
	TaskID         string         `json:"task_id"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	Attachments    []attachment   `json:"attachments,omitempty"`
	RequestedCount int            `json:"requested_count"`
	OutputDir      string         `json:"output_dir"`
	Poll           *pollRequest   `json:"poll,omitempty"`
}

type pollRequest struct {
	MaxStatusReads int `json:"max_status_reads"`
	DeadlineSecs   int `json:"deadline_seconds"`
}

type receipt struct {
	RequestedCount       int            `json:"requested_count,omitempty"`
	Parameters           map[string]any `json:"parameters,omitempty"`
	SubmissionOutcome    string         `json:"submission_outcome,omitempty"`
	DownloadedPaths      []string       `json:"downloaded_paths,omitempty"`
	DownloadedIndexes    []int          `json:"downloaded_indexes,omitempty"`
	DeliveryStatus       string         `json:"delivery_status,omitempty"`
	LocalErrorCode       string         `json:"local_error_code,omitempty"`
	OriginalOperation    string         `json:"original_operation,omitempty"`
	RetryNotBefore       string         `json:"retry_not_before,omitempty"`
	RecordPath           string         `json:"record_path,omitempty"`
	DeliveredIndexes     []int          `json:"delivered_indexes,omitempty"`
	WaitWindowsCompleted int            `json:"wait_windows_completed,omitempty"`
	WaitOutcome          string         `json:"wait_outcome,omitempty"`
	NextStep             string         `json:"next_step,omitempty"`

	OK                     bool   `json:"ok"`
	Kind                   string `json:"kind,omitempty"`
	Operation              string `json:"operation,omitempty"`
	Model                  string `json:"model,omitempty"`
	TaskID                 string `json:"task_id,omitempty"`
	Status                 string `json:"status,omitempty"`
	ReconciliationRequired bool   `json:"reconciliation_required,omitempty"`
	FailurePhase           string `json:"failure_phase,omitempty"`
	HTTPStatus             int    `json:"http_status,omitempty"`
	APIErrorCode           string `json:"api_error_code,omitempty"`
	ErrorMessage           string `json:"error_message,omitempty"`
	NextAction             string `json:"next_action,omitempty"`
	RetryAfterSecs         int    `json:"retry_after_seconds,omitempty"`
}

type initReceipt struct {
	OK                   bool     `json:"ok"`
	Command              string   `json:"command"`
	ExecutorVersion      string   `json:"executor_version"`
	ConfigurationStatus  string   `json:"configuration_status"`
	APIRequestExecuted   bool     `json:"api_request_executed"`
	CredentialVerified   bool     `json:"credential_verified"`
	APIIdentityConfirmed bool     `json:"api_identity_confirmed"`
	HTTPStatus           int      `json:"http_status,omitempty"`
	Message              string   `json:"message"`
	NextAction           string   `json:"next_action"`
	UsageExamples        []string `json:"usage_examples"`
}

type service struct {
	profilesRoot string
	host         string

	baseURL           string
	client            *http.Client
	token             string
	downloadTimeout   time.Duration
	musicFormat       string
	now               func() time.Time
	wait              func(context.Context, time.Duration) bool
	downloadProofs    map[string]downloadProof
	localRecoveryOnly bool
	support           *supportReceiptWriter
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		// Every expected failure is emitted by run as exactly one sanitized
		// receipt. Never append a second receipt that could make callers think a
		// request was repeated or had a different outcome.
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	if len(args) == 2 && args[0] == "operations-receipt" && args[1] == "installed" {
		dispatchOperationsReceipt("installed")
		return nil
	}
	if len(args) == 1 && args[0] == "--build-info" {
		writeJSON(output, map[string]string{"version": executorVersion, "source_sha256": executorSourceSHA256})
		return nil
	}
	if len(args) == 1 && args[0] == "--version" {
		_, err := fmt.Fprintln(output, executorVersion)
		return err
	}
	// Public group names never require a host connection, credential adapter,
	// installed profile, or inference request.
	if len(args) > 0 && args[0] == "model-groups" {
		return runModelGroupsCommand(args[1:], output)
	}
	if len(args) > 0 && (args[0] == "install-inventory" || args[0] == "install-verify") {
		return runInstallationGuard(args, output)
	}
	support := &supportReceiptWriter{output: output}
	defer func() { dispatchOperationsReceipt(support.operationsEvent) }()
	output = support
	if len(args) < 1 {
		writeReceipt(output, validationFailure("Choose init, doctor, connection, balance, models, model-groups, evaluate, audio, audio-verify, preflight, submit, status, wait, content, resume or delivered."))
		return errors.New("missing command")
	}

	command := args[0]
	support.command = command
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	host := flags.String("host", "", "")
	requestFile := flags.String("request", "", "")
	recordFile := flags.String("record", "", "")
	index := flags.Int("index", 0, "")
	outputDir := flags.String("output-dir", "", "")
	parseErr := flags.Parse(args[1:])
	support.host = *host
	if parseErr != nil || (*host == "" && command != "delivered") || len(flags.Args()) != 0 {
		message := "Identify the current host and use supported command options before starting the request."
		if command == "evaluate" {
			writeJSON(output, evaluationFailure("not_submitted", "validation", "evaluation_request_invalid", message, 0))
		} else if command == "audio" || command == "audio-verify" {
			result := audioFailure("not_submitted", "validation", "audio_request_invalid", message, 0)
			result.Command = command
			writeJSON(output, result)
		} else {
			writeReceipt(output, validationFailure(message))
		}
		return errors.New("invalid command")
	}
	isTask := command == "submit" || command == "status" || command == "wait" || command == "content" || command == "resume" || command == "delivered" || command == "preflight"
	if !isTask && command != "init" && command != "doctor" && command != "connection" && command != "balance" && command != "models" && command != "evaluate" && command != "audio" && command != "audio-verify" {
		writeReceipt(output, validationFailure("Choose init, doctor, connection, balance, models, model-groups, evaluate, audio, audio-verify, preflight, submit, status, wait, content, resume or delivered."))
		return errors.New("unknown command")
	}
	if command == "evaluate" || command == "audio" || command == "audio-verify" {
		unsupportedOption := false
		flags.Visit(func(option *flag.Flag) {
			if option.Name != "host" && option.Name != "request" {
				unsupportedOption = true
			}
		})
		if unsupportedOption {
			if command == "audio" || command == "audio-verify" {
				result := audioFailure("not_submitted", "validation", "audio_request_invalid",
					"Audio accepts only --host and --request; task polling and recovery flags do not apply. No API request was sent.", 0)
				result.Command = command
				writeJSON(output, result)
			} else {
				writeJSON(output, evaluationFailure("not_submitted", "validation", "evaluation_request_invalid",
					"Evaluation accepts only --host and --request; media task and delivery options do not apply. No API request was sent.", 0))
			}
			return errors.New("incompatible synchronous command options")
		}
	}
	if (*recordFile != "" && (!isTask || command == "preflight")) || ((*index != 0 || *outputDir != "") && (*recordFile == "" || (command != "content" && command != "delivered"))) || ((command == "resume" || command == "delivered") && *recordFile == "") || (*recordFile != "" && *requestFile != "" && command != "submit") || (!isTask && command != "models" && command != "evaluate" && command != "audio" && command != "audio-verify" && *requestFile != "") {
		writeReceipt(output, validationFailure("Use --request for media JSON, --record for same-task recovery, and --index/--output-dir only for recorded content. Never submit into an existing record."))
		return errors.New("incompatible command options")
	}
	svc := service{baseURL: apiOrigin, client: &http.Client{Timeout: 90 * time.Second, CheckRedirect: rejectRedirect}, profilesRoot: installedSkillsRoot(), host: *host, support: support}
	if command == "evaluate" {
		return runEvaluationCommand(output, svc, *host, *requestFile)
	}
	if command == "audio-verify" {
		return runAudioVerify(output, *requestFile)
	}
	if command == "audio" {
		return runAudioCommand(output, svc, *host, *requestFile)
	}
	if *requestFile != "" {
		data, err := readBoundedFile(*requestFile, maxResponseBytes)
		if err != nil || !json.Valid(data) {
			writeReceipt(output, validationFailure("The request file must contain one readable JSON object; no API request was sent."))
			return errors.New("invalid request file")
		}
		input = bytes.NewReader(data)
	}
	var request taskRequest
	if isTask {
		if *recordFile != "" && command != "submit" {
			record, err := loadTaskRecord(*recordFile)
			if err != nil {
				writeReceipt(output, validationFailure("The task record could not be read. Recover the original receipt; do not submit again."))
				return err
			}
			request = record.request()
		} else {
			var err error
			request, err = decodeTaskRequest(input)
			if err != nil {
				writeReceipt(output, validationFailure("A complete media request JSON is required; no API request was sent."))
				return err
			}
		}
		if command == "status" || command == "wait" || command == "content" || command == "resume" || command == "delivered" {
			request.Operation = "continue"
		}
	}
	if isTask && request.Kind == "music" && (*recordFile == "" || command == "submit") {
		if err := validateTaskRequest(request); err != nil {
			writeReceipt(output, validationFailure("The music request is invalid; no API request was sent."))
			return err
		}
		if command == "submit" || command == "preflight" {
			if err := prepareMusicRequest(&request); err != nil {
				writeReceipt(output, validationFailure(err.Error()))
				return err
			}
		}
	}
	// A lost music submission response cannot be recovered by finding another
	// credential. Preserve its recorded outcome and stop before any host read.
	if isTask && request.Kind == "music" && *recordFile != "" && command != "submit" && request.TaskID == "" {
		svc.localRecoveryOnly = true
		return executeRecordedTask(command, *recordFile, request, *index, *outputDir, output, svc)
	}
	if command == "delivered" {
		return executeRecordedTask(command, *recordFile, request, *index, *outputDir, output, svc)
	}
	// A recorded completed task needs only local proof verification to expose
	// an undelivered file. Do not make attachment recovery depend on credentials.
	if (command == "resume" || command == "wait") && *recordFile != "" && terminalSuccess(request.TaskStatus) && !request.ReconciliationRequired {
		svc.localRecoveryOnly = true
		return executeRecordedTask(command, *recordFile, request, *index, *outputDir, output, svc)
	}
	token, err := credentialForHost(*host)
	if err != nil {
		if command == "init" {
			writeJSON(output, initCredentialFailure(err))
			return err
		}
		if command == "doctor" {
			return executeDoctorCredentialFailure(output, svc, *host, err)
		}
		status, message, next := credentialFailureDetails(err)
		result := receipt{}
		if isTask {
			if !validTaskID(request.TaskID) {
				request.TaskID = ""
			}
			result = taskReceipt(request, request.TaskID, request.TaskStatus)
		}
		writeReceipt(output, mergeFailure(result, receipt{FailurePhase: "validation", LocalErrorCode: status, ErrorMessage: message, NextAction: next}))
		return err
	}
	defer clearString(&token)
	svc.token = token
	if *recordFile != "" {
		return executeRecordedTask(command, *recordFile, request, *index, *outputDir, output, svc)
	}
	if isTask {
		data, _ := json.Marshal(request)
		input = bytes.NewReader(data)
	}
	switch command {
	case "init":
		return executeInit(output, svc)
	case "doctor":
		return executeDoctor(output, svc, *host)
	case "connection":
		return executeReadOnly(output, svc, "/v1", "connection")
	case "balance":
		return executeBalance(output, svc)
	case "models":
		if *requestFile == "" {
			input = nil
		}
		return executeModelQuery(output, svc, input)
	case "preflight":
		return executePreflight(output, svc, request)
	case "submit":
		return executeTask(input, output, svc)
	case "status", "wait", "content":
		return executeExistingTask(command, input, output, svc)
	default:
		writeReceipt(output, validationFailure("This command is unavailable."))
		return errors.New("unavailable command")
	}
}

func executeInit(output io.Writer, svc service) error {
	return executeInitContext(context.Background(), output, svc)
}

func executeInitContext(parent context.Context, output io.Writer, svc service) error {
	// Identity and authentication share one budget, including response bodies.
	ctx, cancel := context.WithTimeout(parent, initTimeout)
	defer cancel()
	result := initReceipt{Command: "init", ExecutorVersion: executorVersion, ConfigurationStatus: "unverified", UsageExamples: usageExamples()}
	body, status, _, code, _, err := svc.request(ctx, http.MethodGet, "/v1", nil, "")
	result.APIRequestExecuted = true
	result.HTTPStatus = status
	if err != nil {
		if status > 0 {
			result.ConfigurationStatus = "api_response_unreadable"
			result.Message = "The fixed API responded, but its identity response could not be read safely."
			result.NextAction = "Check the network path to Pure Tokens, then run init again."
		} else {
			result.ConfigurationStatus = "api_network_unavailable"
			result.Message = "The fixed Pure Tokens API could not be reached from this host session."
			result.NextAction = "Check this host's network permission and connection, then run init again."
		}
		writeJSON(output, result)
		return errors.New("init API request failed")
	}
	if status < 200 || status >= 300 {
		result.ConfigurationStatus = "api_identity_rejected"
		result.Message = "The fixed Pure Tokens API rejected the identity check."
		result.NextAction = "The public identity check did not confirm the API. Try this read-only check later or contact Pure Tokens support; do not infer that the credential is invalid."
		if action := apiErrorAction(code, status); action != "" && status != 401 && status != 403 {
			result.NextAction = action + " Run init again only when requested."
		}
		writeJSON(output, result)
		return errors.New("init connection check failed")
	}
	var identity map[string]any
	if json.Unmarshal(body, &identity) != nil {
		result.ConfigurationStatus = "api_identity_unreadable"
		result.Message = "The fixed Pure Tokens API returned an unreadable identity response."
		result.NextAction = "Run init again in a new host session."
		writeJSON(output, result)
		return errors.New("init identity unreadable")
	}
	statusValue, _ := identity["status"].(string)
	name, _ := identity["name"].(string)
	basePath, _ := identity["base_url"].(string)
	if statusValue == "ok" && name == "Pure Tokens API" && basePath == "/v1" {
		result.APIIdentityConfirmed = true
		authBody, authStatus, _, authCode, _, authErr := svc.request(ctx, http.MethodGet, "/v1/media/models", nil, "")
		authObject, authDecodeErr := readAPIObject(authBody)
		_, catalogOK := authObject["data"].([]any)
		if authErr != nil || authStatus != http.StatusOK || authDecodeErr != nil || !catalogOK {
			result.ConfigurationStatus = "credential_unverified"
			result.HTTPStatus = authStatus
			result.Message = "The public API is reachable, but credential authentication was not confirmed."
			result.NextAction = "The catalog check could not confirm authentication. Try this read-only check later; an unreadable or unavailable catalog does not prove the credential is invalid."
			if action := apiErrorAction(authCode, authStatus); action != "" {
				result.NextAction = action + " Run init again only when requested."
			}
			writeJSON(output, result)
			return errors.New("credential authentication unconfirmed")
		}
		result.OK = true
		result.CredentialVerified = true
		result.ConfigurationStatus = "verified"
		result.Message = "The current host can reach the fixed Pure Tokens API."
		if svc.host == "qoder" {
			result.Message = "The configured Pure Tokens connection can reach the fixed API. This does not identify the connection selected for the current chat."
		}
		if svc.host == "zcode" {
			result.Message = "The enabled ZCode Pure Tokens connection can reach the fixed API. This does not identify the model or connection selected for the current chat."
		}
		result.NextAction = "Use puretokens-image, puretokens-video, puretokens-audio, puretokens-balance, puretokens-models, puretokens-evaluate, or puretokens-connection in a new host conversation."
	} else {
		result.ConfigurationStatus = "api_identity_unconfirmed"
		result.Message = "The fixed Pure Tokens API did not return a confirmable identity."
		result.NextAction = "The public identity response was not confirmable. Try the read-only check later or contact Pure Tokens support; do not infer a credential or balance problem."
	}
	writeJSON(output, result)
	if !result.OK {
		return errors.New("init identity unconfirmed")
	}
	return nil
}

func initCredentialFailure(err error) initReceipt {
	status, message, nextAction := credentialFailureDetails(err)
	return initReceipt{
		Command:             "init",
		ExecutorVersion:     executorVersion,
		ConfigurationStatus: status,
		Message:             message,
		NextAction:          nextAction,
		UsageExamples:       usageExamples(),
	}
}

func usageExamples() []string {
	return []string{
		"Generate an image: create an image of a misty mountain lake at sunrise.",
		"Choose an image model: use grok-imagine-image to create a product illustration.",
		"Generate multiple variants: ask for image variations using a model and count declared by its installed profile.",
		"Generate a video: create a 5-second cinematic shot of a paper boat on a river.",
		"Use a reference: use the attached image as the first frame or visual reference, and say which role you intend.",
		"Check current models: ask which Pure Tokens image/video models and operations are currently available.",
		"Check balance: ask for the current Pure Tokens balance.",
		"Update Skills: ask to update Pure Tokens Skills, then start a new host conversation.",
	}
}

func credentialForHost(host string) (string, error) {
	// A host adapter must only inspect its exact documented effective
	// connection files. It must not scan the home directory or fall back to a
	// random environment variable, because that could send an unrelated key to
	// the fixed Pure Tokens endpoint.
	switch host {
	case "codex":
		return credentialFromCodex()
	case "claude-code":
		return credentialFromClaudeCode()
	case "claude-desktop":
		return credentialFromClaudeDesktop()
	case "dsh-desktop":
		return credentialFromDSHDesktop()
	case "deepseek-harness":
		return credentialFromDeepSeekHarness()
	case "minimax-code":
		return credentialFromMiniMaxCode()
	case "kimi-code", "qoder", "pi":
		return credentialFromNewHost(host)
	case "hermes", "evox", "vscode", "octop":
		return credentialFromFileClient(host)
	case "zcode":
		return credentialFromZCode()
	case "gemini-cli":
		return credentialFromGeminiCLI()
	case "workbuddy":
		return credentialFromWorkBuddy()
	case "grok-build":
		return credentialFromGrokBuild()
	case "opencode":
		return credentialFromOpenCode()
	default:
		return "", credentialFailure("host_credential_adapter_unavailable", "This host has no supported managed connection record for API requests.", "Use a supported host with a configured Pure Tokens connection, then run init again.")
	}
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := io.LimitReader(file, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("configuration exceeds supported size")
	}
	return bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), nil
}

func executeReadOnly(output io.Writer, svc service, path string, command string) error {
	body, status, retryAfter, apiCode, apiMessage, err := svc.request(context.Background(), http.MethodGet, path, nil, "")
	if err != nil || status < 200 || status >= 300 {
		result := apiFailure("submission", status, retryAfter, apiCode, apiMessage, "The read-only API request did not complete. Check network access or offer a later query; do not infer a credential problem.")
		if command == "connection" {
			// Public identity errors cannot diagnose the user's credential.
			result.NextAction = "The public identity check did not confirm the API. Offer a later read-only check or contact Pure Tokens support; this check does not verify credentials, balance or model permission."
			writeJSON(output, result)
		} else {
			writeReceipt(output, result)
		}
		return errors.New("read-only request failed")
	}
	payload, decodeErr := readAPIObject(body)
	if decodeErr != nil {
		writeReceipt(output, receipt{OK: false, FailurePhase: "submission", HTTPStatus: status, ErrorMessage: "The API returned an unreadable response.", NextAction: "Try again in a new host session."})
		return errors.New("invalid json")
	}
	// The command result is retained as JSON, but no host configuration or
	// credential is added to it. The calling Skill selects its user-visible
	// subset from the API response.
	var visible any = payload
	if command == "models" {
		if _, ok := payload["data"].([]any); !ok {
			writeReceipt(output, validationFailure("The API did not return a readable model catalog."))
			return errors.New("invalid catalog")
		}
		visible = projectCatalog(payload)
	}
	if command == "connection" {
		original := jsonObject(payload)
		basePath := ""
		if original["base_url"] == "/v1" {
			basePath = "/v1"
		}
		visible = map[string]any{"status": safePublicString(jsonString(original["status"])), "name": safePublicString(jsonString(original["name"])), "base_url": basePath, "credential_verified": false}
	}
	writeJSON(output, map[string]any{"ok": true, "command": command, "result": visible})
	return nil
}

func executeTask(input io.Reader, output io.Writer, svc service) error {
	request, err := decodeTaskRequest(input)
	if err != nil {
		writeReceipt(output, validationFailure("The media request is incomplete or invalid; no API request was sent."))
		return err
	}
	if err := validateTaskRequest(request); err != nil {
		writeReceipt(output, validationFailure("The requested model operation or attachment input is invalid; no API request was sent."))
		return err
	}
	if request.Operation == "continue" {
		result := taskReceipt(request, request.TaskID, continuationStatus(request.TaskStatus))
		return pollAndDeliver(output, svc, request, result)
	}

	if err := prepareProfileRequest(&request, svc); err != nil {
		writeReceipt(output, validationFailure(err.Error()))
		return err
	}
	return executePreparedTask(output, svc, request)
}

// The record path and ordinary submit path share this single POST entrance.
// All profile validation must finish before creating a submission record.
func executePreparedTask(output io.Writer, svc service, request taskRequest) error {
	path, contentType, body, err := taskRequestBody(request)
	if err != nil {
		writeReceipt(output, validationFailure("The current attachment could not be prepared; no API request was sent."))
		return err
	}
	if closer, ok := body.(io.Closer); ok {
		defer closer.Close()
	}
	response, status, retryAfter, apiCode, apiMessage, err := svc.request(context.Background(), http.MethodPost, path, body, contentType)
	if err != nil || status < 200 || status >= 300 {
		result := mergeFailure(taskReceipt(request, "", ""), apiFailure("submission", status, retryAfter, apiCode, apiMessage, "Review the returned error before changing this request."))
		result.SubmissionOutcome = "rejected"
		if err != nil || status >= 500 || (status >= 300 && status < 400) {
			result.SubmissionOutcome = "unknown"
			result.NextAction = "Submission may have started. Do not automatically repeat the POST or infer billing; ask the user before creating another task."
		}
		if status >= 200 && status < 300 && errors.Is(err, errAPIResponseUnreadable) {
			result = mergeFailure(result, taskIdentityFailure("submission", "task_response_unreadable", status))
		}
		writeReceipt(output, withRetry(result, retryAfter, svc.clock()))
		return errors.New("submission failed")
	}

	taskID, state, reconciliationRequired, identityCode := parseTaskIdentity(response)
	if identityCode != "" {
		result := mergeFailure(taskReceipt(request, "", ""), taskIdentityFailure("submission", identityCode, status))
		result.SubmissionOutcome = "unknown"
		writeReceipt(output, withRetry(result, retryAfter, svc.clock()))
		return errors.New("task id missing")
	}
	acceptedAt := svc.clock()
	result := taskReceipt(request, taskID, state)
	result = withRetry(result, retryAfter, acceptedAt)
	result.SubmissionOutcome = "accepted"
	result.ReconciliationRequired = reconciliationRequired
	if reconciliationRequired {
		result.OK = false
		result.FailurePhase = "status"
		result.ErrorMessage = "The task requires server-side reconciliation before its result can be confirmed."
		result.NextAction = "Keep this task ID and ask the user before continuing the same task. Do not submit a replacement task."
		writeReceipt(output, result)
		return errors.New("task reconciliation required")
	}
	if terminalFailure(state) {
		result.OK = false
		result.FailurePhase = "submission"
		result.APIErrorCode, result.ErrorMessage = publicAPIError(response)
		if result.ErrorMessage == "" {
			result.ErrorMessage = "The API returned a failed task state."
		}
		result.NextAction = "Review the task error before creating a new request."
		writeReceipt(output, result)
		return errors.New("initial failed state")
	}
	if !terminalSuccess(state) && state != "pending" && state != "queued" && state != "processing" && state != "running" && state != "in_progress" {
		result.Status = "unknown"
		result.OK = false
		result.FailurePhase = "status"
		result.ErrorMessage = "The API returned an unsupported initial task state."
		result.NextAction = "Keep this task ID and query the same task later; do not resubmit."
		writeReceipt(output, result)
		return errors.New("unknown initial state")
	}
	// Return the accepted task immediately: no polling or downloads hide its ID.
	if terminalSuccess(state) {
		result.NextAction = "Download the completed task with content, one index at a time."
	} else {
		if request.Kind == "image" && result.RetryNotBefore == "" {
			// Persist the initial wait once, anchored to the accepted response.
			// Existing receipts/records already carry this timestamp across
			// commands. Do not label our local delay as a server Retry-After.
			result.RetryNotBefore = acceptedAt.Add(imageInitialPollDelay).UTC().Format(time.RFC3339Nano)
		}
		result.NextAction = "Keep this task ID and retry_not_before. Use wait or status for this same task; never submit it again."
	}
	writeReceipt(output, result)
	return nil
}

func decodeTaskRequest(input io.Reader) (taskRequest, error) {
	var request taskRequest
	decoder := json.NewDecoder(io.LimitReader(input, maxResponseBytes))
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return taskRequest{}, err
	}
	typed := json.NewDecoder(bytes.NewReader(raw))
	typed.DisallowUnknownFields()
	if err := typed.Decode(&request); err != nil {
		return taskRequest{}, err
	}
	if request.Kind == "music" {
		value, err := strictEvaluationJSON(raw, 64<<10)
		fields, ok := value.(map[string]any)
		if err != nil || !ok {
			return taskRequest{}, errors.New("invalid music JSON")
		}
		for key, v := range fields {
			if v == nil || !contains([]string{"kind", "operation", "model", "prompt", "parameters", "attachments", "output_dir", "poll", "media_operation", "task_id", "task_status", "requested_count", "original_operation", "index", "wait_windows_completed", "reconciliation_required", "retry_not_before"}, key) {
				return taskRequest{}, errors.New("invalid music field")
			}
		}
	}
	// Do not wait for EOF on a terminal pipe after the complete JSON object.
	// --request is preferred; reject additional content already buffered.
	buffered, _ := io.ReadAll(decoder.Buffered())
	if len(bytes.TrimSpace(buffered)) != 0 {
		return taskRequest{}, errors.New("task input must contain exactly one JSON object")
	}
	return request, nil
}

func validateTaskRequest(request taskRequest) error {
	if request.Kind != "image" && request.Kind != "video" && request.Kind != "music" {
		return errors.New("invalid kind")
	}
	if request.Kind == "music" {
		if err := validateMusicRequest(request); err != nil {
			return err
		}
	}
	if request.Operation != "generate" && request.Operation != "edit" && request.Operation != "continue" {
		return errors.New("invalid operation")
	}
	if request.RequestedCount < 0 || request.RequestedCount > 6 {
		return errors.New("invalid requested count")
	}
	if request.WaitWindowsCompleted < 0 || request.WaitWindowsCompleted > 2 {
		return errors.New("invalid wait progress")
	}
	if request.Operation != "continue" && request.OutputDir != "" {
		info, err := os.Stat(request.OutputDir)
		if err != nil || !info.IsDir() {
			return errors.New("invalid output directory")
		}
	}
	if request.Operation == "continue" {
		if !validTaskID(request.TaskID) || len(request.Attachments) != 0 {
			return errors.New("invalid continuation")
		}
		if request.RetryNotBefore != "" {
			if _, err := time.Parse(time.RFC3339, request.RetryNotBefore); err != nil {
				return errors.New("invalid retry time")
			}
		}
		return nil
	}
	if request.TaskID != "" || request.TaskStatus != "" || request.OriginalOperation != "" || request.RetryNotBefore != "" || request.ReconciliationRequired || request.Index != 0 || request.WaitWindowsCompleted != 0 {
		return errors.New("new submissions cannot contain existing-task metadata")
	}
	if strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.Prompt) == "" {
		return errors.New("missing model or prompt")
	}

	limit := attachmentLimit(request.Kind)
	var total int64
	for i, attachment := range request.Attachments {
		if attachment.Field == "" || !filepath.IsAbs(attachment.Path) {
			return errors.New("attachment must have a field and absolute path")
		}
		info, err := os.Stat(attachment.Path)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > limit {
			return errors.New("invalid attachment")
		}
		total += info.Size()
		if total > limit {
			return errors.New("attachments exceed the request size limit")
		}
		request.Attachments[i].snapshot = info
	}
	if len(request.Attachments) > 0 {
		files, err := openAttachments(request)
		if err != nil {
			return err
		}
		defer closeAttachments(files)
		for i, file := range files {
			request.Attachments[i].snapshot = file.info
			request.Attachments[i].snapshotDigest = file.digest
			request.Attachments[i].hasSnapshotDigest = true
		}
	}
	return nil
}

func attachmentLimit(kind string) int64 {
	if kind == "music" {
		return maxMusicBytes
	}
	if kind == "image" {
		return maxImageBytes
	}
	return maxVideoBytes
}

func taskRequestBody(request taskRequest) (string, string, io.Reader, error) {
	if request.Kind == "music" {
		return musicRequestBody(request)
	}
	path := endpointFor(request)
	if len(request.Attachments) == 0 {
		payload := map[string]any{"model": request.Model, "prompt": request.Prompt}
		for key, value := range request.Parameters {
			if key == "model" || key == "prompt" || key == "async" {
				return "", "", nil, errors.New("reserved parameter")
			}
			payload[key] = value
		}
		if request.Kind == "image" {
			payload["async"] = true
		}
		encoded, err := json.Marshal(payload)
		return path, "application/json", bytes.NewReader(encoded), err
	}

	fields := map[string]string{"model": request.Model, "prompt": request.Prompt}
	if request.Kind == "image" {
		fields["async"] = "true"
	}
	for key, value := range request.Parameters {
		if key == "model" || key == "prompt" || key == "async" {
			return "", "", nil, errors.New("reserved parameter")
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", "", nil, err
		}
		if text, ok := value.(string); ok {
			fields[key] = text
		} else {
			fields[key] = string(encoded)
		}
	}
	files, err := openAttachments(request)
	if err != nil {
		return "", "", nil, err
	}
	reader, writerPipe := io.Pipe()
	writer := multipart.NewWriter(writerPipe)
	contentType := writer.FormDataContentType()
	go func() {
		defer writerPipe.Close()
		defer closeAttachments(files)
		for key, value := range fields {
			if err := writer.WriteField(key, value); err != nil {
				_ = writerPipe.CloseWithError(err)
				return
			}
		}
		for i, attachment := range request.Attachments {
			part, err := writer.CreateFormFile(attachment.Field, filepath.Base(attachment.Path))
			if err == nil {
				err = files[i].copyTo(part)
			}
			if err != nil {
				_ = writerPipe.CloseWithError(err)
				return
			}
		}
		if err := writer.Close(); err != nil {
			_ = writerPipe.CloseWithError(err)
		}
	}()
	return path, contentType, reader, nil
}

func endpointFor(request taskRequest) string {
	if request.route != "" {
		return request.route
	}
	if request.Kind == "image" {
		if request.Operation == "edit" {
			return "/v1/images/edits"
		}
		return "/v1/images/generations"
	}
	if request.Operation == "edit" {
		return "/v1/videos/edits"
	}
	return "/v1/videos"
}

func pollAndDeliver(output io.Writer, svc service, request taskRequest, result receipt) error {
	policy := pollingPolicy(request.Kind, request.Poll)
	windowEnd := svc.clock().Add(time.Duration(policy.deadline) * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(policy.deadline)*time.Second)
	defer cancel()
	if request.ReconciliationRequired || terminalFailure(result.Status) || terminalSuccess(result.Status) {
		return finishKnownTask(output, result)
	}
	deferred := false
	for read := 0; read < policy.maxReads; read++ {
		delay := time.Duration(policy.delays[min(read, len(policy.delays)-1)]) * time.Second
		if result.RetryNotBefore != "" {
			delay = retryDelay(result.RetryNotBefore, svc.clock())
		}
		if !svc.waitFor(ctx, delay) {
			deferred = true
			break
		}
		result.RetryNotBefore = ""
		result.RetryAfterSecs = 0
		body, status, retry, code, message, err := svc.request(ctx, http.MethodGet, statusPath(request.Kind, result.TaskID), nil, "")
		if err != nil || status < 200 || status >= 300 {
			failure := apiFailure("status", status, retry, code, message, "Keep this task ID. Continue this same task when the API is available; do not resubmit.")
			if status >= 200 && status < 300 && errors.Is(err, errAPIResponseUnreadable) {
				failure = taskIdentityFailure("status", "task_response_unreadable", status)
			}
			result = withRetry(mergeFailure(result, failure), retry, svc.clock())
			if status == 429 && retry > 0 && read+1 < policy.maxReads {
				continue
			}
			writeReceipt(output, result)
			return errors.New("status read failed")
		}
		result = clearTaskFailure(result)
		result, err = applyTaskStatus(result, body, status)
		result = withRetry(result, retry, svc.clock())
		if err != nil {
			writeReceipt(output, result)
			return err
		}
		if terminalSuccess(result.Status) {
			result.NextAction = "Use content for this task, one index at a time, then attach each downloaded file to the conversation."
			writeReceipt(output, result)
			return nil
		}
	}
	if !result.OK {
		result.OK = false
		result.FailurePhase = "status"
		result.NextAction = "Wait until retry_not_before, then continue this same task. The automatic wait window ended; do not submit again."
		if result.ErrorMessage == "" {
			result.ErrorMessage = "The API requested a wait longer than the remaining automatic wait window."
		}
		writeReceipt(output, result)
		return errors.New("retry exceeds wait window")
	}
	// A bounded foreground wait ending is not a task or network failure.
	// Persist a saturated counter so a host can continue once automatically,
	// without resetting its budget at every receipt/record handoff.
	result.WaitWindowsCompleted = min(2, result.WaitWindowsCompleted+1)
	result.WaitOutcome = "window_ended"
	result.NextAction = "The task is still processing. Continue this same task once more in the active foreground session if delivery is still requested; do not submit again."
	if result.WaitWindowsCompleted >= 2 {
		result.NextAction = "The task is still processing after two wait windows. Keep its ID and ask whether to continue later; do not submit again."
	}
	remainingRetry := retryDelay(result.RetryNotBefore, svc.clock())
	if remainingRetry > 0 && (deferred || remainingRetry >= windowEnd.Sub(svc.clock())) {
		result.WaitOutcome = "retry_deferred"
		result.NextAction = "The task is still processing. Preserve retry_not_before and offer continuation at that time; the remaining wait does not fit this foreground window. Do not submit again."
	}
	writeReceipt(output, result)
	return nil
}

func waitContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	if deadline, ok := ctx.Deadline(); ok && time.Now().Add(delay).After(deadline) {
		return false
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type pollPolicy struct {
	maxReads int
	deadline int
	delays   []int
}

func pollingPolicy(kind string, override *pollRequest) pollPolicy {
	// Fresh images carry their initial delay in retry_not_before. A continued
	// image with no remaining delay can be checked immediately, then every 3s.
	policy := pollPolicy{maxReads: 40, deadline: 120, delays: []int{0, 3}}
	if kind == "video" || kind == "music" {
		policy = pollPolicy{maxReads: 7, deadline: 300, delays: []int{5, 10, 20, 40, 60, 60}}
	}
	if override != nil {
		if override.MaxStatusReads > 0 && override.MaxStatusReads <= policy.maxReads {
			policy.maxReads = override.MaxStatusReads
		}
		if override.DeadlineSecs > 0 && override.DeadlineSecs <= policy.deadline {
			policy.deadline = override.DeadlineSecs
		}
	}
	return policy
}

func deliverTask(output io.Writer, svc service, request taskRequest, result receipt) error {
	// One file per invocation allows the host to hand it off before the next read.
	if request.OutputDir == "" || !filepath.IsAbs(request.OutputDir) {
		writeReceipt(output, mergeFailure(result, validationFailure("Choose an existing absolute output directory for the media file.")))
		return errors.New("output directory required")
	}
	if info, err := os.Stat(request.OutputDir); err != nil || !info.IsDir() {
		result = mergeFailure(result, validationFailure("The output directory is missing or unavailable. Choose an existing output directory, then retrieve this same task."))
		result.LocalErrorCode = "output_directory_unavailable"
		writeReceipt(output, result)
		return errors.New("output directory unavailable")
	}
	if request.Kind == "image" && request.RequestedCount == 0 {
		result = mergeFailure(result, validationFailure("The original image count is unknown. Recover requested_count from the original submission receipt or task record before downloading."))
		result.LocalErrorCode = "original_count_unknown"
		writeReceipt(output, result)
		return errors.New("unknown image count")
	}
	if request.Index < 0 || request.Index >= max(1, request.RequestedCount) || ((request.Kind == "video" || request.Kind == "music") && request.Index != 0) {
		writeReceipt(output, mergeFailure(result, validationFailure("Choose a valid zero-based content index.")))
		return errors.New("invalid index")
	}
	if request.Kind == "music" {
		svc.musicFormat, _ = request.Parameters["response_format"].(string)
	}
	destination, status, retry, code, message, err := svc.download(context.Background(), contentPath(request.Kind, result.TaskID, request.Index), request.Kind, request.OutputDir)
	if err != nil {
		failure := contentFailure(err, status, retry, code, message)
		writeReceipt(output, withRetry(mergeFailure(result, failure), retry, svc.clock()))
		return err
	}
	result.DownloadedPaths = []string{destination}
	result.DownloadedIndexes = []int{request.Index}
	result.DeliveryStatus = "downloaded_awaiting_host_delivery"
	result.Status = "completed"
	result = withRetry(result, retry, svc.clock())
	result.NextAction = "Attach this local file to the user. Download the next missing index only after handing off this file. Keep outputs only as requested by the user."
	writeReceipt(output, result)
	return nil
}

func statusPath(kind, id string) string {
	if kind == "music" {
		return "/v1/audio/music/tasks/" + url.PathEscape(id)
	}
	return "/v1/" + kind + "s/" + url.PathEscape(id)
}

func contentPath(kind, id string, index int) string {
	path := statusPath(kind, id) + "/content"
	if kind == "image" {
		return path + "?index=" + strconv.Itoa(index)
	}
	return path
}

func taskIdentity(body []byte) (string, string, bool, bool) {
	id, state, reconcile, code := parseTaskIdentity(body)
	return id, state, reconcile, code == ""
}

func parseTaskIdentity(body []byte) (string, string, bool, string) {
	var response map[string]any
	if json.Unmarshal(body, &response) != nil || response == nil {
		return "", "", false, "task_response_unreadable"
	}
	identity := response["task_id"]
	if identity == nil || identity == "" {
		identity = response["id"]
	}
	if identity == nil || identity == "" {
		return "", "", false, "task_id_missing"
	}
	id, isString := identity.(string)
	state, _ := response["status"].(string)
	if !isString || !validTaskID(id) {
		return "", "", false, "task_id_invalid"
	}
	reconciliationRequired, _ := response["reconciliation_required"].(bool)
	return id, strings.ToLower(state), reconciliationRequired, ""
}

func taskIdentityFailure(phase, code string, status int) receipt {
	message := "The task response could not be read as a complete JSON object."
	switch code {
	case "task_id_missing":
		message = "The response contains no non-empty top-level task_id or id."
	case "task_id_invalid":
		message = "A task ID field was returned, but its type or format is not supported by this executor."
	}
	return receipt{FailurePhase: phase, HTTPStatus: status, LocalErrorCode: code,
		ErrorMessage: message,
		NextAction:   "Use the existing safe support summary, including executor_version, for diagnosis. Do not infer that generation failed or that no charge occurred. Check the current host's installed version when explicitly requested; do not update, change transport or resubmit automatically."}
}

func terminalSuccess(state string) bool {
	return state == "completed" || state == "succeeded" || state == "success"
}

func terminalFailure(state string) bool {
	return state == "failed" || state == "cancelled" || state == "canceled" || state == "expired" || state == "error"
}

func (svc service) request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, int, int, string, string, error) {
	return svc.requestWithAuthentication(ctx, method, path, body, contentType, true)
}

func (svc service) requestWithAuthentication(ctx context.Context, method, path string, body io.Reader, contentType string, authenticated bool) ([]byte, int, int, string, string, error) {
	bearer := ""
	if authenticated {
		bearer = svc.token
	}
	return svc.requestWithBearer(ctx, method, path, body, contentType, bearer)
}

// Keep the configured credential available for response redaction even when a
// specific endpoint needs a normalized bearer or sends no authentication.
func (svc service) requestWithBearer(ctx context.Context, method, path string, body io.Reader, contentType, bearer string) ([]byte, int, int, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, svc.baseURL+path, body)
	if err != nil {
		return nil, 0, 0, "", "", err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	svc.support.beginRequest(method, path)
	response, err := fixedClient(svc.client).Do(req)
	svc.support.observeResponse(response)
	if err != nil {
		return nil, 0, 0, "", "", err
	}
	defer response.Body.Close()
	bodyBytes, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if readErr != nil || len(bodyBytes) > maxResponseBytes {
		return nil, response.StatusCode, retryAfter(response), "", "", errAPIResponseUnreadable
	}
	if path == evaluationPath && response.StatusCode >= 200 && response.StatusCode < 300 {
		// Validate before sanitization can collapse duplicate object keys.
		if _, err := strictEvaluationJSON(bodyBytes, maxResponseBytes); err != nil {
			return nil, response.StatusCode, retryAfter(response), "", "", errAPIResponseUnreadable
		}
	}
	if bearer != "" && bearer != svc.token {
		bodyBytes = sanitizeResponseJSON(bodyBytes, bearer)
	}
	bodyBytes = sanitizeResponseJSON(bodyBytes, svc.token)
	if bodyBytes == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil, response.StatusCode, retryAfter(response), "", "", errAPIResponseUnreadable
	}
	code, message := publicAPIError(bodyBytes)
	return bodyBytes, response.StatusCode, retryAfter(response), code, message, nil
}

func (svc service) download(ctx context.Context, path, kind, outputDir string) (string, int, int, string, string, error) {
	if !filepath.IsAbs(outputDir) {
		return "", 0, 0, "", "", errors.New("absolute output directory required")
	}
	outputID := fmt.Sprintf("puretokens-%x", sha256.Sum256([]byte(path)))
	for _, format := range []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/avif", "video/mp4", "video/webm", "audio/mpeg", "audio/wav"} {
		candidate := filepath.Join(outputDir, outputID+"."+extensionFor(format, kind))
		if info, err := os.Lstat(candidate); err == nil {
			proof, known := svc.downloadProofs[candidate]
			if info.Mode().IsRegular() && known && proof.MediaType == format &&
				(kind != "music" || svc.musicFormat == "" || format == audioMIME(svc.musicFormat)) &&
				validDownloadProof(proof, kind) && matchesDownloadProof(candidate, proof) {
				return candidate, 200, 0, "", "", nil
			}
			return "", 0, 0, "", "", errors.New("existing output has no matching download proof; preserve it and choose another output directory")
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.baseURL+path, nil)
	if err != nil {
		return "", 0, 0, "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+svc.token)
	client := fixedClient(svc.client)
	client.Timeout = svc.contentTimeout()
	svc.support.beginRequest(http.MethodGet, path)
	response, err := client.Do(req)
	svc.support.observeResponse(response)
	if err != nil {
		return "", 0, 0, "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
		code, message := publicAPIError(sanitizeResponseJSON(body, svc.token))
		return "", response.StatusCode, retryAfter(response), code, message, errors.New("content request rejected")
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if (kind == "image" && !strings.HasPrefix(contentType, "image/")) || (kind == "video" && !strings.HasPrefix(contentType, "video/")) || (kind == "music" && contentType != "audio/mpeg" && contentType != "audio/wav") || contentType == "image/svg+xml" {
		return "", response.StatusCode, retryAfter(response), "", "", errInvalidMedia
	}
	if kind == "music" && svc.musicFormat != "" && contentType != audioMIME(svc.musicFormat) {
		return "", response.StatusCode, retryAfter(response), "", "", errInvalidMedia
	}
	limit := maxImageBytes
	if kind == "video" {
		limit = maxVideoBytes
	} else if kind == "music" {
		limit = maxMusicBytes
	}
	if response.ContentLength > limit {
		return "", response.StatusCode, retryAfter(response), "", "", errMediaTooLarge
	}
	directory := outputDir
	if directory == "" {
		return "", response.StatusCode, 0, "", "", errors.New("explicit output directory required")
	}
	reader := bufio.NewReader(response.Body)
	prefix, _ := reader.Peek(512)
	if !validMediaPrefix(prefix, contentType) {
		return "", response.StatusCode, 0, "", "", errInvalidMedia
	}
	extension := extensionFor(contentType, kind)
	file, err := os.CreateTemp(directory, ".puretokens-part-*."+extension)
	if err != nil {
		return "", response.StatusCode, retryAfter(response), "", "", err
	}
	name := file.Name()
	defer os.Remove(name)
	writer := io.LimitReader(reader, limit+1)
	streamDigest := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, streamDigest), writer)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return "", response.StatusCode, retryAfter(response), "", "", errors.Join(copyErr, closeErr)
	}
	if written > limit {
		return "", response.StatusCode, retryAfter(response), "", "", errMediaTooLarge
	}
	if written == 0 || !validMediaFile(name, contentType) {
		return "", response.StatusCode, retryAfter(response), "", "", errInvalidMedia
	}
	// Hash the bytes as they are downloaded, avoiding an extra complete scan
	// of the temporary file. The finalized file is still re-read and verified.
	proof := downloadProof{SHA256: fmt.Sprintf("%x", streamDigest.Sum(nil)), Bytes: written, MediaType: contentType}
	destination := filepath.Join(directory, outputID+"."+extension)
	if err := os.Link(name, destination); err != nil {
		if os.IsExist(err) || copyDownloadExclusive(name, destination) != nil {
			return "", response.StatusCode, 0, "", "", errors.New("cannot finalize media file without overwriting existing output")
		}
	}
	if !matchesDownloadProof(destination, proof) {
		return "", response.StatusCode, 0, "", "", errors.New("download changed during finalization")
	}
	if svc.downloadProofs != nil {
		svc.downloadProofs[destination] = proof
	}
	return destination, response.StatusCode, retryAfter(response), "", "", nil
}

func extensionFor(contentType, kind string) string {
	switch contentType {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "image/avif":
		return "avif"
	case "video/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	case "audio/mpeg":
		return "mp3"
	case "audio/wav":
		return "wav"
	default:
		return map[string]string{"image": "img", "video": "video"}[kind]
	}
}

func retryAfter(response *http.Response) int {
	value, err := strconv.Atoi(strings.TrimSpace(response.Header.Get("Retry-After")))
	if err != nil {
		if at, parseErr := http.ParseTime(response.Header.Get("Retry-After")); parseErr == nil {
			value = int(time.Until(at).Seconds()) + 1
		}
	}
	if value <= 0 {
		return 0
	}
	return value
}

func publicAPIError(body []byte) (string, string) {
	var document map[string]any
	if json.Unmarshal(body, &document) != nil {
		return "", ""
	}
	if errorValue, ok := document["error"].(map[string]any); ok {
		code, _ := errorValue["code"].(string)
		message, _ := errorValue["message"].(string)
		return publicErrorCategory(code, message)
	}
	code, _ := document["code"].(string)
	message, _ := document["message"].(string)
	return publicErrorCategory(code, message)
}

var publicCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,100}$`)
var credentialPattern = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]+|bearer\s+\S+|(?:api[_ -]?key|token|secret)\s*[:=]\s*\S+)`)

func safePublicString(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if len(value) > 500 || strings.Contains(lower, "http:") || strings.Contains(lower, "https:") || credentialPattern.MatchString(value) || strings.Contains(lower, "stack trace") {
		return ""
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ""
		}
	}
	// Raw hostnames and paths do not belong in public API error messages.
	if regexp.MustCompile(`(?i)\b[a-z0-9-]+\.(com|net|org|internal|local|ai)\b`).MatchString(value) {
		return ""
	}
	return value
}

func apiFailure(phase string, status, retryAfterSecs int, code, message, next string) receipt {
	result := receipt{OK: false, FailurePhase: phase, HTTPStatus: status, APIErrorCode: code, ErrorMessage: message, NextAction: next, RetryAfterSecs: retryAfterSecs}
	if result.ErrorMessage == "" {
		result.ErrorMessage = "The Pure Tokens API request did not complete."
	}
	return result
}

func validationFailure(message string) receipt {
	return receipt{OK: false, FailurePhase: "validation", ErrorMessage: message, NextAction: "Correct the local request before trying again."}
}

func writeReceipt(output io.Writer, result receipt) {
	result = guideReceipt(result)
	if writer, ok := output.(*recordReceiptWriter); ok {
		writer.writeReceipt(result)
		return
	}
	writeJSON(output, result)
}

func writeJSON(output io.Writer, value any) {
	if writer, ok := output.(*supportReceiptWriter); ok {
		writer.writeJSON(value)
		return
	}
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(value)
}

func clearString(value *string) {
	// Go strings cannot be reliably zeroed. Dropping the reference immediately
	// after a request still prevents any intentional persistence or logging.
	*value = ""
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
