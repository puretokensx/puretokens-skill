package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var switchCLIFixtures = map[string]struct{ file, body string }{
	"paseo":          {".paseo/config.json", `{"agents":{"providers":{"puretokens":{"extends":"codex","label":"Pure Tokens","description":"Managed by Pure Tokens Switch","env":{"OPENAI_BASE_URL":"https://api.puretokensx.com","OPENAI_API_KEY":"synthetic-switch-key"},"models":[{"id":"chat-one"}]}}}}`},
	"qwen-code":      {".qwen/settings.json", `{"model":{"name":"chat-one"},"security":{"auth":{"selectedType":"openai"}},"modelProviders":{"openai":[{"id":"chat-one","baseUrl":"https://api.puretokensx.com/v1","envKey":"PURETOKENS_SWITCH_QWEN_API_KEY"}]},"env":{"PURETOKENS_SWITCH_QWEN_API_KEY":"synthetic-switch-key"}}`},
	"codebuddy-code": {".codebuddy/models.json", `[{"id":"chat-one","url":"https://api.puretokensx.com/v1/chat/completions","apiKey":"synthetic-switch-key","supportsToolCall":true},{"id":"chat-two","url":"https://api.puretokensx.com/v1/chat/completions","apiKey":"synthetic-switch-key"}]`},
	"copilot-cli":    {".copilot/providers.json", `{"providers":[{"name":"saved-chat","type":"openai","wireApi":"completions","baseUrl":"https://api.puretokensx.com/v1","apiKey":"synthetic-switch-key"},{"name":"saved-messages","type":"anthropic","baseUrl":"https://api.puretokensx.com","apiKey":"synthetic-switch-key"}],"models":[{"id":"chat-one","provider":"saved-chat"},{"id":"chat-two","provider":"saved-messages"}]}`},
	"factory-droid":  {".factory/settings.json", `{"customModels":[{"model":"chat-one","displayName":"First","provider":"openai","baseUrl":"https://api.puretokensx.com/v1","apiKey":"synthetic-switch-key"},{"model":"chat-two","displayName":"Second","provider":"anthropic","baseUrl":"https://api.puretokensx.com","apiKey":"synthetic-switch-key"}]}`},
	"mimo-coder":     {".config/mimocode/mimocode.jsonc", `{"model":"puretokens-switch/chat-one","small_model":"puretokens-switch/chat-one","provider":{"puretokens-switch":{"npm":"@ai-sdk/openai-compatible","only_configured_models":true,"options":{"baseURL":"https://api.puretokensx.com/v1","apiKey":"synthetic-switch-key"},"models":{"chat-one":{"name":"chat-one"}}}}}`},
}

func setupSwitchCLI(t *testing.T, host string) (string, string) {
	t.Helper()
	home := clientFixtureRoot(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PASEO_HOME", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ProgramData", filepath.Join(home, "program-data"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("MIMOCODE_TEST_HOME", "")
	t.Setenv("MIMOCODE_CONSOLE_TOKEN", "")
	t.Setenv("QWEN_CODE_SYSTEM_SETTINGS_PATH", "")
	t.Setenv("QWEN_CODE_SYSTEM_DEFAULTS_PATH", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("QWEN_MODEL", "")
	t.Setenv("ANTHROPIC_MODEL", "")
	if host == "paseo" {
		t.Setenv("PASEO_HOME", filepath.Join(home, ".paseo"))
		t.Setenv("OPENAI_BASE_URL", apiOrigin)
		t.Setenv("OPENAI_API_KEY", "synthetic-switch-key")
	}
	for _, key := range []string{"QWEN_HOME", "QWEN_SETTINGS_PATH", "QWEN_SYSTEM_SETTINGS_PATH", "QWEN_SYSTEM_SETTINGS_DEFAULTS_PATH", "PURETOKENS_SWITCH_QWEN_API_KEY", "CODEBUDDY_CONFIG_DIR", "COPILOT_HOME", "COPILOT_PROVIDERS_CONFIG", "FACTORY_HOME_OVERRIDE", "MIMOCODE_HOME", "XDG_CONFIG_HOME", "MIMOCODE_CONFIG", "MIMOCODE_CONFIG_DIR", "MIMOCODE_CONFIG_CONTENT", "MIMOCODE_CONFIG_DEFAULTS", "MIMOCODE_AUTH_CONTENT", "MIMOCODE_TEST_MANAGED_CONFIG_DIR"} {
		t.Setenv(key, "")
	}
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	f := switchCLIFixtures[host]
	desktopFixtureFile(t, home, f.file, f.body)
	return home, filepath.Join(home, f.file)
}

func TestSwitchCLIConnectionsAndDedicatedRoutes(t *testing.T) {
	for host := range switchCLIFixtures {
		t.Run(host, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, host)
			token, err := credentialForHost(host)
			if err != nil || token != "synthetic-switch-key" {
				t.Fatalf("saved connection did not resolve: %v", err)
			}
			previous := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = previous })
			exerciseDedicatedHostPermissions(t, host, home, token)
		})
	}
}

