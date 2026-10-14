package main

import "testing"

func TestCurrentCatalogImageTiersAndExactIDs(t *testing.T) {
	for _, model := range []string{"gpt-image-2", "gpt-image-2(Sub)", "gpt-image-2.5", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		for _, size := range []string{"1K", "2K", "4K", "8K"} {
			svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "image", model))
			r := taskRequest{Kind: "image", Operation: "generate", Model: model, Prompt: "synthetic", Parameters: map[string]any{"image_size": size, "aspect_ratio": "16:9"}}
			err := prepareProfileRequest(&r, svc)
			if (err == nil) != (size == "1K") || transport.calls != 0 || r.Model != model {
				t.Fatalf("image profile mismatch: %s/%s", model, size)
			}
		}
	}
	for _, model := range []string{"qwen-image-3.0", "qwen-image-3.0-pro", "wan2.7-image", "wan2.7-image-pro"} {
		svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "image", model))
		r := taskRequest{Kind: "image", Operation: "edit", Model: model, Prompt: "synthetic", Parameters: map[string]any{"images": []any{"https://example.com/reference.png"}}}
		if err := prepareProfileRequest(&r, svc); err != nil || transport.calls != 0 || r.route != "/v1/images/edits" {
			t.Fatalf("JSON URL edit %s: %v", model, err)
		}
		r.Parameters["images"] = []any{"https://127.0.0.1/reference.png"}
		if prepareProfileRequest(&r, svc) == nil {
			t.Fatal("private reference accepted")
		}
	}
}

func TestCurrentCatalogVideoOperations(t *testing.T) {
	for _, model := range []string{"seedance2.0"} {
		for _, count := range []int{1, 2} {
			svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "video", model))
			refs := []any{"https://example.com/reference.png"}
			if count == 2 {
				refs = append(refs, refs[0])
			}
			r := taskRequest{Kind: "video", Operation: "generate", MediaOperation: "reference_image_video", Model: model, Prompt: "synthetic", Parameters: map[string]any{"duration": 8, "reference_images": refs}}
			err := prepareProfileRequest(&r, svc)
			if (err == nil) != (count == 1) || transport.calls != 0 {
				t.Fatalf("video reference %s/%d: %v", model, count, err)
			}
			r.MediaOperation = "image_to_video"
			r.Parameters = map[string]any{"duration": 8, "image": "https://example.com/reference.png"}
			if err := prepareProfileRequest(&r, svc); err != nil || transport.calls != 0 {
				t.Fatalf("scalar image input: %v", err)
			}
			delete(r.Parameters, "duration")
			if prepareProfileRequest(&r, svc) == nil {
				t.Fatal("missing required duration accepted")
			}
		}
	}
	for _, count := range []int{1, 30, 31} {
		svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "video", "seedance-2.5"))
		refs := make([]any, count)
		for i := range refs {
			refs[i] = "https://example.com/reference.png"
		}
		request := taskRequest{Kind: "video", Operation: "generate", MediaOperation: "video_generation", Model: "seedance-2.5", Prompt: "synthetic", Parameters: map[string]any{"reference_images": refs}}
		err := prepareProfileRequest(&request, svc)
		if (err == nil) != (count <= 30) || transport.calls != 0 {
			t.Fatalf("seedance-2.5 JSON references %d: %v", count, err)
		}
	}
	svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "video", "minimax_h3"))
	r := taskRequest{Kind: "video", Operation: "generate", Model: "minimax_h3", Prompt: "synthetic", Parameters: map[string]any{"last_frame_image": "https://example.com/last.png"}}
	if prepareProfileRequest(&r, svc) == nil {
		t.Fatal("last frame without first accepted")
	}
	r.Parameters["first_frame_image"] = "https://example.com/first.png"
	if err := prepareProfileRequest(&r, svc); err != nil {
		t.Fatal(err)
	}
	r.Parameters["video_urls"] = []any{"https://example.com/reference.mp4"}
	if prepareProfileRequest(&r, svc) == nil {
		t.Fatal("mixed frames and references accepted")
	}
	r.Parameters = map[string]any{"video_urls": []any{"https://127.0.0.1/reference.mp4"}}
	if prepareProfileRequest(&r, svc) == nil || transport.calls != 0 {
		t.Fatal("unsafe video URL caused a request")
	}
	svc, transport = catalogProfileService(t, readInstalledProfileFixture(t, "video", "omni"))
	r = taskRequest{Kind: "video", Operation: "edit", Model: "omni", Prompt: "synthetic", Parameters: map[string]any{"video": "https://example.com/reference.mp4", "duration": 4}}
	if err := prepareProfileRequest(&r, svc); err != nil || transport.calls != 0 || r.route != "/v1/videos/edits" {
		t.Fatalf("omni edit: %v", err)
	}
}

func TestCurrentCatalogDoesNotInventAttachmentOperations(t *testing.T) {
	for _, model := range []string{"seedance-2.5", "seedance2.0", "veo_fast", "veo_lite", "veo_quan", "qwen-image-3.0"} {
		kind := "video"
		if model == "qwen-image-3.0" {
			kind = "image"
		}
		svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, kind, model))
		r := taskRequest{Kind: kind, Operation: "generate", Model: model, Prompt: "synthetic", MediaOperation: "image_to_video", Attachments: []attachment{{Field: "first_frame_image", Path: "fixture.png"}}, Parameters: map[string]any{"duration": 8}}
		if kind == "image" {
			r.Operation = "edit"
			r.MediaOperation = "image_edit"
			r.Parameters = nil
			r.Attachments[0].Field = "images"
		}
		if prepareProfileRequest(&r, svc) == nil || transport.calls != 1 {
			t.Fatalf("undeclared native attachment %s accepted or repeated lookup", model)
		}
	}
}

func TestH3FirstLastFramesUseDeclaredImageToVideo(t *testing.T) {
	profile := readInstalledProfileFixture(t, "video", "minimax_h3")
	for _, op := range []string{"image_to_video", "first_last_frame_video"} {
		svc, transport := catalogProfileService(t, profile)
		request := taskRequest{Kind: "video", Operation: "generate", MediaOperation: op, Model: "minimax_h3", Prompt: "synthetic", Attachments: []attachment{{Field: "first_frame_image", Path: "/synthetic/first.png"}, {Field: "last_frame_image", Path: "/synthetic/last.png"}}}
		err := prepareProfileRequest(&request, svc)
		if op == "image_to_video" && (err != nil || transport.calls != 0 || request.route != "/v1/videos") {
			t.Fatalf("declared H3 first/last input failed: %v", err)
		}
		if op == "first_last_frame_video" && (err == nil || transport.calls != 1) {
			t.Fatalf("undeclared operation was not bounded and rejected: %v", err)
		}
		t.Logf("operation=%s accepted=%t simulated_catalog_reads=%d", op, err == nil, transport.calls)
	}
}
