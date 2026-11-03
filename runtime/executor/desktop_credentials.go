package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"
)

// These are documented application roots, not candidate searches. In a remote
// or sandboxed session where the host files are unavailable, resolution stops.
func desktopDataRoot(platform, home, appData, app string) (string, error) {
	switch platform {
	case "darwin":
		if filepath.IsAbs(home) {
			return filepath.Join(home, "Library", "Application Support", app), nil
		}
	case "windows":
		if filepath.IsAbs(appData) {
			return filepath.Join(appData, app), nil
		}
	}
	return "", credentialFailure("host_credential_adapter_unavailable", "The desktop host connection is unavailable in this execution environment.", "Run this Skill in the configured desktop's local execution environment.")
}

func currentDesktopRoot(app string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return desktopDataRoot(runtime.GOOS, home, os.Getenv("APPDATA"), app)
}

func credentialFromClaudeDesktop() (string, error) {
	root, err := currentDesktopRoot("Claude-3p")
	if err != nil {
		return "", err
	}
	return credentialFromClaudeDesktopRoot(root)
}

var desktopProfileID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func credentialFromClaudeDesktopRoot(root string) (string, error) {
	config, err := readJSONObject(filepath.Join(root, "claude_desktop_config.json"))
	if err != nil {
		return "", err
	}
	if jsonString(config["deploymentMode"]) != "3p" {
		return "", desktopSelectionFailure()
	}
	meta, err := readJSONObject(filepath.Join(root, "configLibrary", "_meta.json"))
	if err != nil {
		return "", err
	}
	id := jsonString(meta["appliedId"])
	// Only the active UUID file can be read. Names, entries and unrelated
	// profiles are not credential sources; reject paths before opening a file.
	if !desktopProfileID.MatchString(id) {
		return "", desktopSelectionFailure()
	}
	profile, err := readJSONObject(filepath.Join(root, "configLibrary", id+".json"))
	if err != nil {
		return "", err
	}
	if jsonString(profile["inferenceProvider"]) != "gateway" ||
		jsonString(profile["inferenceCredentialKind"]) != "static" ||
		jsonString(profile["inferenceGatewayAuthScheme"]) != "bearer" {
		return "", desktopSelectionFailure()
	}
	return matchingCredential(jsonString(profile["inferenceGatewayBaseUrl"]), jsonString(profile["inferenceGatewayApiKey"]), "", "/", "/v1", "/v1/")
}

func desktopSelectionFailure() error {
	return credentialFailure("active_connection_unavailable", "The desktop adapter could not resolve a supported effective connection for this request.", "")
}

func credentialFromDSHDesktop() (string, error) {
	root := os.Getenv("DSH_HOME")
	if root != "" {
		if !filepath.IsAbs(root) {
			return "", desktopSelectionFailure()
		}
	} else {
		data, err := currentDesktopRoot("dsh-desktop")
		if err != nil {
			return "", err
		}
		root = filepath.Join(data, "harness")
	}
	return credentialFromDSHDesktopRoot(root)
}

func credentialFromDSHDesktopRoot(root string) (string, error) {
	settings, err := readDesktopYAML(filepath.Join(root, "settings.yaml"))
	if err != nil {
		return "", err
	}
	selected := jsonObject(settings["agent-default-model"])
	id := jsonString(selected["provider"])
	model := jsonString(selected["model"])
	if id == "" || model == "" {
		return "", desktopSelectionFailure()
	}
	providers := jsonObject(jsonObject(settings["llm-pi-ai"])["providers"])
	provider := jsonObject(providers[id])
	endpoint := jsonString(provider["baseURL"])
	if !matchesPureTokensEndpoint(endpoint, "/v1", "/v1/") {
		return matchingCredential(endpoint, "", "/v1", "/v1/")
	}
	found := false
	for _, item := range jsonArray(provider["models"]) {
		if jsonString(jsonObject(item)["id"]) == model {
			found = true
		}
	}
	ref := jsonString(provider["apiKeyEnv"])
	if !found || jsonString(provider["api"]) != "openai-completions" || ref == "" {
		return "", desktopSelectionFailure()
	}
	// The reference selects one saved entry, never an environment fallback.
	credentials, err := readDesktopYAML(filepath.Join(root, ".credentials.yaml"))
	if err != nil {
		return "", err
	}
	if credentials["version"] == 1 {
		return matchingCredential(endpoint, jsonString(jsonObject(credentials["refs"])[ref]), "/v1", "/v1/")
	}
	// Switch preserves the older flat store for installed Harness runtimes.
	// Accept its exact shape, not a versioned/mixed or arbitrary YAML document.
	for name, value := range credentials {
		if name == "version" || name == "refs" || name == "records" ||
			!desktopCredentialReference.MatchString(name) || strings.TrimSpace(jsonString(value)) == "" {
			return "", desktopSelectionFailure()
		}
	}
	if !desktopCredentialReference.MatchString(ref) {
		return "", desktopSelectionFailure()
	}
	return matchingCredential(endpoint, jsonString(credentials[ref]), "/v1", "/v1/")
}

var desktopCredentialReference = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Use a real YAML parser for quoted/flow/block values. Reject ambiguous or
// expanding layouts and multiple documents before decoding into values.
func readDesktopYAML(path string) (map[string]any, error) {
	data, err := readBoundedFile(path, maxConfigBytes)
	if err != nil {
		return nil, errors.New("desktop connection record is unavailable")
	}
	defer clear(data)
	return parseDesktopYAML(data)
}

func parseDesktopYAML(data []byte) (map[string]any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var node yaml.Node
	if decoder.Decode(&node) != nil || !safeDesktopYAML(&node, 0) {
		return nil, errors.New("desktop connection record is unreadable")
	}
	var extra yaml.Node
	if decoder.Decode(&extra) != io.EOF {
		return nil, errors.New("desktop connection record is unreadable")
	}
	var result map[string]any
	if node.Decode(&result) != nil || result == nil {
		return nil, errors.New("desktop connection record is unreadable")
	}
	return result, nil
}

func safeDesktopYAML(node *yaml.Node, depth int) bool {
	if depth > 32 || node.Kind == yaml.AliasNode || node.Anchor != "" {
		return false
	}
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]bool)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" || seen[key.Value] {
				return false
			}
			seen[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if !safeDesktopYAML(child, depth+1) {
			return false
		}
	}
	return true
}
