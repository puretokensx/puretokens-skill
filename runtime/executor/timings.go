package main

import (
	"io"
	"time"
)

// Invocation-local observations. Production time.Now values carry Go's
// monotonic clock; no wall-clock timestamps, paths or input values are retained.
// Phases are exclusive even when a catalog request interrupts preparation or
// media verification interrupts download. Output/host handoff is not measured.
type commandTimings struct {
	now       func() time.Time
	started   time.Time
	last      time.Time
	active    string
	durations map[string]time.Duration
	seen      map[string]bool
}

func newCommandTimings() *commandTimings {
	now := time.Now()
	return &commandTimings{now: time.Now, started: now, last: now, active: "preparation",
		durations: make(map[string]time.Duration), seen: map[string]bool{"preparation": true}}
}

func timingPhaseAllowed(phase string) bool {
	switch phase {
	case "preparation", "catalog", "request", "wait", "download", "verification":
		return true
	}
	return false
}

func (timings *commandTimings) enter(phase string) func() {
	if timings == nil || !timingPhaseAllowed(phase) {
		return func() {}
	}
	previous := timings.active
	timings.transition(phase)
	ended := false
	return func() {
		if !ended {
			ended = true
			timings.transition(previous)
		}
	}
}

func (timings *commandTimings) transition(phase string) {
	now := timings.now()
	timings.durations[timings.active] += max(0, now.Sub(timings.last))
	timings.active, timings.last = phase, now
	timings.seen[phase] = true
}

func (timings *commandTimings) snapshot() map[string]int64 {
	if timings == nil {
		return nil
	}
	now := timings.now()
	result := map[string]int64{"total": max(0, now.Sub(timings.started)).Milliseconds()}
	for phase := range timings.seen {
		duration := timings.durations[phase]
		if phase == timings.active {
			duration += max(0, now.Sub(timings.last))
		}
		result[phase] = duration.Milliseconds()
	}
	return result
}

func timingsForWriter(output io.Writer) *commandTimings {
	switch writer := output.(type) {
	case *supportReceiptWriter:
		return writer.timings
	case *recordReceiptWriter:
		return timingsForWriter(writer.output)
	default:
		return nil
	}
}

func (svc service) timePhase(phase string) func() {
	if svc.support == nil {
		return func() {}
	}
	return svc.support.timings.enter(phase)
}

func timedDownloadProof(timings *commandTimings, path string, proof downloadProof) bool {
	finish := timings.enter("verification")
	defer finish()
	return matchesDownloadProof(path, proof)
}

func (svc service) checkDownloadProof(path string, proof downloadProof) bool {
	finish := svc.timePhase("verification")
	defer finish()
	return matchesDownloadProof(path, proof)
}

func timedMediaFile(svc service, path, mediaType string) bool {
	finish := svc.timePhase("verification")
	defer finish()
	return validMediaFile(path, mediaType)
}
