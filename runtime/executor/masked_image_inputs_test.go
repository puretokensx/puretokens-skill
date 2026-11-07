package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewedMaskedImageUsesLocalMaskAndHTTPSImages(t *testing.T) {
	for _, model := range []string{"gpt-image-2.6", "gpt-image-2.6-flare", "gpt-image-2.6-sunburst"} {
		t.Run(model, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			data := fixturePNG(t)
			os.WriteFile(mask, data, 0600)
			r := taskRequest{Kind: "image", Model: model, Operation: "edit", MediaOperation: "masked_image_edit", Prompt: "fixture", Parameters: map[string]any{"image_urls": []any{"https://example.com/image.png"}}, Attachments: []attachment{{Field: "mask", Path: mask}}}
			if err := validateTaskRequest(r); err != nil {
				t.Fatal(err)
			}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			path, contentType, body, err := taskRequestBody(r)
			if err != nil {
				t.Fatal(err)
			}
			if path != "/v1/images/edits" || contentType != "application/json" {
				t.Fatal("wrong mask route")
			}
			raw, err := io.ReadAll(body)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if json.Unmarshal(raw, &payload) != nil {
				t.Fatal("invalid body")
			}
			if payload["mask"] != "data:image/png;base64,"+base64.StdEncoding.EncodeToString(data) || payload["image_urls"].([]any)[0] != "https://example.com/image.png" || payload["async"] != true {
				t.Fatal("mask or image role changed")
			}
		})
	}
}

func TestReviewedMaskRejectsAmbiguousOrUnsafeInputs(t *testing.T) {
	for _, scenario := range []string{"missing-mask", "duplicate-mask", "caller-data-url", "empty-images", "eleven-images", "private-url", "local-original", "wrong-model", "wrong-operation", "corrupt-mask"} {
		t.Run(scenario, func(t *testing.T) {
			mask := filepath.Join(t.TempDir(), "mask.png")
			os.WriteFile(mask, fixturePNG(t), 0600)
			r := taskRequest{Kind: "image", Model: "gpt-image-2.6-flare", Operation: "edit", MediaOperation: "masked_image_edit", Prompt: "fixture", Parameters: map[string]any{"image_urls": []any{"https://example.com/image.png"}}, Attachments: []attachment{{Field: "mask", Path: mask}}}
			switch scenario {
			case "missing-mask":
				r.Attachments = nil
			case "duplicate-mask":
				r.Attachments = append(r.Attachments, r.Attachments[0])
			case "caller-data-url":
				r.Parameters["mask"] = "data:image/png;base64,AAAA"
			case "empty-images":
				r.Parameters["image_urls"] = []any{}
			case "eleven-images":
				a := []any{}
				for i := 0; i < 11; i++ {
					a = append(a, "https://example.com/image.png")
				}
				r.Parameters["image_urls"] = a
			case "private-url":
				r.Parameters["image_urls"] = []any{"https://127.0.0.1/image.png"}
			case "local-original":
				r.Attachments = append(r.Attachments, attachment{Field: "image_urls", Path: mask})
			case "wrong-model":
				r.Model = "gemini-3.1-flash-image"
			case "wrong-operation":
				r.Operation = "generate"
			case "corrupt-mask":
				os.WriteFile(mask, []byte("not a PNG"), 0600)
			}
			err := prepareProfileRequest(&r, profileService())
			if err == nil {
				_, _, _, err = taskRequestBody(r)
			}
			if err == nil {
				t.Fatal("unsafe mask request accepted")
			}
		})
	}
}

func TestReviewedMaskLiveProfilePreservesLocalTransport(t *testing.T) {
	mask := filepath.Join(t.TempDir(), "mask.png")
	if err := os.WriteFile(mask, fixturePNG(t), 0600); err != nil {
		t.Fatal(err)
	}
	r := taskRequest{Kind: "image", Model: "gpt-image-2.6-flare", Operation: "edit", MediaOperation: "masked_image_edit", Prompt: "fixture", Parameters: map[string]any{"image_urls": []any{"https://example.com/image.png"}}, Attachments: []attachment{{Field: "mask", Path: mask}}}
	profile, err := loadProfile(r, profileService())
	if err != nil {
		t.Fatal(err)
	}
	op := profile.Parameters.Operations["masked_image_edit"]
	input := op.Inputs["mask"]
	input.Transports = []string{"data_url"}
	op.Inputs["mask"] = input
	profile.Parameters.Operations["masked_image_edit"] = op
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.Method != "GET" || req.URL.Path != "/v1/media/models" {
			t.Error("unexpected request")
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": r.Model, "capabilities": []string{"image"}, "input_schema": profile.Parameters}}})
	}))
	defer server.Close()
	svc := fixtureService(server)
	svc.profilesRoot = t.TempDir()
	if err := prepareProfileRequest(&r, svc); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("profile gap must make exactly one catalog read")
	}
	if _, _, _, err := taskRequestBody(r); err != nil {
		t.Fatal(err)
	}
}
