package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const inventoryFile = ".puretokens-managed.json"
const inventoryFormat = "puretokens-managed-files-v1"

var installationNames = []string{"puretokens-balance", "puretokens-connection", "puretokens-models", "puretokens-image", "puretokens-video", "puretokens-audio", "puretokens-evaluate", "puretokens-update", ".puretokens-executor"}

type installationFinding struct {
	Category string `json:"category"`
	Path     string `json:"path"`
	Code     string `json:"code"`
}

type installationInspection struct {
	SchemaVersion int                   `json:"schema_version"`
	Status        string                `json:"status"`
	Findings      []installationFinding `json:"findings"`
	Truncated     bool                  `json:"truncated"`
}

type installationInventory struct {
	Format string            `json:"format"`
	Name   string            `json:"name"`
	Files  map[string]string `json:"files"`
}

// Purely local, read-only installer support. The checksum-verified candidate
// binary checks ownership and emits an inventory; only sync writes installed
// files/markers. These commands never resolve credentials or make requests.
func runInstallationGuard(args []string, output io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	directory := flags.String("directory", "", "")
	name := flags.String("name", "", "")
	source := flags.String("source-directory", "", "")
	target := flags.String("target-directory", "", "")
	fail := func() error {
		fmt.Fprintln(output, "Installation ownership could not be verified, or files were added/modified. Existing files were preserved; review the directory before updating.")
		return errors.New("installation ownership conflict")
	}
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 {
		return fail()
	}
	if args[0] == "install-inspect" {
		return inspectInstallation(*directory, output)
	}
	if args[0] == "install-transaction-verify" {
		if verifyInstallationTransaction(*directory, *target, *source) != nil {
			return fail()
		}
		return nil
	}
	if !validInstallationName(*name) {
		return fail()
	}
	actual, err := installationTree(*directory, *name)
	if err != nil {
		return fail()
	}
	if args[0] == "install-inventory" {
		return json.NewEncoder(output).Encode(actual)
	}
	if args[0] == "install-repair-verify" {
		if verifyMissingOnlyInstallation(*directory, actual) != nil {
			return fail()
		}
		return nil
	}
	if err := verifyInstallation(*directory, actual, *source); err != nil {
		return fail()
	}
	return nil
}

func readInstallationInventory(directory, name string) (installationInventory, error) {
	var recorded installationInventory
	marker := filepath.Join(directory, inventoryFile)
	info, err := os.Lstat(marker)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return recorded, errors.New("unsafe inventory")
	}
	body, err := readBoundedFile(marker, maxConfigBytes)
	if err != nil || json.Unmarshal(bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf}), &recorded) != nil || recorded.Format != inventoryFormat || recorded.Name != name || len(recorded.Files) == 0 || len(recorded.Files) > 2048 {
		return recorded, errors.New("invalid installation inventory")
	}
	for path, digest := range recorded.Files {
		clean := strings.TrimSuffix(path, "/")
		if clean == "" || filepath.IsAbs(clean) || strings.Contains(clean, "\\") || strings.ContainsAny(clean, "\r\n\x00") || filepath.ToSlash(filepath.Clean(clean)) != clean || clean == ".." || strings.HasPrefix(clean, "../") {
			return recorded, errors.New("invalid inventory path")
		}
		if digest != "directory" && (len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "") {
			return recorded, errors.New("invalid inventory hash")
		}
	}
	return recorded, nil
}

func verifyMissingOnlyInstallation(directory string, actual installationInventory) error {
	recorded, err := readInstallationInventory(directory, actual.Name)
	if err != nil {
		return err
	}
	for path, digest := range actual.Files {
		if recorded.Files[path] != digest {
			return errors.New("local additions or modifications")
		}
	}
	return nil
}

func safeInstallationFindingPath(path, fallback string) string {
	if len(path) > 220 {
		return fallback
	}
	for _, r := range path {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._-/", r)) {
			return fallback
		}
	}
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "../") {
		return fallback
	}
	return path
}

