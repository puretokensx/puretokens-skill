package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

// Check a supported binary format, not merely the server's Content-Type.
func validMediaPrefix(data []byte, contentType string) bool {
	switch contentType {
	case "image/png":
		return len(data) >= 24 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) && string(data[12:16]) == "IHDR" && binary.BigEndian.Uint32(data[16:20]) > 0 && binary.BigEndian.Uint32(data[20:24]) > 0
	case "image/jpeg":
		return len(data) >= 4 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
	case "image/gif":
		return len(data) >= 13 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a")
	case "image/webp":
		return len(data) >= 20 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	case "image/avif":
		return len(data) >= 16 && string(data[4:8]) == "ftyp" && (bytes.Contains(data[8:], []byte("avif")) || bytes.Contains(data[8:], []byte("avis")))
	case "audio/mpeg":
		return (len(data) >= 10 && string(data[:3]) == "ID3") || (len(data) >= 4 && data[0] == 0xff && data[1]&0xe0 == 0xe0)
	case "audio/wav":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE"
	case "video/mp4":
		return len(data) >= 24 && string(data[4:8]) == "ftyp"
	case "video/webm":
		return len(data) >= 16 && bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) && bytes.Contains(data, []byte("webm"))
	}
	return false
}

func validMediaFile(path, contentType string) bool {
	if _, err := os.Lstat(path + ".incomplete"); !os.IsNotExist(err) {
		return false
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() || !os.SameFile(before, stat) {
		return false
	}
	if contentType == "image/svg+xml" {
		if stat.Size() <= 0 || stat.Size() > maxSVGBytes {
			return false
		}
		data, err := io.ReadAll(io.LimitReader(f, maxSVGBytes+1))
		return err == nil && int64(len(data)) == stat.Size() && validSVG(data)
	}
	prefix := make([]byte, 512)
	n, _ := f.Read(prefix)
	if !validMediaPrefix(prefix[:n], contentType) {
		return false
	}
	limit := maxImageBytes
	if contentType == "video/mp4" || contentType == "video/webm" {
		limit = maxVideoBytes
	}
	if contentType == "audio/mpeg" || contentType == "audio/wav" {
		limit = maxMusicBytes
	}
	if stat.Size() > limit {
		return false
	}
	if contentType == "audio/mpeg" || contentType == "audio/wav" {
		if stat.Size() > maxMusicBytes {
			return false
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return false
		}
		data, err := io.ReadAll(io.LimitReader(f, maxMusicBytes+1))
		if err != nil || int64(len(data)) != stat.Size() {
			return false
		}
		if contentType == "audio/mpeg" {
			return validMP3(data)
		}
		return validWAV(data)
	}
	if contentType == "image/gif" {
		_, err = f.Seek(0, io.SeekStart)
		return err == nil && validGIF(f)
	}
	if contentType == "video/webm" {
		return validWebM(f, stat.Size())
	}
	if contentType == "image/png" {
		if stat.Size() < 45 {
			return false
		}
		tail := make([]byte, 12)
		_, err = f.ReadAt(tail, stat.Size()-12)
		return err == nil && bytes.Equal(tail, []byte{0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}) && validDecodedImage(f, png.DecodeConfig, png.Decode)
	}
	if contentType == "image/jpeg" {
		tail := make([]byte, 2)
		_, err = f.ReadAt(tail, stat.Size()-2)
		return err == nil && bytes.Equal(tail, []byte{255, 217}) && validDecodedImage(f, jpeg.DecodeConfig, jpeg.Decode)
	}
	if contentType == "image/webp" {
		return stat.Size() == int64(binary.LittleEndian.Uint32(prefix[4:8]))+8 && validWebPChunks(f, 12, stat.Size(), false)
	}
	if contentType == "video/mp4" || contentType == "image/avif" {
		var offset int64
		hasMetadata, hasPayload := false, false
		for offset < stat.Size() {
			var header [16]byte
			if _, err = f.ReadAt(header[:8], offset); err != nil {
				return false
			}
			boxType := string(header[4:8])
			if boxType == "moov" || boxType == "moof" || boxType == "meta" {
				hasMetadata = true
			}
			if boxType == "mdat" {
				hasPayload = true
			}
			size := int64(binary.BigEndian.Uint32(header[:4]))
			minSize := int64(8)
			if size == 1 {
				if _, err = f.ReadAt(header[8:], offset+8); err != nil {
					return false
				}
				raw := binary.BigEndian.Uint64(header[8:])
				if raw > uint64(stat.Size()) {
					return false
				}
				size = int64(raw)
				minSize = 16
			}
			if size == 0 {
				return stat.Size()-offset > 8 && hasMetadata && (hasPayload || contentType == "image/avif")
			}
			if size < minSize || size > stat.Size()-offset {
				return false
			}
			offset += size
		}
		return hasMetadata && (hasPayload || contentType == "image/avif")
	}
	_, err = f.Seek(0, io.SeekStart)
	return err == nil
}

// DecodeConfig bounds allocation before the standard decoder verifies the
// actual pixel stream. The limit accommodates 4K outputs and 8K widescreen.
const maxDecodedImagePixels = 32 << 20

func validDecodedImage(f io.ReadSeeker, config func(io.Reader) (image.Config, error), decode func(io.Reader) (image.Image, error)) bool {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false
	}
	// Both reads use one bounded snapshot so a concurrent file edit cannot
	// replace a checked header with unbounded dimensions before decoding.
	data, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
	if err != nil || int64(len(data)) > maxImageBytes {
		return false
	}
	dimensions, err := config(bytes.NewReader(data))
	if err != nil || dimensions.Width <= 0 || dimensions.Height <= 0 || int64(dimensions.Width) > maxDecodedImagePixels/int64(dimensions.Height) {
		return false
	}
	_, err = decode(bytes.NewReader(data))
	return err == nil
}

