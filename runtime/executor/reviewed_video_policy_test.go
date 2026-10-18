package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func grokAuditEntry(t *testing.T, id string) map[string]any {
	t.Helper()
	profile := readInstalledProfileFixture(t, "video", "grok-imagine-video")
	body, err := json.Marshal(profile.Parameters)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatal(err)
	}
	delete(jsonObject(schema["constraints"]), "unsupported_parameters_by_operation")
	return map[string]any{"id": id, "capabilities": []any{"video"}, "input_schema": schema}
}

func TestReviewedGrokEditingBoundary(t *testing.T) {
	for _, id := range []string{"grok-imagine-video-1.5", "grok-imagine-video-1.5-preview"} {
		t.Run(id, func(t *testing.T) {
			entry := grokAuditEntry(t, id)
			query := modelQuery{Kind: "video", Model: id, Operation: "video_edit"}
			if modelQueryMatches(query, entry) {
				t.Error("stale live catalog advertised editing for a closed model")
			}
			projected := projectCatalog(map[string]any{"data": []any{entry}}).(map[string]any)
			schema := jsonObject(jsonObject(jsonArray(projected["data"])[0])["input_schema"])
			if jsonObject(schema["operations"])["video_edit"] != nil {
				t.Error("unfiltered discovery restored the closed edit operation")
			}
			for _, operation := range []string{"edit", "generate"} {
				profile := readInstalledProfileFixture(t, "video", "grok-imagine-video")
				profile.ID = id
				svc, transport := catalogProfileService(t, profile)
				svc.profilesRoot = t.TempDir() // Missing or old installations cannot bypass the boundary.
				request := taskRequest{Kind: "video", Operation: operation, Model: id, Prompt: "fixture",
					MediaOperation: "video_edit", Attachments: []attachment{{Field: "video", Path: "/fixture/source.mp4"}}}
				if prepareProfileRequest(&request, svc) == nil || transport.calls != 0 {
					t.Error("closed editing must stop before catalog reads or submission")
				}
			}
		})
	}
}

func TestReviewedGrokEditControls(t *testing.T) {
	for _, field := range []string{"duration", "seconds", "aspect_ratio", "aspectRatio", "resolution"} {
		t.Run(field, func(t *testing.T) {
			value := any("16:9")
			if field == "duration" || field == "seconds" {
				value = 5
			} else if field == "resolution" {
				value = "720p"
			}
			profile := readInstalledProfileFixture(t, "video", "grok-imagine-video")
			svc, transport := catalogProfileService(t, profile)
			svc.profilesRoot = t.TempDir()
			request := taskRequest{Kind: "video", Operation: "edit", Model: profile.ID, Prompt: "fixture",
				Parameters: map[string]any{field: value}, Attachments: []attachment{{Field: "video", Path: "/fixture/source.mp4"}}}
			if prepareProfileRequest(&request, svc) == nil || transport.calls != 0 {
				t.Error("source-controlled edit parameters must fail locally without silently dropping them")
			}
			entry := grokAuditEntry(t, profile.ID)
			// Include aliases in this deliberately stale schema.
			jsonObject(jsonObject(entry["input_schema"])["properties"])[field] = map[string]any{"type": "string"}
			if field == "duration" || field == "seconds" {
				jsonObject(jsonObject(entry["input_schema"])["properties"])[field] = map[string]any{"type": "integer"}
			}
			if modelQueryMatches(modelQuery{Kind: "video", Operation: "video_edit", Parameters: request.Parameters}, entry) {
				t.Error("query and submission disagreed about edit controls")
			}
		})
	}
	// The same controls remain valid for generation; a bare standard edit remains valid.
	for _, operation := range []string{"generate", "edit"} {
		request := taskRequest{Kind: "video", Operation: operation, Model: "grok-imagine-video", Prompt: "fixture"}
		if operation == "generate" {
			request.Parameters = map[string]any{"duration": 5, "aspect_ratio": "16:9", "resolution": "720p"}
		} else {
			request.Attachments = []attachment{{Field: "video", Path: "/fixture/source.mp4"}}
		}
		profile := readInstalledProfileFixture(t, "video", request.Model)
		svc, transport := catalogProfileService(t, profile)
		if err := prepareProfileRequest(&request, svc); err != nil || transport.calls != 0 {
			t.Fatalf("valid %s changed: %v", operation, err)
		}
	}
}

func TestGrokRatiosFollowPublishedSchema(t *testing.T) {
	for _, ratio := range []string{"21:9", "5:2"} {
		profile := readInstalledProfileFixture(t, "video", "grok-imagine-video")
		request := taskRequest{Kind: "video", Operation: "generate", Model: profile.ID, Prompt: "fixture",
			Parameters: map[string]any{"aspect_ratio": ratio}}
		// Keep an explicit old-catalog fixture after production adds the ratios.
		profile.Parameters.Properties["aspect_ratio"]["enum"] = []any{"16:9"}
		svc, transport := catalogProfileService(t, profile)
		svc.profilesRoot = writeInstalledProfileFixture(t, profile)
		if prepareProfileRequest(&request, svc) == nil || transport.calls != 0 {
			t.Fatal("unpublished ratio must be rejected from the installed enum without network access")
		}
		// Positive case: a subsequently published schema can declare the new ratios.
		profile.Parameters.Properties["aspect_ratio"]["enum"] = []any{ratio}
		svc, transport = catalogProfileService(t, profile)
		svc.profilesRoot = t.TempDir()
		if err := prepareProfileRequest(&request, svc); err != nil || transport.calls != 1 {
			t.Fatalf("published ratio was not preserved: %v", err)
		}
		// Negative case: public metadata errors never trigger a paid request.
		svc, transport = catalogProfileService(t, profile)
		svc.profilesRoot = t.TempDir()
		transport.status = http.StatusServiceUnavailable
		if prepareProfileRequest(&request, svc) == nil || transport.calls != 1 {
			t.Fatal("unconfirmed ratio must remain unavailable")
		}
	}
}

