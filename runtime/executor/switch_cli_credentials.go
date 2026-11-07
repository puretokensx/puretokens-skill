package main

import (
	"crypto/subtle"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

// These adapters read only the named host's saved connection. Provider display
// names are not evidence of origin, and chat selection is never inferred.
func switchCLIRoot(host, home string, getenv func(string) string) (string, error) {
	if !clientAbsoluteRoot(home) {
		return "", clientFormatFailure()
	}
	var root string
	switch host {
	case "paseo":
		root = getenv("PASEO_HOME")
		if root == "" {
			root = filepath.Join(home, ".paseo")
		}
	case "qwen-code":
		root = getenv("QWEN_HOME")
		if root == "" {
			root = filepath.Join(home, ".qwen")
		}
	case "codebuddy-code":
		root = getenv("CODEBUDDY_CONFIG_DIR")
		if root == "" {
			root = filepath.Join(home, ".codebuddy")
		}
	case "copilot-cli":
		root = getenv("COPILOT_HOME")
		if root == "" {
			root = filepath.Join(home, ".copilot")
		}
	case "factory-droid":
		if getenv("FACTORY_HOME_OVERRIDE") != "" {
			return "", clientFormatFailure()
		}
		root = filepath.Join(home, ".factory")
	case "mimo-coder":
		if base := getenv("MIMOCODE_HOME"); base != "" {
			if !clientAbsoluteRoot(base) {
				return "", clientFormatFailure()
			}
			root = filepath.Join(base, "config")
		} else {
			base := getenv("XDG_CONFIG_HOME")
			if base == "" {
				base = filepath.Join(home, ".config")
			}
			if !clientAbsoluteRoot(base) {
				return "", clientFormatFailure()
			}
			root = filepath.Join(base, "mimocode")
		}
	default:
		return "", clientFormatFailure()
	}
	if !clientAbsoluteRoot(root) {
		return "", clientFormatFailure()
	}
	return root, nil
}

func switchCLIObject(file string, jsonc bool) (map[string]any, error) {
	if jsonc {
		data, err := readClientBytes(file)
		if err != nil {
			return nil, err
		}
		defer clear(data)
		doc, err := parseOpenCodeObject(data, true)
		if err != nil {
			return nil, clientFormatFailure()
		}
		return doc, nil
	}
	value, err := readClientJSON(file, jsonc)
	if err != nil {
		return nil, err
	}
	object := jsonObject(value)
	if object == nil {
		return nil, clientFormatFailure()
	}
	return object, nil
}

func switchCLIOptionalObject(file string, jsonc bool) (map[string]any, error) {
	// A dangling link is not a missing optional file.
	if _, err := os.Lstat(file); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, clientFormatFailure()
	}
	return switchCLIObject(file, jsonc)
}

func switchCLIAmbiguous() error {
	return credentialFailure("active_connection_ambiguous", "The host does not expose one uniquely supported saved Pure Tokens connection; no API request was sent.", "")
}

func switchCLIAddKey(current *string, endpoint, candidate string) error {
	token, err := strictInlineCredential(endpoint, candidate)
	if err != nil {
		return err
	}
	if *current != "" && subtle.ConstantTimeCompare([]byte(*current), []byte(token)) != 1 {
		return switchCLIAmbiguous()
	}
	*current = token
	return nil
}

