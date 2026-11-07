package main

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Keep exact-model safety checks independent of installed/live schema completeness.
func validateReviewedKlingRequest(r taskRequest) error {
	variant := reviewedKlingVariant(r.Model)
	if variant == "" {
		return nil
	}
	if hasKlingNativeEnvelope(r) {
		_, err := validateKlingNativeRequest(r, variant)
		return err
	}
	counts := map[string]int{}
	for _, file := range r.Attachments {
		counts[file.Field]++
	}
	if err := validateKlingParameters(variant, r.Parameters, counts); err != nil {
		return err
	}
	return validateKlingPrompt(variant, r.Prompt, r.Parameters["first_frame_image"] != nil || counts["first_frame_image"] > 0)
}

func reviewedKlingVariant(model string) string {
	switch model {
	case "kling-video-v4":
		return "v4"
	case "kling-video-v3":
		return "v3"
	case "kling-video-v3-turbo":
		return "turbo"
	case "kling-video-v3-omni":
		return "omni"
	case "kling-video-motion-control-v3.0", "kling-video-motion-control-v2.6":
		return "motion"
	}
	return ""
}

func validateKlingPrompt(variant, prompt string, firstFrame bool) error {
	maxPrompt := 3072
	if variant == "motion" || variant == "turbo" || (variant == "v3" && firstFrame) {
		maxPrompt = 2500
	}
	if variant == "v4" {
		latin, other := 0, 0
		for _, c := range prompt {
			if c < 128 {
				latin++
			} else {
				other++
			}
		}
		if latin*11+other*26 <= 286000 {
			return nil
		}
	} else if utf8.RuneCountInString(prompt) <= maxPrompt {
		return nil
	}
	return errors.New("The prompt exceeds this model's limit for the selected operation. Shorten it explicitly before submitting.")
}

func validateKlingParameters(variant string, p map[string]any, counts map[string]int) error {
	invalid := errors.New("The supplied frames, references, audio, ratio or multi_shot combination is outside this model's reviewed contract. No task was created.")
	count := func(key string) int {
		if counts[key] > 0 {
			return counts[key]
		}
		if _, ok := p[key].(string); ok {
			return 1
		}
		return len(array(p[key]))
	}
	first, last := count("first_frame_image"), count("last_frame_image")
	images, videos, voices := count("reference_images"), count("reference_videos"), count("reference_audio_voice_ids")
	if first > 1 || last > 1 || (last > 0 && first == 0) {
		return invalid
	}
	if value, exists := p["reference_audio_voice_ids"]; exists {
		ids, ok := value.([]any)
		if !ok || (variant != "v4" && variant != "omni") {
			return invalid
		}
		seen := map[string]bool{}
		for _, value := range ids {
			id, ok := value.(string)
			if !ok || id == "" || len(id) > 128 || strings.IndexFunc(id, func(c rune) bool { return unicode.IsSpace(c) || unicode.IsControl(c) }) >= 0 || seen[id] {
				return invalid
			}
			seen[id] = true
		}
	}
	if variant == "v4" {
		if images+first+last > 10 || videos > 5 || voices > 3 || images+first+last+videos+voices > 15 {
			return invalid
		}
		references := images+videos+voices > 0
		if p["aspect_ratio"] == "adaptive" && !references {
			return invalid
		}
		if first > 0 && !references && p["aspect_ratio"] != nil {
			return invalid
		}
		return nil
	}
	if variant == "motion" {
		if images != 1 || videos != 1 || first+last+voices > 0 || p["duration"] != nil || p["aspect_ratio"] != nil {
			return invalid
		}
		return nil
	}
	if (first > 0 || videos > 0) && p["aspect_ratio"] != nil {
		return invalid
	}
	if variant != "omni" {
		if images+videos+voices > 0 {
			return invalid
		}
		return nil
	}
	base := p["reference_video_mode"] == "base"
	maxImages := 7
	if videos > 0 {
		maxImages = 4
	}
	if videos > 1 || images > maxImages || voices > 2 || (videos > 0 && (last > 0 || (base && first > 0))) {
		return invalid
	}
	audio := p["audio"]
	if audio == nil {
		if base {
			audio = "original"
		} else {
			audio = "off"
		}
	}
	multi := p["multi_shot"]
	if multi == nil {
		multi = !base
	}
	if voices > 0 && audio == "off" {
		return invalid
	}
	if videos > 0 {
		if base {
			if (audio != "original" && audio != "off") || multi != false {
				return invalid
			}
		} else if audio != "off" || multi != true {
			return invalid
		}
	} else if audio == "original" || p["reference_video_mode"] != nil {
		return invalid
	}
	return nil
}
