package main

// Only an explicitly requested format constrains old or new task downloads.
// Never infer a format from today's model defaults during task recovery.
func imageFormatMatches(format, mediaType string) bool {
	if mediaType == "image/svg+xml" {
		return format == "svg"
	}
	if format == "" {
		return true
	}
	expected := map[string]string{"png": "image/png", "jpeg": "image/jpeg", "webp": "image/webp", "gif": "image/gif"}[format]
	return expected != "" && expected == mediaType
}

func taskImageFormatMatches(request taskRequest, mediaType string) bool {
	if mediaType == "image/svg+xml" {
		return taskAllowsSVG(request)
	}
	if request.Kind != "image" {
		return true
	}
	format, _ := request.Parameters["output_format"].(string)
	return imageFormatMatches(format, mediaType)
}

func taskAllowsSVG(request taskRequest) bool {
	if request.Kind != "image" || request.Parameters["output_format"] != "svg" {
		return false
	}
	switch request.Model {
	case "gpt-image-2.6", "gpt-image-2.6-flare", "gpt-image-2.6-sunburst":
		return true
	}
	return false
}
