package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

var doctorSkillNames = currentSkillNames

var doctorVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([+-][A-Za-z0-9.-]+)?$`)

type doctorComponent struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

type doctorInstallation struct {
	Locations []string          `json:"locations"`
	Loaded    bool              `json:"loaded"`
	Status    string            `json:"status"`
	Skills    []doctorComponent `json:"skills"`
	Executor  doctorComponent   `json:"executor"`
}

type doctorLocalResult struct {
	Status                   string               `json:"status"`
	Installations            []doctorInstallation `json:"installations"`
	DuplicateSkills          []string             `json:"duplicate_skills"`
	ChecksumStatus           string               `json:"checksum_status"`
	ChecksumReason           string               `json:"checksum_reason"`
	AttachmentDeliveryStatus string               `json:"attachment_delivery_status"`
}

type doctorReceipt struct {
	OK              bool              `json:"ok"`
	Command         string            `json:"command"`
	Host            string            `json:"host"`
	ExecutorVersion string            `json:"executor_version"`
	Local           doctorLocalResult `json:"local"`
	Connection      *initReceipt      `json:"connection,omitempty"`
	Message         string            `json:"message"`
	NextAction      string            `json:"next_action"`
}

type doctorLocation struct {
	label string
	path  string
}

// Only documented Skill directories for the selected host are considered.
// Configuration records, credentials and unrelated home-directory entries are
// never opened. The shared Agents directory is also a Gemini discovery alias.
func doctorHostLocations(host, home string, getenv func(string) string) []doctorLocation {
	if home == "" || !filepath.IsAbs(home) {
		return nil
	}
	location := ""
	switch host {
	case "claude-code", "claude-desktop":
		location = getenv("CLAUDE_CONFIG_DIR")
		if location == "" {
			location = filepath.Join(home, ".claude")
		}
	case "codex":
		return []doctorLocation{{"host_skills", filepath.Join(home, ".agents", "skills")}}
	case "vscode":
		return []doctorLocation{
			{"host_skills", filepath.Join(home, ".copilot", "skills")},
			{"shared_agents_skills", filepath.Join(home, ".agents", "skills")},
			{"shared_claude_skills", filepath.Join(home, ".claude", "skills")},
		}
	case "octop":
		// Only the already loaded explicit workspace installation is inspected.
		return nil
	case "hermes", "evox":
		var err error
		location, err = newClientRoot(host, runtime.GOOS, home, getenv)
		if err != nil {
			return nil
		}
	case "workbuddy":
		location = getenv("WORKBUDDY_CONFIG_DIR")
		if location == "" {
			location = getenv("CODEBUDDY_CONFIG_DIR")
		}
		if location == "" {
			location = filepath.Join(home, ".workbuddy")
		}
	case "gemini-cli":
		return []doctorLocation{
			{"host_skills", filepath.Join(home, ".gemini", "skills")},
			{"shared_agents_skills", filepath.Join(home, ".agents", "skills")},
		}
	case "grok-build":
		location = filepath.Join(home, ".grok")
	case "minimax-code":
		root, err := miniMaxCodeRoot(runtime.GOOS, home, getenv)
		if err != nil {
			return nil
		}
		return []doctorLocation{{"host_skills", filepath.Join(root, "skills")}}
	case "deepseek-harness":
		root, err := deepSeekHarnessRoot(home, getenv)
		if err != nil {
			return nil
		}
		shared := getenv("DSH_AGENTS_HOME")
		if shared == "" {
			shared = filepath.Join(home, ".agents")
		}
		result := []doctorLocation{{"host_skills", filepath.Join(root, "skills")}}
		if clientAbsoluteRoot(shared) {
			result = append(result, doctorLocation{"shared_agents_skills", filepath.Join(shared, "skills")})
		}
		return result
	case "dsh-desktop":
		location = getenv("DSH_HOME")
		if location == "" {
			root, err := desktopDataRoot(runtime.GOOS, home, getenv("APPDATA"), "dsh-desktop")
			if err != nil {
				return nil
			}
			location = filepath.Join(root, "harness")
		}
		if !filepath.IsAbs(location) {
			return nil
		}
		shared := getenv("DSH_AGENTS_HOME")
		if shared == "" {
			shared = filepath.Join(home, ".agents")
		}
		result := []doctorLocation{{"host_skills", filepath.Join(location, "skills")}}
		if filepath.IsAbs(shared) {
			result = append(result, doctorLocation{"shared_agents_skills", filepath.Join(shared, "skills")})
		}
		return result
	case "opencode":
		config, err := opencodeXDG(home, "XDG_CONFIG_HOME", ".config", getenv)
		if err != nil {
			return nil
		}
		location = config
		if explicit := getenv("OPENCODE_CONFIG_DIR"); explicit != "" {
			if !opencodeAbsolute(explicit) {
				return nil
			}
			location = explicit
		}
		return []doctorLocation{
			{"host_skills", filepath.Join(location, "skills")},
			{"global_skills", filepath.Join(config, "skills")},
			{"shared_agents_skills", filepath.Join(home, ".agents", "skills")},
			{"shared_claude_skills", filepath.Join(home, ".claude", "skills")},
		}
	case "kimi-code", "qoder", "pi":
		var err error
		location, err = kimiQoderRoot(host, home, getenv)
		if err != nil {
			return nil
		}
		if host == "kimi-code" {
			return []doctorLocation{{"host_skills", filepath.Join(location, "skills")}, {"shared_agents_skills", filepath.Join(home, ".agents", "skills")}}
		}
	case "zcode":
		var err error
		location, err = zcodeRoot(home, getenv("ZCODE_DATA_BASE_DIR"))
		if err != nil {
			return nil
		}
	case "trae":
		return []doctorLocation{
			{"host_skills", filepath.Join(home, ".trae", "skills")},
			{"domestic_host_skills", filepath.Join(home, ".trae-cn", "skills")},
		}
	default:
		return nil
	}
	if !filepath.IsAbs(location) {
		return nil
	}
	return []doctorLocation{{"host_skills", filepath.Join(location, "skills")}}
}

// collectDoctorLocal is usable even when root's credential resolution fails.
// Versions are compared with this running executor, never with an assumed
// latest release. Installed manifests contain no trusted binary checksum.
func collectDoctorLocal(svc service, host string) doctorReceipt {
	home, _ := os.UserHomeDir()
	return collectDoctorAt(svc.profilesRoot, host, home, os.Getenv)
}

func collectDoctorAt(loadedRoot, host, home string, getenv func(string) string) doctorReceipt {
	result := doctorReceipt{
		Command: "doctor", Host: host, ExecutorVersion: executorVersion,
		Local: doctorLocalResult{
			Status: "unavailable", Installations: []doctorInstallation{},
			DuplicateSkills: []string{}, ChecksumStatus: "unverified",
			ChecksumReason:           "Installed runtime metadata has no trusted artifact checksum. Version consistency is not checksum verification.",
			AttachmentDeliveryStatus: "unverified",
		},
		Message:    "Local installation checks do not verify host attachment delivery or authenticate the connection.",
		NextAction: "Review the local findings; run the connection checks and verify attachment handoff in the current host.",
	}
	switch host {
	case "claude-code", "codex", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae", "claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop":
	default:
		result.Host = "unsupported"
		result.NextAction = "Choose a supported current host before running diagnostics."
		return result
	}
	locations := []doctorLocation{}
	if filepath.IsAbs(loadedRoot) {
		locations = append(locations, doctorLocation{"loaded_skills", filepath.Clean(loadedRoot)})
	}
	locations = append(locations, doctorHostLocations(host, home, getenv)...)
	roots := []string{}
	counts := make(map[string]int)
	loadedOK := false
	attention := false
	for _, location := range locations {
		existing := -1
		for i, root := range roots {
			if sameDoctorRoot(root, location.path) {
				existing = i
				break
			}
		}
		if existing >= 0 {
			result.Local.Installations[existing].Locations = append(result.Local.Installations[existing].Locations, location.label)
			continue
		}
		installation := inspectDoctorInstallation(location)
		roots = append(roots, location.path)
		result.Local.Installations = append(result.Local.Installations, installation)
		if installation.Loaded {
			loadedOK = installation.Status == "consistent"
			attention = attention || !loadedOK
		}
		if installation.Status != "missing" && installation.Status != "consistent" {
			attention = true
		}
		for _, skill := range installation.Skills {
			if skill.Status == "installed" || skill.Status == "version_mismatch" {
				counts[skill.Name]++
			}
		}
	}
	for _, name := range doctorSkillNames {
		if counts[name] > 1 {
			result.Local.DuplicateSkills = append(result.Local.DuplicateSkills, name)
			attention = true
		}
	}
	if attention {
		result.Local.Status = "attention_required"
		result.NextAction = "Review missing, unrecognized, mismatched or duplicate managed installations. Use the official updater for the intended directory; do not delete unknown files."
	} else if loadedOK {
		result.Local.Status = "consistent"
		result.OK = true
	}
	return result
}

func sameDoctorRoot(first, second string) bool {
	if filepath.Clean(first) == filepath.Clean(second) {
		return true
	}
	a, aErr := os.Stat(first)
	b, bErr := os.Stat(second)
	return aErr == nil && bErr == nil && a.IsDir() && b.IsDir() && os.SameFile(a, b)
}

func inspectDoctorInstallation(location doctorLocation) doctorInstallation {
	result := doctorInstallation{
		Locations: []string{location.label}, Loaded: location.label == "loaded_skills",
		Status: "consistent", Skills: []doctorComponent{},
		Executor: doctorComponent{Name: "puretokens-api", Status: "missing"},
	}
	if info, err := os.Stat(location.path); err != nil || !info.IsDir() {
		result.Status = "missing"
		if err != nil && !os.IsNotExist(err) {
			result.Status = "unreadable"
		} else if err == nil {
			result.Status = "unrecognized"
		}
	}
	for _, name := range doctorSkillNames {
		skill := doctorComponent{Name: name, Status: "missing"}
		directory := filepath.Join(location.path, name)
		var manifest struct {
			SchemaVersion int    `json:"schemaVersion"`
			Name          string `json:"name"`
			Version       string `json:"version"`
			Entry         string `json:"entry"`
		}
		if info, err := os.Lstat(directory); err == nil {
			skill.Status = "unrecognized"
			if info.IsDir() && readDoctorManifest(filepath.Join(directory, "skill.json"), &manifest) == nil &&
				manifest.SchemaVersion == 1 && manifest.Name == name && manifest.Entry == "SKILL.md" &&
				doctorRegularFile(filepath.Join(directory, "SKILL.md")) && doctorValidVersion(manifest.Version) {
				skill.Version = manifest.Version
				skill.Status = doctorVersionStatus(manifest.Version)
			}
		} else if !os.IsNotExist(err) {
			skill.Status = "unreadable"
		}
		result.Skills = append(result.Skills, skill)
	}
	executorDirectory := filepath.Join(location.path, ".puretokens-executor")
	var manifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		Name          string `json:"name"`
		Version       string `json:"version"`
		Platform      string `json:"platform"`
	}
	if info, err := os.Lstat(executorDirectory); err == nil {
		result.Executor.Status = "unrecognized"
		binaryName := "puretokens-api"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		if info.IsDir() && readDoctorManifest(filepath.Join(executorDirectory, "runtime.json"), &manifest) == nil &&
			manifest.SchemaVersion == 1 && manifest.Name == "puretokens-api-executor" &&
			doctorValidVersion(manifest.Version) && doctorRegularFile(filepath.Join(executorDirectory, binaryName)) {
			result.Executor.Version = manifest.Version
			result.Executor.Status = doctorVersionStatus(manifest.Version)
			if manifest.Platform != runtime.GOOS+"-"+runtime.GOARCH {
				result.Executor.Status = "platform_mismatch"
			}
		}
	} else if !os.IsNotExist(err) {
		result.Executor.Status = "unreadable"
	}
	if result.Status == "consistent" {
		anyOfficialEntry := result.Executor.Status != "missing"
		for _, skill := range result.Skills {
			anyOfficialEntry = anyOfficialEntry || skill.Status != "missing"
			if skill.Status != "installed" {
				result.Status = "attention_required"
			}
		}
		if result.Executor.Status != "installed" {
			result.Status = "attention_required"
		}
		// A discovery alias may contain unrelated Skills. Its mere existence
		// does not make it an incomplete Pure Tokens installation.
		if !anyOfficialEntry {
			result.Status = "missing"
		}
	}
	return result
}

func doctorValidVersion(version string) bool {
	return len(version) <= 64 && doctorVersionPattern.MatchString(version)
}

func doctorVersionStatus(version string) string {
	if version == executorVersion {
		return "installed"
	}
	return "version_mismatch"
}

func doctorRegularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func readDoctorManifest(path string, destination any) error {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() > maxConfigBytes {
		return errors.New("managed manifest unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return errors.New("managed manifest changed")
	}
	body, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil || len(body) > maxConfigBytes {
		return errors.New("managed manifest exceeds supported size")
	}
	return json.Unmarshal(bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf}), destination)
}

func withDoctorConnection(result doctorReceipt, connection initReceipt) doctorReceipt {
	result.Connection = &connection
	result.OK = result.Local.Status == "consistent" && connection.OK
	if !connection.OK {
		result.Message = "Local diagnostics completed; the current connection was not verified. Host attachment delivery remains unverified."
		result.NextAction = connection.NextAction
	} else if result.Local.Status == "consistent" {
		result.Message = "Installed versions are consistent and the read-only connection checks passed. Binary checksum and host attachment delivery remain unverified."
		result.NextAction = "Use the current host to confirm native attachment handoff when media is next requested."
	} else {
		result.Message = "The read-only connection checks passed, but local installation findings need review. Host attachment delivery remains unverified."
	}
	return result
}

func executeDoctor(output io.Writer, svc service, host string) error {
	local := collectDoctorLocal(svc, host)
	if local.Host == "unsupported" {
		writeJSON(output, local)
		return errors.New("unsupported diagnostic host")
	}
	if svc.token == "" {
		return executeDoctorCredentialFailure(output, svc, host, errors.New("credential unavailable"))
	}
	var connectionOutput bytes.Buffer
	err := executeInit(&connectionOutput, svc)
	var connection initReceipt
	if decodeErr := json.Unmarshal(connectionOutput.Bytes(), &connection); decodeErr != nil {
		connection = initReceipt{Command: "init", ConfigurationStatus: "unverified", Message: "The connection check did not return a readable result.", NextAction: "Run doctor again in the current host."}
		err = decodeErr
	}
	result := withDoctorConnection(local, connection)
	writeJSON(output, result)
	if err != nil {
		return err
	}
	if !result.OK {
		return errors.New("local installation needs attention")
	}
	return nil
}

func executeDoctorCredentialFailure(output io.Writer, svc service, host string, credentialErr error) error {
	result := withDoctorConnection(collectDoctorLocal(svc, host), initCredentialFailure(credentialErr))
	writeJSON(output, result)
	if credentialErr == nil {
		return errors.New("credential unavailable")
	}
	return credentialErr
}
