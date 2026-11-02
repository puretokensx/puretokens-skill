package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"os"
)

// Classify typed local errors only. Error strings can contain URLs, proxy
// details, credentials or file paths and must never become receipt evidence.
func transportErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "api_request_cancelled"
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "api_dns_failed"
	}
	var verification *tls.CertificateVerificationError
	var record tls.RecordHeaderError
	var alert tls.AlertError
	var authority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var certificate x509.CertificateInvalidError
	if errors.As(err, &verification) || errors.As(err, &record) || errors.As(err, &alert) ||
		errors.As(err, &authority) || errors.As(err, &hostname) || errors.As(err, &certificate) {
		return "api_tls_failed"
	}
	if errors.Is(err, os.ErrPermission) {
		return "api_network_permission_denied"
	}
	var timed net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timed) && timed.Timeout()) {
		return "api_request_timeout"
	}
	var connection *net.OpError
	if errors.As(err, &connection) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return "api_connection_failed"
	}
	return "api_transport_failed"
}

func withTransportFailure(result receipt, err error) receipt {
	if err == nil || result.HTTPStatus != 0 {
		return result
	}
	result.LocalErrorCode = transportErrorCode(err)
	result.ErrorMessage = "The HTTP exchange failed before a response was available; the underlying network cause is unclassified."
	result.NextAction = "Check this host's network access or contact Pure Tokens support with the existing support summary. Do not infer a credential, model or balance problem."
	switch result.LocalErrorCode {
	case "api_request_timeout":
		result.ErrorMessage = "The HTTP request timed out before a response was available."
		result.NextAction = "Use this invocation's existing timings_ms.request if available. A timeout does not establish whether the server received or accepted the request; contact Pure Tokens support before creating another task."
	case "api_request_cancelled":
		result.ErrorMessage = "The HTTP request was cancelled before a response was available."
	case "api_dns_failed":
		result.ErrorMessage = "Name resolution failed or timed out before an HTTP response was available."
	case "api_tls_failed":
		result.ErrorMessage = "TLS negotiation or certificate verification failed before an HTTP response was available."
		result.NextAction = "Check the host's network path and certificate trust. Do not disable TLS verification or change the API transport."
	case "api_network_permission_denied":
		result.ErrorMessage = "The operating system denied a network operation before an HTTP response was available."
		result.NextAction = "Check whether this host permits the executor's network access. Obtain only the required permission; do not bypass sandbox policy or assume Full Disk Access is required."
	case "api_connection_failed":
		result.ErrorMessage = "The connection failed or closed before an HTTP response was available."
	}
	return result
}
