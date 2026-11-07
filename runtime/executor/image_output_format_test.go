package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image/gif"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRequestedGIFDeliveryPreservesOriginalOrStops(t *testing.T) {
	for _, correct := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject-static-png", true: "preserve-animation"}[correct], func(t *testing.T) {
			data, mime := fixturePNG(t), "image/png"
			if correct {
				data, mime = fixtureGIF(t), "image/gif"
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.RequestURI() != "/v1/images/original-task/content?index=0" {
					t.Error("not the same task content")
				}
				w.Header().Set("Content-Type", mime)
				w.Write(data)
			}))
			defer server.Close()
			dir := t.TempDir()
			r := taskRequest{Kind: "image", Model: "gpt-image-2.6", TaskID: "original-task", RequestedCount: 1, OriginalOperation: "generate", OutputDir: dir, Parameters: map[string]any{"output_format": "gif"}}
			var out bytes.Buffer
			err := deliverTask(&out, fixtureService(server), r, taskReceipt(r, r.TaskID, "completed"))
			result := decodeReceipt(t, &out)
			if calls != 1 || result.TaskID != r.TaskID {
				t.Fatal("task identity or request count changed")
			}
			if !correct {
				files, _ := os.ReadDir(dir)
				if err == nil || len(result.DownloadedPaths) != 0 || len(files) != 0 {
					t.Fatal("static PNG accepted as requested GIF")
				}
				return
			}
			if err != nil || len(result.DownloadedPaths) != 1 {
				t.Fatalf("GIF rejected: %v", err)
			}
			actual, err := os.ReadFile(result.DownloadedPaths[0])
			if err != nil || !bytes.Equal(actual, data) || filepath.Ext(result.DownloadedPaths[0]) != ".gif" {
				t.Fatal("GIF bytes or extension changed")
			}
			frames, err := gif.DecodeAll(bytes.NewReader(actual))
			if err != nil || len(frames.Image) != 2 {
				t.Fatal("animation frames lost")
			}
		})
	}
}

func TestRequestedGIFDoesNotReuseWrongFormatProof(t *testing.T) {
	for _, command := range []string{"content", "resume", "redeliver", "delivered"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, fmt.Sprintf("puretokens-%x.png", sha256.Sum256([]byte(contentPath("image", "original-task", 0)))))
			os.WriteFile(file, fixturePNG(t), 0600)
			proof, err := fingerprintDownload(file, "image/png")
			if err != nil {
				t.Fatal(err)
			}
			record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "original-task", Model: "gpt-image-2.6", OriginalOperation: "generate", RequestedCount: 1, Status: "completed", OutputDir: dir, Parameters: map[string]any{"output_format": "gif"}, Downloaded: map[int]string{0: file}, DownloadProofs: map[int]downloadProof{0: proof}}
			path := filepath.Join(dir, "task.json")
			// Model an old record produced before output-format enforcement.
			if err := saveTaskRecord(path, record, true); err != nil {
				t.Fatal(err)
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(503) }))
			defer server.Close()
			var out bytes.Buffer
			err = executeRecordedTask(command, path, taskRequest{}, 0, "", &out, fixtureService(server))
			result := decodeReceipt(t, &out)
			if err == nil || len(result.DownloadedPaths) > 0 || calls > 0 {
				t.Fatal("wrong format reused, acknowledged or requested again")
			}
			if _, err := os.Stat(file); err != nil {
				t.Fatal("old artifact removed")
			}
		})
	}
}

