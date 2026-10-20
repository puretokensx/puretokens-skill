package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

// Source-reviewed compatibility is separate from public availability. Embed
// these profiles so missing/stale installed files cannot bypass their limits.
// Existing-task reads never consult these submission profiles.
//
//go:embed reviewed-video-profiles.json
var reviewedVideoProfileJSON []byte

var reviewedVideoProfiles = sync.OnceValue(func() map[string]modelProfile {
	var document struct {
		Models []modelProfile `json:"models"`
	}
	if json.Unmarshal(reviewedVideoProfileJSON, &document) != nil {
		panic("Invalid reviewed video profiles")
	}
	profiles := make(map[string]modelProfile)
	for _, profile := range document.Models {
		profiles[profile.ID] = profile
	}
	return profiles
})

func reviewedVideoProfile(id string) (modelProfile, bool) {
	profile, exists := reviewedVideoProfiles()[id]
	return profile, exists
}

func reviewedVideoAllowsEmptyPrompt(request taskRequest) bool {
	profile, exists := reviewedVideoProfile(request.Model)
	if !exists || request.Kind != "video" || request.Operation != "generate" ||
		profile.Parameters.Properties["prompt"]["required"] != false {
		return false
	}
	operation, exists := profile.Parameters.Operations[request.MediaOperation]
	return exists && !contains(operation.Required, "prompt")
}

func prepareReviewedVideoOperation(request *taskRequest) error {
	profile, exists := reviewedVideoProfile(request.Model)
	if !exists || request.Kind != "video" {
		return nil
	}
	if strings.TrimSpace(request.Prompt) == "" && !reviewedVideoAllowsEmptyPrompt(*request) {
		return errors.New("This model operation requires the user's prompt.")
	}
	if request.MediaOperation != "" || request.Operation != "generate" {
		return nil
	}
	if _, exists := profile.Parameters.Operations["video_generation"]; !exists {
		return errors.New("This model requires a declared image/reference operation and its inputs; text-only video generation is unsupported.")
	}
	request.MediaOperation = "video_generation"
	return nil
}
