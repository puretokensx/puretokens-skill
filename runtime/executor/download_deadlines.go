package main

import (
	"context"
	"errors"
	"io"
	"time"
)

const (
	contentResponseTimeout = 90 * time.Second
	contentReadIdleTimeout = 30 * time.Second
)

var (
	errContentResponseTimeout = errors.New("content response deadline exceeded")
	errContentIdleTimeout     = errors.New("content byte progress deadline exceeded")
)

func (svc service) contentResponseBudget() time.Duration {
	if svc.downloadResponseTimeout > 0 {
		return svc.downloadResponseTimeout
	}
	return contentResponseTimeout
}

func (svc service) contentIdleBudget() time.Duration {
	if svc.downloadIdleTimeout > 0 {
		return svc.downloadIdleTimeout
	}
	return contentReadIdleTimeout
}

// Wait for a fired callback before inspecting the cause; Stop alone does not
// guarantee that cancellation has completed at the deadline boundary.
func contentDeadline(cancel context.CancelCauseFunc, budget time.Duration, cause error) func() {
	done := make(chan struct{})
	timer := time.AfterFunc(budget, func() {
		cancel(cause)
		close(done)
	})
	return func() {
		if !timer.Stop() {
			<-done
		}
	}
}

type contentProgressReader struct {
	reader    io.Reader
	ctx       context.Context
	cancel    context.CancelCauseFunc
	budget    time.Duration
	remaining time.Duration
}

func (reader *contentProgressReader) Read(buffer []byte) (int, error) {
	if err := context.Cause(reader.ctx); err != nil {
		return 0, err
	}
	if reader.remaining <= 0 {
		reader.cancel(errContentIdleTimeout)
		return 0, context.Cause(reader.ctx)
	}
	started := time.Now()
	stop := contentDeadline(reader.cancel, reader.remaining, errContentIdleTimeout)
	n, err := reader.reader.Read(buffer)
	stop()
	if cause := context.Cause(reader.ctx); cause != nil {
		return n, cause
	}
	if n > 0 {
		reader.remaining = reader.budget
	} else {
		reader.remaining -= time.Since(started)
	}
	return n, err
}
