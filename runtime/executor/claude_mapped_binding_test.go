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

func mappedSkillsV2Fixture(t *testing.T) (string, map[string]any) {
	t.Helper()
	root, record := mappedSkillsFixture(t)
	record["version"] = 2
	delete(record, "profile_id")
	delete(record, "profile_sha256")
	record["local_gateway_token"] = "pts-local-12345678-1234-4000-8000-123456789abc"
	writeMappedSkillsFixture(t, root, record)
	return root, record
}

func TestClaudeMappedV2IgnoresFormattingAndNonConnectionPreferences(t *testing.T) {
	for _, change := range []string{"none", "whitespace", "field_order", "preferences", "models", "custom_profile_id"} {
		t.Run(change, func(t *testing.T) {
			root, record := mappedSkillsV2Fixture(t)
			path := filepath.Join(root, "configLibrary", fixtureDesktopProfile+".json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var profile map[string]any
			if err := json.Unmarshal(data, &profile); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "whitespace":
				data = []byte(" \r\n" + strings.ReplaceAll(string(data), ",", ",\r\n  ") + "\r\n ")
			case "field_order":
				data, err = json.Marshal(profile)
			case "preferences":
				profile["inferenceModelPricingEnabled"] = false
				profile["displayPreferences"] = map[string]any{"name": "synthetic-label"}
				data, err = json.Marshal(profile)
			case "models":
				profile["inferenceModels"] = []any{map[string]any{"name": "claude-fixture"}}
				profile["modelDiscoveryEnabled"] = true
				data, err = json.Marshal(profile)
			case "custom_profile_id":
				id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
				path = filepath.Join(root, "configLibrary", id+".json")
				desktopFixtureFile(t, root, "configLibrary/_meta.json", `{"appliedId":"`+id+`","label":"custom"}`)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			var id string
			token, err := credentialFromClaudeDesktopConnection(root, &id)
			if err != nil || token != "synthetic-mapped-skills-key" || id != record["connection_id"] {
				t.Fatal("equivalent connection rejected or task binding changed")
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(data) {
				t.Fatal("reader changed the profile")
			}
		})
	}
}

func TestClaudeMappedV2RejectsChangedConnections(t *testing.T) {
	for _, field := range []string{"inferenceProvider", "inferenceCredentialKind", "inferenceGatewayAuthScheme", "inferenceGatewayBaseUrl", "inferenceGatewayApiKey"} {
		t.Run(field, func(t *testing.T) {
			root, _ := mappedSkillsV2Fixture(t)
			path := filepath.Join(root, "configLibrary", fixtureDesktopProfile+".json")
			data, _ := os.ReadFile(path)
			var profile map[string]any
			if err := json.Unmarshal(data, &profile); err != nil {
				t.Fatal(err)
			}
			profile[field] = "unsupported"
			if field == "inferenceGatewayApiKey" {
				profile[field] = "pts-local-87654321-1234-4000-8000-123456789abc"
			}
			data, _ = json.Marshal(profile)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if token, err := credentialFromClaudeDesktopRoot(root); err == nil || token != "" {
				t.Fatal("changed connection accepted")
			}
		})
	}
}

func TestClaudeMappedV2RejectsLegacyHashAndUnknownRecordFields(t *testing.T) {
	for _, change := range []string{"legacy_hash", "unknown_version", "extra_field", "missing_field", "duplicate_profile", "inactive_profile", "unsafe_permissions", "other_token", "other_host", "inactive_record", "third_party_origin", "local_service_key"} {
		t.Run(change, func(t *testing.T) {
			root, record := mappedSkillsV2Fixture(t)
			switch change {
			case "legacy_hash":
				data, _ := os.ReadFile(filepath.Join(root, "configLibrary", fixtureDesktopProfile+".json"))
				digest := sha256.Sum256(data)
				record["profile_sha256"] = hex.EncodeToString(digest[:])
			case "unknown_version":
				record["version"] = 3
			case "extra_field":
				record["extra"] = true
			case "missing_field":
				delete(record, "local_gateway_token")
			case "other_token":
				record["local_gateway_token"] = "pts-local-87654321-1234-4000-8000-123456789abc"
			case "other_host":
				record["host"] = "codex"
			case "inactive_record":
				record["state"] = "inactive"
			case "third_party_origin":
				record["api_base_url"] = "https://other.invalid"
			case "local_service_key":
				record["api_key"] = record["local_gateway_token"]
			case "duplicate_profile":
				path := filepath.Join(root, "configLibrary", fixtureDesktopProfile+".json")
				data, _ := os.ReadFile(path)
				data = append([]byte(`{"inferenceGatewayApiKey":"pts-local-unselected",`), data[1:]...)
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			case "inactive_profile":
				desktopFixtureFile(t, root, "configLibrary/_meta.json", `{"appliedId":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`)
			case "unsafe_permissions":
				if runtime.GOOS == "windows" {
					t.Skip("Unix file modes")
				}
			}
			writeMappedSkillsFixture(t, root, record)
			if change == "unsafe_permissions" {
				if err := os.Chmod(filepath.Join(root, claudeSkillsConnectionFile), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if token, err := credentialFromClaudeDesktopRoot(root); err == nil || token != "" {
				t.Fatal("invalid v2 record accepted")
			}
		})
	}
}

func TestClaudeMappedBindingFailuresExplainVersionWithoutDisclosingConfiguration(t *testing.T) {
	for _, version := range []int{1, 2} {
		root, record := mappedSkillsFixture(t)
		if version == 1 {
			record["profile_sha256"] = strings.Repeat("0", 64)
		} else {
			root, record = mappedSkillsV2Fixture(t)
			record["local_gateway_token"] = "pts-local-other-connection"
		}
		writeMappedSkillsFixture(t, root, record)
		_, err := credentialFromClaudeDesktopRoot(root)
		code, message, next := credentialFailureDetails(err)
		if code != "active_connection_selection_unconfirmed" {
			t.Fatal("binding failure changed diagnostic contract")
		}
		want := "current Claude Desktop connection"
		if version == 1 {
			want = "legacy byte-bound"
		}
		if !strings.Contains(message, want) || !strings.Contains(next, "never automatically resubmit") {
			t.Fatal("binding diagnostic lacks safe actionable context")
		}
		for _, private := range []string{root, "pts-local-", "synthetic-mapped", "127.0.0.1", strings.Repeat("0", 64)} {
			if strings.Contains(message+next, private) {
				t.Fatal("binding diagnostic exposed private data")
			}
		}
	}
}

func TestClaudeMappedConcurrentChecksOnlyRejectEffectiveConnectionChanges(t *testing.T) {
	cases := []struct {
		index                       int
		before, equivalent, changed string
	}{
		{0, `{"deploymentMode":"3p"}`, `{"deploymentMode":"3p","theme":"dark"}`, `{"deploymentMode":"consumer"}`},
		{1, `{"appliedId":"selected"}`, `{"appliedId":"selected","entries":[]}`, `{"appliedId":"other"}`},
		{2, `{"inferenceGatewayApiKey":"pts-local-fixture"}`, `{"inferenceGatewayApiKey":"pts-local-fixture","inferenceModels":[]}`, `{"inferenceGatewayApiKey":"pts-local-other"}`},
		{3, `{"api_key":"synthetic-key","connection_id":"same"}`, `{ "connection_id": "same", "api_key": "synthetic-key" }`, `{"api_key":"synthetic-other","connection_id":"same"}`},
	}
	for _, tc := range cases {
		before, _ := parseClientJSON([]byte(tc.before), false)
		equivalent, _ := parseClientJSON([]byte(tc.equivalent), false)
		changed, _ := parseClientJSON([]byte(tc.changed), false)
		if !sameClaudeConnectionFields(tc.index, jsonObject(before), jsonObject(equivalent)) {
			t.Fatal("unrelated concurrent preference update rejected")
		}
		if sameClaudeConnectionFields(tc.index, jsonObject(before), jsonObject(changed)) || sameClaudeConnectionFields(tc.index, jsonObject(before), nil) {
			t.Fatal("changed or malformed concurrent connection accepted")
		}
	}
}
