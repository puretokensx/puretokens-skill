package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportRequestIDHeaderBoundary(t *testing.T) {
	id := "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99"
	for _, tc := range []struct {
		name     string
		standard []string
		legacy   []string
		want     string
	}{
		{"standard", []string{id}, nil, id},
		{"legacy", nil, []string{id}, id},
		{"both_equal", []string{id}, []string{id}, id},
		{"both_conflict", []string{id}, []string{"45a3d6e3-42dd-4e67-8b08-e5c6058b2f99"}, ""},
		{"malformed_no_fallback", []string{"private-content"}, []string{id}, ""},
		{"duplicate", []string{id, id}, nil, ""},
		{"duplicate_legacy", nil, []string{id, id}, ""},
		{"duplicate_legacy_with_standard", []string{id}, []string{id, id}, ""},
		{"uppercase", []string{strings.ToUpper(id)}, nil, ""},
		{"combined", []string{id + "," + id}, nil, ""},
		{"padded", []string{" " + id}, nil, ""},
		{"missing", nil, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := &supportReceiptWriter{}
			writer.beginRequest(http.MethodGet, "/v1/images/task_fixture/content?index=0")
			header := http.Header{}
			for _, value := range tc.standard {
				header.Add("X-Request-ID", value)
			}
			for _, value := range tc.legacy {
				header.Add("X-Oneapi-Request-Id", value)
			}
			header.Set("X-Upstream-Request-Id", id)
			writer.observeResponse(&http.Response{StatusCode: 200, Header: header})
			if writer.requestID != tc.want || writer.requestPhase != "content" {
				t.Fatalf("wrong safe correlation: %q, %q", writer.requestID, writer.requestPhase)
			}
		})
	}
}

func TestSupportRequestPhaseAndDoctorFailure(t *testing.T) {
	writer := &supportReceiptWriter{command: "doctor"}
	for _, tc := range []struct{ method, path, phase string }{
		{"GET", "/v1", "identity"},
		{"GET", "/v1/media/models", "catalog"},
		{"GET", balanceUsagePath, "balance_usage"},
		{"GET", balanceUnitPath, "balance_metadata"},
		{"POST", "/v1/images/generations", "submission"},
		{"GET", "/v1/images/content-task", "status"},
		{"GET", "/v1/videos/task_fixture/content?index=0", "content"},
	} {
		writer.beginRequest(tc.method, tc.path)
		if writer.requestPhase != tc.phase {
			t.Fatalf("%s: want %s, got %s", tc.path, tc.phase, writer.requestPhase)
		}
	}
	for _, code := range []string{"active_connection_selection_unconfirmed", "private-error"} {
		var output bytes.Buffer
		writer = &supportReceiptWriter{output: &output, command: "doctor"}
		writeJSON(writer, doctorReceipt{Connection: &initReceipt{ConfigurationStatus: code, Message: "private-message"}})
		var document map[string]any
		if err := json.Unmarshal(output.Bytes(), &document); err != nil {
			t.Fatal(err)
		}
		summary := jsonObject(document["support"])
		if code == "private-error" {
			if summary["local_error_code"] != nil {
				t.Fatal("doctor exposed untrusted connection detail")
			}
		} else if summary["local_error_code"] != code || summary["api_request_attempted"] != false {
			t.Fatalf("doctor lost safe local category: %v", summary)
		}
	}
}

func TestSupportInvalidHTTPStatusOmitsCorrelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99")
		w.WriteHeader(600)
		io.WriteString(w, `{"error":{"code":"service_unavailable"}}`)
	}))
	defer server.Close()
	var output bytes.Buffer
	writer := &supportReceiptWriter{output: &output, command: "status", host: "codex"}
	svc := fixtureService(server)
	svc.support = writer
	_, status, _, _, _, err := svc.request(context.Background(), http.MethodGet, "/v1/images/task_fixture", nil, "")
	if err != nil || status != 600 {
		t.Fatalf("nonstandard HTTP response was not exercised: status=%d err=%v", status, err)
	}
	writeJSON(writer, map[string]any{"ok": false})
	var document map[string]any
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	summary := jsonObject(document["support"])
	if summary["api_request_attempted"] != true || summary["request_phase"] != "status" ||
		summary["http_status"] != nil || summary["request_id"] != nil {
		t.Fatalf("invalid HTTP status left orphaned correlation: %v", summary)
	}
}

