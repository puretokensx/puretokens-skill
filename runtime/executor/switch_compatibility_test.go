package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// This fixture is also asserted against the real Switch VSCodeAdapter writer.
// It is synthetic configuration, never a copied user record.
func switchVSCodeFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../test/fixtures/switch-vscode-connection.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSwitchVSCodeConnectionBoundary(t *testing.T) {
	for _, variant := range []string{
		"writer-output", "labels-irrelevant", "different-key", "second-group",
		"extra-header", "group-header", "wrong-api-type", "foreign-model",
		"dynamic-key", "missing-key", "malformed-model", "credential-override", "discovery-override",
		"other-group-unsupported-route", "other-group-discovery",
	} {
		t.Run(variant, func(t *testing.T) {
			var groups []map[string]any
			if json.Unmarshal(switchVSCodeFixture(t), &groups) != nil {
				t.Fatal("invalid synthetic fixture")
			}
			models := groups[0]["models"].([]any)
			second := models[1].(map[string]any)
			switch variant {
			case "labels-irrelevant":
				groups[0]["name"], groups[0]["_puretokensSwitch"] = "not-an-identity", "not-an-identity"
			case "different-key":
				second["requestHeaders"] = map[string]any{"Authorization": "Bearer other-synthetic-key"}
			case "second-group":
				groups = append(groups, groups[0])
			case "extra-header":
				second["requestHeaders"].(map[string]any)["X-Api-Key"] = "other-synthetic-key"
			case "group-header":
				groups[0]["requestHeaders"] = map[string]any{}
			case "wrong-api-type":
				second["apiType"] = "messages"
			case "foreign-model":
				second["url"] = "https://example.invalid/v1/responses"
			case "dynamic-key":
				second["requestHeaders"] = map[string]any{"Authorization": "Bearer ${env:SECRET}"}
			case "missing-key":
				second["requestHeaders"] = map[string]any{"Authorization": "Bearer "}
			case "malformed-model":
				models[1] = nil
			case "credential-override":
				second["apiKey"] = "other-synthetic-key"
			case "discovery-override":
				groups[0]["modelsUrl"] = "https://api.puretokensx.com/v1/models"
			case "other-group-unsupported-route":
				groups = append(groups, map[string]any{"vendor": "customendpoint", "models": []any{
					map[string]any{"url": "https://api.puretokensx.com/v1/unsupported"},
				}})
			case "other-group-discovery":
				groups = append(groups, map[string]any{"vendor": "customendpoint", "modelsUrl": "https://api.puretokensx.com/v1/models"})
			}
			data, _ := json.Marshal(groups)
			file := clientFixtureFile(t, clientFixtureRoot(t), "chatLanguageModels.json", string(data))
			key, err := credentialFromVSCodeFile(file)
			wantOK := variant == "writer-output" || variant == "labels-irrelevant"
			if (err == nil) != wantOK || (!wantOK && key != "") || (wantOK && key != "pts-file-fake-not-real") {
				t.Fatal("connection boundary outcome differs")
			}
			after, _ := os.ReadFile(file)
			if !bytes.Equal(data, after) {
				t.Fatal("adapter wrote configuration")
			}
			if err != nil {
				code, message, next := credentialFailureDetails(err)
				if strings.Contains(code+message+next, "fake-not-real") ||
					strings.Contains(code+message+next, "other-synthetic-key") {
					t.Fatal("diagnostic disclosed comparison inputs")
				}
				if (variant == "different-key" || variant == "second-group") && code != "active_connection_ambiguous" {
					t.Fatal("ambiguous connection was not identified")
				}
			}
		})
	}
}

func TestSwitchVSCodePortableOverrideStopsBeforeNetwork(t *testing.T) {
	t.Setenv("VSCODE_PORTABLE", filepath.Join(t.TempDir(), "portable"))
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	calls := 0
	http.DefaultTransport = evaluationTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return evaluationHTTP(500, `{}`), nil
	})
	var out bytes.Buffer
	err := run([]string{"models", "--host", "vscode"}, strings.NewReader(""), &out)
	if err == nil || calls != 0 || !strings.Contains(out.String(), "active_connection_format_unsupported") {
		t.Fatal("unsupported profile override did not stop before network access")
	}
}

// Exercises the actual command dispatcher, adapter and fixed-request builder.
// HTTP and handoff are simulated: these are not host or gateway acceptance.
func TestSwitchVSCodeDedicatedPermissionsAndFreshCatalog(t *testing.T) {
	home := clientFixtureRoot(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "roaming"))
	t.Setenv("VSCODE_PORTABLE", "")
	// Path selection is independently covered on both supported OSes.
	// This dispatcher test only runs where VS Code's adapter is implemented.
	root, err := newClientRoot("vscode", runtime.GOOS, home, os.Getenv)
	if err != nil {
		t.Skip("VS Code default-profile adapter has no Linux path")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	clientFixtureFile(t, root, "chatLanguageModels.json", string(switchVSCodeFixture(t)))
	exerciseDedicatedHostPermissions(t, "vscode", home, "pts-file-fake-not-real")
}

