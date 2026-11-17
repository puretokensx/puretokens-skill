package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"testing"
)

func TestCurrentCatalogAddedImageModels(t *testing.T) {
	for _, tc := range []struct {
		id, editRoute, size string
		maxReferences       int
	}{
		{"dola-seedream-5.5", "/v1/images/generations", "2K", 10},
		{"gemini-3.1-flash-image", "/v1/images/edits", "1K", 14},
	} {
		t.Run(tc.id, func(t *testing.T) {
			profile := readInstalledProfileFixture(t, "image", tc.id)
			for _, edit := range []bool{false, true} {
				svc, transport := catalogProfileService(t, profile)
				r := taskRequest{Kind: "image", Operation: "generate", Model: tc.id, Prompt: "fixture"}
				wantRoute := "/v1/images/generations"
				if edit {
					r.Operation, wantRoute = "edit", tc.editRoute
					r.Parameters = map[string]any{"images": []any{"https://example.com/reference.png"}}
				}
				if err := prepareProfileRequest(&r, svc); err != nil || transport.calls != 0 {
					t.Fatalf("installed operation needs no discovery: %v", err)
				}
				route, contentType, body, err := taskRequestBody(r)
				if err != nil || route != wantRoute || contentType != "application/json" {
					t.Fatalf("wrong route: %s %s %v", route, contentType, err)
				}
				data, err := io.ReadAll(body)
				var payload map[string]any
				if err != nil || json.Unmarshal(data, &payload) != nil || payload["model"] != tc.id ||
					payload["image_size"] != tc.size || payload["aspect_ratio"] != "1:1" {
					t.Fatal("exact model or declared defaults lost")
				}
			}
			for _, count := range []int{tc.maxReferences, tc.maxReferences + 1} {
				refs := make([]any, count)
				for i := range refs {
					refs[i] = "https://example.com/reference.png"
				}
				svc, transport := catalogProfileService(t, profile)
				r := taskRequest{Kind: "image", Operation: "edit", Model: tc.id, Prompt: "fixture", Parameters: map[string]any{"images": refs}}
				if err := prepareProfileRequest(&r, svc); (err == nil) != (count == tc.maxReferences) || transport.calls != 0 {
					t.Fatalf("reference limit %d: %v", count, err)
				}
			}
		})
	}
	profile := readInstalledProfileFixture(t, "image", "dola-seedream-5.5")
	svc, _ := catalogProfileService(t, profile)
	r := taskRequest{Kind: "image", Operation: "edit", Model: profile.ID, Prompt: "fixture", Attachments: []attachment{{Field: "images", Path: "/synthetic/reference.png"}}}
	if prepareProfileRequest(&r, svc) == nil {
		t.Fatal("URL-only model accepted a local reference")
	}
}

func TestReviewedFlashBoundariesWithoutInstalledFiles(t *testing.T) {
	if _, ok := reviewedVideoProfile("seedance-2.5-flash"); !ok {
		t.Fatal("missing source-reviewed Flash profile")
	}
	for _, tc := range []struct {
		parameters map[string]any
		allowed    bool
	}{
		{map[string]any{}, true},
		{map[string]any{"duration": 4, "resolution": "480p", "aspect_ratio": "21:9"}, true},
		{map[string]any{"duration": 30, "resolution": "720p", "aspect_ratio": "9:16"}, true},
		{map[string]any{"duration": 3}, false}, {map[string]any{"duration": 31}, false},
		{map[string]any{"duration": 4.5}, false}, {map[string]any{"aspect_ratio": "auto"}, false},
		{map[string]any{"resolution": "1080p"}, false}, {map[string]any{"resolution": "2k"}, false},
		{map[string]any{"resolution": "4k"}, false}, {map[string]any{"generate_audio": false}, false},
		{map[string]any{"first_frame_image": "https://example.com/first.png", "reference_images": []any{"https://example.com/ref.png"}}, false},
	} {
		r := taskRequest{Kind: "video", Operation: "generate", Model: "seedance-2.5-flash", Prompt: "fixture", Parameters: tc.parameters}
		if err := prepareProfileRequest(&r, service{}); (err == nil) != tc.allowed {
			t.Fatalf("Flash parameters %v: %v", tc.parameters, err)
		}
	}
	for _, tc := range []struct {
		images, videos, audios int
		allowed                bool
	}{
		{30, 10, 10, true}, {31, 0, 0, false}, {0, 11, 0, false}, {0, 0, 11, false},
	} {
		parameters := map[string]any{}
		for field, count := range map[string]int{"reference_images": tc.images, "reference_videos": tc.videos, "reference_audios": tc.audios} {
			if count == 0 {
				continue
			}
			refs := make([]any, count)
			for i := range refs {
				refs[i] = "https://example.com/reference"
			}
			parameters[field] = refs
		}
		r := taskRequest{Kind: "video", Operation: "generate", Model: "seedance-2.5-flash", Prompt: "fixture", Parameters: parameters}
		if err := prepareProfileRequest(&r, service{}); (err == nil) != tc.allowed {
			t.Fatalf("Flash reference count %+v: %v", tc, err)
		}
	}
}