func TestSwitchCLIRejectsWrongOriginDynamicKeysAndDuplicateFields(t *testing.T) {
	for host, f := range switchCLIFixtures {
		for _, variant := range []string{"origin", "dynamic", "duplicate"} {
			t.Run(host+"/"+variant, func(t *testing.T) {
				_, file := setupSwitchCLI(t, host)
				body := f.body
				switch variant {
				case "origin":
					body = strings.ReplaceAll(body, "api.puretokensx.com", "foreign.invalid")
				case "dynamic":
					body = strings.ReplaceAll(body, "synthetic-switch-key", "$OTHER_KEY")
				case "duplicate":
					if host == "paseo" {
						body = strings.Replace(body, `"OPENAI_API_KEY":`, `"OPENAI_API_KEY":"first","OPENAI_API_KEY":`, 1)
					}
					body = strings.Replace(body, `"apiKey":`, `"apiKey":"first","apiKey":`, 1)
					if host == "qwen-code" {
						body = strings.Replace(body, `"env":`, `"env":{},"env":`, 1)
					}
				}
				if err := os.WriteFile(file, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
				if key, err := credentialForHost(host); err == nil || key != "" {
					t.Fatal("unsafe connection accepted")
				}
			})
		}
	}
}

func TestSwitchCLIMultipleDifferentKeysStop(t *testing.T) {
	for _, host := range []string{"codebuddy-code", "copilot-cli", "factory-droid"} {
		t.Run(host, func(t *testing.T) {
			_, file := setupSwitchCLI(t, host)
			body := strings.Replace(switchCLIFixtures[host].body, "synthetic-switch-key", "synthetic-other-key", 1)
			if err := os.WriteFile(file, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			key, err := credentialForHost(host)
			if err == nil || key != "" {
				t.Fatal("different keys accepted")
			}
			code, _, _ := credentialFailureDetails(err)
			if code != "active_connection_ambiguous" {
				t.Fatalf("wrong diagnostic: %s", code)
			}
		})
	}
}

func TestSwitchCLIProjectAndEnvironmentOverridesStop(t *testing.T) {
	for _, tc := range []struct{ host, file, body, env string }{
		{"qwen-code", ".qwen/settings.json", `{"model":{"name":"other"}}`, ""},
		{"qwen-code", ".env", "PURETOKENS_SWITCH_QWEN_API_KEY=other\n", ""},
		{"qwen-code", "", "", "PURETOKENS_SWITCH_QWEN_API_KEY"},
		{"qwen-code", "", "", "QWEN_CODE_SYSTEM_SETTINGS_PATH"},
		{"qwen-code", "", "", "QWEN_CODE_SYSTEM_DEFAULTS_PATH"},
		{"qwen-code", "", "", "OPENAI_MODEL"},
		{"codebuddy-code", ".codebuddy/models.json", `[]`, ""},
		{"copilot-cli", "", "", "COPILOT_PROVIDERS_CONFIG"},
		{"factory-droid", ".factory/settings.local.json", `{"customModels":[]}`, ""},
		{"factory-droid", "", "", "FACTORY_HOME_OVERRIDE"},
		{"mimo-coder", "mimocode.json", `{"model":"other/model"}`, ""},
		{"mimo-coder", "", "", "MIMOCODE_CONFIG_CONTENT"},
	} {
		t.Run(tc.host+"/"+tc.file+tc.env, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, tc.host)
			if tc.file != "" {
				desktopFixtureFile(t, filepath.Join(home, "work"), tc.file, tc.body)
			}
			if tc.env != "" {
				t.Setenv(tc.env, "unconfirmed")
			}
			if key, err := credentialForHost(tc.host); err == nil || key != "" {
				t.Fatal("unconfirmed override accepted")
			}
		})
	}
}

func TestSwitchCLIUnrelatedPreferencesDoNotBlock(t *testing.T) {
	for _, host := range []string{"qwen-code", "factory-droid", "mimo-coder"} {
		t.Run(host, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, host)
			file := map[string]string{"qwen-code": ".qwen/settings.json", "factory-droid": ".factory/settings.local.json", "mimo-coder": "mimocode.json"}[host]
			desktopFixtureFile(t, filepath.Join(home, "work"), file, `{"theme":"dark"}`)
			if token, err := credentialForHost(host); err != nil || token != "synthetic-switch-key" {
				t.Fatal("unrelated preference rejected")
			}
		})
	}
}

