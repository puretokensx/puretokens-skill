package main

import (
	"strings"
	"testing"
)

func TestFailureGuidanceCategoriesAndTaskBoundaries(t *testing.T) {
	for code := range publicErrorMessages {
		if apiErrorAction(code, 0) == "" {
			t.Errorf("missing guidance for %s", code)
		}
	}
	tests := []struct {
		name string
		in   receipt
		want string
	}{
		{"quota", receipt{APIErrorCode: "insufficient_quota", SubmissionOutcome: "rejected", Kind: "image"}, "wallet"},
		{"balance", receipt{APIErrorCode: "insufficient_balance"}, "recharge"},
		{"unknown submission", receipt{APIErrorCode: "rate_limit", SubmissionOutcome: "unknown"}, "Do not repeat the POST"},
		{"same task", receipt{APIErrorCode: "service_unavailable", TaskID: "task-1"}, "Keep this task ID"},
		{"terminal", receipt{TaskID: "task-1", Status: "failed"}, "Do not continue polling"},
		{"local exception", receipt{LocalErrorCode: "balance_usage_auth_rejected", HTTPStatus: 401, NextAction: "Keep the working media connection."}, "Keep the working media connection."},
		{"unknown code", receipt{APIErrorCode: "untrusted", NextAction: "existing safe action"}, "existing safe action"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guideFailure(tt.in)
			if !strings.Contains(got.NextAction, tt.want) {
				t.Fatalf("action=%s", got.NextAction)
			}
			if guideFailure(got).NextAction != got.NextAction {
				t.Fatal("guidance is not idempotent")
			}
		})
	}
	for _, code := range []string{"insufficient_quota", "insufficient_balance"} {
		if !strings.Contains(apiErrorAction(code, 0), "https://console.puretokensx.com/wallet") {
			t.Errorf("missing official wallet recharge link for %s", code)
		}
	}
	for _, status := range []int{401, 403, 429, 503} {
		if apiErrorAction("", status) == "" {
			t.Errorf("missing status guidance %d", status)
		}
	}
	if strings.Contains(apiErrorAction("", 429), "recharge") {
		t.Fatal("429 mistaken for balance failure")
	}
}

func TestImageDownloadFailureOffersOnlySameTaskLink(t *testing.T) {
	base := receipt{Kind: "image", TaskID: "image-existing", Status: "completed", RequestedCount: 1, FailurePhase: "content", LocalErrorCode: "media_download_stalled", HTTPStatus: 200}
	if got := guideReceipt(base); got.NextStep != "download_link" {
		t.Fatalf("completed image download did not offer bounded link recovery: %s", got.NextStep)
	}
	for _, change := range []func(*receipt){
		func(r *receipt) { r.Kind = "video" },
		func(r *receipt) { r.Status = "pending" },
		func(r *receipt) { r.ReconciliationRequired = true },
		func(r *receipt) { r.HTTPStatus = 403 },
		func(r *receipt) { r.HTTPStatus = 429 },
		func(r *receipt) { r.LocalErrorCode = "invalid_media_content" },
		func(r *receipt) { r.LocalErrorCode = "media_output_format_mismatch" },
		func(r *receipt) { r.LocalErrorCode = "download_link_unavailable" },
		func(r *receipt) { r.SubmissionOutcome = "unknown" },
		func(r *receipt) { r.RequestedCount = 0 },
		func(r *receipt) { r.RetryAfterSecs = 30 },
		func(r *receipt) { r.APIErrorCode = "permission_denied" },
	} {
		in := base
		change(&in)
		if got := guideReceipt(in); got.NextStep != "await_user" {
			t.Fatalf("unsupported recovery continued: %+v", got)
		}
	}
}
