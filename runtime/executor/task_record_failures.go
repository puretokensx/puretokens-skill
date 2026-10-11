package main

// Task-result evidence only. No raw messages, request text, HTTP headers or
// provider metadata survive in a continuation artifact.
type taskFailure struct {
	Category     string `json:"category"`
	Phase        string `json:"phase"`
	TaskStatus   string `json:"task_status"`
	APIErrorCode string `json:"api_error_code,omitempty"`
}

func taskFailureCategory(code string) string {
	switch code {
	case "insufficient_quota":
		return "quota"
	case "insufficient_balance":
		return "balance"
	case "auth_required", "invalid_api_key", "authentication_error":
		return "authentication"
	case "model_not_allowed":
		return "model_permission"
	case "permission_denied", "query_denied":
		return "permission"
	case "model_not_found":
		return "model_unavailable"
	case "invalid_request", "invalid_request_error", "invalid_parameter", "unsupported_parameter", "unsupported_operation":
		return "invalid_request"
	case "content_policy_violation", "content_moderation", "moderation_blocked":
		return "content_policy"
	case "rate_limit", "rate_limited", "rate_limit_exceeded":
		return "rate_limit"
	case "upstream_unavailable", "service_unavailable", "internal_error":
		return "service_unavailable"
	case "task_not_found", "task_expired":
		return "task_unavailable"
	default:
		return "unknown"
	}
}

func taskFailureFromReceipt(result receipt) *taskFailure {
	if !result.taskFailureConfirmed || result.OK || !terminalFailure(result.Status) ||
		(result.FailurePhase != "submission" && result.FailurePhase != "status") {
		return nil
	}
	failure := &taskFailure{Category: taskFailureCategory(result.APIErrorCode), Phase: result.FailurePhase, TaskStatus: result.Status}
	if failure.Category != "unknown" {
		failure.APIErrorCode = result.APIErrorCode
	}
	return failure
}

func validTaskFailure(failure *taskFailure, status string) bool {
	if failure == nil {
		return true
	}
	return terminalFailure(status) && failure.TaskStatus == status &&
		(failure.Phase == "submission" || failure.Phase == "status") &&
		failure.Category == taskFailureCategory(failure.APIErrorCode) &&
		(failure.Category != "unknown" || failure.APIErrorCode == "")
}
