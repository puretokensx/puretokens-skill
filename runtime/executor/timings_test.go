package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLocalCommandTimingDoesNotClaimNetworkOrHostDelivery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.json")
	saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "completed", RequestedCount: 1}, true)
	var out bytes.Buffer
	if err := run([]string{"resume", "--host", "fixture-unsupported", "--record", path}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(out.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	timings, ok := document["timings_ms"].(map[string]any)
	if !ok || timings["total"] == nil || timings["preparation"] == nil {
		t.Fatalf("local command has no measured invocation timing: %s", out.Bytes())
	}
	for _, key := range []string{"request", "catalog", "download", "wait", "host_handoff", "server_generation"} {
		if _, exists := timings[key]; exists {
			t.Fatalf("unobserved phase %q reported", key)
		}
	}
	if support := document["support"].(map[string]any); support["timings_ms"] != nil {
		t.Fatal("timings entered private support evidence")
	}
	saved, _ := loadTaskRecord(path)
	data, _ := json.Marshal(saved)
	if bytes.Contains(data, []byte("timings")) {
		t.Fatal("invocation timing entered persistent task record")
	}
}

func timingClockFixture() (*commandTimings, func(time.Duration)) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	clock := &commandTimings{now: func() time.Time { return now }, started: now, last: now,
		active: "preparation", durations: map[string]time.Duration{}, seen: map[string]bool{"preparation": true}}
	return clock, func(duration time.Duration) { now = now.Add(duration) }
}

func TestTimingPhasesAreExclusiveMeasuredAndAllowlisted(t *testing.T) {
	clock, advance := timingClockFixture()
	advance(2 * time.Millisecond)
	finishDownload := clock.enter("download")
	advance(5 * time.Millisecond)
	finishVerify := clock.enter("verification")
	advance(3 * time.Millisecond)
	if got := clock.snapshot(); !reflect.DeepEqual(got, map[string]int64{"preparation": 2, "download": 5, "verification": 3, "total": 10}) {
		t.Fatalf("active nested phase is missing or counted twice: %v", got)
	}
	finishVerify()
	advance(7 * time.Millisecond)
	finishDownload()
	finishDownload() // Closing a finished phase cannot change accounting.
	clock.enter("private_path_or_server_generation")()
	advance(1 * time.Millisecond)
	if got := clock.snapshot(); !reflect.DeepEqual(got, map[string]int64{"preparation": 3, "download": 12, "verification": 3, "total": 18}) {
		t.Fatalf("unobserved stage or overlapping timing: %v", got)
	}
}

type timingBody struct {
	reader   io.Reader
	advance  func(time.Duration)
	advanced bool
}

func (body *timingBody) Read(bytes []byte) (int, error) {
	if !body.advanced {
		body.advance(11 * time.Millisecond)
		body.advanced = true
	}
	return body.reader.Read(bytes)
}
func (*timingBody) Close() error { return nil }

func TestRequestTimingIncludesBoundedBodyAndUnknownFailures(t *testing.T) {
	for _, scenario := range []string{"catalog", "status", "unknown-submission"} {
		t.Run(scenario, func(t *testing.T) {
			clock, advance := timingClockFixture()
			var output bytes.Buffer
			writer := &supportReceiptWriter{output: &output, command: "submit", timings: clock}
			calls := 0
			svc := service{baseURL: "https://fixture.invalid", token: "synthetic-fixture-token", support: writer}
			svc.client = &http.Client{Transport: evaluationTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				advance(7 * time.Millisecond)
				if scenario == "unknown-submission" {
					return nil, errors.New("fixture network interruption")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: &timingBody{reader: strings.NewReader(`{"id":"paid-task","status":"pending"}`), advance: advance}}, nil
			})}
			path := "/v1/images/paid-task"
			if scenario == "catalog" {
				path = "/v1/media/models"
			}
			if scenario == "unknown-submission" {
				request := taskRequest{Kind: "image", Operation: "generate", Model: "fixture-model", Prompt: "fixture", RequestedCount: 1}
				if err := executePreparedTask(writer, svc, request); err == nil {
					t.Fatal("network failure accepted")
				}
				var result map[string]any
				json.Unmarshal(output.Bytes(), &result)
				if result["submission_outcome"] != "unknown" || result["timings_ms"].(map[string]any)["request"] != float64(7) || calls != 1 {
					t.Fatalf("unknown submission invented timing or retry: %s", output.Bytes())
				}
				return
			}
			if _, _, _, _, _, err := svc.request(context.Background(), http.MethodGet, path, nil, ""); err != nil {
				t.Fatal(err)
			}
			phase := "request"
			if scenario == "catalog" {
				phase = "catalog"
			}
			if got := clock.snapshot(); got[phase] != 18 || got["preparation"] != 0 || got["total"] != 18 || calls != 1 {
				t.Fatalf("body wait was not counted with request: %v", got)
			}
		})
	}
}

func TestPreflightAndWaitTimingDoNotAddRequests(t *testing.T) {
	clock, advance := timingClockFixture()
	var output bytes.Buffer
	writer := &supportReceiptWriter{output: &output, command: "preflight", timings: clock}
	svc := profileService()
	svc.support = writer
	svc.client = &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("valid local preflight made network request for timing")
		return nil, errors.New("network forbidden")
	})}
	if err := executePreflight(writer, svc, taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2.5-flare", Prompt: "fixture"}); err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	json.Unmarshal(output.Bytes(), &receipt)
	if got := receipt["timings_ms"].(map[string]any); len(got) != 2 || got["preparation"] == nil || got["total"] == nil {
		t.Fatalf("local preflight invented network phase: %v", got)
	}
	svc.wait = func(context.Context, time.Duration) bool { advance(9 * time.Millisecond); return true }
	if !svc.waitFor(context.Background(), 5*time.Second) || clock.snapshot()["wait"] != 9 {
		t.Fatal("timing recorded requested delay instead of observed duration")
	}
}

func TestLocalRedeliveryMeasuresVerificationWithoutHandoffOrNetwork(t *testing.T) {
	image := fixturePNG(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "task.json")
	var networkCalls int
	var output bytes.Buffer
	clock, _ := timingClockFixture()
	writer := &supportReceiptWriter{output: &output, command: "content", timings: clock}
	svc := service{baseURL: "https://fixture.invalid", token: "synthetic-fixture-token", support: writer,
		client: &http.Client{Transport: evaluationTransport(func(*http.Request) (*http.Response, error) {
			networkCalls++
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(bytes.NewReader(image))}, nil
		})}}
	saveTaskRecord(path, taskRecord{Format: taskRecordFormat, Kind: "image", TaskID: "paid-task", Status: "completed", RequestedCount: 1}, true)
	if err := executeRecordedTask("content", path, taskRequest{}, 0, dir, writer, svc); err != nil {
		t.Fatal(err)
	}
	clock, _ = timingClockFixture()
	writer.timings = clock
	output.Reset()
	if err := executeRecordedTask("redeliver", path, taskRequest{}, 0, "", writer, svc); err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	json.Unmarshal(output.Bytes(), &receipt)
	timing := receipt["timings_ms"].(map[string]any)
	if _, exists := timing["verification"]; !exists || timing["download"] != nil || timing["request"] != nil || timing["host_handoff"] != nil || networkCalls != 1 {
		t.Fatalf("local verified handoff timings invented work: %s", output.Bytes())
	}
}
