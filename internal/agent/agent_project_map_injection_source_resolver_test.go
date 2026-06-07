package agent

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveProjectMapSourceRootPath_UsesBundleRoot(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))
	rootPath := filepath.Join(cwd, "nested")

	got := resolveProjectMapSourceRootPath(cwd, &config.ProjectInstructionBundle{
		RootPath:   rootPath,
		RootSource: config.ProjectInstructionRootSourceGit,
	})
	want := rootPath
	if got != want {
		t.Fatalf("resolveProjectMapSourceRootPath() = %q, want %q", got, want)
	}
}

func TestResolveProjectMapSourceRootPath_SkipsWithoutProjectRoot(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))

	tests := []struct {
		name string
		b    *config.ProjectInstructionBundle
	}{
		{name: "nil bundle", b: nil},
		{name: "blank root path", b: &config.ProjectInstructionBundle{RootPath: "   "}},
		{name: "cwd fallback", b: &config.ProjectInstructionBundle{RootPath: cwd, RootSource: config.ProjectInstructionRootSourceFallbackCWD}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveProjectMapSourceRootPath(cwd, tt.b)
			if got != "" {
				t.Fatalf("resolveProjectMapSourceRootPath() = %q, want empty root", got)
			}
		})
	}
}

func TestResolveProjectMapSourceRootPathWithFallback_UsesCachedRootPath(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))
	fallbackRootPath := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace-root"))

	got := resolveProjectMapSourceRootPathWithFallback(cwd, nil, fallbackRootPath)
	if got != fallbackRootPath {
		t.Fatalf("resolveProjectMapSourceRootPathWithFallback() = %q, want %q", got, fallbackRootPath)
	}
}

func TestResolveProjectMapSourceRootPathWithFallback_PrefersBundleRootPath(t *testing.T) {
	cwd := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace"))
	bundleRootPath := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "project"))
	fallbackRootPath := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "workspace-root"))

	got := resolveProjectMapSourceRootPathWithFallback(cwd, &config.ProjectInstructionBundle{
		RootPath:   bundleRootPath,
		RootSource: config.ProjectInstructionRootSourceProjectConfig,
	}, fallbackRootPath)
	if got != bundleRootPath {
		t.Fatalf("resolveProjectMapSourceRootPathWithFallback() = %q, want %q", got, bundleRootPath)
	}
}
