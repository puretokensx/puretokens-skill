//go:build content_diagnostic && !windows

package main

import "net/http"

func newDiagnosticNativeTransport(string, *contentDiagnostic, diagnosticBudgets) (http.RoundTripper, func(), error) {
	return nil, nil, errDiagnosticNativeUnsupported
}
