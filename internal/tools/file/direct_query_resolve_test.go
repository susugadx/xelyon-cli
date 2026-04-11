package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveDirectQueryTarget_PrefersInvocationCWD(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "target.go")
	subTarget := filepath.Join(subdir, "target.go")
	if err := os.WriteFile(rootTarget, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subTarget, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	targetInput, ok := parseDirectQueryEntryInput("target.go")
	if !ok {
		t.Fatal("expected direct query input to parse")
	}
	target, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, targetInput)
	if errResult != "" {
		t.Fatalf("expected direct query target to resolve, got %q", errResult)
	}
	if target.ResolvedPath != subTarget {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, subTarget)
	}
}

func TestResolveDirectQueryTarget_ExplicitRelativeUsesCWDButAllowsInRepoParent(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "Makefile")
	if err := os.WriteFile(rootTarget, []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryEntryInput("./Makefile")
	if !ok {
		t.Fatal("expected explicit relative query to parse")
	}
	if _, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, input); errResult == "" {
		t.Fatal("expected explicit relative query to stay within invocation cwd")
	}

	subTarget := filepath.Join(subdir, "Makefile")
	if err := os.WriteFile(subTarget, []byte("all:\n\tgo test ./...\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	target, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, input)
	if errResult != "" {
		t.Fatalf("expected explicit relative query to resolve once file exists in invocation cwd, got %q", errResult)
	}
	if target.ResolvedPath != subTarget {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, subTarget)
	}

	parentInput, ok := parseDirectQueryEntryInput("../Makefile")
	if !ok {
		t.Fatal("expected parent-relative query to parse")
	}
	parentTarget, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, parentInput)
	if errResult != "" {
		t.Fatalf("expected in-repo parent-relative query to resolve, got %q", errResult)
	}
	if parentTarget.ResolvedPath != rootTarget {
		t.Fatalf("parentTarget.ResolvedPath = %q, want %q", parentTarget.ResolvedPath, rootTarget)
	}

	parentDirInput, ok := parseDirectQueryEntryInput("../pkg/")
	if !ok {
		t.Fatal("expected parent-relative directory query to parse")
	}
	parentDirTarget, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, parentDirInput)
	if errResult != "" {
		t.Fatalf("expected in-repo parent-relative directory query to resolve, got %q", errResult)
	}
	if parentDirTarget.Kind != DirectQueryTargetDirectory || parentDirTarget.ResolvedPath != subdir {
		t.Fatalf("unexpected parent-relative directory target: %+v", parentDirTarget)
	}

	outsideRoot := t.TempDir()
	outsideInput, ok := parseDirectQueryEntryInput("../../outside.txt")
	if !ok {
		t.Fatal("expected outside parent-relative query to parse")
	}
	if err := os.WriteFile(filepath.Join(outsideRoot, "outside.txt"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, outsideInput); errResult == "" {
		t.Fatal("expected parent-relative path escaping repo roots to be rejected")
	}
}

func TestResolveDirectQuery_MultiFileAndDirectoryClassification(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	files, errResult := resolveDirectQuery(execCtx, "a.go,b.go:1")
	if errResult != "" {
		t.Fatalf("expected multi-file direct query to resolve, got %q", errResult)
	}
	if files.Kind != DirectQueryResolutionFiles {
		t.Fatalf("files.Kind = %q, want %q", files.Kind, DirectQueryResolutionFiles)
	}
	if len(files.Targets) != 2 {
		t.Fatalf("len(files.Targets) = %d, want 2", len(files.Targets))
	}

	dir, errResult := resolveDirectQuery(execCtx, "pkg")
	if errResult != "" {
		t.Fatalf("expected directory direct query to resolve, got %q", errResult)
	}
	if dir.Kind != DirectQueryResolutionDirectory {
		t.Fatalf("dir.Kind = %q, want %q", dir.Kind, DirectQueryResolutionDirectory)
	}

	if _, errResult := resolveDirectQuery(execCtx, "a.go,pkg"); errResult == "" {
		t.Fatal("expected mixed file+directory direct query to be rejected")
	}
}

