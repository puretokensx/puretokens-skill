package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestContentPrefixTimeoutIsNotInvalidMedia(t *testing.T) {
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	svc := fixtureService(server)
	svc.downloadTimeout = 40 * time.Millisecond
	dir := t.TempDir()
	request := taskRequest{Kind: "image", TaskID: "existing-task", TaskStatus: "completed", RequestedCount: 1, OutputDir: dir}
	var out bytes.Buffer
	err := deliverTask(&out, svc, request, taskReceipt(request, request.TaskID, "completed"))
	result := decodeReceipt(t, &out)
	if err == nil || result.LocalErrorCode != "media_download_timeout" || result.NextStep != "await_user" || reads.Load() != 1 || result.TaskID != request.TaskID {
		t.Fatalf("prefix timeout lost: %+v calls=%d err=%v", result, reads.Load(), err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial output retained: %v %v", entries, err)
	}
}

func TestEveryContentFailureStopsAutomaticRetry(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		err    error
		status int
	}{
		{"incomplete", io.ErrUnexpectedEOF, 200},
		{"timeout", context.DeadlineExceeded, 0},
		{"invalid", errInvalidMedia, 200},
		{"permission", os.ErrPermission, 200},
		{"conflict", errors.New("existing output"), 0},
		{"upstream", errors.New("content request rejected"), 503},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			request := taskRequest{Kind: "image", TaskID: "existing-task", RequestedCount: 1}
			result := guideReceipt(mergeFailure(taskReceipt(request, request.TaskID, "completed"), contentFailure(scenario.err, scenario.status, 0, "", "")))
			if result.NextStep != "await_user" || !strings.Contains(result.NextAction, "Stop automatic retrieval") || !strings.Contains(result.NextAction, "explicit user request") {
				t.Fatalf("ambiguous retry guidance: %+v", result)
			}
		})
	}
}

func TestContentDeadlinesStopOnceAndPreserveTask(t *testing.T) {
	for _, stage := range []string{"response", "first_bytes", "mid_body"} {
		t.Run(stage, func(t *testing.T) {
			var reads atomic.Int32
			image := append(fixturePNG(t), make([]byte, 1024)...)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.Method != http.MethodGet {
					t.Error("content made a non-GET request")
				}
				if stage != "response" {
					w.Header().Set("Content-Type", "image/png")
					w.WriteHeader(http.StatusOK)
					if stage == "mid_body" {
						w.Write(image)
					}
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			svc := fixtureService(server)
			svc.downloadTimeout = 2 * time.Second
			svc.downloadResponseTimeout = 100 * time.Millisecond
			svc.downloadIdleTimeout = 100 * time.Millisecond
			dir := t.TempDir()
			request := taskRequest{Kind: "image", TaskID: "existing-task", TaskStatus: "completed", RequestedCount: 1, OutputDir: dir}
			var out bytes.Buffer
			writer := &supportReceiptWriter{output: &out, host: "codex", command: "content", timings: newCommandTimings()}
			svc.support = writer
			err := deliverTask(writer, svc, request, taskReceipt(request, request.TaskID, "completed"))
			result := decodeReceipt(t, &out)
			wantCode, wantStatus := "media_download_stalled", 200
			if stage == "response" {
				wantCode, wantStatus = "media_content_response_timeout", 0
			}
			if err == nil || result.OK || result.LocalErrorCode != wantCode || result.HTTPStatus != wantStatus || result.TaskID != request.TaskID || result.Status != "completed" || result.NextStep != "await_user" || reads.Load() != 1 || len(result.DownloadedPaths) != 0 {
				t.Fatalf("unsafe stalled download: %+v reads=%d err=%v", result, reads.Load(), err)
			}
			var envelope map[string]any
			json.Unmarshal(out.Bytes(), &envelope)
			support := envelope["support"].(map[string]any)
			if support["local_error_code"] != wantCode || support["timings_ms"] != nil {
				t.Fatalf("unsafe or missing support diagnosis: %v", support)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("partial output retained: %v %v", entries, err)
			}
		})
	}
}

func TestContentProgressResetsOnlyReadIdleBudget(t *testing.T) {
	png := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		for start := 0; start < len(png); start += 4 {
			select {
			case <-time.After(20 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
			w.Write(png[start:min(start+4, len(png))])
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()
	svc := fixtureService(server)
	svc.downloadResponseTimeout = 150 * time.Millisecond
	svc.downloadIdleTimeout = 150 * time.Millisecond
	svc.downloadTimeout = 3 * time.Second
	started := time.Now()
	file, _, _, _, _, err := svc.download(context.Background(), contentPath("image", "existing-task", 0), "image", t.TempDir())
	if err != nil || !validMediaFile(file, "image/png") || time.Since(started) <= svc.downloadIdleTimeout {
		t.Fatalf("healthy slow progress failed: %s %v", file, err)
	}
	if svc.client.Timeout != 0 || (service{}).contentResponseBudget() != 90*time.Second || (service{}).contentIdleBudget() != 30*time.Second || (service{}).contentTimeout() != 10*time.Minute {
		t.Fatal("content budgets changed the shared client or contract")
	}
}

func TestContentTimingSeparatesHeadersFromBody(t *testing.T) {
	clock, advance := timingClockFixture()
	svc := service{baseURL: "https://fixture.invalid", support: &supportReceiptWriter{timings: clock},
		client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
			advance(7 * time.Millisecond)
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}},
				Body: &timingBody{reader: bytes.NewReader(fixturePNG(t)), advance: advance}}, nil
		})}}
	if _, _, _, _, _, err := svc.download(context.Background(), contentPath("image", "existing-task", 0), "image", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	got := clock.snapshot()
	if got["content_response"] != 7 || got["download"] != 11 || got["total"] != 18 {
		t.Fatalf("content timings overlap or include unobserved work: %v", got)
	}
}

