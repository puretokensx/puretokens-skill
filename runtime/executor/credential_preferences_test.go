package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraeConnectionIgnoresChatPreferences(t *testing.T) {
	for _, field := range []string{"display_name", "multimodal", "max_tokens", "prompt_max_tokens", "thinking_enable", "temperature", "top_p", "top_k", "max_turn", "config_detail"} {
		t.Run(field, func(t *testing.T) {
			path := filepath.Join(clientFixtureRoot(t), "state.vscdb")
			traeFixtureDatabase(t, path, func(_, cache map[string]any) {
				model := cache["chat"].([]any)[0].(map[string]any)
				model[field] = 1
				switch field {
				case "display_name":
					model[field] = "My renamed model"
				case "multimodal":
					model[field] = true
				case "config_detail":
					model[field] = map[string]any{"multimodal": true, "max_turn": 100, "model_hyper_params": map[string]any{"thinking_enable": 1, "temperature": 0.4}}
				}
			})
			before, _ := os.ReadFile(path)
			token, err := credentialFromTraeDatabase(path)
			if err != nil || token != "fixture-trae-key" {
				t.Fatalf("unrelated chat preference blocked the saved connection: %v", err)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("credential lookup rewrote host preferences")
			}
		})
	}
}

func TestTraeConnectionChangesStillStop(t *testing.T) {
	for _, field := range []string{"base_url", "provider", "name", "auth_type", "region", "custom_model_type", "config_detail", "nested_auth", "settings_shape"} {
		t.Run(field, func(t *testing.T) {
			path := filepath.Join(clientFixtureRoot(t), "state.vscdb")
			traeFixtureDatabase(t, path, func(status, cache map[string]any) {
				model := cache["chat"].([]any)[0].(map[string]any)
				model[field] = "changed"
				switch field {
				case "config_detail":
					model[field] = map[string]any{"headers": map[string]any{"Authorization": "synthetic-other"}}
				case "nested_auth":
					model["config_detail"] = map[string]any{"model_hyper_params": map[string]any{"api_key": "synthetic-other"}}
				case "settings_shape":
					status["models"].([]any)[0].(map[string]any)["settings"] = []any{}
				}
			})
			if token, err := credentialFromTraeDatabase(path); err == nil || token != "" {
				t.Fatal("changed or unsupported connection released a credential")
			}
		})
	}
}

func ohMyPiPreferenceFixture(t *testing.T) (string, string, map[string]any) {
	t.Helper()
	home := clientFixtureRoot(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{"PI_CONFIG_DIR", "PI_CONFIG_FILES", "PI_PROFILE", "OMP_PROFILE", "PI_CODING_AGENT_DIR", "OMP_CODING_AGENT_DIR", "OMP_CONFIG_DIR"} {
		t.Setenv(name, "")
	}
	project := filepath.Join(home, "project")
	if err := os.MkdirAll(project, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	root := filepath.Join(home, ".omp", "agent")
	desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
	provider := map[string]any{
		"api": "openai-responses", "auth": "apiKey",
		"baseUrl": "https://api.puretokensx.com/v1", "apiKey": "synthetic-omp-key",
		"models": []any{map[string]any{"id": "model-one", "name": "Model"}},
	}
	writeOhMyPiPreferenceProvider(t, root, provider)
	return home, root, provider
}

func writeOhMyPiPreferenceProvider(t *testing.T, root string, provider map[string]any) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"providers": map[string]any{"selected": provider}})
	if err != nil {
		t.Fatal(err)
	}
	desktopFixtureFile(t, root, "models.yml", string(data))
}

func TestOhMyPiAllowsVerifiedHostPreferences(t *testing.T) {
	for _, field := range []string{"name", "reasoning", "contextWindow", "maxTokens", "cost", "disableStrictTools"} {
		t.Run(field, func(t *testing.T) {
			_, root, provider := ohMyPiPreferenceFixture(t)
			model := provider["models"].([]any)[0].(map[string]any)
			switch field {
			case "name":
				model[field] = "My chat model"
			case "reasoning":
				model[field] = true
			case "contextWindow":
				model[field] = 65536
			case "maxTokens":
				model[field] = 4096
			case "cost":
				model[field] = map[string]any{"input": 1, "output": 2, "cacheRead": 0.1, "cacheWrite": 0.2}
			case "disableStrictTools":
				provider[field] = true
			}
			writeOhMyPiPreferenceProvider(t, root, provider)
			before, _ := os.ReadFile(filepath.Join(root, "models.yml"))
			token, err := credentialForHost("oh-my-pi")
			if err != nil || token != "synthetic-omp-key" {
				t.Fatalf("verified host preference blocked the selected connection: %v", err)
			}
			after, _ := os.ReadFile(filepath.Join(root, "models.yml"))
			if string(before) != string(after) {
				t.Fatal("credential lookup rewrote host settings")
			}
		})
	}
}

