package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
)

type modelQuery struct {
	Kind       string         `json:"kind,omitempty"`
	Model      string         `json:"model,omitempty"`
	Operation  string         `json:"operation,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// input is nil for an unfiltered catalog read; callers should not pass an open
// terminal stream when no request file was supplied. Filters stay local: the
// only network operation is one GET to the fixed live catalog.
func executeModelQuery(output io.Writer, svc service, input io.Reader) error {
	query, err := decodeModelQuery(input)
	if err != nil {
		writeReceipt(output, validationFailure("Use an optional model filter object with kind, exact model, declared operation and parameters. No API request was sent."))
		return err
	}
	if query.Kind == "evaluation" {
		return executeEvaluationModelQuery(output, svc, query)
	}
	if query.Kind == "audio" {
		return executeAudioModelQuery(output, svc, query)
	}
	if isRetrievalCommand(query.Kind) {
		return executeRetrievalModelQuery(output, svc, query)
	}
	body, status, retry, code, message, err := svc.request(context.Background(), http.MethodGet, "/v1/media/models", nil, "")
	if err != nil || status < 200 || status >= 300 {
		writeReceipt(output, apiFailure("submission", status, retry, code, message, "The live model catalog could not be read. No media task was submitted."))
		return errors.New("model catalog unavailable")
	}
	catalog, err := readAPIObject(body)
	entries, valid := catalog["data"].([]any)
	if err != nil || !valid {
		writeReceipt(output, apiFailure("submission", status, retry, code, message, "The API did not return a readable model catalog. No media task was submitted."))
		return errors.New("invalid model catalog")
	}
	matched := []any{}
	for _, value := range entries {
		entry, ok := value.(map[string]any)
		if ok && modelQueryMatches(query, entry) {
			matched = append(matched, entry)
		}
	}
	// Reuse the established public projection: no raw upstream fields, prices,
	// provider identities or arbitrary catalog metadata enter the receipt.
	projected := projectCatalog(map[string]any{"data": matched}).(map[string]any)
	parameterNames := make([]string, 0, len(query.Parameters))
	for name := range query.Parameters {
		parameterNames = append(parameterNames, name)
	}
	sort.Strings(parameterNames)
	projected["matched_count"] = len(matched)
	projected["filter"] = map[string]any{
		"kind": query.Kind, "model": query.Model, "operation": query.Operation,
		"parameter_names": parameterNames,
	}
	projected["matching_scope"] = "declared_schema_only"
	projected["note"] = "Matches reflect only supplied filters and returned declarations. Required prompts or media, execution authorization, price and output quality are not verified."
	writeJSON(output, map[string]any{"ok": true, "command": "models", "result": projected})
	return nil
}

func decodeModelQuery(input io.Reader) (modelQuery, error) {
	var query modelQuery
	if input == nil {
		return query, nil
	}
	body, err := io.ReadAll(io.LimitReader(input, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return query, errors.New("model filter unreadable")
	}
	body = bytes.TrimSpace(bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf}))
	if len(body) == 0 {
		return query, nil
	}
	if body[0] != '{' {
		return query, errors.New("model filter must be an object")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&query); err != nil {
		return query, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return query, errors.New("model filter must be one object")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return query, errors.New("model filter must be an object")
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return query, errors.New("model filter fields must not be null")
		}
	}
	if query.Kind != "" && query.Kind != "image" && query.Kind != "video" && query.Kind != "evaluation" && query.Kind != "audio" && !isRetrievalCommand(query.Kind) {
		return query, errors.New("unknown media kind")
	}
	if query.Kind == "evaluation" && (query.Operation != "" || len(query.Parameters) > 0) {
		return query, errors.New("evaluation catalog cannot filter undeclared operation parameters")
	}
	if isRetrievalCommand(query.Kind) && (query.Operation != "" || len(query.Parameters) > 0) {
		return query, errors.New("retrieval discovery accepts no operation or parameter filters")
	}
	if query.Kind == "audio" && (len(query.Parameters) > 0 || (query.Operation != "" && query.Operation != "speech" && query.Operation != "transcribe" && query.Operation != "generate" && query.Operation != "music" && query.Operation != "music_sync" && query.Operation != "voice_change")) {
		return query, errors.New("audio discovery accepts only reviewed operations and no parameter filters")
	}
	if query.Model != "" && (!validModelID(query.Model) || safePublicString(query.Model) != query.Model) {
		return query, errors.New("invalid exact model")
	}
	if query.Operation != "" && safePublicCode(query.Operation) != query.Operation {
		return query, errors.New("invalid operation")
	}
	for name := range query.Parameters {
		if name == "" || safePublicCode(name) != name {
			return query, errors.New("invalid parameter name")
		}
	}
	return query, nil
}

func executeRetrievalModelQuery(output io.Writer, svc service, query modelQuery) error {
	body, status, retry, code, message, err := svc.request(context.Background(), http.MethodGet, "/v1/models", nil, "")
	if err != nil || status < 200 || status >= 300 {
		writeReceipt(output, apiFailure("submission", status, retry, code, message, "Model visibility could not be read. No embedding or rerank request was submitted."))
		return errors.New("retrieval catalog unavailable")
	}
	catalog, err := readAPIObject(body)
	entries, ok := catalog["data"].([]any)
	if err != nil || !ok {
		writeReceipt(output, validationFailure("The API did not return a readable model directory. No inference request was submitted."))
		return errors.New("invalid retrieval catalog")
	}
	matched, seen := []any{}, map[string]bool{}
	for _, entry := range entries {
		id, _ := jsonObject(entry)["id"].(string)
		if retrievalModels[id] != query.Kind || seen[id] || (query.Model != "" && id != query.Model) {
			continue
		}
		seen[id] = true
		matched = append(matched, map[string]any{"id": id, "capabilities": []string{query.Kind}, "input_schema": map[string]any{}})
	}
	writeJSON(output, map[string]any{"ok": true, "command": "models", "result": map[string]any{
		"data": matched, "matched_count": len(matched),
		"filter":         map[string]any{"kind": query.Kind, "model": query.Model, "operation": "", "parameter_names": []string{}},
		"matching_scope": "reviewed_" + query.Kind + "_models_listed_by_api",
		"note":           "Exact reviewed IDs visible in the authenticated directory. This does not verify route deployment, submission permission, price or result quality.",
	}})
	return nil
}

func executeAudioModelQuery(output io.Writer, svc service, query modelQuery) error {
	body, status, retry, code, message, err := svc.request(context.Background(), http.MethodGet, "/v1/models", nil, "")
	if err != nil || status < 200 || status >= 300 {
		writeReceipt(output, apiFailure("submission", status, retry, code, message, "Audio model visibility could not be read; no audio request was submitted."))
		return errors.New("audio catalog unavailable")
	}
	catalog, err := readAPIObject(body)
	entries, ok := catalog["data"].([]any)
	if err != nil || !ok {
		writeReceipt(output, validationFailure("The API did not return a readable directory. No audio request was submitted."))
		return errors.New("invalid audio catalog")
	}
	profiles := audioProfiles()
	matched, seen := []any{}, map[string]bool{}
	for _, entry := range entries {
		id, _ := jsonObject(entry)["id"].(string)
		profile, ok := profiles[id]
		if !ok || seen[id] || (query.Model != "" && id != query.Model) || (query.Operation != "" && profile.Operation != query.Operation) {
			continue
		}
		seen[id] = true
		matched = append(matched, map[string]any{"id": id, "capabilities": []string{"audio"}, "input_schema": map[string]any{}})
	}
	writeJSON(output, map[string]any{"ok": true, "command": "models", "result": map[string]any{
		"data": matched, "matched_count": len(matched),
		"filter":         map[string]any{"kind": "audio", "model": query.Model, "operation": query.Operation, "parameter_names": []string{}},
		"matching_scope": "reviewed_audio_models_listed_by_api",
		"note":           "Reviewed audio IDs visible in the authenticated directory. Operations and model/voice limits come from the installed audio contract, not live capability declarations. This does not verify route deployment, permission, price or output quality.",
	}})
	return nil
}

// The general authenticated catalog declares visibility, not a native schema.
// Only exact reviewed IDs are exposed; no prefix-based capability inference.
func executeEvaluationModelQuery(output io.Writer, svc service, query modelQuery) error {
	body, status, retry, code, message, err := svc.request(context.Background(), http.MethodGet, "/v1/models", nil, "")
	if err != nil || status < 200 || status >= 300 {
		writeReceipt(output, apiFailure("submission", status, retry, code, message, "Evaluation model visibility could not be read; no evaluation was submitted."))
		return errors.New("evaluation catalog unavailable")
	}
	catalog, err := readAPIObject(body)
	entries, valid := catalog["data"].([]any)
	if err != nil || !valid {
		writeReceipt(output, validationFailure("The API did not return a readable model directory. No evaluation was submitted."))
		return errors.New("invalid evaluation catalog")
	}
	matched, seen := []any{}, map[string]bool{}
	for _, value := range entries {
		id, _ := jsonObject(value)["id"].(string)
		if !evaluationModels[id] || seen[id] || (query.Model != "" && query.Model != id) {
			continue
		}
		seen[id] = true
		matched = append(matched, map[string]any{"id": id, "capabilities": []string{"evaluation"}, "input_schema": map[string]any{}})
	}
	writeJSON(output, map[string]any{"ok": true, "command": "models", "result": map[string]any{
		"data": matched, "matched_count": len(matched),
		"filter":         map[string]any{"kind": "evaluation", "model": query.Model, "operation": "", "parameter_names": []string{}},
		"matching_scope": "reviewed_evaluation_models_listed_by_api",
		"note":           "Exact reviewed evaluation IDs present in the authenticated model directory. Native route availability, price, submission permission and output quality are not verified; question types come from the installed evaluation contract, not this directory.",
	}})
	return nil
}

func modelQueryMatches(query modelQuery, entry map[string]any) bool {
	id, _ := entry["id"].(string)
	if !validModelID(id) || safePublicString(id) != id || (query.Model != "" && query.Model != id) {
		return false
	}
	capabilities := array(entry["capabilities"])
	if !hasValue(capabilities, "image") && !hasValue(capabilities, "video") {
		return false
	}
	if query.Kind != "" && !hasValue(capabilities, query.Kind) {
		return false
	}
	// An unfiltered read still exposes the full available public schema, even
	// when a model has no parameter schema. A filter never invents a declaration.
	if query.Operation == "" && len(query.Parameters) == 0 {
		return true
	}
	schemaValue, exists := entry["input_schema"]
	if !exists {
		return false
	}
	body, err := json.Marshal(reviewedMediaSchema(id, jsonObject(schemaValue)))
	var schema parameterSchema
	if err != nil || json.Unmarshal(body, &schema) != nil {
		return false
	}
	if reviewed, exists := reviewedVideoProfile(id); exists && !modelSchemaMatches(query, capabilities, reviewed.Parameters) {
		return false
	}
	return modelSchemaMatches(query, capabilities, schema)
}

func modelSchemaMatches(query modelQuery, capabilities []any, schema parameterSchema) bool {
	var operation mediaOperation
	counts := make(map[string]int)
	if query.Operation != "" {
		var exists bool
		operation, exists = schema.Operations[query.Operation]
		if !exists || operation.Request.Method != http.MethodPost ||
			(operation.Request.ContentType != "application/json" && operation.Request.ContentType != "multipart/form-data") {
			return false
		}
		routeMatches := false
		for _, capability := range []string{"image", "video"} {
			if (query.Kind == "" || query.Kind == capability) && hasValue(capabilities, capability) && allowedMediaPath(capability, operation.Request.Path) {
				routeMatches = true
			}
		}
		if !routeMatches {
			return false
		}
		for _, input := range operation.Inputs {
			if input.Required {
				counts[input.Field] = max(1, input.Min)
			}
		}
		for _, field := range operation.Required {
			if mediaReferenceField(field) {
				counts[field] = max(1, counts[field])
			}
		}
	}
	for name, value := range query.Parameters {
		if validateModelParameter(schema, operation, name, value, counts, operation.Request.ContentType == "multipart/form-data") != nil {
			return false
		}
	}
	return validateModelConstraints(schema, query.Operation, query.Parameters, counts) == nil
}
