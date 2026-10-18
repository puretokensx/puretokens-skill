package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func doctorFixtureInstallation(t *testing.T, root, version string) {
	t.Helper()
	for _, name := range doctorSkillNames {
		directory := filepath.Join(root, name)
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(`{"schemaVersion":1,"name":%q,"version":%q,"entry":"SKILL.md"}`, name, version)
		if err := os.WriteFile(filepath.Join(directory, "skill.json"), []byte(manifest), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("fixture skill instructions"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	directory := filepath.Join(root, ".puretokens-executor")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"schemaVersion":1,"name":"puretokens-api-executor","version":%q,"platform":%q}`, version, runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.WriteFile(filepath.Join(directory, "runtime.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	binaryName := "puretokens-api"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	if err := os.WriteFile(filepath.Join(directory, binaryName), []byte("fixture binary; must never execute"), 0600); err != nil {
		t.Fatal(err)
	}
}

func doctorNoEnvironment(string) string { return "" }

func TestDoctorFindsMissingRetrievalSkillInCurrentInstallation(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	os.RemoveAll(filepath.Join(root, "puretokens-embedding"))
	result := collectDoctorAt(root, "codex", home, doctorNoEnvironment)
	found := false
	for _, installation := range result.Local.Installations {
		for _, skill := range installation.Skills {
			if skill.Name == "puretokens-embedding" && skill.Status == "missing" {
				found = true
			}
		}
	}
	if !found || result.Local.Status != "attention_required" {
		t.Fatal("doctor ignored a missing newly managed Skill", result.Local)
	}
}

func TestOpenCodeDoctorFindsDeclaredCustomAndSharedRoots(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg config")
	custom := filepath.Join(home, "custom config")
	env := func(key string) string {
		return map[string]string{"XDG_CONFIG_HOME": xdg, "OPENCODE_CONFIG_DIR": custom}[key]
	}
	loaded := filepath.Join(custom, "skills")
	doctorFixtureInstallation(t, loaded, executorVersion)
	for _, root := range []string{filepath.Join(xdg, "opencode", "skills"), filepath.Join(home, ".agents", "skills"), filepath.Join(home, ".claude", "skills")} {
		doctorFixtureInstallation(t, root, "0.1.0")
	}
	result := collectDoctorAt(loaded, "opencode", home, env)
	if result.OK || len(result.Local.Installations) != 4 || len(result.Local.DuplicateSkills) != 10 {
		t.Fatalf("OpenCode custom/shared copies missing: %+v", result)
	}
	if !reflect.DeepEqual(result.Local.Installations[0].Locations, []string{"loaded_skills", "host_skills"}) {
		t.Fatal("selected root disagrees with installer")
	}
	// Do not scan an old default directory outside the effective XDG root.
	for _, location := range doctorHostLocations("opencode", home, env) {
		if location.path == filepath.Join(home, ".config", "opencode", "skills") {
			t.Fatal("doctor guessed an inactive default")
		}
	}
	if roots := doctorHostLocations("opencode", home, func(string) string { return "relative" }); len(roots) != 0 {
		t.Fatal("relative OpenCode directory accepted")
	}
}

func TestDesktopDoctorUsesOnlyDeclaredSkillRoots(t *testing.T) {
	home := t.TempDir()
	harness := filepath.Join(home, "Harness with spaces")
	shared := filepath.Join(home, "Shared agents")
	env := func(key string) string {
		return map[string]string{"DSH_HOME": harness, "DSH_AGENTS_HOME": shared}[key]
	}
	root := filepath.Join(harness, "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	doctorFixtureInstallation(t, filepath.Join(shared, "skills"), "0.1.0")
	result := collectDoctorAt(root, "dsh-desktop", home, env)
	if result.Host != "dsh-desktop" || len(result.Local.Installations) != 2 || len(result.Local.DuplicateSkills) != 10 {
		t.Fatal("DSH duplicate user/shared skills not reported")
	}
	claudeRoot := filepath.Join(home, ".claude", "skills")
	doctorFixtureInstallation(t, claudeRoot, executorVersion)
	claude := collectDoctorAt(claudeRoot, "claude-desktop", home, doctorNoEnvironment)
	if !claude.OK || len(claude.Local.Installations) != 1 || claude.Local.AttachmentDeliveryStatus != "unverified" {
		t.Fatal("Desktop local skill discovery or acceptance boundary incorrect")
	}
	if locations := doctorHostLocations("dsh-desktop", home, func(string) string { return "relative" }); len(locations) != 0 {
		t.Fatal("doctor accepted a relative Harness root")
	}
}

func TestDoctorLocalReportsOnlyVersionConsistency(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	result := collectDoctorAt(root, "codex", home, doctorNoEnvironment)
	if !result.OK || result.Local.Status != "consistent" || len(result.Local.Installations) != 1 {
		t.Fatalf("unexpected local inventory: %+v", result)
	}
	if !reflect.DeepEqual(result.Local.Installations[0].Locations, []string{"loaded_skills", "host_skills"}) {
		t.Fatal("loaded root was not deduplicated")
	}
	if result.Connection != nil || result.Local.ChecksumStatus != "unverified" || result.Local.AttachmentDeliveryStatus != "unverified" {
		t.Fatal("local inventory claimed unperformed acceptance or checksum checks")
	}
	for _, skill := range result.Local.Installations[0].Skills {
		if skill.Status != "installed" || skill.Version != executorVersion {
			t.Fatal("matching skill manifest missing from inventory")
		}
	}
	body, _ := json.Marshal(result)
	if strings.Contains(string(body), home) {
		t.Fatal("diagnostics exposed an absolute host directory")
	}
}

func TestDoctorGeminiDetectsSeparateManagedAliases(t *testing.T) {
	home := t.TempDir()
	shared := filepath.Join(home, ".agents", "skills")
	host := filepath.Join(home, ".gemini", "skills")
	doctorFixtureInstallation(t, shared, executorVersion)
	doctorFixtureInstallation(t, host, "0.1.0")
	result := collectDoctorAt(shared, "gemini-cli", home, doctorNoEnvironment)
	if result.OK || result.Local.Status != "attention_required" || len(result.Local.Installations) != 2 || len(result.Local.DuplicateSkills) != 10 {
		t.Fatalf("duplicate installations not reported: %+v", result)
	}
	for _, skill := range result.Local.Installations[1].Skills {
		if skill.Status != "version_mismatch" || skill.Version != "0.1.0" {
			t.Fatal("stale alias version not distinguished")
		}
	}
	// Codex does not inspect the Gemini directory merely because it exists.
	codex := collectDoctorAt(shared, "codex", home, doctorNoEnvironment)
	if !codex.OK || len(codex.Local.Installations) != 1 || len(codex.Local.DuplicateSkills) != 0 {
		t.Fatal("diagnostic scope crossed to another host")
	}
}

func TestDoctorGeminiSymlinkAliasIsOneInstallation(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(home, ".gemini", "skills")); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	result := collectDoctorAt(root, "gemini-cli", home, doctorNoEnvironment)
	if !result.OK || len(result.Local.Installations) != 1 || len(result.Local.DuplicateSkills) != 0 {
		t.Fatal("one physical installation was reported as duplicate aliases")
	}
}

func TestDoctorUnusedGeminiAliasDoesNotRequireSecondInstall(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	unrelated := filepath.Join(home, ".gemini", "skills", "unrelated-skill")
	if err := os.MkdirAll(unrelated, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unrelated, "skill.json"), []byte("not a Pure Tokens record"), 0600); err != nil {
		t.Fatal(err)
	}
	result := collectDoctorAt(root, "gemini-cli", home, doctorNoEnvironment)
	if !result.OK || len(result.Local.Installations) != 2 || result.Local.Installations[1].Status != "missing" {
		t.Fatal("unrelated alias contents were treated as an incomplete installation")
	}
}

func TestDoctorRejectsUnmanagedAndSymlinkedManifests(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	name := doctorSkillNames[0]
	file := filepath.Join(root, name, "skill.json")
	if err := os.WriteFile(file, []byte(`{"schemaVersion":1,"name":"not-an-official-skill","version":"sk-secret-fixture","entry":"SKILL.md"}`), 0600); err != nil {
		t.Fatal(err)
	}
	result := collectDoctorAt(root, "codex", home, doctorNoEnvironment)
	if result.OK || result.Local.Installations[0].Skills[0].Status != "unrecognized" {
		t.Fatal("unmanaged skill was treated as official")
	}
	body, _ := json.Marshal(result)
	if bytes.Contains(body, []byte("sk-secret-fixture")) {
		t.Fatal("untrusted version escaped into the receipt")
	}
	target := filepath.Join(t.TempDir(), "unrelated-record.json")
	if err := os.WriteFile(target, []byte(`{"never":"read this as a managed manifest"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, file); err != nil {
		t.Skipf("file symlinks unavailable: %v", err)
	}
	result = collectDoctorAt(root, "codex", home, doctorNoEnvironment)
	if result.Local.Installations[0].Skills[0].Status != "unrecognized" {
		t.Fatal("symlinked manifest was followed")
	}
}

func TestDoctorHostDirectoryOverridesAreBounded(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(home, "explicit-host-root")
	for _, test := range []struct {
		host string
		env  map[string]string
	}{
		{"claude-code", map[string]string{"CLAUDE_CONFIG_DIR": custom}},
		{"workbuddy", map[string]string{"WORKBUDDY_CONFIG_DIR": custom, "CODEBUDDY_CONFIG_DIR": filepath.Join(home, "unused")}},
		{"workbuddy", map[string]string{"CODEBUDDY_CONFIG_DIR": custom}},
	} {
		got := doctorHostLocations(test.host, home, func(name string) string {
			if name != "CLAUDE_CONFIG_DIR" && name != "WORKBUDDY_CONFIG_DIR" && name != "CODEBUDDY_CONFIG_DIR" {
				t.Fatalf("unexpected environment lookup: %s", name)
			}
			return test.env[name]
		})
		if len(got) != 1 || got[0].path != filepath.Join(custom, "skills") {
			t.Fatalf("documented override not used for %s: %+v", test.host, got)
		}
	}
	if locations := doctorHostLocations("claude-code", home, func(string) string { return "relative-root" }); len(locations) != 0 {
		t.Fatal("relative override should not cause workspace-relative record reads")
	}
}

func doctorIsolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestDoctorCombinesLocalChecksWithReadOnlyInit(t *testing.T) {
	home := doctorIsolatedHome(t)
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if request.Method != http.MethodGet || request.Header.Get("Authorization") != "Bearer synthetic-doctor-fixture" {
			t.Error("doctor attempted an unexpected request")
		}
		if request.URL.Path == "/v1" {
			io.WriteString(w, `{"status":"ok","name":"Pure Tokens API","base_url":"/v1"}`)
		} else {
			io.WriteString(w, `{"data":[]}`)
		}
	}))
	defer server.Close()
	svc := service{baseURL: server.URL, token: "synthetic-doctor-fixture", client: server.Client(), profilesRoot: root}
	var output bytes.Buffer
	if err := executeDoctor(&output, svc, "codex"); err != nil {
		t.Fatal(err)
	}
	var result doctorReceipt
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Connection == nil || !result.Connection.CredentialVerified || !reflect.DeepEqual(paths, []string{"/v1", "/v1/media/models"}) {
		t.Fatal("doctor did not combine both bounded read-only checks")
	}
	if result.Local.ChecksumStatus != "unverified" || result.Local.AttachmentDeliveryStatus != "unverified" {
		t.Fatal("API reachability was mistaken for host acceptance")
	}
}

func TestDoctorRetainsLocalInventoryWithoutCredentials(t *testing.T) {
	home := doctorIsolatedHome(t)
	root := filepath.Join(home, ".agents", "skills")
	doctorFixtureInstallation(t, root, executorVersion)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls++
		t.Error("credential failure must not issue an API request")
	}))
	defer server.Close()
	var output bytes.Buffer
	err := executeDoctorCredentialFailure(&output, service{baseURL: server.URL, client: server.Client(), profilesRoot: root}, "codex", errors.New("synthetic private credential failure"))
	var result doctorReceipt
	if json.Unmarshal(output.Bytes(), &result) != nil || err == nil || result.OK || result.Local.Status != "consistent" || len(result.Local.Installations[0].Skills) != 10 || result.Connection == nil || result.Connection.APIRequestExecuted || calls != 0 {
		t.Fatalf("credential failure discarded inventory or executed a request: %s", output.String())
	}
	if strings.Contains(output.String(), "synthetic private credential failure") {
		t.Fatal("raw credential error was exposed")
	}
}
