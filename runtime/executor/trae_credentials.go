package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// Resolve only the saved Switch-managed connection. A receipt supplies exact
// account-scoped cache keys; never enumerate other users, decrypt the host's
// login/recovery secrets, or infer the model selected in a chat session.
func credentialFromTrae() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", clientSelectionFailure()
	}
	path, err := traeDatabasePath(runtime.GOOS, home, os.Getenv("APPDATA"))
	if err != nil {
		return "", err
	}
	return credentialFromTraeDatabase(path)
}

func traeDatabasePath(goos, home, appData string) (string, error) {
	if !clientAbsoluteRoot(home) {
		return "", clientFormatFailure()
	}
	base := filepath.Join(home, "Library", "Application Support")
	switch goos {
	case "darwin":
	case "windows":
		if !clientAbsoluteRoot(appData) {
			return "", clientFormatFailure()
		}
		base = appData
	default:
		return "", credentialFailure("host_credential_adapter_unavailable", "TRAE requires a supported local desktop execution environment.", "")
	}
	selected := ""
	// Existence checks only. Multiple editions stop BEFORE reading any store.
	for _, edition := range []string{"Trae", "Trae CN", "TRAE SOLO"} {
		path := filepath.Join(base, edition, "User", "globalStorage", "state.vscdb")
		_, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", clientFormatFailure()
		}
		if selected != "" {
			return "", credentialFailure("active_connection_ambiguous", "Multiple local TRAE editions have saved state; the effective host cannot be confirmed. No API request was sent.", "")
		}
		selected = path
	}
	if selected == "" {
		return "", credentialFailure("active_connection_record_missing", "The declared TRAE connection record was not found; no API request was sent.", "")
	}
	return selected, nil
}