func TestReviewedFlashPreservesDedicatedRoutesAndReferenceRoles(t *testing.T) {
	data := fixturePNG(t)
	file := writeFixture(t, "reference.png", string(data))
	for _, tc := range []struct {
		operation string
		fields    []string
	}{
		{"first_frame_video", []string{"first_frame_image"}},
		{"last_frame_video", []string{"last_frame_image"}},
		{"first_last_frame_video", []string{"first_frame_image", "last_frame_image"}},
		{"reference_image_video", []string{"reference_images", "reference_images"}},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			r := taskRequest{Kind: "video", Operation: "generate", Model: "seedance-2.5-flash", Prompt: "fixture", MediaOperation: tc.operation}
			for _, field := range tc.fields {
				r.Attachments = append(r.Attachments, attachment{Field: field, Path: file})
			}
			if err := prepareProfileRequest(&r, service{}); err != nil {
				t.Fatal(err)
			}
			route, contentType, body, err := taskRequestBody(r)
			if err != nil || route != "/v1/videos" {
				t.Fatalf("wrong Flash route: %s %v", route, err)
			}
			defer body.(io.Closer).Close()
			kind, params, err := mime.ParseMediaType(contentType)
			if err != nil || kind != "multipart/form-data" {
				t.Fatal("wrong reference encoding")
			}
			reader := multipart.NewReader(body, params["boundary"])
			count, modelSeen := 0, false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				content, err := io.ReadAll(part)
				if err != nil {
					t.Fatal(err)
				}
				if part.FileName() != "" {
					if count >= len(tc.fields) || part.FormName() != tc.fields[count] || !bytes.Equal(content, data) || part.Header.Get("Content-Type") != "image/png" {
						t.Fatal("reference role, MIME or bytes changed")
					}
					count++
				} else if part.FormName() == "model" {
					modelSeen = string(content) == r.Model
				}
			}
			if !modelSeen || count != len(tc.fields) {
				t.Fatal("lost exact model or references")
			}
		})
	}
	r := taskRequest{Kind: "video", Operation: "generate", Model: "seedance-2.5-flash", Prompt: "fixture",
		Parameters: map[string]any{"duration": 5, "aspect_ratio": "9:16", "resolution": "720p",
			"reference_images": []any{"https://example.com/a.png"},
			"reference_videos": []any{"https://example.com/b.mp4"},
			"reference_audios": []any{"https://example.com/c.mp3"}}}
	if err := prepareProfileRequest(&r, service{}); err != nil {
		t.Fatal(err)
	}
	route, contentType, body, err := taskRequestBody(r)
	if err != nil || route != "/v1/videos" || contentType != "application/json" {
		t.Fatal("wrong mixed-reference route")
	}
	var payload map[string]any
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for key, want := range r.Parameters {
		expected, _ := json.Marshal(want)
		actual, _ := json.Marshal(payload[key])
		if !bytes.Equal(expected, actual) {
			t.Fatalf("changed reference or parameter: %s", key)
		}
	}
	if payload["model"] != r.Model || payload["generate_audio"] != nil {
		t.Fatal("wrong model or inherited audio setting")
	}
}

func TestReviewedFlashDoesNotExpandLiveDiscovery(t *testing.T) {
	live := map[string]any{"id": "seedance-2.5-flash", "capabilities": []any{"video"},
		"input_schema": map[string]any{"properties": map[string]any{"resolution": map[string]any{"type": "string", "enum": []any{"720p", "1080p"}}}}}
	if modelQueryMatches(modelQuery{Kind: "video", Parameters: map[string]any{"resolution": "1080p"}}, live) {
		t.Fatal("stale catalog widened Flash resolution")
	}
	if !modelQueryMatches(modelQuery{Kind: "video", Parameters: map[string]any{"resolution": "720p"}}, live) {
		t.Fatal("lost compatible live intersection")
	}
	if modelQueryMatches(modelQuery{Kind: "video", Operation: "first_last_frame_video"}, live) {
		t.Fatal("source profile invented a live operation")
	}
}
