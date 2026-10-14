package main

import "testing"

func TestImageModelAspectRatios(t *testing.T) {
	for _, model := range []string{"gpt-image-2.5", "gpt-image-2(Sub)", "gpt-image-2"} {
		for _, operation := range []string{"generate", "edit"} {
			for _, ratio := range []string{"3:1", "1:3", "4:5", "5:4", "16:9"} {
				t.Run(model+"/"+operation+"/"+ratio, func(t *testing.T) {
					allowed := ratio != "3:1" && ratio != "1:3"
					svc, transport := catalogProfileService(t, readInstalledProfileFixture(t, "image", model))
					request := taskRequest{
						Kind: "image", Operation: operation, Model: model, Prompt: "fixture",
						Parameters: map[string]any{"aspect_ratio": ratio, "image_size": "1K"},
					}
					if operation == "edit" {
						request.Attachments = []attachment{{Field: "image", Path: "fixture.png"}}
					}
					err := prepareProfileRequest(&request, svc)
					if (err == nil) != allowed {
						t.Fatalf("allowed=%t: %v", allowed, err)
					}
					if transport.calls != 0 {
						t.Fatal("installed ratio validation must not read the live catalog")
					}
					if allowed {
						if recordFromRequest(request).Parameters["aspect_ratio"] != ratio ||
							taskReceipt(request, "", "").Parameters["aspect_ratio"] != ratio {
							t.Fatal("validated ratio lost from the task record or receipt")
						}
					}
				})
			}
		}
	}
}

func TestImage25QualityAndEditBoundaries(t *testing.T) {
	for _, model := range []string{"gpt-image-2", "gpt-image-2.5", "gpt-image-2(Sub)"} {
		for _, quality := range []string{"auto", "low", "medium", "high"} {
			r := taskRequest{Kind: "image", Operation: "generate", Model: model, Prompt: "fixture", Parameters: map[string]any{"quality": quality, "image_size": "1K", "aspect_ratio": "16:9"}}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatal(err)
			}
			if recordFromRequest(r).Parameters["quality"] != quality {
				t.Fatal("quality lost from record")
			}
		}
		r := taskRequest{Kind: "image", Operation: "edit", Model: model, Prompt: "fixture"}
		if prepareProfileRequest(&r, profileService()) == nil {
			t.Fatal("edit without references accepted")
		}
		r.Operation = "generate"
		r.Parameters = map[string]any{"quality": "private free text"}
		if prepareProfileRequest(&r, profileService()) == nil {
			t.Fatal("invalid quality accepted")
		}
		if _, ok := taskReceipt(r, "", "").Parameters["quality"]; ok {
			t.Fatal("unsafe quality retained")
		}
	}
}
func TestSeedanceFrameReferences(t *testing.T) {
	for _, model := range []string{"seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini"} {
		r := taskRequest{Kind: "video", Operation: "generate", Model: model, Prompt: "fixture", Parameters: map[string]any{"first_frame_image": "https://example.com/first.png", "last_frame_image": "https://example.com/last.png"}}
		if err := prepareProfileRequest(&r, profileService()); err != nil {
			t.Fatal(err)
		}
		if len(taskReceipt(r, "", "").Parameters) != 0 {
			t.Fatal("reference persisted")
		}
		r.Parameters["reference_videos"] = []any{"https://example.com/ref.mp4"}
		if prepareProfileRequest(&r, profileService()) == nil {
			t.Fatal("mixed frame/reference accepted")
		}
		r.Parameters = map[string]any{}
		r.MediaOperation = "first_last_frame_video"
		r.Attachments = []attachment{{Field: "first_frame_image", Path: "first.png"}, {Field: "last_frame_image", Path: "last.png"}}
		if err := prepareProfileRequest(&r, profileService()); err != nil {
			t.Fatal(err)
		}
		r.Attachments = r.Attachments[:1]
		if prepareProfileRequest(&r, profileService()) == nil {
			t.Fatal("missing last frame accepted")
		}
	}
}

func TestImage25ReferenceEdits(t *testing.T) {
	for _, model := range []string{"gpt-image-2", "gpt-image-2.5", "gpt-image-2(Sub)"} {
		for _, count := range []int{1, 6, 7, 10, 11} {
			r := taskRequest{Kind: "image", Operation: "edit", Model: model, Prompt: "fixture", Parameters: map[string]any{"quality": "high"}}
			for i := 0; i < count; i++ {
				r.Attachments = append(r.Attachments, attachment{Field: "image", Path: "fixture.png"})
			}
			err := prepareProfileRequest(&r, profileService())
			if (err == nil) != (count <= 10) {
				t.Fatalf("%s count %d: %v", model, count, err)
			}
		}
		for _, count := range []int{1, 6, 7, 10, 11} {
			refs := make([]any, count)
			for i := range refs {
				refs[i] = "https://example.com/ref.png"
			}
			r := taskRequest{Kind: "image", Operation: "generate", Model: model, Prompt: "fixture", Parameters: map[string]any{"image": refs}}
			err := prepareProfileRequest(&r, profileService())
			if (err == nil) != (count <= 10) {
				t.Fatalf("%s URL count %d: %v", model, count, err)
			}
			if _, exists := taskReceipt(r, "", "").Parameters["image"]; exists {
				t.Fatal("references persisted")
			}
		}
	}
}
func TestNanoBananaPortraitLandscapeRatios(t *testing.T) {
	for _, model := range []string{"nano-banana-2", "nano-banana-2-lite", "nano-banana-pro"} {
		for _, ratio := range []string{"3:4", "4:3", "4:5", "5:4"} {
			r := taskRequest{Kind: "image", Operation: "generate", Model: model, Prompt: "fixture", Parameters: map[string]any{"aspect_ratio": ratio, "image_size": "1K"}}
			if err := prepareProfileRequest(&r, profileService()); err != nil {
				t.Fatalf("%s %s: %v", model, ratio, err)
			}
		}
		for _, ratio := range []string{"13:13", "0:1"} {
			r := taskRequest{Kind: "image", Operation: "generate", Model: model, Prompt: "fixture", Parameters: map[string]any{"aspect_ratio": ratio, "image_size": "1K"}}
			if prepareProfileRequest(&r, profileService()) == nil {
				t.Fatalf("%s accepted ratio absent from live schema: %s", model, ratio)
			}
		}
	}
	r := taskRequest{Kind: "image", Operation: "generate", Model: "nano-banana-2-lite", Prompt: "fixture", Parameters: map[string]any{"aspect_ratio": "3:4", "image_size": "2K"}}
	if prepareProfileRequest(&r, profileService()) == nil {
		t.Fatal("lite accepted unsupported 2K")
	}
}
