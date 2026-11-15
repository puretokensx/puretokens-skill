//go:build content_diagnostic && windows

package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This entire transport exists only in the maintainer's tagged test binary.
// WinHTTP owns DNS, connection, TLS and HTTP; net/http is only the call interface.
var diagnosticWinDLL = windows.NewLazySystemDLL("winhttp.dll")
var diagnosticWinOpen = diagnosticWinDLL.NewProc("WinHttpOpen")
var diagnosticWinConnect = diagnosticWinDLL.NewProc("WinHttpConnect")
var diagnosticWinOpenRequest = diagnosticWinDLL.NewProc("WinHttpOpenRequest")
var diagnosticWinSetOption = diagnosticWinDLL.NewProc("WinHttpSetOption")
var diagnosticWinQueryOption = diagnosticWinDLL.NewProc("WinHttpQueryOption")
var diagnosticWinSetTimeouts = diagnosticWinDLL.NewProc("WinHttpSetTimeouts")
var diagnosticWinSetCallback = diagnosticWinDLL.NewProc("WinHttpSetStatusCallback")
var diagnosticWinSend = diagnosticWinDLL.NewProc("WinHttpSendRequest")
var diagnosticWinReceive = diagnosticWinDLL.NewProc("WinHttpReceiveResponse")
var diagnosticWinQueryHeaders = diagnosticWinDLL.NewProc("WinHttpQueryHeaders")
var diagnosticWinRead = diagnosticWinDLL.NewProc("WinHttpReadData")
var diagnosticWinClose = diagnosticWinDLL.NewProc("WinHttpCloseHandle")

const (
	diagnosticWinHandleClosing    = 0x00000800
	diagnosticWinHeadersAvailable = 0x00020000
	diagnosticWinReadComplete     = 0x00080000
	diagnosticWinRequestError     = 0x00200000
	diagnosticWinSendComplete     = 0x00400000
)

type diagnosticWinEvent struct {
	status uint32
	count  uint32
	errno  syscall.Errno
}
type diagnosticWinAsync struct {
	events  chan diagnosticWinEvent
	closed  chan struct{}
	buffer  []byte
	headers []uint16
	pins    runtime.Pinner
}

var diagnosticWinContexts sync.Map
var diagnosticWinSequence atomic.Uint64
var diagnosticWinCallback = windows.NewCallback(func(handle, identity uintptr, status uint32, information unsafe.Pointer, length uint32) uintptr {
	value, ok := diagnosticWinContexts.Load(identity)
	if !ok {
		return 0
	}
	state := value.(*diagnosticWinAsync)
	if status == diagnosticWinHandleClosing {
		diagnosticWinContexts.Delete(identity)
		clear(state.headers)
		state.pins.Unpin()
		close(state.closed)
		return 0
	}
	event := diagnosticWinEvent{status: status, count: length}
	if status == diagnosticWinRequestError && information != nil {
		type asyncResult struct {
			result    uintptr
			errorCode uint32
		}
		if uintptr(length) < unsafe.Sizeof(asyncResult{}) {
			return 0
		}
		event.errno = syscall.Errno((*asyncResult)(information).errorCode)
	}
	// Only one operation is in flight. Never re-enter WinHTTP from the callback.
	select {
	case state.events <- event:
	default:
	}
	return 0
})

type diagnosticWinTransport struct {
	target              string
	result              *contentDiagnostic
	budgets             diagnosticBudgets
	session, connection uintptr
	request             *diagnosticWinBody
	used                bool
}

func diagnosticWinSetDWORD(handle uintptr, option uint32, value uint32) error {
	ok, _, err := diagnosticWinSetOption.Call(handle, uintptr(option), uintptr(unsafe.Pointer(&value)), 4)
	if ok == 0 {
		return err
	}
	return nil
}

func newDiagnosticNativeTransport(target string, result *contentDiagnostic, budgets diagnosticBudgets) (http.RoundTripper, func(), error) {
	address, mode, err := diagnosticProxyForTarget(target)
	if err != nil {
		return nil, nil, err
	}
	transport, err := diagnosticNewWinTransport(target, address, mode, result, budgets)
	if err != nil {
		return nil, nil, err
	}
	return transport, transport.close, nil
}

