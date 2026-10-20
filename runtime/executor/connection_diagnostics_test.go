package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertLocalConnectionDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("unrecognized connection accepted")
	}
	result := initCredentialFailure(err)
	if result.OK || result.APIRequestExecuted || result.CredentialVerified || result.APIIdentityConfirmed {
		t.Fatal("local rejection claimed verification")
	}
	if !strings.Contains(result.NextAction, "Keep existing connections") {
		t.Fatal("local rejection failed to preserve existing connections")
	}
	for _, required := range []string{"Stop this automatic API flow", "do not try another --host", "as a recovery probe", "any existing task ID", "never automatically resubmit", "explicit user selection"} {
		if !strings.Contains(result.NextAction, required) {
			t.Fatalf("local failure omitted recovery boundary: %s", required)
		}
	}
	encoded, _ := json.Marshal(result)
	for _, private := range []string{"example.invalid", "synthetic-secret", "private-config"} {
		if strings.Contains(string(encoded), private) {
			t.Fatal("diagnostic exposed private source data")
		}
	}
}

func TestConnectionDiagnosticsAcrossFileAdapters(t *testing.T) {
	cases := []struct {
		host, body string
		read       func(string) (string, error)
	}{
		{"codex", "model_provider='custom'\n[model_providers.custom]\nbase_url='https://example.invalid/v1'\nexperimental_bearer_token='synthetic-secret'\n", credentialFromCodexFile},
		{"claude-code", `{"env":{"ANTHROPIC_BASE_URL":"https://example.invalid","ANTHROPIC_AUTH_TOKEN":"synthetic-secret"}}`, credentialFromClaudeCodeFile},
		{"gemini-cli", "GOOGLE_GEMINI_BASE_URL=https://example.invalid\nGEMINI_API_KEY=synthetic-secret\n", credentialFromGeminiEnvFile},
		{"grok-build", "[models]\ndefault='custom'\n[model.custom]\nbase_url='https://example.invalid/v1'\napi_key='synthetic-secret'\n", credentialFromGrokBuildFile},
		{"opencode", `{"model":"custom/model","provider":{"custom":{"options":{"baseURL":"https://example.invalid/v1","apiKey":"synthetic-secret"}}}}`, credentialFromOpenCodeFile},
		{"workbuddy", `[{"url":"https://example.invalid/v1/chat/completions","apiKey":"synthetic-secret"}]`, credentialFromWorkBuddyFile},
		{"kimi-code", "default_model='custom'\n[models.custom]\nprovider='custom'\n[providers.custom]\ntype='openai'\nbase_url='https://example.invalid/v1'\napi_key='synthetic-secret'\n", credentialFromKimiFile},
		{"qoder", `{"providers":{"custom":{"type":"openai-compatible","protocol":"openai","baseUrl":"https://example.invalid/v1","apiKey":"synthetic-secret"}}}`, credentialFromQoderFile},
		{"zcode", `{"provider":{"custom":{"enabled":true,"kind":"openai-compatible","options":{"baseURL":"https://example.invalid/v1","apiKey":"synthetic-secret"}}}}`, credentialFromZCodeFile},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, "private-config")
			_, err := tc.read(file)
			assertLocalConnectionDiagnostic(t, err)
			desktopFixtureFile(t, root, "private-config", tc.body)
			token, err := tc.read(file)
			if token != "" {
				t.Fatal("unrecognized endpoint released credential")
			}
			assertLocalConnectionDiagnostic(t, err)
		})
	}
}

func TestDesktopAndUnsupportedHostDiagnostics(t *testing.T) {
	for _, read := range []func(string) (string, error){credentialFromClaudeDesktopRoot, credentialFromDSHDesktopRoot} {
		_, err := read(t.TempDir())
		assertLocalConnectionDiagnostic(t, err)
	}
	assertLocalConnectionDiagnostic(t, desktopSelectionFailure())
	_, err := credentialForHost("unsupported-fixture-host")
	assertLocalConnectionDiagnostic(t, err)
	result := initCredentialFailure(err)
	if result.ConfigurationStatus != "host_credential_adapter_unavailable" || result.APIRequestExecuted {
		t.Fatal("unsupported adapter misreported")
	}
}

func TestQoderDiagnosticDistinctions(t *testing.T) {
	cases := []struct{ body, status string }{
		{`{}`, "active_connection_format_unsupported"},
		{`{"providers":{}}`, "active_connection_selection_unconfirmed"},
		{`{"providers":{"a":{}}}`, "active_connection_endpoint_missing"},
		{`{"providers":{"a":{"baseUrl":"https://api.puretokensx.com/unsupported"}}}`, "active_connection_endpoint_unsupported"},
		{`{"providers":{"a":{"baseUrl":"https://example.invalid/v1"}}}`, "active_connection_not_puretokens"},
		{`{"providers":{"a":{"baseUrl":"https://api.puretokensx.com/v1","type":"other"}}}`, "active_connection_format_unsupported"},
		{`{"providers":{"a":{"baseUrl":"https://api.puretokensx.com/v1","type":"openai-compatible","protocol":"openai"}}}`, "active_connection_credential_missing"},
		{`{"providers":{"a":{"baseUrl":"https://api.puretokensx.com/v1"},"b":{"baseUrl":"https://api.puretokensx.com/v1"}}}`, "active_connection_ambiguous"},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			root := t.TempDir()
			desktopFixtureFile(t, root, "record", tc.body)
			_, err := credentialFromQoderFile(filepath.Join(root, "record"))
			assertLocalConnectionDiagnostic(t, err)
			status, _, _ := credentialFailureDetails(err)
			if status != tc.status {
				t.Fatalf("got %s, want %s", status, tc.status)
			}
		})
	}
}

func TestRawConnectionErrorsStaySanitized(t *testing.T) {
	assertLocalConnectionDiagnostic(t, &os.PathError{Op: "open", Path: "/private-config", Err: os.ErrNotExist})
	assertLocalConnectionDiagnostic(t, errors.New("private-config synthetic-secret"))
}
