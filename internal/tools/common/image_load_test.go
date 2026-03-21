package common

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadImage_PNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	content := []byte("fake-png-data")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	img, err := LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage error: %v", err)
	}
	if img.Path != path {
		t.Errorf("Path = %q, want %q", img.Path, path)
	}
	if img.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want %q", img.MediaType, "image/png")
	}
	if img.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", img.Size, len(content))
	}

	decoded, err := base64.StdEncoding.DecodeString(img.Base64)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if string(decoded) != string(content) {
		t.Errorf("decoded content mismatch")
	}
}

func TestLoadImage_JPEG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}

	img, err := LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage error: %v", err)
	}
	if img.MediaType != "image/jpeg" {
		t.Errorf("MediaType = %q, want image/jpeg", img.MediaType)
	}
}

func TestLoadImage_NotFound(t *testing.T) {
	_, err := LoadImage("/nonexistent/path/image.png")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestLoadImage_Directory(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadImage(dir)
	if err == nil {
		t.Fatal("expected error for directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %v", err)
	}
}

func TestLoadImage_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.png")
	// MaxImageSize を超えるファイルを作成（疎ファイルでサイズだけ大きく）
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxImageSize + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	_, err = LoadImage(path)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error should mention 'too large', got: %v", err)
	}
}

func TestLoadImage_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.bmp")
	if err := os.WriteFile(path, []byte("bmp"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadImage(path)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention 'unsupported', got: %v", err)
	}
}
