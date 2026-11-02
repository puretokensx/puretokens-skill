package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type transportDiagnosticCase struct {
	name, code string
	err        error
}

func transportDiagnosticCases() []transportDiagnosticCase {
	return []transportDiagnosticCase{
		{"timeout", "api_request_timeout", context.DeadlineExceeded},
		{"cancelled", "api_request_cancelled", context.Canceled},
		{"dns", "api_dns_failed", &net.DNSError{Name: "private-host", Server: "private-server", Err: "private-error"}},
		{"dns-timeout", "api_dns_failed", &net.DNSError{Name: "private-host", IsTimeout: true}},
		{"tls", "api_tls_failed", &tls.CertificateVerificationError{Err: errors.New("private-cert")}},
		{"tls-record", "api_tls_failed", tls.RecordHeaderError{Msg: "private-record"}},
		{"permission", "api_network_permission_denied", &net.OpError{Op: "dial", Err: os.ErrPermission}},
		{"reset", "api_connection_failed", &net.OpError{Op: "read", Err: syscall.ECONNRESET}},
		{"refused", "api_connection_failed", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}},
		{"closed", "api_connection_failed", io.EOF},
		{"unclassified", "api_transport_failed", errors.New("private-dns-tls-timeout")},
	}
}

func TestSubmissionTransportDiagnostics(t *testing.T) {
	for _, kind := range []string{"image", "video", "music"} {
		for _, tc := range transportDiagnosticCases() {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				var output bytes.Buffer
				writer := &supportReceiptWriter{output: &output, command: "submit", host: "codex", timings: newCommandTimings()}
				calls := 0
				svc := service{baseURL: apiOrigin, support: writer, client: &http.Client{Transport: pollingFixtureTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != http.MethodPost {
						t.Error("unexpected follow-up read")
					}
					return nil, &url.Error{Op: "Post", URL: "https://private-host/private-path", Err: tc.err}
				})}}
				recordPath := filepath.Join(t.TempDir(), "task.json")
				request := taskRequest{Kind: kind, Operation: "generate", Model: "fixture", Prompt: "fixture", RequestedCount: 1}
				if kind == "music" {
					request.OutputDir = t.TempDir()
					request.Model = "stepaudio-3-music-preview"
					request.Parameters = map[string]any{"instrumental": true, "response_format": "wav"}
				}
				err := executeRecordedTask("submit", recordPath, request, 0, "", writer, svc)
				got := decodeReceipt(t, &output)
				if err == nil || calls != 1 || got.OK || got.TaskID != "" || got.SubmissionOutcome != "unknown" || got.NextStep != "await_user" {
					t.Fatalf("changed unknown-submission boundary: %+v calls=%d", got, calls)
				}
				if got.LocalErrorCode != tc.code || got.HTTPStatus != 0 || got.APIErrorCode != "" || got.ErrorMessage == "The Pure Tokens API request did not complete." {
					t.Fatalf("want safe transport category %s, got %+v", tc.code, got)
				}
				var envelope map[string]any
				if json.Unmarshal(output.Bytes(), &envelope) != nil {
					t.Fatal("invalid receipt")
				}
				support := jsonObject(envelope["support"])
				if support["transport_error_code"] != tc.code || support["http_status"] != nil || support["request_id"] != nil {
					t.Fatalf("incorrect support evidence: %v", support)
				}
				if jsonObject(envelope["timings_ms"])["request"] == nil || support["timings_ms"] != nil {
					t.Fatal("request timing lost or persisted in support")
				}
				record, loadErr := loadTaskRecord(recordPath)
				if loadErr != nil || record.TaskID != "" || record.SubmissionOutcome != "unknown" {
					t.Fatal("unknown task record lost")
				}
				raw, readErr := os.ReadFile(recordPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if strings.Contains(output.String(), "private-") || strings.Contains(string(raw), "api_request_timeout") || strings.Contains(string(raw), "transport_error_code") || strings.Contains(string(raw), "timings_ms") {
					t.Fatal("private error or invocation diagnostics leaked into output/record")
				}
			})
		}
	}
}