func credentialFromTraeDatabase(path string) (string, error) {
	before, err := checkOctopDatabasePath(path)
	if err != nil {
		return "", clientFormatFailure()
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	if !strings.HasPrefix(uri.Path, "/") {
		uri.Path = "/" + uri.Path
	}
	uri.RawQuery = url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)", "trusted_schema(0)", "busy_timeout(250)"}}.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return "", clientFormatFailure()
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", clientFormatFailure()
	}
	defer tx.Rollback()
	var kind, definition string
	if tx.QueryRowContext(ctx, "SELECT type, CASE WHEN length(CAST(sql AS BLOB)) <= 16384 THEN sql END FROM sqlite_schema WHERE name = 'ItemTable'").Scan(&kind, &definition) != nil ||
		kind != "table" || !strings.HasPrefix(strings.ToUpper(definition), "CREATE TABLE") {
		return "", clientFormatFailure()
	}
	read := func(key string) (map[string]any, error) {
		rows, err := tx.QueryContext(ctx, "SELECT CASE WHEN length(CAST(value AS BLOB)) <= ? THEN CAST(value AS BLOB) END FROM ItemTable WHERE key = ? LIMIT 2", maxConfigBytes, key)
		if err != nil {
			return nil, clientFormatFailure()
		}
		defer rows.Close()
		if !rows.Next() {
			if rows.Err() != nil {
				return nil, clientFormatFailure()
			}
			return nil, nil
		}
		var data []byte
		if rows.Scan(&data) != nil || len(data) == 0 {
			return nil, clientFormatFailure()
		}
		defer clear(data)
		if rows.Next() || rows.Err() != nil {
			return nil, clientFormatFailure()
		}
		value, err := parseClientJSON(data, false)
		if err != nil || jsonObject(value) == nil {
			return nil, clientFormatFailure()
		}
		return jsonObject(value), nil
	}
	status, err := read("puretokens.traework.status.v1")
	if err != nil {
		return "", err
	}
	if status == nil || status["schema"] != json.Number("1") || status["phase"] != "active" {
		return "", clientSelectionFailure()
	}
	keys := jsonArray(status["cacheKeys"])
	if len(keys) != 2 {
		return "", clientFormatFailure()
	}
	account, ok := strings.CutSuffix(jsonString(keys[0]), ":AI.agent.model.model_list_map")
	if !ok || len(account) == 0 || len(account) > 100 ||
		strings.IndexFunc(account, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_')
		}) >= 0 || jsonString(keys[1]) != account+"_AI.agent.model.model_list_map" {
		return "", clientFormatFailure()
	}
	models := jsonArray(status["models"])
	if len(models) == 0 || len(models) > 100 {
		return "", clientSelectionFailure()
	}
	// Only one saved connection may supply a request. Fingerprints are native
	// receipt integrity metadata, kept in memory and never included in output.
	fingerprint := ""
	expected := make(map[string]map[string]any)
	for _, value := range models {
		model := jsonObject(value)
		id, digest := jsonString(model["id"]), jsonString(model["credential"])
		decoded, err := hex.DecodeString(digest)
		if id == "" || len(id) > 200 || expected[id] != nil || model["status"] != true ||
			err != nil || len(decoded) != sha256.Size || digest != strings.ToLower(digest) {
			return "", clientFormatFailure()
		}
		if !matchesPureTokensEndpoint(jsonString(model["base_url"]), "/v1", "/v1/", "/v1/chat/completions") {
			return "", endpointRecognitionFailure(jsonString(model["base_url"]), "/v1", "/v1/", "/v1/chat/completions")
		}
		if fingerprint != "" && fingerprint != digest {
			return "", credentialFailure("active_connection_ambiguous", "The saved TRAE connection has multiple credentials; no group is selected implicitly and no API request was sent.", "")
		}
		fingerprint = digest
		expected[id] = model
	}
	var token string
	found := make(map[string]bool)
	for _, key := range keys {
		cache, err := read(jsonString(key))
		if err != nil {
			return "", err
		}
		seen := make(map[string]bool)
		var walk func(any) error
		walk = func(value any) error {
			switch value := value.(type) {
			case map[string]any:
				if rawID, exists := value["custom_model_id"]; exists {
					id := jsonString(rawID)
					if number, ok := rawID.(json.Number); ok {
						id = string(number)
					}
					observed, managed := expected[id]
					if !managed {
						return nil
					}
					if seen[id] {
						return clientFormatFailure()
					}
					seen[id] = true
					if err := verifyTraeModel(value, observed); err != nil {
						return err
					}
					current, err := matchingCredential(jsonString(value["base_url"]), jsonString(value["ak"]), "/v1", "/v1/", "/v1/chat/completions")
					if err != nil {
						return err
					}
					token = current
					found[id] = true
					return nil
				}
				for _, child := range value {
					if err := walk(child); err != nil {
						return err
					}
				}
			case []any:
				for _, child := range value {
					if err := walk(child); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if err := walk(cache); err != nil {
			return "", err
		}
	}
	if len(found) != len(expected) || token == "" {
		return "", clientSelectionFailure()
	}
	after, err := checkOctopDatabasePath(path)
	if err != nil || !os.SameFile(before, after) {
		return "", clientFormatFailure()
	}
	return token, nil
}

func verifyTraeModel(model, observed map[string]any) error {
	// Match native receipt metadata BEFORE retaining the matching API key.
	for _, key := range []string{"name", "provider", "display_name", "base_url", "status", "multimodal"} {
		if !reflect.DeepEqual(model[key], observed[key]) {
			return clientSelectionFailure()
		}
	}
	settings := make([]any, 0, 11)
	for _, key := range []string{"auth_type", "region", "custom_model_type", "max_tokens", "prompt_max_tokens", "thinking_enable", "temperature", "top_p", "top_k", "max_turn", "config_detail"} {
		settings = append(settings, model[key])
	}
	if !reflect.DeepEqual(settings, jsonArray(observed["settings"])) {
		return clientSelectionFailure()
	}
	// Switch computes JSON.stringify([ak ?? null, sk ?? null]). Valid bearer
	// tokens are printable ASCII; reject masked/dynamic/unsupported values.
	token := jsonString(model["ak"])
	if token == "" || len(token) > 8192 || strings.IndexFunc(token, func(r rune) bool {
		return r < 33 || r > 126 || strings.ContainsRune(`*"'\$`, r)
	}) >= 0 || (model["sk"] != nil && model["sk"] != "") {
		return credentialFailure("active_connection_credential_missing", "The saved TRAE connection has no usable inline API credential; no API request was sent.", "")
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if encoder.Encode([]any{token, model["sk"]}) != nil {
		return clientFormatFailure()
	}
	defer clear(encoded.Bytes())
	digest := sha256.Sum256(bytes.TrimSuffix(encoded.Bytes(), []byte("\n")))
	if hex.EncodeToString(digest[:]) != jsonString(observed["credential"]) {
		return clientSelectionFailure()
	}
	return nil
}
