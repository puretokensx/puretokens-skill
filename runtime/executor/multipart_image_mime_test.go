package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestMultipartImageMIMEUsesBytesNotFilename(t *testing.T) {
	webp, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAfQ//73v/+BiOh/AAA=")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, filename, kind, field, want string
		data                              []byte
	}{
		{"png without extension", "reference", "image", "image", "image/png", fixturePNG(t)},
		{"jpeg with wrong extension", "reference.png", "image", "image", "image/jpeg", fixtureJPEG(t)},
		{"webp", "reference.webp", "image", "image", "image/webp", webp},
		{"png mask", "mask.bin", "image", "mask", "image/png", fixturePNG(t)},
		{"video reference", "frame.bin", "video", "image", "image/png", fixturePNG(t)},
		{"unrecognized bytes", "not-an-image.png", "image", "image", "application/octet-stream", []byte("not an image")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFixture(t, tc.filename, string(tc.data))
			r := taskRequest{Kind: tc.kind, Operation: "edit", Model: "gpt-image-2.6", Prompt: "fixture", Attachments: []attachment{{Field: tc.field, Path: path}}}
			if err := validateTaskRequest(r); err != nil {
				t.Fatal(err)
			}
			_, contentType, body, err := taskRequestBody(r)
			if err != nil {
				t.Fatal(err)
			}
			defer body.(io.Closer).Close()
			_, params, err := mime.ParseMediaType(contentType)
			if err != nil {
				t.Fatal(err)
			}
			form, err := multipart.NewReader(body, params["boundary"]).ReadForm(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer form.RemoveAll()
			files := form.File[tc.field]
			if len(files) != 1 || files[0].Filename != filepath.Base(path) {
				t.Fatal("attachment role or filename changed")
			}
			file, err := files[0].Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(file)
			file.Close()
			if err != nil || !bytes.Equal(data, tc.data) {
				t.Fatal("attachment bytes changed")
			}
			if got := files[0].Header.Get("Content-Type"); got != tc.want {
				t.Fatalf("file Content-Type = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMultipartImageEditSubmitsTwoTypedReferencesOnce(t *testing.T) {
	data := [][]byte{fixturePNG(t), fixtureJPEG(t)}
	wantTypes := []string{"image/png", "image/jpeg"}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/edits" {
			t.Error("unexpected API call")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()
		for key, want := range map[string]string{"model": "gpt-image-2.6", "prompt": "two references", "async": "true", "image_size": "1K", "aspect_ratio": "1:1"} {
			if r.FormValue(key) != want {
				t.Errorf("field %s changed", key)
			}
		}
		files := r.MultipartForm.File["image"]
		if len(files) != 2 {
			t.Error("reference count changed")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for index, file := range files {
			if file.Header.Get("Content-Type") != wantTypes[index] {
				w.WriteHeader(http.StatusBadRequest)
				io.WriteString(w, `{"error":{"code":"invalid_request"}}`)
				return
			}
			reader, err := file.Open()
			if err != nil {
				t.Error(err)
				return
			}
			body, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || !bytes.Equal(body, data[index]) {
				t.Error("reference bytes or order changed")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, `{"id":"pic-multipart-fixture","object":"image","status":"pending"}`)
	}))
	defer server.Close()
	r := taskRequest{Kind: "image", Operation: "edit", MediaOperation: "image_edit", Model: "gpt-image-2.6", Prompt: "two references"}
	for _, raw := range data {
		r.Attachments = append(r.Attachments, attachment{Field: "image", Path: writeFixture(t, "reference.bin", string(raw))})
	}
	if err := validateTaskRequest(r); err != nil {
		t.Fatal(err)
	}
	svc := fixtureService(server)
	if err := prepareProfileRequest(&r, svc); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := executePreparedTask(&out, svc, r)
	receipt := decodeReceipt(t, &out)
	if err != nil || !receipt.OK || receipt.SubmissionOutcome != "accepted" || calls != 1 {
		t.Fatalf("edit rejected or repeated: err=%v outcome=%s calls=%d", err, receipt.SubmissionOutcome, calls)
	}
}
