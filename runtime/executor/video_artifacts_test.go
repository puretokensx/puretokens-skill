package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestVideoTailFramePreservesIndependentDelivery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task.json")
	record := taskRecord{Format: taskRecordFormat, Kind: "video", Model: "Super Vega 1.5-pro", TaskID: "video-fixture", OriginalOperation: "generate", RequestedCount: 1, Status: "completed", OutputDir: dir, Parameters: map[string]any{"return_last_frame": true}}
	if err := saveTaskRecord(path, record, true); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	png := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		switch r.URL.RequestURI() {
		case "/v1/videos/video-fixture/content":
			w.Header().Set("Content-Type", "video/mp4")
			w.Write(structuralVideoBytes(128))
		case "/v1/tasks/video-fixture/artifacts":
			fmt.Fprint(w, `{"task_id":"video-fixture","artifacts":[{"key":"video","type":"video"},{"key":"last_frame","type":"image","content_url":"https://example.com/never-follow"}]}`)
		case "/v1/tasks/video-fixture/artifacts/last_frame/content":
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
		default:
			t.Errorf("unexpected request %s", r.URL.RequestURI())
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	svc := fixtureService(server)
	var out bytes.Buffer
	if err := executeRecordedTask("content", path, taskRequest{}, 1, dir, &out, svc); err != nil {
		t.Fatalf("tail content failed: %v (%s)", err, out.String())
	}
	result := decodeReceipt(t, &out)
	if len(calls) != 2 || len(result.DownloadedIndexes) != 1 || result.DownloadedIndexes[0] != 1 || result.NextStep != "deliver" {
		t.Fatalf("invalid tail receipt: %+v, %v", result, calls)
	}
	data, err := os.ReadFile(result.DownloadedPaths[0])
	if err != nil || !bytes.Equal(data, png) {
		t.Fatal("tail bytes changed")
	}
	out.Reset()
	if err := executeRecordedTask("delivered", path, taskRequest{}, 1, "", &out, svc); err != nil {
		t.Fatal(err)
	}
	saved, err := loadTaskRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.deliveryComplete() || len(saved.Delivered) != 1 || saved.Delivered[0] != 1 || saved.Parameters["return_last_frame"] != true {
		t.Fatal("tail handoff completed or corrupted video task")
	}
	out.Reset()
	if err := executeRecordedTask("redeliver", path, taskRequest{}, 1, "", &out, service{}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || decodeReceipt(t, &out).NextStep != "deliver" {
		t.Fatal("local tail redelivery used network or lost handoff")
	}
	out.Reset()
	if err := executeRecordedTask("delivered", path, taskRequest{}, 1, "", &out, service{}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := executeRecordedTask("content", path, taskRequest{}, 0, dir, &out, svc); err != nil {
		t.Fatal(err, out.String())
	}
	if len(calls) != 3 || decodeReceipt(t, &out).NextStep != "deliver" {
		t.Fatal("video identity lost after tail delivery")
	}
	out.Reset()
	if err := executeRecordedTask("delivered", path, taskRequest{}, 0, "", &out, service{}); err != nil {
		t.Fatal(err)
	}
	saved, err = loadTaskRecord(path)
	if err != nil || !saved.deliveryComplete() || saved.RequestedCount != 1 || decodeReceipt(t, &out).NextStep != "done" {
		t.Fatal("two-artifact completion was not recorded")
	}
}

func TestTailFrameRequiresOriginalSelectionAndExactArtifact(t *testing.T) {
	for _, tc := range []struct {
		name, model, body string
		flag              bool
	}{
		{"not-requested", "Super Vega 1.5-pro", "", false},
		{"wrong-model", "MiniMax-H3", "", true},
		{"wrong-task", "Super Vega 1.5-pro", `{"task_id":"other","artifacts":[{"key":"last_frame","type":"image"}]}`, true},
		{"missing", "Super Vega 1.5-pro", `{"task_id":"video-fixture","artifacts":[]}`, true},
		{"wrong-type", "Super Vega 1.5-pro", `{"task_id":"video-fixture","artifacts":[{"key":"last_frame","type":"video"}]}`, true},
		{"duplicate", "Super Vega 1.5-pro", `{"task_id":"video-fixture","artifacts":[{"key":"last_frame","type":"image"},{"key":"last_frame","type":"image"}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.RequestURI() != "/v1/tasks/video-fixture/artifacts" {
					t.Error("content fetched before artifact identity verification")
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			req := taskRequest{Kind: "video", Model: tc.model, Operation: "continue", TaskID: "video-fixture", RequestedCount: 1, TaskStatus: "completed", OutputDir: t.TempDir(), Index: 1, Parameters: map[string]any{"return_last_frame": tc.flag}}
			var out bytes.Buffer
			if err := deliverTask(&out, fixtureService(server), req, taskReceipt(req, req.TaskID, "completed")); err == nil {
				t.Fatal("invalid tail accepted")
			}
			if decodeReceipt(t, &out).NextStep != "await_user" {
				t.Fatal("failure did not stop")
			}
			expected := 1
			if tc.body == "" {
				expected = 0
			}
			if calls != expected {
				t.Fatalf("requests %d want %d", calls, expected)
			}
		})
	}
}