func diagnosticNewWinTransport(target, proxy, proxyMode string, result *contentDiagnostic, budgets diagnosticBudgets) (*diagnosticWinTransport, error) {
	parsed, err := url.Parse(target)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errDiagnosticNativeUnsupported
	}
	// HTTP is only reachable from local DLL fixtures; the CLI always binds apiOrigin.
	if parsed.Scheme == "http" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		return nil, errDiagnosticNativeUnsupported
	}
	access := uintptr(1) // NO_PROXY, never Windows system/PAC discovery.
	var proxyUTF16 *uint16
	if proxy != "" {
		access = 3 // NAMED_PROXY, preserving the Go-selected HTTP proxy.
		proxyUTF16, err = windows.UTF16PtrFromString(proxy)
		if err != nil {
			return nil, errDiagnosticNativeUnsupported
		}
	}
	// Match the existing Go request user agent so this is not an extra variable.
	agent, _ := windows.UTF16PtrFromString("Go-http-client/1.1")
	session, _, err := diagnosticWinOpen.Call(uintptr(unsafe.Pointer(agent)), access, uintptr(unsafe.Pointer(proxyUTF16)), 0, 0x10000000)
	if session == 0 {
		return nil, err
	}
	transport := &diagnosticWinTransport{target: target, result: result, budgets: budgets, session: session}
	success := false
	defer func() {
		if !success {
			transport.close()
		}
	}()
	// Windows versions that cannot enforce the no-replay policy stop before GET.
	retries := [2]uint32{0, 0}
	if ok, _, _ := diagnosticWinSetOption.Call(session, 162, uintptr(unsafe.Pointer(&retries[0])), 8); ok == 0 {
		return nil, errDiagnosticNativeUnsupported
	}
	if diagnosticWinSetDWORD(session, 4, 1) != nil || diagnosticWinSetDWORD(session, 133, 1) != nil {
		return nil, errDiagnosticNativeUnsupported
	}
	timeout := uintptr(budgets.response.Milliseconds())
	if timeout == 0 || budgets.total <= 0 || budgets.idle <= 0 {
		return nil, errDiagnosticNativeUnsupported
	}
	if ok, _, _ := diagnosticWinSetTimeouts.Call(session, timeout, timeout, timeout, timeout); ok == 0 {
		return nil, errDiagnosticNativeUnsupported
	}
	host, _ := windows.UTF16PtrFromString(parsed.Hostname())
	port := 443
	if parsed.Scheme == "http" {
		port = 80
	}
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, errDiagnosticNativeUnsupported
		}
	}
	transport.connection, _, err = diagnosticWinConnect.Call(session, uintptr(unsafe.Pointer(host)), uintptr(port), 0)
	if transport.connection == 0 {
		return nil, err
	}
	result.ProxyMode = proxyMode
	success = true
	return transport, nil
}

func (transport *diagnosticWinTransport) close() {
	if transport.request != nil {
		transport.request.Close()
	}
	if transport.connection != 0 {
		diagnosticWinClose.Call(transport.connection)
		transport.connection = 0
	}
	if transport.session != 0 {
		diagnosticWinClose.Call(transport.session)
		transport.session = 0
	}
}

type diagnosticWinBody struct {
	ctx    context.Context
	mu     sync.Mutex
	handle uintptr
	state  *diagnosticWinAsync
	stop   func() bool
}

// The forwarded pointer arguments must remain alive across the system call.
//
//go:uintptrescapes
func (body *diagnosticWinBody) call(proc *windows.LazyProc, args ...uintptr) error {
	body.mu.Lock()
	defer body.mu.Unlock()
	if body.handle == 0 {
		return context.Canceled
	}
	values := append([]uintptr{body.handle}, args...)
	ok, _, err := proc.Call(values...)
	if ok == 0 {
		return err
	}
	return nil
}

