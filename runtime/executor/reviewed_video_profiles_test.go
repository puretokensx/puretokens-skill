package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewedVideoBoundariesWithoutInstalledFiles(t *testing.T) {
	ref := "https://example.com/reference.png"
	cases := []struct {
		model, operation string
		parameters       map[string]any
		valid            bool
	}{
		{"Super Vega 1.5-pro", "", map[string]any{"duration": 30, "resolution": "4k", "generate_audio": false, "seed": 0}, true},
		{"Super Vega 1.0-pro", "", map[string]any{"duration": 30}, false},
		{"Super Vega 1.0-lite", "", map[string]any{"duration": -1}, false},
		{"Super Vega 1.0-lite", "", map[string]any{"auto_duration": true}, true},
		{"Super Vega 1.0-lite", "video_edit", map[string]any{"video": ref}, false},
		{"Super Vega 1.0-pro", "", map[string]any{"video": ref}, false},
		{"Super Vega 1.0-pro", "video_edit", map[string]any{"video": ref, "reference_videos": []any{ref}}, false},
		{"Super Vega 1.5-pro", "image_to_video", map[string]any{"first_frame_image": ref, "aspect_ratio": "16:9"}, false},
		{"Super Vega 1.5-pro", "", map[string]any{"reference_audios": []any{ref}}, false},
		{"Super Vega 1.0-pro", "", map[string]any{"aspect_ratio": "adaptive"}, false},
		{"MiniMax-H3", "", map[string]any{"duration": 1, "resolution": "768p", "reference_audios": []any{ref}}, true},
		{"MiniMax-H3", "", map[string]any{"resolution": "720p"}, false},
		{"MiniMax-H3", "image_to_video", map[string]any{"first_frame_image": ref, "reference_images": []any{ref}}, false},
		{"kling-v3", "image_to_video", map[string]any{"first_frame_image": ref, "duration": 15, "resolution": "4k"}, true},
		{"kling-v3-turbo", "image_to_video", map[string]any{"first_frame_image": ref}, false},
		{"kling-v3-turbo", "", map[string]any{"resolution": "4k"}, false},
		{"kling-v3", "", map[string]any{"duration": 2}, false},
		{"viduq3-pro", "last_frame_video", map[string]any{"first_frame_image": ref, "last_frame_image": ref, "duration": 16, "generate_audio": false, "seed": 0}, true},
		{"viduq3-turbo", "last_frame_video", map[string]any{"last_frame_image": ref}, false},
		{"viduq4-preview", "", map[string]any{}, false},
		{"viduq4-preview", "reference_audio", map[string]any{"reference_audios": []any{ref}}, false},
		{"viduq4-preview", "reference_audio", map[string]any{"reference_images": []any{ref}, "reference_audios": []any{ref}, "resolution": "2k"}, true},
		{"viduq4-preview", "reference_image_video", map[string]any{"reference_images": []any{ref}, "reference_videos": []any{ref}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.model+"/"+tc.operation, func(t *testing.T) {
			op := "generate"
			if tc.operation == "video_edit" {
				op = "edit"
			}
			r := taskRequest{Kind: "video", Operation: op, Model: tc.model, Prompt: "fixture", MediaOperation: tc.operation, Parameters: tc.parameters}
			err := prepareProfileRequest(&r, service{})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
	for _, tc := range []struct {
		images, videos, audios int
		valid                  bool
	}{
		{9, 3, 0, true}, {9, 3, 1, false}, {10, 0, 0, false},
	} {
		parameters := map[string]any{}
		for key, count := range map[string]int{"reference_images": tc.images, "reference_videos": tc.videos, "reference_audios": tc.audios} {
			if count == 0 {
				continue
			}
			values := make([]any, count)
			for i := range values {
				values[i] = ref
			}
			parameters[key] = values
		}
		r := taskRequest{Kind: "video", Operation: "generate", Model: "Super Vega 1.0-pro", Prompt: "fixture", Parameters: parameters}
		if err := prepareProfileRequest(&r, service{}); (err == nil) != tc.valid {
			t.Fatalf("combined reference limit: %v", err)
		}
	}
}

func TestSpacedVideoIDSubmissionAndRecordKeepExactIdentity(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil || r.Method != "POST" || r.URL.Path != "/v1/videos" ||
			body["model"] != "Super Vega 1.5-pro" || body["generate_audio"] != false || body["seed"] != float64(0) {
			t.Error("exact model, false or zero changed")
		}
		io.WriteString(w, `{"id":"spaced-video-fixture","status":"queued"}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	err := executeTask(strings.NewReader(`{"kind":"video","operation":"generate","model":"Super Vega 1.5-pro","prompt":"fixture","parameters":{"duration":30,"generate_audio":false,"seed":0}}`), &out, fixtureService(server))
	receipt := decodeReceipt(t, &out)
	if err != nil || calls != 1 || receipt.Model != "Super Vega 1.5-pro" || receipt.TaskID != "spaced-video-fixture" {
		t.Fatalf("exact identity or submit-once failed: %v %+v", err, receipt)
	}
	record := recordFromRequest(taskRequest{Kind: "video", Operation: "generate", Model: receipt.Model, RequestedCount: 1})
	if record.Model != receipt.Model {
		t.Fatal("task record changed exact model ID")
	}
}

func TestViduPromptOptionalOnlyForDeclaredImageOperation(t *testing.T) {
	for _, tc := range []struct {
		operation string
		valid     bool
	}{
		{"image_to_video", true}, {"video_generation", false}, {"last_frame_video", false}, {"", false},
	} {
		request := taskRequest{Kind: "video", Operation: "generate", Model: "viduq3-pro", MediaOperation: tc.operation,
			Parameters: map[string]any{"first_frame_image": "https://example.com/frame.png"}}
		err := validateTaskRequest(request)
		if err == nil {
			err = prepareProfileRequest(&request, service{})
		}
		if (err == nil) != tc.valid {
			t.Fatalf("optional prompt %s: %v", tc.operation, err)
		}
	}
}

func TestReviewedVideoDiscoveryCannotExpandStaleLiveDeclarations(t *testing.T) {
	// Live schemas remain a prerequisite. Source review only narrows matches.
	live := map[string]any{"id": "kling-v3-turbo", "capabilities": []any{"video"},
		"input_schema": map[string]any{"properties": map[string]any{"resolution": map[string]any{"type": "string", "enum": []any{"720p", "4k"}}}}}
	if modelQueryMatches(modelQuery{Kind: "video", Parameters: map[string]any{"resolution": "4k"}}, live) {
		t.Fatal("stale live schema expanded Turbo resolution")
	}
	if !modelQueryMatches(modelQuery{Kind: "video", Parameters: map[string]any{"resolution": "720p"}}, live) {
		t.Fatal("compatible live intersection disappeared")
	}
	if modelQueryMatches(modelQuery{Kind: "video", Operation: "video_generation"}, live) {
		t.Fatal("source profile invented a missing live operation")
	}
}
