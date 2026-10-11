package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallationInspectReportsBoundedRelativeDifferences(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "puretokens-image")
	os.Mkdir(dir, 0700)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("original"), 0600)
	os.WriteFile(filepath.Join(dir, "missing.md"), []byte("original"), 0600)
	writeInventoryFixture(t, dir, "puretokens-image")
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("private content never printed"), 0600)
	os.Remove(filepath.Join(dir, "missing.md"))
	os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("private finder bytes"), 0600)
	os.Mkdir(filepath.Join(root, ".puretokens-skill-stage.interrupted"), 0700)
	var output bytes.Buffer
	if err := runInstallationGuard([]string{"install-inspect", "--directory", root}, &output); err != nil {
		t.Fatal(err)
	}
	var result installationInspection
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, finding := range result.Findings {
		seen[finding.Category+":"+finding.Path] = true
	}
	for _, want := range []string{"modified:puretokens-image/SKILL.md", "missing:puretokens-image/missing.md", "additional:puretokens-image/.DS_Store", "transaction:.puretokens-skill-stage.interrupted"} {
		if !seen[want] {
			t.Errorf("missing finding %s: %s", want, output.String())
		}
	}
	if bytes.Contains(output.Bytes(), []byte(root)) || bytes.Contains(output.Bytes(), []byte("private")) {
		t.Fatal("private data disclosed")
	}
}

func TestInstallationRepairOnlyAllowsMissingManagedFiles(t *testing.T) {
	for _, change := range []string{"missing", "modified", "additional", "marker", "symlink"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("original"), 0600)
			writeInventoryFixture(t, dir, "puretokens-image")
			switch change {
			case "missing":
				os.Remove(filepath.Join(dir, "SKILL.md"))
			case "modified":
				os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("custom"), 0600)
			case "additional":
				os.WriteFile(filepath.Join(dir, "mine.txt"), []byte("custom"), 0600)
			case "marker":
				os.Remove(filepath.Join(dir, inventoryFile))
			case "symlink":
				if os.Symlink("SKILL.md", filepath.Join(dir, "link")) != nil {
					t.Skip("symlinks unavailable")
				}
			}
			var output bytes.Buffer
			err := runInstallationGuard([]string{"install-repair-verify", "--directory", dir, "--name", "puretokens-image"}, &output)
			if (err == nil) != (change == "missing") {
				t.Fatalf("%s: %v", change, err)
			}
		})
	}
}

func TestInstallationTransactionVerifiesAllCopiesBeforeRecovery(t *testing.T) {
	for _, change := range []string{"none", "backup", "target", "unknown", "duplicate-plan", "symlink", "unplanned-backup"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			stage := filepath.Join(root, ".puretokens-skill-stage.test")
			os.MkdirAll(filepath.Join(stage, "backup"), 0700)
			os.WriteFile(filepath.Join(stage, "transaction-v1"), nil, 0600)
			os.WriteFile(filepath.Join(stage, "plan"), []byte("replace puretokens-image\n"), 0600)
			for _, base := range []string{root, filepath.Join(stage, "backup")} {
				dir := filepath.Join(base, "puretokens-image")
				os.Mkdir(dir, 0700)
				os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("original"), 0600)
				writeInventoryFixture(t, dir, "puretokens-image")
			}
			switch change {
			case "backup":
				os.WriteFile(filepath.Join(stage, "backup/puretokens-image/SKILL.md"), []byte("keep"), 0600)
			case "target":
				os.WriteFile(filepath.Join(root, "puretokens-image/my.txt"), []byte("keep"), 0600)
			case "unknown":
				os.WriteFile(filepath.Join(stage, "my.txt"), []byte("keep"), 0600)
			case "duplicate-plan":
				os.WriteFile(filepath.Join(stage, "plan"), []byte("replace puretokens-image\nreplace puretokens-image\n"), 0600)
			case "symlink":
				os.Remove(filepath.Join(stage, "plan"))
				if os.Symlink(filepath.Join(root, "puretokens-image/SKILL.md"), filepath.Join(stage, "plan")) != nil {
					t.Skip("symlinks unavailable")
				}
			case "unplanned-backup":
				dir := filepath.Join(stage, "backup/puretokens-video")
				os.Mkdir(dir, 0700)
				os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("keep"), 0600)
				writeInventoryFixture(t, dir, "puretokens-video")
			}
			var output bytes.Buffer
			err := runInstallationGuard([]string{"install-transaction-verify", "--directory", stage, "--target-directory", root}, &output)
			if (err == nil) != (change == "none") {
				t.Fatalf("%s: %v", change, err)
			}
		})
	}
}

