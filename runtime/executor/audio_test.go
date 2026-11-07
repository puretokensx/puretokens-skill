package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func audioWAVFixture() []byte {
	data := make([]byte, 44+320)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], 16000)
	binary.LittleEndian.PutUint32(data[28:], 32000)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], 320)
	return data
}

func audioCLI(t *testing.T, operation string, change func(map[string]any), transport evaluationTransport) (map[string]any, error, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := os.WriteFile(filepath.Join(root, "config.toml"), []byte("model_provider='fixture'\n[model_providers.fixture]\nbase_url='https://api.puretokensx.com/v1'\nexperimental_bearer_token='synthetic-audio-token'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"operation": operation}
	switch operation {
	case "speech":
		request["model"], request["input"], request["voice"] = "stepaudio-2.5-tts", "请原样朗读 private-input-marker", "cixingnansheng"
		request["response_format"], request["output_dir"] = "wav", root
	case "generate":
		request["model"], request["instruction"] = "stepaudio-3-gen-preview", "雨声与脚步声"
		request["response_format"], request["output_dir"] = "wav", root
	case "transcribe":
		file := filepath.Join(root, "private-recording.wav")
		if err := os.WriteFile(file, audioWAVFixture(), 0600); err != nil {
			t.Fatal(err)
		}
		request["model"], request["file"] = "stepaudio-2.5-asr", file
	}
	if change != nil {
		change(request)
	}
	body, _ := json.Marshal(request)
	file := filepath.Join(root, "request.json")
	if err := os.WriteFile(file, body, 0600); err != nil {
		t.Fatal(err)
	}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	var out bytes.Buffer
	err := run([]string{"audio", "--host", "codex", "--request", file}, strings.NewReader(""), &out)
	var receipt map[string]any
	if json.Unmarshal(out.Bytes(), &receipt) != nil {
		t.Fatal("expected one JSON receipt")
	}
	return receipt, err, out.String()
}

func TestAudioSynchronousBinaryAndTranscript(t *testing.T) {
	for _, operation := range []string{"speech", "generate", "transcribe"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			result, err, raw := audioCLI(t, operation, nil, func(r *http.Request) (*http.Response, error) {
				calls++
				path := map[string]string{"speech": "speech", "generate": "generate", "transcribe": "transcriptions"}[operation]
				if r.Method != "POST" || r.URL.String() != apiOrigin+"/v1/audio/"+path ||
					r.Header.Get("Authorization") != "Bearer synthetic-audio-token" {
					t.Error("wrong audio transport")
				}
				if _, ok := r.Context().Deadline(); !ok {
					t.Error("missing deadline")
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
					if string(fields["model"]) != "stepaudio-2.5-asr" || string(fields["response_format"]) != "json" ||
						!bytes.Equal(fields["file"], audioWAVFixture()) || len(fields) != 3 {
						t.Error("transcription changed bytes or added fields")
					}
					response := evaluationHTTP(200, `{"text":"原始转写 private-transcript-marker","debug":"private-response-marker"}`)
					response.Header.Set("Content-Type", "application/json")
					return response, nil
				}
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if body["output_dir"] != nil || body["operation"] != nil || body["async"] != nil {
					t.Error("local/audio task fields leaked")
				}
				if operation == "generate" && (body["task"] != "text_to_audio" || body["stream_format"] != "audio") {
					t.Error("wrong sound protocol")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
			})
			if err != nil || calls != 1 || result["ok"] != true || result["command"] != "audio" || result["submission_outcome"] != "accepted" {
				t.Fatalf("audio failed: %v calls=%d err=%v", result, calls, err)
			}
			if operation == "transcribe" {
				if jsonObject(result["result"])["text"] != "原始转写 private-transcript-marker" || result["next_step"] != "done" {
					t.Fatal("text lost")
				}
			} else {
				artifact := jsonObject(result["artifact"])
				path, _ := artifact["path"].(string)
				data, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(data, audioWAVFixture()) || result["next_step"] != "deliver" || result["delivery_status"] != "downloaded" {
					t.Fatal("audio was not saved for handoff")
				}
			}
			if strings.Contains(raw, "private-input-marker") || strings.Contains(raw, "private-response-marker") || strings.Contains(raw, "synthetic-audio-token") {
				t.Fatal("private response/input leaked")
			}
			support, _ := json.Marshal(result["support"])
			if strings.Contains(string(support), "private-") || strings.Contains(string(support), "artifact") {
				t.Fatal("support leaked user result")
			}
		})
	}
}

func TestAudioInvalidInputStopsBeforeNetwork(t *testing.T) {
	changes := map[string]func(map[string]any){
		"music":                   func(r map[string]any) { r["operation"] = "music" },
		"unknown-model":           func(r map[string]any) { r["model"] = "stepaudio-3-tts" },
		"voice-clone":             func(r map[string]any) { r["voice"] = "cloned-private" },
		"oversize":                func(r map[string]any) { r["input"] = strings.Repeat("字", 1001) },
		"speed":                   func(r map[string]any) { r["speed"] = 0 },
		"stream":                  func(r map[string]any) { r["stream"] = true },
		"relative-output":         func(r map[string]any) { r["output_dir"] = "relative" },
		"file-extra":              func(r map[string]any) { r["file"] = "/unrequested/file.wav" },
		"instruction-unsupported": func(r map[string]any) { r["model"], r["instruction"] = "step-tts-mini", "emotion" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			calls := 0
			result, err, _ := audioCLI(t, "speech", change, func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("should not send") })
			if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" || result["local_error_code"] != "audio_request_invalid" {
				t.Fatalf("bad request reached network: %v", result)
			}
		})
	}
}