func TestRequestedGIFCanRecoverSameTaskIntoNewDirectory(t *testing.T) {
	for _, command := range []string{"content", "redeliver"} {
		t.Run(command, func(t *testing.T) {
			dir, newDir := t.TempDir(), t.TempDir()
			file := filepath.Join(dir, fmt.Sprintf("puretokens-%x.png", sha256.Sum256([]byte(contentPath("image", "original-task", 0)))))
			oldBytes := fixturePNG(t)
			if err := os.WriteFile(file, oldBytes, 0600); err != nil {
				t.Fatal(err)
			}
			proof, err := fingerprintDownload(file, "image/png")
			if err != nil {
				t.Fatal(err)
			}
			record := taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "original-task", Model: "gpt-image-2.6", OriginalOperation: "generate", RequestedCount: 1, Status: "completed", OutputDir: dir, Parameters: map[string]any{"output_format": "gif"}, Downloaded: map[int]string{0: file}, DownloadProofs: map[int]downloadProof{0: proof}}
			if command == "redeliver" {
				record.Delivered = []int{0}
			}
			// Another old wrong-format index must not prevent fixing this one.
			other := filepath.Join(dir, fmt.Sprintf("puretokens-%x.png", sha256.Sum256([]byte(contentPath("image", "original-task", 1)))))
			if err := os.WriteFile(other, oldBytes, 0600); err != nil {
				t.Fatal(err)
			}
			otherProof, err := fingerprintDownload(other, "image/png")
			if err != nil {
				t.Fatal(err)
			}
			record.RequestedCount = 2
			record.Downloaded[1], record.DownloadProofs[1] = other, otherProof
			path := filepath.Join(dir, "task.json")
			if err := saveTaskRecord(path, record, true); err != nil {
				t.Fatal(err)
			}
			calls, gifBytes := 0, fixtureGIF(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.RequestURI() != "/v1/images/original-task/content?index=0" {
					t.Error("recovery changed task or submitted again")
				}
				w.Header().Set("Content-Type", "image/gif")
				w.Write(gifBytes)
			}))
			defer server.Close()
			var out bytes.Buffer
			if err := executeRecordedTask(command, path, taskRequest{}, 0, newDir, &out, fixtureService(server)); err != nil {
				t.Fatal(err)
			}
			result := decodeReceipt(t, &out)
			if !result.OK || result.NextStep != "deliver" || calls != 1 || len(result.DownloadedPaths) != 1 || filepath.Dir(result.DownloadedPaths[0]) != newDir {
				t.Fatalf("invalid recovered receipt: %+v", result)
			}
			actual, err := os.ReadFile(result.DownloadedPaths[0])
			if err != nil || !bytes.Equal(actual, gifBytes) {
				t.Fatal("GIF was not preserved")
			}
			actual, err = os.ReadFile(file)
			if err != nil || !bytes.Equal(actual, oldBytes) {
				t.Fatal("old output was overwritten")
			}
			saved, err := loadTaskRecord(path)
			if err != nil || saved.TaskID != record.TaskID || saved.Parameters["output_format"] != "gif" || saved.DownloadProofs[0].MediaType != "image/gif" {
				t.Fatal("recovery lost task identity, format or proof")
			}
			if command == "redeliver" && (len(saved.Delivered) != 1 || saved.PendingRedeliveryIndex == nil) {
				t.Fatal("new handoff erased historical delivery or was acknowledged early")
			}
			if saved.Downloaded[1] != other || saved.DownloadProofs[1] != otherProof {
				t.Fatal("unrequested index was modified")
			}
			saved.Delivered = []int{0, 1}
			if saved.deliveryComplete() {
				t.Fatal("wrong-format historical output counts as complete")
			}
		})
	}
}

func TestExplicitImageFormatMatchingPreservesLegacyAndNonImageTasks(t *testing.T) {
	for _, entry := range []struct {
		format, mime string
		matches      bool
	}{
		{"png", "image/png", true}, {"jpeg", "image/jpeg", true},
		{"webp", "image/webp", true}, {"gif", "image/gif", true},
		{"gif", "image/png", false}, {"png", "image/gif", false},
		{"svg", "image/svg+xml", false}, {"", "image/png", true},
		{"", "image/gif", true},
	} {
		request := taskRequest{Kind: "image", Parameters: map[string]any{"output_format": entry.format}}
		if taskImageFormatMatches(request, entry.mime) != entry.matches {
			t.Fatalf("format %q and MIME %q", entry.format, entry.mime)
		}
	}
	if !taskImageFormatMatches(taskRequest{Kind: "image"}, "image/jpeg") || !taskImageFormatMatches(taskRequest{Kind: "video"}, "video/mp4") || !taskImageFormatMatches(taskRequest{Kind: "audio_task"}, "audio/wav") {
		t.Fatal("legacy or non-image delivery restricted")
	}
}
