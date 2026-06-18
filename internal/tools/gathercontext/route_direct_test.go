package gathercontext

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/file/directquery"
)

func TestPlanRoute_DirectContracts(t *testing.T) {
	root := setupRoutePlanFixtures(t)

	tests := []struct {
		name           string
		query          string
		path           string
		fileFilter     string
		wantKind       routeKind
		wantDirectKind directquery.RouteKind
		wantReads      []string
	}{
		{
			name:           "explicit range query uses exact range read route",
			query:          "sample.go:10-20",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{"sample.go:10-20"},
		},
		{
			name:       "scoped missing exact file query returns direct error",
			query:      "sample.go",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeDirectError,
		},
		{
			name:           "scoped explicit exact file ignores stale file filter",
			query:          "./README.md",
			path:           "pkg",
			fileFilter:     "go",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{"README.md"},
		},
		{
			name:           "scoped exact file query uses direct read route",
			query:          "target.go",
			path:           "pkg",
			fileFilter:     "go",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{filepath.Join("pkg", "target.go")},
		},
		{
			name:           "scoped explicit-relative file query uses direct read route",
			query:          "./target.go",
			path:           "pkg",
			fileFilter:     "go",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{"target.go"},
		},
		{
			name:           "scoped explicit ranged file query uses direct read route",
			query:          "target.go:2-2",
			path:           "pkg",
			fileFilter:     "go",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{"target.go:2-2"},
		},
		{
			name:           "scoped explicit-relative ranged file query uses direct read route",
			query:          "./target.go:2-2",
			path:           "pkg",
			fileFilter:     "go",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{"target.go:2-2"},
		},
		{
			name:           "scoped exact batch query uses direct read route",
			query:          "impl.go,impl_test.go",
			path:           "pkg",
			fileFilter:     "go",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{filepath.Join("pkg", "impl.go"), filepath.Join("pkg", "impl_test.go")},
		},
		{
			name:       "scoped missing ranged exact query returns direct error",
			query:      "missing.go:1-2",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeDirectError,
		},
		{
			name:       "scoped missing exact batch returns direct error",
			query:      "impl.go,missing_test.go",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeDirectError,
		},
		{
			name:       "explicit directory marker ignores scoped search policy and errors outside cwd",
			query:      "nested/",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeDirectError,
		},
		{
			name:           "explicit slash directory query uses list route",
			query:          filepath.Join("internal", "tools") + string(os.PathSeparator),
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteDirectory,
			wantReads:      []string{filepath.Join("internal", "tools")},
		},
		{
			name:           "explicit directory marker keeps directory route",
			query:          "config/",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteDirectory,
			wantReads:      []string{"config"},
		},
		{
			name:       "missing explicit path returns direct error",
			query:      "./missing.go",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeDirectError,
		},
		{
			name:       "scoped missing bare file batch returns direct error",
			query:      "sample.go,missing.go",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeDirectError,
		},
		{
			name:           "scoped ignored tree exact lookup stays direct",
			query:          "package.json",
			path:           ".",
			fileFilter:     "json",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{"package.json"},
		},
		{
			name:           "exact ignored subtree file lookup stays direct",
			query:          filepath.Join("node_modules", "dep", "package.json"),
			fileFilter:     "json",
			wantKind:       routeDirect,
			wantDirectKind: directquery.RouteRead,
			wantReads:      []string{filepath.Join("node_modules", "dep", "package.json")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildRoutePlan(newRoutePlanExecCtx(root), request{
				query:      tt.query,
				path:       tt.path,
				fileFilter: tt.fileFilter,
			})
			if plan.kind != tt.wantKind {
				t.Fatalf("plan.kind = %q, want %q", plan.kind, tt.wantKind)
			}
			if plan.kind == routeDirectError {
				if plan.direct.err == "" {
					t.Fatal("expected direct error message")
				}
				return
			}
			if plan.direct.route.Kind != tt.wantDirectKind {
				t.Fatalf("plan.direct.route.Kind = %q, want %q", plan.direct.route.Kind, tt.wantDirectKind)
			}
			gotReads := plan.direct.route.RawEntries()
			if len(gotReads) != len(tt.wantReads) {
				t.Fatalf("len(plan.direct.route.RawEntries()) = %d, want %d", len(gotReads), len(tt.wantReads))
			}
			for i, rawEntry := range gotReads {
				if rawEntry != tt.wantReads[i] {
					t.Fatalf("plan.direct.route.RawEntries()[%d] = %q, want %q", i, rawEntry, tt.wantReads[i])
				}
			}
		})
	}
}

