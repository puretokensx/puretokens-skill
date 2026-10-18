package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const embeddingResponseFixture = `{"data":[{"index":1,"embedding":[0.5,-0.75]},{"index":0,"embedding":[1,0]}],"debug":"PRIVATE-RESPONSE"}`
const rerankResponseFixture = `{"results":[{"index":0,"relevance_score":-0.3,"document":{"text":"PRIVATE-RESPONSE"}},{"index":1,"relevance_score":1.2}]}`

func retrievalCLI(t *testing.T, kind string, change func(map[string]any), transport evaluationTransport, options ...string) (map[string]any, error, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("model_provider='fixture'\n[model_providers.fixture]\nbase_url='https://api.puretokensx.com/v1'\nexperimental_bearer_token='synthetic-retrieval-token'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"model": "qwen3.7-text-embedding", "input": []string{"PRIVATE-INPUT-0", "PRIVATE-INPUT-1"}, "output_file": filepath.Join(home, "vectors.jsonl")}
	if kind == "rerank" {
		request = map[string]any{"model": "qwen3.7-text-rerank", "query": "PRIVATE-QUERY", "documents": []string{"PRIVATE-DOC-0", "PRIVATE-DOC-1"}, "output_file": filepath.Join(home, "ranking.json")}
	}
	if change != nil {
		change(request)
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(home, "request.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	var output bytes.Buffer
	err = run(append([]string{kind, "--host", "codex", "--request", file}, options...), strings.NewReader(""), &output)
	var result map[string]any
	if json.Unmarshal(output.Bytes(), &result) != nil {
		t.Fatal("expected exactly one safe JSON receipt", output.String())
	}
	return result, err, output.String()
}

func TestEmbeddingSavesOrderedVectorsWithoutPrintingInputsOrVectors(t *testing.T) {
	calls := 0
	result, err, raw := retrievalCLI(t, "embedding", nil, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.String() != "https://api.puretokensx.com/v1/embeddings" ||
			r.Header.Get("Authorization") != "Bearer synthetic-retrieval-token" ||
			r.Header.Get("Content-Type") != "application/json" {
			t.Error("wrong embedding transport")
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 90*time.Second {
			t.Error("missing bounded synchronous deadline")
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		json.Unmarshal(body, &payload)
		if len(payload) != 3 || payload["model"] != "qwen3.7-text-embedding" || payload["encoding_format"] != "float" ||
			len(payload["input"].([]any)) != 2 || bytes.Contains(body, []byte("output_file")) {
			t.Error("embedding API payload changed or leaked local output path")
		}
		return evaluationHTTP(200, embeddingResponseFixture), nil
	})
	if err != nil || calls != 1 || result["ok"] != true || result["submission_outcome"] != "accepted" {
		t.Fatalf("embedding failed: %v calls=%d %v", err, calls, result)
	}
	if strings.Contains(raw, "PRIVATE-") || strings.Contains(raw, "synthetic-retrieval-token") || strings.Contains(raw, `"embedding":`) {
		t.Fatal("receipt leaked input, raw response, credential or full vectors")
	}
	artifact := jsonObject(result["artifact"])
	data, err := os.ReadFile(artifact["path"].(string))
	if err != nil || string(data) != "{\"index\":0,\"dimensions\":2,\"embedding\":[1,0]}\n{\"index\":1,\"dimensions\":2,\"embedding\":[0.5,-0.75]}\n" {
		t.Fatalf("vector rows lost input ordering or values: %s %v", data, err)
	}
	if artifact["item_count"] != float64(2) || artifact["dimensions"] != float64(2) {
		t.Fatal("missing compact output summary")
	}
}

func TestRerankPreservesOriginalIndicesAndUnnormalizedScores(t *testing.T) {
	calls := 0
	result, err, raw := retrievalCLI(t, "rerank", nil, func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if r.Method != "POST" || r.URL.String() != "https://api.puretokensx.com/v1/rerank" || body["model"] != "qwen3.7-text-rerank" ||
			body["return_documents"] != false || body["top_n"] != float64(2) || body["output_file"] != nil {
			t.Error("wrong rerank protocol")
		}
		return evaluationHTTP(200, rerankResponseFixture), nil
	})
	if err != nil || calls != 1 || result["ok"] != true {
		t.Fatalf("rerank failed: %v calls=%d %v", err, calls, result)
	}
	if strings.Contains(raw, "PRIVATE-") || strings.Contains(raw, "synthetic-retrieval-token") {
		t.Fatal("rerank receipt leaked private content")
	}
	rows := jsonObject(result["result"])["results"].([]any)
	first, second := jsonObject(rows[0]), jsonObject(rows[1])
	if first["rank"] != float64(1) || first["index"] != float64(1) || first["relevance_score"] != 1.2 ||
		second["rank"] != float64(2) || second["index"] != float64(0) || second["relevance_score"] != -0.3 {
		t.Fatal("ranking normalized scores or lost original document indices")
	}
	data, err := os.ReadFile(jsonObject(result["artifact"])["path"].(string))
	if err != nil || !bytes.Contains(data, []byte(`"relevance_score":-0.3`)) || bytes.Contains(data, []byte("PRIVATE-")) {
		t.Fatal("ranking artifact missing or includes upstream documents")
	}
}

func TestRetrievalRejectsInvalidInputsBeforeAnyNetwork(t *testing.T) {
	for _, kind := range []string{"embedding", "rerank"} {
		for name, change := range map[string]func(map[string]any){
			"unknown-model": func(r map[string]any) { r["model"] = "private-unreviewed-model" },
			"wrong-family": func(r map[string]any) {
				r["model"] = "qwen3.7-text-rerank"
				if kind == "rerank" {
					r["model"] = "qwen3.7-text-embedding"
				}
			},
			"extra-field":     func(r map[string]any) { r["base_url"] = "https://private.test" },
			"null-query":      func(r map[string]any) { r["query"] = nil },
			"relative-output": func(r map[string]any) { r["output_file"] = "result.json" },
			"wrong-extension": func(r map[string]any) { r["output_file"] = filepath.Join(t.TempDir(), "result.txt") },
			"missing-directory": func(r map[string]any) {
				r["output_file"] = filepath.Join(t.TempDir(), "absent", filepath.Base(r["output_file"].(string)))
			},
			"empty-inputs": func(r map[string]any) {
				key := "input"
				if kind == "rerank" {
					key = "documents"
				}
				r[key] = []string{""}
			},
			"batch-too-large": func(r map[string]any) {
				items := make([]string, 65)
				for i := range items {
					items[i] = "x"
				}
				key := "input"
				if kind == "rerank" {
					key = "documents"
				}
				r[key] = items
			},
			"long-text": func(r map[string]any) {
				key := "input"
				if kind == "rerank" {
					key = "documents"
				}
				r[key] = []string{strings.Repeat("界", 32769)}
			},
			"null-top": func(r map[string]any) { r["top_n"] = nil },
		} {
			t.Run(kind+"/"+name, func(t *testing.T) {
				calls := 0
				result, err, _ := retrievalCLI(t, kind, change, func(*http.Request) (*http.Response, error) {
					calls++
					return evaluationHTTP(200, embeddingResponseFixture), nil
				})
				if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" || result["command"] != kind {
					t.Fatalf("invalid input submitted or lost typed receipt: %v %v calls=%d", result, err, calls)
				}
			})
		}
		for _, option := range []string{"--record=", "--index=0", "--output-dir=", "--stream", "extra", "--host="} {
			t.Run(kind+"/"+option, func(t *testing.T) {
				calls := 0
				result, err, _ := retrievalCLI(t, kind, nil, func(*http.Request) (*http.Response, error) {
					calls++
					return evaluationHTTP(200, embeddingResponseFixture), nil
				}, option)
				if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" || result["command"] != kind {
					t.Fatalf("invalid CLI submitted or lost typed receipt: %v %v", result, err)
				}
			})
		}
	}
}

func TestRetrievalDoesNotOverwriteExistingOutputs(t *testing.T) {
	for _, kind := range []string{"embedding", "rerank"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			var destination string
			result, err, _ := retrievalCLI(t, kind, func(r map[string]any) {
				destination = r["output_file"].(string)
				if err := os.WriteFile(destination, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}, func(*http.Request) (*http.Response, error) {
				calls++
				return evaluationHTTP(200, embeddingResponseFixture), nil
			})
			data, _ := os.ReadFile(destination)
			if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" || string(data) != "original" {
				t.Fatal("existing output was overwritten or billed")
			}
		})
	}
}

