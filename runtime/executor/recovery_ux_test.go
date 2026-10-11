package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordedFailureKeepsClassificationAcrossResume(t *testing.T) {
	for _, code := range []string{"content_policy_violation", "model_not_allowed", "insufficient_quota", "insufficient_balance", "upstream_unavailable", "private_provider_secret"} {
		t.Run(code, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				fmt.Fprintf(w, `{"id":"paid-task","status":"failed","error":{"code":%q,"message":"private upstream diagnostic"}}`, code)
			}))
			defer server.Close()
			path := filepath.Join(t.TempDir(), "task.json")
			if err := saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "pending", RequestedCount: 1}, true); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := executeRecordedTask("status", path, taskRequest{}, 0, "", &out, fixtureService(server)); err == nil {
				t.Fatal("expected terminal failure")
			}
			saved, _ := os.ReadFile(path)
			if strings.Contains(string(saved), "private") || !bytes.Contains(saved, []byte(`"last_failure"`)) {
				t.Fatalf("record must keep only curated classification: %s", saved)
			}
			out.Reset()
			if err := run([]string{"resume", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out); err == nil {
				t.Fatal("terminal failure must remain failed")
			}
			got := decodeReceipt(t, &out)
			wantCode := code
			if code == "private_provider_secret" {
				wantCode = ""
			}
			var envelope map[string]any
			json.Unmarshal(out.Bytes(), &envelope)
			if got.APIErrorCode != wantCode || envelope["failure_source"] != "task_record" || envelope["failure_category"] == nil || calls != 1 {
				t.Fatalf("failure classification lost or probed credentials: %s", out.Bytes())
			}
		})
	}
}

func TestExplicitRedeliveryKeepsHistoryAndCurrentHandoff(t *testing.T) {
	image := fixturePNG(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/v1/images/paid-task/content" {
			t.Fatalf("redelivery changed task or submitted: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(image)
	}))
	defer server.Close()
	svc := fixtureService(server)
	dir := t.TempDir()
	path := filepath.Join(dir, "task.json")
	saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", OriginalOperation: "generate", RequestedCount: 1, Status: "completed"}, true)
	var out bytes.Buffer
	invoke := func(command, directory string) receipt {
		t.Helper()
		out.Reset()
		if err := executeRecordedTask(command, path, taskRequest{}, 0, directory, &out, svc); err != nil {
			t.Fatalf("%s: %v: %s", command, err, out.Bytes())
		}
		return decodeReceipt(t, &out)
	}
	first := invoke("content", dir)
	invoke("delivered", "")
	if got := invoke("resume", ""); got.NextStep != "done" || len(got.DownloadedPaths) != 0 || calls != 1 {
		t.Fatal("ordinary resume repeated completed delivery")
	}
	out.Reset()
	if err := executeRecordedTask("content", path, taskRequest{}, 0, t.TempDir(), &out, svc); err == nil || calls != 1 {
		t.Fatal("ordinary content bypassed explicit redelivery or downloaded again")
	}
	out.Reset()
	if err := run([]string{"redeliver", "--host", "fixture-unsupported", "--record", path, "--index", "0"}, strings.NewReader(""), &out); err != nil {
		t.Fatalf("local redelivery unexpectedly used credentials: %v %s", err, out.Bytes())
	}
	got := decodeReceipt(t, &out)
	if got.NextStep != "deliver" || len(got.DownloadedPaths) != 1 || got.DownloadedPaths[0] != first.DownloadedPaths[0] || calls != 1 {
		t.Fatalf("local explicit redelivery: %+v", got)
	}
	if got := invoke("resume", ""); got.NextStep != "deliver" || calls != 1 {
		t.Fatal("pending reattachment was lost across resume")
	}
	invoke("delivered", "")
	if err := os.Remove(first.DownloadedPaths[0]); err != nil {
		t.Fatal(err)
	}
	got = invoke("redeliver", t.TempDir())
	if got.NextStep != "deliver" || got.DeliveryStatus != "downloaded_awaiting_host_delivery" || len(got.DownloadedPaths) != 1 || calls != 2 {
		t.Fatalf("new-computer redelivery is not current handoff: %+v calls=%d", got, calls)
	}
	record, err := loadTaskRecord(path)
	if err != nil || len(record.Delivered) != 1 || record.Delivered[0] != 0 {
		t.Fatal("historical delivery was erased")
	}
	if got := invoke("delivered", ""); got.NextStep != "done" {
		t.Fatal("redelivery acknowledgement did not finish")
	}
}

