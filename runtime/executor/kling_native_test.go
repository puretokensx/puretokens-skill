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

func klingNativeFixture(model string) taskRequest {
	return taskRequest{Kind: "video", Operation: "generate", Model: model, MediaOperation: "native_video",
		Parameters: map[string]any{"contents": []any{map[string]any{"type": "prompt", "text": "fixture prompt"}}}}
}

func klingContent(kind, key, value string) map[string]any {
	return map[string]any{"type": kind, key: value}
}

func TestKlingNativeDiscoveryUsesExactEnvelopeRules(t *testing.T) {
	profile := readInstalledProfileFixture(t, "video", "kling-video-v4")
	body, _ := json.Marshal(profile.Parameters)
	var schema map[string]any
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{"id": profile.ID, "capabilities": []any{"video"}, "input_schema": schema}
	query := modelQuery{Kind: "video", Operation: "native_video"}
	if !modelQueryMatches(query, entry) {
		t.Fatal("reviewed native operation missing")
	}
	query.Parameters = klingNativeFixture(profile.ID).Parameters
	if !modelQueryMatches(query, entry) {
		t.Fatal("valid native filter rejected")
	}
	query.Parameters["settings"] = map[string]any{"duration": 100}
	if modelQueryMatches(query, entry) {
		t.Fatal("discovery accepted invalid native settings")
	}
	query.Parameters = map[string]any{"contents": "https://example.com/not-an-envelope"}
	if modelQueryMatches(query, entry) {
		t.Fatal("discovery accepted scalar native content")
	}
}

