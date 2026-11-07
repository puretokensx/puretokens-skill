package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSVGDownloadAndRecordedRecovery(t *testing.T) {
	dir := t.TempDir()
	recordPath := filepath.Join(dir, "task.json")
	record := taskRecord{Format: taskRecordFormat, Kind: "image", Model: "gpt-image-2.6", TaskID: "svg-fixture", OriginalOperation: "generate", RequestedCount: 1, Status: "completed", Parameters: map[string]any{"output_format": "svg"}, OutputDir: dir}
	if err := saveTaskRecord(recordPath, record, true); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.RequestURI() != "/v1/images/svg-fixture/content?index=0" {
			t.Error("unexpected SVG request")
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(staticSVGFixture))
	}))
	defer server.Close()
	var out bytes.Buffer
	if err := executeRecordedTask("content", recordPath, taskRequest{}, 0, dir, &out, fixtureService(server)); err != nil {
		t.Fatal(err, out.String())
	}
	result := decodeReceipt(t, &out)
	if result.NextStep != "deliver" || len(result.DownloadedPaths) != 1 || filepath.Ext(result.DownloadedPaths[0]) != ".svg" {
		t.Fatal(result)
	}
	data, err := os.ReadFile(result.DownloadedPaths[0])
	if err != nil || string(data) != staticSVGFixture {
		t.Fatal("SVG bytes changed")
	}
	out.Reset()
	if err := executeRecordedTask("delivered", recordPath, taskRequest{}, 0, "", &out, service{}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := executeRecordedTask("redeliver", recordPath, taskRequest{}, 0, "", &out, service{}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || decodeReceipt(t, &out).NextStep != "deliver" {
		t.Fatal("SVG recovery did not use verified local original")
	}
}

func TestSVGDownloadRejectsUnreviewedOrPartialContent(t *testing.T) {
	for _, tc := range []struct {
		name, model, format, contentRange, body string
		status                                  int
	}{
		{"unsolicited", "gpt-image-2.6", "", "", staticSVGFixture, 200},
		{"wrong-model", "gpt-image-2", "svg", "", staticSVGFixture, 200},
		{"partial", "gpt-image-2.6", "svg", "", staticSVGFixture, 206},
		{"range", "gpt-image-2.6", "svg", "bytes 0-10/100", staticSVGFixture, 200},
		{"unsafe", "gpt-image-2.6", "svg", "", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/svg+xml")
				if tc.contentRange != "" {
					w.Header().Set("Content-Range", tc.contentRange)
				}
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			dir := t.TempDir()
			request := taskRequest{Kind: "image", Model: tc.model, TaskID: "svg-fixture", RequestedCount: 1, Parameters: map[string]any{"output_format": tc.format}, OutputDir: dir}
			var out bytes.Buffer
			if err := deliverTask(&out, fixtureService(server), request, taskReceipt(request, request.TaskID, "completed")); err == nil {
				t.Fatal("invalid SVG accepted")
			}
			files, _ := os.ReadDir(dir)
			if len(files) != 0 || decodeReceipt(t, &out).NextStep != "await_user" {
				t.Fatal("unsafe SVG finalized or retried")
			}
		})
	}
}
