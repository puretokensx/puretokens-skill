package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Default local omp only. Named profiles and dotenv path composition require
// independent host evidence; do not borrow the unrelated Pi configuration.
func ohMyPiRoot(home string, getenv func(string) string) (string, error) {
	if !clientAbsoluteRoot(home) {
		return "", clientFormatFailure()
	}
	for _, name := range []string{"PI_CONFIG_DIR", "PI_CONFIG_FILES", "OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR"} {
		if getenv(name) != "" {
			return "", clientSelectionFailure()
		}
	}
	for _, name := range []string{"OMP_PROFILE", "PI_PROFILE"} {
		if value := getenv(name); value != "" && value != "default" {
			return "", clientSelectionFailure()
		}
	}
	root := getenv("PI_CODING_AGENT_DIR")
	if root == "" {
		root = filepath.Join(home, ".omp", "agent")
	}
	if !clientAbsoluteRoot(root) || filepath.Clean(root) == filepath.Join(home, ".pi", "agent") {
		return "", clientSelectionFailure()
	}
	return filepath.Clean(root), nil
}

func credentialFromOhMyPi() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", clientSelectionFailure()
	}
	root, err := ohMyPiRoot(home, os.Getenv)
	if err != nil {
		return "", err
	}
	for _, p := range []string{filepath.Join(home, ".env"), filepath.Join(home, ".omp", ".env"), filepath.Join(home, ".omp", "agent", ".env"), filepath.Join(root, ".env")} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			return "", clientSelectionFailure()
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", clientSelectionFailure()
	}
	if err := ohMyPiProjectScope(cwd, home); err != nil {
		return "", err
	}
	return credentialFromOhMyPiRoot(root)
}

// Check only known native project settings locations, never their contents or
// another client's configuration. Compatibility-provider/session overrides
// still require the caller's explicit saved-default host binding.
func ohMyPiProjectScope(cwd, home string) error {
	if !clientAbsoluteRoot(cwd) {
		return clientSelectionFailure()
	}
	for depth := 0; depth < 64; depth++ {
		for _, name := range []string{"config.yml", "config.yaml", "settings.json"} {
			if _, err := os.Lstat(filepath.Join(cwd, ".omp", name)); !os.IsNotExist(err) {
				return clientSelectionFailure()
			}
		}
		parent := filepath.Dir(cwd)
		if cwd == home || parent == cwd {
			return nil
		}
		cwd = parent
	}
	return clientSelectionFailure()
}

func ohMyPiYAML(root, name string) (map[string]any, error) {
	p := filepath.Join(root, name+".yml")
	if _, err := os.Lstat(p); os.IsNotExist(err) {
		p = filepath.Join(root, name+".yaml")
	}
	return readClientYAML(p)
}

func credentialFromOhMyPiRoot(root string) (string, error) {
	settings, err := ohMyPiYAML(root, "config")
	if err != nil {
		return "", clientFormatFailure()
	}
	if hasClientOverride(settings, "model", "provider", "baseUrl", "apiKey", "session", "modelOverrides") {
		return "", clientSelectionFailure()
	}
	for _, key := range []string{"enabledModels", "enabledProviders", "disabledProviders"} {
		if value, ok := settings[key]; ok && value != nil {
			if items, ok := value.([]any); !ok || len(items) != 0 {
				return "", clientSelectionFailure()
			}
		}
	}
	providerID, modelID, ok := strings.Cut(jsonString(jsonObject(settings["modelRoles"])["default"]), "/")
	if !ok || providerID == "" || modelID == "" {
		return "", clientSelectionFailure()
	}
	models, err := ohMyPiYAML(root, "models")
	if err != nil {
		return "", clientFormatFailure()
	}
	provider := jsonObject(jsonObject(models["providers"])[providerID])
	if provider == nil {
		return "", clientSelectionFailure()
	}
	endpoint := jsonString(provider["baseUrl"])
	if !matchesPureTokensEndpoint(endpoint, "/v1", "/v1/") {
		return matchingCredential(endpoint, "", "/v1", "/v1/")
	}
	for key := range provider {
		switch key {
		case "api", "auth", "baseUrl", "apiKey", "models":
		default:
			return "", clientFormatFailure()
		}
	}
	if jsonString(provider["auth"]) != "apiKey" {
		return "", clientFormatFailure()
	}
	switch jsonString(provider["api"]) {
	case "openai-completions", "openai-responses", "anthropic-messages":
	default:
		return "", clientFormatFailure()
	}
	found := false
	for _, raw := range jsonArray(provider["models"]) {
		entry := jsonObject(raw)
		if jsonString(entry["id"]) != modelID {
			continue
		}
		if found {
			return "", clientSelectionFailure()
		}
		for key := range entry {
			switch key {
			case "id", "name", "input":
			default:
				return "", clientFormatFailure()
			}
		}
		found = true
	}
	if !found {
		return "", clientSelectionFailure()
	}
	// Upstream config keys precede stored login authentication. Runtime
	// overrides are outside this saved-default binding; no auth DB is read.
	rawToken := jsonString(provider["apiKey"])
	if rawToken != strings.TrimSpace(rawToken) {
		return "", clientFormatFailure()
	}
	token, err := strictInlineCredential(endpoint, rawToken)
	if err != nil {
		return "", err
	}
	// Upstream resolves even bare config values as environment references.
	// Reject reference-shaped values without resolving them. For a literal,
	// check only exact-name presence; never use or compare the shadow value.
	if ohMyPiEnvironmentReference.MatchString(token) {
		return "", clientFormatFailure()
	}
	if _, exists := os.LookupEnv(token); exists {
		return "", clientSelectionFailure()
	}
	return token, nil
}

var ohMyPiEnvironmentReference = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
