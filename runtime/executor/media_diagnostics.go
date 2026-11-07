package main

import (
	"errors"
	"fmt"
)

// Diagnostic fields and choices are local constants, never caller or catalog text.
type mediaValidation struct {
	Field   string   `json:"field"`
	Reason  string   `json:"reason"`
	Allowed []string `json:"allowed,omitempty"`
}

func (issue *mediaValidation) Error() string {
	return fmt.Sprintf("The %s field failed local validation (%s). Check the selected model profile.", issue.Field, issue.Reason)
}

func mediaPropertyIssue(name, reason string, rule map[string]any) error {
	field := "parameters"
	if contains([]string{"prompt", "quality", "size", "n", "duration", "aspect_ratio", "resolution", "output_format", "background", "seed"}, name) {
		field = name
	}
	issue := &mediaValidation{Field: field, Reason: reason}
	if reason == "enum" {
		choices := map[string][]string{
			"quality":       {"auto", "low", "medium", "high"},
			"output_format": {"png", "jpeg", "webp", "gif", "svg"},
			"background":    {"auto", "transparent", "opaque"},
		}[field]
		for _, choice := range choices {
			if hasValue(array(rule["enum"]), choice) {
				issue.Allowed = append(issue.Allowed, choice)
			}
		}
	}
	return issue
}

func mediaRequestFailure(request taskRequest, err error) receipt {
	result := validationFailure(err.Error())
	if (request.Kind != "image" && request.Kind != "video") || (request.Operation != "generate" && request.Operation != "edit") || request.TaskID != "" || request.TaskStatus != "" || request.OriginalOperation != "" || request.RetryNotBefore != "" || request.ReconciliationRequired || request.Index != 0 || request.WaitWindowsCompleted != 0 {
		return result
	}
	var issue *mediaValidation
	if !errors.As(err, &issue) {
		return result
	}
	result.Kind = request.Kind
	result.Operation = request.Operation
	result.SubmissionOutcome = "not_submitted"
	result.LocalErrorCode = "media_request_invalid"
	result.Validation = issue
	result.ErrorMessage += " No generation request was sent."
	result.NextAction = "Correct only an agent-authored request assembly error once, preserving the user's model, content, attachments and output scope. If the user must choose or intent would change, ask the user instead. This is not permission to retry an existing or uncertain task."
	return result
}
