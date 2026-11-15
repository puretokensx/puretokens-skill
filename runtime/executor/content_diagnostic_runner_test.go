//go:build content_diagnostic

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// This tagged test binary is a maintainer instrument, never an installed Skill
// command or alternate delivery transport. Normal builds exclude this file.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--build-info" {
		writeJSON(os.Stdout, map[string]string{"format": "puretokens-content-diagnostic-v1", "executor_version": executorVersion, "diagnostic_source_sha256": diagnosticSourceSHA256})
		os.Exit(0)
	}
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-test.") {
		os.Exit(m.Run())
	}
	if len(os.Args) > 1 && os.Args[1] == "content-diagnose" {
		os.Exit(runContentDiagnostic(os.Args[2:], os.Stdout))
	}
	writeJSON(os.Stdout, map[string]any{"ok": false, "local_error_code": "diagnostic_request_invalid"})
	os.Exit(1)
}

var diagnosticSourceSHA256 = "unbuilt"

type diagnosticBudgets struct{ total, response, idle time.Duration }
type contentDiagnostic struct {
	Format                 string           `json:"format"`
	OK                     bool             `json:"ok"`
	ExecutorVersion        string           `json:"executor_version"`
	DiagnosticSourceSHA256 string           `json:"diagnostic_source_sha256"`
	OS                     string           `json:"os"`
	Arch                   string           `json:"arch"`
	Probe                  string           `json:"probe"`
	Requests               int              `json:"requests"`
	Protocol               string           `json:"protocol,omitempty"`
	HTTPStatus             int              `json:"http_status,omitempty"`
	RequestID              string           `json:"request_id,omitempty"`
	LocalErrorCode         string           `json:"local_error_code,omitempty"`
	Transfer               *contentTransfer `json:"content_transfer,omitempty"`
	ElapsedMS              int64            `json:"elapsed_ms"`
	TotalBudgetMS          int64            `json:"total_budget_ms"`
	IdleBudgetMS           int64            `json:"idle_budget_ms"`
	NextStep               string           `json:"next_step"`
}

func diagnosticRecordAllowed(record taskRecord, index int) bool {
	return record.Kind == "image" && validTaskID(record.TaskID) && terminalSuccess(record.Status) &&
		validOutputIndex(record.request(), index) && !record.ReconciliationRequired && record.RetryNotBefore == "" &&
		record.SubmissionOutcome != "unknown" && len(record.Downloaded) == 0 && len(record.Delivered) == 0
}

func runContentDiagnostic(args []string, output io.Writer) int {
	fail := func(code string) int {
		writeJSON(output, map[string]any{"format": "puretokens-content-diagnostic-v1", "ok": false, "requests": 0, "local_error_code": code, "next_step": "await_user"})
		return 1
	}
	flags := flag.NewFlagSet("content-diagnose", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	host := flags.String("host", "", "")
	path := flags.String("record", "", "")
	probe := flags.String("probe", "", "")
	index := flags.Int("index", 0, "")
	confirm := flags.Bool("confirm-one-get", false, "")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !*confirm || *host == "" ||
		!contains([]string{"executor", "plain", "http1", "long-idle"}, *probe) {
		return fail("diagnostic_request_invalid")
	}
	record, err := loadTaskRecord(*path)
	if err != nil || !diagnosticRecordAllowed(record, *index) {
		return fail("diagnostic_request_invalid")
	}
	token, connectionID, err := credentialForConnection(*host)
	if err != nil {
		code, _, _ := credentialFailureDetails(err)
		return fail(code)
	}
	defer clearString(&token)
	if record.ConnectionID != connectionID {
		return fail("task_connection_changed")
	}
	svc := service{baseURL: apiOrigin, client: &http.Client{CheckRedirect: rejectRedirect}, token: token, host: *host}
	budgets := diagnosticBudgets{total: 120 * time.Second, response: 30 * time.Second, idle: 30 * time.Second}
	if *probe == "long-idle" {
		budgets.idle = 90 * time.Second
	}
	result := diagnoseContent(context.Background(), svc, record, *index, *probe, budgets)
	writeJSON(output, result)
	if !result.OK {
		return 1
	}
	return 0
}

type diagnosticTransport struct {
	base        http.RoundTripper
	result      *contentDiagnostic
	cancel      context.CancelCauseFunc
	connections atomic.Int32
}

var errDiagnosticReplay = errors.New("diagnostic transport replay blocked")

func (transport *diagnosticTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport.result.Requests != 0 || request.Method != http.MethodGet {
		return nil, errors.New("diagnostic request limit")
	}
	transport.result.Requests++
	// Go may retry a GET before response headers inside RoundTrip. Abort before
	// any second connection assignment can send the same request again.
	trace := &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) {
		if transport.connections.Add(1) > 1 {
			transport.cancel(errDiagnosticReplay)
			if info.Conn != nil {
				_ = info.Conn.Close()
			}
		}
	}}
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
	response, err := transport.base.RoundTrip(request)
	if response != nil {
		if response.Proto == "HTTP/1.1" || response.Proto == "HTTP/2.0" {
			transport.result.Protocol = response.Proto
		}
	}
	return response, err
}