func TestContentProgressDoesNotExtendTotalTimeout(t *testing.T) {
	var reads atomic.Int32
	png := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
		w.(http.Flusher).Flush()
		for {
			select {
			case <-time.After(10 * time.Millisecond):
				w.Write([]byte{0})
				w.(http.Flusher).Flush()
			case <-r.Context().Done():
				return
			}
		}
	}))
	defer server.Close()
	svc := fixtureService(server)
	svc.downloadTimeout = 100 * time.Millisecond
	svc.downloadIdleTimeout = time.Second
	dir := t.TempDir()
	_, status, retry, code, message, err := svc.download(context.Background(), contentPath("image", "existing-task", 0), "image", dir)
	result := contentFailure(err, status, retry, code, message)
	if err == nil || result.LocalErrorCode != "media_download_timeout" || reads.Load() != 1 {
		t.Fatalf("total timeout reset by progress: %+v calls=%d err=%v", result, reads.Load(), err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial output retained: %v %v", entries, err)
	}
}

type emptyContentReader struct{}

func (emptyContentReader) Read([]byte) (int, error) { return 0, nil }

func TestEmptyReadsDoNotResetContentIdleBudget(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	reader := contentProgressReader{reader: emptyContentReader{}, ctx: ctx, cancel: cancel, budget: 20 * time.Millisecond, remaining: 20 * time.Millisecond}
	started := time.Now()
	for {
		_, err := reader.Read(make([]byte, 1))
		if err != nil {
			if !errors.Is(err, errContentIdleTimeout) {
				t.Fatal(err)
			}
			break
		}
		if time.Since(started) > time.Second {
			t.Fatal("empty reads kept resetting the idle budget")
		}
	}
}

func TestContentIdleBudgetExcludesLocalConsumerWork(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	reader := contentProgressReader{reader: strings.NewReader("ab"), ctx: ctx, cancel: cancel, budget: 20 * time.Millisecond, remaining: 20 * time.Millisecond}
	buffer := make([]byte, 1)
	if n, err := reader.Read(buffer); n != 1 || err != nil {
		t.Fatalf("first byte missing: %d %v", n, err)
	}
	// A slow local consumer is not a stalled network read.
	time.Sleep(40 * time.Millisecond)
	if n, err := reader.Read(buffer); n != 1 || err != nil || context.Cause(ctx) != nil {
		t.Fatalf("local work consumed network idle budget: %d %v", n, err)
	}
}

func TestContentCancellationDoesNotRetryOrLeaveOutput(t *testing.T) {
	started := make(chan struct{})
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-started; cancel() }()
	dir := t.TempDir()
	_, _, _, _, _, err := fixtureService(server).download(ctx, contentPath("image", "existing-task", 0), "image", dir)
	if !errors.Is(err, context.Canceled) || reads.Load() != 1 {
		t.Fatalf("cancellation lost or retried: %v calls=%d", err, reads.Load())
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancelled output retained: %v %v", entries, err)
	}
}

func TestContentBudgetContractMatchesExecutor(t *testing.T) {
	data, err := os.ReadFile("../../references/direct-api-execution-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Delivery struct {
			Budgets struct {
				Response int `json:"responseHeadersSeconds"`
				Idle     int `json:"bodyReadIdleSeconds"`
				Total    int `json:"totalTransferSeconds"`
			} `json:"contentReadBudgets"`
		} `json:"mediaDeliveryResourceBounds"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	budgets := contract.Delivery.Budgets
	if time.Duration(budgets.Response)*time.Second != (service{}).contentResponseBudget() || time.Duration(budgets.Idle)*time.Second != (service{}).contentIdleBudget() || time.Duration(budgets.Total)*time.Second != (service{}).contentTimeout() {
		t.Fatal("content deadlines drifted from the shared contract")
	}
}