func TestDSHDesktopDedicatedPermissionsAndFreshCatalog(t *testing.T) {
	for _, layout := range []string{"versioned", "flat"} {
		t.Run(layout, func(t *testing.T) {
			root := dshDesktopFixture(t)
			t.Setenv("DSH_HOME", root)
			t.Setenv("CHOSEN_REF", "synthetic-wrong-environment-key")
			if layout == "flat" {
				desktopFixtureFile(t, root, ".credentials.yaml", "CHOSEN_REF: synthetic-dsh-key\nOTHER_REF: synthetic-other-key\n")
			}
			before, err := os.ReadFile(filepath.Join(root, ".credentials.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			exerciseDedicatedHostPermissions(t, "dsh-desktop", root, "synthetic-dsh-key")
			after, err := os.ReadFile(filepath.Join(root, ".credentials.yaml"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("dedicated calls changed the host credential store")
			}
		})
	}
}

func TestClaudeMappedDedicatedPermissionsAndFreshCatalog(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("local Desktop paths only")
	}
	home := clientFixtureRoot(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "roaming"))
	root, err := currentDesktopRoot("Claude-3p")
	if err != nil {
		t.Fatal(err)
	}
	source, _ := mappedSkillsFixture(t)
	for _, name := range []string{"claude_desktop_config.json", "configLibrary/_meta.json", "configLibrary/" + fixtureDesktopProfile + ".json", claudeSkillsConnectionFile} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		desktopFixtureFile(t, root, name, string(data))
	}
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	calls := 0
	http.DefaultTransport = evaluationTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return evaluationHTTP(500, `{}`), nil
	})
	for _, command := range []string{"submit", "status", "wait", "content"} {
		input := `{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"synthetic","requested_count":1}`
		if command != "submit" {
			input = `{"kind":"image","operation":"continue","task_id":"image_fixture"}`
		}
		var out bytes.Buffer
		if err := run([]string{command, "--host", "claude-desktop"}, strings.NewReader(input), &out); err == nil || calls != 0 || !strings.Contains(out.String(), "explicit task record") {
			t.Fatalf("%s without a mapped task record did not stop before networking", command)
		}
	}
	exerciseDedicatedHostPermissions(t, "claude-desktop", home, "synthetic-mapped-skills-key")
}

func TestTraeDedicatedPermissionsAndFreshCatalog(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("TRAE local desktop adapter supports macOS and Windows")
	}
	for _, edition := range []string{"Trae", "Trae CN", "TRAE SOLO"} {
		t.Run(edition, func(t *testing.T) {
			home, base := traeFixtureHome(t)
			file := filepath.Join(base, edition, "User", "globalStorage", "state.vscdb")
			traeFixtureDatabase(t, file, nil)
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			exerciseDedicatedHostPermissions(t, "trae", home, "fixture-trae-key")
			after, err := os.ReadFile(file)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("dedicated calls changed the host database")
			}
		})
	}
}

