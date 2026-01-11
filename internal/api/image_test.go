package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetImageMediaType(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "PNG file",
			path:    "test.png",
			want:    "image/png",
			wantErr: false,
		},
		{
			name:    "JPG file",
			path:    "test.jpg",
			want:    "image/jpeg",
			wantErr: false,
		},
		{
			name:    "JPEG file",
			path:    "test.jpeg",
			want:    "image/jpeg",
			wantErr: false,
		},
		{
			name:    "GIF file",
			path:    "test.gif",
			want:    "image/gif",
			wantErr: false,
		},
		{
			name:    "WEBP file",
			path:    "test.webp",
			want:    "image/webp",
			wantErr: false,
		},
		{
			name:    "uppercase extension",
			path:    "test.PNG",
			want:    "image/png",
			wantErr: false,
		},
		{
			name:    "unsupported format",
			path:    "test.bmp",
			want:    "",
			wantErr: true,
		},
		{
			name:    "no extension",
			path:    "testfile",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getImageMediaType(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("getImageMediaType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getImageMediaType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatImageSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "kilobytes",
			bytes: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "megabytes",
			bytes: 1048576, // 1MB
			want:  "1.0 MB",
		},
		{
			name:  "large megabytes",
			bytes: 5242880, // 5MB
			want:  "5.0 MB",
		},
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatImageSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatImageSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadImage_FileNotFound(t *testing.T) {
	_, err := LoadImage("/nonexistent/image.png")
	if err == nil {
		t.Error("LoadImage() should return error for non-existent file")
	}
}

func TestLoadImage_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadImage(tmpDir)
	if err == nil {
		t.Error("LoadImage() should return error for directory")
	}
}

func TestLoadImage_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.png")

	// 10MB + 1バイトのファイルを作成
	largeData := make([]byte, MaxImageSize+1)
	if err := os.WriteFile(tmpFile, largeData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := LoadImage(tmpFile)
	if err == nil {
		t.Error("LoadImage() should return error for file larger than MaxImageSize")
	}
}

func TestLoadImage_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.bmp")

	if err := os.WriteFile(tmpFile, []byte("fake bmp data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := LoadImage(tmpFile)
	if err == nil {
		t.Error("LoadImage() should return error for unsupported format")
	}
}

func TestLoadImage_ValidPNG(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	// 小さいPNGデータ（実際のPNGヘッダーではないがテスト用）
	testData := []byte("fake png data for testing")
	if err := os.WriteFile(tmpFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	img, err := LoadImage(tmpFile)
	if err != nil {
		t.Fatalf("LoadImage() failed: %v", err)
	}

	if img.Path != tmpFile {
		t.Errorf("LoadImage() Path = %v, want %v", img.Path, tmpFile)
	}

	if img.MediaType != "image/png" {
		t.Errorf("LoadImage() MediaType = %v, want 'image/png'", img.MediaType)
	}

	if img.Size != int64(len(testData)) {
		t.Errorf("LoadImage() Size = %v, want %v", img.Size, len(testData))
	}

	if img.Base64 == "" {
		t.Error("LoadImage() Base64 should not be empty")
	}
}

func TestLoadImage_ValidJPEG(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jpg")

	testData := []byte("fake jpeg data")
	if err := os.WriteFile(tmpFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	img, err := LoadImage(tmpFile)
	if err != nil {
		t.Fatalf("LoadImage() failed: %v", err)
	}

	if img.MediaType != "image/jpeg" {
		t.Errorf("LoadImage() MediaType = %v, want 'image/jpeg'", img.MediaType)
	}
}
