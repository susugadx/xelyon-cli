package directquery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestPlan_ScopedExactFilenameLookup(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "dep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.go"), []byte("package main\nconst selected = \"root\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "target.go"), []byte("package pkg\nconst selected = \"subdir\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "impl.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "impl_test.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{\"name\":\"root-app\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "dep", "package.json"), []byte("{\"name\":\"dep\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}
	policy := Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	}

	scoped := Plan(execCtx, "target.go", policy)
	if scoped.Kind != OutcomeResolved {
		t.Fatalf("expected scoped exact filename lookup to resolve, got %q (%s)", scoped.Kind, scoped.Error)
	}
	if scoped.Route.Kind != RouteRead {
		t.Fatalf("scoped.Route.Kind = %q, want %q", scoped.Route.Kind, RouteRead)
	}
	if got := scoped.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("pkg", "target.go") {
		t.Fatalf("unexpected scoped route entries: %+v", got)
	}

	ranged := Plan(execCtx, "target.go:2-2", policy)
	if ranged.Kind != OutcomeResolved {
		t.Fatalf("expected scoped ranged filename lookup to resolve, got %q (%s)", ranged.Kind, ranged.Error)
	}
	if got := ranged.Route.RawEntries(); len(got) != 1 || got[0] != "target.go:2-2" {
		t.Fatalf("unexpected scoped ranged route entries: %+v", got)
	}

	explicitRelative := Plan(execCtx, "./target.go", policy)
	if explicitRelative.Kind != OutcomeResolved {
		t.Fatalf("expected scoped explicit-relative filename lookup to resolve, got %q (%s)", explicitRelative.Kind, explicitRelative.Error)
	}
	if got := explicitRelative.Route.RawEntries(); len(got) != 1 || got[0] != "target.go" {
		t.Fatalf("unexpected scoped explicit-relative route entries: %+v", got)
	}

	explicitRelativeRange := Plan(execCtx, "./target.go:2-2", policy)
	if explicitRelativeRange.Kind != OutcomeResolved {
		t.Fatalf("expected scoped explicit-relative ranged lookup to resolve, got %q (%s)", explicitRelativeRange.Kind, explicitRelativeRange.Error)
	}
	if got := explicitRelativeRange.Route.RawEntries(); len(got) != 1 || got[0] != "target.go:2-2" {
		t.Fatalf("unexpected scoped explicit-relative ranged entries: %+v", got)
	}

	batch := Plan(execCtx, "impl.go,impl_test.go", policy)
	if batch.Kind != OutcomeResolved {
		t.Fatalf("expected scoped exact batch lookup to resolve, got %q (%s)", batch.Kind, batch.Error)
	}
	if got := batch.Route.RawEntries(); len(got) != 2 || got[0] != filepath.Join("pkg", "impl.go") || got[1] != filepath.Join("pkg", "impl_test.go") {
		t.Fatalf("unexpected scoped batch route entries: %+v", got)
	}

	ignored := Plan(execCtx, "package.json", Policy{
		ScopedPath: ".",
		FileFilter: "json",
	})
	if ignored.Kind != OutcomeResolved {
		t.Fatalf("expected ignored-tree exact lookup to resolve, got %q (%s)", ignored.Kind, ignored.Error)
	}
	if got := ignored.Route.RawEntries(); len(got) != 1 || got[0] != "package.json" {
		t.Fatalf("unexpected ignored-tree route entries: %+v", got)
	}

	explicitIgnored := Plan(execCtx, filepath.Join("node_modules", "dep", "package.json"), Policy{
		FileFilter: "json",
	})
	if explicitIgnored.Kind != OutcomeResolved {
		t.Fatalf("expected exact ignored-tree file lookup to resolve, got %q (%s)", explicitIgnored.Kind, explicitIgnored.Error)
	}
	if got := explicitIgnored.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("node_modules", "dep", "package.json") {
		t.Fatalf("unexpected exact ignored-tree route entries: %+v", got)
	}

	missing := Plan(execCtx, "missing.go", policy)
	if missing.Kind != OutcomeError {
		t.Fatalf("missing.Kind = %q, want %q", missing.Kind, OutcomeError)
	}
	if !strings.Contains(missing.Error, "direct path not found: missing.go") {
		t.Fatalf("expected scoped missing exact filename error, got: %q", missing.Error)
	}

	missingRange := Plan(execCtx, "missing.go:1-2", policy)
	if missingRange.Kind != OutcomeError {
		t.Fatalf("missingRange.Kind = %q, want %q", missingRange.Kind, OutcomeError)
	}
	if !strings.Contains(missingRange.Error, "direct path not found: missing.go:1-2") {
		t.Fatalf("expected scoped missing ranged filename error, got: %q", missingRange.Error)
	}

	missingBatch := Plan(execCtx, "impl.go,missing_test.go", policy)
	if missingBatch.Kind != OutcomeError {
		t.Fatalf("missingBatch.Kind = %q, want %q", missingBatch.Kind, OutcomeError)
	}
	if !strings.Contains(missingBatch.Error, "direct path not found: missing_test.go") {
		t.Fatalf("expected scoped missing batch error, got: %q", missingBatch.Error)
	}

	if err := os.MkdirAll(filepath.Join(pkgDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "nested", "target.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ambiguous := Plan(execCtx, "target.go", policy)
	if ambiguous.Kind != OutcomeError {
		t.Fatalf("ambiguous.Kind = %q, want %q", ambiguous.Kind, OutcomeError)
	}
	if !strings.Contains(ambiguous.Error, "direct path is ambiguous: target.go") {
		t.Fatalf("expected scoped ambiguous exact filename error, got: %q", ambiguous.Error)
	}
}

func TestPlan_ScopedRelativeTargets(t *testing.T) {
	root := t.TempDir()
	nestedDir := filepath.Join(root, "pkg", "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "impl.go"), []byte("package nested\nconst selected = \"nested\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "README.md"), []byte("nested docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	filePolicy := Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	}
	file := Plan(execCtx, "nested/impl.go", filePolicy)
	if file.Kind != OutcomeResolved {
		t.Fatalf("expected scoped relative file query to resolve, got %q (%s)", file.Kind, file.Error)
	}
	if file.Route.Kind != RouteRead {
		t.Fatalf("file.Route.Kind = %q, want %q", file.Route.Kind, RouteRead)
	}
	if got := file.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("pkg", "nested", "impl.go") {
		t.Fatalf("unexpected scoped relative file entries: %+v", got)
	}

	ranged := Plan(execCtx, "nested/impl.go:2-2", filePolicy)
	if ranged.Kind != OutcomeError {
		t.Fatalf("expected explicit ranged relative query to stay direct and error outside invocation cwd, got %q (%s)", ranged.Kind, ranged.Error)
	}
	if !strings.Contains(ranged.Error, "direct path not found: nested/impl.go:2-2") {
		t.Fatalf("unexpected scoped relative ranged error: %q", ranged.Error)
	}

	dir := Plan(execCtx, "nested/", Policy{ScopedPath: "pkg", FileFilter: "go"})
	if dir.Kind != OutcomeError {
		t.Fatalf("expected explicit directory marker query to stay direct and error outside invocation cwd, got %q (%s)", dir.Kind, dir.Error)
	}
	if !strings.Contains(dir.Error, "direct path not found: nested/") {
		t.Fatalf("unexpected scoped relative directory error: %q", dir.Error)
	}
}

func TestPlan_ScopedSoftAndExplicitDirectFileFilterContract(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	docsDir := filepath.Join(pkgDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(root, "README.md"):    "# root\n",
		filepath.Join(pkgDir, "README.md"):  "# pkg\n",
		filepath.Join(pkgDir, "impl.go"):    "package pkg\n",
		filepath.Join(docsDir, "README.md"): "# docs\n",
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

	softReadme := Plan(execCtx, "README.md", Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	})
	if softReadme.Kind != OutcomeNone {
		t.Fatalf("expected soft bare basename to fall back when filter mismatches, got %q (%s)", softReadme.Kind, softReadme.Error)
	}

	readme := Plan(execCtx, "./README.md", Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	})
	if readme.Kind != OutcomeResolved {
		t.Fatalf("expected exact README read to ignore stale filter, got %q (%s)", readme.Kind, readme.Error)
	}
	if got := readme.Route.RawEntries(); len(got) != 1 || got[0] != "README.md" {
		t.Fatalf("unexpected README route entries: %+v", got)
	}

	readmeRange := Plan(execCtx, "./README.md:1-1", Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	})
	if readmeRange.Kind != OutcomeResolved {
		t.Fatalf("expected exact README range read to ignore stale filter, got %q (%s)", readmeRange.Kind, readmeRange.Error)
	}
	if got := readmeRange.Route.RawEntries(); len(got) != 1 || got[0] != "README.md:1-1" {
		t.Fatalf("unexpected README range route entries: %+v", got)
	}

	impl := Plan(execCtx, "impl.go", Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	})
	if impl.Kind != OutcomeResolved {
		t.Fatalf("expected soft basename to resolve when filter matches, got %q (%s)", impl.Kind, impl.Error)
	}
	if got := impl.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("pkg", "impl.go") {
		t.Fatalf("unexpected impl route entries: %+v", got)
	}

	batch := Plan(execCtx, "./README.md,impl.go", Policy{
		ScopedPath: "pkg",
		FileFilter: "go",
	})
	if batch.Kind != OutcomeResolved {
		t.Fatalf("expected exact batch to ignore stale filter, got %q (%s)", batch.Kind, batch.Error)
	}
	if got := batch.Route.RawEntries(); len(got) != 2 || got[0] != filepath.Join("pkg", "README.md") || got[1] != filepath.Join("pkg", "impl.go") {
		t.Fatalf("unexpected exact batch route entries: %+v", got)
	}

	ambiguous := Plan(execCtx, "README.md", Policy{
		ScopedPath: "pkg",
		FileFilter: "pkg/docs/*.md",
	})
	if ambiguous.Kind != OutcomeResolved {
		t.Fatalf("expected ambiguous README query to use file_filter for disambiguation, got %q (%s)", ambiguous.Kind, ambiguous.Error)
	}
	if got := ambiguous.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("pkg", "docs", "README.md") {
		t.Fatalf("unexpected filtered ambiguous route entries: %+v", got)
	}
}