func switchCLIFields(object map[string]any, allowed ...string) bool {
	if object == nil {
		return false
	}
	for key := range object {
		found := false
		for _, name := range allowed {
			if key == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func switchCLIScopeRoots(cwd string) ([]string, error) {
	if !clientAbsoluteRoot(cwd) {
		return nil, clientFormatFailure()
	}
	var roots []string
	for dir := filepath.Clean(cwd); ; dir = filepath.Dir(dir) {
		roots = append(roots, dir)
		if len(roots) > 64 {
			return nil, clientFormatFailure()
		}
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return nil, clientFormatFailure()
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return roots, nil
}

func switchCLIModelFields(host string, doc map[string]any) bool {
	switch host {
	case "qwen-code":
		return hasClientOverride(doc, "model", "modelProviders", "providerProtocol", "security", "env")
	case "codebuddy-code":
		return hasClientOverride(doc, "models", "availableModels", "env", "model", "apiKeyHelper")
	case "factory-droid":
		return hasClientOverride(doc, "customModels", "custom_models")
	case "mimo-coder":
		if raw, exists := doc["provider"]; exists && jsonObject(raw) == nil {
			return true
		}
		return hasClientOverride(doc, "model", "small_model", "vision_model", "model_groups", "enabled_providers", "disabled_providers", "plugin") || hasClientOverride(jsonObject(doc["provider"]), "puretokens-switch")
	}
	return false
}

func switchCLIOverlays(host, root, home, cwd string) error {
	scopes, err := switchCLIScopeRoots(cwd)
	if err != nil {
		return err
	}
	var relatives []string
	switch host {
	case "qwen-code":
		relatives = []string{".qwen/settings.json"}
	case "codebuddy-code":
		relatives = []string{".codebuddy/models.json", ".codebuddy/settings.json", ".codebuddy/settings.local.json"}
	case "factory-droid":
		relatives = []string{".factory/settings.json", ".factory/settings.local.json", ".factory/config.json"}
	case "mimo-coder":
		relatives = []string{"mimocode.json", "mimocode.jsonc", ".mimocode/mimocode.json", ".mimocode/mimocode.jsonc"}
	}
	for _, scope := range scopes {
		for _, rel := range relatives {
			file := filepath.Join(scope, rel)
			if filepath.Dir(file) == root {
				continue
			}
			if host == "codebuddy-code" && strings.HasSuffix(rel, "models.json") {
				if _, err := os.Lstat(file); !os.IsNotExist(err) {
					return clientFormatFailure()
				}
				continue
			}
			doc, err := switchCLIOptionalObject(file, host == "mimo-coder" || host == "qwen-code")
			if err != nil {
				return err
			}
			if switchCLIModelFields(host, doc) {
				return clientFormatFailure()
			}
		}
	}
	if host == "qwen-code" {
		// Inspect assignment names only; unrelated dotenv preferences are harmless.
		for _, scope := range append(scopes, home, root, filepath.Join(home, ".qwen")) {
			for _, rel := range []string{".env", ".qwen/.env"} {
				if err := switchQwenDotenv(filepath.Join(scope, rel)); err != nil {
					return err
				}
			}
		}
		if err := switchQwenSystemSettings(runtime.GOOS); err != nil {
			return err
		}
	}
	if host == "factory-droid" {
		for _, name := range []string{"settings.local.json", "config.json"} {
			doc, err := switchCLIOptionalObject(filepath.Join(root, name), false)
			if err != nil {
				return err
			}
			if name == "config.json" {
				if rows, ok := doc["custom_models"].([]any); ok && len(rows) == 0 {
					delete(doc, "custom_models")
				}
			}
			if switchCLIModelFields(host, doc) {
				return clientFormatFailure()
			}
		}
	}
	if host == "codebuddy-code" {
		for _, name := range []string{"settings.json", "settings.local.json"} {
			doc, err := switchCLIOptionalObject(filepath.Join(root, name), false)
			if err != nil {
				return err
			}
			if switchCLIModelFields(host, doc) {
				return clientFormatFailure()
			}
		}
	}
	if host == "mimo-coder" {
		for _, name := range []string{"mimocode.json", "mimocode.jsonc"} {
			doc, err := switchCLIOptionalObject(filepath.Join(home, ".mimocode", name), true)
			if err != nil {
				return err
			}
			if switchCLIModelFields(host, doc) {
				return clientFormatFailure()
			}
		}
		for _, scope := range append(scopes, root, filepath.Join(home, ".mimocode")) {
			for _, name := range []string{"plugin", "plugins"} {
				for _, base := range []string{scope, filepath.Join(scope, ".mimocode")} {
					if _, err := os.Lstat(filepath.Join(base, name)); !os.IsNotExist(err) {
						return clientFormatFailure()
					}
				}
			}
		}
		managed := "/etc/opencode"
		if runtime.GOOS == "darwin" {
			managed = "/Library/Application Support/opencode"
			identity, err := user.Current()
			if err != nil {
				return clientFormatFailure()
			}
			for _, file := range []string{filepath.Join("/Library/Managed Preferences", identity.Username, "ai.opencode.managed.plist"), "/Library/Managed Preferences/ai.opencode.managed.plist"} {
				if _, err := os.Lstat(file); !os.IsNotExist(err) {
					return clientFormatFailure()
				}
			}
		}
		if runtime.GOOS == "windows" {
			base := os.Getenv("ProgramData")
			if base == "" {
				base = `C:\ProgramData`
			}
			managed = filepath.Join(base, "opencode")
			if !clientAbsoluteRoot(managed) {
				return clientFormatFailure()
			}
		}
		for _, name := range []string{"mimocode.json", "mimocode.jsonc"} {
			if _, err := os.Lstat(filepath.Join(managed, name)); !os.IsNotExist(err) {
				return clientFormatFailure()
			}
		}
	}
	return nil
}

func credentialFromSwitchCLI(host string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", clientSelectionFailure()
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", clientSelectionFailure()
	}
	root, err := switchCLIRoot(host, home, os.Getenv)
	if err != nil {
		return "", err
	}
	overrides := map[string][]string{
		"qwen-code":   {"QWEN_SETTINGS_PATH", "QWEN_SYSTEM_SETTINGS_PATH", "QWEN_SYSTEM_SETTINGS_DEFAULTS_PATH", "QWEN_CODE_SYSTEM_SETTINGS_PATH", "QWEN_CODE_SYSTEM_DEFAULTS_PATH", "OPENAI_MODEL", "QWEN_MODEL", "ANTHROPIC_MODEL"},
		"copilot-cli": {"COPILOT_PROVIDERS_CONFIG"},
		"mimo-coder":  {"MIMOCODE_CONFIG", "MIMOCODE_CONFIG_DIR", "MIMOCODE_CONFIG_CONTENT", "MIMOCODE_CONFIG_DEFAULTS", "MIMOCODE_AUTH_CONTENT", "MIMOCODE_TEST_MANAGED_CONFIG_DIR", "MIMOCODE_TEST_HOME", "MIMOCODE_CONSOLE_TOKEN"},
	}
	for _, name := range overrides[host] {
		if os.Getenv(name) != "" {
			return "", clientFormatFailure()
		}
	}
	if err := switchCLIOverlays(host, root, home, cwd); err != nil {
		return "", err
	}
	switch host {
	case "paseo":
		return credentialFromPaseoRoot(root)
	case "qwen-code":
		return credentialFromQwenRoot(root)
	case "codebuddy-code":
		return credentialFromCodeBuddyRoot(root)
	case "copilot-cli":
		return credentialFromCopilotRoot(root)
	case "factory-droid":
		return credentialFromDroidRoot(root)
	case "mimo-coder":
		return credentialFromMiMoRoot(root)
	}
	return "", clientSelectionFailure()
}

func credentialFromQwenRoot(root string) (string, error) {
	doc, err := switchCLIObject(filepath.Join(root, "settings.json"), true)
	if err != nil {
		return "", err
	}
	model := jsonObject(doc["model"])
	protocol := jsonString(jsonObject(jsonObject(doc["security"])["auth"])["selectedType"])
	if hasClientOverride(doc, "providerProtocol") || hasClientOverride(jsonObject(jsonObject(doc["security"])["auth"]), "apiKey", "baseUrl") {
		return "", clientFormatFailure()
	}
	if (protocol != "openai" && protocol != "anthropic") || jsonString(model["name"]) == "" || hasClientOverride(model, "baseUrl", "apiKey", "envKey") {
		return "", clientSelectionFailure()
	}
	var selected map[string]any
	for _, raw := range jsonArray(jsonObject(doc["modelProviders"])[protocol]) {
		m := jsonObject(raw)
		if jsonString(m["id"]) == jsonString(model["name"]) {
			if selected != nil {
				return "", switchCLIAmbiguous()
			}
			selected = m
		}
	}
	endpoint := jsonString(selected["baseUrl"])
	route := "/v1"
	if protocol == "anthropic" {
		route = ""
	}
	if !matchesPureTokensEndpoint(endpoint, route) {
		return "", clientSelectionFailure()
	}
	if !switchCLIFields(selected, "id", "name", "baseUrl", "envKey") {
		return "", clientFormatFailure()
	}
	name := jsonString(selected["envKey"])
	if name != "PURETOKENS_SWITCH_QWEN_API_KEY" {
		return "", clientFormatFailure()
	}
	token, err := strictInlineCredential(endpoint, jsonString(jsonObject(doc["env"])[name]))
	if err != nil {
		return "", err
	}
	// Qwen can pass settings.env to its shell. Only a matching inherited value
	// is accepted; never substitute it for the endpoint-verified saved Key.
	if current := os.Getenv(name); current != "" && subtle.ConstantTimeCompare([]byte(token), []byte(current)) != 1 {
		return "", switchCLIAmbiguous()
	}
	return token, nil
}

func credentialFromCodeBuddyRoot(root string) (string, error) {
	value, err := readClientJSON(filepath.Join(root, "models.json"), false)
	if err != nil {
		return "", err
	}
	rows, ok := value.([]any)
	doc := jsonObject(value)
	if !ok {
		rows, ok = doc["models"].([]any)
	}
	if !ok || len(rows) > 512 {
		return "", clientFormatFailure()
	}
	var token string
	ids := map[string]bool{}
	for _, raw := range rows {
		m := jsonObject(raw)
		endpoint := jsonString(m["url"])
		if !matchesPureTokensEndpoint(endpoint, "/v1/chat/completions") {
			if vscodeClaimsPureTokens(endpoint) {
				return "", clientFormatFailure()
			}
			continue
		}
		id := jsonString(m["id"])
		if id == "" || ids[id] || !switchCLIFields(m, "id", "name", "vendor", "url", "apiKey", "ptsManaged", "supportsToolCall", "supportsImages", "supportsReasoning", "onlyReasoning", "useCustomProtocol", "temperature", "maxInputTokens", "maxOutputTokens") || (m["useCustomProtocol"] != nil && m["useCustomProtocol"] != false) {
			return "", clientFormatFailure()
		}
		ids[id] = true
		if available, exists := doc["availableModels"]; exists {
			list, ok := available.([]any)
			if !ok {
				return "", clientFormatFailure()
			}
			if len(list) > 0 {
				found := false
				for _, v := range list {
					if v == id {
						found = true
					}
				}
				if !found {
					continue
				}
			}
		}
		// Only the service identity has matched; provider labels never select Keys.
		if err := switchCLIAddKey(&token, apiOrigin+"/v1", jsonString(m["apiKey"])); err != nil {
			return "", err
		}
	}
	if token == "" {
		return "", clientSelectionFailure()
	}
	return token, nil
}

func credentialFromCopilotRoot(root string) (string, error) {
	doc, err := switchCLIObject(filepath.Join(root, "providers.json"), false)
	if err != nil {
		return "", err
	}
	providers := map[string]map[string]any{}
	for _, raw := range jsonArray(doc["providers"]) {
		p := jsonObject(raw)
		name := jsonString(p["name"])
		if name == "" || providers[name] != nil {
			return "", clientFormatFailure()
		}
		providers[name] = p
	}
	used := map[string]bool{}
	ids := map[string]bool{}
	for _, raw := range jsonArray(doc["models"]) {
		m := jsonObject(raw)
		id, name := jsonString(m["id"]), jsonString(m["provider"])
		if id == "" || providers[name] == nil || ids[name+"/"+id] || !switchCLIFields(m, "id", "provider", "name") {
			return "", clientFormatFailure()
		}
		ids[name+"/"+id] = true
		used[name] = true
	}
	var token string
	for name := range used {
		p := providers[name]
		endpoint := jsonString(p["baseUrl"])
		if !matchesPureTokensEndpoint(endpoint, "", "/v1") {
			if vscodeClaimsPureTokens(endpoint) {
				return "", clientFormatFailure()
			}
			continue
		}
		if !switchCLIFields(p, "name", "type", "baseUrl", "apiKey", "wireApi") {
			return "", clientFormatFailure()
		}
		switch jsonString(p["type"]) {
		case "openai":
			if !matchesPureTokensEndpoint(endpoint, "/v1") || (p["wireApi"] != nil && p["wireApi"] != "responses" && p["wireApi"] != "completions") {
				return "", clientFormatFailure()
			}
		case "anthropic":
			if !matchesPureTokensEndpoint(endpoint, "") || p["wireApi"] != nil {
				return "", clientFormatFailure()
			}
		default:
			return "", clientFormatFailure()
		}
		if err := switchCLIAddKey(&token, endpoint, jsonString(p["apiKey"])); err != nil {
			return "", err
		}
	}
	if token == "" {
		return "", clientSelectionFailure()
	}
	return token, nil
}

func credentialFromDroidRoot(root string) (string, error) {
	doc, err := switchCLIObject(filepath.Join(root, "settings.json"), false)
	if err != nil {
		return "", err
	}
	var token string
	ids := map[string]bool{}
	for _, raw := range jsonArray(doc["customModels"]) {
		m := jsonObject(raw)
		endpoint := jsonString(m["baseUrl"])
		if !matchesPureTokensEndpoint(endpoint, "", "/v1") {
			if vscodeClaimsPureTokens(endpoint) {
				return "", clientFormatFailure()
			}
			continue
		}
		id := jsonString(m["model"])
		if id == "" || ids[id] || !switchCLIFields(m, "model", "displayName", "provider", "baseUrl", "apiKey", "maxOutputTokens") {
			return "", clientFormatFailure()
		}
		ids[id] = true
		switch jsonString(m["provider"]) {
		case "openai", "generic-chat-completion-api":
			if !matchesPureTokensEndpoint(endpoint, "/v1") {
				return "", clientFormatFailure()
			}
		case "anthropic":
			if !matchesPureTokensEndpoint(endpoint, "") {
				return "", clientFormatFailure()
			}
		default:
			return "", clientFormatFailure()
		}
		if err := switchCLIAddKey(&token, endpoint, jsonString(m["apiKey"])); err != nil {
			return "", err
		}
	}
	if token == "" {
		return "", clientSelectionFailure()
	}
	return token, nil
}

func credentialFromMiMoRoot(root string) (string, error) {
	var doc map[string]any
	for _, name := range []string{"config.json", "mimocode.json", "mimocode.jsonc"} {
		candidate, err := switchCLIOptionalObject(filepath.Join(root, name), true)
		if err != nil {
			return "", err
		}
		if switchCLIModelFields("mimo-coder", candidate) {
			if doc != nil {
				return "", clientFormatFailure()
			}
			doc = candidate
		}
	}
	if doc == nil || hasClientOverride(doc, "plugin", "model_groups") {
		return "", clientSelectionFailure()
	}
	providerID, model, ok := strings.Cut(jsonString(doc["model"]), "/")
	if !ok || providerID != "puretokens-switch" || model == "" {
		return "", clientSelectionFailure()
	}
	p := jsonObject(jsonObject(doc["provider"])[providerID])
	options := jsonObject(p["options"])
	endpoint := jsonString(options["baseURL"])
	if !matchesPureTokensEndpoint(endpoint, "/v1") {
		return "", clientSelectionFailure()
	}
	if err := switchMiMoRemoteOverrides(); err != nil {
		return "", err
	}
	if !switchCLIFields(p, "name", "npm", "only_configured_models", "options", "models") || p["npm"] != "@ai-sdk/openai-compatible" || !switchCLIFields(options, "baseURL", "apiKey") {
		return "", clientFormatFailure()
	}
	selected := jsonObject(jsonObject(p["models"])[model])
	if !switchCLIFields(selected, "name", "modalities") {
		return "", clientFormatFailure()
	}
	for _, field := range []string{"disabled_providers", "enabled_providers"} {
		if raw, exists := doc[field]; exists {
			list, ok := raw.([]any)
			if !ok {
				return "", clientFormatFailure()
			}
			found := false
			for _, id := range list {
				if id == providerID {
					found = true
				}
			}
			if field == "enabled_providers" && !found || field == "disabled_providers" && found {
				return "", clientSelectionFailure()
			}
		}
	}
	return strictInlineCredential(endpoint, jsonString(options["apiKey"]))
}

func switchMiMoRemoteOverrides() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return clientSelectionFailure()
	}
	data := os.Getenv("MIMOCODE_HOME")
	if data != "" {
		data = filepath.Join(data, "data")
	} else {
		data = os.Getenv("XDG_DATA_HOME")
		if data == "" {
			data = filepath.Join(home, ".local", "share")
		}
		data = filepath.Join(data, "mimocode")
	}
	if !clientAbsoluteRoot(data) {
		return clientFormatFailure()
	}
	doc, err := switchCLIOptionalObject(filepath.Join(data, "auth.json"), false)
	if err != nil {
		return err
	}
	for _, raw := range doc {
		if jsonString(jsonObject(raw)["type"]) == "wellknown" {
			return clientFormatFailure()
		}
	}
	return nil
}

func switchQwenSystemSettings(goos string) error {
	root := "/etc/qwen-code"
	if goos == "darwin" {
		root = "/Library/Application Support/QwenCode"
	}
	if goos == "windows" {
		// Qwen uses this literal system path, not the ProgramData environment.
		root = `C:\ProgramData\qwen-code`
	}
	if !clientAbsoluteRoot(root) {
		return clientFormatFailure()
	}
	for _, name := range []string{"settings.json", "system-defaults.json"} {
		doc, err := switchCLIOptionalObject(filepath.Join(root, name), true)
		if err != nil {
			return err
		}
		if switchCLIModelFields("qwen-code", doc) {
			return clientFormatFailure()
		}
	}
	return nil
}

func switchQwenDotenv(file string) error {
	if _, err := os.Lstat(file); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return clientFormatFailure()
	}
	data, err := readClientBytes(file)
	if err != nil {
		return err
	}
	defer clear(data)
	if len(data) > 64*1024 || !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return clientFormatFailure()
	}
	text := strings.TrimPrefix(strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n"), "\ufeff")
	for strings.TrimSpace(text) != "" {
		text = strings.TrimLeft(text, " \t\n")
		if strings.HasPrefix(text, "#") {
			_, text, _ = strings.Cut(text, "\n")
			continue
		}
		if strings.HasPrefix(text, "export ") || strings.HasPrefix(text, "export\t") {
			text = strings.TrimLeft(text[6:], " \t")
		}
		end := 0
		for end < len(text) {
			c := text[end]
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-') {
				break
			}
			end++
		}
		if end == 0 {
			return clientFormatFailure()
		}
		key := text[:end]
		text = strings.TrimLeft(text[end:], " \t")
		if strings.HasPrefix(text, "=") {
			text = text[1:]
		} else if strings.HasPrefix(text, ": ") || strings.HasPrefix(text, ":\t") {
			text = text[1:]
		} else {
			return clientFormatFailure()
		}
		for _, blocked := range []string{"PURETOKENS_SWITCH_QWEN_API_KEY", "QWEN_HOME", "QWEN_SETTINGS_PATH", "QWEN_SYSTEM_SETTINGS_PATH", "QWEN_SYSTEM_SETTINGS_DEFAULTS_PATH", "QWEN_CODE_SYSTEM_SETTINGS_PATH", "QWEN_CODE_SYSTEM_DEFAULTS_PATH", "OPENAI_MODEL", "QWEN_MODEL", "ANTHROPIC_MODEL", "HOME", "USERPROFILE"} {
			if key == blocked || runtime.GOOS == "windows" && strings.EqualFold(key, blocked) {
				return clientFormatFailure()
			}
		}
		text = strings.TrimLeft(text, " \t")
		if len(text) > 0 && (text[0] == '\'' || text[0] == '"' || text[0] == '`') {
			quote := text[0]
			text = text[1:]
			escaped, closed := false, false
			for i := 0; i < len(text); i++ {
				if text[i] == quote && !escaped {
					text = strings.TrimLeft(text[i+1:], " \t")
					closed = true
					break
				}
				escaped = text[i] == '\\' && !escaped
			}
			if !closed || text != "" && text[0] != '\n' && text[0] != '#' {
				return clientFormatFailure()
			}
		}
		_, text, _ = strings.Cut(text, "\n")
	}
	return nil
}
