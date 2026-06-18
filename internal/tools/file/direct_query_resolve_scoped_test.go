package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveScopedGatherContextDirectResolution_RelativeScopeUsesInvocationCWD(t *testing.T) {
	tests := []struct {
		query          string
		wantKind       DirectQueryResolutionKind
		wantResolved   []string
		wantRawEntries []string
		setup          func(root, invocationCWD string) error
	}{
		{
			query:          "target.go",
			wantKind:       DirectQueryResolutionFiles,
			wantResolved:   []string{filepath.Join("cmd", "tool", "pkg", "target.go")},
			wantRawEntries: []string{filepath.Join("pkg", "target.go")},
			setup: func(root, invocationCWD string) error {
				files := map[string]string{
					filepath.Join(root, "pkg", "target.go"):          "package rootpkg\nconst selected = \"root\"\n",
					filepath.Join(invocationCWD, "pkg", "target.go"): "package cwdpkg\nconst selected = \"cwd\"\n",
				}
				for path, body := range files {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			query:          "target.go:2-2",
			wantKind:       DirectQueryResolutionFiles,
			wantResolved:   []string{filepath.Join("cmd", "tool", "pkg", "target.go")},
			wantRawEntries: []string{filepath.Join("pkg", "target.go:2-2")},
			setup: func(root, invocationCWD string) error {
				files := map[string]string{
					filepath.Join(root, "pkg", "target.go"):          "package rootpkg\nconst selected = \"root\"\n",
					filepath.Join(invocationCWD, "pkg", "target.go"): "package cwdpkg\nconst selected = \"cwd\"\n",
				}
				for path, body := range files {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			query:          "./target.go",
			wantKind:       DirectQueryResolutionFiles,
			wantResolved:   []string{filepath.Join("cmd", "tool", "pkg", "target.go")},
			wantRawEntries: []string{filepath.Join("pkg", "target.go")},
			setup: func(root, invocationCWD string) error {
				files := map[string]string{
					filepath.Join(root, "target.go"):                 "package rootpkg\nconst selected = \"root-shadow\"\n",
					filepath.Join(root, "pkg", "target.go"):          "package rootpkg\nconst selected = \"root\"\n",
					filepath.Join(invocationCWD, "pkg", "target.go"): "package cwdpkg\nconst selected = \"cwd\"\n",
				}
				for path, body := range files {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			query:          "./target.go:2-2",
			wantKind:       DirectQueryResolutionFiles,
			wantResolved:   []string{filepath.Join("cmd", "tool", "pkg", "target.go")},
			wantRawEntries: []string{filepath.Join("pkg", "target.go:2-2")},
			setup: func(root, invocationCWD string) error {
				files := map[string]string{
					filepath.Join(root, "target.go"):                 "package rootpkg\nconst selected = \"root-shadow\"\n",
					filepath.Join(root, "pkg", "target.go"):          "package rootpkg\nconst selected = \"root\"\n",
					filepath.Join(invocationCWD, "pkg", "target.go"): "package cwdpkg\nconst selected = \"cwd\"\n",
				}
				for path, body := range files {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			query:          "impl.go,impl_test.go",
			wantKind:       DirectQueryResolutionFiles,
			wantResolved:   []string{filepath.Join("cmd", "tool", "pkg", "impl.go"), filepath.Join("cmd", "tool", "pkg", "impl_test.go")},
			wantRawEntries: []string{filepath.Join("pkg", "impl.go"), filepath.Join("pkg", "impl_test.go")},
			setup: func(root, invocationCWD string) error {
				files := map[string]string{
					filepath.Join(root, "pkg", "impl.go"):               "package rootpkg\nconst impl = \"root\"\n",
					filepath.Join(root, "pkg", "impl_test.go"):          "package rootpkg\nconst implTest = \"root\"\n",
					filepath.Join(invocationCWD, "pkg", "impl.go"):      "package cwdpkg\nconst impl = \"cwd\"\n",
					filepath.Join(invocationCWD, "pkg", "impl_test.go"): "package cwdpkg\nconst implTest = \"cwd\"\n",
				}
				for path, body := range files {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			query:          "nested/impl.go",
			wantKind:       DirectQueryResolutionFiles,
			wantResolved:   []string{filepath.Join("cmd", "tool", "pkg", "nested", "impl.go")},
			wantRawEntries: []string{filepath.Join("pkg", "nested", "impl.go")},
			setup: func(root, invocationCWD string) error {
				files := map[string]string{
					filepath.Join(root, "pkg", "nested", "impl.go"):          "package rootnested\nconst nested = \"root\"\n",
					filepath.Join(invocationCWD, "pkg", "nested", "impl.go"): "package cwdnested\nconst nested = \"cwd\"\n",
				}
				for path, body := range files {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			root := t.TempDir()
			invocationCWD := filepath.Join(root, "cmd", "tool")
			if err := tt.setup(root, invocationCWD); err != nil {
				t.Fatal(err)
			}
			input, ok := parseDirectQueryInput(tt.query)
			if !ok {
				t.Fatalf("expected %q to parse as direct query input", tt.query)
			}
			outcome := resolveScopedGatherContextDirectResolution(tools.ExecutionContext{
				ProjectMapRootPath: root,
				InvocationCWD:      invocationCWD,
			}, input, GatherContextDirectRoutePolicy{
				ScopedPath: "pkg",
				FileFilter: "go",
			})
			if outcome.Kind != scopedDirectResolutionResolved {
				t.Fatalf("expected %q to resolve within invocation cwd scoped path", tt.query)
			}
			resolution := outcome.Resolution
			if resolution.Kind != tt.wantKind {
				t.Fatalf("resolution.Kind = %q, want %q", resolution.Kind, tt.wantKind)
			}
			if len(resolution.Targets) != len(tt.wantResolved) {
				t.Fatalf("len(resolution.Targets) = %d, want %d", len(resolution.Targets), len(tt.wantResolved))
			}
			for i, target := range resolution.Targets {
				wantResolved := filepath.Join(root, tt.wantResolved[i])
				if target.ResolvedPath != wantResolved {
					t.Fatalf("target[%d].ResolvedPath = %q, want %q", i, target.ResolvedPath, wantResolved)
				}
				if target.RawEntry != tt.wantRawEntries[i] {
					t.Fatalf("target[%d].RawEntry = %q, want %q", i, target.RawEntry, tt.wantRawEntries[i])
				}
			}
		})
	}
}

func TestResolveScopedGatherContextDirectResolution_ExplicitRelativeWithoutScopeUsesInvocationCWD(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput("./Makefile")
	if !ok {
		t.Fatal("expected explicit relative query to parse as direct query input")
	}

	outcome := resolveScopedGatherContextDirectResolution(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, input, GatherContextDirectRoutePolicy{FileFilter: "go"})
	if outcome.Kind != scopedDirectResolutionMissing {
		t.Fatalf("expected explicit relative query to stay scoped to invocation cwd, got %q (%s)", outcome.Kind, outcome.Error)
	}
	if outcome.Error != "Error: direct path not found: ./Makefile" {
		t.Fatalf("outcome.Error = %q, want missing explicit relative path error", outcome.Error)
	}

	subTarget := filepath.Join(subdir, "Makefile")
	if err := os.WriteFile(subTarget, []byte("all:\n\tgo test ./...\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outcome = resolveScopedGatherContextDirectResolution(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, input, GatherContextDirectRoutePolicy{FileFilter: "go"})
	if outcome.Kind != scopedDirectResolutionResolved {
		t.Fatalf("expected explicit relative query to resolve once invocation cwd file exists, got %q (%s)", outcome.Kind, outcome.Error)
	}
	if len(outcome.Resolution.Targets) != 1 {
		t.Fatalf("len(outcome.Resolution.Targets) = %d, want 1", len(outcome.Resolution.Targets))
	}
	if outcome.Resolution.Targets[0].ResolvedPath != subTarget {
		t.Fatalf("ResolvedPath = %q, want %q", outcome.Resolution.Targets[0].ResolvedPath, subTarget)
	}
}

func TestResolveScopedGatherContextDirectResolution_SoftBasenameHonorsFileFilter(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	docsDir := filepath.Join(pkgDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(pkgDir, "README.md"):  "# pkg\n",
		filepath.Join(pkgDir, "impl.go"):    "package pkg\nconst impl = true\n",
		filepath.Join(docsDir, "README.md"): "# docs\n",
		filepath.Join(docsDir, "worker.go"): "package docs\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	tests := []struct {
		name           string
		query          string
		fileFilter     string
		wantKind       scopedDirectResolutionKind
		wantResolved   []string
		wantRawEntries []string
	}{
		{
			name:       "soft bare basename filtered out by stale filter",
			query:      "README.md",
			fileFilter: "go",
			wantKind:   scopedDirectResolutionFiltered,
		},
		{
			name:           "single explicit file ignores stale filter",
			query:          "./README.md",
			fileFilter:     "go",
			wantKind:       scopedDirectResolutionResolved,
			wantResolved:   []string{filepath.Join(pkgDir, "README.md")},
			wantRawEntries: []string{filepath.Join("pkg", "README.md")},
		},
		{
			name:           "single explicit range ignores stale filter",
			query:          "./README.md:1-1",
			fileFilter:     "go",
			wantKind:       scopedDirectResolutionResolved,
			wantResolved:   []string{filepath.Join(pkgDir, "README.md")},
			wantRawEntries: []string{filepath.Join("pkg", "README.md:1-1")},
		},
		{
			name:           "mixed explicit batch preserves exact reads",
			query:          "./README.md,impl.go",
			fileFilter:     "go",
			wantKind:       scopedDirectResolutionResolved,
			wantResolved:   []string{filepath.Join(pkgDir, "README.md"), filepath.Join(pkgDir, "impl.go")},
			wantRawEntries: []string{filepath.Join("pkg", "README.md"), filepath.Join("pkg", "impl.go")},
		},
		{
			name:           "soft bare basename still resolves when filter matches",
			query:          "impl.go",
			fileFilter:     "go",
			wantKind:       scopedDirectResolutionResolved,
			wantResolved:   []string{filepath.Join(pkgDir, "impl.go")},
			wantRawEntries: []string{filepath.Join("pkg", "impl.go")},
		},
		{
			name:           "ambiguous bare filename still uses filter to choose exact target",
			query:          "README.md",
			fileFilter:     "pkg/docs/*.md",
			wantKind:       scopedDirectResolutionResolved,
			wantResolved:   []string{filepath.Join(docsDir, "README.md")},
			wantRawEntries: []string{filepath.Join("pkg", "docs", "README.md")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, ok := parseDirectQueryInput(tt.query)
			if !ok {
				t.Fatalf("expected %q to parse as direct query input", tt.query)
			}
			outcome := resolveScopedGatherContextDirectResolution(execCtx, input, GatherContextDirectRoutePolicy{
				ScopedPath: "pkg",
				FileFilter: tt.fileFilter,
			})
			if outcome.Kind != tt.wantKind {
				t.Fatalf("outcome.Kind = %q, want %q (%s)", outcome.Kind, tt.wantKind, outcome.Error)
			}
			if outcome.Kind != scopedDirectResolutionResolved {
				return
			}
			if len(outcome.Resolution.Targets) != len(tt.wantResolved) {
				t.Fatalf("len(outcome.Resolution.Targets) = %d, want %d", len(outcome.Resolution.Targets), len(tt.wantResolved))
			}
			for i, target := range outcome.Resolution.Targets {
				if target.ResolvedPath != tt.wantResolved[i] {
					t.Fatalf("target[%d].ResolvedPath = %q, want %q", i, target.ResolvedPath, tt.wantResolved[i])
				}
				if target.RawEntry != tt.wantRawEntries[i] {
					t.Fatalf("target[%d].RawEntry = %q, want %q", i, target.RawEntry, tt.wantRawEntries[i])
				}
			}
		})
	}
}
