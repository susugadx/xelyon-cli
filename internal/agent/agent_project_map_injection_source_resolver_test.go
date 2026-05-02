package agent

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveProjectMapSourceRootPath_UsesBundleRoot(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))
	rootPath := filepath.Join(cwd, "nested")

	got := resolveProjectMapSourceRootPath(cwd, &config.ProjectInstructionBundle{RootPath: rootPath})
	want := rootPath
	if got != want {
		t.Fatalf("resolveProjectMapSourceRootPath() = %q, want %q", got, want)
	}
}

func TestResolveProjectMapSourceRootPath_FallsBackToCWD(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))

	tests := []struct {
		name string
		b    *config.ProjectInstructionBundle
	}{
		{name: "nil bundle", b: nil},
		{name: "blank root path", b: &config.ProjectInstructionBundle{RootPath: "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveProjectMapSourceRootPath(cwd, tt.b)
			if got != cwd {
				t.Fatalf("resolveProjectMapSourceRootPath() = %q, want %q", got, cwd)
			}
		})
	}
}
