package main

import (
	"errors"
	"math"
	"strings"
	"unicode"
)

var errKlingNative = errors.New("The native video envelope is outside this exact model's reviewed contract. Use declared content roles, existing resource IDs and settings without mixing flat inputs. No task was created.")

func hasKlingNativeEnvelope(r taskRequest) bool {
	if r.MediaOperation == "native_video" {
		return true
	}
	for _, key := range []string{"contents", "settings", "options"} {
		if _, exists := r.Parameters[key]; exists {
			return true
		}
	}
	return false
}

func prepareKlingNativeRequest(r *taskRequest) error {
	variant := reviewedKlingVariant(r.Model)
	if !hasKlingNativeEnvelope(*r) || (variant == "" && r.MediaOperation != "native_video") {
		return nil
	}
	prompt, err := validateKlingNativeRequest(*r, variant)
	if err != nil {
		return err
	}
	r.Prompt = prompt
	r.MediaOperation = "native_video"
	return nil
}

// This grants only the prompt-source exception; the complete envelope must pass.
func reviewedKlingAllowsNativePrompt(r taskRequest) bool {
	variant := reviewedKlingVariant(r.Model)
	if variant == "" || !hasKlingNativeEnvelope(r) {
		return false
	}
	_, err := validateKlingNativeRequest(r, variant)
	return err == nil
}

func validateKlingNativeRequest(r taskRequest, variant string) (string, error) {
	if variant == "" || r.Kind != "video" || r.Operation != "generate" ||
		(r.MediaOperation != "" && r.MediaOperation != "native_video") || len(r.Attachments) != 0 || r.RequestedCount < 0 || r.RequestedCount > 1 {
		return "", errKlingNative
	}
	// The gateway replaces whole flat groups when native values exist. Disallow
	// mixed representations rather than silently dropping any caller intent.
	for key := range r.Parameters {
		if !contains([]string{"contents", "settings", "options"}, key) {
			return "", errKlingNative
		}
	}
	contents, ok := r.Parameters["contents"].([]any)
	if !ok || len(contents) == 0 || len(contents) > 16 {
		return "", errKlingNative
	}
	types := []string{"prompt", "first_frame", "last_frame", "element"}
	switch variant {
	case "v4":
		types = append(types, "refer_image", "feature_video", "voice")
	case "omni":
		types = append(types, "refer_image", "feature_video", "base_video", "voice")
	case "turbo":
		types = []string{"prompt", "first_frame"}
	case "motion":
		types = []string{"prompt", "image", "video"}
		if r.Model == "kling-video-motion-control-v3.0" {
			types = append(types, "element")
		}
	}
	counts, ids := map[string]int{}, map[string]bool{}
	prompt := r.Prompt
	for _, value := range contents {
		item, ok := value.(map[string]any)
		if !ok || item == nil {
			return "", errKlingNative
		}
		kind, ok := item["type"].(string)
		if !ok || !contains(types, kind) {
			return "", errKlingNative
		}
		fields := []string{"type", "url"}
		switch kind {
		case "prompt":
			fields = []string{"type", "text"}
		case "element":
			fields = []string{"type", "element_id"}
			if variant == "motion" {
				fields = append(fields, "id")
			}
		case "voice":
			fields = []string{"type", "voice_id", "id"}
		}
		if len(item) != len(fields) {
			return "", errKlingNative
		}
		for key := range item {
			if !contains(fields, key) {
				return "", errKlingNative
			}
		}
		counts[kind]++
		switch kind {
		case "prompt":
			text, ok := item["text"].(string)
			if !ok || strings.TrimSpace(text) == "" || counts[kind] > 1 || (r.Prompt != "" && r.Prompt != text) {
				return "", errKlingNative
			}
			prompt = text
		case "element", "voice":
			if !klingNativeIdentifier(item[kind+"_id"]) {
				return "", errKlingNative
			}
			if kind == "voice" || variant == "motion" {
				id, ok := item["id"].(string)
				if !ok || !klingNativeIdentifier(id) || ids[id] {
					return "", errKlingNative
				}
				ids[id] = true
			}
		default:
			url, ok := item["url"].(string)
			if !ok || len(url) > 8192 || validateReferences(url, []any{"public_https_url"}) != nil {
				return "", errKlingNative
			}
		}
	}
	if variant != "motion" && strings.TrimSpace(prompt) == "" {
		return "", errKlingNative
	}
	first, last, refs := counts["first_frame"], counts["last_frame"], counts["refer_image"]
	feature, base, elements, voices := counts["feature_video"], counts["base_video"], counts["element"], counts["voice"]
	if first > 1 || last > 1 || (last > 0 && first == 0) {
		return "", errKlingNative
	}
	if variant == "v4" && (first+last+refs > 10 || feature > 5 || elements > 7 || voices > 3 || first+last+refs+feature+elements+voices > 15) {
		return "", errKlingNative
	}
	maxOmniRefs := 7
	if feature+base > 0 {
		maxOmniRefs = 4
	}
	if variant == "omni" && (feature+base > 1 || (feature > 0 && last > 0) || (base > 0 && first+last > 0) || refs+elements > maxOmniRefs || voices > 2) {
		return "", errKlingNative
	}
	if variant == "motion" && (counts["image"] != 1 || counts["video"] != 1 || elements > 1) {
		return "", errKlingNative
	}
	if err := validateKlingPrompt(variant, prompt, first > 0); err != nil {
		return "", err
	}
	settings := map[string]any{}
	if value, exists := r.Parameters["settings"]; exists {
		settings, ok = value.(map[string]any)
		if !ok || settings == nil {
			return "", errKlingNative
		}
	}
	if err := validateKlingNativeSettings(variant, settings, counts); err != nil {
		return "", err
	}
	if value, exists := r.Parameters["options"]; exists {
		options, ok := value.(map[string]any)
		if !ok || options == nil {
			return "", errKlingNative
		}
		for key, value := range options {
			switch key {
			case "external_task_id":
				if !klingNativeIdentifier(value) {
					return "", errKlingNative
				}
			case "watermark_info":
				watermark, ok := value.(map[string]any)
				if !ok || watermark == nil || len(watermark) > 1 {
					return "", errKlingNative
				}
				for name, enabled := range watermark {
					if _, ok := enabled.(bool); name != "enabled" || !ok {
						return "", errKlingNative
					}
				}
			default:
				return "", errKlingNative
			}
		}
	}
	return prompt, nil
}

