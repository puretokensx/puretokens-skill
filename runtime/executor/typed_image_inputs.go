package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
)

// Exact reviewed routes accept inline raster bytes with typed roles. Never
// rehost user files or collapse frame/reference roles into input_reference.
func typedImageRequestBody(r taskRequest) (string, string, io.Reader, error) {
	invalid := errors.New("The local reference is unavailable, changed, oversized or not PNG, JPEG or WebP. No task was sent.")
	masked := r.MediaOperation == "masked_image_edit"
	if masked {
		urls := array(r.Parameters["image_urls"])
		if r.Kind != "image" || r.Operation != "edit" || reviewedMediaPolicies()[r.Model].LocalMaskEncoding != "data_url" || len(r.Attachments) != 1 || r.Attachments[0].Field != "mask" || len(urls) < 1 || len(urls) > 10 || validateReferences(r.Parameters["image_urls"], []string{"https_url"}) != nil {
			return "", "", nil, invalid
		}
	}
	files, err := openAttachments(r)
	if err != nil {
		return "", "", nil, invalid
	}
	defer closeAttachments(files)
	payload := map[string]any{"model": r.Model, "prompt": r.Prompt}
	for k, v := range r.Parameters {
		if k == "model" || k == "prompt" || k == "async" {
			return "", "", nil, invalid
		}
		payload[k] = v
	}
	if r.Kind == "image" {
		payload["async"] = true
	}
	total := 0
	for i, file := range files {
		path := r.Attachments[i].Path
		current, err := os.Lstat(path)
		if err != nil || !sameAttachment(file.info, current) {
			return "", "", nil, invalid
		}
		if _, err := os.Lstat(path + ".incomplete"); !os.IsNotExist(err) {
			return "", "", nil, invalid
		}
		field := r.Attachments[i].Field
		if (!contains([]string{"images", "first_frame_image", "last_frame_image", "reference_images"}, field) && !(masked && field == "mask")) || r.Parameters[field] != nil {
			return "", "", nil, invalid
		}
		// Bound allocation before reading. The cap covers the entire encoded body.
		if file.info.Size() > 14<<20 || total+base64.StdEncoding.EncodedLen(int(file.info.Size()))+64 > 20<<20 {
			return "", "", nil, invalid
		}
		var raw bytes.Buffer
		if file.copyTo(&raw) != nil {
			return "", "", nil, invalid
		}
		mime := http.DetectContentType(raw.Bytes())
		if !validRasterSnapshot(raw.Bytes(), mime) || (masked && mime != "image/png") {
			return "", "", nil, invalid
		}
		value := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw.Bytes())
		total += len(value)
		if field == "images" || field == "reference_images" {
			a, _ := payload[field].([]string)
			payload[field] = append(a, value)
		} else {
			if payload[field] != nil {
				return "", "", nil, invalid
			}
			payload[field] = value
		}
	}
	data, err := json.Marshal(payload)
	if err != nil || len(data) > 20<<20 {
		return "", "", nil, invalid
	}
	return endpointFor(r), "application/json", bytes.NewReader(data), nil
}

// Validate the immutable bytes that will be sent, not a reopened file path.
func validRasterSnapshot(data []byte, mime string) bool {
	if len(data) > 14<<20 || !validMediaPrefix(data, mime) {
		return false
	}
	switch mime {
	case "image/png":
		return len(data) >= 45 && bytes.Equal(data[len(data)-12:], []byte{0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}) && validDecodedImage(bytes.NewReader(data), png.DecodeConfig, png.Decode)
	case "image/jpeg":
		return bytes.Equal(data[len(data)-2:], []byte{255, 217}) && validDecodedImage(bytes.NewReader(data), jpeg.DecodeConfig, jpeg.Decode)
	case "image/webp":
		return int64(len(data)) == int64(binary.LittleEndian.Uint32(data[4:8]))+8 && validWebPChunks(bytes.NewReader(data), 12, int64(len(data)), false)
	}
	return false
}
