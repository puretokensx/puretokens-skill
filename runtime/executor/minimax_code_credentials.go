package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Desktop 3.1.0 only. The saved default is usable only in a local default
// conversation confirmed by the host binding; it cannot reveal a session override.
func credentialFromMiniMaxCode() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", clientSelectionFailure()
	}
	root, err := miniMaxCodeRoot(runtime.GOOS, home, os.Getenv)
	if err != nil {
		return "", err
	}
	return credentialFromMiniMaxCodeRoot(root)
}

func miniMaxCLIRoot(home string, getenv func(string) string) (string, error) {
	if !clientAbsoluteRoot(home) {
		return "", clientFormatFailure()
	}
	for _, name := range []string{"MAVIS_PROFILE", "MINIMAX_PROFILE", "AGENTARCHON_PROFILE", "AGENTARCHON_DATA_DIR", "__MAVIS_RUNTIME_PROFILE", "__MAVIS_RUNTIME_DATA_DIR"} {
		if strings.TrimSpace(getenv(name)) != "" {
			return "", clientSelectionFailure()
		}
	}
	root := strings.TrimSpace(getenv("MINIMAX_DATA_DIR"))
	if root == "" {
		root = strings.TrimSpace(getenv("MAVIS_DATA_DIR"))
	}
	if root == "" {
		return miniMaxDefaultRoot(home)
	}
	if !clientAbsoluteRoot(root) {
		return "", clientFormatFailure()
	}
	return filepath.Clean(root), nil
}

func credentialFromMiniMaxCodeCLI() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", clientSelectionFailure()
	}
	root, err := miniMaxCLIRoot(home, os.Getenv)
	if err != nil {
		return "", err
	}
	return credentialFromMiniMaxCodeRoot(root)
}

func miniMaxCodeRoot(goos, home string, getenv func(string) string) (string, error) {
	if (goos != "darwin" && goos != "windows") || !clientAbsoluteRoot(home) {
		return "", clientFormatFailure()
	}
	// CLI profiles and unrepresented runtime composition are not Desktop defaults.
	for _, name := range []string{"MAVIS_PROFILE", "MINIMAX_PROFILE", "AGENTARCHON_PROFILE", "AGENTARCHON_DATA_DIR", "__MAVIS_RUNTIME_PROFILE", "__MAVIS_RUNTIME_DATA_DIR"} {
		if strings.TrimSpace(getenv(name)) != "" {
			return "", clientSelectionFailure()
		}
	}
	for _, name := range []string{"MINIMAX_DATA_DIR", "MAVIS_DATA_DIR"} {
		if raw := strings.TrimSpace(getenv(name)); raw != "" {
			if !clientAbsoluteRoot(raw) {
				return "", clientFormatFailure()
			}
			return filepath.Clean(raw), nil
		}
	}
	prefsRoot := filepath.Join(home, "Library", "Application Support")
	if goos == "windows" {
		prefsRoot = getenv("APPDATA")
	}
	if !clientAbsoluteRoot(prefsRoot) {
		return "", clientFormatFailure()
	}
	selected := ""
	for _, app := range []string{"MiniMax Code", "MiniMax", "MiniMax Agent"} {
		for _, store := range []string{"minimax-agent-cn-config.json", "minimax-agent-config.json"} {
			path := filepath.Join(prefsRoot, app, store)
			absent, err := miniMaxPathAbsent(path)
			if err != nil {
				return "", err
			}
			if absent {
				continue
			}
			value, err := readClientJSON(path, false)
			if err != nil {
				return "", err
			}
			doc := jsonObject(value)
			if doc == nil {
				return "", clientFormatFailure()
			}
			config := jsonObject(doc["config"])
			if v, ok := doc["config"]; ok && (v == nil || config == nil) {
				return "", clientFormatFailure()
			}
			rawValue, exists := config["localRuntimeDataParentDir"]
			if !exists || rawValue == nil {
				continue
			}
			raw, ok := rawValue.(string)
			if !ok {
				return "", clientFormatFailure()
			}
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if !clientAbsoluteRoot(raw) {
				return "", clientFormatFailure()
			}
			candidate := filepath.Join(raw, ".minimax")
			if selected != "" && selected != candidate {
				return "", clientSelectionFailure()
			}
			selected = candidate
		}
	}
	if selected != "" {
		return selected, nil
	}
	return miniMaxDefaultRoot(home)
}

