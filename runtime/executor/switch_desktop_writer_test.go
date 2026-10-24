package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Committed synthetic Switch writer fixtures, never installed host configuration.
func switchDesktopWriterFiles(t *testing.T, host, platform, state string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../test/fixtures/switch-client-configs", host, platform+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Client   string `json:"client"`
		Platform string `json:"platform"`
		States   map[string]struct {
			Files map[string]string `json:"files"`
		} `json:"states"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	files := fixture.States[state].Files
	if fixture.Client != host || fixture.Platform != platform || len(files) == 0 {
		t.Fatal("missing or mismatched synthetic writer fixture")
	}
	return files
}

func assertSwitchFixtureUnchanged(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		after, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(after) != body {
			t.Fatal("read-only adapter changed a fixture")
		}
	}
}

func assertSwitchWriterCredential(t *testing.T, token string, err error, valid bool) {
	t.Helper()
	if valid && (err != nil || token != "pts-token") {
		t.Fatal("current Switch writer connection did not resolve")
	}
	if !valid {
		if err == nil || token != "" {
			t.Fatal("invalid active selection borrowed another connection")
		}
		code, message, next := credentialFailureDetails(err)
		if strings.Contains(code+message+next, "pts-token") || strings.Contains(code+message+next, "fixture-other") {
			t.Fatal("failure disclosed a fixture credential")
		}
	}
}

func TestSwitchClaudeDesktopWriterCompatibility(t *testing.T) {
	for _, platform := range []string{"macos", "windows"} {
		for _, variant := range []string{"default-first", "reordered-models", "legacy-default", "foreign-active-profile", "missing-active-profile"} {
			t.Run(platform+"/"+variant, func(t *testing.T) {
				root := t.TempDir()
				files := map[string]string{}
				for name, body := range switchDesktopWriterFiles(t, "claude-desktop", platform, "configured") {
					_, relative, ok := strings.Cut(name, "Claude-3p/")
					if !ok {
						t.Fatal("unexpected fixture path")
					}
					files[relative] = body
				}
				profilePath := "configLibrary/" + fixtureDesktopProfile + ".json"
				var profile map[string]any
				if json.Unmarshal([]byte(files[profilePath]), &profile) != nil {
					t.Fatal("invalid synthetic profile")
				}
				if _, exists := profile["inferenceDefaultModel"]; exists {
					t.Fatal("current writer emits the retired default field")
				}
				models := profile["inferenceModels"].([]any)
				if len(models) != 3 || models[0].(map[string]any)["name"] != "claude-sonnet-4.6" {
					t.Fatal("current writer lost default-first ordering")
				}
				switch variant {
				case "reordered-models":
					models[0], models[1] = models[1], models[0]
					encoded, err := json.Marshal(profile)
					if err != nil {
						t.Fatal(err)
					}
					files[profilePath] = string(encoded)
				case "legacy-default":
					for name, body := range switchDesktopWriterFiles(t, "claude-desktop", platform, "legacy_default") {
						_, relative, ok := strings.Cut(name, "Claude-3p/")
						if !ok || relative != profilePath {
							t.Fatal("unexpected legacy fixture path")
						}
						files[relative] = body
					}
				case "foreign-active-profile":
					files["configLibrary/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.json"] = files[profilePath]
					files[profilePath] = strings.Replace(files[profilePath], "api.puretokensx.com", "other.invalid", 1)
				case "missing-active-profile":
					files["configLibrary/_meta.json"] = `{"appliedId":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`
				}
				for name, body := range files {
					desktopFixtureFile(t, root, name, body)
				}
				token, err := credentialFromClaudeDesktopRoot(root)
				assertSwitchWriterCredential(t, token, err, variant == "default-first" || variant == "reordered-models" || variant == "legacy-default")
				assertSwitchFixtureUnchanged(t, root, files)
			})
		}
	}
}

func TestSwitchGrokBuildWriterCompatibility(t *testing.T) {
	for _, platform := range []string{"macos", "windows"} {
		for _, variant := range []string{"exact-label", "legacy-label", "foreign-default", "missing-default"} {
			t.Run(platform+"/"+variant, func(t *testing.T) {
				files := switchDesktopWriterFiles(t, "grok-build", platform, "configured")
				body := files["home/.grok/config.toml"]
				if !strings.Contains(body, `name = "grok-3"`) || !strings.Contains(body, `default = "pts-grok-3"`) {
					t.Fatal("writer exact-label fixture changed")
				}
				if variant == "legacy-label" {
					body = strings.Replace(body, `name = "grok-3"`, `name = "Pure Tokens - grok-3"`, 1)
				}
				body += "\n[model.other]\nbase_url = \"https://other.invalid/v1\"\napi_key = \"fixture-other\"\n"
				if variant == "foreign-default" {
					body = strings.Replace(body, `default = "pts-grok-3"`, `default = "other"`, 1)
				}
				if variant == "missing-default" {
					body = strings.Replace(body, `default = "pts-grok-3"`, `default = "absent"`, 1)
				}
				root := t.TempDir()
				file := clientFixtureFile(t, root, "config.toml", body)
				token, err := credentialFromGrokBuildFile(file)
				assertSwitchWriterCredential(t, token, err, variant == "exact-label" || variant == "legacy-label")
				assertSwitchFixtureUnchanged(t, root, map[string]string{"config.toml": body})
			})
		}
	}
}
