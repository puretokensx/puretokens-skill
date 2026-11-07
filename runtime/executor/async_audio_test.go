package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAsyncAudioLifecycleUsesDedicatedRouteAndSameTask(t *testing.T) {
	for _, model := range []string{"gemini-3.8-flash-tts", "lyria-3.5", "lyria-3-pro-preview"} {
		t.Run(model, func(t *testing.T) {
			calls := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path)
				switch r.URL.Path {
				case "/v1/tasks/product-yutoux":
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					field := "prompt"
					if model == "gemini-3.8-flash-tts" {
						field = "input"
					}
					if p[field] != "private creative text" || p["model"] != model || p["caption"] != nil || p["group"] != nil {
						t.Error("wrong wire contract")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					io.WriteString(w, `{"id":"audio_fixture","task_id":"audio_fixture","status":"queued"}`)
				case "/v1/tasks/audio_fixture":
					io.WriteString(w, `{"task_id":"audio_fixture","platform":"product-yutoux","status":"SUCCESS"}`)
				case "/v1/tasks/audio_fixture/artifacts/audio/content":
					if model == "gemini-3.8-flash-tts" {
						w.Header().Set("Content-Type", "audio/wav")
						w.Write(audioWAVFixture())
					} else {
						w.Header().Set("Content-Type", "audio/mpeg")
						w.Write(audioMP3Fixture())
					}
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			record := filepath.Join(root, "task.json")
			svc := fixtureService(server)
			req := taskRequest{Kind: "audio_task", Operation: "generate", Model: model, Prompt: "private creative text", OutputDir: root, Parameters: map[string]any{}}
			if model == "gemini-3.8-flash-tts" {
				req.Parameters["instructions"] = "private direction"
			} else {
				req.Parameters["lyrics"] = "private lyrics"
			}
			var out bytes.Buffer
			if err := executeRecordedTask("submit", record, req, 0, "", &out, svc); err != nil {
				t.Fatal(err, out.String())
			}
			if len(calls) != 1 {
				t.Fatal("submission must return before polling")
			}
			for _, cmd := range []string{"status", "content", "resume", "delivered", "redeliver", "delivered"} {
				out.Reset()
				if err := executeRecordedTask(cmd, record, taskRequest{}, 0, root, &out, svc); err != nil {
					t.Fatal(cmd, err, out.String())
				}
			}
			if len(calls) != 3 || !strings.Contains(out.String(), `"next_step":"done"`) {
				t.Fatal("same-task handoff failed", calls, out.String())
			}
			data, err := os.ReadFile(record)
			if err != nil || bytes.Contains(data, []byte("private")) {
				t.Fatal("private request persisted")
			}
		})
	}
}

func TestAsyncAudioUncertainSubmissionNeverRepeats(t *testing.T) {
	for _, scenario := range []string{"missing-id", "server", "timeout"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			svc := service{baseURL: "https://fixture.invalid", client: &http.Client{Transport: evaluationTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodPost || r.URL.Path != asyncAudioPath {
					t.Error("unexpected request")
				}
				switch scenario {
				case "timeout":
					return nil, context.DeadlineExceeded
				case "server":
					return evaluationHTTP(503, `{"error":"private response"}`), nil
				default:
					return evaluationHTTP(200, `{"status":"SUCCESS"}`), nil
				}
			})}}
			root := t.TempDir()
			record := filepath.Join(root, "task.json")
			request := taskRequest{Kind: "audio_task", Operation: "generate", Model: "lyria-3.5", Prompt: "private text", OutputDir: root}
			var out bytes.Buffer
			if executeRecordedTask("submit", record, request, 0, "", &out, svc) == nil {
				t.Fatal("uncertain submission accepted")
			}
			saved, err := loadTaskRecord(record)
			if err != nil || saved.SubmissionOutcome != "unknown" || saved.TaskID != "" {
				t.Fatal("lost uncertainty", err)
			}
			out.Reset()
			if executeRecordedTask("resume", record, taskRequest{}, 0, "", &out, svc) == nil || calls != 1 {
				t.Fatal("unknown task retried", calls)
			}
			if !strings.Contains(out.String(), `"submission_outcome":"unknown"`) || strings.Contains(out.String(), "private") {
				t.Fatal("unsafe recovery receipt")
			}
		})
	}
}