func TestAudioFailureNeverRepeats(t *testing.T) {
	for name, transport := range map[string]evaluationTransport{
		"network": func(*http.Request) (*http.Response, error) { return nil, errors.New("private-response-marker") },
		"redirect": func(*http.Request) (*http.Response, error) {
			r := evaluationHTTP(302, "private-response-marker")
			r.Header.Set("Location", "https://other.invalid")
			return r, nil
		},
		"server": func(*http.Request) (*http.Response, error) {
			return evaluationHTTP(500, `{"detail":"private-response-marker"}`), nil
		},
		"fake-audio": func(*http.Request) (*http.Response, error) {
			r := evaluationHTTP(200, "private-response-marker")
			r.Header.Set("Content-Type", "audio/wav")
			return r, nil
		},
		"rejected": func(*http.Request) (*http.Response, error) {
			return evaluationHTTP(422, `{"detail":"private-response-marker"}`), nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			result, err, raw := audioCLI(t, "speech", nil, func(r *http.Request) (*http.Response, error) { calls++; return transport(r) })
			outcome := "unknown"
			if name == "rejected" {
				outcome = "rejected"
			}
			if err == nil || calls != 1 || result["submission_outcome"] != outcome || result["artifact"] != nil || strings.Contains(raw, "private-response-marker") {
				t.Fatalf("unsafe failure: %v calls=%d", result, calls)
			}
		})
	}
}

func audioMP3Fixture() []byte {
	frame := make([]byte, 417)
	copy(frame, []byte{0xff, 0xfb, 0x90, 0x00})
	return append(append([]byte{}, frame...), frame...)
}

func audioOggFixture() []byte {
	head := make([]byte, 19)
	copy(head, "OpusHead")
	head[8] = 1
	head[9] = 1
	tags := append([]byte("OpusTags"), make([]byte, 8)...)
	var result []byte
	for i, packet := range [][]byte{head, tags, {0xf8, 0xff, 0xfe}} {
		page := make([]byte, 28+len(packet))
		copy(page, "OggS")
		page[26] = 1
		page[27] = byte(len(packet))
		if i == 0 {
			page[5] = 2
		}
		if i == 2 {
			page[5] = 4
		}
		binary.LittleEndian.PutUint32(page[14:], 42)
		binary.LittleEndian.PutUint32(page[18:], uint32(i))
		copy(page[28:], packet)
		crc := uint32(0)
		for _, b := range page {
			crc ^= uint32(b) << 24
			for bit := 0; bit < 8; bit++ {
				if crc&0x80000000 != 0 {
					crc = (crc << 1) ^ 0x04c11db7
				} else {
					crc <<= 1
				}
			}
		}
		binary.LittleEndian.PutUint32(page[22:], crc)
		result = append(result, page...)
	}
	return result
}

func TestAudioStructureRequiresCompleteMedia(t *testing.T) {
	for format, data := range map[string][]byte{"wav": audioWAVFixture(), "mp3": audioMP3Fixture(), "ogg": audioOggFixture()} {
		t.Run(format, func(t *testing.T) {
			if !validAudioBytes(data, format) {
				t.Fatal("valid fixture rejected")
			}
			for _, n := range []int{0, 1, 4, len(data) / 2, len(data) - 1} {
				if validAudioBytes(data[:n], format) {
					t.Fatalf("accepted truncation at %d", n)
				}
			}
			changed := append([]byte{}, data...)
			changed[0] ^= 1
			if validAudioBytes(changed, format) || validAudioBytes([]byte("<html>audio</html>"), format) {
				t.Fatal("accepted wrong signature")
			}
		})
	}
	data := audioOggFixture()
	data[len(data)-1] ^= 1
	if validOggAudio(data) {
		t.Fatal("accepted corrupt OGG CRC")
	}
	data = audioWAVFixture()
	binary.LittleEndian.PutUint16(data[32:], 0)
	if validWAV(data) {
		t.Fatal("accepted invalid block size")
	}
}