func TestTransportSupportResetsBeforeNextAttempt(t *testing.T) {
	var output bytes.Buffer
	writer := &supportReceiptWriter{output: &output, command: "status", host: "codex"}
	calls := 0
	svc := service{baseURL: apiOrigin, support: writer, client: &http.Client{Transport: pollingFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, context.DeadlineExceeded
		}
		return &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"service_unavailable"}}`))}, nil
	})}}
	svc.request(context.Background(), http.MethodGet, "/v1/videos/original-task", nil, "")
	writeReceipt(writer, receipt{OK: false})
	var first map[string]any
	if json.Unmarshal(output.Bytes(), &first) != nil || jsonObject(first["support"])["transport_error_code"] != "api_request_timeout" {
		t.Fatal("first attempt was not classified")
	}
	output.Reset()
	svc.request(context.Background(), http.MethodGet, "/v1/videos/original-task", nil, "")
	writeReceipt(writer, receipt{OK: false})
	var second map[string]any
	if json.Unmarshal(output.Bytes(), &second) != nil {
		t.Fatal("invalid second receipt")
	}
	summary := jsonObject(second["support"])
	if summary["transport_error_code"] != nil || summary["http_status"] != float64(503) || calls != 2 {
		t.Fatal("stale transport failure masked HTTP error")
	}
}

func TestTransportDiagnosticsPreserveExistingTask(t *testing.T) {
	for _, command := range []string{"status", "wait"} {
		for _, scenario := range transportDiagnosticCases() {
			t.Run(command+"/"+scenario.name, func(t *testing.T) {
				var output bytes.Buffer
				writer := &supportReceiptWriter{output: &output, command: command, host: "codex"}
				calls := 0
				svc := service{baseURL: apiOrigin, support: writer, wait: func(context.Context, time.Duration) bool { return true }, client: &http.Client{Transport: pollingFixtureTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != http.MethodGet || r.URL.Path != "/v1/videos/original-task" {
						t.Error("recovery changed task or issued POST")
					}
					return nil, scenario.err
				})}}
				input := `{"kind":"video","task_id":"original-task","model":"fixture","original_operation":"generate","task_status":"pending","wait_windows_completed":1}`
				err := executeExistingTask(command, strings.NewReader(input), writer, svc)
				got := decodeReceipt(t, &output)
				if err == nil || calls != 1 || got.TaskID != "original-task" || got.SubmissionOutcome != "" || got.WaitWindowsCompleted != 1 || got.NextStep != "await_user" || got.LocalErrorCode != scenario.code || !strings.Contains(got.NextAction, "Keep this task ID") {
					t.Fatalf("transport failure broke same-task recovery: %+v calls=%d", got, calls)
				}
			})
		}
	}
}

func TestSubmissionRealClientDeadlineIsNotBodyFailure(t *testing.T) {
	for _, sendHeaders := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-headers", true: "after-headers"}[sendHeaders], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				if sendHeaders {
					w.Header().Set("X-Request-ID", "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99")
					w.WriteHeader(200)
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			var output bytes.Buffer
			writer := &supportReceiptWriter{output: &output, command: "submit", host: "codex"}
			svc := fixtureService(server)
			svc.client.Timeout = 200 * time.Millisecond
			svc.support = writer
			err := executePreparedTask(writer, svc, taskRequest{Kind: "video", Operation: "generate", Model: "fixture", Prompt: "fixture"})
			got := decodeReceipt(t, &output)
			if err == nil || got.SubmissionOutcome != "unknown" {
				t.Fatal("deadline lost uncertain submission")
			}
			var envelope map[string]any
			if json.Unmarshal(output.Bytes(), &envelope) != nil {
				t.Fatal("invalid receipt")
			}
			support := jsonObject(envelope["support"])
			if sendHeaders {
				if got.LocalErrorCode != "task_response_unreadable" || got.HTTPStatus != 200 || support["transport_error_code"] != nil || support["request_id"] == nil {
					t.Fatal("body timeout erased observed HTTP response")
				}
			} else if got.LocalErrorCode != "api_request_timeout" || got.HTTPStatus != 0 || support["transport_error_code"] != "api_request_timeout" {
				t.Fatalf("real client deadline was not classified: %+v", got)
			}
		})
	}
}
