package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMediaValidationExplainsLocalFieldWithoutRepeatingInput(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "recorded"}[recorded], func(t *testing.T) {
			svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "image", "gpt-image-2.6"))
			request := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2.6", Prompt: "private-prompt", Parameters: map[string]any{"quality": "private-value"}}
			var out bytes.Buffer
			var err error
			if recorded {
				err = executeRecordedTask("submit", filepath.Join(t.TempDir(), "task.json"), request, -1, "", &out, svc)
			} else {
				data, _ := json.Marshal(request)
				err = executeTask(bytes.NewReader(data), &out, svc)
			}
			var result map[string]any
			if json.Unmarshal(out.Bytes(), &result) != nil || err == nil || transport.calls != 0 {
				t.Fatalf("expected local failure without requests: %s", out.String())
			}
			issue, ok := result["validation"].(map[string]any)
			if !ok || issue["field"] != "quality" || issue["reason"] != "enum" || result["submission_outcome"] != "not_submitted" {
				t.Fatalf("missing actionable validation: %s", out.String())
			}
			if len(issue["allowed"].([]any)) != 4 || result["local_error_code"] != "media_request_invalid" || result["next_step"] != "correct_request" {
				t.Fatalf("missing reviewed allowed values: %s", out.String())
			}
			if strings.Contains(out.String(), "private-") {
				t.Fatal("input leaked")
			}
		})
	}
}

func TestMediaValidationPreservesAnExistingRecord(t *testing.T) {
	svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "image", "gpt-image-2.6"))
	path := filepath.Join(t.TempDir(), "task.json")
	original := []byte(`{"submission_outcome":"unknown"}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	request := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2.6", Prompt: "private-prompt", Parameters: map[string]any{"quality": "invalid"}}
	if executeRecordedTask("submit", path, request, -1, "", &out, svc) == nil {
		t.Fatal("expected existing-record refusal")
	}
	after, _ := os.ReadFile(path)
	if transport.calls != 0 || !bytes.Equal(after, original) || strings.Contains(out.String(), "correct_request") || strings.Contains(out.String(), "not_submitted") {
		t.Fatalf("existing task became a new request: %s", out.String())
	}
}

func TestMediaValidationNeverEchoesUnreviewedProfileFieldsOrValues(t *testing.T) {
	for _, field := range []string{"private-field", "quality"} {
		profile := readInstalledProfileFixture(t, "image", "gpt-image-2.6")
		profile.Parameters.Properties[field] = map[string]any{"type": "string", "enum": []any{"private-catalog-value"}}
		svc, transport := catalogProfileService(t, profile)
		svc.profilesRoot = writeInstalledProfileFixture(t, profile)
		var out bytes.Buffer
		request := taskRequest{Kind: "image", Operation: "generate", Model: profile.ID, Prompt: "private-prompt", Parameters: map[string]any{field: "private-input"}}
		data, _ := json.Marshal(request)
		if executeTask(bytes.NewReader(data), &out, svc) == nil || transport.calls != 0 {
			t.Fatal("expected local refusal")
		}
		if strings.Contains(out.String(), "private-") {
			t.Fatalf("unreviewed profile data leaked: %s", out.String())
		}
	}
}

func TestMediaValidationDoesNotAuthorizeReplacementOfExistingTask(t *testing.T) {
	var out bytes.Buffer
	request := taskRequest{Kind: "image", Operation: "continue", TaskID: "original-task", RequestedCount: 99}
	data, _ := json.Marshal(request)
	if executeTask(bytes.NewReader(data), &out, service{}) == nil {
		t.Fatal("expected failure")
	}
	if strings.Contains(out.String(), "not_submitted") || strings.Contains(out.String(), `"validation":`) {
		t.Fatalf("continuation mislabeled as new request: %s", out.String())
	}
}
