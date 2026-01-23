package tools

import (
	"testing"
)

func TestGetMediaType(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "PNG file",
			path:    "image.png",
			want:    "image/png",
			wantErr: false,
		},
		{
			name:    "PNG uppercase",
			path:    "IMAGE.PNG",
			want:    "image/png",
			wantErr: false,
		},
		{
			name:    "JPG file",
			path:    "photo.jpg",
			want:    "image/jpeg",
			wantErr: false,
		},
		{
			name:    "JPEG file",
			path:    "photo.jpeg",
			want:    "image/jpeg",
			wantErr: false,
		},
		{
			name:    "GIF file",
			path:    "animation.gif",
			want:    "image/gif",
			wantErr: false,
		},
		{
			name:    "WebP file",
			path:    "modern.webp",
			want:    "image/webp",
			wantErr: false,
		},
		{
			name:    "path with directory",
			path:    "/home/user/images/photo.png",
			want:    "image/png",
			wantErr: false,
		},
		{
			name:    "unsupported format BMP",
			path:    "image.bmp",
			want:    "",
			wantErr: true,
		},
		{
			name:    "unsupported format TIFF",
			path:    "image.tiff",
			want:    "",
			wantErr: true,
		},
		{
			name:    "text file",
			path:    "document.txt",
			want:    "",
			wantErr: true,
		},
		{
			name:    "no extension",
			path:    "filename",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMediaType(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMediaType(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getMediaType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "0 bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "1 byte",
			bytes: 1,
			want:  "1 B",
		},
		{
			name:  "512 bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "1023 bytes",
			bytes: 1023,
			want:  "1023 B",
		},
		{
			name:  "1 KB (1024 bytes)",
			bytes: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "1.5 KB",
			bytes: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "1 MB",
			bytes: 1024 * 1024,
			want:  "1.0 MB",
		},
		{
			name:  "2.5 MB",
			bytes: int64(2.5 * 1024 * 1024),
			want:  "2.5 MB",
		},
		{
			name:  "1 GB",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GB",
		},
		{
			name:  "1 TB",
			bytes: 1024 * 1024 * 1024 * 1024,
			want:  "1.0 TB",
		},
		{
			name:  "typical image size 500KB",
			bytes: 512000,
			want:  "500.0 KB",
		},
		{
			name:  "typical photo 3.5MB",
			bytes: int64(3.5 * 1024 * 1024),
			want:  "3.5 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
