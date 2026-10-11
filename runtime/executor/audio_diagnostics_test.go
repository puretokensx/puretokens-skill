package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAudioValidationReportsActionableSafeFieldsBeforeCredentials(t *testing.T) {
	for _, tc := range []struct {
		name, field, reason string
		limit               int
		change              func(map[string]any)
	}{
		{"voice", "voice", "unsupported", 0, func(r map[string]any) { r["voice"] = "PRIVATE-VOICE-INPUT" }},
		{"length", "input", "max_characters", 1000, func(r map[string]any) { r["input"] = strings.Repeat("语", 1001) }},
		{"speed", "speed", "out_of_range", 0, func(r map[string]any) { r["speed"] = 3 }},
		{"format", "response_format", "unsupported", 0, func(r map[string]any) { r["response_format"] = "PRIVATE-FORMAT" }},
		{"unknown-field", "request", "undeclared_field", 0, func(r map[string]any) { r["PRIVATE-FIELD"] = "PRIVATE-VALUE" }},
		{"null", "speed", "invalid_type", 0, func(r map[string]any) { r["speed"] = nil }},
		{"model", "model", "unsupported", 0, func(r map[string]any) { r["model"] = "PRIVATE-MODEL" }},
		{"operation", "operation", "model_mismatch", 0, func(r map[string]any) { r["operation"] = "transcribe" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := map[string]any{"operation": "speech", "model": "stepaudio-2.5-tts", "input": "PRIVATE-INPUT", "voice": "cixingnansheng", "response_format": "mp3", "output_dir": t.TempDir()}
			tc.change(request)
			data, _ := json.Marshal(request)
			file := filepath.Join(t.TempDir(), "request.json")
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			// No adapter for this host: reaching credential resolution would return a
			// credential error, proving local validation did not short-circuit.
			if err := runAudioCommand(&output, service{}, "no-such-host", file); err == nil {
				t.Fatal("invalid request accepted")
			}
			var got audioReceipt
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Validation == nil || got.Validation.Field != tc.field || got.Validation.Reason != tc.reason || got.Validation.Limit != tc.limit || got.SubmissionOutcome != "not_submitted" || got.LocalErrorCode != "audio_request_invalid" {
				t.Fatalf("unexpected safe validation: %+v", got)
			}
			if strings.Contains(output.String(), "PRIVATE-") || strings.Contains(output.String(), "语") {
				t.Fatal("request data leaked")
			}
		})
	}
}

func TestAudioValidationUnicodeBoundary(t *testing.T) {
	r := map[string]any{"operation": "speech", "model": "stepaudio-2.5-tts", "input": strings.Repeat("语", 1000), "voice": "cixingnansheng", "response_format": "mp3", "output_dir": t.TempDir()}
	data, _ := json.Marshal(r)
	if _, err := decodeAudioRequest(data); err != nil {
		t.Fatalf("unicode character limit counted bytes: %v", err)
	}
}