func TestRetrievalRejectsMalformedResultsWithoutLeavingArtifacts(t *testing.T) {
	cases := []struct{ kind, name, response string }{
		{"embedding", "missing-index", `{"data":[{"embedding":[1,2]},{"index":1,"embedding":[3,4]}]}`},
		{"embedding", "duplicate-index", `{"data":[{"index":0,"embedding":[1,2]},{"index":0,"embedding":[3,4]}]}`},
		{"embedding", "dimension-mismatch", `{"data":[{"index":0,"embedding":[1]},{"index":1,"embedding":[3,4]}]}`},
		{"embedding", "empty-vector", `{"data":[{"index":0,"embedding":[]},{"index":1,"embedding":[]}]}`},
		{"embedding", "nonfinite", `{"data":[{"index":0,"embedding":[1e1000]},{"index":1,"embedding":[0]}]}`},
		{"embedding", "string-number", `{"data":[{"index":0,"embedding":["1"]},{"index":1,"embedding":[0]}]}`},
		{"embedding", "missing-row", `{"data":[{"index":0,"embedding":[1]}]}`},
		{"embedding", "duplicate-json-key", `{"data":[],"data":[{"index":0,"embedding":[1]},{"index":1,"embedding":[2]}]}`},
		{"rerank", "duplicate-index", `{"results":[{"index":0,"relevance_score":1},{"index":0,"relevance_score":0}]}`},
		{"rerank", "missing-score", `{"results":[{"index":0},{"index":1,"relevance_score":0}]}`},
		{"rerank", "out-of-bounds-index", `{"results":[{"index":2,"relevance_score":1},{"index":1,"relevance_score":0}]}`},
		{"rerank", "fractional-index", `{"results":[{"index":0.5,"relevance_score":1},{"index":1,"relevance_score":0}]}`},
		{"rerank", "missing-row", `{"results":[]}`},
		{"rerank", "error-with-results", `{"error":{"message":"PRIVATE-ERROR"},"results":[{"index":0,"relevance_score":1},{"index":1,"relevance_score":0}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.name, func(t *testing.T) {
			calls := 0
			var destination string
			result, err, raw := retrievalCLI(t, tc.kind, func(r map[string]any) { destination = r["output_file"].(string) },
				func(*http.Request) (*http.Response, error) {
					calls++
					return evaluationHTTP(200, tc.response), nil
				})
			_, statErr := os.Lstat(destination)
			if err == nil || calls != 1 || result["submission_outcome"] != "unknown" ||
				result["local_error_code"] != tc.kind+"_response_invalid" || !os.IsNotExist(statErr) || strings.Contains(raw, "PRIVATE-") {
				t.Fatalf("unverified response became successful artifact: %v %v calls=%d", result, err, calls)
			}
		})
	}
}

func TestRetrievalNeverRetriesRejectedUnknownOrRedirectedRequests(t *testing.T) {
	for _, kind := range []string{"embedding", "rerank"} {
		for _, status := range []int{0, 302, 401, 429, 503} {
			calls := 0
			result, err, raw := retrievalCLI(t, kind, nil, func(*http.Request) (*http.Response, error) {
				calls++
				if status == 0 {
					return nil, errors.New("PRIVATE-NETWORK")
				}
				response := evaluationHTTP(status, `{"error":{"message":"PRIVATE-ERROR","code":"private-code"}}`)
				response.Header.Set("Location", "https://private.test/credential-sink")
				response.Header.Set("Retry-After", "2")
				return response, nil
			})
			outcome := "unknown"
			if status >= 400 && status < 500 {
				outcome = "rejected"
			}
			if err == nil || calls != 1 || result["submission_outcome"] != outcome || strings.Contains(raw, "PRIVATE-") {
				t.Fatalf("%s HTTP %d unsafe retry/outcome: %v calls=%d %v", kind, status, result, calls, err)
			}
		}
	}
}

func TestRetrievalOutputRaceDoesNotOverwriteOrTriggerRegeneration(t *testing.T) {
	var destination string
	calls := 0
	result, err, _ := retrievalCLI(t, "embedding", func(r map[string]any) { destination = r["output_file"].(string) },
		func(*http.Request) (*http.Response, error) {
			calls++
			os.WriteFile(destination, []byte("concurrent-user-file"), 0600)
			return evaluationHTTP(200, embeddingResponseFixture), nil
		})
	data, _ := os.ReadFile(destination)
	if err == nil || calls != 1 || string(data) != "concurrent-user-file" || result["submission_outcome"] != "accepted" ||
		result["failure_phase"] != "content" || result["local_error_code"] != "embedding_output_unavailable" {
		t.Fatalf("output race overwrote a file or mislabeled paid success: %v %v", result, err)
	}
}

func TestRetrievalDiscoveryUsesAuthenticatedDirectoryAndReviewedExactIDs(t *testing.T) {
	for _, kind := range []string{"embedding", "rerank"} {
		var output bytes.Buffer
		calls := 0
		svc := service{baseURL: apiOrigin, client: &http.Client{Transport: evaluationTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "GET" || r.URL.String() != "https://api.puretokensx.com/v1/models" {
				t.Error("discovery used the wrong directory or submitted inference")
			}
			return evaluationHTTP(200, `{"data":[{"id":"qwen3.7-text-embedding"},{"id":"qwen3.7-text-embedding-flash"},{"id":"qwen3.7-text-rerank"},{"id":"qwen3.7-text-rerank-guessed"},{"id":"private-model"},{"id":"qwen3.7-text-rerank"}]}`), nil
		})}}
		err := executeModelQuery(&output, svc, strings.NewReader(`{"kind":"`+kind+`"}`))
		var result map[string]any
		json.Unmarshal(output.Bytes(), &result)
		count := float64(2)
		if kind == "rerank" {
			count = 1
		}
		if err != nil || calls != 1 || jsonObject(result["result"])["matched_count"] != count || strings.Contains(output.String(), "private-model") {
			t.Fatalf("discovery guessed capability or leaked catalog: %v %v", result, err)
		}
	}
}

func TestRetrievalRequestAndResultBounds(t *testing.T) {
	for _, top := range []any{0, 3, 1.5, "1", true} {
		calls := 0
		_, err, _ := retrievalCLI(t, "rerank", func(r map[string]any) { r["top_n"] = top }, func(*http.Request) (*http.Response, error) {
			calls++
			return evaluationHTTP(200, rerankResponseFixture), nil
		})
		if err == nil || calls != 0 {
			t.Fatal("invalid top_n submitted")
		}
	}
	result, err, _ := retrievalCLI(t, "rerank", func(r map[string]any) { r["top_n"] = 1 }, func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["top_n"] != float64(1) {
			t.Fatal("top_n changed before submission")
		}
		return evaluationHTTP(200, `{"results":[{"index":1,"relevance_score":0.1}]}`), nil
	})
	if err != nil || result["ok"] != true || jsonObject(result["artifact"])["item_count"] != float64(1) {
		t.Fatal("explicit top_n was not honored")
	}
	for _, response := range []string{
		strings.Repeat(" ", 2<<20) + embeddingResponseFixture,
		`{"data":[{"index":0,"embedding":[1,2]},{"index":1,"embedding":[1,2]}]} {}`,
	} {
		calls := 0
		result, err, _ := retrievalCLI(t, "embedding", nil, func(*http.Request) (*http.Response, error) {
			calls++
			return evaluationHTTP(200, response), nil
		})
		if err == nil || calls != 1 || result["submission_outcome"] != "unknown" {
			t.Fatal("oversized/trailing response was not rejected safely")
		}
	}
}

func TestRetrievalIncompleteOutputAndCredentialFailureDoNotSubmit(t *testing.T) {
	for _, kind := range []string{"embedding", "rerank"} {
		calls := 0
		result, err, _ := retrievalCLI(t, kind, func(r map[string]any) {
			if err := os.Mkdir(r["output_file"].(string)+".incomplete", 0700); err != nil {
				t.Fatal(err)
			}
		}, func(*http.Request) (*http.Response, error) {
			calls++
			return evaluationHTTP(200, embeddingResponseFixture), nil
		})
		if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" {
			t.Fatal("unresolved output conflict was billed")
		}
		result, err, raw := retrievalCLI(t, kind, nil, func(*http.Request) (*http.Response, error) {
			calls++
			return evaluationHTTP(200, embeddingResponseFixture), nil
		}, "--host", "fixture-unsupported")
		if err == nil || calls != 0 || result["submission_outcome"] != "not_submitted" || strings.Contains(raw, "synthetic-retrieval-token") {
			t.Fatal("unsupported host was submitted or borrowed another host credential")
		}
	}
}
