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
	UnsupportedOperations            []string            `json:"unsupportedOperations"`
	UnsupportedParametersByOperation map[string][]string `json:"unsupportedParametersByOperation"`
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
	if contains(policy.UnsupportedOperations, operation) {
		return errors.New("This model's video editing is unavailable. No task was created; choose a supported operation or explicitly select another model.")
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
	policy, exists := reviewedMediaPolicies()[id]
	if !exists {
		return schema
	}
	result := maps.Clone(schema)
	if result == nil {
		result = make(map[string]any)
	}
	constraints := maps.Clone(jsonObject(schema["constraints"]))
	if constraints == nil {
		constraints = make(map[string]any)
	}
	if len(policy.UnsupportedOperations) > 0 {
		operations := maps.Clone(jsonObject(schema["operations"]))
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
	result["constraints"] = constraints
	return result
}
