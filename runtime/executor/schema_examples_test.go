package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

type schemaExample struct {
	Name     string          `json:"name"`
	Schema   string          `json:"schema"`
	Document json.RawMessage `json:"document"`
}

// Node consumes actual JSON emitted by the executor, not hand-maintained
// replicas of its structs. Every request uses local fixtures and local mocks.
// The test also runs its behavioral checks when invoked directly by go test.
func TestContractExamples(t *testing.T) {
	var examples []schemaExample
	capture := func(name, schema string, data []byte) {
		t.Helper()
		if !json.Valid(data) {
			t.Fatalf("%s did not emit a JSON document", name)
		}
		examples = append(examples, schemaExample{name, schema, append(json.RawMessage(nil), data...)})
		var envelope map[string]json.RawMessage
		if json.Unmarshal(data, &envelope) == nil && envelope["support"] != nil {
			examples = append(examples, schemaExample{"support-" + name, "support-summary.schema.json", envelope["support"]})
		}
		if envelope["timings_ms"] != nil {
			examples = append(examples, schemaExample{"timings-" + name, "timings.schema.json", envelope["timings_ms"]})
		}
	}
	captureValue := func(name, schema string, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		capture(name, schema, data)
	}

	t.Run("synchronous embeddings and rerank", func(t *testing.T) {
		for _, kind := range []string{"embedding", "rerank"} {
			for _, scenario := range []string{"success", "invalid-request", "invalid-response", "rejected", "network", "output-race"} {
				t.Run(kind+"/"+scenario, func(t *testing.T) {
					var destination string
					result, err, raw := retrievalCLI(t, kind, func(r map[string]any) {
						if scenario == "invalid-request" {
							r["model"] = "not-reviewed"
						}
						destination = r["output_file"].(string)
						if scenario == "success" {
							captureValue(kind+"-request", kind+"-request.schema.json", r)
						}
					}, func(*http.Request) (*http.Response, error) {
						switch scenario {
						case "invalid-response":
							return evaluationHTTP(200, `{}`), nil
						case "rejected":
							return evaluationHTTP(403, `{"error":{"message":"private"}}`), nil
						case "network":
							return nil, errors.New("fixture network")
						case "output-race":
							os.WriteFile(destination, []byte("user-file"), 0600)
						}
						response := embeddingResponseFixture
						if kind == "rerank" {
							response = rerankResponseFixture
						}
						return evaluationHTTP(200, response), nil
					})
					if (err == nil) != (scenario == "success") {
						t.Fatal("unexpected retrieval fixture result", err)
					}
					capture(kind+"-"+scenario, "retrieval-receipt.schema.json", []byte(raw))
					if scenario == "success" {
						artifact, err := os.ReadFile(jsonObject(result["artifact"])["path"].(string))
						if err != nil {
							t.Fatal(err)
						}
						if kind == "embedding" {
							for index, row := range bytes.Split(bytes.TrimSpace(artifact), []byte("\n")) {
								capture(fmt.Sprintf("embedding-output-%d", index), "embedding-output.schema.json", row)
							}
						} else {
							capture("rerank-output", "rerank-output.schema.json", artifact)
						}
					}
				})
			}
			var out bytes.Buffer
			svc := service{baseURL: apiOrigin, client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
				return evaluationHTTP(200, `{"data":[{"id":"qwen3.7-text-embedding"},{"id":"qwen3.7-text-rerank"}]}`), nil
			})}}
			query := `{"kind":"` + kind + `"}`
			if executeModelQuery(&out, svc, strings.NewReader(query)) != nil {
				t.Fatal("retrieval discovery failed")
			}
			capture(kind+"-model-query", "model-query.schema.json", []byte(query))
			capture(kind+"-model-receipt", "model-query-receipt.schema.json", out.Bytes())
			text, err := os.ReadFile("../../skills/puretokens-" + kind + "/references/executor-usage.md")
			if err != nil {
				t.Fatal(err)
			}
			blocks := regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```").FindAllSubmatch(text, -1)
			if len(blocks) == 0 {
				t.Fatal("missing documented retrieval request")
			}
			for index, block := range blocks {
				if index > 0 {
					capture(fmt.Sprintf("docs-%s-output-%d", kind, index), kind+"-output.schema.json", block[1])
					continue
				}
				capture("docs-"+kind+"-request", kind+"-request.schema.json", block[1])
				var input map[string]any
				if json.Unmarshal(block[1], &input) != nil {
					t.Fatal("invalid documented request")
				}
				_, err, _ := retrievalCLI(t, kind, func(r map[string]any) {
					destination := r["output_file"]
					for key := range r {
						delete(r, key)
					}
					for key, value := range input {
						r[key] = value
					}
					r["output_file"] = destination
				}, func(*http.Request) (*http.Response, error) {
					response := embeddingResponseFixture
					if kind == "rerank" {
						response = rerankResponseFixture
					}
					return evaluationHTTP(200, response), nil
				})
				if err != nil {
					t.Fatal("documented retrieval request rejected", err)
				}
			}
		}
	})

	t.Run("public model groups", func(t *testing.T) {
		request := `{"model":"nano-banana-2"}`
		capture("model-groups-request", "model-groups-request.schema.json", []byte(request))
		for _, scenario := range []string{"available", "unavailable", "http-failure", "invalid"} {
			t.Run(scenario, func(t *testing.T) {
				input := request
				if scenario == "invalid" {
					input = `{}`
				}
				_, err, raw := modelGroupsCLI(t, input, func(r *http.Request) (*http.Response, error) {
					checkAnonymousModelGroupsRequest(t, r)
					switch scenario {
					case "unavailable":
						return evaluationHTTP(200, `{"data":[{"model_name":"nano-banana-2","enable_groups":["not-a-display-name"]}]}`), nil
					case "http-failure":
						return evaluationHTTP(401, `{"message":"private"}`), nil
					case "invalid":
						t.Fatal("invalid group request made a network call")
					}
					return evaluationHTTP(200, modelGroupsFixture), nil
				})
				if (err == nil) != (scenario == "available" || scenario == "unavailable") {
					t.Fatal("unexpected public group fixture result")
				}
				capture("model-groups-"+scenario, "model-groups-receipt.schema.json", []byte(raw))
			})
		}
	})

	t.Run("safe audio construction diagnostics", func(t *testing.T) {
		for _, name := range []string{"unknown-voice", "unicode-limit"} {
			request := map[string]any{"operation": "speech", "model": "stepaudio-2.5-tts", "input": "PRIVATE-INPUT", "voice": "cixingnansheng", "response_format": "mp3", "output_dir": t.TempDir()}
			if name == "unknown-voice" {
				request["voice"] = "PRIVATE-UNKNOWN-VOICE"
			} else {
				request["input"] = strings.Repeat("🎧", 1001)
			}
			data, _ := json.Marshal(request)
			file := writeFixture(t, "request.json", string(data))
			var output bytes.Buffer
			if err := run([]string{"audio", "--host", "fixture-unsupported", "--request", file}, strings.NewReader(""), &output); err == nil {
				t.Fatal("invalid audio input accepted")
			}
			if strings.Contains(output.String(), "PRIVATE-") || strings.Contains(output.String(), "🎧") {
				t.Fatal("audio diagnostic retained user-supplied content")
			}
			capture("audio-diagnostic-"+name, "audio-receipt.schema.json", output.Bytes())
		}
	})

	t.Run("asynchronous music", func(t *testing.T) {
		text, err := os.ReadFile("../../skills/puretokens-audio/references/music-usage.md")
		if err != nil {
			t.Fatal(err)
		}
		blocks := regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```").FindAllSubmatch(text, -1)
		if len(blocks) != 2 {
			t.Fatal("music examples missing")
		}
		for index, block := range blocks {
			t.Run(fmt.Sprintf("music-doc-%d", index), func(t *testing.T) {
				capture(fmt.Sprintf("docs-music-%d", index), "executor-request.schema.json", block[1])
				request, err := decodeTaskRequest(bytes.NewReader(block[1]))
				if err != nil {
					t.Fatal(err)
				}
				request.OutputDir = t.TempDir()
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method == "POST" {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, `{"id":"music_schema","status":"pending"}`)
					} else if strings.HasSuffix(r.URL.Path, "/content") {
						format := request.Parameters["response_format"].(string)
						w.Header().Set("Content-Type", audioMIME(format))
						data := audioWAVFixture()
						if format == "mp3" {
							data = audioMP3Fixture()
						}
						w.Write(data)
					} else {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, `{"id":"music_schema","status":"succeeded"}`)
					}
				}))
				defer server.Close()
				record := filepath.Join(request.OutputDir, "music-task.json")
				for n, command := range []string{"submit", "status", "content", "resume", "delivered"} {
					var out bytes.Buffer
					writer := &supportReceiptWriter{output: &out, host: "codex", command: command}
					svc := fixtureService(server)
					svc.support = writer
					if err := executeRecordedTask(command, record, request, 0, request.OutputDir, writer, svc); err != nil {
						t.Fatal(err, out.String())
					}
					capture(fmt.Sprintf("music-%d-%s", index, command), "executor-receipt.schema.json", out.Bytes())
					saved, err := os.ReadFile(record)
					if err != nil {
						t.Fatal(err)
					}
					capture(fmt.Sprintf("music-%d-%s-record", index, command), "task-record.schema.json", saved)
					if n == 0 && calls != 1 {
						t.Fatal("music submit must immediately return receipt")
					}
				}
				if calls != 3 {
					t.Fatal("music recovery repeated network work")
				}
			})
		}
		var out bytes.Buffer
		query := `{"kind":"audio","operation":"music"}`
		svc := service{baseURL: apiOrigin, client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
			return evaluationHTTP(200, `{"data":[{"id":"stepaudio-3-music-preview"},{"id":"stepaudio-2.5-tts"}]}`), nil
		})}}
		if executeModelQuery(&out, svc, strings.NewReader(query)) != nil {
			t.Fatal("music directory query failed")
		}
		capture("music-model-query", "model-query.schema.json", []byte(query))
		capture("music-model-receipt", "model-query-receipt.schema.json", out.Bytes())
	})

	t.Run("synchronous audio", func(t *testing.T) {
		for _, scenario := range []string{"speech", "generate", "transcribe", "rejected", "unknown", "invalid"} {
			t.Run(scenario, func(t *testing.T) {
				operation := scenario
				if scenario == "rejected" || scenario == "unknown" || scenario == "invalid" {
					operation = "speech"
				}
				result, err, raw := audioCLI(t, operation, func(r map[string]any) {
					if scenario == "invalid" {
						r["model"] = "unsupported"
					}
				}, func(*http.Request) (*http.Response, error) {
					switch scenario {
					case "rejected":
						return evaluationHTTP(422, `{"detail":"private"}`), nil
					case "unknown":
						return nil, errors.New("fixture network")
					case "transcribe":
						r := evaluationHTTP(200, `{"text":"转写文本"}`)
						r.Header.Set("Content-Type", "application/json")
						return r, nil
					default:
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"audio/wav"}}, Body: io.NopCloser(bytes.NewReader(audioWAVFixture()))}, nil
					}
				})
				expected := scenario == "speech" || scenario == "generate" || scenario == "transcribe"
				if (err == nil) != expected {
					t.Fatal("unexpected audio fixture result")
				}
				capture("audio-"+scenario, "audio-receipt.schema.json", []byte(raw))
				if scenario == "speech" {
					input := map[string]any{"artifact": result["artifact"]}
					captureValue("audio-verify-request", "audio-artifact-verify-request.schema.json", input)
					encoded, _ := json.Marshal(input)
					file := filepath.Join(t.TempDir(), "verify.json")
					os.WriteFile(file, encoded, 0600)
					for _, tamper := range []bool{false, true} {
						if tamper {
							os.WriteFile(jsonObject(result["artifact"])["path"].(string), []byte("changed"), 0600)
						}
						var out bytes.Buffer
						err := run([]string{"audio-verify", "--host", "codex", "--request", file}, strings.NewReader(""), &out)
						if (err == nil) == tamper {
							t.Fatal("unexpected verification result")
						}
						capture(fmt.Sprintf("audio-verify-%t", tamper), "audio-receipt.schema.json", out.Bytes())
					}
				}
			})
		}
		text, err := os.ReadFile("../../skills/puretokens-audio/references/executor-usage.md")
		if err != nil {
			t.Fatal(err)
		}
		blocks := regexp.MustCompile("(?s)```json\\s*\n(.*?)\n```").FindAllSubmatch(text, -1)
		if len(blocks) != 3 {
			t.Fatal("audio examples missing")
		}
		for index, block := range blocks {
			var input map[string]any
			json.Unmarshal(block[1], &input)
			operation := input["operation"].(string)
			// Keep documentation portable: replace only illustrative absolute paths
			// with temp output/attachment paths created by the CLI fixture.
			_, err, _ := audioCLI(t, operation, func(r map[string]any) {
				output, file := r["output_dir"], r["file"]
				for key := range r {
					delete(r, key)
				}
				for key, value := range input {
					r[key] = value
				}
				if operation == "transcribe" {
					r["file"] = file
				} else {
					r["output_dir"] = output
				}
			}, func(*http.Request) (*http.Response, error) {
				if operation == "transcribe" {
					r := evaluationHTTP(200, `{"text":"fixture text"}`)
					r.Header.Set("Content-Type", "application/json")
					return r, nil
				}
				format := input["response_format"].(string)
				data := audioWAVFixture()
				if format == "mp3" {
					data = audioMP3Fixture()
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {audioMIME(format)}}, Body: io.NopCloser(bytes.NewReader(data))}, nil
			})
			if err != nil {
				t.Fatal("documented audio request rejected")
			}
			capture(fmt.Sprintf("docs-audio-%d", index), "audio-request.schema.json", block[1])
		}
		var out bytes.Buffer
		svc := service{baseURL: apiOrigin, client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
			return evaluationHTTP(200, `{"data":[{"id":"stepaudio-2.5-tts"}]}`), nil
		})}}
		query := `{"kind":"audio","operation":"speech"}`
		if executeModelQuery(&out, svc, strings.NewReader(query)) != nil {
			t.Fatal("audio catalog failed")
		}
		capture("audio-model-query", "model-query.schema.json", []byte(query))
		capture("audio-model-receipt", "model-query-receipt.schema.json", out.Bytes())
	})

	t.Run("synchronous evaluation", func(t *testing.T) {
		for _, scenario := range []string{"success", "invalid-response", "rejected", "network", "invalid-request"} {
			t.Run(scenario, func(t *testing.T) {
				input := evaluationInputFixture
				if scenario == "invalid-request" {
					input = `{}`
				}
				_, err, raw := evaluationCLI(t, input, func(*http.Request) (*http.Response, error) {
					switch scenario {
					case "invalid-response":
						return evaluationHTTP(200, `{"answers":{}}`), nil
					case "rejected":
						return evaluationHTTP(422, `{"detail":"private"}`), nil
					case "network":
						return nil, errors.New("fixture network")
					default:
						return evaluationHTTP(200, evaluationOutputFixture), nil
					}
				})
				if (err == nil) != (scenario == "success") {
					t.Fatal("unexpected evaluation fixture result")
				}
				capture("evaluation-"+scenario, "evaluation-receipt.schema.json", []byte(raw))
			})
		}
		capture("evaluation-request", "evaluation-request.schema.json", []byte(evaluationInputFixture))
		text, err := os.ReadFile("../../skills/puretokens-evaluate/references/executor-usage.md")
		if err != nil {
			t.Fatal(err)
		}
		blocks := regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```").FindAllSubmatch(text, -1)
		if len(blocks) != 1 {
			t.Fatal("evaluation request example missing")
		}
		for index, block := range blocks {
			if _, err := decodeEvaluationRequest(block[1]); err != nil {
				t.Fatal("documented evaluation request rejected")
			}
			capture(fmt.Sprintf("docs-evaluation-%d", index), "evaluation-request.schema.json", block[1])
		}
		var output bytes.Buffer
		svc := service{baseURL: apiOrigin, client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
			return evaluationHTTP(200, `{"data":[{"id":"jev-latest"}]}`), nil
		})}}
		query := `{"kind":"evaluation"}`
		if executeModelQuery(&output, svc, strings.NewReader(query)) != nil {
			t.Fatal("evaluation catalog rejected")
		}
		capture("evaluation-model-query", "model-query.schema.json", []byte(query))
		capture("evaluation-model-receipt", "model-query-receipt.schema.json", output.Bytes())
	})

	t.Run("submission identity diagnostics", func(t *testing.T) {
		for _, scenario := range submissionIdentityCases() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if scenario.name == "short-http-body" {
					w.Header().Set("Content-Length", "4096")
				}
				io.WriteString(w, scenario.body)
			}))
			var output bytes.Buffer
			writer := &supportReceiptWriter{output: &output, command: "submit", host: "codex"}
			svc := fixtureService(server)
			svc.support = writer
			err := executePreparedTask(writer, svc, taskRequest{Kind: "video", Operation: "generate",
				Model: "seedance-2.0", Prompt: "fixture"})
			server.Close()
			if err == nil || decodeReceipt(t, &output).LocalErrorCode != scenario.code {
				t.Fatal("expected a classified unknown submission")
			}
			capture("identity-"+scenario.name, "executor-receipt.schema.json", output.Bytes())
		}
	})

	t.Run("public task identity round trips", func(t *testing.T) {
		for index, id := range taskIDBoundaryFixtures(t).Accepted {
			name := fmt.Sprintf("task-id-%d", index)
			request := taskRequest{Kind: "image", Operation: "continue", TaskID: id,
				Model: "gpt-image-2.5", OriginalOperation: "generate", RequestedCount: 1}
			if err := validateTaskRequest(request); err != nil {
				t.Fatalf("%s rejected a public task ID: %v", name, err)
			}
			captureValue(name+"-request", "executor-request.schema.json", map[string]any{
				"kind": request.Kind, "operation": request.Operation, "task_id": id,
				"model": request.Model, "original_operation": request.OriginalOperation,
				"requested_count": request.RequestedCount,
			})
			var output bytes.Buffer
			writer := &supportReceiptWriter{output: &output, host: "codex", command: "status"}
			writeReceipt(writer, taskReceipt(request, id, "pending"))
			capture(name+"-receipt", "executor-receipt.schema.json", output.Bytes())
			var document struct {
				Support struct {
					TaskID string `json:"task_id"`
				} `json:"support"`
			}
			expectedSupportID := id
			// Keep the existing credential-like text redaction: this legacy
			// fixture contains "sk-", but remains usable in the private record.
			if id == "pic-task-mu578qtd-yipyym:image-2" {
				expectedSupportID = ""
			}
			if json.Unmarshal(output.Bytes(), &document) != nil || document.Support.TaskID != expectedSupportID {
				t.Fatalf("%s dropped its ID from the support summary", name)
			}
			record := recordFromRequest(request)
			record.TaskID, record.Status = id, "pending"
			file := filepath.Join(t.TempDir(), "task.json")
			if err := saveTaskRecord(file, record, true); err != nil {
				t.Fatal(err)
			}
			restored, err := loadTaskRecord(file)
			if err != nil || restored.TaskID != id {
				t.Fatalf("%s could not resume its exact recorded ID: %v", name, err)
			}
			captureValue(name+"-artifact", "task-record.schema.json", restored)
		}
	})

	t.Run("documented media requests execute", func(t *testing.T) {
		pngPath := filepath.Join(t.TempDir(), "current.png")
		if err := os.WriteFile(pngPath, fixturePNG(t), 0600); err != nil {
			t.Fatal(err)
		}
		fences := regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```")
		for _, kind := range []string{"image", "video"} {
			document, err := os.ReadFile(filepath.Join("../../skills", "puretokens-"+kind, "references/executor-usage.md"))
			if err != nil {
				t.Fatal(err)
			}
			blocks := fences.FindAllSubmatch(document, -1)
			expected := 4
			if kind == "video" {
				expected++ // Separate local-file and public-URL reference examples.
			}
			if len(blocks) != expected {
				t.Fatalf("%s guide: expected %d executable examples", kind, expected)
			}
			for index, block := range blocks {
				name := fmt.Sprintf("docs-%s-%d", kind, index)
				capture(name, "executor-request.schema.json", block[1])
				request, err := decodeTaskRequest(bytes.NewReader(block[1]))
				if err != nil || request.Kind != kind {
					t.Fatalf("%s is not a kind-correct native request", name)
				}
				if request.TaskID != "" {
					request.Operation = "continue"
				}
				for index := range request.Attachments {
					request.Attachments[index].Path = pngPath
				}
				if request.OutputDir != "" {
					request.OutputDir = t.TempDir()
				}
				if err := validateTaskRequest(request); err != nil {
					t.Fatalf("%s fails native request validation: %v", name, err)
				}
				if request.Operation == "continue" {
					continue
				}
				if err := prepareProfileRequest(&request, profileService()); err != nil {
					t.Fatalf("%s is incompatible with its installed model profile: %v", name, err)
				}
				route := endpointFor(request)
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.Header().Set("X-Request-ID", "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99")
					if r.Method != http.MethodPost || r.URL.Path != route {
						t.Error("documented submission did an unexpected request")
					}
					if len(request.Attachments) > 0 {
						if err := r.ParseMultipartForm(1 << 20); err != nil {
							t.Error(err)
						} else {
							defer r.MultipartForm.RemoveAll()
							if len(r.MultipartForm.File[request.Attachments[0].Field]) != 1 {
								t.Error("documented native attachment was not sent")
							}
						}
					} else {
						io.Copy(io.Discard, r.Body)
					}
					io.WriteString(w, `{"id":"fixture-doc-task","status":"queued"}`)
				}))
				var output bytes.Buffer
				writer := &supportReceiptWriter{output: &output, command: "submit", host: "codex"}
				svc := fixtureService(server)
				svc.support = writer
				body, _ := json.Marshal(request)
				err = executeTask(bytes.NewReader(body), writer, svc)
				server.Close()
				if err != nil || calls != 1 {
					t.Fatalf("%s did not submit once without status/content reads: %v (%d calls)", name, err, calls)
				}
				capture(name+"-receipt", "executor-receipt.schema.json", output.Bytes())
			}
		}
	})

	t.Run("recorded lifecycle and delivery", func(t *testing.T) {
		pngBytes := fixturePNG(t)
		var calls []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.Method+" "+r.URL.RequestURI())
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/v1/images/generations":
				io.WriteString(w, `{"id":"fixture-record-task","status":"queued"}`)
			case r.Method == http.MethodGet && r.URL.Path == "/v1/images/fixture-record-task":
				io.WriteString(w, `{"id":"fixture-record-task","status":"completed"}`)
			case r.Method == http.MethodGet && r.URL.Path == "/v1/images/fixture-record-task/content":
				w.Header().Set("Content-Type", "image/png")
				w.Write(pngBytes)
			default:
				t.Error("record lifecycle used an unexpected request")
				http.Error(w, "unexpected route", 500)
			}
		}))
		defer server.Close()
		recordPath := filepath.Join(t.TempDir(), "task.json")
		outputDir := t.TempDir()
		request := taskRequest{Kind: "image", Operation: "generate", Model: "fixture-multi-image",
			Prompt: "synthetic-private-prompt", Parameters: map[string]any{"n": 2.0, "size": "2048x2048", "image_urls": []any{"https://reference.example.test/private.png"}}}
		steps := []struct {
			command string
			index   int
		}{{"submit", 0}, {"resume", 0}, {"content", 0}, {"delivered", 0}, {"content", 1}, {"delivered", 1}}
		svc := syntheticMultiImageService(t, fixtureService(server))
		now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
		svc.now = func() time.Time { return now }
		svc.wait = func(_ context.Context, delay time.Duration) bool {
			now = now.Add(delay)
			return true
		}
		for index, step := range steps {
			var output bytes.Buffer
			dir := ""
			if step.command == "content" {
				dir = outputDir
			}
			if err := executeRecordedTask(step.command, recordPath, request, step.index, dir, &output, svc); err != nil {
				t.Fatalf("%s: %v", step.command, err)
			}
			name := fmt.Sprintf("record-%d-%s", index, step.command)
			capture(name+"-receipt", "executor-receipt.schema.json", output.Bytes())
			recordData, err := os.ReadFile(recordPath)
			if err != nil {
				t.Fatal(err)
			}
			capture(name+"-artifact", "task-record.schema.json", recordData)
			for _, secret := range []string{"synthetic-private-prompt", "reference.example.test", "synthetic-fixture-token", `"prompt"`} {
				if bytes.Contains(recordData, []byte(secret)) || bytes.Contains(output.Bytes(), []byte(secret)) {
					t.Fatalf("%s retained private request data", name)
				}
			}
			result := decodeReceipt(t, &output)
			if result.TaskID != "fixture-record-task" || result.OriginalOperation != "generate" || result.RequestedCount != 2 {
				t.Fatalf("%s lost original task context", name)
			}
			if step.command == "delivered" {
				expected := "partially_delivered"
				if step.index == 1 {
					expected = "delivered"
				}
				if result.DeliveryStatus != expected || len(result.DeliveredIndexes) != step.index+1 {
					t.Fatal("delivery acknowledgement lost progress")
				}
			}
		}
		if !reflect.DeepEqual(calls, []string{
			"POST /v1/images/generations", "GET /v1/images/fixture-record-task",
			"GET /v1/images/fixture-record-task/content?index=0", "GET /v1/images/fixture-record-task/content?index=1",
		}) {
			t.Fatalf("record continuation resubmitted, redownloaded or used extra requests: %v", calls)
		}
	})

	t.Run("submission failure and retry receipts", func(t *testing.T) {
		for _, status := range []int{400, 503} {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(status)
				io.WriteString(w, `{"error":{"code":"fixture_failure","message":"The request failed."}}`)
			}))
			var output bytes.Buffer
			err := executeTask(strings.NewReader(`{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"fixture"}`), &output, fixtureService(server))
			server.Close()
			if err == nil || calls != 1 {
				t.Fatal("failed submission was not one-shot")
			}
			capture(fmt.Sprintf("submission-%d", status), "executor-receipt.schema.json", output.Bytes())
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "10")
			w.WriteHeader(429)
			io.WriteString(w, `{"error":{"code":"rate_limit","message":"Read this task later."}}`)
		}))
		defer server.Close()
		svc := fixtureService(server)
		svc.now = func() time.Time { return time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC) }
		var output bytes.Buffer
		err := executeExistingTask("status", strings.NewReader(`{"kind":"image","task_id":"fixture-retry","original_operation":"image_edit","model":"gpt-image-2","requested_count":2}`), &output, svc)
		if err == nil || decodeReceipt(t, &output).RetryNotBefore != "2026-09-05T00:00:10Z" {
			t.Fatal("status retry did not retain its absolute bound")
		}
		capture("status-retry", "executor-receipt.schema.json", output.Bytes())
	})

	t.Run("explicit validation", func(t *testing.T) {
		svc := profileService()
		svc.client = &http.Client{Transport: schemaRejectNetwork{t}}
		var output bytes.Buffer
		request := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2", Prompt: "fixture"}
		if err := executePreflight(&output, svc, request); err != nil {
			t.Fatal(err)
		}
		capture("validation-success", "executor-receipt.schema.json", output.Bytes())
		output.Reset()
		request.Parameters = map[string]any{"n": 7.0}
		if err := executePreflight(&output, svc, request); err == nil {
			t.Fatal("invalid count passed validation")
		}
		capture("validation-failure", "executor-receipt.schema.json", output.Bytes())
		output.Reset()
		svc = syntheticMultiImageService(t, svc)
		request.Model = "fixture-multi-image"
		request.Parameters = map[string]any{"width": true, "height": "large", "strength": false}
		if err := executePreflight(&output, svc, request); err == nil {
			t.Fatal("invalid parameter types passed validation")
		}
		capture("validation-invalid-types", "executor-receipt.schema.json", output.Bytes())
	})

	t.Run("credential failure retains continuation context", func(t *testing.T) {
		var output bytes.Buffer
		err := run([]string{"status", "--host", "unsupported-fixture-host"},
			strings.NewReader(`{"kind":"image","task_id":"fixture-existing","original_operation":"image_edit","model":"gpt-image-2","requested_count":1}`), &output)
		if err == nil {
			t.Fatal("unsupported fixture host unexpectedly resolved credentials")
		}
		result := decodeReceipt(t, &output)
		if result.Operation != "continue" || result.TaskID != "fixture-existing" || result.OriginalOperation != "image_edit" {
			t.Fatal("credential failure lost continuation context")
		}
		capture("continuation-credential-failure", "executor-receipt.schema.json", output.Bytes())
	})

	t.Run("balance usage snapshot and published units", func(t *testing.T) {
		var calls []string
		svc := balanceFixtureService(t, func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.Method+" "+r.URL.Path)
			if r.URL.Path == balanceUsagePath {
				io.WriteString(w, balanceUsageFixture)
			} else {
				io.WriteString(w, balanceUnitFixture)
			}
		})
		var output bytes.Buffer
		writer := &supportReceiptWriter{output: &output, command: "balance", host: "codex"}
		svc.support = writer
		if err := executeBalance(writer, svc); err != nil {
			t.Fatal(err)
		}
		capture("balance", "balance-receipt.schema.json", output.Bytes())
		var envelope struct{ Result json.RawMessage }
		if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		capture("balance-projection", "balance-snapshot.schema.json", envelope.Result)
		var result map[string]any
		json.Unmarshal(envelope.Result, &result)
		if result["total"] != 29.672244 || result["used"] != 19.672244 || result["remaining"] != float64(10) ||
			!reflect.DeepEqual(calls, []string{"GET " + balanceUsagePath, "GET " + balanceUnitPath}) {
			t.Fatal("balance did not project actual available quota and units correctly")
		}
	})

	t.Run("balance-specific failure diagnostics", func(t *testing.T) {
		for _, phase := range []string{"usage", "unit"} {
			svc := balanceFixtureService(t, func(w http.ResponseWriter, r *http.Request) {
				if phase == "unit" && r.URL.Path == balanceUsagePath {
					io.WriteString(w, balanceUsageFixture)
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				io.WriteString(w, `{"success":false,"error":{"code":"auth_required","message":"Authentication required"}}`)
			})
			var output bytes.Buffer
			if executeBalance(&output, svc) == nil {
				t.Fatal("balance rejection accepted")
			}
			capture("balance-"+phase+"-failure", "executor-receipt.schema.json", output.Bytes())
		}
	})

	t.Run("documented model filter", func(t *testing.T) {
		document, err := os.ReadFile("../../skills/puretokens-models/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		match := regexp.MustCompile("`(\\{\"kind\":\"video\",\"operation\":\"image_to_video\",\"parameters\":\\{[^`]+\\}\\})`").FindSubmatch(document)
		if len(match) != 2 {
			t.Fatal("model Skill does not contain its executable filter example")
		}
		capture("models-filter", "model-query.schema.json", match[1])
		for _, filtered := range []bool{false, true} {
			var input io.Reader
			if filtered {
				input = bytes.NewReader(match[1])
			}
			result, output, err := runModelQueryFixture(t, input, modelQueryFixture, 200)
			if err != nil {
				t.Fatal(err)
			}
			if filtered && !reflect.DeepEqual(modelQueryIDs(t, result), []string{"video-audio"}) {
				t.Fatal("documented filter did not match declared capabilities")
			}
			capture(fmt.Sprintf("models-%t", filtered), "model-query-receipt.schema.json", []byte(output))
		}
	})

	t.Run("doctor and init", func(t *testing.T) {
		home := doctorIsolatedHome(t)
		root := filepath.Join(home, ".agents", "skills")
		doctorFixtureInstallation(t, root, executorVersion)
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method != http.MethodGet {
				t.Error("doctor created a task")
			}
			if r.URL.Path == "/v1" {
				io.WriteString(w, `{"status":"ok","name":"Pure Tokens API","base_url":"/v1"}`)
			} else if r.URL.Path == "/v1/media/models" {
				io.WriteString(w, `{"data":[]}`)
			} else {
				t.Error("doctor used an unexpected route")
			}
		}))
		defer server.Close()
		svc := service{baseURL: server.URL, client: server.Client(), token: "synthetic-doctor-fixture", profilesRoot: root}
		var output bytes.Buffer
		writer := &supportReceiptWriter{output: &output, command: "doctor", host: "codex"}
		svc.support = writer
		if err := executeDoctor(writer, svc, "codex"); err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatal("doctor did not use exactly the two read-only checks")
		}
		capture("doctor-success", "doctor-receipt.schema.json", output.Bytes())
		var result doctorReceipt
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		captureValue("init-success", "init-receipt.schema.json", result.Connection)
		output.Reset()
		svc.token = ""
		writer = &supportReceiptWriter{output: &output, command: "doctor", host: "codex"}
		svc.support = writer
		if err := executeDoctorCredentialFailure(writer, svc, "codex", errors.New("synthetic-private-error")); err == nil || calls != 2 {
			t.Fatal("doctor credential failure used network")
		}
		capture("doctor-no-credential", "doctor-receipt.schema.json", output.Bytes())
		captureValue("init-no-credential", "init-receipt.schema.json", initCredentialFailure(errors.New("synthetic-private-error")))
	})

	t.Run("complete-local-failure-fields", func(t *testing.T) {
		var output bytes.Buffer
		request := taskRequest{Kind: "image", Operation: "continue", TaskID: "existing-task", RetryNotBefore: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
		data, _ := json.Marshal(request)
		if err := executeExistingTask("status", bytes.NewReader(data), &output, service{}); err == nil {
			t.Fatal("early status read accepted")
		}
		capture("status-not-due", "executor-receipt.schema.json", output.Bytes())
		output.Reset()
		writer := recordReceiptWriter{output: &output, path: filepath.Join(t.TempDir(), "absent.json"), record: taskRecord{Format: taskRecordFormat, Kind: "image"}}
		writer.writeReceipt(receipt{OK: true, Kind: "image", Operation: "generate", TaskID: "existing-task", Status: "pending", SubmissionOutcome: "accepted"})
		if writer.err == nil {
			t.Fatal("missing record write succeeded")
		}
		capture("accepted-record-write-failure", "executor-receipt.schema.json", output.Bytes())
	})

	t.Run("recorded failure and explicit redelivery", func(t *testing.T) {
		media := fixturePNG(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/content") {
				w.Header().Set("Content-Type", "image/png")
				w.Write(media)
				return
			}
			io.WriteString(w, `{"id":"schema-recovery","status":"failed","error":{"code":"model_not_allowed","message":"private diagnostic"}}`)
		}))
		defer server.Close()
		svc := fixtureService(server)
		dir := t.TempDir()
		path := filepath.Join(dir, "failed.json")
		saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "schema-recovery", Status: "pending", RequestedCount: 1}, true)
		var output bytes.Buffer
		if err := executeRecordedTask("status", path, taskRequest{}, 0, "", &output, svc); err == nil {
			t.Fatal("fixture must fail")
		}
		capture("recorded-failure-live", "executor-receipt.schema.json", output.Bytes())
		saved, _ := os.ReadFile(path)
		capture("recorded-failure-artifact", "task-record.schema.json", saved)
		output.Reset()
		if err := run([]string{"resume", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &output); err == nil {
			t.Fatal("restored failure must remain failed")
		}
		capture("recorded-failure-resume", "executor-receipt.schema.json", output.Bytes())

		path = filepath.Join(dir, "completed.json")
		saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "schema-recovery", Status: "completed", RequestedCount: 1}, true)
		for _, command := range []string{"content", "delivered", "redeliver", "resume", "delivered"} {
			output.Reset()
			if err := executeRecordedTask(command, path, taskRequest{}, 0, dir, &output, svc); err != nil {
				t.Fatal(err)
			}
			if command == "redeliver" {
				capture("explicit-redelivery-receipt", "executor-receipt.schema.json", output.Bytes())
				saved, _ = os.ReadFile(path)
				capture("explicit-redelivery-artifact", "task-record.schema.json", saved)
			}
			if command == "resume" {
				capture("explicit-redelivery-resume", "executor-receipt.schema.json", output.Bytes())
			}
		}
		capture("explicit-redelivery-ack", "executor-receipt.schema.json", output.Bytes())
		captureValue("adaptive-receipt", "executor-receipt.schema.json", guideReceipt(taskReceipt(taskRequest{Kind: "video", Operation: "generate", Model: "minimax_h3", Parameters: map[string]any{"aspect_ratio": "adaptive"}}, "schema-adaptive", "pending")))
	})

	if outputPath := os.Getenv("PURETOKENS_SCHEMA_EXAMPLES_OUT"); outputPath != "" {
		data, err := json.MarshalIndent(examples, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(outputPath, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

type schemaRejectNetwork struct{ t *testing.T }

func (transport schemaRejectNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	transport.t.Error("installed-profile validation attempted network")
	return nil, errors.New("network forbidden in this fixture")
}