func TestKlingNativeValidation(t *testing.T) {
	for _, tc := range []struct {
		name, model string
		mutate      func(*taskRequest)
		valid       bool
	}{
		{"v4 existing resources", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("element", "element_id", "subject_1"), map[string]any{"type": "voice", "voice_id": "voice_1", "id": "speaker_1"})
			r.Parameters["settings"] = map[string]any{"duration": 30, "resolution": "1080p", "aspect_ratio": "adaptive"}
			r.Parameters["options"] = map[string]any{"external_task_id": "customer_1", "watermark_info": map[string]any{"enabled": false}}
		}, true},
		{"v3 element and first frame", "kling-video-v3", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("first_frame", "url", "https://example.com/first.png"), klingContent("element", "element_id", "subject_1"))
			r.Parameters["settings"] = map[string]any{"duration": 15, "resolution": "4k", "audio": "native", "multi_shot": false}
		}, true},
		{"omni base video edit stays create", "kling-video-v3-omni", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("base_video", "url", "https://example.com/base.mp4"), map[string]any{"type": "voice", "voice_id": "voice_1", "id": "speaker_1"})
			r.Parameters["settings"] = map[string]any{"audio": "original", "multi_shot": false}
		}, true},
		{"motion existing subject", "kling-video-motion-control-v3.0", func(r *taskRequest) {
			r.Parameters["contents"] = []any{klingContent("image", "url", "https://example.com/character.png"), klingContent("video", "url", "https://example.com/motion.mp4"), map[string]any{"type": "element", "element_id": "subject_1", "id": "character_1"}}
			r.Parameters["settings"] = map[string]any{"character_orientation": "video", "audio": "original"}
		}, true},
		{"unknown native type", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = []any{klingContent("unreviewed", "url", "https://example.com/a")}
		}, false},
		{"non-array contents", "kling-video-v4", func(r *taskRequest) { r.Parameters["contents"] = map[string]any{} }, false},
		{"empty contents", "kling-video-v4", func(r *taskRequest) { r.Parameters["contents"] = []any{} }, false},
		{"null contents", "kling-video-v4", func(r *taskRequest) { r.Parameters["contents"] = nil }, false},
		{"prompt conflict", "kling-video-v4", func(r *taskRequest) { r.Prompt = "different intent" }, false},
		{"duplicate prompt", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("prompt", "text", "second"))
		}, false},
		{"extra prompt fields", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"].([]any)[0].(map[string]any)["url"] = "https://example.com/prompt"
		}, false},
		{"empty prompt", "kling-video-v4", func(r *taskRequest) { r.Parameters["contents"].([]any)[0].(map[string]any)["text"] = " " }, false},
		{"wrong text type", "kling-video-v4", func(r *taskRequest) { r.Parameters["contents"].([]any)[0].(map[string]any)["text"] = 123 }, false},
		{"ignored flat media forbidden", "kling-video-v4", func(r *taskRequest) { r.Parameters["reference_images"] = []any{"https://example.com/ignored.png"} }, false},
		{"ignored flat setting forbidden", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["duration"] = 10
			r.Parameters["settings"] = map[string]any{"duration": 5}
		}, false},
		{"local attachments forbidden", "kling-video-v4", func(r *taskRequest) { r.Attachments = []attachment{{Field: "image", Path: "/fixture/image.png"}} }, false},
		{"native settings without contents", "kling-video-v4", func(r *taskRequest) { delete(r.Parameters, "contents"); r.Parameters["settings"] = map[string]any{} }, false},
		{"native wrong operation", "kling-video-v4", func(r *taskRequest) { r.MediaOperation = "reference_video" }, false},
		{"native edit endpoint forbidden", "kling-video-v3-omni", func(r *taskRequest) { r.Operation = "edit" }, false},
		{"multiple outputs forbidden", "kling-video-v4", func(r *taskRequest) { r.RequestedCount = 2 }, false},
		{"resource creation forbidden", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), map[string]any{"type": "element", "element_id": "subject_1", "images": []any{"https://example.com/new.png"}})
		}, false},
		{"opaque id cannot contain spaces", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("element", "element_id", "bad id"))
		}, false},
		{"opaque id length bounded", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("element", "element_id", strings.Repeat("a", 129)))
		}, false},
		{"private url forbidden", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("refer_image", "url", "https://127.0.0.1/secret.png"))
		}, false},
		{"data URL forbidden", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("refer_image", "url", "data:image/png;base64,AAAA"))
		}, false},
		{"v4 controls forbidden", "kling-video-v4", func(r *taskRequest) { r.Parameters["settings"] = map[string]any{"audio": "off"} }, false},
		{"v4 adaptive text forbidden", "kling-video-v4", func(r *taskRequest) { r.Parameters["settings"] = map[string]any{"aspect_ratio": "adaptive"} }, false},
		{"v4 duration upper bound", "kling-video-v4", func(r *taskRequest) { r.Parameters["settings"] = map[string]any{"duration": 31} }, false},
		{"v4 duration cannot be string", "kling-video-v4", func(r *taskRequest) { r.Parameters["settings"] = map[string]any{"duration": "5"} }, false},
		{"null settings forbidden", "kling-video-v4", func(r *taskRequest) { r.Parameters["settings"] = nil }, false},
		{"null options forbidden", "kling-video-v4", func(r *taskRequest) { r.Parameters["options"] = nil }, false},
		{"unknown option forbidden", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["options"] = map[string]any{"callback_url": "https://example.com/receive"}
		}, false},
		{"option task id bounded", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["options"] = map[string]any{"external_task_id": strings.Repeat("a", 129)}
		}, false},
		{"watermark typed", "kling-video-v4", func(r *taskRequest) {
			r.Parameters["options"] = map[string]any{"watermark_info": map[string]any{"enabled": "false"}}
		}, false},
		{"turbo last frame forbidden", "kling-video-v3-turbo", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("last_frame", "url", "https://example.com/last.png"))
		}, false},
		{"turbo subject forbidden", "kling-video-v3-turbo", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("element", "element_id", "subject_1"))
		}, false},
		{"turbo multi shot forbidden", "kling-video-v3-turbo", func(r *taskRequest) { r.Parameters["settings"] = map[string]any{"multi_shot": false} }, false},
		{"v3 first frame prompt limit", "kling-video-v3", func(r *taskRequest) {
			r.Parameters["contents"] = []any{klingContent("prompt", "text", strings.Repeat("a", 2501)), klingContent("first_frame", "url", "https://example.com/first.png")}
		}, false},
		{"first frame ratio forbidden", "kling-video-v3", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("first_frame", "url", "https://example.com/first.png"))
			r.Parameters["settings"] = map[string]any{"aspect_ratio": "16:9"}
		}, false},
		{"omni feature audio forbidden", "kling-video-v3-omni", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("feature_video", "url", "https://example.com/feature.mp4"))
			r.Parameters["settings"] = map[string]any{"audio": "native"}
		}, false},
		{"omni base multishot forbidden", "kling-video-v3-omni", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("base_video", "url", "https://example.com/base.mp4"))
			r.Parameters["settings"] = map[string]any{"multi_shot": true}
		}, false},
		{"omni voice requires audio", "kling-video-v3-omni", func(r *taskRequest) {
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), map[string]any{"type": "voice", "voice_id": "voice_1", "id": "speaker_1"})
		}, false},
		{"motion requires orientation", "kling-video-motion-control-v3.0", func(r *taskRequest) {
			r.Parameters["contents"] = []any{klingContent("image", "url", "https://example.com/character.png"), klingContent("video", "url", "https://example.com/motion.mp4")}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := klingNativeFixture(tc.model)
			tc.mutate(&r)
			err := validateReviewedKlingRequest(r)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
			if err != nil && (strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "subject_1") || strings.Contains(err.Error(), "different intent")) {
				t.Fatal("private request detail leaked in error")
			}
		})
	}
}

