package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

const asyncAudioPath = "/v1/tasks/product-yutoux"

func isTaskAudio(kind string) bool { return kind == "music" || kind == "audio_task" }

func asyncAudioFormat(model string) string {
	if model == "gemini-3.8-flash-tts" {
		return "wav"
	}
	return "mp3"
}

func validateAsyncAudioRequest(r taskRequest) error {
	invalid := errors.New("Use a reviewed asynchronous audio model and its declared text parameters, one output and no attachments.")
	if !contains([]string{"gemini-3.8-flash-tts", "lyria-3.5", "lyria-3-pro-preview"}, r.Model) || r.MediaOperation != "" || len(r.Attachments) != 0 || r.Index != 0 || r.RequestedCount > 1 {
		return invalid
	}
	if r.Operation == "continue" {
		if r.Prompt != "" || (r.OriginalOperation != "" && r.OriginalOperation != "generate") {
			return invalid
		}
		for k, v := range r.Parameters {
			if k != "response_format" || v != asyncAudioFormat(r.Model) {
				return invalid
			}
		}
		return nil
	}
	validText := func(v any, max int, required bool) bool {
		s, ok := v.(string)
		return ok && utf8.ValidString(s) && utf8.RuneCountInString(s) <= max && (!required || strings.TrimSpace(s) != "")
	}
	if r.Operation != "generate" || !validText(r.Prompt, 8000, true) {
		return invalid
	}
	speech := r.Model == "gemini-3.8-flash-tts"
	for key, value := range r.Parameters {
		switch key {
		case "temperature":
			n, ok := number(value)
			if !ok || n < 0 || n > 2 {
				return invalid
			}
		case "response_format":
			if value != asyncAudioFormat(r.Model) {
				return invalid
			}
		case "voice":
			if !speech || !validText(value, 120, true) {
				return invalid
			}
		case "instructions":
			if !speech || !validText(value, 4000, false) {
				return invalid
			}
		case "lyrics":
			if speech || !validText(value, 8000, false) {
				return invalid
			}
		case "speakers":
			if !speech {
				return invalid
			}
			speakers, ok := value.([]any)
			if !ok || len(speakers) != 2 {
				return invalid
			}
			names := map[string]bool{}
			for _, item := range speakers {
				speaker, ok := item.(map[string]any)
				if !ok || len(speaker) != 2 || !validText(speaker["name"], 120, true) || !validText(speaker["voice"], 120, true) {
					return invalid
				}
				name := speaker["name"].(string)
				if names[name] {
					return invalid
				}
				names[name] = true
			}
		default:
			return invalid
		}
	}
	return nil
}

func prepareAsyncAudioRequest(r *taskRequest) error {
	if err := validateAsyncAudioRequest(*r); err != nil {
		return err
	}
	if err := prepareTaskAudioOutput(r); err != nil {
		return err
	}
	if r.Parameters == nil {
		r.Parameters = map[string]any{}
	}
	r.Parameters["response_format"] = asyncAudioFormat(r.Model)
	return nil
}

func asyncAudioRequestBody(r taskRequest) (string, string, io.Reader, error) {
	if err := validateAsyncAudioRequest(r); err != nil {
		return "", "", nil, err
	}
	body := map[string]any{"model": r.Model}
	field := "prompt"
	if r.Model == "gemini-3.8-flash-tts" {
		field = "input"
	}
	body[field] = r.Prompt
	for k, v := range r.Parameters {
		if k != "response_format" || field == "input" {
			body[k] = v
		}
	}
	data, err := json.Marshal(body)
	return asyncAudioPath, "application/json", bytes.NewReader(data), err
}

// Native submit omits platform; same-task reads must prove the task family.
func parseKindTaskIdentity(kind string, body []byte, submission bool) (string, string, bool, string) {
	id, state, reconcile, code := parseTaskIdentity(body)
	if code != "" || kind != "audio_task" {
		return id, state, reconcile, code
	}
	var raw map[string]any
	if json.Unmarshal(body, &raw) != nil {
		return "", "", false, "task_response_unreadable"
	}
	if (raw["platform"] != nil || !submission) && raw["platform"] != "product-yutoux" {
		return "", "", false, "task_platform_mismatch"
	}
	if raw["id"] != nil && raw["id"] != id {
		return "", "", false, "task_id_invalid"
	}
	switch state {
	case "failure":
		state = "failed"
	case "not_start", "submitted":
		state = "queued"
	}
	return id, state, reconcile, code
}
