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
			return "relative.yml"
		}
		return ""
	}); err == nil {
		t.Fatal("relative config overlay accepted")
	}
}

func TestOhMyPiExplicitConfigLayers(t *testing.T) {
	for _, variant := range []string{"selected-overlay", "later-wins", "partial-merge", "different-key", "missing", "malformed", "foreign", "ambiguous", "duplicate", "symlink", "hardlink"} {
		t.Run(variant, func(t *testing.T) {
			home := clientFixtureRoot(t)
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			for _, name := range []string{"PI_CONFIG_DIR", "PI_PROFILE", "OMP_PROFILE", "PI_CODING_AGENT_DIR", "OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR"} {
				t.Setenv(name, "")
			}
			root := filepath.Join(home, ".omp/agent")
			desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: initial/old-model}\n")
			desktopFixtureFile(t, root, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-overlay-key, models: [{id: model-one}]}, foreign: {api: openai-responses, auth: apiKey, baseUrl: 'https://other.invalid/v1', apiKey: never-use, models: [{id: model-one}]}}\n")
			content := "modelRoles: {default: selected/model-one}\n"
			switch variant {
			case "malformed":
				content = "modelRoles: ["
			case "foreign":
				content = "modelRoles: {default: foreign/model-one}\n"
			case "ambiguous":
				content = "modelRoles: {default: [selected/model-one, foreign/model-one]}\n"
			case "partial-merge":
				desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
				content = "modelRoles: {smol: foreign/model-one}\n"
			case "different-key":
				desktopFixtureFile(t, root, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-overlay-key, models: [{id: model-one}]}, initial: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-other-key, models: [{id: old-model}]}}\n")
			case "duplicate":
				content += "modelRoles: {default: foreign/model-one}\n"
			}
			overlay := filepath.Join(home, "overlay.yml")
			if variant != "missing" {
				desktopFixtureFile(t, home, "overlay.yml", content)
			}
			if variant == "symlink" || variant == "hardlink" {
				if err := os.Rename(overlay, overlay+".fixture"); err != nil {
					t.Fatal(err)
				}
				link := os.Link
				if variant == "symlink" {
					link = os.Symlink
				}
				if err := link(overlay+".fixture", overlay); err != nil {
					t.Skip("links unavailable")
				}
			}
			if variant == "later-wins" {
				desktopFixtureFile(t, home, "first.yml", "modelRoles: {default: foreign/model-one}\n")
				overlay = filepath.Join(home, "first.yml") + string(os.PathListSeparator) + overlay
			}
			t.Setenv("PI_CONFIG_FILES", overlay)
			token, err := credentialForHost("oh-my-pi")
			wantOK := variant == "selected-overlay" || variant == "later-wins" || variant == "partial-merge" || variant == "different-key"
			if (err == nil) != wantOK || (wantOK && token != "synthetic-overlay-key") || (!wantOK && token != "") {
				t.Fatal("incorrect explicit configuration precedence")
			}
		})
	}
}

func TestOhMyPiConfigFileBounds(t *testing.T) {
	home := clientFixtureRoot(t)
	file := filepath.Join(home, "overlay with spaces.yml")
	separator := string(os.PathListSeparator)
	for _, count := range []int{1, 4, 5} {
		raw := strings.Repeat(file+separator, count)
		files, err := ohMyPiConfigFiles(raw)
		if (err == nil) != (count <= 4) || (err == nil && len(files) != count) {
			t.Fatal("incorrect explicit overlay bound")
		}
	}
	for _, raw := range []string{"relative.yml", "~/overlay.yml", home + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "other.yml", file + "\n"} {
		if _, err := ohMyPiConfigFiles(raw); err == nil {
			t.Fatal("unsafe overlay path accepted")
		}
	}
}

func TestOhMyPiProfileRoots(t *testing.T) {
	home := clientFixtureRoot(t)
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"named", map[string]string{"OMP_PROFILE": "work"}, filepath.Join(home, ".omp/profiles/work/agent")},
		{"legacy", map[string]string{"PI_PROFILE": "legacy"}, filepath.Join(home, ".omp/profiles/legacy/agent")},
		{"custom-config", map[string]string{"PI_CONFIG_DIR": ".omp-work", "OMP_PROFILE": " work "}, filepath.Join(home, ".omp-work/profiles/work/agent")},
		{"named-ignores-root", map[string]string{"OMP_PROFILE": "work", "PI_CODING_AGENT_DIR": filepath.Join(home, "unrelated")}, filepath.Join(home, ".omp/profiles/work/agent")},
		{"explicit-default", map[string]string{"OMP_PROFILE": "default", "PI_PROFILE": "legacy"}, filepath.Join(home, ".omp/agent")},
		{"traversal", map[string]string{"OMP_PROFILE": "../other"}, ""},
		{"device", map[string]string{"OMP_PROFILE": "con.txt"}, ""},
		{"trailing-dot", map[string]string{"OMP_PROFILE": "work."}, ""},
		{"foreign-config", map[string]string{"PI_CONFIG_DIR": ".pi"}, ""},
		{"config-traversal", map[string]string{"PI_CONFIG_DIR": "../other"}, ""},
		{"stale-profile-root", map[string]string{"OMP_PROFILE": "default", "PI_PROFILE": "legacy", "PI_CODING_AGENT_DIR": filepath.Join(home, ".omp/profiles/legacy/agent")}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ohMyPiRoot(home, func(k string) string { return tc.env[k] })
			if (err == nil) != (tc.want != "") || got != tc.want {
				t.Fatalf("profile root mismatch: error=%v", err)
			}
		})
	}
}