func diagnoseContent(ctx context.Context, svc service, record taskRecord, index int, probe string, budgets diagnosticBudgets) contentDiagnostic {
	result := contentDiagnostic{Format: "puretokens-content-diagnostic-v1", ExecutorVersion: executorVersion, DiagnosticSourceSHA256: diagnosticSourceSHA256, OS: runtime.GOOS, Arch: runtime.GOARCH, Probe: probe, NextStep: "await_user", TotalBudgetMS: budgets.total.Milliseconds(), IdleBudgetMS: budgets.idle.Milliseconds()}
	if !diagnosticRecordAllowed(record, index) || !contains([]string{"executor", "plain", "http1", "long-idle"}, probe) {
		result.LocalErrorCode = "diagnostic_request_invalid"
		return result
	}
	started := time.Now()
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	client := fixedClient(svc.client)
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	transport, ok := base.(*http.Transport)
	if !ok {
		result.LocalErrorCode = "diagnostic_transport_unsupported"
		return result
	}
	transport = transport.Clone()
	defer transport.CloseIdleConnections()
	if probe == "http1" {
		transport.Protocols = new(http.Protocols)
		transport.Protocols.SetHTTP1(true)
		if transport.TLSClientConfig != nil {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
			transport.TLSClientConfig.NextProtos = []string{"http/1.1"}
		}
	}
	client.Transport = &diagnosticTransport{base: transport, result: &result, cancel: cancel}
	client.Timeout = budgets.total
	svc.client = client
	svc.support = &supportReceiptWriter{command: "content", host: svc.host}
	svc.downloadTimeout, svc.downloadResponseTimeout, svc.downloadIdleTimeout = budgets.total, budgets.response, budgets.idle
	svc.imageFormat, _ = record.Parameters["output_format"].(string)
	svc.allowSVG = taskAllowsSVG(record.request())
	directory, err := os.MkdirTemp("", "puretokens-content-diagnostic-")
	if err != nil {
		result.LocalErrorCode = "output_directory_unavailable"
		return result
	}
	defer os.RemoveAll(directory)
	path := contentPath("image", record.TaskID, index)
	status := 0
	if probe == "plain" {
		status, err = diagnosticPlainDownload(ctx, svc, path, directory, budgets)
	} else {
		_, status, _, _, _, err = svc.download(ctx, path, "image", directory)
	}
	result.HTTPStatus, result.RequestID = svc.support.httpStatus, svc.support.requestID
	result.Transfer = svc.support.contentTransfer
	result.ElapsedMS = time.Since(started).Milliseconds()
	result.OK = err == nil && status == 200 && result.Transfer != nil && result.Transfer.ReadComplete && result.Transfer.FileVerified
	if !result.OK {
		if errors.Is(context.Cause(ctx), errDiagnosticReplay) {
			result.LocalErrorCode = "diagnostic_transport_replay_blocked"
		} else if (status != 0 && status != 200) || (result.HTTPStatus != 0 && result.HTTPStatus != 200) {
			result.LocalErrorCode = "content_request_rejected"
		} else if result.HTTPStatus == 0 {
			result.LocalErrorCode = transportErrorCode(err)
		} else {
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
			result.LocalErrorCode = contentFailure(err, status, 0, "", "").LocalErrorCode
		}
	}
	return result
}

