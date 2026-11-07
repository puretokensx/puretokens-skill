package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

func reviewedTailFrameModel(model string) bool {
	switch model {
	case "Super Vega 1.0-pro", "Super Vega 1.0-pro-fast", "Super Vega 1.0-pro-mini", "Super Vega 1.5-pro":
		return true
	}
	return false
}

func taskHasTailFrame(request taskRequest) bool {
	return request.Kind == "video" && reviewedTailFrameModel(request.Model) && request.Parameters["return_last_frame"] == true
}

// RequestedCount describes generations; a selected tail frame is another artifact.
func taskOutputCount(request taskRequest) int {
	if taskHasTailFrame(request) {
		return 2
	}
	if request.Kind == "video" || isTaskAudio(request.Kind) {
		return 1
	}
	return max(1, request.RequestedCount)
}

func validOutputIndex(request taskRequest, index int) bool {
	return index >= 0 && index < taskOutputCount(request)
}

func artifactKind(kind string, index int) string {
	if kind == "video" && index == 1 {
		return "image"
	}
	return kind
}

func taskArtifactFormatMatches(request taskRequest, index int, mediaType string) bool {
	if request.Kind == "video" && index == 1 {
		return taskHasTailFrame(request) && contains([]string{"image/png", "image/jpeg", "image/webp"}, mediaType)
	}
	return taskImageFormatMatches(request, mediaType)
}

// Only the declared artifact key is trusted, never a returned content URL.
func (svc service) verifyTailFrame(ctx context.Context, id string) (int, int, string, string, error) {
	body, status, retry, code, message, err := svc.request(ctx, http.MethodGet, "/v1/tasks/"+url.PathEscape(id)+"/artifacts", nil, "")
	if err != nil {
		return status, retry, code, message, err
	}
	if status != http.StatusOK {
		return status, retry, code, message, errors.New("artifact manifest rejected")
	}
	var manifest struct {
		TaskID    string `json:"task_id"`
		Artifacts []struct {
			Key  string `json:"key"`
			Type string `json:"type"`
		} `json:"artifacts"`
	}
	if json.Unmarshal(body, &manifest) != nil || manifest.TaskID != id || len(manifest.Artifacts) > 16 {
		return status, retry, "", "", errors.New("artifact identity unverified")
	}
	count := 0
	for _, artifact := range manifest.Artifacts {
		if artifact.Key == "last_frame" {
			if artifact.Type != "image" {
				return status, retry, "", "", errors.New("artifact type unverified")
			}
			count++
		}
	}
	if count != 1 {
		return status, retry, "", "", errors.New("tail frame unavailable or ambiguous")
	}
	return status, retry, "", "", nil
}