func TestKlingNativeBodyRoundTripPreservesEnvelope(t *testing.T) {
	r := klingNativeFixture("kling-video-v3-omni")
	r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("base_video", "url", "https://example.com/base.mp4"))
	r.Parameters["settings"] = map[string]any{"duration": float64(8), "resolution": "4k", "audio": "off", "multi_shot": false}
	r.Parameters["options"] = map[string]any{"external_task_id": "customer_1", "watermark_info": map[string]any{"enabled": false}}
	path, kind, reader, err := taskRequestBody(r)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/videos" || kind != "application/json" {
		t.Fatalf("unexpected native target %s %s", path, kind)
	}
	encoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"contents", "settings", "options"} {
		if !reflect.DeepEqual(body[key], r.Parameters[key]) {
			t.Fatalf("native %s changed during serialization", key)
		}
	}
}

func TestKlingNativeProfilePreparationPreservesIntent(t *testing.T) {
	for _, model := range []string{"kling-video-v4", "kling-video-v3", "kling-video-v3-turbo", "kling-video-v3-omni", "kling-video-motion-control-v3.0", "kling-video-motion-control-v2.6"} {
		t.Run(model, func(t *testing.T) {
			r := klingNativeFixture(model)
			r.MediaOperation = ""
			if strings.Contains(model, "motion-control") {
				r.Parameters["contents"] = []any{klingContent("image", "url", "https://example.com/character.png"), klingContent("video", "url", "https://example.com/motion.mp4")}
				r.Parameters["settings"] = map[string]any{"character_orientation": "video", "audio": "off"}
			}
			before, err := json.Marshal(r.Parameters)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateTaskRequest(r); err != nil {
				t.Fatalf("native prompt exception: %v", err)
			}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			after, err := json.Marshal(r.Parameters)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("native input intent mutated")
			}
			if r.MediaOperation != "native_video" || r.route != "/v1/videos" || r.RequestedCount != 1 {
				t.Fatalf("native operation not prepared: %#v", r)
			}
			if !strings.Contains(model, "motion-control") && r.Prompt != "fixture prompt" {
				t.Fatal("native prompt not preserved")
			}
		})
	}
}

func TestKlingNativeBoundaryCounts(t *testing.T) {
	for _, tc := range []struct {
		model, kind, key string
		count            int
		valid            bool
	}{
		{"kling-video-v4", "element", "element_id", 7, true},
		{"kling-video-v4", "element", "element_id", 8, false},
		{"kling-video-v4", "refer_image", "url", 10, true},
		{"kling-video-v4", "refer_image", "url", 11, false},
		{"kling-video-v4", "feature_video", "url", 5, true},
		{"kling-video-v4", "feature_video", "url", 6, false},
		{"kling-video-v3-omni", "element", "element_id", 7, true},
		{"kling-video-v3-omni", "element", "element_id", 8, false},
	} {
		r := klingNativeFixture(tc.model)
		for i := 0; i < tc.count; i++ {
			value := "subject_1"
			if tc.key == "url" {
				value = "https://example.com/reference"
			}
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent(tc.kind, tc.key, value))
		}
		if err := validateReviewedKlingRequest(r); (err == nil) != tc.valid {
			t.Errorf("%s %s count=%d valid=%v: %v", tc.model, tc.kind, tc.count, tc.valid, err)
		}
	}
}

func TestKlingNativeReceiptExcludesPrivateEnvelope(t *testing.T) {
	r := klingNativeFixture("kling-video-v4")
	r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("refer_image", "url", "https://example.com/private-image.png"), klingContent("element", "element_id", "private-subject-id"), map[string]any{"type": "voice", "voice_id": "private-voice-id", "id": "private-speaker-id"})
	r.Parameters["options"] = map[string]any{"external_task_id": "private-business-id"}
	r.Parameters["settings"] = map[string]any{"duration": 5, "resolution": "720p"}
	encoded, err := json.Marshal(taskReceipt(r, "video-task-1", "pending"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-", "example.com", "fixture prompt", "contents", "element_id", "voice_id", "external_task_id"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("receipt leaked %s", forbidden)
		}
	}
}

