package main

import "strings"

// Only dimensions are defaulted. Counts, quality and other optional controls
// retain their existing semantics; a default must never create another output.
func applyImageDimensionDefaults(request *taskRequest, schema parameterSchema, operation mediaOperation, opName string) error {
	parameters := request.Parameters
	allowed := func(field string) bool {
		if _, excluded := schema.Constraints["unsupported_inputs"][field]; excluded {
			return false
		}
		if hasValue(array(schema.Constraints["unsupported_parameters_by_operation"][opName]), field) {
			return false
		}
		if modes, limited := schema.Constraints["parameter_modes"][field]; limited && !hasValue(array(modes), opName) {
			return false
		}
		return contains([]string{"image_size", "resolution", "size", "aspect_ratio"}, field)
	}
	apply := func(field string) error {
		if _, supplied := parameters[field]; supplied || !allowed(field) {
			return nil
		}
		value, declared := schema.Properties[field]["default"]
		if !declared {
			return nil
		}
		if err := validateModelParameter(schema, operation, field, value, nil, false); err != nil {
			return err
		}
		parameters[field] = value
		return nil
	}

	// A combined size or explicit pixel dimensions already express the user's
	// geometry. Do not introduce a competing tier or aspect ratio.
	for _, field := range []string{"size", "width", "height"} {
		if _, supplied := parameters[field]; supplied {
			return nil
		}
	}
	if order := array(schema.Constraints["size_expression_precedence"]["order"]); len(order) > 0 {
		// Fill only the selected expression, never promote a default over an
		// expression the caller supplied. Unsupported expressions stay untouched.
		var selected []string
		for _, expression := range order {
			text, ok := expression.(string)
			if !ok {
				continue
			}
			fields := strings.Split(text, "+")
			complete := true
			for _, field := range fields {
				if _, supplied := parameters[field]; !supplied {
					complete = false
				}
			}
			if complete {
				return nil
			}
		}
		for _, expression := range order {
			text, ok := expression.(string)
			if !ok {
				continue
			}
			fields := strings.Split(text, "+")
			for _, field := range fields {
				if _, supplied := parameters[field]; supplied {
					selected = fields
					break
				}
			}
			if selected != nil {
				break
			}
		}
		if selected == nil {
			for _, expression := range order {
				text, ok := expression.(string)
				if !ok {
					continue
				}
				fields := strings.Split(text, "+")
				complete := true
				for _, field := range fields {
					if _, declared := schema.Properties[field]["default"]; !declared || !allowed(field) {
						complete = false
					}
				}
				if complete {
					selected = fields
					break
				}
			}
		}
		for _, field := range selected {
			if err := apply(field); err != nil {
				return err
			}
		}
		return nil
	}
	_, hasTier := parameters["image_size"]
	_, hasResolution := parameters["resolution"]
	if !hasTier && !hasResolution {
		for _, field := range []string{"image_size", "resolution", "size"} {
			if err := apply(field); err != nil {
				return err
			}
			if _, present := parameters[field]; present {
				break
			}
		}
	}
	return apply("aspect_ratio")
}
