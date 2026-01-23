package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePathImpl(t *testing.T) {
	// カレントディレクトリを取得
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path is empty",
		},
		{
			name:    "valid relative path",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "valid subdirectory path",
			path:    "subdir/test.txt",
			wantErr: false,
		},
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "parent directory escape",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "path escape",
		},
		{
			name:    "absolute path outside cwd",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "path escape",
		},
		{
			name:    "valid absolute path in cwd",
			path:    filepath.Join(cwd, "test.txt"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePathImpl(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePathImpl(%q) expected error containing %q, got nil", tt.path, tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePathImpl(%q) error = %q, want error containing %q", tt.path, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePathImpl(%q) unexpected error: %v", tt.path, err)
					return
				}
				// 正常系では結果がcwd配下であることを確認
				if !strings.HasPrefix(result, cwd) {
					t.Errorf("validatePathImpl(%q) = %q, not under cwd %q", tt.path, result, cwd)
				}
			}
		})
	}
}

func TestValidatePathAllowParent(t *testing.T) {
	// テンポラリディレクトリを使用
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path is empty",
		},
		{
			name:    "blocked /etc",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "system directory is blocked",
		},
		{
			name:    "blocked /var",
			path:    "/var/log/syslog",
			wantErr: true,
			errMsg:  "system directory is blocked",
		},
		{
			name:    "blocked /usr",
			path:    "/usr/bin/bash",
			wantErr: true,
			errMsg:  "system directory is blocked",
		},
		{
			name:    "blocked /root",
			path:    "/root/.bashrc",
			wantErr: true,
			errMsg:  "system directory is blocked",
		},
		{
			name:    "valid temp directory",
			path:    filepath.Join(tmpDir, "test.txt"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePathAllowParent(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePathAllowParent(%q) expected error containing %q, got nil", tt.path, tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePathAllowParent(%q) error = %q, want error containing %q", tt.path, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePathAllowParent(%q) unexpected error: %v", tt.path, err)
					return
				}
				// 結果が空でないことを確認
				if result == "" {
					t.Errorf("ValidatePathAllowParent(%q) returned empty result", tt.path)
				}
			}
		})
	}
}
