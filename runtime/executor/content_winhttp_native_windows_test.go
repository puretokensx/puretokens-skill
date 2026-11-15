//go:build content_diagnostic && windows

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// These fixtures exercise the real DLL, synthetic bytes and loopback only.
// They never load host credentials, resolve an external host, or submit media.
func TestContentWinHTTPNativeFixtures(t *testing.T) {
	payload := fixturePNG(t)
	for _, mode := range []string{"complete", "chunked", "truncated", "oversized", "invalid", "redirect", "rejected", "throttled", "request-timeout", "disconnect", "idle", "headers", "total", "duplicate-id", "encoded", "cancelled", "untrusted-tls"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			done := make(chan struct{})
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/v1/images/pic-fixture/content" || r.URL.RawQuery != "index=0" || r.Header.Get("Authorization") != "Bearer synthetic-fixture-token" {
					t.Error("request identity changed")
				}
				w.Header().Set("X-Request-ID", "11111111-1111-4111-8111-111111111111")
				w.Header().Set("Content-Type", "image/png")
				if mode == "duplicate-id" {
					w.Header().Add("X-Request-ID", "22222222-2222-4222-8222-222222222222")
				}
				switch mode {
				case "request-timeout":
					w.WriteHeader(408)
					return
				case "disconnect":
					connection, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error("fixture hijack failed")
						return
					}
					connection.Close()
					return
				case "redirect":
					w.Header().Set("Location", "/other")
					w.WriteHeader(302)
					return
				case "rejected":
					w.Header().Set("WWW-Authenticate", "Basic realm=fixture")
					w.WriteHeader(401)
					return
				case "throttled":
					w.WriteHeader(429)
					return
				case "headers":
					<-done
					return
				case "chunked":
					w.WriteHeader(200)
					w.(http.Flusher).Flush()
					w.Write(payload)
					return
				case "encoded":
					w.Header().Set("Content-Encoding", "gzip")
					w.Write(payload)
					return
				case "oversized":
					w.Header().Set("Content-Length", strconv.FormatInt(maxImageBytes+1, 10))
					w.WriteHeader(200)
					return
				case "truncated":
					w.Header().Set("Content-Length", strconv.Itoa(len(payload)+100))
					w.Write(payload)
					return
				case "invalid":
					w.Write([]byte("private-invalid-content"))
					return
				case "idle", "total":
					w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
					w.Write(payload[:32])
					w.(http.Flusher).Flush()
					<-done
					return
				}
				w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				w.Write(payload)
			}))
			if mode == "untrusted-tls" {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()
			defer close(done)
			previous := diagnosticNativeFactory
			var native *diagnosticWinTransport
			diagnosticNativeFactory = func(target string, result *contentDiagnostic, budgets diagnosticBudgets) (http.RoundTripper, func(), error) {
				transport, err := diagnosticNewWinTransport(target, "", "direct", result, budgets)
				if err != nil {
					return nil, nil, err
				}
				native = transport
				return transport, transport.close, nil
			}
			defer func() { diagnosticNativeFactory = previous }()
			budgets := diagnosticBudgets{total: 3 * time.Second, response: time.Second, idle: time.Second}
			if mode == "idle" {
				budgets.idle = 150 * time.Millisecond
			}
			if mode == "headers" {
				budgets.response = 150 * time.Millisecond
			}
			if mode == "total" {
				budgets.total = 150 * time.Millisecond
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			result := diagnoseContent(ctx, fixtureService(server), taskRecord{Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1}, 0, "winhttp", budgets)
			expectedCalls := int32(1)
			if mode == "cancelled" || mode == "untrusted-tls" {
				expectedCalls = 0
			}
			if calls.Load() != expectedCalls {
				t.Fatalf("native request count=%d expected=%d code=%s", calls.Load(), expectedCalls, result.LocalErrorCode)
			}
			wantSuccess := mode == "complete" || mode == "chunked" || mode == "duplicate-id"
			if result.OK != wantSuccess {
				t.Fatalf("unexpected native result: %+v", result)
			}
			if wantSuccess && (result.Transfer == nil || result.Transfer.BytesReceived != int64(len(payload)) || !result.Transfer.ReadComplete || !result.Transfer.FileVerified) {
				t.Fatal("native file not verified")
			}
			if mode == "duplicate-id" && result.RequestID != "" {
				t.Fatal("ambiguous request ID disclosed")
			}
			if mode == "idle" && result.LocalErrorCode != "media_download_stalled" {
				t.Fatalf("idle not reported: %+v", result)
			}
			if mode == "encoded" && (result.LocalErrorCode != "diagnostic_content_encoding_unsupported" || result.HTTPStatus != 200) {
				t.Fatal("encoded content misclassified")
			}
			if !wantSuccess && result.Transfer != nil && result.Transfer.FileVerified {
				t.Fatal("invalid native content verified")
			}
			if result.ElapsedMS > 4000 {
				t.Fatal("native request exceeded bounded cancellation")
			}
			encoded, _ := json.Marshal(result)
			for _, secret := range []string{"synthetic-fixture-token", server.URL, "private-invalid-content", "pic-fixture"} {
				if bytes.Contains(encoded, []byte(secret)) {
					t.Fatal("native receipt leaked input")
				}
			}
			if native != nil && native.request != nil {
				select {
				case <-native.request.state.closed:
				case <-time.After(time.Second):
					t.Fatal("native callback buffers not released")
				}
			}
		})
	}
}

func TestContentWinHTTPNativeBlocksReplayAndPOST(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(401) }))
	defer server.Close()
	for _, method := range []string{"GET", "POST"} {
		result := &contentDiagnostic{}
		transport, err := diagnosticNewWinTransport(server.URL, "", "direct", result, diagnosticBudgets{time.Second, time.Second, time.Second})
		if err != nil {
			t.Fatal("native policy unavailable")
		}
		defer transport.close()
		request, _ := http.NewRequest(method, server.URL, nil)
		request.Header.Set("Authorization", "Bearer synthetic-fixture-token")
		before := calls.Load()
		response, err := transport.RoundTrip(request)
		if response != nil {
			response.Body.Close()
		}
		if method == "POST" {
			if err == nil || calls.Load() != before {
				t.Fatal("POST allowed")
			}
			continue
		}
		if err != nil || calls.Load() != before+1 {
			t.Fatal("expected one GET")
		}
		if _, err := transport.RoundTrip(request); err == nil || calls.Load() != before+1 {
			t.Fatal("GET replay allowed")
		}
	}
}