func TestAudioVerifyUsesOriginalArtifactWithoutNetworkOrCredential(t *testing.T) {
	result, err, _ := audioCLI(t, "speech", nil, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", t.TempDir())
	http.DefaultTransport = evaluationTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("local verify made a network call")
		return nil, nil
	})
	artifact := result["artifact"]
	body, _ := json.Marshal(map[string]any{"artifact": artifact})
	request := filepath.Join(t.TempDir(), "verify.json")
	os.WriteFile(request, body, 0600)
	for _, tampered := range []bool{false, true} {
		if tampered {
			path := jsonObject(artifact)["path"].(string)
			os.WriteFile(path, []byte("changed"), 0600)
		}
		var out bytes.Buffer
		err := run([]string{"audio-verify", "--host", "codex", "--request", request}, strings.NewReader(""), &out)
		var receipt map[string]any
		json.Unmarshal(out.Bytes(), &receipt)
		if (err == nil) == tampered || receipt["command"] != "audio-verify" || receipt["submission_outcome"] != "not_submitted" || jsonObject(receipt["support"])["api_request_attempted"] != false {
			t.Fatalf("unsafe verification: %s", out.String())
		}
	}
}

func TestAudioMalformedRequestAndUnsupportedFlags(t *testing.T) {
	for _, input := range []string{`null`, `[]`, `{"operation":"speech","operation":"generate"}`, `{"model":"stepaudio-2.5-tts","operation":"speech","input":"ok","voice":"cixingnansheng","response_format":"wav","output_dir":"/tmp","speed":null}`} {
		if _, err := decodeAudioRequest([]byte(input)); err == nil {
			t.Fatal("accepted malformed request")
		}
	}
	for _, command := range []string{"audio", "audio-verify"} {
		for _, args := range [][]string{{command, "--host", "codex", "--index", "0"}, {command, "--wrong"}, {command, "--host", "codex", "--request", "relative"}} {
			var out bytes.Buffer
			if run(args, strings.NewReader(""), &out) == nil {
				t.Fatal("accepted malformed CLI")
			}
			var result map[string]any
			json.Unmarshal(out.Bytes(), &result)
			if result["command"] != command || result["submission_outcome"] != "not_submitted" {
				t.Fatal("lost audio command")
			}
		}
	}
	input := `{"operation":"speech","model":"stepaudio-2.5-tts","input":"ok","voice":"cixingnansheng","response_format":"wav","output_dir":` + strconv.Quote(t.TempDir()) + `}`
	if _, err := decodeAudioRequest(append([]byte{0xef, 0xbb, 0xbf}, []byte(input)...)); err != nil {
		t.Fatal("UTF8 BOM not accepted")
	}
}

func TestAudioAttachmentAndOutputPreflight(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"missing":     func(r map[string]any) { r["file"] = filepath.Join(t.TempDir(), "missing.wav") },
		"corrupt":     func(r map[string]any) { os.WriteFile(r["file"].(string), []byte("fake"), 0600) },
		"oversized":   func(r map[string]any) { os.WriteFile(r["file"].(string), make([]byte, maxAudioInput+1), 0600) },
		"unsupported": func(r map[string]any) { r["response_format"] = "srt" },
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			result, err, _ := audioCLI(t, "transcribe", change, func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") })
			if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" {
				t.Fatal("bad recording sent")
			}
		})
	}
	result, err, _ := audioCLI(t, "speech", func(r map[string]any) { r["output_dir"] = filepath.Join(t.TempDir(), "missing") }, func(*http.Request) (*http.Response, error) { t.Fatal("sent without output directory"); return nil, nil })
	if err == nil || result["submission_outcome"] != "not_submitted" {
		t.Fatal("missing output not rejected")
	}
}

func TestAudioTranscriptRejectsPartialAndAmbiguousJSON(t *testing.T) {
	for _, body := range []string{`{"text":"first","text":"second"}`, `{"text":null}`, `data: {"text":"partial"}`, `{"text":"ok","error":{}}`, strings.Repeat("x", 128<<10+1)} {
		calls := 0
		result, err, raw := audioCLI(t, "transcribe", nil, func(*http.Request) (*http.Response, error) {
			calls++
			r := evaluationHTTP(200, body)
			r.Header.Set("Content-Type", "application/json")
			return r, nil
		})
		if err == nil || calls != 1 || result["submission_outcome"] != "unknown" || result["result"] != nil || strings.Contains(raw, "partial") {
			t.Fatal("unsafe transcript failure")
		}
	}
}