func (body *diagnosticWinBody) wait(status uint32) (diagnosticWinEvent, error) {
	select {
	case <-body.ctx.Done():
		return diagnosticWinEvent{}, context.Cause(body.ctx)
	case <-body.state.closed:
		return diagnosticWinEvent{}, context.Canceled
	case event := <-body.state.events:
		if event.status == diagnosticWinRequestError {
			if event.errno == 12002 {
				if status == diagnosticWinReadComplete {
					return event, errContentIdleTimeout
				}
				return event, errContentResponseTimeout
			}
			return event, event.errno
		}
		if event.status != status {
			return event, errors.New("unexpected native completion")
		}
		return event, nil
	}
}

func (body *diagnosticWinBody) Close() error {
	body.mu.Lock()
	defer body.mu.Unlock()
	if body.stop != nil {
		body.stop()
	}
	if body.handle != 0 {
		diagnosticWinClose.Call(body.handle)
		body.handle = 0
	}
	return nil
}

func (body *diagnosticWinBody) Read(output []byte) (int, error) {
	if len(output) == 0 {
		return 0, nil
	}
	count := min(len(output), len(body.state.buffer))
	if err := body.call(diagnosticWinRead, uintptr(unsafe.Pointer(&body.state.buffer[0])), uintptr(count), 0); err != nil {
		return 0, err
	}
	event, err := body.wait(diagnosticWinReadComplete)
	if err != nil {
		return 0, err
	}
	if event.count == 0 {
		return 0, io.EOF
	}
	if event.count > uint32(count) {
		return 0, errors.New("invalid native read length")
	}
	return copy(output, body.state.buffer[:event.count]), nil
}

func (body *diagnosticWinBody) queryHeader(name string, index *uint32) (string, bool, error) {
	key, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return "", false, err
	}
	buffer := make([]uint16, 4096)
	size := uint32(len(buffer) * 2)
	err = body.call(diagnosticWinQueryHeaders, 65535, uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(index)))
	if err == syscall.Errno(12150) {
		return "", false, nil
	}
	if err != nil || size > uint32(len(buffer)*2) {
		return "", false, errors.New("native response headers invalid")
	}
	return windows.UTF16ToString(buffer[:size/2]), true, nil
}

