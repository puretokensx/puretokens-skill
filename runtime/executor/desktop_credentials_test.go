package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mappedSkillsFixture(t *testing.T) (string, map[string]any) {
	t.Helper()
	root := claudeDesktopFixture(t)
	profile := `{"inferenceProvider":"gateway","inferenceCredentialKind":"static","inferenceGatewayAuthScheme":"bearer","inferenceGatewayBaseUrl":"http://127.0.0.1:15722/claude-desktop","inferenceGatewayApiKey":"pts-local-12345678-1234-4000-8000-123456789abc"}`
	desktopFixtureFile(t, root, "configLibrary/"+fixtureDesktopProfile+".json", profile)
	digest := sha256.Sum256([]byte(profile))
	record := map[string]any{"version": 1, "host": "claude-desktop", "state": "active", "profile_id": fixtureDesktopProfile, "profile_sha256": hex.EncodeToString(digest[:]), "api_base_url": "https://api.puretokensx.com", "api_key": "synthetic-mapped-skills-key", "connection_id": "11111111-1111-4111-8111-111111111111"}
	writeMappedSkillsFixture(t, root, record)
	return root, record
}

func writeMappedSkillsFixture(t *testing.T, root string, record map[string]any) {
	t.Helper()
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	desktopFixtureFile(t, root, "puretokens-skills-connection.json", string(data))
	privateMappedFixtureFile(t, filepath.Join(root, claudeSkillsConnectionFile))
}

func TestClaudeDesktopMappedSkillsUsesOnlyBoundConnection(t *testing.T) {
	root, _ := mappedSkillsFixture(t)
	token, err := credentialFromClaudeDesktopRoot(root)
	if err != nil || token != "synthetic-mapped-skills-key" {
		t.Fatal("mapped profile did not resolve its Switch-managed dedicated connection")
	}
}

func TestClaudeDesktopSwitchWriterFixture(t *testing.T) {
	root := os.Getenv("PTP_MAPPED_FIXTURE_INPUT")
	if root == "" {
		t.Skip("explicit synthetic Switch writer export required")
	}
	for _, platform := range []string{"macos", "windows"} {
		var connectionID string
		token, err := credentialFromClaudeDesktopConnection(filepath.Join(root, platform), &connectionID)
		if err != nil || token != "pts-cross-repo-fixture" || !desktopProfileID.MatchString(connectionID) {
			t.Fatalf("Switch %s writer output rejected by Skills reader", platform)
		}
	}
}

func TestClaudeDesktopMappedSkillsRejectsInvalidBindings(t *testing.T) {
	for _, tc := range []struct {
		field string
		value any
	}{
		{"version", 2}, {"host", "codex"}, {"state", "inactive"},
		{"profile_id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		{"profile_sha256", strings.Repeat("0", 64)},
		{"api_base_url", "https://third-party.invalid"},
		{"api_key", ""}, {"api_key", "pts-local-12345678-1234-4000-8000-123456789abc"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			root, record := mappedSkillsFixture(t)
			record[tc.field] = tc.value
			writeMappedSkillsFixture(t, root, record)
			if token, err := credentialFromClaudeDesktopRoot(root); err == nil || token != "" {
				t.Fatal("invalid mapped connection accepted")
			}
		})
	}
	root, _ := mappedSkillsFixture(t)
	desktopFixtureFile(t, root, "configLibrary/_meta.json", `{"appliedId":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`)
	if token, err := credentialFromClaudeDesktopRoot(root); err == nil || token != "" {
		t.Fatal("inactive mapped profile accepted")
	}
}

func TestClaudeDesktopDirectIgnoresMappedSkillsRecord(t *testing.T) {
	root := claudeDesktopFixture(t)
	desktopFixtureFile(t, root, "puretokens-skills-connection.json", "not JSON")
	if token, err := credentialFromClaudeDesktopRoot(root); err != nil || token != "synthetic-desktop-key" {
		t.Fatal("companion must not override direct credentials")
	}
}

