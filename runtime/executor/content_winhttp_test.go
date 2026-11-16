//go:build content_diagnostic

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestContentDiagnosticWinHTTPUnavailableBeforeCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has the native transport")
	}
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "absent"))
	path := writeFixture(t, "completed-status.json", diagnosticStatusFixture(t))
	var output bytes.Buffer
	code := runContentDiagnostic([]string{"--host", "codex", "--status-receipt", path, "--task-id", "pic-fixture", "--probe", "winhttp", "--confirm-one-get"}, &output)
	if code == 0 || !strings.Contains(output.String(), "diagnostic_transport_unsupported") || !strings.Contains(output.String(), `"requests":0`) {
		t.Fatalf("unavailable native mode did not stop before credentials: %s", &output)
	}
}

func TestContentDiagnosticNativeProxyPolicy(t *testing.T) {
	for _, tc := range []struct {
		raw, want string
		allowed   bool
	}{
		{"", "direct", true},
		{"http://127.0.0.1:7890", "environment_http_proxy", true},
		{"http://proxy.example", "environment_http_proxy", true},
		{"https://proxy.example:443", "", false},
		{"socks5://127.0.0.1:1080", "", false},
		{"http://private-user:private-password@proxy.example", "", false},
		{"http://proxy.example/path", "", false},
		{"http://proxy.example/?private=value", "", false},
		{"http://proxy.example/#fragment", "", false},
		{"http://[::1]:7890", "environment_http_proxy", true},
	} {
		var proxy *url.URL
		if tc.raw != "" {
			var err error
			proxy, err = url.Parse(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
		}
		address, mode, err := diagnosticNativeProxy(proxy)
		if (err == nil) != tc.allowed || mode != tc.want {
			t.Fatalf("proxy policy mismatch for case %q", tc.want)
		}
		if !tc.allowed && (address != "" || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "example")) {
			t.Fatal("unsupported proxy leaked details")
		}
		if tc.allowed && proxy != nil && address != proxy.Host {
			t.Fatal("proxy destination changed")
		}
	}
}

// This tests selection and validation at the native factory boundary, not
// Windows acceptance. Native DLL tests live in the Windows-only test file.
func TestContentDiagnosticWinHTTPResponseErrors(t *testing.T) {
	previous := diagnosticNativeFactory
	t.Cleanup(func() { diagnosticNativeFactory = previous })
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"response-deadline", errContentResponseTimeout, "media_content_response_timeout"},
		{"cancellation", context.Canceled, "api_request_cancelled"},
		{"unclassified", errors.New("private-native-error"), "api_transport_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diagnosticNativeFactory = func(target string, result *contentDiagnostic, budgets diagnosticBudgets) (http.RoundTripper, func(), error) {
				return evaluationTransport(func(r *http.Request) (*http.Response, error) {
					result.Requests++
					return nil, tc.err
				}), func() {}, nil
			}
			result := diagnoseContent(context.Background(), service{baseURL: apiOrigin, client: &http.Client{}, token: "synthetic-fixture-token"},
				taskRecord{Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1},
				0, "winhttp", diagnosticBudgets{time.Second, time.Second, time.Second})
			if result.OK || result.Requests != 1 || result.HTTPStatus != 0 || result.Transfer != nil || result.LocalErrorCode != tc.want {
				t.Fatalf("native response error misclassified: %+v", result)
			}
			encoded, _ := json.Marshal(result)
			for _, forbidden := range []string{"private-native-error", "synthetic-fixture-token", apiOrigin} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatal("native error receipt exposed private details")
				}
			}
		})
	}
}

func TestContentDiagnosticWinHTTPUsesIndependentTransport(t *testing.T) {
	payload := fixturePNG(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer synthetic-fixture-token" {
			t.Error("wrong request identity")
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(payload)
	}))
	defer server.Close()
	fixtureTransport := http.DefaultTransport.(*http.Transport).Clone()
	fixtureTransport.Proxy = nil
	defer fixtureTransport.CloseIdleConnections()
	previous := diagnosticNativeFactory
	t.Cleanup(func() { diagnosticNativeFactory = previous })
	diagnosticNativeFactory = func(target string, result *contentDiagnostic, budgets diagnosticBudgets) (http.RoundTripper, func(), error) {
		return evaluationTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != target || result.Requests != 0 {
				t.Fatal("native request unbound or replayed")
			}
			result.Requests++
			return fixtureTransport.RoundTrip(r)
		}), func() {}, nil
	}
	svc := fixtureService(server)
	svc.client.Transport = evaluationTransport(func(r *http.Request) (*http.Response, error) {
		t.Fatal("native mode used Go transport")
		return nil, nil
	})
	record := taskRecord{Kind: "image", TaskID: "pic-fixture", Status: "completed", RequestedCount: 1}
	result := diagnoseContent(context.Background(), svc, record, 0, "winhttp", diagnosticBudgets{time.Second, time.Second, time.Second})
	if !result.OK || calls != 1 || result.Requests != 1 || result.Transfer == nil || !result.Transfer.FileVerified {
		t.Fatalf("native selection failed: %+v", result)
	}
}
