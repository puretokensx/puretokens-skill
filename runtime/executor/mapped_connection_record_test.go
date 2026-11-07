package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMappedTaskStopsBeforeUsingChangedConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.json")
	record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "fixture-task", Status: "completed", RequestedCount: 1, ConnectionID: "11111111-1111-4111-8111-111111111111"}
	if err := saveTaskRecord(path, record, true); err != nil {
		t.Fatal(err)
	}
	for _, current := range []string{"22222222-2222-4222-8222-222222222222", ""} {
		for _, command := range []string{"status", "wait", "resume", "content", "redeliver"} {
			var output bytes.Buffer
			calls := 0
			svc := service{host: "claude-desktop", connectionID: current, token: "synthetic-unselected", client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) { calls++; return evaluationHTTP(500, `{}`), nil })}}
			if err := executeRecordedTask(command, path, taskRequest{}, 0, t.TempDir(), &output, svc); err == nil {
				t.Fatal("changed connection accepted")
			}
			if !bytes.Contains(output.Bytes(), []byte("task_connection_changed")) || calls != 0 {
				t.Fatal("connection change was not identified before networking")
			}
			saved, err := loadTaskRecord(path)
			if err != nil || saved.ConnectionID != record.ConnectionID || saved.TaskID != record.TaskID {
				t.Fatal("existing task identity changed")
			}
		}
	}
}

func TestMappedTaskLocalHandoffSurvivesConnectionChange(t *testing.T) {
	dir := t.TempDir()
	id := "mapped-local-task"
	file := filepath.Join(dir, fmt.Sprintf("puretokens-%x.png", sha256.Sum256([]byte(contentPath("image", id, 0)))))
	if err := os.WriteFile(file, fixturePNG(t), 0600); err != nil {
		t.Fatal(err)
	}
	proof, err := fingerprintDownload(file, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "task.json")
	record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: id, Status: "completed", RequestedCount: 1,
		ConnectionID: "11111111-1111-4111-8111-111111111111", OutputDir: dir, Downloaded: map[int]string{0: file}, DownloadProofs: map[int]downloadProof{0: proof}}
	if err := saveTaskRecord(path, record, true); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"resume", "redeliver", "delivered"} {
		var out bytes.Buffer
		if err := run([]string{command, "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out); err != nil {
			t.Fatalf("%s tried to resolve a credential for local handoff", command)
		}
		got := decodeReceipt(t, &out)
		want := "deliver"
		if command == "delivered" {
			want = "done"
		}
		if got.NextStep != want {
			t.Fatalf("%s lost local delivery", command)
		}
	}
}
