package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestInitSharesOneTwentySecondDeadline(t *testing.T) {
	var deadlines []time.Time
	svc := service{baseURL: "https://fixture.invalid", client: &http.Client{
		Transport: pollingFixtureTransport(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > 20*time.Second || time.Until(deadline) <= 0 {
				t.Fatalf("missing or incorrect init deadline: %v", deadline)
			}
			deadlines = append(deadlines, deadline)
			body := `{"status":"ok","name":"Pure Tokens API","base_url":"/v1"}`
			if len(deadlines) == 2 {
				body = `{"data":[]}`
			}
			if r.Method != http.MethodGet {
				t.Fatal("init must be read-only")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
		}),
	}}
	var out bytes.Buffer
	if err := executeInit(&out, svc); err != nil {
		t.Fatal(err)
	}
	if len(deadlines) != 2 || !deadlines[0].Equal(deadlines[1]) {
		t.Fatalf("init reset its deadline: %v", deadlines)
	}
	var result initReceipt
	if json.Unmarshal(out.Bytes(), &result) != nil || !strings.Contains(result.NextAction, "already available") || !strings.Contains(result.NextAction, "only when required") {
		t.Fatalf("init unnecessarily restarts the conversation: %s", out.String())
	}
	if strings.Contains(strings.Join(result.UsageExamples, " "), "then start a new host conversation") {
		t.Fatal("usage example forces a new conversation")
	}
}

func TestInitDeadlineStopsWithoutRetryAndPreservesIdentityResult(t *testing.T) {
	for _, blockAt := range []int{1, 2} {
		t.Run(string(rune('0'+blockAt)), func(t *testing.T) {
			calls := 0
			svc := service{baseURL: "https://fixture.invalid", client: &http.Client{
				Transport: pollingFixtureTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls >= blockAt {
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"status":"ok","name":"Pure Tokens API","base_url":"/v1"}`)), Header: http.Header{}}, nil
				}),
			}}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			var out bytes.Buffer
			start := time.Now()
			err := executeInitContext(ctx, &out, svc)
			var result initReceipt
			if json.Unmarshal(out.Bytes(), &result) != nil {
				t.Fatal("missing sanitized init receipt")
			}
			if err == nil || result.OK || result.CredentialVerified || calls != blockAt || time.Since(start) > time.Second {
				t.Fatalf("deadline/retry failure: calls=%d receipt=%+v err=%v", calls, result, err)
			}
			if result.APIIdentityConfirmed != (blockAt == 2) {
				t.Fatalf("identity evidence lost: %+v", result)
			}
		})
	}
}

func TestInitSharedBudgetIncludesResponseBodies(t *testing.T) {
	for _, blockAt := range []int32{1, 2} {
		t.Run(fmt.Sprint(blockAt), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) < blockAt {
					fmt.Fprint(w, `{"status":"ok","name":"Pure Tokens API","base_url":"/v1"}`)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Length", "200")
				fmt.Fprint(w, `{"`)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			var output bytes.Buffer
			start := time.Now()
			err := executeInitContext(ctx, &output, service{baseURL: server.URL, client: server.Client()})
			var result initReceipt
			if json.Unmarshal(output.Bytes(), &result) != nil {
				t.Fatal("missing sanitized init receipt")
			}
			if err == nil || result.OK || result.CredentialVerified || calls.Load() != blockAt || time.Since(start) > time.Second {
				t.Fatalf("body deadline was not enforced: calls=%d result=%+v err=%v", calls.Load(), result, err)
			}
			if result.APIIdentityConfirmed != (blockAt == 2) {
				t.Fatal("completed identity evidence was lost")
			}
		})
	}
}