func miniMaxDefaultRoot(home string) (string, error) {
	root := filepath.Join(home, ".minimax")
	absent, err := miniMaxPathAbsent(filepath.Join(root, "config.yaml"))
	if err != nil {
		return "", err
	}
	if absent {
		if _, err := os.Lstat(filepath.Join(home, ".mavis")); !os.IsNotExist(err) {
			return "", clientSelectionFailure()
		}
	}
	return root, nil
}

// Missing preference files are normal, but a linked parent is never an absent
// preference. Do not follow it or guess the default directory in its place.
func miniMaxPathAbsent(path string) (bool, error) {
	absent := false
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			absent = true
		} else if err != nil || !clientPathSafe(current, info) ||
			(current != path && !info.IsDir()) || (current == path && !info.Mode().IsRegular()) {
			return false, clientFormatFailure()
		}
		if filepath.Dir(current) == current {
			return absent, nil
		}
	}
}

func credentialFromMiniMaxCodeRoot(root string) (string, error) {
	cfg, err := readClientYAML(filepath.Join(root, "config.yaml"))
	if err != nil {
		return "", clientFormatFailure()
	}
	if hasClientOverride(cfg, "model", "effectiveModel", "session", "modelOverrides", "apiKey", "baseURL", "headers", "auth") {
		return "", clientSelectionFailure()
	}
	if agents, exists := cfg["agents"]; exists {
		entries := jsonObject(agents)
		if entries == nil {
			return "", clientSelectionFailure()
		}
		for id, agent := range entries {
			if id != "default" || jsonObject(agent) == nil || hasClientOverride(jsonObject(agent), "model", "provider", "modelId", "modelID", "apiKey", "baseURL", "headers") {
				return "", clientSelectionFailure()
			}
		}
	}
	selection := jsonString(cfg["defaultModel"])
	if !strings.HasPrefix(selection, "custom_provider:") {
		return "", clientSelectionFailure()
	}
	providerID, modelID, ok := strings.Cut(strings.TrimPrefix(selection, "custom_provider:"), "/")
	if !ok || providerID == "" || modelID == "" {
		return "", clientSelectionFailure()
	}
	provider := jsonObject(jsonObject(cfg["custom_provider"])[providerID])
	if provider == nil || provider["enabled"] != true || jsonString(provider["kind"]) != "custom" {
		return "", clientSelectionFailure()
	}
	for key := range provider {
		switch key {
		case "name", "kind", "api", "enabled", "options", "models":
		default:
			return "", clientFormatFailure()
		}
	}
	options := jsonObject(provider["options"])
	if options == nil {
		return "", clientFormatFailure()
	}
	for key := range options {
		if key != "baseURL" && key != "apiKey" {
			return "", clientFormatFailure()
		}
	}
	endpoint := jsonString(options["baseURL"])
	if !matchesPureTokensEndpoint(endpoint, "/v1", "/v1/") {
		return matchingCredential(endpoint, "", "/v1", "/v1/")
	}
	switch jsonString(provider["api"]) {
	case "openai-completions", "openai-responses", "anthropic-messages":
	default:
		return "", clientFormatFailure()
	}
	model := jsonObject(jsonObject(provider["models"])[modelID])
	if model == nil || model["enabled"] != true {
		return "", clientSelectionFailure()
	}
	for key := range model {
		switch key {
		case "name", "enabled", "limit", "modalities", "capabilities", "thinking", "variants", "contextWindowOptions":
		default:
			return "", clientFormatFailure()
		}
	}
	// Do not inspect sibling providers' secrets or official minimax_api login.
	return strictInlineCredential(endpoint, jsonString(options["apiKey"]))
}
