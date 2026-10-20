package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Synthetic Switch native receipt/cache, never a real TRAE account or key.
func traeFixtureModel(id, token string) (map[string]any, map[string]any) {
	model := map[string]any{
		"custom_model_id": id, "name": "custom_openai_compatible//fixture-chat",
		"provider": "custom_openai_compatible", "display_name": "Pure Tokens Fixture",
		"base_url": "https://api.puretokensx.com/v1", "status": true,
		"multimodal": false, "ak": token, "sk": nil,
	}
	raw, _ := json.Marshal([]any{token, nil})
	digest := sha256.Sum256(raw)
	observed := map[string]any{
		"id": id, "name": model["name"], "provider": model["provider"],
		"display_name": model["display_name"], "base_url": model["base_url"],
		"status": true, "multimodal": false, "credential": hex.EncodeToString(digest[:]),
		"settings": []any{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil},
	}
	return model, observed
}

func traeFixtureDatabase(t *testing.T, path string, mutate func(map[string]any, map[string]any)) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	model, observed := traeFixtureModel("managed-1", "fixture-trae-key")
	cache := map[string]any{"chat": []any{model}}
	status := map[string]any{
		"schema": 1, "phase": "active", "account": "fixture-switch-account",
		"receipt":   "12345678-1234-1234-1234-123456789abc",
		"cacheKeys": []any{"fixture-user:AI.agent.model.model_list_map", "fixture-user_AI.agent.model.model_list_map"},
		"models":    []any{observed},
	}
	if mutate != nil {
		mutate(status, cache)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("CREATE TABLE ItemTable (key TEXT UNIQUE, value BLOB)"); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]any{
		"puretokens.traework.status.v1":                 status,
		"fixture-user:AI.agent.model.model_list_map":    cache,
		"fixture-user_AI.agent.model.model_list_map":    cache,
		"other-user:AI.agent.model.model_list_map":      map[string]any{"private": "unrelated"},
		"secret://puretokens/traework-configuration-v1": "not-an-executor-credential-source",
	} {
		data, _ := json.Marshal(value)
		if _, err = db.Exec("INSERT INTO ItemTable(key,value) VALUES(?,?)", key, string(data)); err != nil {
			t.Fatal(err)
		}
	}
}

func traeFixtureHome(t *testing.T) (string, string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	base := filepath.Join(home, "Library", "Application Support")
	if runtime.GOOS == "windows" {
		base = filepath.Join(home, "AppData", "Roaming")
	}
	return home, base
}

func TestTraeSwitchConnection(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("TRAE local desktop adapter supports macOS and Windows")
	}
	for _, edition := range []string{"Trae", "Trae CN", "TRAE SOLO"} {
		t.Run(edition, func(t *testing.T) {
			_, base := traeFixtureHome(t)
			path := filepath.Join(base, edition, "User", "globalStorage", "state.vscdb")
			traeFixtureDatabase(t, path, nil)
			before, _ := os.ReadFile(path)
			token, err := credentialForHost("trae")
			if err != nil || token != "fixture-trae-key" {
				t.Fatalf("managed TRAE connection was not resolved: %v", err)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("credential resolution changed host database")
			}
		})
	}
}

func TestTraeUnsafeSavedConnectionsStop(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("TRAE local desktop adapter supports macOS and Windows")
	}
	cases := []struct {
		name   string
		mutate func(map[string]any, map[string]any)
	}{
		{"unfinished transaction", func(status, _ map[string]any) { status["phase"] = "applying" }},
		{"cross account cache", func(status, _ map[string]any) {
			status["cacheKeys"] = []any{"fixture-user:AI.agent.model.model_list_map", "other-user_AI.agent.model.model_list_map"}
		}},
		{"unrecognized endpoint", func(status, _ map[string]any) {
			status["models"].([]any)[0].(map[string]any)["base_url"] = "https://example.invalid/v1"
		}},
		{"edited credential", func(_, cache map[string]any) { cache["chat"].([]any)[0].(map[string]any)["ak"] = "changed-key" }},
		{"masked credential", func(_, cache map[string]any) { cache["chat"].([]any)[0].(map[string]any)["ak"] = "****" }},
		{"disabled model", func(_, cache map[string]any) { cache["chat"].([]any)[0].(map[string]any)["status"] = false }},
		{"missing model", func(_, cache map[string]any) { delete(cache, "chat") }},
		{"different group credentials", func(status, cache map[string]any) {
			model, observed := traeFixtureModel("managed-2", "other-group-key")
			status["models"] = append(status["models"].([]any), observed)
			cache["chat"] = append(cache["chat"].([]any), model)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, base := traeFixtureHome(t)
			traeFixtureDatabase(t, filepath.Join(base, "Trae", "User", "globalStorage", "state.vscdb"), tc.mutate)
			token, err := credentialForHost("trae")
			if err == nil || token != "" {
				t.Fatal("unsafe saved connection released a credential")
			}
			assertLocalConnectionDiagnostic(t, err)
		})
	}
}

func TestTraeMultipleEditionsDoNotGuess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("TRAE local desktop adapter supports macOS and Windows")
	}
	_, base := traeFixtureHome(t)
	for _, edition := range []string{"Trae", "TRAE SOLO"} {
		traeFixtureDatabase(t, filepath.Join(base, edition, "User", "globalStorage", "state.vscdb"), nil)
	}
	token, err := credentialForHost("trae")
	status, _, _ := credentialFailureDetails(err)
	if token != "" || status != "active_connection_ambiguous" {
		t.Fatal("coinstalled editions were silently selected")
	}
}

