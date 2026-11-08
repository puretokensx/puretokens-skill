package main

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

var errWanNative = errors.New("The native video input is outside this exact model's reviewed contract. Use declared public HTTPS media roles and settings without mixing flat inputs. No task was created.")

func reviewedWanModel(model string) bool {
	return model == "wan3.0-video" || model == "wan3.0-video-prime"
}

func hasWanNativeEnvelope(r taskRequest) bool {
	if r.MediaOperation == "native_generation" {
		return true
	}
	if !reviewedWanModel(r.Model) {
		return false
	}
	for _, key := range []string{"input", "parameters"} {
		if _, exists := r.Parameters[key]; exists {
			return true
		}
	}
	return false
}

func prepareWanNativeRequest(r *taskRequest) error {
	if !hasWanNativeEnvelope(*r) {
		return nil
	}
	prompt, err := validateWanNativeRequest(*r)
	if err != nil {
		return err
	}
	r.Prompt, r.MediaOperation = prompt, "native_generation"
	return nil
}

func reviewedWanAllowsNativePrompt(r taskRequest) bool {
	if !reviewedWanModel(r.Model) || !hasWanNativeEnvelope(r) {
		return false
	}
	_, err := validateWanNativeRequest(r)
	return err == nil
}

func validateWanNativeRequest(r taskRequest) (string, error) {
	if !reviewedWanModel(r.Model) || r.Kind != "video" || r.Operation != "generate" ||
		(r.MediaOperation != "" && r.MediaOperation != "native_generation") || len(r.Attachments) != 0 || r.RequestedCount < 0 || r.RequestedCount > 1 {
		return "", errWanNative
	}
	for key := range r.Parameters {
		if key != "input" && key != "parameters" {
			return "", errWanNative
		}
	}
	input, ok := r.Parameters["input"].(map[string]any)
	if !ok || input == nil {
		return "", errWanNative
	}
	for key := range input {
		if key != "prompt" && key != "media" {
			return "", errWanNative
		}
	}
	prompt := r.Prompt
	if value, exists := input["prompt"]; exists {
		text, valid := value.(string)
		if !valid || (prompt != "" && prompt != text) {
			return "", errWanNative
		}
		prompt = text
	}
	if utf8.RuneCountInString(prompt) > 20000 {
		return "", errWanNative
	}
	counts := map[string]int{}
	total := 0
	if value, exists := input["media"]; exists {
		media, valid := value.([]any)
		if !valid || len(media) > 21 {
			return "", errWanNative
		}
		total = len(media)
		for _, value := range media {
			item, valid := value.(map[string]any)
			if !valid || len(item) != 2 {
				return "", errWanNative
			}
			kind, valid := item["type"].(string)
			if !valid || !contains([]string{"first_frame", "last_frame", "reference_image", "reference_video", "reference_audio", "file", "link"}, kind) {
				return "", errWanNative
			}
			url, valid := item["url"].(string)
			if !valid || len(url) > 8192 || validateReferences(url, []any{"public_https_url"}) != nil {
				return "", errWanNative
			}
			counts[kind]++
		}
	}
	first, last := counts["first_frame"], counts["last_frame"]
	images, videos, audios := counts["reference_image"], counts["reference_video"], counts["reference_audio"]
	documents := counts["file"] + counts["link"]
	if (strings.TrimSpace(prompt) == "" && total == 0) || first > 1 || last > 1 || (last > 0 && first == 0) ||
		(first+last > 0 && total != first+last) || images > 10 || videos > 5 || audios > 5 || images+videos+audios > 20 || documents > 1 {
		return "", errWanNative
	}
	settings := map[string]any{}
	if value, exists := r.Parameters["parameters"]; exists {
		settings, ok = value.(map[string]any)
		if !ok || settings == nil {
			return "", errWanNative
		}
	}
	for key, value := range settings {
		switch key {
		case "duration":
			n, valid := number(value)
			if !valid || math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || (n != -1 && (n < 2 || n > 30)) {
				return "", errWanNative
			}
		case "resolution":
			text, valid := value.(string)
			if !valid || !contains([]string{"480P", "720P", "1080P"}, strings.ToUpper(text)) {
				return "", errWanNative
			}
		case "ratio":
			text, valid := value.(string)
			if !valid || !contains([]string{"adaptive", "21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}, text) {
				return "", errWanNative
			}
		case "audio", "prompt_extend", "watermark":
			if _, valid := value.(bool); !valid {
				return "", errWanNative
			}
		default:
			return "", errWanNative
		}
	}
	// File/link processing requires prompt extension; never silently override
	// an explicit false or rely on a hidden gateway default for this operation.
	if documents > 0 && settings["prompt_extend"] != true {
		return "", errWanNative
	}
	return prompt, nil
}
