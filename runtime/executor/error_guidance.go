package main

import "strings"

// This is a local orchestration hint, never authorization to create a task.
// Recompute after record persistence/handoff checks, which can change success.
func guideReceipt(result receipt) receipt {
	result = guideFailure(result)
	result.NextStep = ""
	if result.Kind == "" {
		return result
	}
	switch {
	case !result.OK:
		result.WaitOutcome = ""
		result.NextStep = "await_user"
	case result.DeliveryStatus == "delivered":
		result.NextStep = "done"
	case len(result.DownloadedPaths) > 0:
		result.NextStep = "deliver"
	case result.TaskID == "":
		result.NextStep = "await_user"
	case result.ReconciliationRequired || terminalFailure(result.Status) || result.Status == "unknown":
		result.NextStep = "await_user"
	case terminalSuccess(result.Status):
		result.NextStep = "content"
	case result.WaitOutcome == "retry_deferred" || result.WaitWindowsCompleted >= 2:
		result.NextStep = "await_user"
	default:
		result.NextStep = "wait"
	}
	return result
}

// Curated actions only. Never derive a diagnosis or an action from raw server text.
func apiErrorAction(code string, status int) string {
	switch code {
	case "insufficient_quota":
		return "Visit the Pure Tokens official Wallet & Recharge page at https://console.puretokensx.com/wallet to check your wallet and recharge as needed. Also check this API key's allowance: a quota error does not identify which limit is exhausted, and recharging does not necessarily resolve a key limit."
	case "insufficient_balance":
		return "Visit the Pure Tokens official Wallet & Recharge page at https://console.puretokensx.com/wallet to check your balance and recharge as needed. No missing amount or successful retry is guaranteed."
	case "auth_required", "invalid_api_key", "authentication_error":
		return "Check the selected Pure Tokens connection in the host's own settings and its credential validity. Do not paste credentials into chat."
	case "permission_denied", "query_denied", "model_not_allowed":
		return "Check this API key's permission for the requested endpoint or model; contact Pure Tokens support if the permission should be present. Do not replace the connection or switch models automatically."
	case "model_not_found":
		return "Check the exact model ID and its declared availability with an explicit model query. Choose an alternative only with the user's agreement."
	case "invalid_request", "invalid_request_error", "invalid_parameter", "unsupported_parameter", "unsupported_operation":
		return "Check the selected model's declared fields, values and operation, then correct the request while preserving the user's intent. Use preflight only if a separate check is requested."
	case "content_policy_violation", "content_moderation", "moderation_blocked":
		return "Explain the content rejection and ask the user to revise the request. Do not change models or rewrite it automatically to bypass the rejection."
	case "rate_limit", "rate_limited", "rate_limit_exceeded":
		return "Honor the returned retry_not_before or Retry-After before any further read. If none was returned, offer a later retry without inventing a wait time."
	case "upstream_unavailable", "service_unavailable", "internal_error":
		return "The service could not complete this request. Offer a later retry or contact Pure Tokens support if it persists; do not infer a credential or balance problem."
	case "task_not_found", "task_expired":
		return "Check the original task ID, media kind and receipt. Do not search for other tasks or assume a refund; contact Pure Tokens support if the original task should remain available."
	}
	// HTTP status describes this request only; never invent a public API code.
	switch {
	case status == 401:
		return "This endpoint rejected authentication. Check the current connection in the host's settings without sharing credentials; this does not establish a balance problem."
	case status == 403:
		return "This endpoint denied the request. Check its permissions or contact Pure Tokens support; do not assume the credential itself is invalid."
	case status == 429:
		return "Honor any returned retry_not_before or Retry-After. Otherwise offer a later retry; do not infer insufficient balance from HTTP 429."
	case status >= 500:
		return "The service could not complete the request. Offer a later retry or contact Pure Tokens support; do not infer a credential or balance problem."
	}
	return ""
}

func guideFailure(result receipt) receipt {
	if result.OK {
		return result
	}
	if result.taskFailureConfirmed && result.FailureCategory == "" {
		result.FailureCategory = taskFailureCategory(result.APIErrorCode)
	}
	// Preserve command-specific local diagnoses, especially the balance exception.
	if result.LocalErrorCode == "" {
		if action := apiErrorAction(result.APIErrorCode, result.HTTPStatus); action != "" {
			result.NextAction = action
		}
	}
	var boundary string
	switch {
	case result.SubmissionOutcome == "unknown":
		boundary = "Submission may have started. Do not repeat the POST automatically or infer billing; a new paid request requires explicit user agreement."
	case result.TaskID != "" && result.ReconciliationRequired:
		boundary = "Keep this task ID. Stop automatic waiting; on explicit continuation check this same task once. Never submit a replacement or infer a refund."
	case result.TaskID != "" && terminalFailure(result.Status):
		boundary = "Keep this task ID and its failed state. Do not continue polling a terminal failure or automatically create another task."
	case result.TaskID != "":
		boundary = "Keep this task ID and the original operation/count. Continue only this same task when appropriate; do not submit a replacement."
	case result.Kind != "" && result.SubmissionOutcome == "rejected":
		boundary = "The submission was rejected. Do not automatically repeat it, switch models or create another paid task."
	}
	if boundary != "" && !strings.Contains(result.NextAction, boundary) {
		result.NextAction = strings.TrimSpace(result.NextAction + " " + boundary)
	}
	return result
}
