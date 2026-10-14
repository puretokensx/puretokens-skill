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

// Offline command trajectories: the test explicitly issues discovery and the
// newly authorized submission. This is not an LLM consent or real-host test.
func TestExplicitAlternativeKeepsOriginalTaskAndReference(t *testing.T) {
	for _, tc := range []struct {
		kind, first, alternative, operation, mediaOperation, route, oldField, newField string
		parameters                                                                     map[string]any
	}{
		{"image", "gpt-image-2", "nano-banana-2", "edit", "image_edit", "/v1/images/edits", "image", "image", map[string]any{"aspect_ratio": "1:1"}},
		{"video", "grok-imagine-video-1.5", "minimax_h3", "generate", "image_to_video", "/v1/videos", "image", "first_frame_image", map[string]any{"duration": 5}},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			root := t.TempDir()
			originalRecord, newRecord := filepath.Join(root, "original.json"), filepath.Join(root, "alternative.json")
			media := fixturePNG(t)
			input := writeFixture(t, "reference.png", string(media))
			prompt := "只改背景，保留文字：原文 A\nDo not rewrite."
			var calls []string
			posts := 0
			profile := readInstalledProfileFixture(t, tc.kind, tc.alternative)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path)
				if r.Method == http.MethodGet && r.URL.Path == "/v1/media/models" {
					json.NewEncoder(w).Encode(map[string]any{"data": []any{
						map[string]any{"id": profile.ID, "capabilities": []string{tc.kind}, "input_schema": profile.Parameters},
						map[string]any{"id": "wrong-operation", "capabilities": []string{tc.kind}, "input_schema": map[string]any{"properties": map[string]any{}}},
					}})
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != tc.route {
					t.Errorf("unexpected recovery or automatic request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected", 500)
					return
				}
				posts++
				reader, err := r.MultipartReader()
				if err != nil {
					t.Error("reference was downgraded to text or JSON")
					http.Error(w, "invalid", 400)
					return
				}
				fields := map[string][]byte{}
				for {
					part, err := reader.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Error(err)
						return
					}
					if _, exists := fields[part.FormName()]; exists {
						t.Error("duplicate field")
					}
					fields[part.FormName()], err = io.ReadAll(part)
					if err != nil {
						t.Error(err)
						return
					}
				}
				model, field := tc.first, tc.oldField
				if posts == 2 {
					model, field = tc.alternative, tc.newField
				}
				if string(fields["model"]) != model || string(fields["prompt"]) != prompt || !bytes.Equal(fields[field], media) {
					t.Error("model selection changed the prompt, reference bytes or reference role")
				}
				expectedFields := 3 + len(tc.parameters)
				if tc.kind == "image" {
					expectedFields++
					if string(fields["async"]) != "true" {
						t.Error("image alternative changed the public asynchronous contract")
					}
				}
				if len(fields) != expectedFields {
					t.Error("new request contains dropped or stale task fields")
				}
				for name, value := range tc.parameters {
					encoded, _ := json.Marshal(value)
					if string(fields[name]) != strings.Trim(string(encoded), `"`) {
						t.Errorf("changed required parameter %s", name)
					}
				}
				if posts == 1 {
					io.WriteString(w, `{"id":"original_failed","status":"failed"}`)
				} else {
					io.WriteString(w, `{"id":"new_task","status":"pending"}`)
				}
			}))
			defer server.Close()
			svc := fixtureService(server)
			request := taskRequest{Kind: tc.kind, Operation: tc.operation, Model: tc.first, Prompt: prompt,
				MediaOperation: tc.mediaOperation, Parameters: tc.parameters, Attachments: []attachment{{Field: tc.oldField, Path: input}}}
			var out bytes.Buffer
			if err := executeRecordedTask("submit", originalRecord, request, 0, "", &out, svc); err == nil {
				t.Fatal("terminal failure must stop")
			}
			failed := decodeReceipt(t, &out)
			if posts != 1 || len(calls) != 1 || failed.TaskID != "original_failed" || failed.NextStep != "await_user" || failed.ReconciliationRequired {
				t.Fatalf("failure did not stop at the original task: %+v", failed)
			}
			original, err := os.ReadFile(originalRecord)
			if err != nil {
				t.Fatal(err)
			}
			// Explicit permission to inspect alternatives authorizes a single GET.
			query, _ := json.Marshal(modelQuery{Kind: tc.kind, Operation: tc.mediaOperation, Parameters: tc.parameters})
			out.Reset()
			if err := executeModelQuery(&out, svc, bytes.NewReader(query)); err != nil {
				t.Fatal(err)
			}
			var catalog map[string]any
			if err := json.Unmarshal(out.Bytes(), &catalog); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(modelQueryIDs(t, catalog), []string{tc.alternative}) || posts != 1 || len(calls) != 2 {
				t.Fatal("discovery included an incompatible model or submitted a task")
			}
			// A new-task command after separate consent cannot reuse the old record.
			request.Model, request.Attachments = tc.alternative, []attachment{{Field: tc.newField, Path: input}}
			out.Reset()
			if err := executeRecordedTask("submit", originalRecord, request, 0, "", &out, svc); err == nil || len(calls) != 2 {
				t.Fatal("new model overwrote the original task or submitted before record validation")
			}
			out.Reset()
			if err := executeRecordedTask("submit", newRecord, request, 0, "", &out, svc); err != nil {
				t.Fatal(err)
			}
			created := decodeReceipt(t, &out)
			if created.TaskID != "new_task" || created.Model != tc.alternative || created.NextStep != "wait" || posts != 2 {
				t.Fatalf("new task was not independently submitted once: %+v", created)
			}
			after, err := os.ReadFile(originalRecord)
			if err != nil || !bytes.Equal(original, after) {
				t.Fatal("original failure record changed")
			}
			if !reflect.DeepEqual(calls, []string{"POST " + tc.route, "GET /v1/media/models", "POST " + tc.route}) {
				t.Fatalf("unexpected requests or automatic follow-up: %v", calls)
			}
		})
	}
}

