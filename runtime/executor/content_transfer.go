package main

import (
	"io"
	"net/http"
	"time"
)

// Numeric evidence for one response body, never a task record or remote export.
// Received bytes are not proof of a verified file or host attachment delivery.
type contentTransfer struct {
	BytesReceived  int64  `json:"bytes_received"`
	ExpectedBytes  *int64 `json:"expected_bytes,omitempty"`
	FirstByteMS    *int64 `json:"first_byte_ms,omitempty"`
	LastProgressMS *int64 `json:"last_progress_ms,omitempty"`
	ReadWaitMS     int64  `json:"read_wait_ms"`
	ReadComplete   bool   `json:"read_complete"`
	FileVerified   bool   `json:"file_verified"`
	started        time.Time
	readWait       time.Duration
}

func newContentTransfer(response *http.Response) *contentTransfer {
	transfer := &contentTransfer{started: time.Now()}
	if response.ContentLength >= 0 && !response.Uncompressed {
		length := response.ContentLength
		transfer.ExpectedBytes = &length
	}
	return transfer
}

func (transfer *contentTransfer) observeRead(n int, err error, wait time.Duration) {
	if transfer == nil {
		return
	}
	transfer.readWait += wait
	transfer.ReadWaitMS = transfer.readWait.Milliseconds()
	if n > 0 {
		elapsed := time.Since(transfer.started).Milliseconds()
		if transfer.FirstByteMS == nil {
			transfer.FirstByteMS = &elapsed
		}
		transfer.LastProgressMS = &elapsed
		transfer.BytesReceived += int64(n)
	}
	if err == io.EOF {
		transfer.ReadComplete = true
	}
}