func exerciseDedicatedHostPermissions(t *testing.T, host, home, token string) {
	t.Helper()
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	for _, spec := range []struct{ name, command, input, route, response string }{
		{"image", "submit", `{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"synthetic","requested_count":1}`, "/v1/images/generations", `{"id":"image_fixture","status":"pending"}`},
		{"video", "submit", `{"kind":"video","operation":"generate","model":"grok-imagine-video-1.5","prompt":"synthetic","requested_count":1}`, "/v1/videos", `{"id":"video_fixture","status":"pending"}`},
		{"audio", "audio", `{"operation":"speech","model":"stepaudio-2.5-tts","input":"synthetic","voice":"cixingnansheng","response_format":"wav"}`, "/v1/audio/speech", ""},
		{"jev", "evaluate", evaluationInputFixture, "/typesafe/v1/systemone", evaluationOutputFixture},
		{"embedding", "embedding", `{"model":"qwen3.7-text-embedding","input":["synthetic one","synthetic two"]}`, "/v1/embeddings", embeddingResponseFixture},
		{"rerank", "rerank", `{"model":"qwen3.7-text-rerank","query":"synthetic query","documents":["synthetic one","synthetic two"]}`, "/v1/rerank", rerankResponseFixture},
	} {
		t.Run(spec.name, func(t *testing.T) {
			var expected map[string]any
			json.Unmarshal([]byte(spec.input), &expected)
			exactModel := expected["model"].(string)
			// A fresh catalog can change without modifying chat-picker entries.
			for _, visible := range []bool{true, false, true} {
				calls := 0
				http.DefaultTransport = evaluationTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer "+token {
						t.Error("wrong discovery request")
					}
					wantPath := "/v1/media/models"
					if spec.name != "image" && spec.name != "video" {
						wantPath = "/v1/models"
					}
					if r.URL.String() != apiOrigin+wantPath {
						t.Error("wrong discovery endpoint")
					}
					models := []any{}
					if visible {
						kind := spec.name
						if kind == "jev" {
							kind = "evaluation"
						}
						models = append(models, map[string]any{"id": exactModel, "capabilities": []string{kind}})
					}
					payload, _ := json.Marshal(map[string]any{"data": models})
					return evaluationHTTP(200, string(payload)), nil
				})
				kind := spec.name
				if kind == "jev" {
					kind = "evaluation"
				}
				filter, _ := json.Marshal(map[string]any{"kind": kind, "model": exactModel})
				file := clientFixtureFile(t, home, "query.json", string(filter))
				var output bytes.Buffer
				err := run([]string{"models", "--host", host, "--request", file}, strings.NewReader(""), &output)
				var result map[string]any
				json.Unmarshal(output.Bytes(), &result)
				matched := jsonObject(result["result"])["matched_count"]
				want := float64(0)
				if visible {
					want = 1
				}
				if err != nil || calls != 1 || matched != want {
					t.Fatalf("catalog refresh did not reflect the latest synthetic authorization: calls=%d", calls)
				}
			}
			// Visibility never substitutes for POST admission. In particular,
			// 401/403 must not trigger another key, model, lookup or submission.
			for _, status := range []int{200, 401, 403} {
				calls := []string{}
				http.DefaultTransport = evaluationTransport(func(r *http.Request) (*http.Response, error) {
					calls = append(calls, r.Method+" "+r.URL.Path)
					if r.URL.String() != apiOrigin+spec.route || r.Method != "POST" ||
						r.Header.Get("Authorization") != "Bearer "+token {
						t.Error("dedicated request used a chat route or different connection")
						return evaluationHTTP(500, `{}`), nil
					}
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					if body["model"] != exactModel {
						t.Error("exact dedicated model identity was lost")
					}
					if status != 200 {
						return evaluationHTTP(status, `{"error":{"message":"private-upstream pts-file-fake-not-real","code":"model_not_allowed"}}`), nil
					}
					if spec.name == "audio" {
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
					}
					return evaluationHTTP(200, spec.response), nil
				})
				request := map[string]any{}
				json.Unmarshal([]byte(spec.input), &request)
				if spec.name == "audio" {
					request["output_dir"] = t.TempDir()
				} else if spec.name == "embedding" || spec.name == "rerank" {
					extension := ".json"
					if spec.name == "embedding" {
						extension = ".jsonl"
					}
					request["output_file"] = filepath.Join(t.TempDir(), "result"+extension)
				}
				body, _ := json.Marshal(request)
				file := clientFixtureFile(t, home, "request.json", string(body))
				var out bytes.Buffer
				args := []string{spec.command, "--host", host, "--request", file}
				if host == "claude-desktop" && spec.command == "submit" {
					args = append(args, "--record", filepath.Join(t.TempDir(), "task.json"))
				}
				err := run(args, strings.NewReader(""), &out)
				if (err == nil) != (status == 200) || !reflect.DeepEqual(calls, []string{"POST " + spec.route}) {
					t.Fatalf("dedicated authorization or submit-once boundary failed: %s status=%d calls=%v", spec.name, status, calls)
				}
				if strings.Contains(out.String(), token) || strings.Contains(out.String(), "fake-not-real") || strings.Contains(out.String(), "private-upstream") {
					t.Fatal("receipt leaked synthetic private data")
				}
				var receipt map[string]any
				if json.Unmarshal(out.Bytes(), &receipt) != nil {
					t.Fatal("missing structured receipt")
				}
				if status != 200 {
					if receipt["artifact"] != nil || receipt["result"] != nil {
						t.Fatal("rejected request offered output")
					}
					continue
				}
				switch spec.name {
				case "audio", "embedding", "rerank":
					artifact := jsonObject(receipt["artifact"])
					data, err := os.ReadFile(jsonString(artifact["path"]))
					if err != nil || len(data) == 0 {
						t.Fatal("successful dedicated command did not save output bytes")
					}
					switch spec.name {
					case "audio":
						if !bytes.Equal(data, audioWAVFixture()) || receipt["next_step"] != "deliver" {
							t.Fatal("audio bytes or host handoff instruction changed")
						}
					case "embedding":
						if string(data) != "{\"index\":0,\"dimensions\":2,\"embedding\":[1,0]}\n{\"index\":1,\"dimensions\":2,\"embedding\":[0.5,-0.75]}\n" {
							t.Fatal("embedding artifact lost input ordering or values")
						}
					case "rerank":
						var ranking map[string]any
						if json.Unmarshal(data, &ranking) != nil || !reflect.DeepEqual(ranking, receipt["result"]) {
							t.Fatal("ranking artifact differs from typed result")
						}
					}
				case "jev":
					result := jsonObject(receipt["result"])
					answers := jsonObject(result["answers"])
					if result["model"] != "jev-1.13.0" || receipt["next_step"] != "done" ||
						jsonObject(answers["department"])["choice"] != "billing" ||
						jsonObject(answers["severity"])["score"] != 1.1 || jsonObject(answers["urgency"])["noul"] != 0.9 {
						t.Fatal("evaluation omitted its typed result")
					}
				}
			}
		})
	}
}
