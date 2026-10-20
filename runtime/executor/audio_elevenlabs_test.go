package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func elevenAudioRequest(t *testing.T, r map[string]any, model, operation string) {
	t.Helper()
	root := r["output_dir"].(string)
	for key := range r {
		delete(r, key)
	}
	r["model"], r["operation"] = model, operation
	if operation != "transcribe" {
		r["output_dir"], r["response_format"] = root, "mp3"
	}
	switch operation {
	case "speech":
		r["input"], r["voice"] = "原文\n[happy] Hello", "JBFqnCBsd6RMkjVDRZzb"
	case "generate":
		r["instruction"], r["duration_seconds"] = "雨声", 0.5
	case "music_sync":
		r["instruction"], r["duration_seconds"], r["instrumental"] = "原样歌词", 3.125, false
	case "transcribe", "voice_change":
		r["file"] = filepath.Join(root, "source.wav")
		if err := os.WriteFile(r["file"].(string), audioWAVFixture(), 0600); err != nil {
			t.Fatal(err)
		}
		if operation == "voice_change" {
			r["voice"] = "JBFqnCBsd6RMkjVDRZzb"
		}
	}
}

func TestElevenLabsExactAudioWireAndArtifact(t *testing.T) {
	models := map[string]string{
		"eleven_v3": "speech", "eleven_v4": "speech", "eleven_multilingual_v2": "speech", "eleven_flash_v2_5": "speech",
		"scribe_v2": "transcribe", "scribe_v2_medical": "transcribe",
		"eleven_text_to_sound_v2": "generate", "music_v2_5": "music_sync", "eleven_multilingual_sts_v2": "voice_change",
	}
	for model, operation := range models {
		t.Run(model, func(t *testing.T) {
			calls := 0
			receipt, err, _ := audioCLI(t, "speech", func(r map[string]any) {
				elevenAudioRequest(t, r, model, operation)
			}, func(r *http.Request) (*http.Response, error) {
				calls++
				path := "/v1/audio/generate"
				if operation == "speech" {
					path = "/v1/audio/speech"
				}
				if operation == "transcribe" {
					path = "/v1/audio/transcriptions"
				}
				if r.Method != "POST" || r.URL.String() != apiOrigin+path {
					t.Fatal("wrong dedicated API")
				}
				if operation == "transcribe" {
					reader, err := r.MultipartReader()
					if err != nil {
						t.Fatal(err)
					}
					fields := map[string][]byte{}
					for {
						part, err := reader.NextPart()
						if err == io.EOF {
							break
						}
						if err != nil {
							t.Fatal(err)
						}
						fields[part.FormName()], _ = io.ReadAll(part)
					}
					if len(fields) != 3 || string(fields["model"]) != model || !bytes.Equal(fields["file"], audioWAVFixture()) {
						t.Fatal("transcription changed original bytes")
					}
					response := evaluationHTTP(200, `{"text":"转写结果"}`)
					response.Header.Set("Content-Type", "application/json")
					return response, nil
				}
				var body map[string]any
				if json.NewDecoder(r.Body).Decode(&body) != nil || body["model"] != model || body["response_format"] != "mp3" ||
					body["operation"] != nil || body["output_dir"] != nil || body["file"] != nil || body["speed"] != nil || body["lyrics"] != nil {
					t.Fatal("invalid audio body")
				}
				switch operation {
				case "speech":
					if len(body) != 4 || body["input"] != "原文\n[happy] Hello" {
						t.Fatal("speech changed input or inherited fields")
					}
				case "generate":
					if len(body) != 6 || body["task"] != "text_to_audio" || body["duration_seconds"] != 0.5 {
						t.Fatal("wrong sound contract")
					}
				case "music_sync":
					if len(body) != 7 || body["task"] != "text_to_music" || body["duration_seconds"] != 3.125 || body["instrumental"] != false {
						t.Fatal("sync music lost explicit false/duration")
					}
				case "voice_change":
					data, err := base64.StdEncoding.DecodeString(jsonString(body["audio"]))
					if err != nil || len(body) != 7 || body["task"] != "voice_change" || body["audio_format"] != "wav" || !bytes.Equal(data, audioWAVFixture()) {
						t.Fatal("voice change did not preserve original bytes")
					}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/mpeg"}}, Body: io.NopCloser(bytes.NewReader(audioMP3Fixture()))}, nil
			})
			if err != nil || calls != 1 || receipt["ok"] != true || receipt["operation"] != operation {
				t.Fatalf("dedicated audio failed: %v calls=%d err=%v", receipt, calls, err)
			}
			if operation != "transcribe" {
				saved, err := os.ReadFile(jsonString(jsonObject(receipt["artifact"])["path"]))
				if err != nil || !bytes.Equal(saved, audioMP3Fixture()) || receipt["next_step"] != "deliver" {
					t.Fatal("original MP3 not saved for host handoff")
				}
			}
		})
	}
}

func TestElevenLabsRejectsInheritedOrAmbiguousFieldsBeforePOST(t *testing.T) {
	cases := []struct {
		model, operation string
		change           func(map[string]any)
	}{
		{"eleven_v3", "speech", func(r map[string]any) { r["speed"] = 1 }},
		{"eleven_v4", "speech", func(r map[string]any) { r["instruction"] = "风格" }},
		{"eleven_v3", "speech", func(r map[string]any) { r["response_format"] = "wav" }},
		{"eleven_v3", "speech", func(r map[string]any) { r["voice"] = "" }},
		{"eleven_text_to_sound_v2", "generate", func(r map[string]any) { delete(r, "duration_seconds") }},
		{"eleven_text_to_sound_v2", "generate", func(r map[string]any) { r["duration_seconds"] = 30.1 }},
		{"music_v2_5", "music_sync", func(r map[string]any) { delete(r, "instrumental") }},
		{"music_v2_5", "music_sync", func(r map[string]any) { r["duration_seconds"] = 3.0001 }},
		{"music_v2_5", "music_sync", func(r map[string]any) { r["lyrics"] = "另填" }},
		{"music_v2_5", "music_sync", func(r map[string]any) { r["operation"] = "music" }},
		{"eleven_multilingual_sts_v2", "voice_change", func(r map[string]any) { r["instruction"] = "extra" }},
		{"eleven_multilingual_sts_v2", "voice_change", func(r map[string]any) { r["audio"] = "caller-supplied" }},
	}
	for _, tc := range cases {
		calls := 0
		receipt, err, _ := audioCLI(t, "speech", func(r map[string]any) {
			elevenAudioRequest(t, r, tc.model, tc.operation)
			tc.change(r)
		}, func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected POST") })
		if err == nil || calls != 0 || receipt["submission_outcome"] != "not_submitted" {
			t.Fatalf("invalid audio reached POST: %v", receipt)
		}
		if tc.model == "eleven_v3" && tc.operation == "speech" {
			request := map[string]any{}
			tc.change(request)
			if request["speed"] != nil {
				validation := jsonObject(receipt["validation"])
				if validation["field"] != "speed" || validation["reason"] != "unsupported" ||
					strings.Contains(jsonString(receipt["message"]), "0.5 and 2") {
					t.Fatal("unsupported speed inherited StepFun guidance")
				}
			}
		}
	}
}

func TestElevenLabsMusicMillisecondPrecision(t *testing.T) {
	for _, duration := range []float64{3, 3.001, 4.001, 10.003, 599.999, 600} {
		request := map[string]any{
			"operation": "music_sync", "model": "music_v2_5", "instruction": "原样歌词",
			"duration_seconds": duration, "instrumental": false, "response_format": "mp3",
			"output_dir": t.TempDir(),
		}
		body, _ := json.Marshal(request)
		decoded, err := decodeAudioRequest(body)
		if err != nil || decoded.DurationSeconds == nil || *decoded.DurationSeconds != duration {
			t.Fatalf("valid millisecond duration %v was rejected: %v", duration, err)
		}
		request["duration_seconds"] = duration + 0.0001
		body, _ = json.Marshal(request)
		if _, err := decodeAudioRequest(body); err == nil {
			t.Fatalf("sub-millisecond duration %v was accepted", request["duration_seconds"])
		}
	}
}
