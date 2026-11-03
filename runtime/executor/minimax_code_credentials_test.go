package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func miniMaxFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../test/fixtures/switch-minimax-code-connection.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Configuration string }
	if json.Unmarshal(data, &fixture) != nil || fixture.Configuration == "" {
		t.Fatal("invalid synthetic fixture")
	}
	return fixture.Configuration
}

func TestMiniMaxCodeSelectedConnection(t *testing.T) {
	for _, variant := range []string{"writer", "renamed-provider", "selected-slash-model", "different-sibling-key", "no-default", "missing-model", "disabled-model", "disabled-provider", "foreign-endpoint", "dynamic-key", "oauth", "option-headers", "model-headers", "provider-override", "agent-override", "session-override", "malformed", "duplicate-field", "yaml-alias", "yaml-documents", "missing-key", "symlink", "hardlink"} {
		t.Run(variant, func(t *testing.T) {
			root := clientFixtureRoot(t)
			raw := miniMaxFixture(t)
			var cfg map[string]any
			if json.Unmarshal([]byte(raw), &cfg) != nil {
				t.Fatal("invalid fixture")
			}
			providers := jsonObject(cfg["custom_provider"])
			selected := jsonObject(providers["puretokens-switch-responses"])
			options := jsonObject(selected["options"])
			model := jsonObject(jsonObject(selected["models"])["responses-fixture"])
			switch variant {
			case "renamed-provider":
				providers["user-choice"] = selected
				delete(providers, "puretokens-switch-responses")
				cfg["defaultModel"] = "custom_provider:user-choice/responses-fixture"
			case "selected-slash-model":
				cfg["defaultModel"] = "custom_provider:puretokens-switch-anthropic/org/messages-fixture"
			case "different-sibling-key":
				jsonObject(jsonObject(providers["puretokens-switch-chat"])["options"])["apiKey"] = "another-synthetic-key"
			case "no-default":
				delete(cfg, "defaultModel")
			case "missing-model":
				cfg["defaultModel"] = "custom_provider:puretokens-switch-responses/not-present"
			case "disabled-model":
				model["enabled"] = false
			case "disabled-provider":
				selected["enabled"] = false
			case "foreign-endpoint":
				options["baseURL"] = "https://example.invalid/v1"
			case "dynamic-key":
				options["apiKey"] = "${TEST_SYNTHETIC_TOKEN}"
			case "oauth":
				selected["kind"] = "oauth"
			case "option-headers":
				options["headers"] = map[string]any{}
			case "model-headers":
				model["headers"] = map[string]any{}
			case "provider-override":
				selected["baseURL"] = "https://example.invalid/v1"
			case "agent-override":
				cfg["agents"] = map[string]any{"default": map[string]any{"model": "different"}}
			case "session-override":
				cfg["effectiveModel"] = "other/model"
			case "missing-key":
				delete(options, "apiKey")
			}
			data, _ := json.Marshal(cfg)
			raw = string(data)
			switch variant {
			case "malformed":
				raw = "custom_provider: ["
			case "duplicate-field":
				raw = strings.Replace(raw, `"defaultModel":`, `"defaultModel":"other", "defaultModel":`, 1)
			case "yaml-alias":
				raw = "value: &x {}\ncustom_provider: *x\n"
			case "yaml-documents":
				raw += "\n---\n{}"
			}
			clientFixtureFile(t, root, "config.yaml", raw)
			if variant == "symlink" || variant == "hardlink" {
				if err := os.Rename(filepath.Join(root, "config.yaml"), filepath.Join(root, "other.yaml")); err != nil {
					t.Fatal(err)
				}
				link := os.Link
				if variant == "symlink" {
					link = os.Symlink
				}
				if err := link(filepath.Join(root, "other.yaml"), filepath.Join(root, "config.yaml")); err != nil {
					t.Skip("filesystem link not available")
				}
			}
			key, err := credentialFromMiniMaxCodeRoot(root)
			ok := variant == "writer" || variant == "renamed-provider" || variant == "selected-slash-model" || variant == "different-sibling-key"
			if (err == nil) != ok || (ok && key != "pts-minimax-fixture-not-real") || (!ok && key != "") {
				t.Fatal("incorrect selected connection boundary")
			}
			after, _ := os.ReadFile(filepath.Join(root, "config.yaml"))
			if string(after) != raw {
				t.Fatal("configuration modified")
			}
			if err != nil {
				code, message, next := credentialFailureDetails(err)
				if strings.Contains(code+message+next, "fixture-not-real") {
					t.Fatal("secret disclosed")
				}
			}
		})
	}
}

