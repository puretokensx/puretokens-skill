//go:build content_diagnostic

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestContentDiagnosticOfflineSelfTestEntrypoint(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", t.TempDir())
	for _, args := range [][]string{
		{"self-test"},
		{"self-test", "--confirm-one-get"},
		{"self-test", "-test.run=.*"},
		{"self-test", "content-diagnose"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal("offline self-test did not finish within its bound")
			}
			if runtime.GOOS == "windows" && len(args) == 1 {
				if err != nil || !strings.Contains(string(output), "--- PASS: TestContentWinHTTPNativeFixtures") ||
					!strings.Contains(string(output), "--- PASS: TestContentWinHTTPNativeBlocksReplayAndPOST") {
					t.Fatalf("native-only self-test failed: %v\n%s", err, output)
				}
				return
			}
			var receipt struct {
				OK       bool   `json:"ok"`
				Code     string `json:"local_error_code"`
				Requests *int   `json:"requests"`
			}
			want := "diagnostic_request_invalid"
			if len(args) == 1 {
				want = "diagnostic_transport_unsupported"
			}
			if err == nil || json.Unmarshal(output, &receipt) != nil || receipt.OK || receipt.Code != want ||
				receipt.Requests == nil || *receipt.Requests != 0 {
				t.Fatalf("self-test did not stop safely: err=%v output=%s", err, output)
			}
		})
	}
}