func TestClaudeDesktopMappedSkillsRejectsChangedFilesAndUnsafeStorage(t *testing.T) {
	for _, mutation := range []string{"profile", "duplicate", "symlink", "permissions"} {
		t.Run(mutation, func(t *testing.T) {
			root, _ := mappedSkillsFixture(t)
			path := filepath.Join(root, claudeSkillsConnectionFile)
			switch mutation {
			case "profile":
				p := filepath.Join(root, "configLibrary", fixtureDesktopProfile+".json")
				data, _ := os.ReadFile(p)
				if err := os.WriteFile(p, append(data, ' '), 0600); err != nil {
					t.Fatal(err)
				}
			case "duplicate":
				data, _ := os.ReadFile(path)
				data = append([]byte(`{"api_key":"unselected-key",`), data[1:]...)
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+".other"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".other", path); err != nil {
					t.Skip("symlinks unavailable")
				}
			case "permissions":
				if runtime.GOOS == "windows" {
					t.Skip("Unix file modes")
				}
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if token, err := credentialFromClaudeDesktopRoot(root); err == nil || token != "" {
				t.Fatal("unsafe connection accepted")
			}
		})
	}
}

const fixtureDesktopProfile = "70757265-746f-4000-8000-636c61756465"

func desktopFixtureFile(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

// Field names and shapes mirror Switch's independent desktop adapters.
func claudeDesktopFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	desktopFixtureFile(t, root, "claude_desktop_config.json", `{"deploymentMode":"3p"}`)
	desktopFixtureFile(t, root, "configLibrary/_meta.json", `{"appliedId":"`+fixtureDesktopProfile+`","entries":[{"id":"unrelated","name":"Pure Tokens"}]}`)
	desktopFixtureFile(t, root, "configLibrary/"+fixtureDesktopProfile+".json", `{"inferenceProvider":"gateway","inferenceCredentialKind":"static","inferenceGatewayAuthScheme":"bearer","inferenceGatewayBaseUrl":"https://api.puretokensx.com","inferenceGatewayApiKey":"synthetic-desktop-key"}`)
	return root
}

func TestClaudeDesktopActiveProfile(t *testing.T) {
	root := claudeDesktopFixture(t)
	token, err := credentialFromClaudeDesktopRoot(root)
	if err != nil || token != "synthetic-desktop-key" {
		t.Fatal("Switch Desktop profile did not resolve")
	}
	// A manually named profile resolves by active UUID, not the Switch label.
	other := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	data, _ := os.ReadFile(filepath.Join(root, "configLibrary", fixtureDesktopProfile+".json"))
	desktopFixtureFile(t, root, "configLibrary/"+other+".json", string(data))
	desktopFixtureFile(t, root, "configLibrary/_meta.json", `{"appliedId":"`+other+`"}`)
	if token, err := credentialFromClaudeDesktopRoot(root); err != nil || token != "synthetic-desktop-key" {
		t.Fatal("active UUID selection incorrectly depends on a label")
	}
}

func TestClaudeDesktopMappingExplainsCredentialBoundary(t *testing.T) {
	root := claudeDesktopFixture(t)
	desktopFixtureFile(t, root, "configLibrary/"+fixtureDesktopProfile+".json", `{"inferenceProvider":"gateway","inferenceCredentialKind":"static","inferenceGatewayAuthScheme":"bearer","inferenceGatewayBaseUrl":"http://127.0.0.1:15722/claude-desktop","inferenceGatewayApiKey":"pts-local-12345678-1234-4000-8000-123456789abc"}`)
	token, err := credentialFromClaudeDesktopRoot(root)
	if err == nil || token != "" {
		t.Fatal("mapped chat token must not authorize fixed public APIs")
	}
	result := initCredentialFailure(err)
	if result.ConfigurationStatus != "active_connection_endpoint_unsupported" || result.APIRequestExecuted || result.CredentialVerified {
		t.Fatal("mapping must stop locally with a supported diagnostic")
	}
	if !strings.Contains(result.Message, "local chat token") || !strings.Contains(result.Message, "not an installation failure") {
		t.Fatal("mapping diagnostic must explain why reinstalling cannot fix authorization")
	}
	for _, secret := range []string{root, "127.0.0.1", "pts-local-", "synthetic"} {
		if strings.Contains(result.Message+result.NextAction, secret) {
			t.Fatal("mapping diagnostic exposed connection data")
		}
	}
	if !strings.Contains(result.NextAction, "never automatically resubmit") {
		t.Fatal("mapping must preserve the existing no-retry guidance")
	}
}