func TestAudioCatalogIntersectsReviewedModels(t *testing.T) {
	calls := 0
	svc := service{baseURL: apiOrigin, client: &http.Client{Transport: evaluationTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.String() != apiOrigin+"/v1/models" {
			t.Fatal("wrong discovery route")
		}
		return evaluationHTTP(200, `{"data":[{"id":"stepaudio-2.5-tts","provider":"private"},{"id":"stepaudio-2.5-tts"},{"id":"stepaudio-3-gen-preview"},{"id":"stepaudio-3-tts"},{"id":"invented"}]}`), nil
	})}}
	var out bytes.Buffer
	if executeModelQuery(&out, svc, strings.NewReader(`{"kind":"audio","operation":"speech"}`)) != nil {
		t.Fatal("catalog failed")
	}
	var receipt map[string]any
	json.Unmarshal(out.Bytes(), &receipt)
	result := jsonObject(receipt["result"])
	if calls != 1 || result["matched_count"] != float64(1) || result["matching_scope"] != "reviewed_audio_models_listed_by_api" || strings.Contains(out.String(), "private") {
		t.Fatalf("unreviewed discovery: %s", out.String())
	}
	for _, bad := range []string{`{"kind":"audio","parameters":{"speed":1}}`, `{"kind":"audio","operation":"clone"}`} {
		if executeModelQuery(io.Discard, svc, strings.NewReader(bad)) == nil || calls != 1 {
			t.Fatal("invalid filter reached network")
		}
	}
}

type audioBrokenReader struct{}

func (audioBrokenReader) Read([]byte) (int, error) { return 0, errors.New("private-stream-error") }
func (audioBrokenReader) Close() error             { return nil }

func TestAudioResponseBoundsAndFileConflict(t *testing.T) {
	for name, body := range map[string]io.ReadCloser{
		"oversized": io.NopCloser(bytes.NewReader(make([]byte, maxAudioOutput+1))),
		"broken":    audioBrokenReader{},
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			result, err, raw := audioCLI(t, "speech", nil, func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: body}, nil
			})
			if err == nil || calls != 1 || result["submission_outcome"] != "unknown" || result["artifact"] != nil || strings.Contains(raw, "private-stream-error") {
				t.Fatal("unsafe read failure")
			}
		})
	}
	t.Run("wrong-mime", func(t *testing.T) {
		result, err, _ := audioCLI(t, "speech", nil, func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/octet-stream"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
		})
		if err == nil || result["submission_outcome"] != "unknown" {
			t.Fatal("accepted mismatched MIME")
		}
	})
	t.Run("output-conflict", func(t *testing.T) {
		var dir, destination string
		calls := 0
		result, err, _ := audioCLI(t, "speech", func(r map[string]any) { dir = r["output_dir"].(string) }, func(*http.Request) (*http.Response, error) {
			calls++
			parts, _ := filepath.Glob(filepath.Join(dir, ".puretokens-audio-part-*"))
			if len(parts) != 1 {
				t.Fatal("output not reserved before POST")
			}
			destination = filepath.Join(dir, "puretokens-audio-"+strings.TrimPrefix(filepath.Base(parts[0]), ".puretokens-audio-part-")+".wav")
			if os.WriteFile(destination, []byte("user-existing-file"), 0600) != nil {
				t.Fatal("fixture create failed")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
		})
		data, _ := os.ReadFile(destination)
		if err == nil || calls != 1 || result["submission_outcome"] != "accepted" || result["failure_phase"] != "content" || string(data) != "user-existing-file" {
			t.Fatal("unsafe overwrite or output failure")
		}
		parts, _ := filepath.Glob(filepath.Join(dir, ".puretokens-audio-part-*"))
		if len(parts) != 0 {
			t.Fatal("temporary audio bytes left behind")
		}
	})
}

func TestAudioAttachmentSymlinkRejected(t *testing.T) {
	calls := 0
	result, err, _ := audioCLI(t, "transcribe", func(r map[string]any) {
		file := r["file"].(string)
		link := filepath.Join(filepath.Dir(file), "link.wav")
		if err := os.Symlink(file, link); err != nil {
			t.Skip("symlinks unavailable")
		}
		r["file"] = link
	}, func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") })
	if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" {
		t.Fatal("symlink read as audio")
	}
}
