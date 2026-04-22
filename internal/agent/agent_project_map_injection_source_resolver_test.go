package agent

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveProjectMapSourceRootPath_UsesProjectConfigDir(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))
	configPath := filepath.Join(cwd, "nested", "xelyon.yaml")

	got := resolveProjectMapSourceRootPath(cwd, &config.ProjectConfig{FilePath: configPath})
	want := filepath.Dir(configPath)
	if got != want {
		t.Fatalf("resolveProjectMapSourceRootPath() = %q, want %q", got, want)
	}
}

func TestResolveProjectMapSourceRootPath_FallsBackToCWD(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))

	tests := []struct {
		name string
		pc   *config.ProjectConfig
	}{
		{name: "nil config", pc: nil},
		{name: "blank filepath", pc: &config.ProjectConfig{FilePath: "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveProjectMapSourceRootPath(cwd, tt.pc)
			if got != cwd {
				t.Fatalf("resolveProjectMapSourceRootPath() = %q, want %q", got, cwd)
			}
		})
	}
}
