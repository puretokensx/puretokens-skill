package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVoyageEmbeddingPreservesReviewedControlsAndOutput(t *testing.T) {
	for _, dimension := range []int{0, 256, 512, 1024, 2048} {
		calls := 0
		result, err, raw := retrievalCLI(t, "embedding", func(r map[string]any) {
			r["model"], r["input_type"] = "voyage-4-large", "document"
			if dimension != 0 {
				r["output_dimension"] = dimension
			}
		}, func(r *http.Request) (*http.Response, error) {
			calls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if r.Method != "POST" || r.URL.String() != "https://api.puretokensx.com/v1/embeddings" || body["model"] != "voyage-4-large" || body["input_type"] != "document" || body["output_dtype"] != "float" || body["truncation"] != false || body["encoding_format"] != "float" {
				t.Error("wrong Voyage protocol or silent truncation enabled")
			}
			if body["dimensions"] != nil || body["output_file"] != nil {
				t.Error("unexpected local or OpenAI-specific field")
			}
			want := dimension
			if want == 0 {
				want = 1024
				if body["output_dimension"] != nil {
					t.Error("invented dimension override")
				}
			} else if body["output_dimension"] != float64(want) {
				t.Error("dimension changed")
			}
			b, _ := json.Marshal(map[string]any{"data": []any{map[string]any{"index": 1, "embedding": make([]float64, want)}, map[string]any{"index": 0, "embedding": make([]float64, want)}}})
			return evaluationHTTP(200, string(b)), nil
		})
		if err != nil || calls != 1 || result["ok"] != true {
			t.Fatalf("Voyage failed: %v calls=%d %v", err, calls, result)
		}
		artifact := jsonObject(result["artifact"])
		data, readErr := os.ReadFile(artifact["path"].(string))
		want := dimension
		if want == 0 {
			want = 1024
		}
		if readErr != nil || artifact["dimensions"] != float64(want) || !strings.HasPrefix(string(data), `{"index":0,`) || strings.Contains(raw, "PRIVATE-") {
			t.Fatal("unverified or leaking Voyage output")
		}
	}
}

func TestVoyageRejectsUnreviewedControlsBeforeNetwork(t *testing.T) {
	for name, value := range map[string]any{"input_type": "chat", "output_dimension": 768, "truncation": true, "output_dtype": "binary", "dimensions": 512} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			result, err, _ := retrievalCLI(t, "embedding", func(r map[string]any) { r["model"] = "voyage-4-large"; r[name] = value }, func(*http.Request) (*http.Response, error) {
				calls++
				return evaluationHTTP(200, embeddingResponseFixture), nil
			})
			if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" {
				t.Fatal("unreviewed controls reached API")
			}
		})
	}
}

func TestVoyageRejectsWrongReturnedDimensionWithoutResubmit(t *testing.T) {
	calls := 0
	result, err, _ := retrievalCLI(t, "embedding", func(r map[string]any) { r["model"] = "voyage-4-large"; r["output_dimension"] = 256 }, func(*http.Request) (*http.Response, error) {
		calls++
		return evaluationHTTP(200, embeddingResponseFixture), nil
	})
	if err == nil || calls != 1 || result["local_error_code"] != "embedding_response_invalid" || result["submission_outcome"] != "unknown" {
		t.Fatalf("incorrect dimension accepted or retried: %v %v calls=%d", result, err, calls)
	}
}

func wanNativeFixture(model string) taskRequest {
	return taskRequest{Kind: "video", Operation: "generate", Model: model, MediaOperation: "native_generation", Parameters: map[string]any{
		"input":      map[string]any{"prompt": "fixture private prompt", "media": []any{map[string]any{"type": "file", "url": "https://example.com/reference.pdf"}}},
		"parameters": map[string]any{"duration": float64(-1), "resolution": "720P", "ratio": "16:9", "prompt_extend": true, "audio": false, "watermark": false},
	}}
}

func TestWanNativeInputsPrepareWithoutChangingIntent(t *testing.T) {
	for _, model := range []string{"wan3.0-video", "wan3.0-video-prime"} {
		for _, kind := range []string{"file", "link", "reference_video", "reference_audio", "reference_image", "first_frame"} {
			r := wanNativeFixture(model)
			r.Parameters["input"].(map[string]any)["media"].([]any)[0].(map[string]any)["type"] = kind
			before, _ := json.Marshal(r.Parameters)
			if err := validateTaskRequest(r); err != nil {
				t.Fatalf("native prompt rejected: %v", err)
			}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(r.Parameters)
			if string(before) != string(after) || r.route != "/v1/videos" || r.RequestedCount != 1 || r.Prompt != "fixture private prompt" {
				t.Fatal("native intent or route changed")
			}
		}
	}
}

