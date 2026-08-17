package services

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// saveUpload persists an uploaded image under uploads/subdir/<id><ext> and
// returns the public URL path. Rejects files that are too large or not in the
// allowed MIME set (header sniffing with filename extension fallback).
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
	name := id + "." + allowedImageTypes[mime]
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o640); err != nil {
		return "", err
	}
	return "/uploads/" + subdir + "/" + name, nil
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