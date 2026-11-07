package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Query and submission use the same value and cross-field rules. Queries infer
// required operation inputs; submissions validate actual attachment counts.
func validateModelParameter(schema parameterSchema, operation mediaOperation, name string, value any, counts map[string]int, multipart bool) error {
	if counts[name] > 0 && multipart {
		return errors.New("Do not send the same media field as both a file and a JSON parameter.")
	}
	rule, exists := schema.Properties[name]
	input, declaredInput := operation.Inputs[name]
	if declaredInput {
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
	referenceCount := func(key string) int {
		if counts[key] > 0 {
			return counts[key]
		}
		if _, ok := parameters[key].(string); ok {
			return 1
		}
		return len(array(parameters[key]))
	}
	effective := func(key string) any {
		if value, exists := parameters[key]; exists {
			return value
		}
		return schema.Properties[key]["default"]
	}
	if _, kling := schema.Constraints["kling_settings"]; kling {
		variant := "v3"
		if _, ok := schema.Constraints["kling_settings"]["adaptive_requires_reference"]; ok {
			variant = "v4"
		} else if _, ok := schema.Constraints["kling_settings"]["feature_video_audio"]; ok {
			variant = "omni"
		} else if _, ok := schema.Constraints["motion_control"]; ok {
			variant = "motion"
		}
		if err := validateKlingParameters(variant, parameters, counts); err != nil {
			return err
		}
	}
	if value := effective("n"); value != nil {
		count, ok := number(value)
		if !ok || count < 1 || count > maxImageCount || count != float64(int(count)) {
			return errors.New("The executor supports an integer output count from 1 through 8.")
		}
	}
	for field, modes := range schema.Constraints["parameter_modes"] {
		if present(field) && !hasValue(array(modes), operation) {
			return errors.New("A supplied parameter is unavailable for this operation. Choose the declared operation before submitting.")
		}
	}
	imageMode := "text"
	if operation == "image_edit" || operation == "masked_image_edit" || present("image") || present("images") || present("image_urls") {
		imageMode = "edit"
	}
	sequential := parameters["enable_sequential"] == true || parameters["sequential_image_generation"] == "auto"
	if allowed, exists := schema.Constraints["output_format_by_image_mode"][imageMode]; exists && effective("output_format") != nil && !hasValue(array(allowed), effective("output_format")) {
		return errors.New("This output format is unavailable for the selected image operation.")
	}
	size := effective("image_size")
	if parameters["size"] != nil {
		size = parameters["size"]
	}
	if size == nil {
		size = effective("size")
	}
	if allowed, exists := schema.Constraints["output_format_resolutions"][fmt.Sprint(effective("output_format"))]; exists && size != nil && !hasValue(array(allowed), strings.ToLower(fmt.Sprint(size))) {
		return errors.New("This output format is unavailable at the selected image size.")
	}
	resolutionMode := imageMode
	if sequential {
		resolutionMode = "sequential"
	}
	resolution := strings.ToLower(fmt.Sprint(size))
	// Mode limits describe named tiers, not explicit WIDTHxHEIGHT dimensions.
	namedResolution := resolution == "1k" || resolution == "2k" || resolution == "4k"
	if allowed, exists := schema.Constraints["resolution_by_image_mode"][resolutionMode]; exists && namedResolution && !hasValue(array(allowed), resolution) {
		return errors.New("This image size is unavailable for the selected operation or sequence.")
	}
	if rules, exists := schema.Constraints["image_size_format"]; exists && size != nil && !hasValue(array(rules["presets"]), size) {
		parts := strings.Split(fmt.Sprint(size), "x")
		valid := len(parts) == 2 && hasValue(array(rules["custom"]), "WIDTHxHEIGHT")
		for _, part := range parts {
			n, err := strconv.Atoi(part)
			min, _ := number(firstConstraintValue(rules["dimension_min"]))
			max, _ := number(firstConstraintValue(rules["dimension_max"]))
			multiple, _ := number(firstConstraintValue(rules["dimension_multiple"]))
			if err != nil || n <= 0 || strconv.Itoa(n) != part || float64(n) < min || (max > 0 && float64(n) > max) || (multiple > 0 && (multiple < 1 || multiple > float64(n) || multiple != float64(int(multiple)) || n%int(multiple) != 0)) {
				valid = false
			}
		}
		if !valid {
			return errors.New("Image size must use a declared preset or bounded WIDTHxHEIGHT dimensions.")
		}
	}
	if _, exists := schema.Constraints["image_size_format"]; exists && parameters["size"] != nil && parameters["image_size"] != nil && parameters["size"] != parameters["image_size"] {
		return errors.New("The explicit size and image_size values conflict. Choose one consistent size.")
	}
	if rules, exists := schema.Constraints["output_count"]; exists {
		n, ok := number(effective("n"))
		if !ok {
			n = 1
		}
		if _, declared := schema.Properties["sequential_image_generation_options"]; declared {
			options, hasOptions := parameters["sequential_image_generation_options"]
			count := float64(1)
			if hasOptions {
				object, ok := options.(map[string]any)
				if !ok || !sequential {
					return errors.New("Sequential options require the declared sequence mode.")
				}
				for key := range object {
					if key != "max_images" {
						return errors.New("Unknown sequential option.")
					}
				}
				if value, exists := object["max_images"]; exists {
					count, ok = number(value)
					if !ok || count < 1 || count > maxImageCount || count != float64(int(count)) {
						return errors.New("Sequential output count must be an integer from 1 through 8.")
					}
				}
			}
			if n != count {
				return errors.New("The output count must match sequential max_images; no values were changed.")
			}
		}
		limit := "standard_max"
		if sequential {
			limit = "sequential_max"
		}
		if maximum, ok := number(firstConstraintValue(rules[limit])); ok && n > maximum {
			return errors.New("The output count exceeds the selected standard or sequential mode limit.")
		}
		if maximum, ok := number(firstConstraintValue(rules["combined_input_output_max"])); ok {
			inputs := referenceCount("image") + referenceCount("images") + referenceCount("image_urls")
			if float64(inputs)+n > maximum {
				return errors.New("The combined input and output image count exceeds the model limit.")
			}
		}
	}
	if limits, exists := schema.Constraints["reference_counts"]; exists {
		total := 0
		for key, limit := range limits {
			if key == "max_total" {
				continue
			}
			count := referenceCount(key)
			if maximum, ok := number(limit); ok && float64(count) > maximum {
				return errors.New("The reference count exceeds this model's declared limit.")
			}
			total += count
		}
		if maximum, ok := number(limits["max_total"]); ok && float64(total) > maximum {
			return errors.New("The combined reference count exceeds this model's declared limit.")
		}
	}
	visual := present("first_frame_image") || present("last_frame_image") || present("reference_images") || present("reference_videos") || present("video")
	frames := present("first_frame_image") || present("last_frame_image")
	if rules, exists := schema.Constraints["reference_combination"]; exists {
		if hasValue(array(rules["audio_requires_visual"]), true) && present("reference_audios") && !visual {
			return errors.New("Audio references require a visual input for this model.")
		}
		if hasValue(array(rules["adaptive_requires_visual"]), true) && parameters["aspect_ratio"] == "adaptive" && !visual {
			return errors.New("Adaptive aspect ratio requires a visual input for this model.")
		}
		if hasValue(array(rules["frames_exclusive"]), true) && frames &&
			(present("reference_images") || present("reference_videos") || present("reference_audios")) {
			return errors.New("First/last frames cannot be combined with additional reference media.")
		}
	}
	if frames {
		for _, key := range array(schema.Constraints["frame_parameters"]["unsupported"]) {
			if present(fmt.Sprint(key)) {
				return errors.New("This model derives geometry from frame input. Omit incompatible frame parameters rather than silently discarding them.")
			}
		}
	}
	if schema.Constraints["audio_requires_reference_images"]["enabled"] == true && present("reference_audios") && !present("reference_images") {
		return errors.New("Audio references require reference images for this model.")
	}
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
	if present("reference_images") || present("reference_videos") || present("reference_audios") || operation == "reference_image_video" || operation == "reference_video" || operation == "reference_audio" || operation == "reference_audio_video" {
		mode = "reference"
	}
	if allowed, exists := schema.Constraints["resolution_by_mode"][mode]; exists && parameters["resolution"] != nil && !hasValue(array(allowed), parameters["resolution"]) {
		return errors.New("This resolution is unavailable for the selected reference mode. Choose a declared resolution.")
	}
	return nil
}

func firstConstraintValue(value any) any {
	if values := array(value); len(values) > 0 {
		return values[0]
	}
	return value
}