func inspectInstallation(root string, output io.Writer) error {
	report := installationInspection{SchemaVersion: 1, Status: "clean", Findings: []installationFinding{}}
	add := func(category, path, code string) {
		report.Status = "attention_required"
		if len(report.Findings) >= 128 {
			report.Truncated = true
			return
		}
		report.Findings = append(report.Findings, installationFinding{category, path, code})
	}
	info, err := os.Lstat(root)
	if !filepath.IsAbs(root) || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		add("unreadable", ".", "installation_root_unavailable")
		return json.NewEncoder(output).Encode(report)
	}
	// Only enumerate the selected root and inspect the nine managed directories.
	entries, err := os.ReadDir(root)
	if err != nil {
		add("unreadable", ".", "installation_root_unavailable")
	} else {
		for _, entry := range entries {
			name := entry.Name()
			if name == ".puretokens-install-lock" || strings.HasPrefix(name, ".puretokens-skill-stage.") || strings.HasPrefix(name, ".puretokens-skill-stage-") {
				add("transaction", safeInstallationFindingPath(name, "."), "transaction_requires_review")
			}
		}
	}
	for _, name := range installationNames {
		directory := filepath.Join(root, name)
		if _, err := os.Lstat(directory); os.IsNotExist(err) {
			add("missing", name, "managed_directory_missing")
			continue
		}
		actual, err := installationTree(directory, name)
		if err != nil {
			add("unreadable", name, "managed_directory_unreadable_or_unsafe")
			continue
		}
		recorded, err := readInstallationInventory(directory, name)
		if err != nil {
			add("unreadable", name+"/"+inventoryFile, "managed_inventory_unavailable")
			continue
		}
		paths := make([]string, 0, len(recorded.Files)+len(actual.Files))
		for path := range recorded.Files {
			paths = append(paths, path)
		}
		for path := range actual.Files {
			if _, ok := recorded.Files[path]; !ok {
				paths = append(paths, path)
			}
		}
		sort.Strings(paths)
		for _, path := range paths {
			found, present := actual.Files[path]
			expected, managed := recorded.Files[path]
			relative := safeInstallationFindingPath(name+"/"+path, name)
			if !present {
				add("missing", relative, "managed_file_missing")
			} else if !managed {
				add("additional", relative, "unmanaged_entry_present")
			} else if found != expected {
				add("modified", relative, "managed_file_modified")
			}
		}
	}
	return json.NewEncoder(output).Encode(report)
}

// A recovery stage is untrusted input. Verify all entries and both sides of
// every replacement before a shell/PowerShell recovery may mutate anything.
func verifyInstallationTransaction(stage, target, source string) error {
	if !filepath.IsAbs(stage) || !filepath.IsAbs(target) || filepath.Dir(stage) != target {
		return errors.New("invalid transaction root")
	}
	base := filepath.Base(stage)
	if !strings.HasPrefix(base, ".puretokens-skill-stage.") && !strings.HasPrefix(base, ".puretokens-skill-stage-") {
		return errors.New("invalid stage name")
	}
	info, err := os.Lstat(stage)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe stage")
	}
	entries, err := os.ReadDir(stage)
	if err != nil {
		return err
	}
	marker := false
	plans := 0
	repairStage := false
	// Legacy exact source copies may be adopted by sync. The same explicit
	// source proof must remain available when validating their rollback copies.
	sourceFor := func(name string) string {
		if source == "" {
			return ""
		}
		return filepath.Join(source, "skills", name)
	}
	if info, err := os.Lstat(filepath.Join(stage, "repair-v1")); err == nil && info.Mode().IsRegular() && info.Size() == 0 {
		repairStage = true
	}
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(stage, name)
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("unsafe stage entry")
		}
		switch name {
		case "transaction-v1", "repair-v1", "committed", "plan", "plan.json":
			if !entry.Type().IsRegular() {
				return errors.New("unsafe transaction metadata")
			}
			body, err := readBoundedFile(path, maxConfigBytes)
			if err != nil {
				return err
			}
			if name == "transaction-v1" {
				marker = true
			}
			if name == "plan" || name == "plan.json" {
				plans++
			}
			if (name == "transaction-v1" || name == "repair-v1" || name == "committed") && len(body) != 0 {
				return errors.New("changed marker")
			}
		case "backup":
			if !entry.IsDir() {
				return errors.New("unsafe backup")
			}
			backups, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			for _, backup := range backups {
				if !validInstallationName(backup.Name()) || backup.Type()&os.ModeSymlink != 0 {
					return errors.New("unknown backup")
				}
				actual, err := installationTree(filepath.Join(path, backup.Name()), backup.Name())
				verified := verifyInstallation(filepath.Join(path, backup.Name()), actual, sourceFor(backup.Name()))
				if verified != nil && repairStage {
					verified = verifyMissingOnlyInstallation(filepath.Join(path, backup.Name()), actual)
				}
				if err != nil || verified != nil {
					return errors.New("changed backup")
				}
			}
		default:
			if !validInstallationName(name) {
				return errors.New("unknown stage entry")
			}
			actual, err := installationTree(path, name)
			if err != nil || verifyInstallation(path, actual, "") != nil {
				return errors.New("changed stage content")
			}
		}
	}
	if !marker || plans != 1 {
		return errors.New("missing transaction marker")
	}
	type planEntry struct {
		Action string `json:"action"`
		Name   string `json:"name"`
	}
	var plan []planEntry
	if body, err := readBoundedFile(filepath.Join(stage, "plan"), maxConfigBytes); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				return errors.New("invalid plan")
			}
			plan = append(plan, planEntry{fields[0], fields[1]})
		}
	} else if body, err := readBoundedFile(filepath.Join(stage, "plan.json"), maxConfigBytes); err == nil {
		if json.Unmarshal(bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf}), &plan) != nil {
			return errors.New("invalid plan")
		}
	} else {
		return errors.New("missing transaction plan")
	}
	seen := map[string]bool{}
	for _, entry := range plan {
		if !validInstallationName(entry.Name) || seen[entry.Name] || (entry.Action != "create" && entry.Action != "replace") {
			return errors.New("unknown recovery entry")
		}
		seen[entry.Name] = true
		path := filepath.Join(target, entry.Name)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			continue
		}
		actual, err := installationTree(path, entry.Name)
		verified := verifyInstallation(path, actual, sourceFor(entry.Name))
		if verified != nil && repairStage && entry.Action == "replace" {
			// Before replacement, or after a completed rollback move, the old
			// directory can legitimately still lack managed files. A present
			// backup means this is the newly installed target and must be exact.
			if _, backupErr := os.Lstat(filepath.Join(stage, "backup", entry.Name)); os.IsNotExist(backupErr) {
				verified = verifyMissingOnlyInstallation(path, actual)
			}
		}
		if err != nil || verified != nil {
			return errors.New("changed recovery target")
		}
	}
	if len(plan) == 0 {
		return errors.New("empty transaction plan")
	}
	for _, entry := range entries {
		if validInstallationName(entry.Name()) && !seen[entry.Name()] {
			return errors.New("unplanned stage entry")
		}
	}
	if backups, err := os.ReadDir(filepath.Join(stage, "backup")); err == nil {
		for _, backup := range backups {
			if !seen[backup.Name()] {
				return errors.New("unplanned backup")
			}
			for _, entry := range plan {
				if entry.Name == backup.Name() && entry.Action != "replace" {
					return errors.New("unexpected backup action")
				}
			}
		}
	}
	return nil
}

