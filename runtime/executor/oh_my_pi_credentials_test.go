package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOhMyPiRejectsConfigOverlay(t *testing.T) {
	home := clientFixtureRoot(t)
	if _, err := ohMyPiRoot(home, func(k string) string {
		if k == "PI_CONFIG_FILES" {
			return filepath.Join(home, "override.yml")
		}
		return ""
	}); err == nil {
		t.Fatal("config overlay accepted")
	}
}

func TestOhMyPiProjectOverrideStopsBeforeCredentials(t *testing.T) {
	root := clientFixtureRoot(t)
	desktopFixtureFile(t, root, ".omp/config.yml", "modelRoles: {default: other/model}\n")
	if err := ohMyPiProjectScope(filepath.Join(root, "child"), root); err == nil {
		t.Fatal("project overlay was accepted")
	}
}

func TestOhMyPiSelectedConnectionBoundaries(t *testing.T) {
	base := "providers:\n  selected:\n    api: openai-responses\n    auth: apiKey\n    baseUrl: https://api.puretokensx.com/v1\n    apiKey: synthetic-omp-key\n    models: [{id: model-one, name: Model}]\n"
	for _, variant := range []string{"yaml-fallback", "yml-priority", "foreign-endpoint", "dynamic-key", "padded-key", "headers", "oauth", "duplicate-model", "missing-model", "model-override", "duplicate-yaml", "symlink", "hardlink"} {
		t.Run(variant, func(t *testing.T) {
			root := clientFixtureRoot(t)
			settings, models := "modelRoles: {default: selected/model-one}\n", base
			suffix := ".yml"
			switch variant {
			case "yaml-fallback":
				suffix = ".yaml"
			case "yml-priority":
				desktopFixtureFile(t, root, "models.yaml", "invalid: [")
			case "foreign-endpoint":
				models = strings.ReplaceAll(models, "https://api.puretokensx.com/v1", "https://other.invalid/v1")
			case "dynamic-key":
				models = strings.ReplaceAll(models, "synthetic-omp-key", "${SYNTHETIC_KEY}")
			case "padded-key":
				models = strings.ReplaceAll(models, "synthetic-omp-key", "' synthetic-omp-key '")
			case "headers":
				models += "    headers: {}\n"
			case "oauth":
				models = strings.ReplaceAll(models, "auth: apiKey", "auth: oauth")
			case "duplicate-model":
				models = strings.ReplaceAll(models, "[{id: model-one, name: Model}]", "[{id: model-one}, {id: model-one}]")
			case "missing-model":
				settings = "modelRoles: {default: selected/absent}\n"
			case "model-override":
				models = strings.ReplaceAll(models, "name: Model", "name: Model, apiKey: other")
			case "duplicate-yaml":
				settings += "modelRoles: {}\n"
			}
			desktopFixtureFile(t, root, "config"+suffix, settings)
			desktopFixtureFile(t, root, "models"+suffix, models)
			if variant == "symlink" || variant == "hardlink" {
				p := filepath.Join(root, "models"+suffix)
				if err := os.Rename(p, p+".fixture"); err != nil {
					t.Fatal(err)
				}
				link := os.Link
				if variant == "symlink" {
					link = os.Symlink
				}
				if err := link(p+".fixture", p); err != nil {
					t.Skip("links unavailable")
				}
			}
			key, err := credentialFromOhMyPiRoot(root)
			wantOK := variant == "yaml-fallback" || variant == "yml-priority"
			if (err == nil) != wantOK || (wantOK && key != "synthetic-omp-key") || (!wantOK && key != "") {
				t.Fatal("incorrect connection boundary")
			}
		})
	}
}