func TestWanNativeRejectsInvalidEnvelope(t *testing.T) {
	for name, change := range map[string]func(*taskRequest){
		"wrong model":      func(r *taskRequest) { r.Model = "MiniMax-H3" },
		"wrong kind":       func(r *taskRequest) { r.Kind = "image" },
		"wrong operation":  func(r *taskRequest) { r.Operation = "edit" },
		"flat operation":   func(r *taskRequest) { r.MediaOperation = "reference_video" },
		"multiple outputs": func(r *taskRequest) { r.RequestedCount = 2 },
		"local attachment": func(r *taskRequest) { r.Attachments = []attachment{{Field: "file", Path: "/fixture/file.pdf"}} },
		"flat media":       func(r *taskRequest) { r.Parameters["reference_images"] = []any{"https://example.com/image.png"} },
		"flat setting":     func(r *taskRequest) { r.Parameters["duration"] = 5 },
		"idempotency":      func(r *taskRequest) { r.Parameters["idempotency_key"] = "unverified" },
		"missing input":    func(r *taskRequest) { delete(r.Parameters, "input") },
		"null input":       func(r *taskRequest) { r.Parameters["input"] = nil },
		"extra input":      func(r *taskRequest) { r.Parameters["input"].(map[string]any)["secret"] = "private" },
		"prompt conflict":  func(r *taskRequest) { r.Prompt = "different" },
		"long prompt":      func(r *taskRequest) { r.Parameters["input"].(map[string]any)["prompt"] = strings.Repeat("界", 20001) },
		"null prompt":      func(r *taskRequest) { r.Parameters["input"].(map[string]any)["prompt"] = nil },
		"null media":       func(r *taskRequest) { r.Parameters["input"].(map[string]any)["media"] = nil },
		"empty intent":     func(r *taskRequest) { r.Parameters["input"] = map[string]any{} },
		"unknown role": func(r *taskRequest) {
			r.Parameters["input"].(map[string]any)["media"] = []any{map[string]any{"type": "subject", "url": "https://example.com/ref"}}
		},
		"private URL": func(r *taskRequest) {
			r.Parameters["input"].(map[string]any)["media"] = []any{map[string]any{"type": "file", "url": "https://127.0.0.1/ref"}}
		},
		"data URL": func(r *taskRequest) {
			r.Parameters["input"].(map[string]any)["media"] = []any{map[string]any{"type": "file", "url": "data:text/plain;base64,AAAA"}}
		},
		"extra media field": func(r *taskRequest) {
			r.Parameters["input"].(map[string]any)["media"].([]any)[0].(map[string]any)["headers"] = map[string]any{"Authorization": "private"}
		},
		"null settings":          func(r *taskRequest) { r.Parameters["parameters"] = nil },
		"unknown settings":       func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["seed"] = 5 },
		"string duration":        func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["duration"] = "5" },
		"duration below minimum": func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["duration"] = 1 },
		"duration over maximum":  func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["duration"] = 31 },
		"fractional duration":    func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["duration"] = 5.5 },
		"unknown resolution":     func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["resolution"] = "4K" },
		"unknown ratio":          func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["ratio"] = "2:1" },
		"boolean string":         func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["audio"] = "false" },
		"disabled extension":     func(r *taskRequest) { r.Parameters["parameters"].(map[string]any)["prompt_extend"] = false },
		"implicit extension":     func(r *taskRequest) { delete(r.Parameters["parameters"].(map[string]any), "prompt_extend") },
	} {
		t.Run(name, func(t *testing.T) {
			r := wanNativeFixture("wan3.0-video")
			change(&r)
			if err := prepareProfileRequest(&r, profileService()); err == nil {
				t.Fatal("invalid native envelope accepted")
			}
		})
	}
}

func TestWanNativeCombinationBounds(t *testing.T) {
	for _, tc := range []struct {
		roles map[string]int
		valid bool
	}{
		{map[string]int{"first_frame": 1, "last_frame": 1}, true},
		{map[string]int{"last_frame": 1}, false},
		{map[string]int{"first_frame": 2}, false},
		{map[string]int{"first_frame": 1, "reference_image": 1}, false},
		{map[string]int{"first_frame": 1, "file": 1}, false},
		{map[string]int{"reference_image": 10, "reference_video": 5, "reference_audio": 5}, true},
		{map[string]int{"reference_image": 11}, false},
		{map[string]int{"reference_video": 6}, false},
		{map[string]int{"reference_audio": 6}, false},
		{map[string]int{"file": 1, "link": 1}, false},
	} {
		r := wanNativeFixture("wan3.0-video-prime")
		media := []any{}
		for kind, count := range tc.roles {
			for i := 0; i < count; i++ {
				media = append(media, map[string]any{"type": kind, "url": "https://example.com/ref"})
			}
		}
		r.Parameters["input"].(map[string]any)["media"] = media
		if err := prepareProfileRequest(&r, profileService()); (err == nil) != tc.valid {
			t.Fatalf("roles=%v valid=%v err=%v", tc.roles, tc.valid, err)
		}
	}
}