func TestAdaptiveAspectRatioSurvivesReceiptAndRecord(t *testing.T) {
	request := taskRequest{Kind: "video", Operation: "generate", Model: "minimax_h3", Prompt: "fixture", Parameters: map[string]any{"aspect_ratio": "adaptive"}}
	if err := prepareProfileRequest(&request, profileService()); err != nil {
		t.Fatal(err)
	}
	result := taskReceipt(request, "paid-task", "pending")
	path := filepath.Join(t.TempDir(), "task.json")
	record := recordFromRequest(request)
	record.TaskID, record.Status = result.TaskID, result.Status
	if err := saveTaskRecord(path, record, true); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadTaskRecord(path)
	if err != nil || result.Parameters["aspect_ratio"] != "adaptive" || loaded.Parameters["aspect_ratio"] != "adaptive" {
		t.Fatalf("legal adaptive ratio lost: receipt=%v record=%v err=%v", result.Parameters, loaded.Parameters, err)
	}
}

func TestRecordedFailureDoesNotConfuseReadErrorsWithTaskResult(t *testing.T) {
	for _, status := range []int{401, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				io.WriteString(w, `{"error":{"code":"invalid_api_key","message":"temporary read failure"}}`)
			}))
			defer server.Close()
			path := filepath.Join(t.TempDir(), "task.json")
			original := &taskFailure{Category: "content_policy", Phase: "status", TaskStatus: "failed", APIErrorCode: "content_policy_violation"}
			saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "failed", RequestedCount: 1, LastFailure: original}, true)
			var out bytes.Buffer
			executeRecordedTask("status", path, taskRequest{}, 0, "", &out, fixtureService(server))
			record, err := loadTaskRecord(path)
			if err != nil || record.LastFailure == nil || *record.LastFailure != *original {
				t.Fatalf("HTTP read error replaced task failure: %+v %v", record.LastFailure, err)
			}
		})
	}
}

func TestRecordedFailureRetainsKnownCauseUntilSpecificEvidenceOrStateChange(t *testing.T) {
	for _, tc := range []struct {
		name, status, code string
		want               *taskFailure
	}{
		{"same-state-no-code", "failed", "", &taskFailure{Category: "content_policy", Phase: "status", TaskStatus: "failed", APIErrorCode: "content_policy_violation"}},
		{"same-state-unreviewed-code", "failed", "private_provider_secret", &taskFailure{Category: "content_policy", Phase: "status", TaskStatus: "failed", APIErrorCode: "content_policy_violation"}},
		{"same-state-new-known-cause", "failed", "insufficient_balance", &taskFailure{Category: "balance", Phase: "status", TaskStatus: "failed", APIErrorCode: "insufficient_balance"}},
		{"changed-terminal-state", "expired", "", &taskFailure{Category: "unknown", Phase: "status", TaskStatus: "expired"}},
		{"changed-success-state", "completed", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/v1/images/paid-task" {
					t.Errorf("unexpected task request: %s %s", r.Method, r.URL.Path)
				}
				response := map[string]any{"id": "paid-task", "status": tc.status}
				if tc.code != "" {
					response["error"] = map[string]string{"code": tc.code}
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			path := filepath.Join(t.TempDir(), "task.json")
			original := &taskFailure{Category: "content_policy", Phase: "status", TaskStatus: "failed", APIErrorCode: "content_policy_violation"}
			if err := saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "failed", RequestedCount: 1, LastFailure: original}, true); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err := executeRecordedTask("status", path, taskRequest{}, 0, "", &out, fixtureService(server))
			if (err != nil) != terminalFailure(tc.status) {
				t.Fatalf("wrong terminal result: %v %s", err, out.Bytes())
			}
			record, err := loadTaskRecord(path)
			if err != nil || record.Status != tc.status || calls != 1 {
				t.Fatalf("same-task refresh failed: %+v %v calls=%d", record, err, calls)
			}
			if (record.LastFailure == nil) != (tc.want == nil) || (tc.want != nil && *record.LastFailure != *tc.want) {
				t.Fatalf("unexpected saved failure: got=%+v want=%+v", record.LastFailure, tc.want)
			}
			if tc.want != nil {
				out.Reset()
				if err := run([]string{"resume", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out); err == nil {
					t.Fatal("terminal failure must remain failed")
				}
				got := decodeReceipt(t, &out)
				if got.FailureCategory != tc.want.Category || got.APIErrorCode != tc.want.APIErrorCode || got.FailureSource != "task_record" || calls != 1 {
					t.Fatalf("resume lost the saved failure or made a request: %+v calls=%d", got, calls)
				}
			}
		})
	}
}

