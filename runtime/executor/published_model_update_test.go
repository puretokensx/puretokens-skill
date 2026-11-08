package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPublishedModelUpdate(t *testing.T) {
	for _, tc := range []struct {
		name, kind, model, operation, mediaOperation string
		parameters                                   map[string]any
		fields                                       []string
		count                                        int
		valid                                        bool
	}{
		{name: "gemini-local-typed-input", kind: "image", model: "gemini-nano-banana-2.1", operation: "edit", fields: []string{"images"}, valid: true},
		{name: "gemini-https", kind: "image", model: "gemini-nano-banana-2.1", operation: "edit", parameters: map[string]any{"images": []any{"https://example.com/ref.png"}}, valid: true},
		{name: "gemini-missing", kind: "image", model: "gemini-nano-banana-2.1", operation: "edit"},
		{name: "veo-local-typed-pair", kind: "video", model: "veo-3.1-fast", mediaOperation: "first_last_frame_video", fields: []string{"first_frame_image", "last_frame_image"}, valid: true},
		{name: "veo-local-missing", kind: "video", model: "veo-3.1-fast", mediaOperation: "first_last_frame_video", fields: []string{"first_frame_image"}},
		{name: "veo-reference-1080", kind: "video", model: "veo-3.1-fast", parameters: map[string]any{"reference_images": []any{"https://example.com/ref.png"}, "resolution": "1080p"}},
		{name: "wan-audio-boolean", kind: "video", model: "wan3.0-video", parameters: map[string]any{"audio": true, "duration": -1}, valid: true},
		{name: "wan-unreviewed-native", kind: "video", model: "wan3.0-video", parameters: map[string]any{"input": map[string]any{"media": "fixture"}}},
		{name: "gpt-svg", kind: "image", model: "gpt-image-2.6", parameters: map[string]any{"output_format": "svg"}, valid: true},
		{name: "gpt-gif", kind: "image", model: "gpt-image-2.6", parameters: map[string]any{"output_format": "gif"}, valid: true},
		{name: "gpt-base-edit-gif", kind: "image", model: "gpt-image-2.6", operation: "edit", fields: []string{"image"}, parameters: map[string]any{"output_format": "gif"}},
		{name: "gpt-flare-edit-gif", kind: "image", model: "gpt-image-2.6-flare", operation: "edit", fields: []string{"image"}, parameters: map[string]any{"output_format": "gif"}, valid: true},
		{name: "gpt-flare-gif-2k", kind: "image", model: "gpt-image-2.6-flare", parameters: map[string]any{"output_format": "gif", "image_size": "2K"}},
		{name: "wan-standard-five", kind: "image", model: "wan2.7-image", count: 5},
		{name: "wan-sequential-five", kind: "image", model: "wan2.7-image", count: 5, parameters: map[string]any{"enable_sequential": true}, valid: true},
		{name: "wan-pro-sequence-4k", kind: "image", model: "wan2.7-image-pro", parameters: map[string]any{"enable_sequential": true, "size": "4K"}},
		{name: "wan-pro-explicit-pixels", kind: "image", model: "wan2.7-image-pro", parameters: map[string]any{"size": "1024x1024"}, valid: true},
		{name: "seedream-standard-two", kind: "image", model: "seedream-4-5-251128", count: 2},
		{name: "seedream-sequence-two", kind: "image", model: "seedream-4-5-251128", count: 2, parameters: map[string]any{"sequential_image_generation": "auto", "sequential_image_generation_options": map[string]any{"max_images": 2}}, valid: true},
		{name: "seedream-sequence-count-mismatch", kind: "image", model: "seedream-4-5-251128", count: 2, parameters: map[string]any{"sequential_image_generation": "auto"}},
		{name: "seedream-size-conflict", kind: "image", model: "seedream-4-5-251128", parameters: map[string]any{"size": "2K", "image_size": "1K"}},
		{name: "seedream-bad-dimensions", kind: "image", model: "seedream-4-5-251128", parameters: map[string]any{"size": "1025x2048"}},
		{name: "seedream-good-dimensions", kind: "image", model: "seedream-4-5-251128", parameters: map[string]any{"size": "1024x2048"}, valid: true},
		{name: "vidu-text-voice", kind: "video", model: "viduq3-pro", parameters: map[string]any{"voice_id": "fixture"}},
		{name: "vega-last-frame-deliverable", kind: "video", model: "Super Vega 1.0-pro", parameters: map[string]any{"return_last_frame": true}, valid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.operation
			if op == "" {
				op = "generate"
			}
			r := taskRequest{Kind: tc.kind, Model: tc.model, Operation: op, MediaOperation: tc.mediaOperation, Prompt: "fixture", Parameters: tc.parameters, RequestedCount: tc.count}
			for _, field := range tc.fields {
				r.Attachments = append(r.Attachments, attachment{Field: field})
			}
			err := prepareProfileRequest(&r, profileService())
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestDeclaredAlternateMultipart(t *testing.T) {
	var p modelProfile
	json.Unmarshal([]byte(`{"id":"fixture","capability":"image","parameterSchema":{"operations":{"image_edit":{"request":{"method":"POST","path":"/v1/images/edits","contentType":"application/json"},"requiredBodyFields":["model","prompt","images"],"inputs":{"images":{"field":"images","required":true,"minItems":1,"maxItems":2,"transports":["https_url","multipart_file"]}}}}}}`), &p)
	svc, transport := catalogProfileService(t, p)
	svc.profilesRoot = writeInstalledProfileFixture(t, p)
	r := taskRequest{Kind: "image", Model: "fixture", Operation: "edit", Prompt: "fixture", Attachments: []attachment{{Field: "images"}}}
	if err := prepareProfileRequest(&r, svc); err != nil || transport.calls != 0 {
		t.Fatalf("declared transport: %v calls=%d", err, transport.calls)
	}
}

func TestPublishedSVGQueryReviewed(t *testing.T) {
	r := taskRequest{Kind: "image", Model: "gpt-image-2.6", Operation: "generate", Prompt: "fixture"}
	p, err := loadProfile(r, profileService())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p.Parameters)
	var schema any
	json.Unmarshal(raw, &schema)
	entry := map[string]any{"id": r.Model, "capabilities": []any{"image"}, "input_schema": schema}
	if !modelQueryMatches(modelQuery{Kind: "image", Parameters: map[string]any{"output_format": "svg"}}, entry) {
		t.Fatal("query omitted reviewed SVG")
	}
}

func TestPublishedGIFContinuationPreservesFormat(t *testing.T) {
	r := taskRequest{Kind: "image", Model: "gpt-image-2.6", Operation: "generate", Prompt: "fixture", Parameters: map[string]any{"output_format": "gif"}}
	if err := prepareProfileRequest(&r, profileService()); err != nil {
		t.Fatal(err)
	}
	if recordFromRequest(r).Parameters["output_format"] != "gif" || taskReceipt(r, "fixture", "pending").Parameters["output_format"] != "gif" {
		t.Fatal("GIF format lost in continuation")
	}
}

func TestPublishedDiscoveryRespectsExecutorOutputBound(t *testing.T) {
	p := readInstalledProfileFixture(t, "image", "wan2.7-image")
	for _, count := range []int{1, 8, 9, 12} {
		matched := modelSchemaMatches(modelQuery{Kind: "image", Parameters: map[string]any{"n": count, "enable_sequential": true}}, []any{"image"}, p.Parameters)
		if matched != (count <= maxImageCount) {
			t.Errorf("count=%d matched=%v: discovery must respect the executor output limit", count, matched)
		}
	}
}

func TestPublishedPolicyProjectionPreservesEvidence(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"output_format": map[string]any{"enum": []any{"png", "gif", "svg"}},
			"input":         map[string]any{"type": "object"},
		},
		"operations": map[string]any{
			"image_edit": map[string]any{"inputs": map[string]any{
				"images": map[string]any{"transports": []any{"https_url", "multipart_file", "data_url"}},
			}},
			"native_generation": map[string]any{"declared": true},
		},
	}
	before, _ := json.Marshal(schema)
	for _, id := range []string{"gemini-3.0-pro-image", "gemini-3.1-flash-image", "gemini-nano-banana-2.1", "veo-3.1", "veo-3.1-fast", "veo-3.1-lite", "omni-flash"} {
		projected := reviewedMediaSchema(id, schema)
		inputs := jsonObject(jsonObject(jsonObject(projected["operations"])["image_edit"])["inputs"])
		transports := array(jsonObject(inputs["images"])["transports"])
		if len(transports) != 2 || transports[0] != "https_url" || transports[1] != "local_file_data_url" {
			t.Fatalf("%s restored an unsupported transport", id)
		}
	}
	wan := reviewedMediaSchema("wan3.0-video", schema)
	if jsonObject(jsonObject(wan["properties"])["input"])["type"] != "json" ||
		jsonObject(jsonObject(jsonObject(wan["operations"])["native_generation"])["request"])["path"] != "/v1/videos" ||
		jsonObject(jsonObject(wan["constraints"])["unsupported_inputs"])["idempotency_key"] != true {
		t.Fatal("Wan projection lost its reviewed native operation or idempotency restriction")
	}
	unknown := reviewedMediaSchema("fixture-unrestricted-model", schema)
	formats := array(jsonObject(jsonObject(unknown["properties"])["output_format"])["enum"])
	if len(formats) != 2 || !hasValue(formats, "png") || !hasValue(formats, "gif") || jsonObject(unknown["properties"])["input"] == nil {
		t.Fatal("global format restriction changed unrelated model fields")
	}
	after, _ := json.Marshal(schema)
	if !bytes.Equal(before, after) {
		t.Fatal("reviewed projection mutated original catalog evidence")
	}
}

