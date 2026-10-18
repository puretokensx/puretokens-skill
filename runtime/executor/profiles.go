package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type modelProfile struct {
	ID         string          `json:"id"`
	Capability string          `json:"capability"`
	Parameters parameterSchema `json:"parameterSchema"`
}
type parameterSchema struct {
	Properties  map[string]map[string]any `json:"properties"`
	Constraints map[string]map[string]any `json:"constraints"`
	Operations  map[string]mediaOperation `json:"operations"`
}
type mediaOperation struct {
	Request struct {
		Method      string `json:"method"`
		Path        string `json:"path"`
		ContentType string `json:"contentType"`
	} `json:"request"`
	Required []string `json:"requiredBodyFields"`
	Inputs   map[string]struct {
		Field      string   `json:"field"`
		Required   bool     `json:"required"`
		Min        int      `json:"minItems"`
		Max        int      `json:"maxItems"`
		Transports []string `json:"transports"`
	} `json:"inputs"`
}

func loadProfile(request taskRequest, svc service) (modelProfile, error) {
	var profile modelProfile
	if !validModelID(request.Model) {
		return profile, errors.New("Choose an exact model ID from the model index.")
	}
	if svc.profilesRoot != "" {
		file := filepath.Join(svc.profilesRoot, "puretokens-"+request.Kind, "references", "profiles", request.Model+".json")
		if body, err := readBoundedFile(file, maxResponseBytes); err == nil && json.Unmarshal(body, &profile) == nil && profile.ID == request.Model && profile.Capability == request.Kind {
			gap, err := profileRequestGap(request, profile)
			if err != nil {
				return profile, err
			}
			if !gap {
				return profile, nil
			}
			return loadLiveProfile(request, svc)
		}
	}
	// Unknown exact IDs can still submit core text requests without a preflight.
	if request.Operation == "generate" && len(request.Parameters) == 0 && len(request.Attachments) == 0 && request.MediaOperation == "" && request.RequestedCount <= 1 {
		return modelProfile{ID: request.Model, Capability: request.Kind}, nil
	}
	return loadLiveProfile(request, svc)
}

