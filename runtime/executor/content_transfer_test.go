package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestContentTransferKnownAndTruncatedBodies(t *testing.T) {
	for _, truncated := range []bool{false, true} {
		t.Run(strconv.FormatBool(truncated), func(t *testing.T) {
			body := fixturePNG(t)
			expected := len(body)
			if truncated {
				expected += 32
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", strconv.Itoa(expected))
				w.Write(body)
			}))
			defer server.Close()
			var output bytes.Buffer
			writer := &supportReceiptWriter{output: &output, command: "content", host: "codex"}
			svc := fixtureService(server)
			svc.support = writer
			dir := t.TempDir()
			request := taskRequest{Kind: "image", TaskID: "fixture-transfer", TaskStatus: "completed", RequestedCount: 1, OutputDir: dir}
			recordPath := filepath.Join(t.TempDir(), "task.json")
			if err := saveTaskRecord(recordPath, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: request.TaskID, Status: "completed", RequestedCount: 1, SubmissionOutcome: "accepted"}, true); err != nil {
				t.Fatal(err)
			}
			err := executeRecordedTask("content", recordPath, request, 0, dir, writer, svc)
			if (err != nil) != truncated || calls != 1 {
				t.Fatalf("outcome=%v calls=%d", err, calls)
			}
			transfer := writer.contentTransfer
			if transfer == nil || transfer.BytesReceived != int64(len(body)) || transfer.ExpectedBytes == nil || *transfer.ExpectedBytes != int64(expected) || transfer.ReadComplete == truncated || transfer.FileVerified == truncated {
				t.Fatalf("incorrect body evidence: %+v", transfer)
			}
			if transfer.FirstByteMS == nil || transfer.LastProgressMS == nil || *transfer.LastProgressMS < *transfer.FirstByteMS {
				t.Fatal("missing progress evidence")
			}
			if truncated {
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 0 {
					t.Fatal("truncated file retained")
				}
			}
			var document map[string]any
			if err := json.Unmarshal(output.Bytes(), &document); err != nil {
				t.Fatal(err)
			}
			if document["support"].(map[string]any)["content_transfer"] != nil {
				t.Fatal("exported transfer evidence")
			}
			var raw map[string]json.RawMessage
			json.Unmarshal(output.Bytes(), &raw)
			delete(raw, "content_transfer")
			if writer.operationsEvent != operationsEvent(writer.command, writer.attempted, raw) {
				t.Fatal("diagnostics changed operations event")
			}
			saved, err := os.ReadFile(recordPath)
			if err != nil || bytes.Contains(saved, []byte("content_transfer")) || bytes.Contains(saved, []byte("bytes_received")) {
				t.Fatal("diagnostics entered task record")
			}
			writer.beginRequest(http.MethodGet, "/v1/models")
			if writer.contentTransfer != nil {
				t.Fatal("transfer leaked into next request")
			}
		})
	}
}

func TestContentTransferReadSemantics(t *testing.T) {
	for _, response := range []*http.Response{{ContentLength: -1}, {ContentLength: 123, Uncompressed: true}} {
		if transfer := newContentTransfer(response); transfer.ExpectedBytes != nil {
			t.Fatal("invented comparable wire length")
		}
	}
	transfer := newContentTransfer(&http.Response{ContentLength: 0})
	transfer.observeRead(0, nil, 1500*time.Microsecond)
	transfer.observeRead(0, context.Canceled, 1500*time.Microsecond)
	if transfer.ReadWaitMS != 3 || transfer.ReadComplete || transfer.FirstByteMS != nil || transfer.LastProgressMS != nil {
		t.Fatal("incorrect zero-byte evidence")
	}
	transfer.observeRead(3, io.EOF, time.Millisecond)
	if !transfer.ReadComplete || transfer.FileVerified || transfer.BytesReceived != 3 || transfer.FirstByteMS == nil {
		t.Fatal("EOF confused with file verification")
	}
}
