package search

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func TestJSFamilyLSPPathsResolveRelativeLocationsFromAdapterAndWorkspaceRoots(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.ts": "export function buildUser(id: string) { return id }\n",
		"packages/other/src/app.ts": "buildUser('semantic workspace')\n",
	})
	subdir := filepath.Join(dir, "packages", "app")
	locationOpts := newJSFamilyLSPLocationOptions(SearchOptions{
		Path:               filepath.Join(dir, "packages", "app", "src", "build.ts"),
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
	})

	tests := []struct {
		name string
		file string
	}{
		{
			name: "workspace relative mock location",
			file: "packages/other/src/app.ts",
		},
		{
			name: "adapter relative real location",
			file: "../other/src/app.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			displayPath, absPath := jsFamilyLSPPaths(tt.file, locationOpts)

			if displayPath != "packages/other/src/app.ts" {
				t.Fatalf("displayPath = %q, want workspace-relative LSP path", displayPath)
			}
			wantAbsPath := filepath.Join(dir, "packages", "other", "src", "app.ts")
			if absPath != wantAbsPath {
				t.Fatalf("absPath = %q, want %q", absPath, wantAbsPath)
			}
		})
	}
}

func TestJSFamilyRefFromLSPLocationRejectsIgnoredPath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/generated.js": "buildUser('generated')\n",
	})
	opts := SearchOptions{
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
		ignoreMatcher: pathmatch.NewMatcher([]string{"src/generated.js"}),
	}
	lspOpts := newJSFamilyLSPReferenceOptions(opts, opts, opts)

	_, ok := jsFamilyRefFromLSPLocation("buildUser", navigation.LSPLocation{
		File:      "src/generated.js",
		Line:      1,
		Character: 1,
	}, lspOpts)

	if ok {
		t.Fatal("jsFamilyRefFromLSPLocation accepted ignored path, want reject")
	}
}
