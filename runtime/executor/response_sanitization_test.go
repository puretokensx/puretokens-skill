package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestSanitizeResponseJSONDecodesEscapedCredentials(t *testing.T) {
	token := "synthetic-fixture-token"
	body := []byte(`{"error":{"code":"content_policy_violation","message":"Content policy rejected synthetic\u002dfixture-token. Revise the prompt."},"synthetic\u002dfixture-token":{"nested":["synthetic-fixture-token",{"value":"synthetic\u002dfixture-token"}]}}`)
	sanitized := sanitizeResponseJSON(body, token)
	var got any
	if err := json.Unmarshal(sanitized, &got); err != nil {
		t.Fatal("sanitized response is not valid JSON", err)
	}
	want := map[string]any{
		"error": map[string]any{
			"code":    "content_policy_violation",
			"message": "Content policy rejected [redacted credential]. Revise the prompt.",
		},
		"[redacted credential]": map[string]any{
			"nested": []any{"[redacted credential]", map[string]any{"value": "[redacted credential]"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("decoded response strings or keys were not redacted")
	}
	code, message := publicAPIError(sanitized)
	if code != "content_policy_violation" || message != "The API rejected the content under its content policy. Revise the request." {
		t.Fatal("redaction discarded the public moderation reason")
	}
}

func TestSanitizeResponseJSONPreservesTypesAndNumbers(t *testing.T) {
	body := []byte(`{"integer":9007199254740993,"decimal":1.234567890123456789,"scientific":1e100,"boolean":false,"null":null,"array":[],"object":{},"text":"A public validation message."}`)
	for _, token := range []string{"", "synthetic-fixture-token"} {
		sanitized := sanitizeResponseJSON(body, token)
		decode := func(data []byte) any {
			t.Helper()
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.UseNumber()
			var result any
			if err := decoder.Decode(&result); err != nil {
				t.Fatal(err)
			}
			return result
		}
		if !reflect.DeepEqual(decode(body), decode(sanitized)) {
			t.Fatal("sanitization changed unrelated JSON values or number precision")
		}
	}
}

func TestSanitizeResponseJSONRejectsUnreadableInput(t *testing.T) {
	for _, body := range []string{
		"",
		`{"error":`,
		`{"error":{"message":"synthetic-fixture-token"}`,
		`{"error":"synthetic-fixture-token"} {"other":true}`,
		`{"error":"synthetic-fixture-token"} trailing`,
		`<html>synthetic-fixture-token</html>`,
		`{"synthetic-fixture-token":1,"[redacted credential]":2}`,
	} {
		if got := sanitizeResponseJSON([]byte(body), "synthetic-fixture-token"); len(got) != 0 {
			t.Fatal("unreadable response was not rejected")
		}
	}
}

func TestSanitizeResponseJSONHandlesEscapedTokenCharacters(t *testing.T) {
	token := "synthetic<&\"\\\nfixture"
	body, err := json.Marshal(map[string]any{
		token: []any{token, "Rejected " + token},
	})
	if err != nil {
		t.Fatal(err)
	}
	sanitized := sanitizeResponseJSON(body, token)
	var got map[string]any
	if json.Unmarshal(sanitized, &got) != nil {
		t.Fatal("escaped token prevented valid response sanitization")
	}
	if !reflect.DeepEqual(got["[redacted credential]"], []any{"[redacted credential]", "Rejected [redacted credential]"}) {
		t.Fatal("escaped token characters bypassed redaction")
	}
}

func TestResponseSanitizationProtectsAPIErrorReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":{"code":"content_policy_violation","message":"Content policy rejected synthetic\u002dfixture-token. Revise the prompt."}}`)
	}))
	defer server.Close()
	var output bytes.Buffer
	if executeReadOnly(&output, fixtureService(server), "/v1/media/models", "models") == nil {
		t.Fatal("API rejection did not return failure")
	}
	result := decodeReceipt(t, &output)
	if result.ErrorMessage != "The API rejected the content under its content policy. Revise the request." || result.APIErrorCode != "content_policy_violation" {
		t.Fatal("API error receipt exposed a credential or lost the moderation reason")
	}
}

func TestResponseSanitizationProtectsProjectedCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[{"id":"gpt-image-2","capabilities":["image"],"input_schema":{"properties":{"synthetic\u002dfixture-token":{"type":"string"},"prompt":{"type":"string","description":"Do not include synthetic\u002dfixture-token","examples":["synthetic\u002dfixture-token"]}},"constraints":{"nested":{"value":"synthetic\u002dfixture-token"}}}}]}`)
	}))
	defer server.Close()
	var output bytes.Buffer
	if err := executeReadOnly(&output, fixtureService(server), "/v1/media/models", "models"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "synthetic-fixture-token") || strings.Contains(output.String(), `synthetic\u002dfixture-token`) {
		t.Fatal("projected catalog leaked a credential in a nested string or schema key")
	}
	if !strings.Contains(output.String(), "gpt-image-2") || !strings.Contains(output.String(), `"prompt":{"type":"string"}`) {
		t.Fatal("catalog projection discarded safe model data")
	}
}

func TestResponseSanitizationProtectsDownloadErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"error":{"message":"Rejected synthetic\u002dfixture-token"}}`)
	}))
	defer server.Close()
	_, _, _, _, message, err := fixtureService(server).download(context.Background(), "/v1/images/task/content", "image", t.TempDir())
	if err == nil || message != "" {
		t.Fatal("unrecognized server error text was forwarded")
	}
}