func TestMiniMaxCodeDesktopRoots(t *testing.T) {
	for _, goos := range []string{"darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			home := clientFixtureRoot(t)
			appdata := filepath.Join(home, "Roaming")
			vars := map[string]string{"APPDATA": appdata}
			getenv := func(k string) string { return vars[k] }
			got, err := miniMaxCodeRoot(goos, home, getenv)
			if err != nil || got != filepath.Join(home, ".minimax") {
				t.Fatal("default root")
			}
			prefs := filepath.Join(home, "Library", "Application Support")
			if goos == "windows" {
				prefs = appdata
			}
			parent := filepath.Join(home, "Custom Space")
			body, _ := json.Marshal(map[string]any{"config": map[string]any{"localRuntimeDataParentDir": parent}})
			if err := os.MkdirAll(filepath.Join(prefs, "MiniMax Code"), 0700); err != nil {
				t.Fatal(err)
			}
			file := "MiniMax Code/minimax-agent-cn-config.json"
			clientFixtureFile(t, prefs, file, string(body))
			got, err = miniMaxCodeRoot(goos, home, getenv)
			if err != nil || got != filepath.Join(parent, ".minimax") {
				t.Fatal("saved preference root")
			}
			if err := os.MkdirAll(filepath.Join(prefs, "MiniMax"), 0700); err != nil {
				t.Fatal(err)
			}
			clientFixtureFile(t, prefs, "MiniMax/minimax-agent-config.json", `{"config":{"localRuntimeDataParentDir":"`+filepath.ToSlash(home)+`"}}`)
			if _, err = miniMaxCodeRoot(goos, home, getenv); err == nil {
				t.Fatal("conflicting regional preferences accepted")
			}
			vars["MAVIS_DATA_DIR"] = filepath.Join(home, "legacy-override")
			vars["MINIMAX_DATA_DIR"] = filepath.Join(home, "current-override")
			got, err = miniMaxCodeRoot(goos, home, getenv)
			if err != nil || got != vars["MINIMAX_DATA_DIR"] {
				t.Fatal("override priority")
			}
			for _, bad := range []string{"relative", home + "/../other", home + "\ncontrol"} {
				vars["MINIMAX_DATA_DIR"] = bad
				if _, err = miniMaxCodeRoot(goos, home, getenv); err == nil {
					t.Fatal("unsafe root accepted")
				}
			}
			vars["MINIMAX_DATA_DIR"] = parent
			vars["MAVIS_PROFILE"] = "other"
			if _, err = miniMaxCodeRoot(goos, home, getenv); err == nil {
				t.Fatal("CLI profile accepted")
			}
		})
	}
	home := clientFixtureRoot(t)
	getenv := func(string) string { return "" }
	if _, err := miniMaxCodeRoot("linux", home, getenv); err == nil {
		t.Fatal("unsupported Desktop OS")
	}
	if err := os.Mkdir(filepath.Join(home, ".mavis"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := miniMaxCodeRoot("darwin", home, getenv); err == nil {
		t.Fatal("legacy migration bypassed")
	}
	home = clientFixtureRoot(t)
	prefs := filepath.Join(home, "Library", "Application Support", "MiniMax Code")
	if err := os.MkdirAll(prefs, 0700); err != nil {
		t.Fatal(err)
	}
	clientFixtureFile(t, prefs, "minimax-agent-config.json", `{"config":{},"config":{}}`)
	if _, err := miniMaxCodeRoot("darwin", home, getenv); err == nil {
		t.Fatal("duplicate preference accepted")
	}
}

func TestMiniMaxCodeDispatcherDoesNotRetryOrBorrowKey(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("Desktop platform only")
	}
	root := clientFixtureRoot(t)
	t.Setenv("MINIMAX_DATA_DIR", root)
	for _, name := range []string{"MAVIS_PROFILE", "MINIMAX_PROFILE", "AGENTARCHON_PROFILE", "AGENTARCHON_DATA_DIR", "__MAVIS_RUNTIME_PROFILE", "__MAVIS_RUNTIME_DATA_DIR"} {
		t.Setenv(name, "")
	}
	clientFixtureFile(t, root, "config.yaml", miniMaxFixture(t))
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	exerciseDedicatedHostPermissions(t, "minimax-code", root, "pts-minimax-fixture-not-real")
	t.Setenv("MAVIS_PROFILE", "other")
	calls := 0
	http.DefaultTransport = evaluationTransport(func(*http.Request) (*http.Response, error) { calls++; return evaluationHTTP(500, `{}`), nil })
	var out bytes.Buffer
	if run([]string{"models", "--host", "minimax-code"}, strings.NewReader(""), &out) == nil || calls != 0 {
		t.Fatal("override did not stop before network")
	}
}

func TestMiniMaxCLIRootPreservesPendingLegacyMigration(t *testing.T) {
	for _, variant := range []string{"absent", "legacy", "legacy-link", "current", "current-link", "explicit", "whitespace-explicit", "whitespace-fallback"} {
		t.Run(variant, func(t *testing.T) {
			home := clientFixtureRoot(t)
			vars := map[string]string{}
			getenv := func(name string) string { return vars[name] }
			want := filepath.Join(home, ".minimax")
			if variant != "absent" {
				if variant == "legacy-link" {
					if err := os.Symlink(filepath.Join(home, "missing"), filepath.Join(home, ".mavis")); err != nil {
						t.Skip("filesystem link not available")
					}
				} else if err := os.Mkdir(filepath.Join(home, ".mavis"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			switch variant {
			case "current":
				desktopFixtureFile(t, want, "config.yaml", miniMaxFixture(t))
			case "current-link":
				if err := os.Symlink(filepath.Join(home, "missing"), want); err != nil {
					t.Skip("filesystem link not available")
				}
			case "explicit", "whitespace-explicit", "whitespace-fallback":
				want = filepath.Join(home, "selected root")
				vars["MINIMAX_DATA_DIR"] = want
				if variant == "whitespace-explicit" {
					vars["MINIMAX_DATA_DIR"] = " \t" + want + " \t"
				} else if variant == "whitespace-fallback" {
					vars["MINIMAX_DATA_DIR"] = " \t"
					vars["MAVIS_DATA_DIR"] = " \t" + want + " \t"
				}
			}
			got, err := miniMaxCLIRoot(home, getenv)
			stop := variant == "legacy" || variant == "legacy-link" || variant == "current-link"
			if (err != nil) != stop || (!stop && got != want) || (stop && got != "") {
				t.Fatal("CLI root did not preserve explicit selection or pending migration")
			}
			if variant != "current" {
				if _, err := os.Stat(filepath.Join(home, ".minimax", "config.yaml")); !os.IsNotExist(err) {
					t.Fatal("root resolution wrote a configuration")
				}
			}
		})
	}
}