func TestBuildRoutePlan_TrimsScopedDirectInputs(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "pkg", "target.go"): "package pkg\n",
	})

	plan := buildRoutePlan(newRoutePlanExecCtx(root), request{
		query:      "target.go",
		path:       "  pkg  ",
		fileFilter: "  go  ",
	})
	if plan.kind != routeDirect {
		t.Fatalf("plan.kind = %q, want %q", plan.kind, routeDirect)
	}
	if got := plan.direct.route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("pkg", "target.go") {
		t.Fatalf("unexpected trimmed scoped direct route entries: %+v", got)
	}
}

func TestBuildRoutePlan_PreservesExplicitPathsContainingSearchWords(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "README.md"):                          "# root\n",
		filepath.Join(root, "red or blue.md"):                     "red or blue\n",
		filepath.Join(root, "red or blue in docs.md"):             "red or blue in docs\n",
		filepath.Join(root, "release notes in docs.md"):           "release notes\n",
		filepath.Join(root, "files in docs", "summary.md"):        "files\n",
		filepath.Join(root, "files under docs", "summary.md"):     "files under\n",
		filepath.Join(root, "red or blue", "summary.md"):          "red or blue\n",
		filepath.Join(root, "search results", "summary.md"):       "results\n",
		filepath.Join(root, "terms and conditions", "terms.md"):   "terms\n",
		filepath.Join(root, "A or B in docs", "notes.md"):         "explicit relative directory\n",
		filepath.Join(root, "error-or-warning in docs", "log.md"): "literal\n",
	})

	filePlan := buildRoutePlan(newRoutePlanExecCtx(root), request{
		query: "./release notes in docs.md",
	})
	if filePlan.kind != routeDirect {
		t.Fatalf("filePlan.kind = %q, want %q", filePlan.kind, routeDirect)
	}
	if filePlan.direct.route.Kind != directquery.RouteRead {
		t.Fatalf("filePlan.direct.route.Kind = %q, want %q", filePlan.direct.route.Kind, directquery.RouteRead)
	}
	if got := filePlan.direct.route.RawEntries(); len(got) != 1 || got[0] != "release notes in docs.md" {
		t.Fatalf("unexpected explicit path entries: %+v", got)
	}

	explicitRelativeDirPlan := buildRoutePlan(newRoutePlanExecCtx(root), request{
		query: "./A or B in docs/",
	})
	if explicitRelativeDirPlan.kind != routeDirect {
		t.Fatalf("explicitRelativeDirPlan.kind = %q, want %q", explicitRelativeDirPlan.kind, routeDirect)
	}
	if explicitRelativeDirPlan.direct.route.Kind != directquery.RouteDirectory {
		t.Fatalf("explicitRelativeDirPlan.direct.route.Kind = %q, want %q", explicitRelativeDirPlan.direct.route.Kind, directquery.RouteDirectory)
	}

	batchPlan := buildRoutePlan(newRoutePlanExecCtx(root), request{
		query: "./red or blue.md,README.md",
	})
	if batchPlan.kind != routeDirect {
		t.Fatalf("batchPlan.kind = %q, want %q", batchPlan.kind, routeDirect)
	}
	if batchPlan.direct.route.Kind != directquery.RouteRead {
		t.Fatalf("batchPlan.direct.route.Kind = %q, want %q", batchPlan.direct.route.Kind, directquery.RouteRead)
	}
	if got := batchPlan.direct.route.RawEntries(); len(got) != 2 || got[0] != "red or blue.md" || got[1] != "README.md" {
		t.Fatalf("unexpected direct batch entries: %+v", got)
	}

	for _, tt := range []struct {
		query string
		want  []string
	}{
		{
			query: "README.md,./red or blue in docs.md",
			want:  []string{"README.md", "red or blue in docs.md"},
		},
		{
			query: "./red or blue in docs.md,README.md",
			want:  []string{"red or blue in docs.md", "README.md"},
		},
	} {
		t.Run(tt.query, func(t *testing.T) {
			inlineScopeBatchPlan := buildRoutePlan(newRoutePlanExecCtx(root), request{
				query: tt.query,
			})
			if inlineScopeBatchPlan.kind != routeDirect {
				t.Fatalf("inlineScopeBatchPlan.kind = %q, want %q", inlineScopeBatchPlan.kind, routeDirect)
			}
			if inlineScopeBatchPlan.direct.route.Kind != directquery.RouteRead {
				t.Fatalf("inlineScopeBatchPlan.direct.route.Kind = %q, want %q", inlineScopeBatchPlan.direct.route.Kind, directquery.RouteRead)
			}
			got := inlineScopeBatchPlan.direct.route.RawEntries()
			if len(got) != len(tt.want) {
				t.Fatalf("direct inline-scope batch entries = %+v, want %+v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("direct inline-scope batch entries = %+v, want %+v", got, tt.want)
				}
			}
		})
	}

	for _, query := range []string{"files in docs/", "files under docs/", "red or blue/", "search results/", "terms and conditions/", "error-or-warning in docs/"} {
		t.Run(query, func(t *testing.T) {
			dirPlan := buildRoutePlan(newRoutePlanExecCtx(root), request{
				query: query,
			})
			if dirPlan.kind != routeDirect {
				t.Fatalf("dirPlan.kind = %q, want %q", dirPlan.kind, routeDirect)
			}
			if dirPlan.direct.route.Kind != directquery.RouteDirectory {
				t.Fatalf("dirPlan.direct.route.Kind = %q, want %q", dirPlan.direct.route.Kind, directquery.RouteDirectory)
			}
		})
	}
}

