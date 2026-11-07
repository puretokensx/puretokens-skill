package main

import (
	"crypto/subtle"
	"os"
	"path/filepath"
)

// Paseo supplies this profile to its Codex app-server in memory. Do not read
// ~/.codex/auth.json: it is not the configured Paseo connection.
func credentialFromPaseoRoot(root string) (string, error) {
	if os.Getenv("PASEO_HOME") == "" {
		return "", clientSelectionFailure()
	}
	doc, err := switchCLIObject(filepath.Join(root, "config.json"), true)
	if err != nil {
		return "", err
	}
	agents := jsonObject(doc["agents"])
	providers := jsonObject(agents["providers"])
	if hasClientOverride(agents, "env", "options", "command") || providers["codex"] != nil {
		return "", clientFormatFailure()
	}
	var selected map[string]any
	for _, raw := range providers {
		p := jsonObject(raw)
		env := jsonObject(p["env"])
		endpoint := jsonString(env["OPENAI_BASE_URL"])
		if !matchesPureTokensEndpoint(endpoint, "", "/v1") {
			if vscodeClaimsPureTokens(endpoint) {
				return "", clientFormatFailure()
			}
			continue
		}
		if selected != nil {
			return "", switchCLIAmbiguous()
		}
		selected = p
	}
	if selected == nil {
		return "", clientSelectionFailure()
	}
	if selected["extends"] != "codex" || !switchCLIFields(selected, "extends", "label", "description", "env", "models") {
		return "", clientFormatFailure()
	}
	env := jsonObject(selected["env"])
	if !switchCLIFields(env, "OPENAI_BASE_URL", "OPENAI_API_KEY") {
		return "", clientFormatFailure()
	}
	models, ok := selected["models"].([]any)
	if !ok || len(models) == 0 || len(models) > 512 {
		return "", clientFormatFailure()
	}
	ids := map[string]bool{}
	for _, raw := range models {
		m := jsonObject(raw)
		id := jsonString(m["id"])
		if id == "" || ids[id] || !switchCLIFields(m, "id", "name", "label", "description", "contextWindow", "thinkingOptions") {
			return "", clientFormatFailure()
		}
		ids[id] = true
	}
	endpoint := jsonString(env["OPENAI_BASE_URL"])
	token, err := strictInlineCredential(endpoint, jsonString(env["OPENAI_API_KEY"]))
	if err != nil {
		return "", err
	}
	// These two exact fields are declared by this endpoint-verified Paseo profile.
	// Compare only to detect stale or different launch contexts; never use a
	// process value as a fallback credential or read another host's auth store.
	if os.Getenv("OPENAI_BASE_URL") != endpoint || subtle.ConstantTimeCompare([]byte(os.Getenv("OPENAI_API_KEY")), []byte(token)) != 1 {
		return "", clientSelectionFailure()
	}
	return token, nil
}