func loadLiveProfile(request taskRequest, svc service) (modelProfile, error) {
	body, status, _, _, _, err := svc.request(context.Background(), http.MethodGet, "/v1/media/models", nil, "")
	if err == nil && status == 200 {
		var catalog struct {
			Error   any   `json:"error"`
			Success *bool `json:"success"`
			Data    []struct {
				ID           string          `json:"id"`
				Capabilities []string        `json:"capabilities"`
				InputSchema  parameterSchema `json:"input_schema"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &catalog) == nil && catalog.Error == nil && (catalog.Success == nil || *catalog.Success) {
			for _, entry := range catalog.Data {
				if entry.ID == request.Model && contains(entry.Capabilities, request.Kind) {
					profile := modelProfile{ID: entry.ID, Capability: request.Kind, Parameters: entry.InputSchema}
					gap, err := profileRequestGap(request, profile)
					if err != nil {
						return profile, err
					}
					if !gap {
						return profile, nil
					}
					break
				}
			}
		}
	}
	return modelProfile{}, errors.New("The selected model's required parameters or operation could not be confirmed. Ask for current model details; no media task was created.")
}

// Refresh missing declarations only. Existing type, range and enum failures
// remain invalid even when another part of the request needs a refresh.
func profileRequestGap(request taskRequest, profile modelProfile) (bool, error) {
	if rule, exists := profile.Parameters.Properties["prompt"]; exists {
		if err := validateProperty("prompt", request.Prompt, rule); err != nil {
			return false, err
		}
	}
	for key, value := range request.Parameters {
		if key == "model" || key == "prompt" || key == "async" {
			return false, errors.New("Model, prompt and async are reserved request fields.")
		}
		if rule, exists := profile.Parameters.Properties[key]; exists {
			if err := validateProperty(key, value, rule); err != nil {
				return false, err
			}
		}
		if _, unsupported := profile.Parameters.Constraints["unsupported_inputs"][key]; unsupported {
			return false, errors.New("The selected model explicitly excludes this input.")
		}
	}
	if request.RequestedCount > 1 {
		if rule, exists := profile.Parameters.Properties["n"]; exists {
			if err := validateProperty("n", request.RequestedCount, rule); err != nil {
				return false, err
			}
		}
	}
	opName, err := requestedMediaOperation(request)
	if err != nil {
		return false, err
	}
	operation, declared := profile.Parameters.Operations[opName]
	gap := opName != "" && !declared
	if opName != "" && declared {
		if operation.Request.Method == "" || operation.Request.Path == "" || operation.Request.ContentType == "" {
			gap = true
		}
		if (len(request.Attachments) > 0 && operation.Request.ContentType != "multipart/form-data") ||
			(len(request.Attachments) == 0 && operation.Request.ContentType == "multipart/form-data") {
			gap = true
		}
	}
	counts := make(map[string]int)
	for _, file := range request.Attachments {
		counts[file.Field]++
		found := false
		for _, input := range operation.Inputs {
			if input.Field != file.Field {
				continue
			}
			found = contains(input.Transports, "multipart_file")
			if input.Max > 0 && counts[file.Field] > input.Max {
				return false, fmt.Errorf("The attachment count exceeds the declared maximum of %d.", input.Max)
			}
		}
		if !found {
			gap = true
		}
	}
	for key, value := range request.Parameters {
		if counts[key] > 0 {
			return false, errors.New("Do not send the same media field as both a file and a JSON parameter.")
		}
		_, propertyExists := profile.Parameters.Properties[key]
		input, inputExists := operation.Inputs[key]
		if !propertyExists && (!inputExists || operation.Request.ContentType != "application/json") {
			gap = true
		}
		transports, referenceDeclared := profile.Parameters.Constraints["reference_transport"][key]
		if referenceDeclared || inputExists || mediaReferenceField(key) {
			if inputExists && opName != "" {
				if !contains(input.Transports, "public_https_url") && !contains(input.Transports, "https_url") {
					gap = true
				}
				if !referenceDeclared {
					transports = input.Transports
				}
			}
			if !hasValue(array(transports), "public_https_url") && !hasValue(array(transports), "https_url") {
				gap = true
			}
			// Validate public-host and count restrictions before any refresh.
			if err := validateReferences(value, append(array(transports), "public_https_url")); err != nil {
				return false, err
			}
		}
	}
	if request.RequestedCount > 1 {
		if _, exists := profile.Parameters.Properties["n"]; !exists {
			gap = true
		}
	}
	return gap, nil
}

func mediaReferenceField(key string) bool {
	switch key {
	case "image", "image_urls", "audio_urls", "video_urls", "first_frame_image", "last_frame_image", "reference_images", "reference_videos", "reference_audios", "video", "audio":
		return true
	}
	return false
}

func requestedMediaOperation(request taskRequest) (string, error) {
	opName := request.MediaOperation
	if request.Operation == "edit" && opName == "" {
		opName = request.Kind + "_edit"
	}
	if request.Operation == "edit" && opName != request.Kind+"_edit" {
		return "", errors.New("The edit request must select the model's declared edit operation.")
	}
	if len(request.Attachments) > 0 && opName == "" {
		return "", errors.New("Specify the attachment role as media_operation, such as image_to_video or reference_image_video.")
	}
	return opName, nil
}
func prepareProfileRequest(request *taskRequest, svc service) error {
	if request.Kind == "music" {
		return prepareMusicRequest(request)
	}
	opName, err := requestedMediaOperation(*request)
	if err != nil {
		return err
	}
	if err := validateReviewedMediaOperation(request.Model, opName, request.Parameters); err != nil {
		return err
	}
	if request.Parameters == nil {
		request.Parameters = make(map[string]any)
	}
	profile, err := loadProfile(*request, svc)
	if err != nil {
		return err
	}
	if rule, exists := profile.Parameters.Properties["prompt"]; exists {
		if err = validateProperty("prompt", request.Prompt, rule); err != nil {
			return err
		}
	}
	var operation mediaOperation
	if opName != "" {
		var exists bool
		operation, exists = profile.Parameters.Operations[opName]
		if !exists {
			return errors.New("The selected model does not declare this media operation. Choose a declared operation before submitting.")
		}
		if operation.Request.Method != "POST" || !allowedMediaPath(request.Kind, operation.Request.Path) {
			return errors.New("The model operation has no supported fixed API path.")
		}
		request.route = operation.Request.Path
		if operation.Request.ContentType == "multipart/form-data" && len(request.Attachments) == 0 {
			return errors.New("This operation requires a current local attachment. For a public URL use the declared generation reference field instead.")
		}
		if len(request.Attachments) > 0 && operation.Request.ContentType != "multipart/form-data" {
			return errors.New("Local attachments require a declared multipart operation.")
		}
		if operation.Request.ContentType != "multipart/form-data" && operation.Request.ContentType != "application/json" {
			return errors.New("The model operation uses an unsupported request representation.")
		}
	}
	counts := make(map[string]int)
	for _, file := range request.Attachments {
		matched := false
		for _, input := range operation.Inputs {
			if input.Field == file.Field && contains(input.Transports, "multipart_file") {
				matched = true
				counts[file.Field]++
				if input.Max > 0 && counts[file.Field] > input.Max {
					return fmt.Errorf("The attachment count exceeds the declared maximum of %d.", input.Max)
				}
			}
		}
		if !matched {
			return errors.New("An attachment field or transport is not declared for this operation.")
		}
	}
	for key, value := range request.Parameters {
		if err = validateModelParameter(profile.Parameters, operation, key, value, counts, len(request.Attachments) > 0); err != nil {
			return err
		}
	}
	for _, input := range operation.Inputs {
		if operation.Request.ContentType == "multipart/form-data" && input.Required && counts[input.Field] < max(1, input.Min) {
			return errors.New("A required current attachment is missing for this operation.")
		}
	}
	for key, rule := range profile.Parameters.Properties {
		if rule["required"] == true && key != "prompt" && key != "model" {
			if _, exists := request.Parameters[key]; !exists && counts[key] == 0 {
				return errors.New("A required model parameter is missing.")
			}
		}
	}
	for _, key := range operation.Required {
		if key != "prompt" && key != "model" && counts[key] == 0 {
			if _, exists := request.Parameters[key]; !exists {
				return errors.New("A required operation input is missing.")
			}
		}
	}
	if err = validateModelConstraints(profile.Parameters, opName, request.Parameters, counts); err != nil {
		return err
	}
	// Deterministically apply declared size precedence, without guessing values.
	order := array(profile.Parameters.Constraints["size_expression_precedence"]["order"])
	chosen := ""
	for _, item := range order {
		expr := fmt.Sprint(item)
		fields := strings.Split(expr, "+")
		present := true
		for _, field := range fields {
			if _, ok := request.Parameters[field]; !ok {
				present = false
			}
		}
		if present && chosen == "" {
			chosen = expr
			continue
		}
		if chosen != "" {
			for _, field := range fields {
				delete(request.Parameters, field)
			}
		}
	}
	count := 1
	if n, exists := request.Parameters["n"]; exists {
		value, ok := number(n)
		if !ok || value != math.Trunc(value) || value < 1 || value > maxImageCount {
			return errors.New("Image count must be an integer from 1 through 8.")
		}
		count = int(value)
	} else if request.RequestedCount > 1 {
		rule, exists := profile.Parameters.Properties["n"]
		if !exists {
			return errors.New("The selected model does not declare multiple outputs.")
		}
		if err = validateProperty("n", float64(request.RequestedCount), rule); err != nil {
			return err
		}
		count = request.RequestedCount
		request.Parameters["n"] = count
	}
	if request.RequestedCount != 0 && request.RequestedCount != count {
		return errors.New("requested_count and n disagree; correct the count before submitting.")
	}
	if request.Kind == "video" && count != 1 {
		return errors.New("One video task returns one video.")
	}
	request.RequestedCount = count
	return nil
}
func allowedMediaPath(kind, path string) bool {
	return (kind == "image" && (path == "/v1/images/generations" || path == "/v1/images/edits")) || (kind == "video" && (path == "/v1/videos" || path == "/v1/videos/edits"))
}
func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
func array(value any) []any {
	if values, ok := value.([]any); ok {
		return values
	}
	if values, ok := value.([]string); ok {
		result := make([]any, len(values))
		for i, v := range values {
			result[i] = v
		}
		return result
	}
	return nil
}
func number(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		v, e := n.Float64()
		return v, e == nil
	}
	return 0, false
}
func hasValue(values []any, value any) bool {
	for _, v := range values {
		if reflect.DeepEqual(v, value) {
			return true
		}
		a, ok := number(v)
		b, ok2 := number(value)
		if ok && ok2 && a == b {
			return true
		}
	}
	return false
}
func validateProperty(name string, value any, rule map[string]any) error {
	invalid := func() error {
		return fmt.Errorf("Parameter %s does not match the model's declared type, range or supported values.", name)
	}
	var length int
	switch rule["type"] {
	case "integer", "number":
		n, ok := number(value)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) || (rule["type"] == "integer" && n != math.Trunc(n)) {
			return invalid()
		}
		for _, key := range []string{"min", "minimum"} {
			if min, ok := number(rule[key]); ok && n < min {
				return invalid()
			}
		}
		for _, key := range []string{"max", "maximum"} {
			if max, ok := number(rule[key]); ok && n > max {
				return invalid()
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return invalid()
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return invalid()
		}
		length = utf8.RuneCountInString(text)
	case "string[]":
		values := array(value)
		if values == nil {
			return invalid()
		}
		length = len(values)
		for _, v := range values {
			if _, ok := v.(string); !ok {
				return invalid()
			}
		}
	case "json":
		if _, err := json.Marshal(value); err != nil {
			return invalid()
		}
	default:
		return invalid()
	}
	if min, ok := number(rule["minLength"]); ok && length < int(min) {
		return invalid()
	}
	if max, ok := number(rule["maxLength"]); ok && length > int(max) {
		return invalid()
	}
	if options := array(rule["enum"]); len(options) > 0 && !hasValue(options, value) {
		return invalid()
	}
	return nil
}
func validateReferences(value, transport any) error {
	values := array(value)
	if text, ok := value.(string); ok {
		values = []any{text}
	} else if object, ok := value.(map[string]any); ok {
		values = []any{object}
	}
	allowsURL := false
	maxCount := 0
	for _, entry := range array(transport) {
		text := fmt.Sprint(entry)
		if text == "public_https_url" || text == "https_url" {
			allowsURL = true
		}
		if strings.HasPrefix(text, "max_") {
			maxCount, _ = strconv.Atoi(strings.TrimPrefix(text, "max_"))
		}
	}
	if !allowsURL || len(values) == 0 || (maxCount > 0 && len(values) > maxCount) {
		return errors.New("The model does not support this reference representation or count.")
	}
	for _, v := range values {
		text, ok := v.(string)
		if object, isObject := v.(map[string]any); isObject && len(object) == 1 {
			text, ok = object["url"].(string)
			if !ok {
				text, ok = object["image_url"].(string)
			}
		}
		if !ok {
			return errors.New("Reference inputs must contain only public HTTPS URLs.")
		}
		u, err := url.Parse(text)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return errors.New("Reference URLs must be explicit public HTTPS URLs without embedded credentials.")
		}
		// A trailing DNS dot does not turn a local host into a public one.
		// Scoped IPv6 addresses identify an interface and are never public URLs.
		host := strings.TrimRight(strings.ToLower(u.Hostname()), ".")
		if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.Contains(host, "%") {
			return errors.New("Reference URLs must be explicit public HTTPS URLs without embedded credentials.")
		}
		if ip := net.ParseIP(host); ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()) {
			return errors.New("Reference URLs must use publicly reachable hosts.")
		}
	}
	return nil
}