func TestOhMyPiNamedProfileCredentialIsolation(t *testing.T) {
	for _, profile := range []string{"work", ""} {
		t.Run("canonical-"+profile, func(t *testing.T) {
			home := clientFixtureRoot(t)
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			for _, name := range []string{"PI_CONFIG_DIR", "PI_CONFIG_FILES", "PI_CODING_AGENT_DIR", "OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR"} {
				t.Setenv(name, "")
			}
			t.Setenv("OMP_PROFILE", profile)
			t.Setenv("PI_PROFILE", "legacy")
			for _, relative := range []string{".omp/agent", ".omp/profiles/legacy/agent"} {
				sibling := filepath.Join(home, relative)
				desktopFixtureFile(t, sibling, "config.yml", "modelRoles: {default: selected/model-one}\n")
				desktopFixtureFile(t, sibling, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-sibling-key, models: [{id: model-one}]}}\n")
			}
			root := filepath.Join(home, ".omp/agent")
			if profile != "" {
				root = filepath.Join(home, ".omp/profiles", profile, "agent")
			}
			desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
			desktopFixtureFile(t, root, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-profile-key, models: [{id: model-one}]}}\n")
			token, err := credentialForHost("oh-my-pi")
			if err != nil || token != "synthetic-profile-key" {
				t.Fatal("selected profile did not resolve independently")
			}
			if err := os.Remove(filepath.Join(root, "models.yml")); err != nil {
				t.Fatal(err)
			}
			if token, err := credentialForHost("oh-my-pi"); err == nil || token != "" {
				t.Fatal("missing selected profile borrowed a sibling credential")
			}
			locations := doctorHostLocations("oh-my-pi", home, func(k string) string {
				if k == "OMP_PROFILE" && profile == "" {
					return "default"
				}
				return os.Getenv(k)
			})
			if len(locations) != 1 || locations[0].path != filepath.Join(root, "skills") {
				t.Fatal("doctor differs from credential root")
			}
		})
	}
}

func TestOhMyPiProjectOverrideStopsBeforeCredentials(t *testing.T) {
	root := clientFixtureRoot(t)
	desktopFixtureFile(t, root, ".omp/config.yml", "modelRoles: {default: other/model}\n")
	if err := ohMyPiProjectScope(filepath.Join(root, "child"), root); err == nil {
		t.Fatal("project overlay was accepted")
	}
}

func TestOhMyPiProjectDotenvStopsBeforeCredentials(t *testing.T) {
	root := clientFixtureRoot(t)
	desktopFixtureFile(t, root, ".env", "OMP_PROFILE=other\n")
	if err := ohMyPiProjectScope(root, root); err == nil {
		t.Fatal("project dotenv selection was ignored")
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
	for _, variant := range []string{"oh-my-pi", "oh-my-pi-profile-overlay", "minimax-code-cli"} {
		t.Run(variant, func(t *testing.T) {
			host := variant
			if variant == "oh-my-pi-profile-overlay" {
				host = "oh-my-pi"
			}
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
				if variant == "oh-my-pi-profile-overlay" {
					t.Setenv("OMP_PROFILE", "work")
					root = filepath.Join(home, ".omp/profiles/work/agent")
					t.Setenv("PI_CONFIG_FILES", filepath.Join(home, "overlay.yml"))
					desktopFixtureFile(t, home, "overlay.yml", "modelRoles: {default: selected/model-one}\n")
				}
				desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
				desktopFixtureFile(t, root, "models.yml", "providers: {selected: {api: openai-responses, auth: apiKey, baseUrl: 'https://api.puretokensx.com/v1', apiKey: synthetic-omp-key, models: [{id: model-one}]}}\n")
				if variant == "oh-my-pi-profile-overlay" {
					desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: absent/model-one}\n")
				}
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
