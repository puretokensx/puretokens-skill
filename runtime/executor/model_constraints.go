package main

import (
	"errors"
	"fmt"
)

// Query and submission use the same value and cross-field rules. Queries infer
// required operation inputs; submissions validate actual attachment counts.
func validateModelParameter(schema parameterSchema, operation mediaOperation, name string, value any, counts map[string]int, multipart bool) error {
	if counts[name] > 0 && multipart {
		return errors.New("Do not send the same media field as both a file and a JSON parameter.")
	}
	rule, exists := schema.Properties[name]
	input, declaredInput := operation.Inputs[name]
	if declaredInput && !multipart {
		if !contains(input.Transports, "public_https_url") && !contains(input.Transports, "https_url") {
			return errors.New("This JSON attachment transport is not supported by the model.")
		}
		if err := validateReferences(value, input.Transports); err != nil {
			return err
		}
		count := len(array(value))
		if _, ok := value.(string); ok {
			count = 1
		}
		if count < max(1, input.Min) || (input.Max > 0 && count > input.Max) {
			return errors.New("The reference count does not match the declared operation.")
		}
	}
	if !exists {
		if !declaredInput || operation.Request.ContentType != "application/json" {
			return errors.New("An optional field is not declared by this model; check its supported parameters before submitting.")
		}
		rule = map[string]any{"type": "string[]"}
		if _, ok := value.(string); ok {
			rule["type"] = "string"
		}
	}
	if err := validateProperty(name, value, rule); err != nil {
		return err
	}
	if transports, exists := schema.Constraints["reference_transport"][name]; exists {
		if multipart {
			if _, declared := operation.Inputs[name]; !declared {
				return errors.New("Mixing URL references and local attachments requires an explicitly declared combination operation.")
			}
		}
		if err := validateReferences(value, transports); err != nil {
			return err
		}
	}
	if _, unsupported := schema.Constraints["unsupported_inputs"][name]; unsupported {
		return errors.New("The selected model explicitly excludes this input.")
	}
	return nil
}

func validateModelConstraints(schema parameterSchema, operation string, parameters map[string]any, counts map[string]int) error {
	if schema.Constraints["unsupported_operations"][operation] == true {
		return errors.New("The selected model explicitly excludes this media operation.")
	}
	for _, field := range array(schema.Constraints["unsupported_parameters_by_operation"][operation]) {
		if _, present := parameters[fmt.Sprint(field)]; present {
			return errors.New("A supplied parameter is excluded for this media operation; preserve the requested requirements and correct them before submitting.")
		}
	}
	if matrix, exists := schema.Constraints["aspect_ratio_by_image_size"]; exists {
		// Omitted values use the model's declared defaults, not invented tiers.
		size := parameters["image_size"]
		if size == nil {
			size = schema.Properties["image_size"]["default"]
		}
		ratio := parameters["aspect_ratio"]
		if ratio == nil {
			ratio = schema.Properties["aspect_ratio"]["default"]
		}
		if size != nil && ratio != nil && !hasValue(array(matrix[fmt.Sprint(size)]), ratio) {
			return errors.New("This aspect ratio is unavailable at the selected image size. Choose a declared combination.")
		}
	}
	present := func(key string) bool { return counts[key] > 0 || parameters[key] != nil }
	if rules, ok := schema.Constraints["frame_exclusivity"]; ok {
		if hasValue(array(rules["last_frame_image"]), "requires_first_frame_image") && present("last_frame_image") && !present("first_frame_image") {
			return errors.New("A last frame requires the first frame for this model.")
		}
		if hasValue(array(rules["first_frame_image"]), "cannot_mix_reference_images_videos_audios") && present("first_frame_image") {
			for _, field := range []string{"image_urls", "audio_urls", "video_urls", "reference_images", "reference_videos", "reference_audios"} {
				if present(field) {
					return errors.New("First/last frames cannot be combined with additional reference media.")
				}
			}
		}
	}
	groups := 0
	for _, fields := range schema.Constraints["exclusive_reference_sets"] {
		for _, field := range array(fields) {
			if present(fmt.Sprint(field)) {
				groups++
				break
			}
		}
	}
	if groups > 1 {
		return errors.New("First/last frames cannot be combined with additional reference media.")
	}
	for key, companions := range schema.Constraints["requires_together"] {
		if present(key) {
			for _, companion := range array(companions) {
				if name, ok := companion.(string); !ok || !present(name) {
					return errors.New("Required companion parameters must be supplied together as declared by the model.")
				}
			}
		}
	}
	mode := "text"
	if present("image") || present("first_frame_image") || present("last_frame_image") || operation == "image_to_video" {
		mode = "image"
	}
	if operation == "reference_image_video" || operation == "reference_video" || operation == "reference_audio" {
		mode = "reference"
	}
	if allowed, exists := schema.Constraints["resolution_by_mode"][mode]; exists && parameters["resolution"] != nil && !hasValue(array(allowed), parameters["resolution"]) {
		return errors.New("This resolution is unavailable for the selected reference mode. Choose a declared resolution.")
	}
	return nil
}