func TestClaudeDesktopRejectsInactiveUnsafeAndUnsupportedProfiles(t *testing.T) {
	for _, tc := range []struct{ file, body string }{
		{"claude_desktop_config.json", `{"deploymentMode":"consumer"}`},
		{"configLibrary/_meta.json", `{"appliedId":"../../outside"}`},
		{"configLibrary/_meta.json", `{"appliedId":""}`},
		{"configLibrary/" + fixtureDesktopProfile + ".json", `{"inferenceProvider":"bedrock"}`},
		{"configLibrary/" + fixtureDesktopProfile + ".json", `{"inferenceProvider":"gateway","inferenceCredentialKind":"static","inferenceGatewayAuthScheme":"bearer","inferenceGatewayBaseUrl":"https://other.invalid","inferenceGatewayApiKey":"synthetic-private-key"}`},
		{"configLibrary/" + fixtureDesktopProfile + ".json", `{"inferenceProvider":"gateway","inferenceCredentialKind":"static","inferenceGatewayAuthScheme":"bearer","inferenceGatewayBaseUrl":"https://api.puretokensx.com"}`},
	} {
		t.Run(tc.file+tc.body[:10], func(t *testing.T) {
			root := claudeDesktopFixture(t)
			desktopFixtureFile(t, root, tc.file, tc.body)
			token, err := credentialFromClaudeDesktopRoot(root)
			if err == nil || token != "" {
				t.Fatal("unsupported Desktop selection accepted")
			}
			_, message, next := credentialFailureDetails(err)
			if strings.Contains(message+next, "synthetic") || strings.Contains(message+next, root) {
				t.Fatal("desktop failure disclosed private state")
			}
		})
	}
}

const dshFixtureSettings = `# Switch-format configuration; labels are not identity.
llm-pi-ai:
  providers:
    chosen-route:
      displayName: "Arbitrary name"
      apiKeyEnv: CHOSEN_REF
      api: openai-completions
      baseURL: "https://api.puretokensx.com/v1"
      models:
        - id: "gpt-5.6"
          name: "gpt-5.6"
    other:
      baseURL: "https://other.invalid/v1"
      apiKeyEnv: OTHER_REF
agent-default-model:
  provider: chosen-route
  model: "gpt-5.6"
`

func dshDesktopFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	desktopFixtureFile(t, root, "settings.yaml", dshFixtureSettings)
	desktopFixtureFile(t, root, ".credentials.yaml", "version: 1\nrefs:\n  CHOSEN_REF: \"synthetic-dsh-key\"\n  OTHER_REF: other-key\n")
	return root
}

func TestDSHDesktopSelectedProviderAndReference(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		root := dshDesktopFixture(t)
		desktopFixtureFile(t, root, "settings.yaml", strings.ReplaceAll(dshFixtureSettings, "\n", newline))
		t.Setenv("CHOSEN_REF", "wrong-environment-key")
		token, err := credentialFromDSHDesktopRoot(root)
		if err != nil || token != "synthetic-dsh-key" {
			t.Fatal("DSH selected versioned reference did not resolve")
		}
		t.Setenv("DSH_HOME", root)
		if token, err := credentialForHost("dsh-desktop"); err != nil || token != "synthetic-dsh-key" {
			t.Fatal("explicit Harness directory did not resolve")
		}
	}
}

