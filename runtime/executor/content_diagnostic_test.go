//go:build content_diagnostic

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestContentDiagnosticSingleGET(t *testing.T) {
	payload := fixturePNG(t)
	for _, mode := range []string{"executor", "plain", "http1", "long-idle"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/v1/images/pic-fixture/content" || r.URL.RawQuery != "index=0" || r.Header.Get("Authorization") != "Bearer synthetic-fixture-token" {
					t.Error("unexpected request identity")
				}
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				w.Header().Set("X-Request-ID", "11111111-1111-4111-8111-111111111111")
				w.Header().Set("X-Private-Debug", "private-upstream-marker")
				w.Write(payload)
			}))
			server.EnableHTTP2 = true
			server.StartTLS()
			defer server.Close()
			svc := fixtureService(server)
			svc.host = "codex"
			record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1, OriginalOperation: "generate"}
			result := diagnoseContent(context.Background(), svc, record, 0, mode, diagnosticBudgets{total: time.Second, response: time.Second, idle: time.Second})
			if !result.OK || calls.Load() != 1 || result.Requests != 1 || result.Transfer == nil || result.Transfer.BytesReceived != int64(len(payload)) || !result.Transfer.ReadComplete || !result.Transfer.FileVerified {
				t.Fatalf("incomplete diagnostic: %+v", result)
			}
			wantProtocol := "HTTP/2.0"
			if mode == "http1" {
				wantProtocol = "HTTP/1.1"
			}
			if result.Protocol != wantProtocol {
				t.Fatalf("protocol=%s", result.Protocol)
			}
			encoded, _ := json.Marshal(result)
			for _, forbidden := range []string{svc.token, server.URL, "private-upstream-marker", "downloaded_paths", "connection_id"} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatal("private diagnostic evidence")
				}
			}
		})
	}
}

func TestContentDiagnosticValidationBeforeCredentials(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "absent"))
	base := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1, OriginalOperation: "generate"}
	for _, mode := range []string{"pending", "unknown", "delivered", "downloaded", "reconcile", "retry", "video", "invalid-probe", "missing-consent", "index"} {
		t.Run(mode, func(t *testing.T) {
			record := base
			switch mode {
			case "pending":
				record.Status = "pending"
			case "unknown":
				record.TaskID = ""
				record.SubmissionOutcome = "unknown"
			case "delivered":
				record.Delivered = []int{0}
			case "downloaded":
				record.Downloaded = map[int]string{0: "unverified-file"}
			case "reconcile":
				record.ReconciliationRequired = true
			case "retry":
				record.RetryNotBefore = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
			case "video":
				record.Kind = "video"
			}
			data, _ := json.Marshal(record)
			path := writeFixture(t, "task.json", string(data))
			args := []string{"--host", "codex", "--record", path, "--probe", "executor"}
			if mode == "invalid-probe" {
				args[5] = "unknown"
			}
			if mode != "missing-consent" {
				args = append(args, "--confirm-one-get")
			}
			if mode == "index" {
				args = append(args, "--index", "1")
			}
			var output bytes.Buffer
			if runContentDiagnostic(args, &output) == 0 || !strings.Contains(output.String(), "diagnostic_request_invalid") {
				t.Fatalf("validation skipped: %s", &output)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, data) {
				t.Fatal("record changed")
			}
		})
	}
}

func TestContentDiagnosticStopsRedirectAndRejection(t *testing.T) {
	for _, mode := range []string{"executor", "plain"} {
		for _, status := range []int{206, 302, 401, 429} {
			t.Run(mode+strconv.Itoa(status), func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.Header().Set("Location", "/unexpected")
					w.WriteHeader(status)
					w.Write([]byte("private-response-marker"))
				}))
				defer server.Close()
				result := diagnoseContent(context.Background(), fixtureService(server), taskRecord{Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1}, 0, mode, diagnosticBudgets{total: time.Second, response: time.Second, idle: time.Second})
				encoded, _ := json.Marshal(result)
				if result.OK || calls != 1 || bytes.Contains(encoded, []byte("private-response-marker")) {
					t.Fatalf("unsafe rejection: %s", encoded)
				}
			})
		}
	}
}

