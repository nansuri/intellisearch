package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// allowedImageTypes maps a MIME type to the file extension to store on disk.
var allowedImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// ErrUploadRejected signals a file that is too large or not an allowed image.
var ErrUploadRejected = errors.New("upload rejected")

// saveUpload persists an uploaded image under uploads/subdir/<id>-<nonce><ext>
// and returns the public URL path. Rejects files that are too large or not in
// the allowed MIME set (header sniffing with filename extension fallback).
//
// The random nonce makes every upload a fresh URL, so browsers never serve a
// stale cached copy after a re-upload of the same image type (the previous
// fixed name like logo.png kept the same URL and the old image lingered).
func saveUpload(baseDir, subdir, id, filename string, data []byte, maxBytes int64) (string, error) {
	if int64(len(data)) > maxBytes {
		return "", ErrUploadRejected
	}
	mime := uploadMIME(filename, data)
	if _, ok := allowedImageTypes[mime]; !ok {
		return "", ErrUploadRejected
	}
	dir := filepath.Join(baseDir, subdir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	name := id + "-" + shortNonce() + "." + allowedImageTypes[mime]
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o640); err != nil {
		return "", err
	}
	return "/uploads/" + subdir + "/" + name, nil
}

// shortNonce returns 8 lowercase hex characters from crypto/rand.
func shortNonce() string {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand never fails on supported platforms; a time fallback keeps
		// uploads working regardless.
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))[:8]
	}
	return hex.EncodeToString(raw)
}

// removeStoredUpload deletes the file behind a stored /uploads/... URL,
// best-effort. Used when a logo/favicon/avatar is replaced or removed so old
// files don't accumulate. Never follows paths outside baseDir.
func removeStoredUpload(baseDir, url string) {
	if baseDir == "" || url == "" {
		return
	}
	rel := strings.TrimPrefix(url, "/uploads/")
	if rel == url || strings.Contains(rel, "..") {
		return
	}
	_ = os.Remove(filepath.Join(baseDir, rel))
}

// uploadMIME resolves the content type from the file data, preferring the
// declared filename extension when present.
func uploadMIME(filename string, data []byte) string {
	switch ext := strings.ToLower(filepath.Ext(filename)); ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	}
	detected := http.DetectContentType(data)
	if _, ok := allowedImageTypes[detected]; ok {
		return detected
	}
	return "application/octet-stream"
}