func TestAsyncAudioReadFailureKeepsOriginalTask(t *testing.T) {
	for _, scenario := range []string{"status", "download", "wrong-platform"} {
		t.Run(scenario, func(t *testing.T) {
			posts, reads := 0, 0
			svc := service{baseURL: "https://fixture.invalid", client: &http.Client{Transport: evaluationTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPost {
					posts++
					return evaluationHTTP(200, `{"id":"audio_saved","status":"SUBMITTED"}`), nil
				}
				reads++
				if strings.HasSuffix(r.URL.Path, "/content") {
					return evaluationHTTP(410, `{"error":{"code":"task_expired"}}`), nil
				}
				if scenario == "status" {
					return nil, context.DeadlineExceeded
				}
				if scenario == "wrong-platform" {
					return evaluationHTTP(200, `{"task_id":"audio_saved","platform":"other","status":"SUCCESS"}`), nil
				}
				return evaluationHTTP(200, `{"task_id":"audio_saved","platform":"product-yutoux","status":"SUCCESS"}`), nil
			})}}
			root := t.TempDir()
			record := filepath.Join(root, "task.json")
			var out bytes.Buffer
			request := taskRequest{Kind: "audio_task", Operation: "generate", Model: "lyria-3.5", Prompt: "test", OutputDir: root}
			if err := executeRecordedTask("submit", record, request, 0, "", &out, svc); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			err := executeRecordedTask("status", record, taskRequest{}, 0, "", &out, svc)
			if scenario == "download" {
				if err != nil {
					t.Fatal(err)
				}
				out.Reset()
				err = executeRecordedTask("content", record, taskRequest{}, 0, root, &out, svc)
			}
			if err == nil || posts != 1 || reads > 2 {
				t.Fatal("read failure did not stop")
			}
			saved, err := loadTaskRecord(record)
			if err != nil || saved.TaskID != "audio_saved" {
				t.Fatal("lost original task")
			}
			if !strings.Contains(out.String(), `"next_step":"await_user"`) {
				t.Fatal("failure did not await user", out.String())
			}
		})
	}
}

func TestAsyncAudioWaitBudgetAndProtocolIsolation(t *testing.T) {
	reads := 0
	svc := service{baseURL: "https://fixture.invalid", client: &http.Client{Transport: evaluationTransport(func(r *http.Request) (*http.Response, error) {
		reads++
		if r.Method != http.MethodGet || r.URL.Path != "/v1/tasks/audio_pending" {
			t.Error("unexpected request")
		}
		return evaluationHTTP(200, `{"task_id":"audio_pending","platform":"product-yutoux","status":"NOT_START"}`), nil
	})}, wait: func(context.Context, time.Duration) bool { return true }}
	root := t.TempDir()
	record := filepath.Join(root, "task.json")
	request := taskRequest{Kind: "audio_task", Operation: "continue", OriginalOperation: "generate", Model: "lyria-3.5", TaskID: "audio_pending", TaskStatus: "queued", RequestedCount: 1, Parameters: map[string]any{"response_format": "mp3"}}
	if err := saveTaskRecord(record, recordFromRequest(request), true); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		var out bytes.Buffer
		if err := executeRecordedTask("wait", record, taskRequest{}, 0, "", &out, svc); err != nil {
			t.Fatal(err)
		}
		saved, err := loadTaskRecord(record)
		if err != nil || saved.WaitWindowsCompleted != i {
			t.Fatal("lost wait budget")
		}
		if i == 2 && !strings.Contains(out.String(), `"next_step":"await_user"`) {
			t.Fatal("wait budget did not stop")
		}
	}
	if reads != 14 {
		t.Fatal("unexpected wait budget", reads)
	}
	for _, model := range []string{"gemini-3.8-flash-tts", "lyria-3.5", "lyria-3-pro-preview"} {
		data, _ := json.Marshal(map[string]any{"operation": "speech", "model": model, "input": "test", "output_dir": root})
		if _, err := decodeAudioRequest(data); err == nil {
			t.Fatal("async model entered synchronous audio")
		}
	}
	speakers := []any{map[string]any{"name": "Alice", "voice": "Kore"}, map[string]any{"name": "Bob", "voice": "Puck"}}
	req := taskRequest{Kind: "audio_task", Operation: "generate", Model: "gemini-3.8-flash-tts", Prompt: "Alice: hello\nBob: hi", Parameters: map[string]any{"speakers": speakers}}
	if err := validateTaskRequest(req); err != nil {
		t.Fatal(err)
	}
	_, _, body, err := asyncAudioRequestBody(req)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if json.NewDecoder(body).Decode(&wire) != nil || wire["input"] != req.Prompt || len(array(wire["speakers"])) != 2 {
		t.Fatal("dual speech changed")
	}
}

func TestAsyncAudioRejectsAmbiguousResponsesAndUnsafeParameters(t *testing.T) {
	for _, body := range []string{
		`{"task_id":"audio_fixture","platform":"other","status":"SUCCESS"}`,
		`{"task_id":"audio_fixture","status":"SUCCESS"}`,
		`{"task_id":"audio_fixture","platform":"product-yutoux","status":"FAILURE"}`,
		`{"task_id":"different","platform":"product-yutoux","status":"SUCCESS"}`,
		`{"task_id":"audio_fixture","platform":"product-yutoux","status":"UNREVIEWED"}`,
	} {
		r, err := applyTaskStatus(receipt{OK: true, Kind: "audio_task", Model: "lyria-3.5", TaskID: "audio_fixture"}, []byte(body), 200)
		if err == nil || r.OK {
			t.Fatal("unsafe audio status accepted", body)
		}
	}
	for _, params := range []map[string]any{{"response_format": "pcm"}, {"temperature": 3}, {"group": "other"}, {"duration": 10}, {"speakers": []any{map[string]any{"name": "same", "voice": "Kore"}, map[string]any{"name": "same", "voice": "Puck"}}}} {
		req := taskRequest{Kind: "audio_task", Operation: "generate", Model: "gemini-3.8-flash-tts", Prompt: "test", Parameters: params}
		if validateTaskRequest(req) == nil {
			t.Fatal("unsafe audio request accepted")
		}
	}
}
