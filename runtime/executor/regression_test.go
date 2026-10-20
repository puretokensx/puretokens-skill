package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixturePNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func fixtureService(server *httptest.Server) service {
	root, _ := filepath.Abs("../../skills")
	return service{baseURL: server.URL, client: server.Client(), profilesRoot: root, token: "synthetic-fixture-token", downloadProofs: make(map[string]downloadProof)}
}
func profileService() service {
	root, _ := filepath.Abs("../../skills")
	return service{profilesRoot: root}
}
func writeFixture(t *testing.T, name, contents string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func decodeReceipt(t *testing.T, b *bytes.Buffer) receipt {
	t.Helper()
	var r receipt
	if err := json.Unmarshal(b.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCodexAcceptsRealTOMLShapes(t *testing.T) {
	p := writeFixture(t, "config.toml", `model_provider = 'puretokens_switch'
model_context_window = 200000
[features]
shell = true
[model_providers.puretokens_switch]
base_url = "https://api.puretokensx.com/v1"
wire_api = "responses"
requires_openai_auth = false
experimental_bearer_token = "fixture"
http_headers = { "X-Test" = "ignored" }
`)
	token, err := credentialFromCodexFile(p)
	if err != nil || token != "fixture" {
		t.Fatal("valid Switch configuration rejected", err)
	}
}
func TestCodexUsesSelectedProfileAndExplicitAuthFile(t *testing.T) {
	p := writeFixture(t, "config.toml", `model_provider = "other"
profile = "selected"
[profiles.selected]
model_provider = "arbitrary-label"
[model_providers.arbitrary-label]
base_url = "https://api.puretokensx.com/v1"
requires_openai_auth = true
`)
	os.WriteFile(filepath.Join(filepath.Dir(p), "auth.json"), []byte(`{"OPENAI_API_KEY":"fixture"}`), 0600)
	token, err := credentialFromCodexFile(p)
	if err != nil || token != "fixture" {
		t.Fatal("explicit auth source failed", err)
	}
	t.Setenv("CODEX_HOME", filepath.Dir(p))
	token, err = credentialFromCodex()
	if err != nil || token != "fixture" {
		t.Fatal("CODEX_HOME ignored", err)
	}
}
func TestCodexOnlyUsesDeclaredEnvironmentCredential(t *testing.T) {
	t.Setenv("PT_TEST_SELECTED_KEY", "fixture")
	p := writeFixture(t, "config.toml", `model_provider = "selected"
[model_providers.selected]
base_url = "https://api.puretokensx.com/v1"
env_key = "PT_TEST_SELECTED_KEY"
`)
	token, err := credentialFromCodexFile(p)
	if err != nil || token != "fixture" {
		t.Fatal(err)
	}
}
func TestSubmissionReturnsIDWithoutStatusRead(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" {
			t.Error("submission did extra read")
		}
		io.WriteString(w, `{"id":"paid-task","status":"pending"}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	err := executeTask(strings.NewReader(`{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"test"}`), &out, fixtureService(server))
	r := decodeReceipt(t, &out)
	if err != nil || calls != 1 || r.TaskID != "paid-task" || r.SubmissionOutcome != "accepted" || r.RequestedCount != 1 {
		t.Fatalf("incomplete immediate receipt: %+v %v", r, err)
	}
}
func TestCompleteStdinDoesNotWaitForEOF(t *testing.T) {
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	defer reader.Close()
	defer writer.Close()
	go func() { _, err := decodeTaskRequest(reader); done <- err }()
	io.WriteString(writer, `{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"test"}`)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("complete JSON blocked on EOF")
	}
}
func TestProfileAllowsPublicImageAndValidatesCount(t *testing.T) {
	r := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2", Prompt: "reference", Parameters: map[string]any{"image": []string{"https://example.invalid/current.png"}}}
	if err := prepareProfileRequest(&r, profileService()); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := taskRequestBody(r); err != nil {
		t.Fatal(err)
	}
	r = taskRequest{Kind: "image", Operation: "generate", Model: "fixture-multi-image", Prompt: "test", Parameters: map[string]any{"n": float64(3)}, RequestedCount: 1}
	if prepareProfileRequest(&r, syntheticMultiImageService(t, profileService())) == nil {
		t.Fatal("mismatched count accepted")
	}
	r.RequestedCount = 0
	if err := prepareProfileRequest(&r, syntheticMultiImageService(t, profileService())); err != nil || r.RequestedCount != 3 {
		t.Fatal("count not derived from n", err)
	}
}
func TestAllInstalledProfilesAcceptCoreParameters(t *testing.T) {
	for _, kind := range []string{"image", "video"} {
		files, _ := filepath.Glob("../../skills/puretokens-" + kind + "/references/profiles/*.json")
		for _, file := range files {
			var p modelProfile
			b, _ := os.ReadFile(file)
			json.Unmarshal(b, &p)
			t.Run(p.ID, func(t *testing.T) {
				r := taskRequest{Kind: kind, Operation: "generate", Model: p.ID, Prompt: "test prompt"}
				parameters := requiredProfileFixtureParameters(t, p)
				if len(parameters) > 0 && prepareProfileRequest(&r, profileService()) == nil {
					t.Fatal("missing required parameters accepted")
				}
				r.Parameters = parameters
				completeProfileFixtureContext(&r, p)
				if err := prepareProfileRequest(&r, profileService()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestInstalledJSONReferenceProfiles(t *testing.T) {
	for _, kind := range []string{"image", "video"} {
		files, _ := filepath.Glob("../../skills/puretokens-" + kind + "/references/profiles/*.json")
		for _, file := range files {
			var profile modelProfile
			data, err := os.ReadFile(file)
			if err != nil || json.Unmarshal(data, &profile) != nil {
				t.Fatal("unreadable installed profile")
			}
			if profile.Parameters.Properties["image"]["type"] != "json" {
				continue
			}
			t.Run(profile.ID, func(t *testing.T) {
				for _, reference := range []any{
					"https://example.invalid/current.png",
					[]any{"https://example.invalid/current.png"},
					map[string]any{"url": "https://example.invalid/current.png"},
				} {
					request := taskRequest{Kind: kind, Operation: "generate", Model: profile.ID, Prompt: "reference", Parameters: map[string]any{"image": reference}}
					if err := prepareProfileRequest(&request, profileService()); err != nil {
						t.Fatal("declared JSON reference rejected", err)
					}
					_, _, body, err := taskRequestBody(request)
					if err != nil {
						t.Fatal(err)
					}
					var payload map[string]any
					if json.NewDecoder(body).Decode(&payload) != nil {
						t.Fatal("invalid request body")
					}
					want, _ := json.Marshal(reference)
					got, _ := json.Marshal(payload["image"])
					if !bytes.Equal(want, got) {
						t.Fatal("reference representation changed")
					}
				}
				for _, reference := range []any{
					nil, true, map[string]any{"url": "http://example.invalid/current.png"},
					map[string]any{"url": "https://127.0.0.1/current.png"},
					map[string]any{"url": "https://example.invalid/current.png", "extra": "not declared"},
					[]any{"https://example.invalid/a.png", "https://example.invalid/b.png"},
				} {
					request := taskRequest{Kind: kind, Operation: "generate", Model: profile.ID, Prompt: "reference", Parameters: map[string]any{"image": reference}}
					if prepareProfileRequest(&request, profileService()) == nil {
						t.Fatal("unsupported JSON reference accepted")
					}
				}
			})
		}
	}
}
func TestProfileRejectsUnsupportedValuesBeforePOST(t *testing.T) {
	for _, parameters := range []map[string]any{{"n": 2}, {"image_size": "200cmx230cm"}, {"width": 500}, {"unknown": true}} {
		r := taskRequest{Kind: "image", Operation: "generate", Model: "gpt-image-2", Prompt: "test", Parameters: parameters}
		if prepareProfileRequest(&r, profileService()) == nil {
			t.Fatal("invalid parameters accepted")
		}
	}
	r := taskRequest{Kind: "image", Operation: "generate", Model: "fixture-multi-image", Prompt: "test", Parameters: map[string]any{"width": 1024}}
	if prepareProfileRequest(&r, syntheticMultiImageService(t, profileService())) == nil {
		t.Fatal("unpaired dimensions accepted")
	}
}
func TestReferenceRoleCountAndResolution(t *testing.T) {
	p := writeFixture(t, "reference.png", string(fixturePNG(t)))
	r := taskRequest{Kind: "video", Operation: "generate", Model: "seedance-2.0-mini", Prompt: "reference", MediaOperation: "reference_image_video", Attachments: []attachment{{Field: "reference_images", Path: p}}}
	if err := prepareProfileRequest(&r, profileService()); err != nil {
		t.Fatal(err)
	}
	r.Attachments[0].Field = "image"
	if prepareProfileRequest(&r, profileService()) == nil {
		t.Fatal("wrong reference role accepted")
	}
	r.Attachments[0].Field = "reference_images"
	r.Model = "grok-imagine-video-1.5"
	r.Parameters = map[string]any{"resolution": "1080p"}
	if prepareProfileRequest(&r, profileService()) == nil {
		t.Fatal("reference mode resolution ignored")
	}
}
func TestMultipartPreservesStringEscapes(t *testing.T) {
	p := writeFixture(t, "reference.png", string(fixturePNG(t)))
	value := "line\nquoted \"value\"\\"
	_, contentType, body, err := taskRequestBody(taskRequest{Kind: "image", Operation: "edit", Model: "gpt-image-2", Prompt: "test", Parameters: map[string]any{"fixture": value}, Attachments: []attachment{{Field: "image", Path: p}}})
	if err != nil {
		t.Fatal(err)
	}
	defer body.(io.Closer).Close()
	_, params, _ := mime.ParseMediaType(contentType)
	reader := multipart.NewReader(body, params["boundary"])
	found := false
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(part)
		if part.FormName() == "fixture" {
			found = true
			if string(data) != value {
				t.Fatal("string was JSON escaped")
			}
		}
	}
	if !found {
		t.Fatal("field missing")
	}
}
func TestStatusFailurePreservesTaskAndActualMessage(t *testing.T) {
	for _, failedHTTP := range []bool{true, false} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if failedHTTP {
				w.WriteHeader(503)
				return
			}
			io.WriteString(w, `{"id":"paid-task","status":"failed","error":{"message":"The generated images appear to be unsafe. Try modifying the prompt or seeds."}}`)
		}))
		var out bytes.Buffer
		err := executeExistingTask("status", strings.NewReader(`{"kind":"image","task_id":"paid-task","model":"gpt-image-2"}`), &out, fixtureService(server))
		server.Close()
		got := decodeReceipt(t, &out)
		if err == nil || got.TaskID != "paid-task" || got.Model != "gpt-image-2" {
			t.Fatalf("lost task identity: %+v", got)
		}
		if !failedHTTP && !strings.Contains(got.ErrorMessage, "unsafe") {
			t.Fatal("public moderation message lost")
		}
	}
}
func TestWrongTaskIDAndUnknownStateStop(t *testing.T) {
	for _, body := range []string{`{"id":"different","status":"completed"}`, `{"id":"same","status":"surprise"}`} {
		got, err := applyTaskStatus(receipt{TaskID: "same"}, []byte(body), 200)
		if err == nil || got.TaskID != "same" {
			t.Fatal("bad task response accepted")
		}
	}
}
func TestTransportFailureSubmissionIsUnknown(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { conn, _, _ := w.(http.Hijacker).Hijack(); conn.Close() }))
	defer s.Close()
	var out bytes.Buffer
	executeTask(strings.NewReader(`{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"test"}`), &out, fixtureService(s))
	if decodeReceipt(t, &out).SubmissionOutcome != "unknown" {
		t.Fatal("lost unknown submission outcome")
	}
}
func TestRedirectNeverRepeatsPOST(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "/again")
		w.WriteHeader(307)
	}))
	defer s.Close()
	_, status, _, _, _, err := fixtureService(s).request(context.Background(), "POST", "/start", strings.NewReader(`{}`), "application/json")
	if err != nil || status != 307 || calls != 1 {
		t.Fatal("redirect repeated POST", calls, err)
	}
}
func TestDeadlineCancelsHTTPAndBackoff(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, _, _, _, _, err := fixtureService(s).request(ctx, "GET", "/status", nil, "")
	if err == nil {
		t.Fatal("deadline ignored")
	}
	if waitContext(ctx, time.Hour) {
		t.Fatal("backoff ignored cancellation")
	}
}
func TestRetryAfterDate(t *testing.T) {
	r := &http.Response{Header: make(http.Header)}
	r.Header.Set("Retry-After", time.Now().Add(10*time.Second).UTC().Format(http.TimeFormat))
	if n := retryAfter(r); n < 1 || n > 11 {
		t.Fatal("HTTP-date ignored")
	}
}
func TestMediaValidationAndDownloadResume(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("<html>not media</html>"), fixturePNG(t)[:24]} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Write(data)
		}))
		dir := t.TempDir()
		_, _, _, _, _, err := fixtureService(s).download(context.Background(), "/content", "image", dir)
		s.Close()
		entries, _ := os.ReadDir(dir)
		if err == nil || len(entries) != 0 {
			t.Fatal("invalid media accepted or partial file retained")
		}
	}
	calls := 0
	pngBytes := fixturePNG(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes)
	}))
	defer s.Close()
	dir := t.TempDir()
	svc := fixtureService(s)
	first, _, _, _, _, err := svc.download(context.Background(), "/content?index=0", "image", dir)
	if err != nil {
		t.Fatal(err)
	}
	second, _, _, _, _, err := svc.download(context.Background(), "/content?index=0", "image", dir)
	if err != nil || first != second || calls != 1 {
		t.Fatal("downloaded content fetched again")
	}
}
func TestDownloadRejectsDestinationCreatedDuringRequest(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprintf("symlink=%t", symlink), func(t *testing.T) {
			directory := t.TempDir()
			content := fixturePNG(t)
			private := writeFixture(t, "private.png", string(content))
			destination := filepath.Join(directory, fmt.Sprintf("puretokens-%x.png", sha256.Sum256([]byte("/content"))))
			if symlink {
				if err := os.Symlink(private, destination); err != nil {
					t.Skip("symlink creation unavailable on this platform")
				}
				if err := os.Remove(destination); err != nil {
					t.Fatal(err)
				}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error
				if symlink {
					err = os.Symlink(private, destination)
				} else {
					err = os.WriteFile(destination, content, 0600)
				}
				if err != nil {
					t.Error(err)
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Type", "image/png")
				w.Write(content)
			}))
			defer server.Close()
			file, _, _, _, _, err := fixtureService(server).download(context.Background(), "/content", "image", directory)
			if err == nil || file != "" {
				t.Fatal("unrelated concurrent output accepted as downloaded media")
			}
			entries, _ := os.ReadDir(directory)
			if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
				t.Fatal("partial download retained or concurrent file removed")
			}
			if symlink && validMediaFile(destination, "image/png") {
				t.Fatal("symlink accepted as a reusable output")
			}
		})
	}
}
func TestExclusiveDownloadCopyWithoutHardLinks(t *testing.T) {
	content := fixturePNG(t)
	source := writeFixture(t, "complete.png", string(content))
	destination := filepath.Join(t.TempDir(), "download.png")
	if err := copyDownloadExclusive(source, destination); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, content) || !validMediaFile(destination, "image/png") {
		t.Fatal("exclusive copy lost downloaded bytes")
	}
	if err := copyDownloadExclusive(source, destination); !os.IsExist(err) {
		t.Fatal("existing output was not protected")
	}
	got, _ = os.ReadFile(destination)
	if !bytes.Equal(got, content) {
		t.Fatal("existing output modified")
	}
	if _, err := os.Lstat(destination + ".incomplete"); !os.IsNotExist(err) {
		t.Fatal("completed copy retained its incomplete marker")
	}
}
func TestIncompleteDownloadCannotBeReused(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, fmt.Sprintf("puretokens-%x.png", sha256.Sum256([]byte("/content"))))
	if err := os.Mkdir(destination+".incomplete", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, fixturePNG(t), 0600); err != nil {
		t.Fatal(err)
	}
	if validMediaFile(destination, "image/png") {
		t.Fatal("incomplete output accepted despite pending publication")
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	file, _, _, _, _, err := fixtureService(server).download(context.Background(), "/content", "image", directory)
	if err == nil || file != "" || calls != 0 {
		t.Fatal("incomplete output reused or silently downloaded again")
	}
}
func TestContentDownloadsOnlyRequestedIndexAndPreservesTaskOnFailure(t *testing.T) {
	calls := 0
	imageBytes := fixturePNG(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("index") != "1" {
			t.Error("wrong index")
		}
		if calls == 1 {
			w.Header().Set("Content-Type", "image/png")
			w.Write(imageBytes)
		} else {
			w.WriteHeader(503)
		}
	}))
	defer s.Close()
	for i := 0; i < 2; i++ {
		payload, _ := json.Marshal(map[string]any{"kind": "image", "task_id": "paid-task", "task_status": "completed", "requested_count": 3, "index": 1, "output_dir": t.TempDir()})
		var out bytes.Buffer
		err := executeExistingTask("content", bytes.NewReader(payload), &out, fixtureService(s))
		r := decodeReceipt(t, &out)
		if r.TaskID != "paid-task" {
			t.Fatal("lost content task")
		}
		if i == 0 && (err != nil || r.DeliveryStatus != "downloaded_awaiting_host_delivery" || len(r.DownloadedPaths) != 1) {
			t.Fatal("false delivery status")
		}
		if i == 1 && (err == nil || r.Status != "completed") {
			t.Fatal("missing failure or completed task status lost")
		}
	}
}
func TestInitDoesNotVerifyInvalidCredential(t *testing.T) {
	reads := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		if r.URL.Path == "/v1" {
			io.WriteString(w, `{"status":"ok","name":"Pure Tokens API","base_url":"/v1"}`)
		} else {
			w.WriteHeader(401)
		}
	}))
	defer s.Close()
	var out bytes.Buffer
	err := executeInit(&out, fixtureService(s))
	var got initReceipt
	json.Unmarshal(out.Bytes(), &got)
	if err == nil || got.CredentialVerified || !got.APIIdentityConfirmed || reads != 2 {
		t.Fatal("public identity treated as key authentication")
	}
}
func TestInstalledModelDoesNotFetchCatalogBeforeSubmit(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/images/generations" || r.Method != "POST" {
			t.Error("unexpected catalog preflight")
		}
		io.WriteString(w, `{"id":"task-1","status":"pending"}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	err := executeTask(strings.NewReader(`{"kind":"image","operation":"generate","model":"gpt-image-2","prompt":"test","parameters":{"image_size":"1K"}}`), &out, fixtureService(server))
	if err != nil || calls != 1 {
		t.Fatal("core request preflight", err, calls)
	}
}
func TestProfileGapFetchesOnceAndPreservesExactID(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "GET" {
			io.WriteString(w, `{"data":[{"id":"new-model","capabilities":["image"],"input_schema":{"properties":{"n":{"type":"integer","enum":[1]}}}}]}`)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "new-model" {
			t.Error("model changed")
		}
		io.WriteString(w, `{"id":"task-1","status":"pending"}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	err := executeTask(strings.NewReader(`{"kind":"image","operation":"generate","model":"new-model","prompt":"test","parameters":{"n":1}}`), &out, fixtureService(server))
	if err != nil || calls != 2 {
		t.Fatal("profile gap did not resolve once", err, calls)
	}
}
func TestErrorEchoDoesNotExposeMatchingCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		io.WriteString(w, `{"error":{"message":"Rejected synthetic-fixture-token"}}`)
	}))
	defer server.Close()
	_, _, _, _, message, _ := fixtureService(server).request(context.Background(), "GET", "/status", nil, "")
	if strings.Contains(message, "synthetic-fixture-token") {
		t.Fatal("credential echo leaked")
	}
	for _, value := range []string{"bad\x1b[31m", "token=fixture", "see HTTPS://private.example", "api_key: fixture"} {
		if safePublicString(value) != "" {
			t.Fatal("unsafe message accepted")
		}
	}
}
func TestBOMConfigAndWorkBuddyMissingCredentialClassified(t *testing.T) {
	path := writeFixture(t, "models.json", "\ufeff"+`[{"url":"https://api.puretokensx.com/v1/chat/completions","apiKey":""}]`)
	_, err := credentialFromWorkBuddyFile(path)
	status, _, _ := credentialFailureDetails(err)
	if status != "workbuddy_credential_missing" {
		t.Fatal("missing credential reported ambiguous", status)
	}
}

func TestMetadataOnlyMP4IsNotACompletedVideo(t *testing.T) {
	data := []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0, 'i', 's', 'o', 'm', 'm', 'p', '4', '2'}
	p := writeFixture(t, "not-video.mp4", string(data))
	if validMediaFile(p, "video/mp4") {
		t.Fatal("ftyp metadata without video payload accepted")
	}
}
