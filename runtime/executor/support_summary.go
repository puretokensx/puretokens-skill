package main

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var canonicalRequestID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// One invocation's evidence, never a credential store or a task record. The
// latest HTTP attempt replaces the previous one, even if it got no response.
type supportReceiptWriter struct {
	output          io.Writer
	host            string
	command         string
	attempted       bool
	requestPhase    string
	requestID       string
	httpStatus      int
	operationsEvent string
	timings         *commandTimings
}

func (writer *supportReceiptWriter) Write(data []byte) (int, error) {
	return writer.output.Write(data)
}

func (writer *supportReceiptWriter) beginRequest(method, path string) {
	if writer == nil {
		return
	}
	writer.attempted = true
	writer.requestID, writer.requestPhase, writer.httpStatus = "", "", 0
	path, _, _ = strings.Cut(path, "?")
	switch {
	case path == "/v1":
		writer.requestPhase = "identity"
	case path == "/v1/media/models" || path == "/v1/models" || path == modelGroupsPath:
		writer.requestPhase = "catalog"
	case path == balanceUsagePath:
		writer.requestPhase = "balance_usage"
	case path == balanceUnitPath:
		writer.requestPhase = "balance_metadata"
	case method == http.MethodPost:
		writer.requestPhase = "submission"
	case strings.HasSuffix(path, "/content"):
		writer.requestPhase = "content"
	default:
		writer.requestPhase = "status"
	}
}

func (writer *supportReceiptWriter) observeResponse(response *http.Response) {
	if writer == nil || response == nil {
		return
	}
	writer.httpStatus = response.StatusCode
	standard, legacy := response.Header.Values("X-Request-ID"), response.Header.Values("X-Oneapi-Request-Id")
	// Never fall back from a malformed/conflicting header or copy an upstream
	// request ID. Older gateways expose only the legacy gateway header.
	values := standard
	if len(values) == 0 {
		values = legacy
	}
	if len(values) != 1 || !canonicalRequestID.MatchString(values[0]) {
		return
	}
	if len(standard) > 0 && len(legacy) > 0 && (len(legacy) != 1 || legacy[0] != standard[0]) {
		return
	}
	writer.requestID = values[0]
}

func (writer *supportReceiptWriter) writeJSON(value any) {
	encoded, err := json.Marshal(value)
	if err != nil {
		writeJSON(writer.output, value)
		return
	}
	var document map[string]json.RawMessage
	if json.Unmarshal(encoded, &document) != nil || document["ok"] == nil {
		writeJSON(writer.output, value)
		return
	}
	// Project explicit fields rather than removing known secrets from a copy.
	// Existing local paths, free text, parameters and result objects never
	// enter the shareable block.
	summary := writer.summary(document)
	writer.operationsEvent = operationsEvent(writer.command, writer.attempted, document)
	document["support"], _ = json.Marshal(summary)
	if timings := writer.timings.snapshot(); timings != nil {
		document["timings_ms"], _ = json.Marshal(timings)
	}
	writeJSON(writer.output, document)
}

