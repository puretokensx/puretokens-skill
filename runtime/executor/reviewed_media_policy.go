package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"maps"
	"sync"
)

// Reviewed, exact-model restrictions only. Neither a stale catalog nor a
// missing installed profile may expand this boundary. Existing task reads do
// not consult it and retain their original task/model identity.
//
//go:embed reviewed-media-policy.json
var reviewedMediaPolicyJSON []byte

type reviewedMediaPolicy struct {
	Retired                          bool                `json:"retired"`
	UnsupportedOperations            []string            `json:"unsupportedOperations"`
	UnsupportedParametersByOperation map[string][]string `json:"unsupportedParametersByOperation"`
	UnsupportedInputs                []string            `json:"unsupportedInputs"`
	UnsupportedTransports            []string            `json:"unsupportedTransports"`
	LocalFileEncoding                string              `json:"localFileEncoding"`
	LocalMaskEncoding                string              `json:"localMaskEncoding"`
	SVGOutput                        bool                `json:"svgOutput"`
	TailFrameOutput                  bool                `json:"tailFrameOutput"`
	KlingNative                      bool                `json:"klingNative"`
}

var reviewedMediaPolicies = sync.OnceValue(func() map[string]reviewedMediaPolicy {
	var document struct {
		SchemaVersion int                            `json:"schemaVersion"`
		Models        map[string]reviewedMediaPolicy `json:"models"`
	}
	if json.Unmarshal(reviewedMediaPolicyJSON, &document) != nil || document.SchemaVersion != 1 || document.Models == nil {
		panic("Invalid embedded reviewed media policy.")
	}
	return document.Models
})

func validateReviewedMediaOperation(id, operation string, parameters map[string]any) error {
	policy := reviewedMediaPolicies()[id]
	if parameters["return_last_frame"] == true && !policy.TailFrameOutput {
		return errors.New("This exact model has no reviewed separate tail-frame contract. No task was sent; confirm video-only output before proceeding.")
	}
	if parameters["output_format"] == "svg" && !policy.SVGOutput {
		return errors.New("This exact model has no reviewed SVG output contract. Choose a supported raster format before submitting.")
	}
	if operation == "masked_image_edit" && policy.LocalMaskEncoding != "data_url" {
		return errors.New("This model has no reviewed local-mask JSON contract. No task was created.")
	}
	if policy.Retired {
		return errors.New("This exact model is retired for new tasks. Preserve existing tasks for recovery; choose a currently listed model only with explicit approval.")
	}
	for _, field := range policy.UnsupportedInputs {
		if _, exists := parameters[field]; exists {
			return errors.New("This native input has no reviewed execution contract. Use the declared flat parameters; no task was created.")
		}
	}
	if contains(policy.UnsupportedOperations, operation) {
		return errors.New("This model operation is unavailable. No task was created; choose a supported operation or explicitly select another model.")
	}
	for _, field := range policy.UnsupportedParametersByOperation[operation] {
		if _, present := parameters[field]; present {
			return errors.New("This edit inherits duration, aspect ratio and resolution from the source video. Remove generation controls before submitting; they will not be silently discarded.")
		}
	}
	return nil
}

