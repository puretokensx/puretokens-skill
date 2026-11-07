package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Stored only inside an explicit task record. Kind/task/index are bound by the
// record and its deterministic filename; SHA256 binds the actual downloaded bytes.
type downloadProof struct {
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
	MediaType string `json:"media_type"`
}

func validDownloadProof(proof downloadProof, kind string) bool {
	digest, err := hex.DecodeString(proof.SHA256)
	return err == nil && len(digest) == sha256.Size && strings.ToLower(proof.SHA256) == proof.SHA256 &&
		proof.Bytes > 0 && proof.Bytes <= attachmentLimit(kind) && (proof.MediaType != "image/svg+xml" || proof.Bytes <= maxSVGBytes) &&
		strings.HasPrefix(proof.MediaType, mediaKindPrefix(kind)) &&
		contains([]string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/avif", "image/svg+xml", "video/mp4", "video/webm", "audio/mpeg", "audio/wav"}, proof.MediaType)
}

func fingerprintDownload(path, mediaType string) (downloadProof, error) {
	var proof downloadProof
	if !filepath.IsAbs(path) {
		return proof, errors.New("download is not a local absolute path")
	}
	if !validMediaFile(path, mediaType) {
		return proof, errors.New("download is not valid media")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return proof, errors.New("download unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return proof, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !sameAttachment(before, info) {
		return proof, errors.New("download changed")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(file, info.Size()+1))
	after, statErr := file.Stat()
	pathInfo, pathErr := os.Lstat(path)
	if err != nil || statErr != nil || pathErr != nil || n != info.Size() ||
		!sameAttachment(info, after) || !sameAttachment(info, pathInfo) {
		return proof, errors.New("download changed while verifying")
	}
	return downloadProof{SHA256: hex.EncodeToString(h.Sum(nil)), Bytes: n, MediaType: mediaType}, nil
}

func matchesDownloadProof(path string, proof downloadProof) bool {
	current, err := fingerprintDownload(path, proof.MediaType)
	return err == nil && current == proof
}