func validInstallationName(name string) bool {
	switch name {
	case ".puretokens-executor", "puretokens-balance", "puretokens-connection", "puretokens-image", "puretokens-video", "puretokens-models", "puretokens-audio", "puretokens-evaluate", "puretokens-update":
		return true
	}
	return false
}

func installationTree(directory, name string) (installationInventory, error) {
	result := installationInventory{Format: inventoryFormat, Name: name, Files: map[string]string{}}
	info, err := os.Lstat(directory)
	if !filepath.IsAbs(directory) || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return result, errors.New("invalid installation directory")
	}
	var total int64
	err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == directory {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if len(result.Files) >= 2048 || strings.ContainsAny(relative, "\r\n") || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("unsupported installation entry")
		}
		if entry.IsDir() {
			result.Files[relative+"/"] = "directory"
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("unsupported installation file")
		}
		if relative == inventoryFile {
			return nil
		}
		total += info.Size()
		if info.Size() > 64<<20 || total > 128<<20 {
			return errors.New("installation inventory too large")
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		opened, err := file.Stat()
		if err != nil || !os.SameFile(info, opened) {
			return errors.New("installation changed during inspection")
		}
		hash := sha256.New()
		if relative == "runtime.json" {
			data, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
			var object map[string]any
			if err != nil || len(data) > maxConfigBytes || json.Unmarshal(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), &object) != nil || object == nil {
				return errors.New("invalid runtime manifest")
			}
			encoded, _ := json.Marshal(object)
			hash.Write(encoded)
		} else {
			n, err := io.Copy(hash, io.LimitReader(file, info.Size()+1))
			if err != nil || n != info.Size() {
				return errors.New("installation changed during inspection")
			}
		}
		result.Files[relative] = fmt.Sprintf("%x", hash.Sum(nil))
		return nil
	})
	return result, err
}

func verifyInstallation(directory string, actual installationInventory, source string) error {
	marker := filepath.Join(directory, inventoryFile)
	if _, err := os.Lstat(marker); err == nil {
		body, err := readBoundedFile(marker, maxConfigBytes)
		var recorded installationInventory
		if err != nil || json.Unmarshal(body, &recorded) != nil || recorded.Format != inventoryFormat || recorded.Name != actual.Name || !reflect.DeepEqual(actual.Files, recorded.Files) {
			return errors.New("managed files changed")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	// Exact current source copies can be adopted without relying on names.
	if source != "" {
		expected, err := installationTree(source, actual.Name)
		if err == nil && reflect.DeepEqual(expected.Files, actual.Files) {
			return nil
		}
	}
	return errors.New("installation provenance unknown")
}