func writeInventoryFixture(t *testing.T, directory, name string) {
	t.Helper()
	var out bytes.Buffer
	if err := runInstallationGuard([]string{"install-inventory", "--directory", directory, "--name", name}, &out); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, inventoryFile), out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestInterruptedRepairAcceptsOnlyMissingOriginalTargets(t *testing.T) {
	for _, state := range []string{"before-move", "after-rollback", "changed-original", "missing-new-target", "ordinary-update"} {
		t.Run(state, func(t *testing.T) {
			root := t.TempDir()
			name := "puretokens-image"
			stage := filepath.Join(root, ".puretokens-skill-stage.repair")
			os.MkdirAll(filepath.Join(stage, "backup"), 0700)
			os.WriteFile(filepath.Join(stage, "transaction-v1"), nil, 0600)
			if state != "ordinary-update" {
				os.WriteFile(filepath.Join(stage, "repair-v1"), nil, 0600)
			}
			os.WriteFile(filepath.Join(stage, "plan"), []byte("replace "+name+"\n"), 0600)
			for _, base := range []string{root, stage} {
				dir := filepath.Join(base, name)
				os.Mkdir(dir, 0700)
				os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("original"), 0600)
				os.WriteFile(filepath.Join(dir, "guide.md"), []byte("guide"), 0600)
				writeInventoryFixture(t, dir, name)
			}
			os.Remove(filepath.Join(root, name, "guide.md"))
			switch state {
			case "after-rollback":
				os.RemoveAll(filepath.Join(stage, name))
			case "changed-original":
				os.WriteFile(filepath.Join(root, name, "SKILL.md"), []byte("keep my changes"), 0600)
			case "missing-new-target":
				os.Rename(filepath.Join(stage, name), filepath.Join(stage, "backup", name))
			}
			err := verifyInstallationTransaction(stage, root, "")
			wantValid := state == "before-move" || state == "after-rollback"
			if (err == nil) != wantValid {
				t.Fatalf("%s: %v", state, err)
			}
		})
	}
}

func TestInstallationInventoryAcceptsPowerShellUTF8BOM(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("original"), 0600)
	writeInventoryFixture(t, dir, "puretokens-image")
	marker := filepath.Join(dir, inventoryFile)
	body, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(marker, append([]byte{0xef, 0xbb, 0xbf}, body...), 0600)
	for _, command := range []string{"install-verify", "install-repair-verify"} {
		var output bytes.Buffer
		if err := runInstallationGuard([]string{command, "--directory", dir, "--name", "puretokens-image"}, &output); err != nil {
			t.Fatalf("%s: %v", command, err)
		}
	}
}

func TestInstallationTransactionAdoptsOnlyExactExplicitSource(t *testing.T) {
	root, source := t.TempDir(), t.TempDir()
	name := "puretokens-image"
	stage := filepath.Join(root, ".puretokens-skill-stage.legacy")
	backup := filepath.Join(stage, "backup", name)
	for _, dir := range []string{backup, filepath.Join(source, "skills", name)} {
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("official"), 0600)
	}
	os.WriteFile(filepath.Join(stage, "transaction-v1"), nil, 0600)
	os.WriteFile(filepath.Join(stage, "plan"), []byte("replace "+name+"\n"), 0600)
	if verifyInstallationTransaction(stage, root, "") == nil {
		t.Fatal("unmarked backup accepted without explicit source")
	}
	if err := verifyInstallationTransaction(stage, root, source); err != nil {
		t.Fatalf("verified exact legacy source was rejected: %v", err)
	}
	os.WriteFile(filepath.Join(backup, "SKILL.md"), []byte("keep my changes"), 0600)
	if verifyInstallationTransaction(stage, root, source) == nil {
		t.Fatal("changed legacy backup accepted")
	}
}

