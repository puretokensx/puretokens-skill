package main

import (
	"strings"
	"testing"
)

func TestKlingFlatRequestsAndCombinations(t *testing.T) {
	for _, tc := range []struct {
		model, op, prompt string
		params            map[string]any
		valid             bool
	}{
		{"kling-video-motion-control-v3.0", "reference_video", "", map[string]any{"character_orientation": "video", "audio": "original", "reference_images": []any{"https://example.com/a.png"}, "reference_videos": []any{"https://example.com/v.mp4"}}, true},
		{"kling-video-v3", "video_generation", "test", map[string]any{"audio": "off", "duration": 5}, true},
		{"kling-video-v3-omni", "reference_video", "test", map[string]any{"reference_videos": []any{"https://example.com/v.mp4"}, "audio": "native"}, false},
		{"kling-video-v3-omni", "reference_video", "test", map[string]any{"reference_videos": []any{"https://example.com/v.mp4"}, "multi_shot": false}, false},
		{"kling-video-v3-omni", "reference_video", "test", map[string]any{"reference_videos": []any{"https://example.com/v.mp4"}, "reference_video_mode": "base", "audio": "original", "multi_shot": false}, true},
		{"kling-video-v3-omni", "reference_video", "test", map[string]any{"reference_videos": []any{"https://example.com/v.mp4"}, "reference_video_mode": "base", "multi_shot": true}, false},
		{"kling-video-v4", "video_generation", "test", map[string]any{"last_frame_image": "https://example.com/a.png"}, false},
		{"kling-video-v3-turbo", "video_generation", "test", map[string]any{"contents": []any{map[string]any{"type": "unreviewed"}}}, false},
		{"kling-video-v4", "video_generation", "test", map[string]any{"aspect_ratio": "adaptive"}, false},
		{"kling-video-v4", "reference_video", "test", map[string]any{"reference_videos": []any{"https://example.com/v.mp4"}, "aspect_ratio": "adaptive"}, true},
		{"kling-video-v4", "video_generation", "test", map[string]any{"reference_audio_voice_ids": []any{"voice_a", "voice_a"}}, false},
		{"kling-video-v4", "video_generation", "test", map[string]any{"reference_audio_voice_ids": []any{"voice_a"}, "aspect_ratio": "adaptive"}, true},
		{"kling-video-v3", "image_to_video", strings.Repeat("a", 2501), map[string]any{"first_frame_image": "https://example.com/a.png"}, false},
		{"kling-video-v4", "video_generation", strings.Repeat("a", 26001), map[string]any{}, false},
	} {
		r := taskRequest{Kind: "video", Operation: "generate", Model: tc.model, MediaOperation: tc.op, Prompt: tc.prompt, Parameters: tc.params}
		err := validateTaskRequest(r)
		if err == nil {
			err = prepareProfileRequest(&r, profileService())
		}
		if (err == nil) != tc.valid {
			t.Errorf("%s %v valid=%v: %v", tc.model, tc.params, tc.valid, err)
		}
	}
}