func TestIncompatibleAlternativeDoesNotDropReferencesOrSubmit(t *testing.T) {
	input := writeFixture(t, "reference.png", string(fixturePNG(t)))
	images := make([]attachment, 17)
	for i := range images {
		images[i] = attachment{Field: "image", Path: input}
	}
	for _, request := range []taskRequest{
		{Kind: "image", Operation: "edit", Model: "nano-banana-2", Prompt: "Use all seventeen references.",
			MediaOperation: "image_edit", Attachments: images},
		{Kind: "video", Operation: "generate", Model: "grok-imagine-video-1.5", Prompt: "Keep both the opening and closing frame.",
			MediaOperation: "image_to_video", Attachments: []attachment{{Field: "image", Path: input}, {Field: "last_frame_image", Path: input}}},
	} {
		t.Run(request.Kind, func(t *testing.T) {
			calls := 0
			profile := readInstalledProfileFixture(t, request.Kind, request.Model)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				// A missing field may trigger one declared profile-gap lookup;
				// the unchanged catalog must not cause the last frame to be dropped.
				if request.Kind == "video" && r.Method == http.MethodGet && r.URL.Path == "/v1/media/models" {
					json.NewEncoder(w).Encode(map[string]any{"data": []any{
						map[string]any{"id": profile.ID, "capabilities": []string{request.Kind}, "input_schema": profile.Parameters},
					}})
					return
				}
				t.Error("incompatible alternative submitted media or made an undeclared request")
				http.Error(w, "unexpected", 500)
			}))
			defer server.Close()
			path := filepath.Join(t.TempDir(), "alternative.json")
			var out bytes.Buffer
			if err := executeRecordedTask("submit", path, request, 0, "", &out, fixtureService(server)); err == nil {
				t.Fatal("incompatible media requirements were silently dropped")
			}
			result := decodeReceipt(t, &out)
			wantedReads := 0
			if request.Kind == "video" {
				wantedReads = 1
			}
			if calls != wantedReads || result.FailurePhase != "validation" || result.TaskID != "" {
				t.Fatal("incompatible alternative did not stop before submission")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("invalid alternative created a task record")
			}
		})
	}
}