func klingNativeIdentifier(value any) bool {
	id, ok := value.(string)
	return ok && id != "" && len(id) <= 128 && strings.IndexFunc(id, func(c rune) bool { return unicode.IsSpace(c) || unicode.IsControl(c) }) < 0
}

func validateKlingNativeSettings(variant string, settings map[string]any, counts map[string]int) error {
	allowed := []string{"duration", "resolution", "aspect_ratio"}
	if variant == "motion" {
		allowed = []string{"resolution", "audio", "character_orientation"}
	} else if variant == "v3" || variant == "omni" {
		allowed = append(allowed, "audio", "multi_shot")
	}
	for key, value := range settings {
		if !contains(allowed, key) || value == nil {
			return errKlingNative
		}
		switch key {
		case "duration":
			duration, ok := number(value)
			maxDuration := 15.0
			if variant == "v4" {
				maxDuration = 30
			}
			if !ok || math.IsNaN(duration) || math.IsInf(duration, 0) || duration != math.Trunc(duration) || duration < 3 || duration > maxDuration {
				return errKlingNative
			}
		case "multi_shot":
			if _, ok := value.(bool); !ok {
				return errKlingNative
			}
		default:
			if _, ok := value.(string); !ok {
				return errKlingNative
			}
		}
	}
	resolution, exists := settings["resolution"]
	if exists && resolution != "720p" && resolution != "1080p" && !((variant == "v3" || variant == "omni") && resolution == "4k") {
		return errKlingNative
	}
	if ratio, exists := settings["aspect_ratio"]; exists {
		refs := counts["refer_image"]+counts["feature_video"]+counts["element"]+counts["voice"] > 0
		if variant == "v4" {
			if (counts["first_frame"] > 0 && !refs) || (ratio != "21:9" && ratio != "16:9" && ratio != "9:16" && ratio != "1:1" && !(refs && ratio == "adaptive")) {
				return errKlingNative
			}
		} else if counts["first_frame"]+counts["feature_video"]+counts["base_video"] > 0 || (ratio != "16:9" && ratio != "9:16" && ratio != "1:1") {
			return errKlingNative
		}
	}
	if variant == "v4" || variant == "turbo" {
		return nil
	}
	audio := settings["audio"]
	if audio == nil {
		audio = "off"
		if variant == "motion" || counts["base_video"] > 0 {
			audio = "original"
		}
	}
	allowedAudio := []string{"native", "off"}
	if variant == "motion" || counts["base_video"] > 0 {
		allowedAudio = []string{"original", "off"}
	} else if counts["feature_video"] > 0 {
		allowedAudio = []string{"off"}
	}
	if !contains(allowedAudio, audio.(string)) || (counts["voice"] > 0 && audio == "off") {
		return errKlingNative
	}
	if variant == "motion" {
		orientation := settings["character_orientation"]
		if (orientation != "image" && orientation != "video") || (counts["element"] > 0 && orientation != "video") {
			return errKlingNative
		}
		return nil
	}
	multi := settings["multi_shot"]
	if multi == nil {
		multi = counts["base_video"] == 0
	}
	if (counts["feature_video"] > 0 && multi != true) || (counts["base_video"] > 0 && multi != false) {
		return errKlingNative
	}
	return nil
}