func TestResolveImplicitDirectFileQuery(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n\tgo test ./...\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	targets, ok := resolveImplicitDirectFileQuery(execCtx, "Makefile")
	if !ok {
		t.Fatal("expected Makefile to resolve as implicit direct file query")
	}
	if len(targets) != 1 || targets[0].Kind != DirectQueryTargetFile {
		t.Fatalf("unexpected implicit targets: %+v", targets)
	}

	if _, ok := resolveImplicitDirectFileQuery(execCtx, "config"); ok {
		t.Fatal("expected directory name to stay out of implicit file route")
	}
	if _, ok := resolveImplicitDirectFileQuery(execCtx, "./Makefile"); ok {
		t.Fatal("expected explicit path-like query to be handled by explicit route")
	}
}

func TestResolveDirectReadTargets_BatchMissingEntryReturnsError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput("sample.go,missing.go")
	if !ok {
		t.Fatal("expected batch direct query to parse")
	}

	_, errResult := resolveDirectReadTargets(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, input)
	if errResult == "" {
		t.Fatal("expected strict direct-read resolution to report missing batch entry")
	}
	if errResult != "Error: direct path not found: missing.go" {
		t.Fatalf("errResult = %q, want missing-path direct error", errResult)
	}
}

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

func TestResolveDirectQueryInput_ExactIgnoredTreePathBypassesIgnores(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "dep"), 0o755); err != nil {
		t.Fatal(err)
	}
	depPackage := filepath.Join(root, "node_modules", "dep", "package.json")
	if err := os.WriteFile(depPackage, []byte("{\"name\":\"dep\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput(filepath.Join("node_modules", "dep", "package.json"))
	if !ok {
		t.Fatal("expected exact ignored-tree path query to parse")
	}

	resolution, errResult := resolveDirectQueryInput(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, input)
	if errResult != "" {
		t.Fatalf("expected exact ignored-tree path to resolve, got %q", errResult)
	}
	if resolution.Kind != DirectQueryResolutionFiles {
		t.Fatalf("resolution.Kind = %q, want %q", resolution.Kind, DirectQueryResolutionFiles)
	}
	if len(resolution.Targets) != 1 {
		t.Fatalf("len(resolution.Targets) = %d, want 1", len(resolution.Targets))
	}
	target := resolution.Targets[0]
	if target.ResolvedPath != depPackage {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, depPackage)
	}
	if target.RawEntry != filepath.Join("node_modules", "dep", "package.json") {
		t.Fatalf("target.RawEntry = %q, want dependency package.json entry", target.RawEntry)
	}
	if !target.BypassIgnores {
		t.Fatal("expected exact ignored-tree path target to bypass ignores")
	}
}

func TestResolveDirectQueryInput_ExactIgnoredTreeRangeBypassesIgnores(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "dep"), 0o755); err != nil {
		t.Fatal(err)
	}
	depPackage := filepath.Join(root, "node_modules", "dep", "package.json")
	if err := os.WriteFile(depPackage, []byte("{\n  \"name\": \"dep\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput(filepath.Join("node_modules", "dep", "package.json:2-2"))
	if !ok {
		t.Fatal("expected exact ignored-tree range query to parse")
	}

	resolution, errResult := resolveDirectQueryInput(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, input)
	if errResult != "" {
		t.Fatalf("expected exact ignored-tree range to resolve, got %q", errResult)
	}
	if resolution.Kind != DirectQueryResolutionFiles || len(resolution.Targets) != 1 {
		t.Fatalf("unexpected resolution: %+v", resolution)
	}
	target := resolution.Targets[0]
	if target.ResolvedPath != depPackage {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, depPackage)
	}
	if !target.BypassIgnores {
		t.Fatal("expected exact ignored-tree range target to bypass ignores")
	}
	if target.StartLine != 2 || target.EndLine != 2 {
		t.Fatalf("target range = %d-%d, want 2-2", target.StartLine, target.EndLine)
	}
}