func TestKlingNativeRecordedSubmitUsesOneFixedPost(t *testing.T) {
	r := klingNativeFixture("kling-video-v3-omni")
	r.Parameters["contents"] = append(r.Parameters["contents"].([]any), klingContent("base_video", "url", "https://example.com/private-video.mp4"), klingContent("element", "element_id", "private-subject-id"), map[string]any{"type": "voice", "voice_id": "private-voice-id", "id": "private-speaker-id"})
	r.Parameters["settings"] = map[string]any{"duration": float64(7), "resolution": "4k", "audio": "original", "multi_shot": false}
	r.Parameters["options"] = map[string]any{"external_task_id": "private-business-id", "watermark_info": map[string]any{"enabled": false}}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls++
		if request.Method != http.MethodPost || request.URL.Path != "/v1/videos" {
			t.Errorf("native submit made unexpected request: %s %s", request.Method, request.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload["model"] != r.Model || payload["prompt"] != "fixture prompt" || len(payload) != 5 {
			t.Error("native submit changed envelope fields")
		}
		for _, key := range []string{"contents", "settings", "options"} {
			if !reflect.DeepEqual(payload[key], r.Parameters[key]) {
				t.Errorf("native submit changed %s", key)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"id":"kling_native_fixture","status":"pending"}`)
	}))
	defer server.Close()
	directory := t.TempDir()
	recordPath := filepath.Join(directory, "task.json")
	r.OutputDir = directory
	var output bytes.Buffer
	if err := executeRecordedTask("submit", recordPath, r, 0, "", &output, fixtureService(server)); err != nil {
		t.Fatal(err, output.String())
	}
	if calls != 1 {
		t.Fatalf("submit must return before polling or downloading: calls=%d", calls)
	}
	var result receipt
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.TaskID != "kling_native_fixture" || result.Status != "pending" || result.OriginalOperation != "native_video" {
		t.Fatalf("unexpected native receipt: %#v", result)
	}
	record, err := loadTaskRecord(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if record.TaskID != "kling_native_fixture" || record.OriginalOperation != "native_video" || record.RequestedCount != 1 {
		t.Fatal("native task identity not retained")
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-", "example.com", "fixture prompt", "contents", "settings", "options", "element_id", "voice_id", "external_task_id"} {
		if bytes.Contains(data, []byte(forbidden)) || bytes.Contains(output.Bytes(), []byte(forbidden)) {
			t.Fatalf("native receipt or record leaked %s", forbidden)
		}
	}
}

func TestKlingNativeResourceCombinations(t *testing.T) {
	for _, tc := range []struct {
		name, model string
		contents    []any
		settings    map[string]any
		valid       bool
	}{
		{"duplicate voice aliases", "kling-video-v4", []any{map[string]any{"type": "voice", "voice_id": "voice_1", "id": "speaker"}, map[string]any{"type": "voice", "voice_id": "voice_2", "id": "speaker"}}, nil, false},
		{"same voice distinct aliases", "kling-video-v4", []any{map[string]any{"type": "voice", "voice_id": "voice_1", "id": "speaker_1"}, map[string]any{"type": "voice", "voice_id": "voice_1", "id": "speaker_2"}}, nil, true},
		{"base and feature mutually exclusive", "kling-video-v3-omni", []any{klingContent("base_video", "url", "https://example.com/base.mp4"), klingContent("feature_video", "url", "https://example.com/feature.mp4")}, nil, false},
		{"base excludes frames", "kling-video-v3-omni", []any{klingContent("base_video", "url", "https://example.com/base.mp4"), klingContent("first_frame", "url", "https://example.com/first.png")}, nil, false},
		{"feature excludes last frame", "kling-video-v3-omni", []any{klingContent("feature_video", "url", "https://example.com/feature.mp4"), klingContent("first_frame", "url", "https://example.com/first.png"), klingContent("last_frame", "url", "https://example.com/last.png")}, nil, false},
		{"motion 2.6 excludes element", "kling-video-motion-control-v2.6", []any{klingContent("image", "url", "https://example.com/character.png"), klingContent("video", "url", "https://example.com/motion.mp4"), map[string]any{"type": "element", "element_id": "subject", "id": "actor"}}, map[string]any{"character_orientation": "video"}, false},
		{"motion subject needs video orientation", "kling-video-motion-control-v3.0", []any{klingContent("image", "url", "https://example.com/character.png"), klingContent("video", "url", "https://example.com/motion.mp4"), map[string]any{"type": "element", "element_id": "subject", "id": "actor"}}, map[string]any{"character_orientation": "image"}, false},
		{"motion follows source duration", "kling-video-motion-control-v3.0", []any{klingContent("image", "url", "https://example.com/character.png"), klingContent("video", "url", "https://example.com/motion.mp4")}, map[string]any{"character_orientation": "video", "duration": 5}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := klingNativeFixture(tc.model)
			r.Parameters["contents"] = append(r.Parameters["contents"].([]any), tc.contents...)
			if tc.settings != nil {
				r.Parameters["settings"] = tc.settings
			}
			if err := validateReviewedKlingRequest(r); (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}