func TestImmediateTerminalSubmissionKeepsCuratedFailure(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		if r.Method != http.MethodPost {
			t.Fatal("unexpected discovery or retry")
		}
		io.WriteString(w, `{"id":"paid-task","status":"failed","error":{"code":"insufficient_balance","message":"private diagnostic"}}`)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "task.json")
	request := taskRequest{Kind: "image", Operation: "generate", Model: "fixture-model", Prompt: "fixture"}
	var out bytes.Buffer
	if err := executeRecordedTask("submit", path, request, 0, "", &out, fixtureService(server)); err == nil {
		t.Fatal("expected failure")
	}
	record, err := loadTaskRecord(path)
	if err != nil || record.LastFailure == nil || record.LastFailure.Phase != "submission" || record.LastFailure.Category != "balance" || posts != 1 {
		t.Fatalf("accepted terminal task evidence missing: %+v %v", record.LastFailure, err)
	}
}

func TestVideoAndMusicRedeliveryStayOnSameTask(t *testing.T) {
	for _, kind := range []string{"video", "music"} {
		t.Run(kind, func(t *testing.T) {
			media, mime := fixtureWebM(false, false), "video/webm"
			record := taskRecord{Format: taskRecordFormat, Kind: kind, TaskID: "paid-task", Status: "completed", RequestedCount: 1}
			if kind == "music" {
				media, mime = audioMP3Fixture(), "audio/mpeg"
				record.Model, record.OriginalOperation = "stepaudio-3-music-preview", "generate"
				record.Parameters = map[string]any{"instrumental": true, "response_format": "mp3"}
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != contentPath(kind, "paid-task", 0) {
					t.Fatalf("wrong task or route: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", mime)
				w.Write(media)
			}))
			defer server.Close()
			dir := t.TempDir()
			path := filepath.Join(dir, "task.json")
			saveTaskRecord(path, record, true)
			var out bytes.Buffer
			svc := fixtureService(server)
			for _, command := range []string{"content", "delivered"} {
				out.Reset()
				if err := executeRecordedTask(command, path, taskRequest{}, 0, dir, &out, svc); err != nil {
					t.Fatal(err)
				}
			}
			out.Reset()
			if err := run([]string{"redeliver", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out); err != nil {
				t.Fatalf("local redelivery failed: %v %s", err, out.Bytes())
			}
			if got := decodeReceipt(t, &out); got.NextStep != "deliver" || got.RedeliveryIndex == nil || *got.RedeliveryIndex != 0 || calls != 1 {
				t.Fatalf("redelivery lost same-task output: %+v", got)
			}
		})
	}
}

func TestLegacyFailedRecordRemainsUnknownWithoutCredentialRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.json")
	saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "failed", RequestedCount: 1}, true)
	var out bytes.Buffer
	run([]string{"resume", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out)
	got := decodeReceipt(t, &out)
	if got.FailureCategory != "unknown" || got.FailureSource != "task_record" || got.APIErrorCode != "" || got.LocalErrorCode != "" || !strings.Contains(got.NextAction, "before proposing") {
		t.Fatalf("legacy failure invented cause or probed credentials: %+v", got)
	}
}