func TestInstallationGuardProtectsOwnedFilesAndUnmanagedNames(t *testing.T) {
	for _, change := range []string{"none", "extra", "modified", "deleted", "empty-directory", "marker", "symlink"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("original"), 0600)
			os.WriteFile(filepath.Join(dir, "skill.json"), []byte(`{"name":"puretokens-image"}`), 0600)
			var out bytes.Buffer
			args := []string{"install-verify", "--directory", dir, "--name", "puretokens-image"}
			if runInstallationGuard(args, &out) == nil {
				t.Fatal("name alone established ownership")
			}
			writeInventoryFixture(t, dir, "puretokens-image")
			switch change {
			case "extra":
				os.WriteFile(filepath.Join(dir, "personal.txt"), []byte("keep"), 0600)
			case "modified":
				os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("local edit"), 0600)
			case "deleted":
				os.Remove(filepath.Join(dir, "SKILL.md"))
			case "empty-directory":
				os.Mkdir(filepath.Join(dir, "my-folder"), 0700)
			case "marker":
				os.WriteFile(filepath.Join(dir, inventoryFile), []byte(`{}`), 0600)
			case "symlink":
				if os.Symlink(filepath.Join(dir, "SKILL.md"), filepath.Join(dir, "link")) != nil {
					t.Skip("symlinks unavailable")
				}
			}
			out.Reset()
			err := runInstallationGuard(args, &out)
			if (err == nil) != (change == "none") {
				t.Fatalf("change %s: %v", change, err)
			}
		})
	}
}

func TestInstallationGuardAdoptsOnlyExactCurrentSource(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	for _, dir := range []string{source, target} {
		os.Mkdir(filepath.Join(dir, "references"), 0700)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("official"), 0600)
		os.WriteFile(filepath.Join(dir, "skill.json"), []byte(`{"name":"puretokens-image"}`), 0600)
	}
	var out bytes.Buffer
	args := []string{"install-verify", "--directory", target, "--name", "puretokens-image", "--source-directory", source}
	if err := runInstallationGuard(args, &out); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(target, "personal.txt"), []byte("keep"), 0600)
	if runInstallationGuard(args, &out) == nil {
		t.Fatal("source adoption ignored extra user file")
	}
}

func TestInstallationGuardAcceptsPowerShellRuntimeFormatting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "runtime.json"), []byte(`{"schemaVersion":1,"name":"puretokens-api-executor","version":"0.17.1","platform":"windows-amd64"}`), 0600)
	writeInventoryFixture(t, dir, ".puretokens-executor")
	os.WriteFile(filepath.Join(dir, "runtime.json"), append([]byte{0xef, 0xbb, 0xbf}, []byte("{\r\n\"version\": \"0.17.1\", \"platform\":\"windows-amd64\", \"schemaVersion\":1,\"name\":\"puretokens-api-executor\"\r\n}")...), 0600)
	var out bytes.Buffer
	if err := runInstallationGuard([]string{"install-verify", "--directory", dir, "--name", ".puretokens-executor"}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestRetiredInstallationNamesAreNotManaged(t *testing.T) {
	for _, name := range []string{".puretokens-runtime", "puretokens_image", "puretokens_workbuddy_router"} {
		if validInstallationName(name) {
			t.Fatal("retired installation accepted")
		}
	}
}

func TestEvaluationInstallationInventoryProtectsNewSkill(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("official evaluation guide"), 0600)
	os.WriteFile(filepath.Join(dir, "skill.json"), []byte(`{"name":"puretokens-evaluate"}`), 0600)
	writeInventoryFixture(t, dir, "puretokens-evaluate")
	var out bytes.Buffer
	args := []string{"install-verify", "--directory", dir, "--name", "puretokens-evaluate"}
	if err := runInstallationGuard(args, &out); err != nil {
		t.Fatal("new Skill inventory cannot be verified")
	}
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("personal edit"), 0600)
	if runInstallationGuard(args, &out) == nil {
		t.Fatal("new Skill overwrites local edits")
	}
}