func TestReviewedGrokDispatcherAndExistingTask(t *testing.T) {
	for _, id := range []string{"grok-imagine-video", "grok-imagine-video-1.5", "grok-imagine-video-1.5-preview"} {
		t.Run(id, func(t *testing.T) {
			calls := 0
			svc := service{
				baseURL: "https://fixture.invalid",
				client: &http.Client{Transport: evaluationTransport(func(request *http.Request) (*http.Response, error) {
					calls++
					if request.Method != http.MethodGet || request.URL.Path != "/v1/videos/existing-grok-edit" {
						t.Fatalf("only the existing task may be read: %s %s", request.Method, request.URL.Path)
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Request: request,
						Body: io.NopCloser(strings.NewReader(`{"id":"existing-grok-edit","status":"completed"}`))}, nil
				})},
				profilesRoot: t.TempDir(),
			}
			output := new(bytes.Buffer)
			request := taskRequest{Kind: "video", Operation: "edit", Model: id, Prompt: "fixture",
				Parameters:  map[string]any{"duration": 5},
				Attachments: []attachment{{Field: "video", Path: writeFixture(t, "source.webm", string(fixtureWebM(false, false)))}}}
			if err := validateTaskRequest(request); err != nil {
				t.Fatalf("fixture must reach reviewed model validation: %v", err)
			}
			body, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			if executeTask(bytes.NewReader(body), output, svc) == nil || calls != 0 {
				t.Fatal("invalid new edit must fail before catalog reads or POST")
			}
			if result := decodeReceipt(t, output); result.FailurePhase != "validation" || result.TaskID != "" {
				t.Fatal("rejected edit must have a safe validation receipt without task identity")
			} else if (id == "grok-imagine-video" && !strings.Contains(result.ErrorMessage, "inherits")) ||
				(id != "grok-imagine-video" && !strings.Contains(result.ErrorMessage, "unavailable")) {
				t.Fatal("dispatcher stopped before exercising the reviewed model restriction")
			}
			output.Reset()
			input := `{"kind":"video","task_id":"existing-grok-edit","model":"` + id + `","original_operation":"video_edit","parameters":{"duration":5}}`
			if err := executeExistingTask("status", strings.NewReader(input), output, svc); err != nil {
				t.Fatal(err)
			}
			if result := decodeReceipt(t, output); !result.OK || result.Model != id || result.TaskID != "existing-grok-edit" ||
				result.OriginalOperation != "video_edit" || calls != 1 {
				t.Fatal("reviewed submission restrictions changed existing-task recovery")
			}
		})
	}
}

func TestReviewedMediaSchemaPreservesEvidenceAndRestrictions(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{"aspect_ratio": map[string]any{"enum": []any{"16:9"}}},
		"operations": map[string]any{"video_edit": map[string]any{"declared": true}, "image_to_video": map[string]any{"declared": true}},
		"constraints": map[string]any{
			"unsupported_operations":              map[string]any{"unrelated_operation": true},
			"unsupported_parameters_by_operation": map[string]any{"video_edit": []any{"already_excluded"}},
		},
		"future_field": "preserved",
	}
	before, _ := json.Marshal(schema)
	projected := reviewedMediaSchema("grok-imagine-video-1.5", schema)
	if jsonObject(projected["operations"])["video_edit"] != nil ||
		jsonObject(jsonObject(projected["constraints"])["unsupported_operations"])["unrelated_operation"] != true ||
		projected["future_field"] != "preserved" {
		t.Fatal("projection must narrow editing while preserving unrelated declarations")
	}
	projected = reviewedMediaSchema("grok-imagine-video", schema)
	fields := array(jsonObject(jsonObject(projected["constraints"])["unsupported_parameters_by_operation"])["video_edit"])
	if !hasValue(fields, "already_excluded") || !hasValue(fields, "duration") {
		t.Fatal("reviewed policy must merge with, not replace, existing exclusions")
	}
	after, _ := json.Marshal(schema)
	if !bytes.Equal(before, after) {
		t.Fatal("reviewed policy mutated original API evidence")
	}
	projected = reviewedMediaSchema("grok-imagine-video-other", schema)
	unknown, _ := json.Marshal(projected)
	if !bytes.Equal(before, unknown) || validateReviewedMediaOperation("grok-imagine-video-other", "video_edit", map[string]any{"duration": 5}) != nil {
		t.Fatal("exact-model restrictions leaked into another model ID")
	}
}