func TestOhMyPiDotenvAssignments(t *testing.T) {
	cases := []struct {
		name, text string
		allowed    bool
	}{
		{"unrelated", "APP_MODE=development\nUNRELATED_API_KEY=fixture-only\n", true},
		{"quoted", "\ufeff# OMP_PROFILE=ignored\r\nexport TEXT=\"first\nOMP_PROFILE=quoted-text\nlast\"\nAPP=ok\n", true},
		{"unevaluated", "APP=$(do-not-execute)\nTEXT='literal'\n", true},
		{"profile", "OMP_PROFILE=other\n", false},
		{"overlay", "PI_CONFIG_FILES=/other.yml\n", false},
		{"root", "export PI_CONFIG_DIR = other\n", false},
		{"home", "HOME=/other\n", false},
		{"shadow-key", "synthetic-omp-key=synthetic-other\n", false},
		{"malformed", "not an assignment\n", false},
		{"unclosed", "APP=\"unfinished\n", false},
		{"invalid-utf8", "APP=\xff\n", false},
		{"nul", "APP=\x00\n", false},
		{"oversized", "APP=" + strings.Repeat("x", 64*1024), false},
	}
	for _, scope := range []string{"home", "config", "agent", "project", "profile", "custom-agent"} {
		for _, tc := range cases {
			t.Run(scope+"/"+tc.name, func(t *testing.T) {
				home, root, provider := ohMyPiPreferenceFixture(t)
				dir := home
				switch scope {
				case "config":
					dir = filepath.Dir(root)
				case "agent":
					dir = root
				case "project":
					dir = filepath.Join(home, "project")
				case "profile", "custom-agent":
					root = filepath.Join(home, ".omp/profiles/work/agent")
					t.Setenv("OMP_PROFILE", "work")
					if scope == "custom-agent" {
						root = filepath.Join(home, "custom-agent")
						t.Setenv("OMP_PROFILE", "default")
						t.Setenv("PI_CODING_AGENT_DIR", root)
					}
					desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n")
					writeOhMyPiPreferenceProvider(t, root, provider)
					dir = root
				}
				desktopFixtureFile(t, dir, ".env", tc.text)
				token, err := credentialForHost("oh-my-pi")
				if (err == nil) != tc.allowed || tc.allowed && token != "synthetic-omp-key" || !tc.allowed && token != "" {
					t.Fatalf("dotenv assignment boundary incorrect (allowed=%v): %v", tc.allowed, err)
				}
				after, readErr := os.ReadFile(filepath.Join(dir, ".env"))
				if readErr != nil || string(after) != tc.text {
					t.Fatal("dotenv was modified")
				}
			})
		}
	}
}

func TestOhMyPiSelectedConnectionFilters(t *testing.T) {
	for _, tc := range []struct {
		settings string
		allowed  bool
	}{
		{"enabledProviders: [personal, selected]\ndisabledProviders: [unused-personal]\nenabledModels: []\n", true},
		{"enabledModels: [selected/model-one]\n", true},
		{"enabledModels: ['selected/*']\n", true},
		{"enabledModels: ['*']\n", true},
		{"enabledProviders: [other]\n", false},
		{"disabledProviders: [selected]\n", false},
		{"enabledModels: [other/model-one]\n", false},
		{"enabledProviders: [5]\n", false},
		{"enabledModels: wrong-type\n", false},
		{"provider: other\n", false},
		{"session: other\n", false},
	} {
		t.Run(strings.TrimSpace(tc.settings), func(t *testing.T) {
			_, root, _ := ohMyPiPreferenceFixture(t)
			desktopFixtureFile(t, root, "config.yml", "modelRoles: {default: selected/model-one}\n"+tc.settings)
			token, err := credentialForHost("oh-my-pi")
			if (err == nil) != tc.allowed || tc.allowed && token != "synthetic-omp-key" || !tc.allowed && token != "" {
				t.Fatalf("selected connection filter boundary incorrect (allowed=%v): %v", tc.allowed, err)
			}
		})
	}
}

func TestOhMyPiPreferenceAllowlistKeepsConnectionOverridesBlocked(t *testing.T) {
	for _, scope := range []string{"provider", "model"} {
		for _, field := range []string{"headers", "authHeader", "transport", "modelOverrides", "unknownField"} {
			t.Run(scope+"/"+field, func(t *testing.T) {
				_, root, provider := ohMyPiPreferenceFixture(t)
				value := provider
				if scope == "model" {
					value = provider["models"].([]any)[0].(map[string]any)
				}
				value[field] = map[string]any{"apiKey": "synthetic-other"}
				writeOhMyPiPreferenceProvider(t, root, provider)
				if token, err := credentialForHost("oh-my-pi"); err == nil || token != "" {
					t.Fatal("unsupported connection override released a credential")
				}
			})
		}
	}
}

func TestOhMyPiUnsafeDotenvFilesStop(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			home, _, _ := ohMyPiPreferenceFixture(t)
			p := filepath.Join(home, ".env")
			var err error
			if kind == "directory" {
				err = os.Mkdir(p, 0700)
			} else {
				desktopFixtureFile(t, home, "dotenv.fixture", "APP=local\n")
				link := os.Link
				if kind == "symlink" {
					link = os.Symlink
				}
				err = link(filepath.Join(home, "dotenv.fixture"), p)
			}
			if err != nil {
				t.Fatal(err)
			}
			if token, err := credentialForHost("oh-my-pi"); err == nil || token != "" {
				t.Fatal("unsafe dotenv file released a credential")
			}
		})
	}
}
