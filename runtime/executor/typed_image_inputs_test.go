package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTypedLocalReferencesPreserveRolesInJSON(t *testing.T) {
	for _, tc := range []struct {
		model, kind, op, media string
		fields                 []string
	}{
		{"gemini-3.1-flash-image", "image", "edit", "", []string{"images", "images"}},
		{"veo-3.1", "video", "generate", "first_last_frame_video", []string{"first_frame_image", "last_frame_image"}},
		{"omni-flash", "video", "generate", "reference_image_video", []string{"reference_images", "reference_images"}},
	} {
		t.Run(tc.model, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, "reference.png")
			data := fixturePNG(t)
			os.WriteFile(file, data, 0600)
			r := taskRequest{Kind: tc.kind, Model: tc.model, Operation: tc.op, MediaOperation: tc.media, Prompt: "fixture"}
			for _, field := range tc.fields {
				r.Attachments = append(r.Attachments, attachment{Field: field, Path: file})
			}
			if err := validateTaskRequest(r); err != nil {
				t.Fatal(err)
			}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			path, ctype, body, err := taskRequestBody(r)
			if err != nil {
				t.Fatal(err)
			}
			if ctype != "application/json" || (tc.kind == "video" && path != "/v1/videos") {
				t.Fatal("wrong route/encoding", path, ctype)
			}
			raw, err := io.ReadAll(body)
			if err != nil {
				t.Fatal(err)
			}
			var p map[string]any
			if json.Unmarshal(raw, &p) != nil {
				t.Fatal("invalid JSON")
			}
			want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
			for _, field := range tc.fields {
				if field == "images" || field == "reference_images" {
					a, ok := p[field].([]any)
					if !ok || len(a) != 2 || a[0] != want || a[1] != want {
						t.Fatal("reference array lost")
					}
				} else if p[field] != want {
					t.Fatal("frame role lost")
				}
			}
			if p["input_reference"] != nil || bytes.Contains(raw, []byte(root)) {
				t.Fatal("role collapsed or path leaked")
			}
			os.WriteFile(file, []byte("changed"), 0600)
			if _, _, _, err = taskRequestBody(r); err == nil {
				t.Fatal("changed file accepted")
			}
		})
	}
}

func TestTypedReferenceValidatesCapturedBytes(t *testing.T) {
	original := fixturePNG(t)
	snapshot := append([]byte(nil), original...)
	if !validRasterSnapshot(snapshot, "image/png") {
		t.Fatal("valid snapshot rejected")
	}
	snapshot[len(snapshot)/2] ^= 0xff
	if validRasterSnapshot(snapshot, "image/png") {
		t.Fatal("corrupted captured bytes accepted")
	}
	if validRasterSnapshot(original[:len(original)-1], "image/png") || validRasterSnapshot(original, "image/jpeg") {
		t.Fatal("truncated or mismatched snapshot accepted")
	}
}

func TestTypedReferenceRejectsIncompleteOrLinkedFiles(t *testing.T) {
	for _, scenario := range []string{"incomplete", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "reference.png")
			if err := os.WriteFile(path, fixturePNG(t), 0600); err != nil {
				t.Fatal(err)
			}
			if scenario == "incomplete" {
				if err := os.WriteFile(path+".incomplete", []byte("in progress"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				link := filepath.Join(root, "link.png")
				if err := os.Symlink(path, link); err != nil {
					t.Skip("symlinks unavailable")
				}
				path = link
			}
			req := taskRequest{Kind: "image", Model: "gemini-3.1-flash-image", Operation: "edit", Prompt: "test", Attachments: []attachment{{Field: "images", Path: path}}}
			if _, _, _, err := typedImageRequestBody(req); err == nil {
				t.Fatal("unsafe file accepted")
			}
		})
	}
}
