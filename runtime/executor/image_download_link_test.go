package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImageDownloadLinkIsOneReadNotDelivery(t *testing.T) {
	link := apiOrigin + "/v1/tasks/image-existing/artifacts/image-0/content?access=v2." + base64.RawURLEncoding.EncodeToString(make([]byte, 44))
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.RequestURI() != "/v1/images/image-existing/download-link?index=0" || r.Header.Get("Authorization") != "Bearer synthetic-fixture-token" {
			t.Errorf("unexpected link request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"task_id": "image-existing", "index": 0, "url": link, "expires_at": time.Now().Add(9 * time.Minute).UTC().Format(time.RFC3339)})
	}))
	defer server.Close()
	record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "image-existing", Status: "completed", RequestedCount: 1, SubmissionOutcome: "accepted"}
	path := filepath.Join(t.TempDir(), "task.json")
	if err := saveTaskRecord(path, record, true); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	var output bytes.Buffer
	writer := &supportReceiptWriter{output: &output, command: "download-link", host: "codex"}
	svc := fixtureService(server)
	svc.support = writer
	err := executeRecordedTask("download-link", path, record.request(), 0, "", writer, svc)
	if err != nil {
		t.Fatalf("link failed: %v: %s", err, output.String())
	}
	var result map[string]any
	json.Unmarshal(output.Bytes(), &result)
	if calls != 1 || result["ok"] != true || result["next_step"] != "present_link" || result["delivery_status"] != "link_available" {
		t.Fatalf("invalid link receipt: %s calls=%d", output.String(), calls)
	}
	if result["downloaded_paths"] != nil || result["delivered_indexes"] != nil {
		t.Fatal("link claimed attachment delivery")
	}
	support := jsonObject(result["support"])
	if support["command"] != "download-link" || support["next_step"] != "present_link" || support["request_phase"] != "content" {
		t.Fatalf("link support metadata missing: %v", support)
	}
	encodedSupport, _ := json.Marshal(support)
	if strings.Contains(string(encodedSupport), "access") || strings.Contains(string(encodedSupport), "url") {
		t.Fatal("support leaked capability")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) || strings.Contains(string(after), "access=") {
		t.Fatal("link changed original task evidence")
	}
}

func TestImageDownloadLinkRejectsUnsafeResponsesWithoutFollowing(t *testing.T) {
	good := apiOrigin + "/v1/tasks/image-existing/artifacts/image-0/content?access=v2." + base64.RawURLEncoding.EncodeToString(make([]byte, 44))
	for _, tc := range []struct {
		name, url, task, expiry string
		index, status           int
	}{
		{"other-origin", strings.Replace(good, "api.puretokensx.com", "private.invalid", 1), "image-existing", "", 0, 200},
		{"other-task", strings.Replace(good, "image-existing", "other-task", 1), "image-existing", "", 0, 200},
		{"extra-query", good + "&key=private", "image-existing", "", 0, 200},
		{"fragment", good + "#private", "image-existing", "", 0, 200},
		{"wrong-index", good, "image-existing", "", 1, 200},
		{"wrong-identity", good, "other-task", "", 0, 200},
		{"expired", good, "image-existing", "2000-01-01T00:00:00Z", 0, 200},
		{"far-expiry", good, "image-existing", "2099-01-01T00:00:00Z", 0, 200},
		{"denied", good, "image-existing", "", 0, 403},
		{"throttled", good, "image-existing", "", 0, 429},
		{"redirect", good, "image-existing", "", 0, 302},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/never-follow")
				w.WriteHeader(tc.status)
				expires := tc.expiry
				if expires == "" {
					expires = time.Now().Add(9 * time.Minute).UTC().Format(time.RFC3339)
				}
				json.NewEncoder(w).Encode(map[string]any{"task_id": tc.task, "index": tc.index, "url": tc.url, "expires_at": expires})
			}))
			defer server.Close()
			var output bytes.Buffer
			err := executeExistingTask("download-link", strings.NewReader(`{"kind":"image","operation":"continue","task_id":"image-existing","task_status":"completed","requested_count":1}`), &output, fixtureService(server))
			result := decodeReceipt(t, &output)
			if err == nil || result.OK || result.NextStep != "await_user" || calls != 1 || strings.Contains(output.String(), "access=") {
				t.Fatalf("unsafe response escaped: %s calls=%d", output.String(), calls)
			}
		})
	}
}

func TestImageDownloadLinkRejectsInvalidTaskBeforeNetwork(t *testing.T) {
	for _, body := range []string{
		`{"kind":"video","operation":"continue","task_id":"original","task_status":"completed","requested_count":1}`,
		`{"kind":"image","operation":"continue","task_id":"original","task_status":"pending","requested_count":1}`,
		`{"kind":"image","operation":"continue","task_id":"original","task_status":"completed","requested_count":1,"reconciliation_required":true}`,
		`{"kind":"image","operation":"continue","task_id":"original","task_status":"completed","requested_count":1,"index":1}`,
		`{"kind":"image","operation":"continue","task_id":"original","task_status":"completed"}`,
	} {
		var out bytes.Buffer
		svc := service{client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid request reached network")
			return nil, nil
		})}}
		if executeExistingTask("download-link", strings.NewReader(body), &out, svc) == nil || decodeReceipt(t, &out).NextStep != "await_user" {
			t.Fatalf("invalid link request accepted: %s", out.String())
		}
	}
}

func TestImageDownloadLinkRejectsLogicalAndDuplicateResponses(t *testing.T) {
	base := map[string]any{"task_id": "image-existing", "index": 0, "url": apiOrigin + "/v1/tasks/image-existing/artifacts/image-0/content?access=v2." + base64.RawURLEncoding.EncodeToString(make([]byte, 44)), "expires_at": time.Now().Add(9 * time.Minute).UTC().Format(time.RFC3339)}
	good, _ := json.Marshal(base)
	for _, suffix := range []string{`,"error":{"code":"permission_denied"}}`, `,"success":false}`, `,"index":0}`} {
		body := string(good[:len(good)-1]) + suffix
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		var output bytes.Buffer
		err := executeExistingTask("download-link", strings.NewReader(`{"kind":"image","operation":"continue","task_id":"image-existing","task_status":"completed","requested_count":1}`), &output, fixtureService(server))
		server.Close()
		if err == nil || strings.Contains(output.String(), "access=") {
			t.Fatal("logical/ambiguous link response accepted")
		}
	}
}
