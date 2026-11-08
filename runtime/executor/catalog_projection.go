package main

import (
	_ "embed"
	"encoding/json"
	"math"
	"strings"
	"sync"
)

// Shared with the catalog importer. Only reviewed schema structure may reach
// public receipts; live submission validation still uses the complete schema.
//
//go:embed public-catalog-projection.json
var publicCatalogProjectionJSON []byte

var publicCatalogProjection = sync.OnceValue(func() map[string]any {
	var policy map[string]any
	if json.Unmarshal(publicCatalogProjectionJSON, &policy) != nil || policy == nil {
		panic("Invalid public catalog projection policy.")
	}
	return policy
})

func publicCatalogKey(key string) bool {
	if !publicCodePattern.MatchString(key) || safePublicString(key) != key {
		return false
	}
	switch strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key)) {
	case "apikey", "token", "secret", "authorization", "headers", "provider", "upstream", "supplier", "channel", "credentials", "constructor", "prototype", "proto":
		return false
	}
	return true
}

func publicCatalogScalar(value any) (any, bool) {
	switch v := value.(type) {
	case string:
		return v, safePublicString(v) == v
	case float64:
		return v, !math.IsInf(v, 0) && !math.IsNaN(v)
	case int, bool, nil:
		return v, true
	}
	return nil, false
}

func projectPublicCatalogNode(value, policy any) (any, bool) {
	if choices, ok := policy.([]any); ok {
		return value, hasValue(choices, value)
	}
	if fields, ok := policy.(map[string]any); ok {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		result := map[string]any{}
		for key, item := range object {
			if !publicCatalogKey(key) {
				continue
			}
			rule, exists := fields[key]
			if !exists {
				rule, exists = fields["*"]
			}
			if !exists {
				continue
			}
			if selected, valid := projectPublicCatalogNode(item, rule); valid {
				result[key] = selected
			}
		}
		return result, true
	}
	switch policy {
	case "values", "value":
		if values := array(value); values != nil {
			result := make([]any, 0, len(values))
			for _, item := range values {
				if selected, valid := publicCatalogScalar(item); valid {
					result = append(result, selected)
				}
			}
			return result, true
		}
		if policy == "value" {
			return publicCatalogScalar(value)
		}
	case "string":
		if _, ok := value.(string); ok {
			return publicCatalogScalar(value)
		}
	case "number":
		if n, ok := number(value); ok && !math.IsNaN(n) && !math.IsInf(n, 0) {
			return value, true
		}
	case "boolean":
		if _, ok := value.(bool); ok {
			return value, true
		}
	case "method":
		if value == "POST" || value == "GET" {
			return value, true
		}
	case "path":
		if path, ok := value.(string); ok && (allowedMediaPath("image", path) || allowedMediaPath("video", path) ||
			path == "/v1/images/{image_id}" || path == "/v1/videos/{task_id}") {
			return path, true
		}
	}
	return nil, false
}

func projectPublicCatalogSchema(schema map[string]any) map[string]any {
	value, _ := projectPublicCatalogNode(schema, publicCatalogProjection())
	return jsonObject(value)
}