// Preserve unrelated live schema fields and restrictions while removing
// closed operations. Projection is a copy, never a mutation of API evidence.
func reviewedMediaSchema(id string, schema map[string]any) map[string]any {
	policy := reviewedMediaPolicies()[id]
	result := maps.Clone(schema)
	if result == nil {
		result = make(map[string]any)
	}
	constraints := maps.Clone(jsonObject(schema["constraints"]))
	if constraints == nil {
		constraints = make(map[string]any)
	}
	properties := maps.Clone(jsonObject(schema["properties"]))
	if field := jsonObject(properties["return_last_frame"]); field != nil && !policy.TailFrameOutput {
		field = maps.Clone(field)
		field["enum"] = []any{false}
		delete(field, "default")
		properties["return_last_frame"] = field
	}
	if format := jsonObject(properties["output_format"]); hasValue(array(format["enum"]), "svg") && !policy.SVGOutput {
		format = maps.Clone(format)
		values := []any{}
		for _, value := range array(format["enum"]) {
			if value != "svg" {
				values = append(values, value)
			}
		}
		format["enum"] = values
		properties["output_format"] = format
	}
	if len(policy.UnsupportedInputs) > 0 {
		unsupported := maps.Clone(jsonObject(constraints["unsupported_inputs"]))
		if unsupported == nil {
			unsupported = make(map[string]any)
		}
		for _, field := range policy.UnsupportedInputs {
			delete(properties, field)
			unsupported[field] = true
		}
		constraints["unsupported_inputs"] = unsupported
	}
	if properties != nil {
		result["properties"] = properties
	}
	if len(policy.UnsupportedTransports) > 0 || policy.LocalFileEncoding != "" || policy.LocalMaskEncoding != "" {
		operations := maps.Clone(jsonObject(schema["operations"]))
		for name, value := range operations {
			op := maps.Clone(jsonObject(value))
			if op == nil {
				continue
			}
			inputs := maps.Clone(jsonObject(op["inputs"]))
			for key, value := range inputs {
				input := maps.Clone(jsonObject(value))
				if input == nil {
					continue
				}
				transports := []any{}
				for _, value := range array(input["transports"]) {
					if name == "masked_image_edit" && key == "mask" && value == "data_url" && policy.LocalMaskEncoding == "data_url" {
						value = "local_file_data_url"
					}
					if value == "multipart_file" && policy.LocalFileEncoding == "data_url" {
						value = "local_file_data_url"
					}
					if transport, ok := value.(string); ok && !contains(policy.UnsupportedTransports, transport) {
						transports = append(transports, value)
					}
				}
				input["transports"] = transports
				inputs[key] = input
			}
			if inputs != nil {
				op["inputs"] = inputs
			}
			operations[name] = op
		}
		result["operations"] = operations
	}
	if len(policy.UnsupportedOperations) > 0 {
		operations := maps.Clone(jsonObject(result["operations"]))
		unsupported := maps.Clone(jsonObject(constraints["unsupported_operations"]))
		if unsupported == nil {
			unsupported = make(map[string]any)
		}
		for _, operation := range policy.UnsupportedOperations {
			delete(operations, operation)
			unsupported[operation] = true
		}
		result["operations"] = operations
		constraints["unsupported_operations"] = unsupported
	}
	if policy.KlingNative {
		unsupported := maps.Clone(jsonObject(constraints["unsupported_inputs"]))
		for _, field := range []string{"contents", "settings", "options"} {
			delete(unsupported, field)
		}
		if unsupported != nil {
			constraints["unsupported_inputs"] = unsupported
		}
		if properties == nil {
			properties = make(map[string]any)
		}
		for _, field := range []string{"contents", "settings", "options"} {
			properties[field] = map[string]any{"type": "json"}
		}
		result["properties"] = properties
		operations := maps.Clone(jsonObject(result["operations"]))
		if operations == nil {
			operations = make(map[string]any)
		}
		operations["native_video"] = map[string]any{
			"request":            map[string]any{"method": "POST", "path": "/v1/videos", "contentType": "application/json"},
			"requiredBodyFields": []any{"model", "contents"},
		}
		result["operations"] = operations
	}
	if len(policy.UnsupportedParametersByOperation) > 0 {
		unsupported := maps.Clone(jsonObject(constraints["unsupported_parameters_by_operation"]))
		if unsupported == nil {
			unsupported = make(map[string]any)
		}
		for operation, fields := range policy.UnsupportedParametersByOperation {
			values := append([]any{}, array(unsupported[operation])...)
			for _, field := range fields {
				if !hasValue(values, field) {
					values = append(values, field)
				}
			}
			unsupported[operation] = values
		}
		constraints["unsupported_parameters_by_operation"] = unsupported
	}
	if len(constraints) > 0 {
		result["constraints"] = constraints
	}
	return result
}