// WebP uses structural validation, not a full codec decode. Require complete,
// nonempty image payloads (including each animation frame), not just RIFF size.
func validWebPChunks(r io.ReaderAt, start, end int64, frame bool) bool {
	images, frames := 0, 0
	animated, animationHeader := false, false
	for offset := start; offset < end; {
		var header [8]byte
		if end-offset < 8 {
			return false
		}
		if _, err := r.ReadAt(header[:], offset); err != nil {
			return false
		}
		length := int64(binary.LittleEndian.Uint32(header[4:]))
		body := offset + 8
		next := body + length + length%2
		if next > end {
			return false
		}
		var data [10]byte
		switch string(header[:4]) {
		case "VP8 ":
			if length <= 10 {
				return false
			}
			if _, err := r.ReadAt(data[:], body); err != nil || data[0]&1 != 0 || !bytes.Equal(data[3:6], []byte{0x9d, 0x01, 0x2a}) ||
				binary.LittleEndian.Uint16(data[6:8])&0x3fff == 0 || binary.LittleEndian.Uint16(data[8:10])&0x3fff == 0 {
				return false
			}
			images++
		case "VP8L":
			if length <= 5 {
				return false
			}
			if _, err := r.ReadAt(data[:5], body); err != nil || data[0] != 0x2f || data[4]&0xe0 != 0 {
				return false
			}
			images++
		case "VP8X":
			if frame || offset != start || length != 10 {
				return false
			}
			if _, err := r.ReadAt(data[:], body); err != nil {
				return false
			}
			animated = data[0]&2 != 0
		case "ANIM":
			if frame || !animated || animationHeader || frames > 0 || length != 6 {
				return false
			}
			animationHeader = true
		case "ANMF":
			if frame || !animated || !animationHeader || length <= 16 || !validWebPChunks(r, body+16, body+length, true) {
				return false
			}
			frames++
		}
		offset = next
	}
	return (images == 1 && frames == 0 && !animated) || (images == 0 && frames > 0 && animated)
}

// Filesystems such as exFAT cannot hard-link. The transient marker prevents
// reuse until copying has completed, including after an interrupted process.
func copyDownloadExclusive(source, destination string) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	marker := destination + ".incomplete"
	if err := os.Mkdir(marker, 0700); err != nil {
		return err
	}
	removeMarker := true
	defer func() {
		if !removeMarker {
			return
		}
		if removeErr := os.Remove(marker); err == nil {
			err = removeErr
		}
	}()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if removeErr := os.Remove(destination); removeErr != nil && !os.IsNotExist(removeErr) {
				removeMarker = false
			}
		}
	}()
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