func TestTaskRecordRejectsUncuratedOrMismatchedFailureEvidence(t *testing.T) {
	for _, failure := range []*taskFailure{
		{Category: "unknown", Phase: "status", TaskStatus: "failed", APIErrorCode: "private_provider_secret"},
		{Category: "content_policy", Phase: "status", TaskStatus: "failed", APIErrorCode: "insufficient_quota"},
		{Category: "quota", Phase: "status", TaskStatus: "pending", APIErrorCode: "insufficient_quota"},
		{Category: "quota", Phase: "content", TaskStatus: "failed", APIErrorCode: "insufficient_quota"},
	} {
		path := filepath.Join(t.TempDir(), "task.json")
		saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "failed", LastFailure: failure}, true)
		if _, err := loadTaskRecord(path); err == nil {
			t.Fatalf("unsafe recorded evidence accepted: %+v", failure)
		}
	}
}

func TestRedeliveryPreservesChangedFileAndPendingIndex(t *testing.T) {
	data := fixturePNG(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet {
			t.Fatal("redelivery must never submit")
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}))
	defer server.Close()
	svc := fixtureService(server)
	dir := t.TempDir()
	path := filepath.Join(dir, "task.json")
	saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "completed", RequestedCount: 2}, true)
	var out bytes.Buffer
	executeRecordedTask("content", path, taskRequest{}, 0, dir, &out, svc)
	first := decodeReceipt(t, &out)
	out.Reset()
	executeRecordedTask("delivered", path, taskRequest{}, 0, "", &out, svc)
	os.WriteFile(first.DownloadedPaths[0], []byte("user-changed-file"), 0600)
	out.Reset()
	if err := executeRecordedTask("redeliver", path, taskRequest{}, 0, dir, &out, svc); err == nil || calls != 1 {
		t.Fatal("changed file was overwritten or fetched into occupied output")
	}
	if saved, _ := os.ReadFile(first.DownloadedPaths[0]); string(saved) != "user-changed-file" {
		t.Fatal("user file changed")
	}
	out.Reset()
	if err := executeRecordedTask("redeliver", path, taskRequest{}, 0, t.TempDir(), &out, svc); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := executeRecordedTask("redeliver", path, taskRequest{}, 1, "", &out, svc); err == nil || calls != 2 {
		t.Fatal("another index replaced pending explicit handoff")
	}
	record, err := loadTaskRecord(path)
	if err != nil || record.PendingRedeliveryIndex == nil || *record.PendingRedeliveryIndex != 0 || len(record.Delivered) != 1 {
		t.Fatalf("pending/history state lost: %+v %v", record, err)
	}
}

func TestRedeliveryRejectsNewOrUnfinishedRequestsBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{
		{"redeliver", "--host", "fixture-unsupported"},
		{"redeliver", "--host", "fixture-unsupported", "--request", "/missing.json"},
	} {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err == nil || strings.Contains(out.String(), "host_credential_adapter_unavailable") {
			t.Fatalf("invalid redelivery reached credentials: %s", out.Bytes())
		}
	}
	for _, state := range []string{"pending", "failed", "completed"} {
		path := filepath.Join(t.TempDir(), "task.json")
		record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: state, RequestedCount: 1, ReconciliationRequired: state == "completed"}
		saveTaskRecord(path, record, true)
		var out bytes.Buffer
		if err := run([]string{"redeliver", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out); err == nil || strings.Contains(out.String(), "host_credential_adapter_unavailable") {
			t.Fatalf("unfinished redelivery reached credentials: %s", out.Bytes())
		}
	}
}

// Keep an explicit third window usable without changing its saturated budget.
func TestExplicitThirdWindowCanFinish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"paid-task","status":"completed"}`)
	}))
	defer server.Close()
	svc := fixtureService(server)
	path := filepath.Join(t.TempDir(), "task.json")
	saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "pending", RequestedCount: 1, WaitWindowsCompleted: 2}, true)
	var out bytes.Buffer
	if err := executeRecordedTask("resume", path, taskRequest{}, 0, "", &out, svc); err != nil {
		t.Fatal(err)
	}
	if got := decodeReceipt(t, &out); got.NextStep != "content" || got.WaitWindowsCompleted != 2 {
		t.Fatalf("explicit continuation blocked: %+v", got)
	}
}
