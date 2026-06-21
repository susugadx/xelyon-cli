package listtool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestExecuteListDir_IgnoreDefaultDirs(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	if err := os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("Failed to create ignored directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := ExecuteListDir(tmpDir, 2)
	if strings.Contains(output, "node_modules") {
		t.Errorf("node_modules should be ignored, got: %s", output)
	}
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be listed, got: %s", output)
	}
}

func TestExecuteListDir_CustomIgnoreDirs(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.Mkdir(filepath.Join(tmpDir, "coverage"), 0755); err != nil {
		t.Fatalf("Failed to create coverage directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("Failed to create keep file: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ListDir.AdditionalIgnoreDirs = []string{"coverage"}

	output := ExecuteListDirWithRuntime(cfg, nil, tmpDir, 1)
	if strings.Contains(output, "coverage") {
		t.Errorf("coverage should be ignored by custom list_dir config, got: %s", output)
	}
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be listed, got: %s", output)
	}
}

func TestExecuteListDir_ProjectIgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - coverage\n"), 0644); err != nil {
		t.Fatalf("Failed to create xelyon.yaml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "coverage"), 0755); err != nil {
		t.Fatalf("Failed to create coverage directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("Failed to create keep file: %v", err)
	}

	output := ExecuteListDirWithRuntime(config.DefaultConfig(), nil, tmpDir, 1)
	if strings.Contains(output, "coverage") {
		t.Errorf("coverage should be ignored by xelyon.yaml ignore.patterns, got: %s", output)
	}
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be listed, got: %s", output)
	}
}
