package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestImageDimensionDefaultsReachJSONRequest(t *testing.T) {
	for _, tc := range []struct {
		name, model string
		input, want map[string]any
	}{
		{"gpt omitted", "gpt-image-2.6", nil, map[string]any{"image_size": "1K", "aspect_ratio": "1:1"}},
		{"gpt partial", "gpt-image-2.6", map[string]any{"image_size": "1K"}, map[string]any{"image_size": "1K", "aspect_ratio": "1:1"}},
		{"explicit landscape", "gpt-image-2.6", map[string]any{"aspect_ratio": "16:9"}, map[string]any{"image_size": "1K", "aspect_ratio": "16:9"}},
		{"explicit high resolution", "gpt-image-2.6", map[string]any{"image_size": "4K", "aspect_ratio": "9:16"}, map[string]any{"image_size": "4K", "aspect_ratio": "9:16"}},
		{"gemini", "gemini-nano-banana-2.1", nil, map[string]any{"image_size": "1K", "aspect_ratio": "1:1"}},
		{"grok own fields", "grok-imagine-image-2.0", nil, map[string]any{"resolution": "1k", "aspect_ratio": "16:9"}},
		{"seedream one size expression", "seedream-4-5-251128", nil, map[string]any{"image_size": "2K", "aspect_ratio": "auto"}},
		{"explicit pixel size", "seedream-4-5-251128", map[string]any{"size": "2048x1024"}, map[string]any{"size": "2048x1024"}},
		{"no declared defaults", "qwen-image-3.0", nil, map[string]any{}},
		{"unknown model", "fixture-unknown-image", nil, map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := taskRequest{Kind: "image", Operation: "generate", Model: tc.model, Prompt: "fixture", Parameters: tc.input}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r.Parameters, tc.want) {
				t.Fatalf("effective parameters = %v, want %v", r.Parameters, tc.want)
			}
			path, contentType, body, err := taskRequestBody(r)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.NewDecoder(body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if path != "/v1/images/generations" || contentType != "application/json" || payload["async"] != true || payload["model"] != tc.model || payload["prompt"] != "fixture" {
				t.Fatal("defaults changed the model, prompt or submission protocol")
			}
			delete(payload, "async")
			delete(payload, "model")
			delete(payload, "prompt")
			if !reflect.DeepEqual(payload, tc.want) {
				t.Fatalf("wire parameters = %v, want %v", payload, tc.want)
			}
		})
	}
}

func TestImageDimensionDefaultsRespectDeclaredPrecedence(t *testing.T) {
	profile := modelProfile{ID: "fixture-dimension-precedence", Capability: "image"}
	if err := json.Unmarshal([]byte(`{"properties":{"image_size":{"type":"string","enum":["1K","2K"],"default":"1K"},"aspect_ratio":{"type":"string","enum":["1:1","16:9"],"default":"1:1"},"resolution":{"type":"string","enum":["1k","2k"],"default":"1k"}},"constraints":{"size_expression_precedence":{"order":["image_size+aspect_ratio","resolution"]}}}`), &profile.Parameters); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		input, want map[string]any
	}{
		{"declared default expression", nil, map[string]any{"image_size": "1K", "aspect_ratio": "1:1"}},
		{"explicit complete expression wins over defaults", map[string]any{"resolution": "2k", "aspect_ratio": "16:9"}, map[string]any{"resolution": "2k", "aspect_ratio": "16:9"}},
		{"partial expression completes", map[string]any{"aspect_ratio": "16:9"}, map[string]any{"image_size": "1K", "aspect_ratio": "16:9"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, transport := catalogProfileService(t, profile)
			svc.profilesRoot = writeInstalledProfileFixture(t, profile)
			r := taskRequest{Kind: "image", Operation: "generate", Model: profile.ID, Prompt: "fixture", Parameters: tc.input}
			if err := prepareProfileRequest(&r, svc); err != nil {
				t.Fatal(err)
			}
			if transport.calls != 0 || !reflect.DeepEqual(r.Parameters, tc.want) {
				t.Fatalf("calls=%d parameters=%v want=%v", transport.calls, r.Parameters, tc.want)
			}
		})
	}
}

func TestImageDimensionDefaultsValidateBeforeNetworking(t *testing.T) {
	for _, scenario := range []string{"invalid default enum", "invalid default type", "incompatible default ratio", "explicit invalid value", "explicit null"} {
		t.Run(scenario, func(t *testing.T) {
			profile := readInstalledProfileFixture(t, "image", "gpt-image-2.6")
			r := taskRequest{Kind: "image", Operation: "generate", Model: profile.ID, Prompt: "fixture"}
			switch scenario {
			case "invalid default enum":
				profile.Parameters.Properties["image_size"]["default"] = "8K"
			case "invalid default type":
				profile.Parameters.Properties["image_size"]["default"] = 1024
			case "incompatible default ratio":
				profile.Parameters.Constraints["aspect_ratio_by_image_size"]["1K"] = []any{"16:9"}
			case "explicit invalid value":
				r.Parameters = map[string]any{"image_size": "8K"}
			case "explicit null":
				r.Parameters = map[string]any{"image_size": nil}
			}
			svc, transport := catalogProfileService(t, profile)
			svc.profilesRoot = writeInstalledProfileFixture(t, profile)
			if prepareProfileRequest(&r, svc) == nil || transport.calls != 0 {
				t.Fatal("invalid dimensions passed validation or caused a catalog read")
			}
		})
	}
}