func TestPlanRoute_UsesImplicitDirectFileRouteOnlyWithoutSearchScope(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "Makefile"): "all:\n",
	})

	implicit := buildRoutePlan(newRoutePlanExecCtx(root), request{query: "Makefile"})
	if implicit.kind != routeDirect {
		t.Fatalf("implicit.kind = %q, want %q", implicit.kind, routeDirect)
	}
	if implicit.direct.route.Kind != directquery.RouteRead {
		t.Fatalf("implicit.direct.route.Kind = %q, want %q", implicit.direct.route.Kind, directquery.RouteRead)
	}
	if gotReads := implicit.direct.route.RawEntries(); len(gotReads) != 1 || gotReads[0] != "Makefile" {
		t.Fatalf("unexpected implicit read targets: %+v", gotReads)
	}

	scoped := buildRoutePlan(newRoutePlanExecCtx(root), request{query: "Makefile", path: "pkg"})
	if scoped.kind != routeSearch {
		t.Fatalf("scoped.kind = %q, want %q", scoped.kind, routeSearch)
	}
}

func TestPlanRoute_ExplicitRelativeQueryDoesNotWidenToProjectRoot(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	withGatherContextWorkingDir(t, root)

	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "Makefile"): "all:\n",
	})

	plan := buildRoutePlan(newRoutePlanExecCtx(root, withGatherContextInvocationCWD(subdir)), request{
		query:      "./Makefile",
		fileFilter: "go",
	})
	if plan.kind != routeDirectError {
		t.Fatalf("plan.kind = %q, want %q", plan.kind, routeDirectError)
	}
	if plan.direct.err == "" {
		t.Fatal("expected direct error for missing cwd-relative file")
	}
}

func TestPlanRoute_ExplicitParentRelativeQueryStaysDirectWithinRepo(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	withGatherContextWorkingDir(t, root)

	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "README.md"): "# root\n",
	})

	readPlan := buildRoutePlan(newRoutePlanExecCtx(root, withGatherContextInvocationCWD(subdir)), request{
		query:      "../README.md",
		fileFilter: "go",
	})
	if readPlan.kind != routeDirect {
		t.Fatalf("readPlan.kind = %q, want %q", readPlan.kind, routeDirect)
	}
	if got := readPlan.direct.route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("..", "README.md") {
		t.Fatalf("unexpected parent-relative read entries: %+v", got)
	}

	dirPlan := buildRoutePlan(newRoutePlanExecCtx(root, withGatherContextInvocationCWD(subdir)), request{
		query:      "../pkg/",
		fileFilter: "go",
	})
	if dirPlan.kind != routeDirect {
		t.Fatalf("dirPlan.kind = %q, want %q", dirPlan.kind, routeDirect)
	}
	if dirPlan.direct.route.Kind != directquery.RouteDirectory {
		t.Fatalf("dirPlan.direct.route.Kind = %q, want %q", dirPlan.direct.route.Kind, directquery.RouteDirectory)
	}

	outsidePlan := buildRoutePlan(newRoutePlanExecCtx(root, withGatherContextInvocationCWD(subdir)), request{
		query:      "../../outside.txt",
		fileFilter: "go",
	})
	if outsidePlan.kind != routeDirectError {
		t.Fatalf("outsidePlan.kind = %q, want %q", outsidePlan.kind, routeDirectError)
	}
}