func TestPlan_ExplicitVsSoftDirectContractMatrix(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg", "nested")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles := map[string]string{
		filepath.Join(root, "README.md"):        "# root\n",
		filepath.Join(root, "pkg", "README.md"): "# pkg\n",
	}
	for path, body := range writeFiles {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rootExecCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}
	nestedExecCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}

	tests := []struct {
		name       string
		execCtx    tools.ExecutionContext
		query      string
		policy     Policy
		wantKind   OutcomeKind
		wantRoute  RouteKind
		wantReads  []string
		wantErrSub string
	}{
		{
			name:      "explicit exact file stays direct with matching filter",
			execCtx:   rootExecCtx,
			query:     "./README.md",
			policy:    Policy{FileFilter: "md"},
			wantKind:  OutcomeResolved,
			wantRoute: RouteRead,
			wantReads: []string{"README.md"},
		},
		{
			name:      "explicit exact file stays direct with stale filter",
			execCtx:   rootExecCtx,
			query:     "./README.md",
			policy:    Policy{FileFilter: "go"},
			wantKind:  OutcomeResolved,
			wantRoute: RouteRead,
			wantReads: []string{"README.md"},
		},
		{
			name:     "soft basename with stale scoped filter falls back",
			execCtx:  rootExecCtx,
			query:    "README.md",
			policy:   Policy{ScopedPath: "pkg", FileFilter: "go"},
			wantKind: OutcomeNone,
		},
		{
			name:      "soft basename with matching scoped filter resolves scoped direct",
			execCtx:   rootExecCtx,
			query:     "README.md",
			policy:    Policy{ScopedPath: "pkg", FileFilter: "md"},
			wantKind:  OutcomeResolved,
			wantRoute: RouteRead,
			wantReads: []string{filepath.Join("pkg", "README.md")},
		},
		{
			name:      "parent relative in repo stays direct",
			execCtx:   nestedExecCtx,
			query:     "../README.md",
			policy:    Policy{FileFilter: "go"},
			wantKind:  OutcomeResolved,
			wantRoute: RouteRead,
			wantReads: []string{filepath.Join("..", "README.md")},
		},
		{
			name:       "escaping parent relative errors",
			execCtx:    nestedExecCtx,
			query:      "../../outside.txt",
			policy:     Policy{FileFilter: "go"},
			wantKind:   OutcomeError,
			wantErrSub: "outside.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome := Plan(tt.execCtx, tt.query, tt.policy)
			if outcome.Kind != tt.wantKind {
				t.Fatalf("outcome.Kind = %q, want %q (%s)", outcome.Kind, tt.wantKind, outcome.Error)
			}
			if tt.wantKind == OutcomeResolved {
				if outcome.Route.Kind != tt.wantRoute {
					t.Fatalf("outcome.Route.Kind = %q, want %q", outcome.Route.Kind, tt.wantRoute)
				}
				if got := outcome.Route.RawEntries(); len(got) != len(tt.wantReads) {
					t.Fatalf("len(outcome.Route.RawEntries()) = %d, want %d", len(got), len(tt.wantReads))
				} else {
					for i := range got {
						if got[i] != tt.wantReads[i] {
							t.Fatalf("outcome.Route.RawEntries()[%d] = %q, want %q", i, got[i], tt.wantReads[i])
						}
					}
				}
			}
			if tt.wantKind == OutcomeError && !strings.Contains(outcome.Error, tt.wantErrSub) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErrSub, outcome.Error)
			}
		})
	}
}
