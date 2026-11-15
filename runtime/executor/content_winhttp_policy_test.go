//go:build content_diagnostic

package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

var diagnosticNativeFactory = newDiagnosticNativeTransport
var errDiagnosticNativeUnsupported = errors.New("diagnostic native transport unsupported")
var errDiagnosticNativeEncoding = errors.New("diagnostic native content encoding unsupported")

// Preserve Go's resolved environment proxy choice; never silently switch to
// Windows system/PAC discovery or borrow proxy credentials.
func diagnosticNativeProxy(proxy *url.URL) (string, string, error) {
	if proxy == nil {
		return "", "direct", nil
	}
	if proxy.Scheme != "http" || proxy.Hostname() == "" || proxy.User != nil ||
		(proxy.Path != "" && proxy.Path != "/") || proxy.RawQuery != "" || proxy.ForceQuery || proxy.Fragment != "" ||
		strings.ContainsAny(proxy.Host, "\r\n; ") {
		return "", "", errDiagnosticNativeUnsupported
	}
	return proxy.Host, "environment_http_proxy", nil
}

func diagnosticProxyForTarget(target string) (string, string, error) {
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", "", errDiagnosticNativeUnsupported
	}
	proxy, err := http.ProxyFromEnvironment(request)
	if err != nil {
		return "", "", errDiagnosticNativeUnsupported
	}
	return diagnosticNativeProxy(proxy)
}