func TestPublishedStepAudioParameters(t *testing.T) {
	calls := 0
	result, err, _ := audioCLI(t, "speech", func(r map[string]any) { r["volume"] = 0.1 }, func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["volume"] != 0.1 {
			t.Error("volume omitted or changed")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
	})
	if err != nil || calls != 1 || result["ok"] != true {
		t.Fatalf("volume request: %v calls=%d %v", result, calls, err)
	}
}

func TestPublishedStepTranscriptionText(t *testing.T) {
	for _, model := range []string{"stepaudio-2.5-asr", "step-asr"} {
		t.Run(model, func(t *testing.T) {
			calls := 0
			result, err, raw := audioCLI(t, "transcribe", func(r map[string]any) {
				r["model"] = model
				r["hotwords"] = []string{"PureToken", "fixture"}
				r["response_format"] = "text"
			}, func(r *http.Request) (*http.Response, error) {
				calls++
				reader, e := r.MultipartReader()
				if e != nil {
					t.Fatal(e)
				}
				form, e := reader.ReadForm(maxAudioInput)
				if e != nil {
					t.Fatal(e)
				}
				defer form.RemoveAll()
				if form.Value["response_format"][0] != "text" || form.Value["hotwords"][0] != `["PureToken","fixture"]` {
					t.Error("wrong transcription fields")
				}
				response := evaluationHTTP(200, "recognized synthetic-audio-token")
				response.Header.Set("Content-Type", "text/plain; charset=utf-8")
				return response, nil
			})
			if err != nil || calls != 1 || result["ok"] != true || strings.Contains(raw, "synthetic-audio-token") {
				t.Fatalf("text request: %v calls=%d %v", result, calls, err)
			}
		})
	}
}

