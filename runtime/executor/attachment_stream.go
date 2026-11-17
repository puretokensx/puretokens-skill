package main

import (
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"os"
)

// Only open handles and digests live in memory. No attachment copies or caches.
type openAttachment struct {
	file     *os.File
	info     os.FileInfo
	digest   [sha256.Size]byte
	mimeType string
}

func sameAttachment(a, b os.FileInfo) bool {
	return a != nil && b != nil && os.SameFile(a, b) &&
		b.Mode().IsRegular() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func closeAttachments(files []openAttachment) {
	for _, file := range files {
		_ = file.file.Close()
	}
}

func openAttachments(request taskRequest) (files []openAttachment, err error) {
	defer func() {
		if err != nil {
			closeAttachments(files)
		}
	}()
	var total int64
	limit := attachmentLimit(request.Kind)
	for _, attachment := range request.Attachments {
		f, openErr := os.Open(attachment.Path)
		if openErr != nil {
			return files, errors.New("attachment unavailable")
		}
		files = append(files, openAttachment{file: f})
		info, statErr := f.Stat()
		if statErr != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > limit {
			return files, errors.New("attachment changed or exceeds limit")
		}
		if attachment.snapshot != nil && !sameAttachment(attachment.snapshot, info) {
			return files, errors.New("attachment changed after validation")
		}
		total += info.Size()
		if total > limit {
			return files, errors.New("attachments exceed total limit")
		}
		// Sniff the same bytes included in the snapshot digest, not the filename.
		var prefix [512]byte
		prefixSize, readErr := io.ReadFull(f, prefix[:])
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return files, errors.New("attachment unavailable")
		}
		h := sha256.New()
		_, _ = h.Write(prefix[:prefixSize])
		n, readErr := io.Copy(h, io.LimitReader(f, info.Size()+1-int64(prefixSize)))
		n += int64(prefixSize)
		after, statErr := f.Stat()
		if readErr != nil || statErr != nil || n != info.Size() || !sameAttachment(info, after) {
			return files, errors.New("attachment changed while preparing")
		}
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return files, err
		}
		files[len(files)-1].info = info
		files[len(files)-1].mimeType = "application/octet-stream"
		switch detected := http.DetectContentType(prefix[:prefixSize]); detected {
		case "image/png", "image/jpeg", "image/webp":
			files[len(files)-1].mimeType = detected
		}
		copy(files[len(files)-1].digest[:], h.Sum(nil))
		if attachment.hasSnapshotDigest && attachment.snapshotDigest != files[len(files)-1].digest {
			return files, errors.New("attachment content changed after validation")
		}
	}
	return files, nil
}

func (attachment openAttachment) copyTo(output io.Writer) error {
	h := sha256.New()
	// Do not transmit even one byte beyond the verified length.
	n, err := io.Copy(io.MultiWriter(output, h), io.LimitReader(attachment.file, attachment.info.Size()))
	var extra [1]byte
	more, endErr := attachment.file.Read(extra[:])
	after, statErr := attachment.file.Stat()
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))
	if err != nil || n != attachment.info.Size() || more != 0 || endErr != io.EOF ||
		statErr != nil || !sameAttachment(attachment.info, after) || digest != attachment.digest {
		return errors.New("attachment changed during transmission")
	}
	return nil
}