func TestContentDiagnosticBlocksTransportReplay(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.puretokensx.com/v1/images/pic-fixture/content?index=0", nil)
	result := &contentDiagnostic{}
	calls := 0
	transport := &diagnosticTransport{result: result, cancel: cancel, base: evaluationTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		trace := httptrace.ContextClientTrace(request.Context())
		trace.GotConn(httptrace.GotConnInfo{})
		if request.Context().Err() != nil {
			t.Fatal("first request blocked")
		}
		trace.GotConn(httptrace.GotConnInfo{Reused: true})
		if context.Cause(request.Context()) != errDiagnosticReplay {
			t.Fatal("internal transport replay not blocked")
		}
		return nil, context.Cause(request.Context())
	})}
	if _, err := transport.RoundTrip(request); err == nil {
		t.Fatal("replay error missing")
	}
	if _, err := transport.RoundTrip(request); err == nil || calls != 1 || result.Requests != 1 {
		t.Fatal("second application GET allowed")
	}
}

func TestContentDiagnosticMatchedIdleBudgets(t *testing.T) {
	payload := fixturePNG(t)
	for _, mode := range []string{"executor", "plain", "long-idle"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				w.Write(payload[:32])
				w.(http.Flusher).Flush()
				select {
				case <-time.After(300 * time.Millisecond):
					w.Write(payload[32:])
				case <-r.Context().Done():
				}
			}))
			defer server.Close()
			budgets := diagnosticBudgets{total: time.Second, response: time.Second, idle: 100 * time.Millisecond}
			if mode == "long-idle" {
				budgets.idle = 800 * time.Millisecond
			}
			result := diagnoseContent(context.Background(), fixtureService(server), taskRecord{Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1}, 0, mode, budgets)
			if calls != 1 || result.Transfer == nil {
				t.Fatal("missing single-request evidence")
			}
			if mode == "long-idle" {
				if !result.OK || !result.Transfer.FileVerified {
					t.Fatalf("long-idle incomplete: %+v", result)
				}
			} else if result.OK || result.LocalErrorCode != "media_download_stalled" || result.Transfer.BytesReceived != 32 || result.Transfer.ReadComplete || result.Transfer.FileVerified {
				t.Fatalf("idle policies differ: %+v", result)
			}
		})
	}
}

func TestContentDiagnosticRejectsTruncatedAndOversizedBody(t *testing.T) {
	payload := fixturePNG(t)
	for _, mode := range []string{"executor", "plain"} {
		for _, length := range []int{len(payload) + 100, int(maxImageBytes) + 1} {
			t.Run(mode+strconv.Itoa(length), func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.Header().Set("Content-Type", "image/png")
					w.Header().Set("Content-Length", strconv.Itoa(length))
					w.Write(payload)
				}))
				defer server.Close()
				result := diagnoseContent(context.Background(), fixtureService(server), taskRecord{Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1}, 0, mode, diagnosticBudgets{total: time.Second, response: time.Second, idle: time.Second})
				if result.OK || calls != 1 || (result.Transfer != nil && result.Transfer.FileVerified) {
					t.Fatalf("invalid bytes accepted: %+v", result)
				}
			})
		}
	}
}

func TestContentDiagnosticCLIUsesSameHostConnection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("model_provider='fixture'\n[model_providers.fixture]\nbase_url='https://api.puretokensx.com/v1'\nexperimental_bearer_token='synthetic-fixture-token'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	payload := fixturePNG(t)
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Host != "api.puretokensx.com" || r.Header.Get("Authorization") != "Bearer synthetic-fixture-token" {
			t.Error("wrong active connection")
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(payload)
	}))
	defer server.Close()
	transport := server.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "api.puretokensx.com:443" {
			t.Error("unexpected request target")
		}
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous; transport.CloseIdleConnections() })
	for _, binding := range []string{"", "11111111-1111-4111-8111-111111111111"} {
		record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1, OriginalOperation: "generate", ConnectionID: binding}
		data, _ := json.Marshal(record)
		path := writeFixture(t, "task.json", string(data))
		var output bytes.Buffer
		before := calls.Load()
		code := runContentDiagnostic([]string{"--host", "codex", "--record", path, "--probe", "plain", "--confirm-one-get"}, &output)
		if binding == "" {
			if code != 0 || calls.Load() != before+1 {
				t.Fatalf("same-host request failed: %s", &output)
			}
		} else if code == 0 || calls.Load() != before || !strings.Contains(output.String(), "task_connection_changed") {
			t.Fatalf("changed connection accepted: %s", &output)
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(data, after) || strings.Contains(output.String(), "synthetic-fixture-token") {
			t.Fatal("record or credential exposed")
		}
	}
}
