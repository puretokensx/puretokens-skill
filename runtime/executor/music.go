package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const musicModel = "stepaudio-3-music-preview"
const maxMusicBytes int64 = 32 << 20
const musicSubmitPath = "/v1/audio/music/submit"

// Music shares the existing async task engine, but only its reviewed text-to-
// music contract is admitted. Lyrics are sent once and never persisted in receipts.
func validateMusicRequest(r taskRequest) error {
	invalid := errors.New("Use reviewed text-to-music parameters, one output, and no attachments or task metadata on submission.")
	if r.Model != musicModel || r.MediaOperation != "" || len(r.Attachments) != 0 || r.Index != 0 || r.RequestedCount > 1 {
		return invalid
	}
	if r.Operation == "continue" {
		if r.Prompt != "" || (r.OriginalOperation != "" && r.OriginalOperation != "generate") {
			return invalid
		}
		for k, v := range r.Parameters {
			if k == "instrumental" {
				if _, ok := v.(bool); !ok {
					return invalid
				}
			} else if k == "response_format" {
				if v != "mp3" && v != "wav" {
					return invalid
				}
			} else {
				return invalid
			}
		}
		return nil
	}
	if r.Operation != "generate" || !utf8.ValidString(r.Prompt) || strings.TrimSpace(r.Prompt) == "" || utf8.RuneCountInString(r.Prompt) > 1000 {
		return invalid
	}
	instrumental, ok := r.Parameters["instrumental"].(bool)
	if !ok || (r.Parameters["response_format"] != "mp3" && r.Parameters["response_format"] != "wav") {
		return invalid
	}
	for k, v := range r.Parameters {
		if k == "lyrics" {
			text, ok := v.(string)
			if !ok || !utf8.ValidString(text) || utf8.RuneCountInString(text) > 4000 || (instrumental && text != "") {
				return invalid
			}
		} else if k != "instrumental" && k != "response_format" {
			return invalid
		}
	}
	return nil
}

func prepareMusicRequest(r *taskRequest) error {
	if err := validateMusicRequest(*r); err != nil {
		return err
	}
	return prepareTaskAudioOutput(r)
}

func prepareTaskAudioOutput(r *taskRequest) error {
	if !filepath.IsAbs(r.OutputDir) {
		return errors.New("Choose an existing absolute output directory before submitting music.")
	}
	info, err := os.Stat(r.OutputDir)
	if err != nil || !info.IsDir() {
		return errors.New("The music output directory is unavailable; no task was sent.")
	}
	f, err := os.CreateTemp(r.OutputDir, ".puretokens-music-check-*")
	if err != nil {
		return errors.New("The music output directory is not writable; no task was sent.")
	}
	name := f.Name()
	closeErr := f.Close()
	removeErr := os.Remove(name)
	if closeErr != nil || removeErr != nil {
		return errors.New("The music output directory could not be verified; no task was sent.")
	}
	r.RequestedCount = 1
	return nil
}

func musicRequestBody(r taskRequest) (string, string, io.Reader, error) {
	if err := validateMusicRequest(r); err != nil {
		return "", "", nil, err
	}
	payload := map[string]any{"model": r.Model, "caption": r.Prompt, "instrumental": r.Parameters["instrumental"], "format": r.Parameters["response_format"]}
	if text, ok := r.Parameters["lyrics"].(string); ok {
		payload["lyrics"] = text
	}
	data, err := json.Marshal(payload)
	return musicSubmitPath, "application/json", bytes.NewReader(data), err
}

func mediaKindPrefix(kind string) string {
	if isTaskAudio(kind) {
		return "audio/"
	}
	return kind + "/"
}
