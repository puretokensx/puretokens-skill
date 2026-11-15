package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// Local TLS fixtures compare the actual download pipeline with an unwrapped
// response-body read. They do not exercise production or a user's network.
func TestContentTransportParity(t *testing.T) {
	pixels := image.NewRGBA(image.Rect(0, 0, 768, 512))
	state := uint32(1)
	for index := range pixels.Pix {
		state = state*1664525 + 1013904223
		pixels.Pix[index] = byte(state >> 24)
		if index%4 == 3 {
			pixels.Pix[index] = 255
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, pixels, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	payload := encoded.Bytes()
	if len(payload) <= 128<<10 {
		t.Fatal("fixture must cross the 128 KiB boundary")
	}

	for _, protocol := range []int{1, 2} {
		for _, mode := range []string{"known_length", "continuous_chunks", "pause_after_128k"} {
			for _, reader := range []string{"plain", "executor", "executor_longer_idle"} {
				t.Run("http"+strconv.Itoa(protocol)+"/"+mode+"/"+reader, func(t *testing.T) {
					var calls, observedProtocol atomic.Int32
					server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						observedProtocol.Store(int32(r.ProtoMajor))
						if r.Method != http.MethodGet || r.URL.Path != "/fixture" {
							t.Error("unexpected fixture request")
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						w.Header().Set("Content-Type", "image/jpeg")
						if mode != "continuous_chunks" {
							w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
						}
						for offset := 0; offset < len(payload); offset += 16 << 10 {
							pause := time.Duration(0)
							if mode == "continuous_chunks" {
								pause = 20 * time.Millisecond
							} else if mode == "pause_after_128k" && offset == 128<<10 {
								pause = 350 * time.Millisecond
							}
							if pause > 0 {
								select {
								case <-time.After(pause):
								case <-r.Context().Done():
									return
								}
							}
							if _, err := w.Write(payload[offset:min(offset+(16<<10), len(payload))]); err != nil {
								return
							}
							w.(http.Flusher).Flush()
						}
					}))
					server.EnableHTTP2 = protocol == 2
					server.StartTLS()
					defer server.Close()
					client := server.Client()
					client.Timeout = 3 * time.Second
					var got []byte
					if reader == "plain" {
						response, err := client.Get(server.URL + "/fixture")
						if err != nil {
							t.Fatal(err)
						}
						defer response.Body.Close()
						got, err = io.ReadAll(response.Body)
						if err != nil {
							t.Fatal(err)
						}
					} else {
						svc := fixtureService(server)
						svc.downloadTimeout = 3 * time.Second
						svc.downloadIdleTimeout = 200 * time.Millisecond
						if reader == "executor_longer_idle" {
							svc.downloadIdleTimeout = 600 * time.Millisecond
						}
						svc.support = &supportReceiptWriter{command: "content", host: "codex"}
						dir := t.TempDir()
						file, status, _, _, _, err := svc.download(context.Background(), "/fixture", "image", dir)
						transfer := svc.support.contentTransfer
						if mode == "pause_after_128k" && reader == "executor" {
							if !errors.Is(err, errContentIdleTimeout) || status != 200 || file != "" || transfer == nil || transfer.BytesReceived != 128<<10 || transfer.ReadComplete || transfer.FileVerified {
								t.Fatalf("unexpected idle evidence: status=%d transfer=%+v err=%v", status, transfer, err)
							}
							entries, readErr := os.ReadDir(dir)
							if readErr != nil || len(entries) != 0 {
								t.Fatal("partial output retained")
							}
							if calls.Load() != 1 || observedProtocol.Load() != int32(protocol) {
								t.Fatal("request count or negotiated protocol mismatch")
							}
							t.Logf("expected idle stop: received=%d, no retry", transfer.BytesReceived)
							return
						}
						if err != nil || status != 200 || transfer == nil || !transfer.ReadComplete || !transfer.FileVerified || transfer.BytesReceived != int64(len(payload)) {
							t.Fatalf("healthy transfer failed: status=%d transfer=%+v err=%v", status, transfer, err)
						}
						got, err = os.ReadFile(file)
						if err != nil {
							t.Fatal(err)
						}
					}
					if !bytes.Equal(got, payload) || calls.Load() != 1 || observedProtocol.Load() != int32(protocol) {
						t.Fatal("bytes, request count or negotiated protocol mismatch")
					}
					t.Logf("complete: bytes=%d, protocol=HTTP/%d, requests=1", len(got), protocol)
				})
			}
		}
	}
}