func TestSupportProjectionExcludesFreeTextAndPrivateFields(t *testing.T) {
	var output bytes.Buffer
	writer := &supportReceiptWriter{output: &output, host: "codex", command: "submit"}
	writer.beginRequest(http.MethodPost, "/v1/images/generations")
	writeJSON(writer, map[string]any{
		"ok": false, "task_id": "task_fixture", "kind": "image", "status": "pending",
		"failure_phase": "submission", "api_error_code": "service_unavailable",
		"submission_outcome": "unknown", "next_step": "await_user",
		"prompt": "private-prompt", "model": "private-label", "parameters": map[string]any{"image": "private-url"},
		"error_message": "private-error", "next_action": "private-action", "local_error_code": "private-code",
		"record_path": "/private/record", "downloaded_paths": []string{"/private/output"},
		"configuration": "private-config", "result": map[string]any{"raw": "private-response"},
	})
	var document map[string]json.RawMessage
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	summary := string(document["support"])
	for _, fragment := range []string{"private", "prompt", "parameters", "error_message", "next_action", "record_path", "downloaded_paths", "configuration", `"model"`} {
		if strings.Contains(summary, fragment) {
			t.Fatalf("support projection leaked %s", fragment)
		}
	}
	for _, fragment := range []string{`"task_id":"task_fixture"`, `"host":"codex"`, `"api_error_code":"service_unavailable"`, `"submission_outcome":"unknown"`} {
		if !strings.Contains(summary, fragment) {
			t.Fatalf("support projection lost %s", fragment)
		}
	}
}

func TestSupportUnknownSubmissionClearsEarlierCatalogCorrelation(t *testing.T) {
	var output bytes.Buffer
	writer := &supportReceiptWriter{output: &output, host: "codex", command: "submit"}
	calls := 0
	svc := service{baseURL: apiOrigin, support: writer, client: &http.Client{Transport: pollingFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"55a3d6e3-42dd-4e67-8b08-e5c6058b2f99"}}, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))}, nil
		}
		return nil, errors.New("private-network-failure")
	})}}
	if _, _, _, _, _, err := svc.request(context.Background(), http.MethodGet, "/v1/media/models", nil, ""); err != nil {
		t.Fatal(err)
	}
	err := executePreparedTask(writer, svc, taskRequest{Kind: "image", Operation: "generate", Model: "fixture", Prompt: "private-prompt"})
	if err == nil || calls != 2 {
		t.Fatalf("expected one catalog read and one failed submission, calls=%d err=%v", calls, err)
	}
	var result struct {
		Support map[string]any `json:"support"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Support["request_id"] != nil || result.Support["http_status"] != nil ||
		result.Support["api_request_attempted"] != true || result.Support["request_phase"] != "submission" ||
		result.Support["submission_outcome"] != "unknown" {
		t.Fatalf("stale or misleading support evidence: %v", result.Support)
	}
}

func TestSupportRecordedLifecycleDoesNotPersistOrReplayDiagnostics(t *testing.T) {
	png := fixturePNG(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-ID", "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99")
		if r.Method == http.MethodPost {
			io.WriteString(w, `{"id":"task_fixture","status":"completed"}`)
		} else {
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
		}
	}))
	defer server.Close()
	directory := t.TempDir()
	recordPath := filepath.Join(directory, "task.json")
	request := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2", Prompt: "private-prompt"}
	svc := fixtureService(server)
	svc.profilesRoot = "../../skills"
	for _, command := range []string{"submit", "content", "resume"} {
		var output bytes.Buffer
		writer := &supportReceiptWriter{output: &output, host: "codex", command: command}
		svc.support = writer
		svc.localRecoveryOnly = command == "resume"
		if command != "submit" {
			record, err := loadTaskRecord(recordPath)
			if err != nil {
				t.Fatal(err)
			}
			request = record.request()
		}
		err := executeRecordedTask(command, recordPath, request, 0, directory, writer, svc)
		if err != nil {
			t.Fatalf("%s: %v", command, err)
		}
		var envelope map[string]any
		if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		summary := jsonObject(envelope["support"])
		if command == "resume" {
			if summary["api_request_attempted"] != false || summary["request_id"] != nil || calls != 2 {
				t.Fatalf("local recovery invented HTTP evidence: %v, calls=%d", summary, calls)
			}
		} else if summary["request_id"] == nil || summary["api_request_attempted"] != true {
			t.Fatalf("lost real correlation: %v", summary)
		}
		data, err := os.ReadFile(recordPath)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(`"support"`)) || bytes.Contains(data, []byte(`"request_id"`)) {
			t.Fatal("support metadata must not change persistent task records")
		}
	}
}

func TestSupportLocalCLIRejectsWithoutNetworkOrUntrustedMetadata(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"status", "--host", "private-host"}, strings.NewReader(`{"kind":"image","task_id":"task_fixture"}`), &output)
	if err == nil {
		t.Fatal("unrecognized host was accepted")
	}
	var document map[string]any
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	summary := jsonObject(document["support"])
	if summary["api_request_attempted"] != false || summary["host"] != nil ||
		summary["local_error_code"] != "host_credential_adapter_unavailable" ||
		summary["request_id"] != nil || summary["task_id"] != "task_fixture" {
		t.Fatalf("unsafe local failure evidence: %v", summary)
	}
}