func TestOhMyPiRejectsEnvironmentBackedCredentials(t *testing.T) {
	for _, token := range []string{"SYNTHETIC_OMP_KEY", "synthetic_omp_key", "synthetic-omp-shadow"} {
		t.Run(token, func(t *testing.T) {
			root := clientFixtureRoot(t)
			if token == "synthetic-omp-shadow" {
				t.Setenv(token, "synthetic-other-value")
			}
			desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
			desktopFixtureFile(t, root, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: "+token+", models: [{id: model-one}]}}\n")
			key, err := credentialFromOhMyPiRoot(root)
			if err == nil || key != "" {
				t.Fatal("ambiguous environment-backed credential was accepted")
			}
		})
	}
}

func TestNewCLIHostDedicatedRoutes(t *testing.T) {
	for _, host := range []string{"oh-my-pi", "minimax-code-cli"} {
		t.Run(host, func(t *testing.T) {
			home := clientFixtureRoot(t)
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			for _, name := range []string{"PI_CONFIG_DIR", "PI_CONFIG_FILES", "PI_PROFILE", "OMP_PROFILE", "PI_CODING_AGENT_DIR", "OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR", "MINIMAX_DATA_DIR", "MAVIS_DATA_DIR", "MAVIS_PROFILE", "MINIMAX_PROFILE", "AGENTARCHON_PROFILE", "AGENTARCHON_DATA_DIR", "__MAVIS_RUNTIME_PROFILE", "__MAVIS_RUNTIME_DATA_DIR"} {
				t.Setenv(name, "")
			}
			key := "pts-minimax-fixture-not-real"
			if host == "oh-my-pi" {
				key = "synthetic-omp-key"
				root := filepath.Join(home, ".omp", "agent")
				desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
				desktopFixtureFile(t, root, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-omp-key, models: [{id: model-one}]}}\n")
			} else {
				desktopFixtureFile(t, filepath.Join(home, ".minimax"), "config.yaml", miniMaxFixture(t))
			}
			previous := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = previous })
			exerciseDedicatedHostPermissions(t, host, home, key)
		})
	}
}

func TestOhMyPiDefaultConnection(t *testing.T) {
	for _, protocol := range []string{"openai-completions", "openai-responses", "anthropic-messages"} {
		t.Run(protocol, func(t *testing.T) {
			home := clientFixtureRoot(t)
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			for _, name := range []string{"PI_CONFIG_DIR", "PI_PROFILE", "OMP_PROFILE", "PI_CODING_AGENT_DIR", "OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR"} {
				t.Setenv(name, "")
			}
			root := filepath.Join(home, ".omp", "agent")
			desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
			desktopFixtureFile(t, root, "models.yml", "providers:\n  selected:\n    api: "+protocol+"\n    auth: apiKey\n    baseUrl: https://api.puretokensx.com/v1\n    apiKey: synthetic-omp-key\n    models: [{id: model-one, name: Model}]\n  sibling:\n    apiKey: synthetic-other-key\n    baseUrl: https://other.invalid/v1\n")
			token, err := credentialForHost("oh-my-pi")
			if err != nil || token != "synthetic-omp-key" {
				t.Fatal("Oh My Pi default selected connection did not resolve")
			}
		})
	}
}

func TestMiniMaxCLIDoesNotUseDesktopPreference(t *testing.T) {
	home := clientFixtureRoot(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{"MINIMAX_DATA_DIR", "MAVIS_DATA_DIR", "MAVIS_PROFILE", "MINIMAX_PROFILE", "AGENTARCHON_PROFILE", "AGENTARCHON_DATA_DIR", "__MAVIS_RUNTIME_PROFILE", "__MAVIS_RUNTIME_DATA_DIR"} {
		t.Setenv(name, "")
	}
	desktopFixtureFile(t, filepath.Join(home, ".minimax"), "config.yaml", miniMaxFixture(t))
	desktopFixtureFile(t, filepath.Join(home, "Library", "Application Support", "MiniMax Code"), "minimax-agent-config.json", `{"config":{"localRuntimeDataParentDir":"/synthetic-unrelated-desktop"}}`)
	token, err := credentialForHost("minimax-code-cli")
	if err != nil || token == "" {
		t.Fatal("MiniMax CLI did not resolve its independent default root")
	}
}