func TestDSHDesktopRejectsAmbiguityAndNeverFallsBack(t *testing.T) {
	for _, tc := range []struct{ file, body string }{
		{"settings.yaml", strings.Replace(dshFixtureSettings, "provider: chosen-route", "provider: other", 1)},
		{"settings.yaml", strings.Replace(dshFixtureSettings, `model: "gpt-5.6"`, `model: "missing"`, 1)},
		{"settings.yaml", strings.Replace(dshFixtureSettings, "api: openai-completions", "api: unknown", 1)},
		{"settings.yaml", dshFixtureSettings + "\nagent-default-model: {provider: other}\n"},
		{"settings.yaml", dshFixtureSettings + "\n---\na: b\n"},
		{"settings.yaml", "a: &route {baseURL: value}\nb: *route\n"},
		{".credentials.yaml", "version: 2\nrefs: {CHOSEN_REF: secret}\n"},
		{".credentials.yaml", "version: 1\nrefs: {OTHER_REF: secret}\n"},
		{".credentials.yaml", "version: 1\nrefs: {CHOSEN_REF: first, CHOSEN_REF: second}\n"},
		{".credentials.yaml", "version: 1\nrefs: {CHOSEN_REF: 1234}\n"},
	} {
		root := dshDesktopFixture(t)
		desktopFixtureFile(t, root, tc.file, tc.body)
		t.Setenv("CHOSEN_REF", "synthetic-environment-fallback")
		if token, err := credentialFromDSHDesktopRoot(root); token != "" || err == nil {
			t.Fatalf("unsupported %s accepted", tc.file)
		}
	}
	// Endpoint verification must precede any attempt to open the secret file.
	root := dshDesktopFixture(t)
	desktopFixtureFile(t, root, "settings.yaml", strings.ReplaceAll(dshFixtureSettings, "api.puretokensx.com", "other.invalid"))
	if err := os.Remove(filepath.Join(root, ".credentials.yaml")); err != nil {
		t.Fatal(err)
	}
	_, err := credentialFromDSHDesktopRoot(root)
	status, _, _ := credentialFailureDetails(err)
	if status != "active_connection_not_puretokens" {
		t.Fatal("credential store read before verifying selected endpoint")
	}
	t.Setenv("DSH_HOME", "relative")
	if _, err := credentialFromDSHDesktop(); err == nil {
		t.Fatal("relative Harness override accepted")
	}
}

func TestDSHDesktopLegacyFlatReference(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		root := dshDesktopFixture(t)
		desktopFixtureFile(t, root, ".credentials.yaml", strings.ReplaceAll("CHOSEN_REF: synthetic-dsh-key\nOTHER_REF: synthetic-other-key\n", "\n", newline))
		t.Setenv("CHOSEN_REF", "synthetic-wrong-environment-key")
		token, err := credentialFromDSHDesktopRoot(root)
		if err != nil || token != "synthetic-dsh-key" {
			t.Fatal("Switch legacy flat reference did not resolve the selected connection")
		}
	}
}

func TestDSHDesktopRejectsMalformedFlatReferences(t *testing.T) {
	for _, body := range []string{
		"CHOSEN_REF: synthetic-key\nrefs: legacy\n",
		"CHOSEN_REF: synthetic-key\nrecords: legacy\n",
		"CHOSEN_REF: synthetic-key\nversion: 2\n",
		"CHOSEN_REF: synthetic-key\nOTHER_REF: {}\n",
		"CHOSEN_REF: synthetic-key\nOTHER_REF: ''\n",
		"CHOSEN_REF: synthetic-key\ninvalid-name: synthetic-other\n",
		"CHOSEN_REF: synthetic-key\n2INVALID: synthetic-other\n",
		"CHOSEN_REF: first\nCHOSEN_REF: second\n",
		"OTHER_REF: synthetic-key\n",
	} {
		root := dshDesktopFixture(t)
		desktopFixtureFile(t, root, ".credentials.yaml", body)
		t.Setenv("CHOSEN_REF", "synthetic-environment-fallback")
		if token, err := credentialFromDSHDesktopRoot(root); err == nil || token != "" {
			t.Fatal("ambiguous legacy credential store accepted")
		}
	}
}

func TestDesktopDataRoots(t *testing.T) {
	home := t.TempDir()
	appData := filepath.Join(home, "Roaming")
	for _, tc := range []struct{ platform, data, want string }{
		{"darwin", "", filepath.Join(home, "Library", "Application Support", "Claude-3p")},
		{"windows", appData, filepath.Join(appData, "Claude-3p")},
	} {
		got, err := desktopDataRoot(tc.platform, home, tc.data, "Claude-3p")
		if err != nil || got != tc.want {
			t.Fatal("desktop platform root mismatch")
		}
	}
	for _, platform := range []string{"linux", "windows"} {
		if _, err := desktopDataRoot(platform, home, "", "Claude-3p"); err == nil {
			t.Fatal("unavailable desktop root guessed")
		}
	}
}
