package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Preserve the upstream distinction between an unset canonical profile and an
// explicitly empty one. The latter selects default, not the legacy profile.
func ohMyPiGetenv(name string) string {
	if name == "OMP_PROFILE" {
		if value, exists := os.LookupEnv(name); exists && strings.TrimSpace(value) == "" {
			return "default"
		}
	}
	return os.Getenv(name)
}

func ohMyPiProfileRoot(home string, getenv func(string) string) (string, string, error) {
	config := getenv("PI_CONFIG_DIR")
	if config == "" {
		config = ".omp"
	}
	if !ohMyPiConfigName.MatchString(config) || config == "." || config == ".." || strings.EqualFold(config, ".pi") || strings.HasSuffix(config, ".") || ohMyPiReservedName.MatchString(config) {
		return "", "", clientSelectionFailure()
	}
	profile := getenv("OMP_PROFILE")
	if profile == "" {
		profile = getenv("PI_PROFILE")
	}
	profile = strings.TrimSpace(profile)
	root := filepath.Join(home, config)
	if profile == "" || profile == "default" {
		return root, "", nil
	}
	if !ohMyPiProfileName.MatchString(profile) || strings.HasSuffix(profile, ".") || ohMyPiReservedName.MatchString(profile) {
		return "", "", clientSelectionFailure()
	}
	return filepath.Join(root, "profiles", profile), profile, nil
}

// Read only the selected host profile. Never borrow Pi or another OMP profile.
func ohMyPiRoot(home string, getenv func(string) string) (string, error) {
	if !clientAbsoluteRoot(home) {
		return "", clientFormatFailure()
	}
	for _, name := range []string{"OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR"} {
		if getenv(name) != "" {
			return "", clientSelectionFailure()
		}
	}
	if _, err := ohMyPiConfigFiles(getenv("PI_CONFIG_FILES")); err != nil {
		return "", err
	}
	configRoot, profile, err := ohMyPiProfileRoot(home, getenv)
	if err != nil {
		return "", err
	}
	root := filepath.Join(configRoot, "agent")
	if profile == "" && getenv("PI_CODING_AGENT_DIR") != "" {
		// A parent may have exported a directory derived from a previous profile.
		if legacy := strings.TrimSpace(getenv("PI_PROFILE")); legacy != "" && legacy != "default" {
			return "", clientSelectionFailure()
		}
		root = getenv("PI_CODING_AGENT_DIR")
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
	root, err := ohMyPiRoot(home, ohMyPiGetenv)
	if err != nil {
		return "", err
	}
	configRoot, _, err := ohMyPiProfileRoot(home, ohMyPiGetenv)
	if err != nil {
		return "", err
	}
	for _, p := range []string{filepath.Join(home, ".env"), filepath.Join(configRoot, ".env"), filepath.Join(configRoot, "agent", ".env"), filepath.Join(root, ".env")} {
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
	overlays, err := ohMyPiConfigFiles(os.Getenv("PI_CONFIG_FILES"))
	if err != nil {
		return "", err
	}
	return credentialFromOhMyPiLayers(root, overlays)
}

func ohMyPiConfigFiles(raw string) ([]string, error) {
	files := []string{}
	for _, file := range filepath.SplitList(raw) {
		if file == "" {
			continue
		}
		if !clientAbsoluteRoot(file) || len(files) == 4 {
			return nil, clientSelectionFailure()
		}
		files = append(files, file)
	}
	return files, nil
}

// Check only known native project settings locations, never their contents or
// another client's configuration. Compatibility-provider/session overrides
// still require the caller's explicit saved-default host binding.
func ohMyPiProjectScope(cwd, home string) error {
	if !clientAbsoluteRoot(cwd) {
		return clientSelectionFailure()
	}
	if _, err := os.Lstat(filepath.Join(cwd, ".env")); !os.IsNotExist(err) {
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
	return credentialFromOhMyPiLayers(root, nil)
}

func mergeOhMyPiSettings(base, layer map[string]any) {
	for key, value := range layer {
		if nested, current := jsonObject(value), jsonObject(base[key]); nested != nil && current != nil {
			mergeOhMyPiSettings(current, nested)
		} else {
			base[key] = value
		}
	}
}

func credentialFromOhMyPiLayers(root string, overlays []string) (string, error) {
	settings, err := ohMyPiYAML(root, "config")
	if err != nil {
		return "", clientFormatFailure()
	}
	// Explicit ordered settings overlays select a connection, never supply a
	// second credential store. Each bounded YAML layer must be readable.
	for _, file := range overlays {
		layer, err := readClientYAML(file)
		if err != nil {
			return "", clientFormatFailure()
		}
		mergeOhMyPiSettings(settings, layer)
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
var ohMyPiConfigName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var ohMyPiProfileName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var ohMyPiReservedName = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])([.]|$)`)
