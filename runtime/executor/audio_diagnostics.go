package main

import "fmt"

// All fields originate in reviewed local validation, never caller field names,
// supplied values or server text. This object is not part of support exports.
type audioValidation struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
	Limit  int    `json:"limit,omitempty"`
}

func audioIssue(field, reason string, limit int) *audioValidation {
	return &audioValidation{Field: field, Reason: reason, Limit: limit}
}

func (issue *audioValidation) Error() string { return "audio request validation failed" }

func (issue *audioValidation) message() string {
	if issue.Reason == "max_characters" {
		return fmt.Sprintf("The %s field exceeds the limit of %d Unicode characters. No API request was sent.", issue.Field, issue.Limit)
	}
	if issue.Field == "voice" {
		return "The selected model requires a declared preset or an explicit permitted voice ID according to its installed profile. No API request was sent."
	}
	if issue.Field == "speed" {
		if issue.Reason == "unsupported" {
			return "Speech speed control is not supported by the selected model. Check its installed profile. No API request was sent."
		}
		return "Speech speed must be between 0.5 and 2. No API request was sent."
	}
	if issue.Field == "response_format" {
		return "The selected model does not support this audio output format. Use its installed profile. No API request was sent."
	}
	return fmt.Sprintf("The %s field failed local validation (%s). Use the selected audio profile and declared request fields. No API request was sent.", issue.Field, issue.Reason)
}