func TestPublishedAudioParametersFailBeforeNetwork(t *testing.T) {
	for _, tc := range []struct {
		name, op string
		change   func(map[string]any)
	}{
		{"volume-low", "speech", func(r map[string]any) { r["volume"] = 0.09 }},
		{"volume-high", "speech", func(r map[string]any) { r["volume"] = 2.01 }},
		{"volume-other-model", "speech", func(r map[string]any) { r["model"] = "step-tts-2"; r["volume"] = 1 }},
		{"hotwords-string", "transcribe", func(r map[string]any) { r["hotwords"] = "a,b" }},
		{"hotwords-empty-word", "transcribe", func(r map[string]any) { r["hotwords"] = []string{" "} }},
		{"hotwords-max-model", "transcribe", func(r map[string]any) { r["model"] = "stepaudio-3-asr-max"; r["hotwords"] = []string{"fixture"} }},
		{"text-max-model", "transcribe", func(r map[string]any) { r["model"] = "stepaudio-3-asr-max"; r["response_format"] = "text" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			result, err, _ := audioCLI(t, tc.op, tc.change, func(*http.Request) (*http.Response, error) { calls++; return evaluationHTTP(500, ""), nil })
			if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" {
				t.Fatalf("invalid extension sent: %v", result)
			}
		})
	}
}

func TestPublishedTextTranscriptionRejectsInvalidResponseOnce(t *testing.T) {
	for _, tc := range []struct{ name, mime, body string }{
		{"wrong-mime", "application/json", `{"text":"fixture"}`},
		{"empty", "text/plain", " "},
		{"invalid-utf8", "text/plain", string([]byte{0xff})},
		{"nul", "text/plain", "fixture\x00text"},
		{"oversized", "text/plain", strings.Repeat("x", 128<<10+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			result, err, _ := audioCLI(t, "transcribe", func(r map[string]any) { r["response_format"] = "text" }, func(*http.Request) (*http.Response, error) {
				calls++
				r := evaluationHTTP(200, tc.body)
				r.Header.Set("Content-Type", tc.mime)
				return r, nil
			})
			if err == nil || calls != 1 || result["submission_outcome"] != "unknown" || result["next_step"] != "await_user" {
				t.Fatalf("invalid text response: %v calls=%d", result, calls)
			}
		})
	}
}