func TestTraeSavedModelSet(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
	}{
		{"one credential across models", "fixture-trae-key"},
		{"different group credentials", "different-group-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "state.vscdb")
			traeFixtureDatabase(t, path, func(status, cache map[string]any) {
				model, observed := traeFixtureModel("managed-2", tc.token)
				// The display name is opaque native metadata, not identity.
				model["display_name"], observed["display_name"] = "Unrelated label", "Unrelated label"
				status["models"] = append(status["models"].([]any), observed)
				cache["chat"] = append(cache["chat"].([]any), model)
			})
			token, err := credentialFromTraeDatabase(path)
			if tc.token == "fixture-trae-key" {
				if err != nil || token != "fixture-trae-key" {
					t.Fatal("multiple models in the same saved connection did not resolve")
				}
			} else if err == nil || token != "" {
				t.Fatal("different saved group credentials were silently selected")
			}
		})
	}
}

func TestTraeCommittedWALAndDisagreeingCaches(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "state.vscdb")
	traeFixtureDatabase(t, path, nil)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	// A newer committed edit in WAL must invalidate the old receipt, even if
	// the other cache and the main database file still contain the old value.
	changed := `{"chat":[{"custom_model_id":"managed-1","name":"custom_openai_compatible//fixture-chat","provider":"custom_openai_compatible","display_name":"Pure Tokens Fixture","base_url":"https://api.puretokensx.com/v1","status":false,"multimodal":false,"ak":"fixture-trae-key","sk":null}]}`
	if _, err := db.Exec("UPDATE ItemTable SET value=? WHERE key=?", changed, "fixture-user_AI.agent.model.model_list_map"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + "-wal"); err != nil {
		t.Fatal("fixture did not create WAL")
	}
	if token, err := credentialFromTraeDatabase(path); err == nil || token != "" {
		t.Fatal("adapter ignored the current committed cache or fell back to stale data")
	}
}

func TestTraeRejectsRedirectedDatabaseAndLeavesMissingStoreAbsent(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing.vscdb")
	if token, err := credentialFromTraeDatabase(missing); err == nil || token != "" {
		t.Fatal("missing database accepted")
	}
	if _, err := os.Lstat(missing); !os.IsNotExist(err) {
		t.Fatal("missing database was created")
	}
	path := filepath.Join(root, "state.vscdb")
	traeFixtureDatabase(t, path, nil)
	redirect := filepath.Join(root, "redirect.vscdb")
	if err := os.Symlink(path, redirect); err != nil {
		t.Skip("symlinks unavailable on this host")
	}
	if token, err := credentialFromTraeDatabase(redirect); err == nil || token != "" {
		t.Fatal("redirected database accepted")
	}
}

func TestTraeDuplicateStorageKeysDoNotSelectFirstRecord(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "state.vscdb")
	traeFixtureDatabase(t, path, nil)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`ALTER TABLE ItemTable RENAME TO PreviousItems;
		CREATE TABLE ItemTable (key TEXT, value BLOB);
		INSERT INTO ItemTable SELECT key,value FROM PreviousItems;
		INSERT INTO ItemTable SELECT key,value FROM PreviousItems WHERE key='puretokens.traework.status.v1'`); err != nil {
		t.Fatal(err)
	}
	if token, err := credentialFromTraeDatabase(path); err == nil || token != "" {
		t.Fatal("ambiguous duplicate storage records selected the first row")
	}
}

func TestTraeSwitchRuntimeWriterContract(t *testing.T) {
	data, err := os.ReadFile("testdata/trae-switch-skill-connection.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := parseClientJSON(data, false)
	if err != nil {
		t.Fatal(err)
	}
	fixture := jsonObject(value)
	status := jsonObject(fixture["receipt"])
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "state.vscdb")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE ItemTable (key TEXT PRIMARY KEY, value BLOB)"); err != nil {
		t.Fatal(err)
	}
	entries := map[string]any{jsonString(fixture["receiptKey"]): status}
	for _, key := range jsonArray(status["cacheKeys"]) {
		entries[jsonString(key)] = fixture["cache"]
	}
	for key, value := range entries {
		data, _ := json.Marshal(value)
		if _, err := db.Exec("INSERT INTO ItemTable(key,value) VALUES(?,?)", key, string(data)); err != nil {
			t.Fatal(err)
		}
	}
	token, err := credentialFromTraeDatabase(path)
	if err != nil || token != "pts-trae-fixture-not-real" {
		t.Fatalf("Switch native runtime's independent connection fixture did not resolve: %v", err)
	}
}

func TestTraeDualPlatformDefaultPaths(t *testing.T) {
	for _, goos := range []string{"darwin", "windows"} {
		for _, edition := range []string{"Trae", "Trae CN", "TRAE SOLO"} {
			t.Run(goos+"/"+edition, func(t *testing.T) {
				home := t.TempDir()
				appData := filepath.Join(home, "AppData", "Roaming")
				base := filepath.Join(home, "Library", "Application Support")
				if goos == "windows" {
					base = appData
				}
				expected := filepath.Join(base, edition, "User", "globalStorage", "state.vscdb")
				if err := os.MkdirAll(filepath.Dir(expected), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(expected, []byte("existence fixture only"), 0600); err != nil {
					t.Fatal(err)
				}
				got, err := traeDatabasePath(goos, home, appData)
				if err != nil || got != expected {
					t.Fatalf("wrong default %s edition path: %v", goos, err)
				}
			})
		}
	}
	if _, err := traeDatabasePath("windows", t.TempDir(), "relative"); err == nil {
		t.Fatal("relative Windows APPDATA was accepted")
	}
}