func TestSwitchCLIFixturesContainNoRealCredentials(t *testing.T) {
	for _, f := range switchCLIFixtures {
		var v any
		if json.Unmarshal([]byte(f.body), &v) != nil {
			t.Fatal("invalid fixture")
		}
	}
}

func TestSwitchCLIHarmlessAdditionalFiles(t *testing.T) {
	for _, tc := range []struct{ host, file, body string }{
		{"qwen-code", "work/.env", "EDITOR=vim\nFEATURE_FLAG=true\n"},
		{"factory-droid", ".factory/config.json", `{"custom_models":[]}`},
		{"mimo-coder", ".config/mimocode/config.json", `{"provider":{"personal":{"name":"keep"}},"theme":"dark"}`},
	} {
		t.Run(tc.host, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, tc.host)
			desktopFixtureFile(t, home, tc.file, tc.body)
			if key, err := credentialForHost(tc.host); err != nil || key != "synthetic-switch-key" {
				t.Fatalf("harmless overlay rejected: %v", err)
			}
		})
	}
}

func TestSwitchCLIPaseoRequiresSameSessionConnection(t *testing.T) {
	for _, name := range []string{"PASEO_HOME", "OPENAI_BASE_URL", "OPENAI_API_KEY"} {
		for _, value := range []string{"", "unrelated"} {
			t.Run(name+value, func(t *testing.T) {
				setupSwitchCLI(t, "paseo")
				t.Setenv(name, value)
				if key, err := credentialForHost("paseo"); err == nil || key != "" {
					t.Fatal("unbound Paseo session accepted")
				}
			})
		}
	}
}

func TestSwitchCLIActualWriterFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/switch-cli-writers.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Cases []struct {
			Host, Platform string
			Files          map[string]string
		}
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures.Cases) != 10 {
		t.Fatal("expected five writers for two platforms")
	}
	for _, f := range fixtures.Cases {
		t.Run(f.Host+"/"+f.Platform, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, f.Host)
			for name, body := range f.Files {
				desktopFixtureFile(t, home, strings.TrimPrefix(name, "home/"), body)
			}
			want := "pts-fixture-only-token"
			if f.Host == "mimo-coder" {
				want = "pts-mimo-fixture-token"
			}
			key, err := credentialForHost(f.Host)
			if err != nil || key != want {
				t.Fatalf("Switch writer did not resolve: %v", err)
			}
			for name, body := range f.Files {
				actual, err := os.ReadFile(filepath.Join(home, strings.TrimPrefix(name, "home/")))
				if err != nil || string(actual) != body {
					t.Fatal("source config was modified")
				}
			}
		})
	}
}

func TestSwitchCLIMiMoRemoteAuthAndSubstitutionStop(t *testing.T) {
	for _, tc := range []struct{ file, body string }{
		{".local/share/mimocode/auth.json", `{"https://remote.invalid":{"type":"wellknown","key":"REMOTE","token":"synthetic"}}`},
		{".config/mimocode/config.json", `{"theme":"{file:/unreadable}"}`},
	} {
		t.Run(tc.file, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, "mimo-coder")
			desktopFixtureFile(t, home, tc.file, tc.body)
			if key, err := credentialForHost("mimo-coder"); err == nil || key != "" {
				t.Fatal("remote or substituted config accepted")
			}
		})
	}
}

func TestSwitchCLIRejectsLinkedConfig(t *testing.T) {
	for host := range switchCLIFixtures {
		t.Run(host, func(t *testing.T) {
			_, file := setupSwitchCLI(t, host)
			target := file + ".original"
			if err := os.Rename(file, target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, file); err != nil {
				t.Skip("symlink unavailable")
			}
			if key, err := credentialForHost(host); err == nil || key != "" {
				t.Fatal("linked config accepted")
			}
		})
	}
}

func TestSwitchCLIMalformedProviderOverlayStops(t *testing.T) {
	for _, body := range []string{`{"provider":null}`, `{"provider":[]}`, `{"provider":"external"}`} {
		t.Run(body, func(t *testing.T) {
			home, _ := setupSwitchCLI(t, "mimo-coder")
			desktopFixtureFile(t, home, ".config/mimocode/mimocode.json", body)
			if key, err := credentialForHost("mimo-coder"); err == nil || key != "" {
				t.Fatal("malformed higher-priority provider was ignored")
			}
		})
	}
}

func TestSwitchCLIMissingKeysStop(t *testing.T) {
	for host, f := range switchCLIFixtures {
		t.Run(host, func(t *testing.T) {
			_, file := setupSwitchCLI(t, host)
			body := strings.ReplaceAll(f.body, "synthetic-switch-key", "")
			if err := os.WriteFile(file, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			if key, err := credentialForHost(host); err == nil || key != "" {
				t.Fatal("missing credential accepted")
			}
		})
	}
}
