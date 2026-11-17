package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const claudeManagedProfile = "70757265-746f-4000-8000-636c61756465"
const claudeSkillsConnectionFile = "puretokens-skills-connection.json"

// The companion belongs to this host's applied configuration, not to Switch's
// current login or another host. No request is sent to the local chat mapper.
func credentialFromClaudeMappedConnection(root, id string, connectionID *string) (string, error) {
	paths := []string{
		filepath.Join(root, "claude_desktop_config.json"),
		filepath.Join(root, "configLibrary", "_meta.json"),
		filepath.Join(root, "configLibrary", id+".json"),
		filepath.Join(root, claudeSkillsConnectionFile),
	}
	snapshots := make([][]byte, len(paths))
	defer func() {
		for _, data := range snapshots {
			clear(data)
		}
	}()
	documents := make([]map[string]any, len(paths))
	for i, path := range paths {
		if i == 3 {
			if _, err := os.Lstat(path); err == nil && !privateConnectionFile(path) {
				return "", clientFormatFailure()
			}
		}
		data, err := readClientBytes(path)
		if i == 3 && os.IsNotExist(err) {
			return "", &credentialResolutionError{
				status:     "active_connection_endpoint_unsupported",
				message:    "The mapped connection has only a local chat token and no dedicated Skills connection record. This is not an installation failure. No API request was sent.",
				nextAction: "Stop this automatic flow and preserve the original task. Apply this client's configuration in a Switch version that supports mapped Skills connections, then explicitly continue on this same host. Do not borrow another host's credential or reinstall to repair this record; never automatically resubmit.",
			}
		}
		if err != nil {
			return "", err
		}
		snapshots[i] = data
		value, err := parseClientJSON(data, false)
		if err != nil || jsonObject(value) == nil {
			return "", clientFormatFailure()
		}
		documents[i] = jsonObject(value)
	}
	config, meta, profile, record := documents[0], documents[1], documents[2], documents[3]
	if config["deploymentMode"] != "3p" || meta["appliedId"] != id ||
		profile["inferenceProvider"] != "gateway" || profile["inferenceCredentialKind"] != "static" ||
		profile["inferenceGatewayAuthScheme"] != "bearer" || profile["inferenceGatewayBaseUrl"] != "http://127.0.0.1:15722/claude-desktop" ||
		!strings.HasPrefix(jsonString(profile["inferenceGatewayApiKey"]), "pts-local-") {
		return "", clientSelectionFailure()
	}
	version := record["version"]
	if record["host"] != "claude-desktop" || record["state"] != "active" {
		return "", clientFormatFailure()
	}
	if !desktopProfileID.MatchString(jsonString(record["connection_id"])) {
		return "", clientFormatFailure()
	}
	switch version {
	case json.Number("1"):
		if len(record) != 8 || id != claudeManagedProfile || record["profile_id"] != id {
			return "", clientFormatFailure()
		}
		digest := sha256.Sum256(snapshots[2])
		if record["profile_sha256"] != hex.EncodeToString(digest[:]) {
			return "", credentialFailure("active_connection_selection_unconfirmed", "The legacy byte-bound Claude Desktop record no longer matches its profile file. This does not prove the Key or service is invalid. Apply this client's configuration with a compatible Switch version to update the connection automatically; do not edit a fingerprint or reinstall Skills. No API request was sent.", "")
		}
	case json.Number("2"):
		if len(record) != 7 || jsonString(record["local_gateway_token"]) == "" {
			return "", clientFormatFailure()
		}
		if record["local_gateway_token"] != profile["inferenceGatewayApiKey"] {
			return "", credentialFailure("active_connection_selection_unconfirmed", "The saved Skills credential does not belong to the current Claude Desktop connection. Apply this client's configuration in Switch to update the connection automatically; do not edit configuration files or reinstall Skills. No API request was sent.", "")
		}
	default:
		return "", clientFormatFailure()
	}
	key := jsonString(record["api_key"])
	if strings.HasPrefix(strings.TrimSpace(key), "pts-local-") || strings.ContainsAny(key, "\r\n") {
		return "", clientFormatFailure()
	}
	token, err := matchingCredential(jsonString(record["api_base_url"]), key, "", "/", "/v1", "/v1/")
	if err != nil {
		return "", err
	}
	// Fail if a concurrent apply changed any part of the effective connection.
	for i, path := range paths {
		if i == 3 && !privateConnectionFile(path) {
			return "", clientFormatFailure()
		}
		after, err := readClientBytes(path)
		same := err == nil && bytes.Equal(snapshots[i], after)
		if err == nil && !same && version == json.Number("2") {
			value, parseErr := parseClientJSON(after, false)
			same = parseErr == nil && sameClaudeConnectionFields(i, documents[i], jsonObject(value))
		}
		clear(after)
		if !same {
			return "", clientSelectionFailure()
		}
	}
	if connectionID != nil {
		*connectionID = jsonString(record["connection_id"])
	}
	return token, nil
}

// A concurrent model/preference update must not invalidate an unchanged
// connection. Compare only routing/auth fields; the credential record is exact.
func sameClaudeConnectionFields(index int, before, after map[string]any) bool {
	if before == nil || after == nil {
		return false
	}
	fields := [][]string{
		{"deploymentMode"},
		{"appliedId"},
		{"inferenceProvider", "inferenceCredentialKind", "inferenceGatewayAuthScheme", "inferenceGatewayBaseUrl", "inferenceGatewayApiKey"},
	}
	if index == 3 {
		return reflect.DeepEqual(before, after)
	}
	for _, field := range fields[index] {
		if !reflect.DeepEqual(before[field], after[field]) {
			return false
		}
	}
	return true
}

func credentialForConnection(host string) (string, string, error) {
	if host != "claude-desktop" {
		token, err := credentialForHost(host)
		return token, "", err
	}
	root, err := currentDesktopRoot("Claude-3p")
	if err != nil {
		return "", "", err
	}
	var binding string
	token, err := credentialFromClaudeDesktopConnection(root, &binding)
	return token, binding, err
}