// A separate supervisor enforces the same read-idle budget without using the
// production progress reader, bufio, file copy, or finalization path.
func diagnosticRead(ctx context.Context, cancel context.CancelCauseFunc, body io.ReadCloser, transfer *contentTransfer, limit int64, idle time.Duration) ([]byte, error) {
	type chunk struct {
		data []byte
		err  error
	}
	reads := make(chan chunk)
	done := make(chan struct{})
	defer close(done)
	defer body.Close()
	go func() {
		defer close(reads)
		for {
			buffer := make([]byte, 32<<10)
			started := time.Now()
			n, err := body.Read(buffer)
			transfer.observeRead(n, err, time.Since(started))
			select {
			case reads <- chunk{buffer[:n], err}:
			case <-done:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	// Wait for the sole reader before returning its diagnostic fields.
	defer func() {
		body.Close()
		for range reads {
		}
	}()
	timer := time.NewTimer(idle)
	defer timer.Stop()
	var data []byte
	for {
		select {
		case <-ctx.Done():
			return data, context.Cause(ctx)
		case <-timer.C:
			cancel(errContentIdleTimeout)
			return data, errContentIdleTimeout
		case part, open := <-reads:
			if !open {
				return data, io.ErrUnexpectedEOF
			}
			data = append(data, part.data...)
			if int64(len(data)) > limit {
				return data, errMediaTooLarge
			}
			if len(part.data) > 0 {
				timer.Reset(idle)
			}
			if part.err == io.EOF {
				return data, nil
			}
			if part.err != nil {
				return data, part.err
			}
		}
	}
}

func diagnosticPlainDownload(ctx context.Context, svc service, path, directory string, budgets diagnosticBudgets) (int, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+svc.token)
	svc.support.beginRequest(http.MethodGet, path)
	stop := contentDeadline(cancel, budgets.response, errContentResponseTimeout)
	response, err := svc.client.Do(request)
	stop()
	svc.support.observeResponse(response)
	if response != nil {
		defer response.Body.Close()
	}
	if cause := context.Cause(ctx); cause != nil {
		return 0, cause
	}
	if err != nil {
		return 0, err
	}
	if response.StatusCode != 200 {
		return response.StatusCode, errors.New("content request rejected")
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if !imageFormatMatches(svc.imageFormat, mediaType) {
		return 200, errMediaFormatMismatch
	}
	if !contains([]string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/avif", "image/svg+xml"}, mediaType) || (mediaType == "image/svg+xml" && (!svc.allowSVG || response.Header.Get("Content-Range") != "")) {
		return 200, errInvalidMedia
	}
	limit := maxImageBytes
	if mediaType == "image/svg+xml" {
		limit = maxSVGBytes
	}
	if response.ContentLength > limit {
		return 200, errMediaTooLarge
	}
	transfer := newContentTransfer(response)
	svc.support.contentTransfer = transfer
	data, err := diagnosticRead(ctx, cancel, response.Body, transfer, limit, budgets.idle)
	if err != nil {
		return 200, err
	}
	if transfer.ExpectedBytes != nil && int64(len(data)) != *transfer.ExpectedBytes {
		return 200, io.ErrUnexpectedEOF
	}
	file := filepath.Join(directory, "diagnostic-content")
	if err := os.WriteFile(file, data, 0600); err != nil {
		return 200, err
	}
	if !validMediaFile(file, mediaType) {
		return 200, errInvalidMedia
	}
	transfer.FileVerified = true
	return 200, nil
}
