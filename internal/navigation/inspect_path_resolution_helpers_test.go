package navigation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCandidateAbsPath_UsesAbsoluteRootAndCWD(t *testing.T) {
	rootDir := t.TempDir()
	rootFile := filepath.Join(rootDir, "pkg", "root.go")
	if err := os.MkdirAll(filepath.Dir(rootFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwdDir := t.TempDir()
	cwdFile := filepath.Join(cwdDir, "cwd.go")
	if err := os.WriteFile(cwdFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwdDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	tests := []struct {
		name string
		cand SymbolCandidate
		want string
	}{
		{
			name: "empty file returns empty path",
			cand: SymbolCandidate{},
			want: "",
		},
		{
			name: "absolute file is cleaned",
			cand: SymbolCandidate{File: rootFile},
			want: filepath.Clean(rootFile),
		},
		{
			name: "root path resolves relative file",
			cand: SymbolCandidate{File: "pkg/root.go", RootPath: rootDir},
			want: rootFile,
		},
		{
			name: "cwd resolves relative file without root",
			cand: SymbolCandidate{File: "cwd.go"},
			want: cwdFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := candidateAbsPath(tt.cand); got != tt.want {
				t.Fatalf("candidateAbsPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreferredInspectRootPath_ChoosesBroaderKnownRoot(t *testing.T) {
	parent := t.TempDir()
	projectRoot := filepath.Join(parent, "repo")
	symbolRoot := filepath.Join(projectRoot, "pkg")
	unrelated := filepath.Join(parent, "other")
	if err := os.MkdirAll(symbolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		symbolRoot  string
		projectRoot string
		want        string
	}{
		{"prefer project when symbol root empty", "", projectRoot, projectRoot},
		{"prefer symbol when project root empty", symbolRoot, "", symbolRoot},
		{"prefer nested symbol root", symbolRoot, projectRoot, symbolRoot},
		{"prefer nested project root", projectRoot, symbolRoot, symbolRoot},
		{"prefer project on unrelated roots", unrelated, projectRoot, projectRoot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := preferredInspectRootPath(tt.symbolRoot, tt.projectRoot); got != tt.want {
				t.Fatalf("preferredInspectRootPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeResultFilePath_RecoversSnapshotRelativePaths(t *testing.T) {
	rootDir := t.TempDir()
	targetRoot := filepath.Join(rootDir, "repo")
	sourceBase := filepath.Join(targetRoot, "pkg")
	absFile := filepath.Join(targetRoot, "pkg", "run.go")
	if err := os.MkdirAll(filepath.Dir(absFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(targetRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	tests := []struct {
		name       string
		path       string
		targetRoot string
		sourceBase string
		want       string
	}{
		{"absolute path becomes relative", absFile, targetRoot, sourceBase, "pkg/run.go"},
		{"root relative existing path stays relative", "pkg/run.go", targetRoot, sourceBase, "pkg/run.go"},
		{"source base relative file is recovered", "run.go", targetRoot, sourceBase, "pkg/run.go"},
		{"cwd relative file is recovered", filepath.ToSlash(filepath.Join("pkg", "run.go")), targetRoot, "", "pkg/run.go"},
		{"source base fallback keeps nested relative path", "pkg/../missing.go", targetRoot, sourceBase, "pkg/missing.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeResultFilePath(tt.path, tt.targetRoot, tt.sourceBase); got != tt.want {
				t.Fatalf("normalizeResultFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