func (writer *supportReceiptWriter) summary(document map[string]json.RawMessage) map[string]any {
	var ok bool
	_ = json.Unmarshal(document["ok"], &ok)
	summary := map[string]any{
		"format": "puretokens-support-v1", "executor_version": executorVersion,
		"os": runtime.GOOS, "arch": runtime.GOARCH,
		"observed_at": time.Now().UTC().Format(time.RFC3339),
		"ok":          ok, "api_request_attempted": writer.attempted,
	}
	switch writer.host {
	case "codex", "claude-code", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae",
		"claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop":
		summary["host"] = writer.host
	}
	switch writer.command {
	case "init", "doctor", "connection", "balance", "models", "model-groups", "preflight", "submit",
		"status", "wait", "content", "resume", "redeliver", "delivered", "evaluate", "audio", "audio-verify", "embedding", "rerank":
		summary["command"] = writer.command
	}
	if writer.attempted {
		summary["request_phase"] = writer.requestPhase
	}
	if writer.httpStatus >= 100 && writer.httpStatus <= 599 {
		summary["http_status"] = writer.httpStatus
		if writer.requestID != "" {
			summary["request_id"] = writer.requestID
		}
	}
	text := func(field string) string {
		var value string
		_ = json.Unmarshal(document[field], &value)
		return value
	}
	if id := text("task_id"); validTaskID(id) && safePublicString(id) == id {
		summary["task_id"] = id
	}
	if code := text("api_error_code"); writer.attempted && text("failure_source") != "task_record" && publicErrorMessages[code] != "" {
		summary["api_error_code"] = code
	}
	for key, allowed := range map[string]string{
		"kind":               "|image|video|music|",
		"failure_phase":      "|validation|submission|status|content|",
		"submission_outcome": "|accepted|rejected|unknown|not_submitted|",
		"next_step":          "|wait|content|deliver|await_user|done|",
		"status":             "|pending|queued|processing|running|in_progress|completed|succeeded|success|failed|cancelled|canceled|expired|error|unknown|",
	} {
		if value := text(key); value != "" && !strings.Contains(value, "|") && strings.Contains(allowed, "|"+value+"|") {
			summary[key] = value
		}
	}
	var reconciliation bool
	if json.Unmarshal(document["reconciliation_required"], &reconciliation) == nil {
		summary["reconciliation_required"] = reconciliation
	}
	// Connection failures use one of the executor's fixed local categories.
	// These are not arbitrary error strings supplied by a server or caller.
	code := text("local_error_code")
	if code == "" {
		code = text("configuration_status")
	}
	if code == "" && writer.command == "doctor" {
		var connection initReceipt
		if json.Unmarshal(document["connection"], &connection) == nil {
			code = connection.ConfigurationStatus
		}
	}
	if supportLocalErrorCodes[code] {
		summary["local_error_code"] = code
	}
	return summary
}

var supportLocalErrorCodes = map[string]bool{
	"active_connection_endpoint_missing": true, "active_connection_endpoint_unsupported": true,
	"active_connection_not_puretokens": true, "active_connection_unavailable": true,
	"active_connection_credential_missing": true, "active_connection_format_unsupported": true,
	"active_connection_selection_unconfirmed": true, "host_credential_adapter_unavailable": true,
	"active_connection_record_missing": true, "active_connection_ambiguous": true,
	"workbuddy_record_format_unsupported": true, "workbuddy_connection_not_found": true,
	"workbuddy_record_missing": true, "workbuddy_record_unreadable": true,
	"workbuddy_connection_ambiguous": true, "workbuddy_credential_missing": true,
	"balance_usage_unavailable": true, "balance_usage_auth_rejected": true, "balance_unit_metadata_unavailable": true,
	"download_integrity_unverified": true, "task_record_write_failed": true,
	"task_record_unavailable": true, "retry_wait_required": true, "output_directory_unavailable": true,
	"original_count_unknown": true, "media_download_incomplete": true, "media_size_limit_exceeded": true,
	"invalid_media_content": true, "output_permission_denied": true, "media_download_timeout": true, "output_file_conflict": true,
	"api_response_unreadable": true, "api_network_unavailable": true, "api_identity_rejected": true,
	"api_identity_unreadable": true, "credential_unverified": true, "api_identity_unconfirmed": true,
	"task_response_unreadable": true, "task_id_missing": true, "task_id_invalid": true,
	"evaluation_request_invalid": true, "evaluation_response_invalid": true, "evaluation_request_failed": true,
	"audio_artifact_invalid": true, "audio_request_invalid": true, "audio_request_failed": true, "audio_response_invalid": true, "audio_output_unavailable": true,
	"embedding_request_invalid": true, "embedding_request_failed": true, "embedding_response_invalid": true, "embedding_output_unavailable": true,
	"rerank_request_invalid": true, "rerank_request_failed": true, "rerank_response_invalid": true, "rerank_output_unavailable": true,
}