func TestImageDimensionDefaultsDoNotAddExcludedOrNonImageFields(t *testing.T) {
	for _, scenario := range []string{"unsupported input", "unsupported operation parameter", "different parameter mode", "no defaults", "video", "continuation"} {
		t.Run(scenario, func(t *testing.T) {
			profile := readInstalledProfileFixture(t, "image", "gpt-image-2.6")
			r := taskRequest{Kind: "image", Operation: "generate", Model: profile.ID, Prompt: "fixture"}
			want := map[string]any{"aspect_ratio": "1:1"}
			switch scenario {
			case "unsupported input":
				profile.Parameters.Constraints["unsupported_inputs"] = map[string]any{"image_size": true}
			case "unsupported operation parameter":
				profile.Parameters.Constraints["unsupported_parameters_by_operation"] = map[string]any{"": []any{"image_size"}}
			case "different parameter mode":
				profile.Parameters.Constraints["parameter_modes"] = map[string]any{"image_size": []any{"image_edit"}}
			case "no defaults":
				delete(profile.Parameters.Properties["image_size"], "default")
				delete(profile.Parameters.Properties["aspect_ratio"], "default")
				want = map[string]any{}
			case "video":
				profile.Capability, r.Kind = "video", "video"
				want = map[string]any{}
			case "continuation":
				r.Operation = "continue"
				want = map[string]any{}
			}
			svc, transport := catalogProfileService(t, profile)
			svc.profilesRoot = writeInstalledProfileFixture(t, profile)
			if err := prepareProfileRequest(&r, svc); err != nil {
				t.Fatal(err)
			}
			if transport.calls != 0 || !reflect.DeepEqual(r.Parameters, want) {
				t.Fatalf("calls=%d parameters=%v want=%v", transport.calls, r.Parameters, want)
			}
		})
	}
}

func TestImageDimensionDefaultsReachEditTransports(t *testing.T) {
	for _, tc := range []struct {
		model, field string
		json         bool
	}{
		{"gpt-image-2.6", "image", false},
		{"gemini-nano-banana-2.1", "images", true},
	} {
		t.Run(tc.model, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "reference.png")
			if err := os.WriteFile(file, fixturePNG(t), 0600); err != nil {
				t.Fatal(err)
			}
			r := taskRequest{Kind: "image", Operation: "edit", Model: tc.model, Prompt: "fixture", Attachments: []attachment{{Field: tc.field, Path: file}}}
			if err := validateTaskRequest(r); err != nil {
				t.Fatal(err)
			}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			_, contentType, body, err := taskRequestBody(r)
			if err != nil {
				t.Fatal(err)
			}
			if closer, ok := body.(io.Closer); ok {
				defer closer.Close()
			}
			if tc.json {
				var payload map[string]any
				if err := json.NewDecoder(body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if payload["image_size"] != "1K" || payload["aspect_ratio"] != "1:1" || payload[tc.field] == nil {
					t.Fatal("JSON edit lost dimensions or reference")
				}
			} else {
				_, params, err := mime.ParseMediaType(contentType)
				if err != nil {
					t.Fatal(err)
				}
				form, err := multipart.NewReader(body, params["boundary"]).ReadForm(1 << 20)
				if err != nil {
					t.Fatal(err)
				}
				defer form.RemoveAll()
				if !reflect.DeepEqual(form.Value["image_size"], []string{"1K"}) || !reflect.DeepEqual(form.Value["aspect_ratio"], []string{"1:1"}) || len(form.File[tc.field]) != 1 {
					t.Fatal("multipart edit lost dimensions or reference")
				}
			}
		})
	}
}

func TestImageDimensionDefaultsSubmitOnceAndPersistEffectiveParameters(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		t.Run(map[bool]string{false: "short request", true: "recorded request"}[recorded], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if r.Method != http.MethodPost || r.URL.Path != "/v1/images/generations" || payload["image_size"] != "1K" || payload["aspect_ratio"] != "1:1" {
					t.Error("submission omitted defaults or added a preflight request")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_, _ = io.WriteString(w, `{"id":"fixture-image-defaults","object":"image","status":"pending"}`)
			}))
			defer server.Close()
			r := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2.6", Prompt: "fixture"}
			var output bytes.Buffer
			path := filepath.Join(t.TempDir(), "task.json")
			var err error
			if recorded {
				err = executeRecordedTask("submit", path, r, 0, "", &output, fixtureService(server))
			} else {
				data, marshalErr := json.Marshal(r)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				err = executeTask(bytes.NewReader(data), &output, fixtureService(server))
			}
			if err != nil {
				t.Fatal(err)
			}
			receipt := decodeReceipt(t, &output)
			if calls != 1 || receipt.SubmissionOutcome != "accepted" || receipt.Parameters["image_size"] != "1K" || receipt.Parameters["aspect_ratio"] != "1:1" {
				t.Fatal("receipt does not preserve the single effective submission")
			}
			if recorded {
				record, err := loadTaskRecord(path)
				if err != nil {
					t.Fatal(err)
				}
				if record.Parameters["image_size"] != "1K" || record.Parameters["aspect_ratio"] != "1:1" {
					t.Fatal("task record omitted effective defaults")
				}
			}
		})
	}
}
