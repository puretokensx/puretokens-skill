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
	"reflect"
	"strings"
	"testing"
	"time"
)

// This is a command trajectory with simulated host handoff, not an LLM planner
// or real-client acceptance. It deliberately uses only local synthetic media.
func TestImageToVideoKeepsBytesAndRecoversOnlyTheFailedStep(t *testing.T) {
	for _, failure := range []string{"content", "unknown-submit"} {
		t.Run(failure, func(t *testing.T) {
			imageBytes, videoBytes := fixturePNG(t), fixtureWebM(false, false)
			var calls []string
			videoDownloads := 0
			imageHandedOff := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path)
				switch r.Method + " " + r.URL.Path {
				case "POST /v1/images/generations":
					var payload map[string]any
					if json.NewDecoder(r.Body).Decode(&payload) != nil || payload["prompt"] != "原样文字\nDo not rewrite." {
						t.Error("image prompt was changed")
					}
					io.WriteString(w, `{"id":"image_workflow","status":"pending"}`)
				case "GET /v1/images/image_workflow":
					io.WriteString(w, `{"id":"image_workflow","status":"completed"}`)
				case "GET /v1/images/image_workflow/content":
					w.Header().Set("Content-Type", "image/png")
					w.Write(imageBytes)
				case "POST /v1/videos":
					if !imageHandedOff {
						t.Error("next paid step preceded image handoff")
					}
					reader, err := r.MultipartReader()
					if err != nil {
						t.Error("first frame was downgraded to JSON/text")
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
							t.Error("duplicate first-frame field")
						}
						fields[part.FormName()], err = io.ReadAll(part)
						if err != nil {
							t.Error(err)
						}
					}
					if !bytes.Equal(fields["image"], imageBytes) || string(fields["prompt"]) != "缓慢推进，不加人物。" ||
						string(fields["model"]) != "grok-imagine-video-1.5" ||
						string(fields["duration"]) != "5" || len(fields) != 4 {
						t.Error("downstream request lost, transformed or added fields")
					}
					if failure == "unknown-submit" {
						http.Error(w, "synthetic upstream error", 503)
					} else {
						io.WriteString(w, `{"id":"video_workflow","status":"pending"}`)
					}
				case "GET /v1/videos/video_workflow":
					io.WriteString(w, `{"id":"video_workflow","status":"completed"}`)
				case "GET /v1/videos/video_workflow/content":
					videoDownloads++
					w.Header().Set("Content-Type", "video/webm")
					if videoDownloads == 1 {
						w.Write([]byte("not a video"))
					} else {
						w.Write(videoBytes)
					}
				default:
					t.Errorf("unexpected preflight/upload/replacement request: %s", r.URL.Path)
					http.Error(w, "unexpected", 500)
				}
			}))
			defer server.Close()
			svc := fixtureService(server)
			svc.profilesRoot = historicalProfilesRoot(t, "grok-imagine-video-1.5")
			now := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
			svc.now = func() time.Time { return now }
			svc.wait = func(_ context.Context, delay time.Duration) bool { now = now.Add(delay); return true }
			root := filepath.Join(t.TempDir(), "中文 artifacts with spaces")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			imageRecord, videoRecord := filepath.Join(root, "image.json"), filepath.Join(root, "video.json")
			invoke := func(command, record string, request taskRequest) (receipt, error) {
				t.Helper()
				var out bytes.Buffer
				dir := ""
				if command == "content" {
					dir = root
				}
				err := executeRecordedTask(command, record, request, 0, dir, &out, svc)
				return decodeReceipt(t, &out), err
			}
			must := func(command, record string, request taskRequest, next string) receipt {
				t.Helper()
				got, err := invoke(command, record, request)
				if err != nil || got.NextStep != next {
					t.Fatalf("%s: %v %+v", command, err, got)
				}
				return got
			}
			must("submit", imageRecord, taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2",
				Prompt: "原样文字\nDo not rewrite.", RequestedCount: 1}, "wait")
			must("resume", imageRecord, taskRequest{}, "content")
			image := must("content", imageRecord, taskRequest{}, "deliver")
			if len(image.DownloadedPaths) != 1 {
				t.Fatal("missing image file")
			}
			imagePath := image.DownloadedPaths[0]
			got, err := os.ReadFile(imagePath)
			if err != nil || !bytes.Equal(got, imageBytes) {
				t.Fatal("simulated host did not receive image bytes")
			}
			imageHandedOff = true
			must("delivered", imageRecord, taskRequest{}, "done")
			imageRecordBefore, err := os.ReadFile(imageRecord)
			if err != nil {
				t.Fatal(err)
			}
			videoRequest := taskRequest{Kind: "video", Operation: "generate", Model: "grok-imagine-video-1.5",
				Prompt: "缓慢推进，不加人物。", MediaOperation: "image_to_video",
				Parameters: map[string]any{"duration": 5}, Attachments: []attachment{{Field: "image", Path: imagePath}}}
			video, submitErr := invoke("submit", videoRecord, videoRequest)
			if failure == "unknown-submit" {
				if submitErr == nil || video.SubmissionOutcome != "unknown" || video.NextStep != "await_user" {
					t.Fatalf("uncertain downstream submission was not stopped: %+v", video)
				}
				before := len(calls)
				if _, err := invoke("resume", videoRecord, taskRequest{}); err == nil || len(calls) != before {
					t.Fatal("unknown submission attempted discovery or replacement")
				}
			} else {
				if submitErr != nil || video.NextStep != "wait" {
					t.Fatal("video did not return before polling")
				}
				must("resume", videoRecord, taskRequest{}, "content")
				broken, err := invoke("content", videoRecord, taskRequest{})
				if err == nil || broken.OK || broken.FailurePhase != "content" || len(broken.DownloadedPaths) != 0 ||
					broken.DeliveryStatus != "" || broken.NextStep != "await_user" {
					t.Fatalf("failed download looked delivered: %+v", broken)
				}
				before := len(calls)
				if _, err := invoke("delivered", videoRecord, taskRequest{}); err == nil || len(calls) != before {
					t.Fatal("failed download acknowledged")
				}
				// Explicit continuation downloads only this original video, not the image.
				delivered := must("content", videoRecord, taskRequest{}, "deliver")
				if len(delivered.DownloadedPaths) != 1 {
					t.Fatal("missing recovered video")
				}
				data, err := os.ReadFile(delivered.DownloadedPaths[0])
				if err != nil || !bytes.Equal(data, videoBytes) {
					t.Fatal("simulated video handoff failed")
				}
				must("delivered", videoRecord, taskRequest{}, "done")
			}
			imageRecordAfter, err := os.ReadFile(imageRecord)
			if err != nil || !bytes.Equal(imageRecordBefore, imageRecordAfter) {
				t.Fatal("downstream failure changed the completed image task")
			}
			expected := []string{"POST /v1/images/generations", "GET /v1/images/image_workflow",
				"GET /v1/images/image_workflow/content", "POST /v1/videos"}
			if failure == "content" {
				expected = append(expected, "GET /v1/videos/video_workflow",
					"GET /v1/videos/video_workflow/content", "GET /v1/videos/video_workflow/content")
			}
			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("extra requests or repeated steps:\n%s", strings.Join(calls, "\n"))
			}
		})
	}
}

func TestSpeechPreservesExplicitTextVoiceStyleAndSpeed(t *testing.T) {
	calls := 0
	text, style := "产品 A\n“保持原文！”", "清晰、温和"
	result, err, _ := audioCLI(t, "speech", func(request map[string]any) {
		request["input"], request["instruction"], request["speed"] = text, style, 0.75
	}, func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{"model": "stepaudio-2.5-tts", "input": text, "instruction": style,
			"voice": "cixingnansheng", "speed": 0.75, "response_format": "wav"}
		if !reflect.DeepEqual(body, expected) {
			t.Errorf("explicit speech inputs were changed or dropped: %#v", body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}},
			Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
	})
	if err != nil || calls != 1 || result["next_step"] != "deliver" || result["delivery_status"] != "downloaded" {
		t.Fatalf("unexpected speech completion: %+v %v", result, err)
	}
}