func TestWanNativeDiscoveryAndPromptlessReference(t *testing.T) {
	r := wanNativeFixture("wan3.0-video")
	delete(r.Parameters["input"].(map[string]any), "prompt")
	if err := validateTaskRequest(r); err != nil {
		t.Fatal(err)
	}
	if err := prepareProfileRequest(&r, profileService()); err != nil {
		t.Fatal(err)
	}
	profile := readInstalledProfileFixture(t, "video", r.Model)
	data, _ := json.Marshal(profile.Parameters)
	var schema map[string]any
	json.Unmarshal(data, &schema)
	entry := map[string]any{"id": r.Model, "capabilities": []any{"video"}, "input_schema": schema}
	query := modelQuery{Kind: "video", Operation: "native_generation"}
	if !modelQueryMatches(query, entry) {
		t.Fatal("native operation missing in discovery")
	}
	query.Parameters = r.Parameters
	if !modelQueryMatches(query, entry) {
		t.Fatal("valid native filter rejected")
	}
	query.Parameters["parameters"].(map[string]any)["prompt_extend"] = false
	if modelQueryMatches(query, entry) {
		t.Fatal("invalid native filter accepted")
	}
}

func TestWanNativeSubmitOnceWithPrivateInputsExcludedFromRecord(t *testing.T) {
	r := wanNativeFixture("wan3.0-video")
	if err := prepareProfileRequest(&r, profileService()); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls++
		if request.Method != "POST" || request.URL.Path != "/v1/videos" {
			t.Error("unexpected task request")
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload) != 4 || payload["model"] != r.Model || payload["prompt"] != r.Prompt || !reflect.DeepEqual(payload["input"], r.Parameters["input"]) || !reflect.DeepEqual(payload["parameters"], r.Parameters["parameters"]) {
			t.Error("native intent changed on wire")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		io.WriteString(w, `{"id":"wan_native_fixture","status":"pending"}`)
	}))
	defer server.Close()
	r.OutputDir = t.TempDir()
	recordPath := filepath.Join(r.OutputDir, "task.json")
	var output bytes.Buffer
	if err := executeRecordedTask("submit", recordPath, r, 0, "", &output, fixtureService(server)); err != nil {
		t.Fatal(err)
	}
	var result receipt
	json.Unmarshal(output.Bytes(), &result)
	if calls != 1 || !result.OK || result.TaskID != "wan_native_fixture" || result.OriginalOperation != "native_generation" {
		t.Fatal("submission retried, polled or lost identity")
	}
	record, err := loadTaskRecord(recordPath)
	if err != nil || record.TaskID != result.TaskID || record.OriginalOperation != result.OriginalOperation {
		t.Fatal("task recovery identity missing")
	}
	data, _ := os.ReadFile(recordPath)
	for _, secret := range []string{"fixture private prompt", "example.com", "reference.pdf", `"input"`, `"media"`} {
		if bytes.Contains(data, []byte(secret)) || bytes.Contains(output.Bytes(), []byte(secret)) {
			t.Fatal("native input leaked")
		}
	}
}

func TestUnlistedImageModelKeepsSameTaskRecovery(t *testing.T) {
	dir := t.TempDir()
	recordPath := filepath.Join(dir, "task.json")
	record := taskRecord{Format: taskRecordFormat, Kind: "image", Model: "gemini-3.1-flash-image", TaskID: "existing_gemini_task", OriginalOperation: "image_edit", RequestedCount: 1, Status: "completed", OutputDir: dir}
	if err := saveTaskRecord(recordPath, record, true); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/v1/images/existing_gemini_task/content" {
			t.Error("recovery performed discovery, resubmit or used a different task")
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(fixturePNG(t))
	}))
	defer server.Close()
	var out bytes.Buffer
	if err := executeRecordedTask("content", recordPath, taskRequest{}, 0, dir, &out, fixtureService(server)); err != nil {
		t.Fatal(err)
	}
	restored, err := loadTaskRecord(recordPath)
	if err != nil || calls != 1 || restored.Model != record.Model || restored.TaskID != record.TaskID || len(restored.Downloaded) != 1 {
		t.Fatal("catalog removal broke recovery")
	}
}

func TestVoyagePermissionDenialNeverRetriesOrChangesModel(t *testing.T) {
	for _, status := range []int{401, 403, 429, 503} {
		calls := 0
		result, err, raw := retrievalCLI(t, "embedding", func(r map[string]any) { r["model"] = "voyage-4-large" }, func(r *http.Request) (*http.Response, error) {
			calls++
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["model"] != "voyage-4-large" {
				t.Fatal("model changed")
			}
			return evaluationHTTP(status, `{"error":{"message":"PRIVATE-UPSTREAM"}}`), nil
		})
		outcome := "rejected"
		if status == 503 {
			outcome = "unknown"
		}
		if err == nil || calls != 1 || result["submission_outcome"] != outcome || strings.Contains(raw, "PRIVATE-UPSTREAM") {
			t.Fatalf("denial handling: %v %v calls=%d", result, err, calls)
		}
	}
}