func (transport *diagnosticWinTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport.used || request.Method != http.MethodGet || request.URL.String() != transport.target || request.Body != nil {
		return nil, errors.New("diagnostic native request boundary")
	}
	transport.used = true
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	method, _ := windows.UTF16PtrFromString("GET")
	path, _ := windows.UTF16PtrFromString(request.URL.RequestURI())
	flags := uintptr(0)
	if request.URL.Scheme == "https" {
		flags = 0x00800000
	}
	handle, _, err := diagnosticWinOpenRequest.Call(transport.connection, uintptr(unsafe.Pointer(method)), uintptr(unsafe.Pointer(path)), 0, 0, 0, flags)
	if handle == 0 {
		return nil, err
	}
	// No redirect, cookies, integrated auth or implicit credential negotiation.
	if diagnosticWinSetDWORD(handle, 63, 1|2|4) != nil || diagnosticWinSetDWORD(handle, 77, 2) != nil {
		diagnosticWinClose.Call(handle)
		return nil, errDiagnosticNativeUnsupported
	}
	authorization := request.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") || strings.ContainsAny(authorization, "\r\n\x00") {
		diagnosticWinClose.Call(handle)
		return nil, errors.New("invalid native authorization")
	}
	headers, err := windows.UTF16FromString("Authorization: " + authorization + "\r\nAccept-Encoding: gzip\r\n")
	if err != nil {
		diagnosticWinClose.Call(handle)
		return nil, err
	}
	state := &diagnosticWinAsync{events: make(chan diagnosticWinEvent, 8), closed: make(chan struct{}), buffer: make([]byte, 32<<10), headers: headers}
	state.pins.Pin(&state.buffer[0])
	state.pins.Pin(&state.headers[0])
	identity := uintptr(diagnosticWinSequence.Add(1))
	diagnosticWinContexts.Store(identity, state)
	body := &diagnosticWinBody{ctx: request.Context(), handle: handle, state: state}
	contextOK, _, _ := diagnosticWinSetOption.Call(handle, 45, uintptr(unsafe.Pointer(&identity)), unsafe.Sizeof(identity))
	if contextOK == 0 {
		diagnosticWinClose.Call(handle)
		diagnosticWinContexts.Delete(identity)
		clear(headers)
		state.pins.Unpin()
		return nil, errDiagnosticNativeUnsupported
	}
	callback, _, _ := diagnosticWinSetCallback.Call(handle, diagnosticWinCallback, diagnosticWinHandleClosing|diagnosticWinHeadersAvailable|diagnosticWinReadComplete|diagnosticWinRequestError|diagnosticWinSendComplete, 0)
	if callback == ^uintptr(0) {
		diagnosticWinClose.Call(handle)
		// Registration failed before asynchronous I/O. No callback owns the buffers.
		diagnosticWinContexts.Delete(identity)
		clear(headers)
		state.pins.Unpin()
		return nil, errDiagnosticNativeUnsupported
	}
	transport.request = body
	body.mu.Lock()
	body.stop = context.AfterFunc(request.Context(), func() { body.Close() })
	body.mu.Unlock()
	success := false
	defer func() {
		if !success {
			body.Close()
		}
	}()
	transport.result.Requests++
	if err = body.call(diagnosticWinSend, uintptr(unsafe.Pointer(&headers[0])), uintptr(len(headers)-1), 0, 0, 0, identity); err != nil {
		return nil, err
	}
	if _, err = body.wait(diagnosticWinSendComplete); err != nil {
		return nil, err
	}
	if err = body.call(diagnosticWinReceive, 0); err != nil {
		return nil, err
	}
	if _, err = body.wait(diagnosticWinHeadersAvailable); err != nil {
		return nil, err
	}
	var status uint32
	size := uint32(4)
	if err = body.call(diagnosticWinQueryHeaders, 19|0x20000000, 0, uintptr(unsafe.Pointer(&status)), uintptr(unsafe.Pointer(&size)), 0); err != nil {
		return nil, err
	}
	var protocol uint32
	size = 4
	if err = body.call(diagnosticWinQueryOption, 134, uintptr(unsafe.Pointer(&protocol)), uintptr(unsafe.Pointer(&size))); err != nil {
		return nil, err
	}
	negotiated := "HTTP/1.1"
	if protocol&1 != 0 {
		negotiated = "HTTP/2.0"
	}
	transport.result.Protocol = negotiated
	transport.result.HTTPStatus = int(status)
	response := &http.Response{StatusCode: int(status), Proto: negotiated, Header: make(http.Header), Body: body, ContentLength: -1, Request: request}
	for _, name := range []string{"Content-Type", "Content-Length", "Content-Range", "Content-Encoding", "X-Request-ID", "X-Oneapi-Request-Id"} {
		var index uint32
		for count := 0; ; count++ {
			value, found, queryErr := body.queryHeader(name, &index)
			if queryErr != nil {
				return nil, queryErr
			}
			if !found {
				break
			}
			if count >= 8 {
				return nil, errors.New("native response headers exceed bound")
			}
			response.Header.Add(name, value)
		}
	}
	if values := response.Header.Values("Content-Length"); len(values) > 0 {
		if len(values) != 1 {
			return nil, errors.New("ambiguous native length")
		}
		response.ContentLength, err = strconv.ParseInt(values[0], 10, 64)
		if err != nil || response.ContentLength < 0 {
			return nil, errors.New("invalid native length")
		}
	}
	// Native media is normally uncompressed. Do not silently compare encoded
	// bytes against Go's transparently decompressed response.
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return nil, errDiagnosticNativeEncoding
	}
	idle := uint32(max(1, transport.budgets.idle.Milliseconds()))
	body.mu.Lock()
	if body.handle == 0 {
		err = context.Canceled
	} else {
		err = diagnosticWinSetDWORD(body.handle, 6, idle)
	}
	body.mu.Unlock()
	if err != nil {
		return nil, err
	}
	success = true
	return response, nil
}
