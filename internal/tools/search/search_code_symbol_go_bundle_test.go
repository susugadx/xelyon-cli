package search

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestGoSymbolResolver_LocatorRegistryDoesNotRegisterHiddenIDs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": `package example

type Builder interface {
	Build() string
}
`,
		"builder_impl.go": `package example

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`,
		"builder_test.go": `package example

func TestBuild(t *testing.T) {
	var b FileBuilder
	_ = b.Build()
}
`,
	})

	reg := locator.NewRegistry()
	resolved := goSymbolResolver{}.Resolve("Builder", SearchOptions{
		Path:            dir,
		LocatorRegistry: reg,
		LSPClient: &mockGoSymbolLSPClient{
			refs:  []navigation.LSPLocation{{File: "builder_test.go", Line: 5, Character: 1, EndLine: 5, EndChar: 6}},
			impls: []navigation.LSPLocation{{File: "builder_impl.go", Line: 3, Character: 1, EndLine: 3, EndChar: 11}},
		},
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}

	ids := visibleLocatorIDs(resolved.Output)
	if len(ids) == 0 {
		t.Fatalf("expected visible locator IDs in output, got:\n%s", resolved.Output)
	}
	for i, id := range ids {
		want := "[L" + strconv.Itoa(i+1) + "]"
		if id != want {
			t.Fatalf("expected sequential locator %s, got %s in output:\n%s", want, id, resolved.Output)
		}
	}
	if _, ok := reg.Resolve("[L" + strconv.Itoa(len(ids)+1) + "]"); ok {
		t.Fatalf("expected no hidden locator beyond visible IDs, got extra registry entry after %d visible IDs", len(ids))
	}
}

func TestGoSymbolResolver_LocatorRegistryMatchesImplementation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": `package example

type Builder interface {
	Build() string
}
`,
		"builder_impl.go": `package example

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`,
	})

	reg := locator.NewRegistry()
	resolved := goSymbolResolver{}.Resolve("Builder", SearchOptions{
		Path:            dir,
		LocatorRegistry: reg,
		LSPClient: &mockGoSymbolLSPClient{
			refs:  []navigation.LSPLocation{{File: "builder_test.go", Line: 5, Character: 1, EndLine: 5, EndChar: 6}},
			impls: []navigation.LSPLocation{{File: "builder_impl.go", Line: 3, Character: 1, EndLine: 3, EndChar: 11}},
		},
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}

	implID := locatorIDForLine(t, resolved.Output, "builder_impl.go:3")
	implLoc, ok := reg.Resolve(implID)
	if !ok {
		t.Fatalf("expected implementation locator %s to resolve", implID)
	}
	if implLoc.FilePath != "builder_impl.go" || implLoc.Line != 3 {
		t.Fatalf("unexpected implementation locator target: %+v", implLoc)
	}
}

func TestFormatSymbolBundle_LocatorRegistryMatchesRelatedTest(t *testing.T) {
	reg := locator.NewRegistry()
	bundle := buildGoSymbolBundle("Close", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:     "Close",
			Kind:     "method",
			File:     "agent.go",
			Line:     5,
			EndLine:  7,
			Receiver: "*Agent",
		},
		Body: []string{
			"5: func (a *Agent) Close() error {",
			"6: \treturn nil",
			"7: }",
		},
		Tests: []navigation.TestRef{
			{File: "agent_test.go", Line: 4, Name: "TestClose"},
		},
		TotalTests: 1,
	})
	output := formatSymbolBundle(bundle, reg, nil)

	testID := locatorIDForLine(t, output, "agent_test.go:4")
	testLoc, ok := reg.Resolve(testID)
	if !ok {
		t.Fatalf("expected test locator %s to resolve", testID)
	}
	if testLoc.FilePath != "agent_test.go" || testLoc.Line != 4 {
		t.Fatalf("unexpected test locator target: %+v", testLoc)
	}
}

func TestFormatSymbolBundle_SectionItemLocatorsPreferResolvedPath(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootShadow := filepath.Join(root, "target.go")
	subdirTarget := filepath.Join(subdir, "target.go")
	for _, path := range []string{rootShadow, subdirTarget} {
		if err := os.WriteFile(path, []byte("package pkg\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	reg := locator.NewRegistry()
	bundle := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:     "Run",
			Kind:     "function",
			File:     "pkg/run.go",
			Line:     3,
			EndLine:  3,
			RootPath: root,
		},
		Body: []string{"3: func Run() {}"},
		Refs: []navigation.Reference{
			{File: "target.go", ResolvedPath: subdirTarget, Line: 8, Snippet: "Run()"},
		},
	})

	output := formatSymbolBundle(bundle, reg, nil)
	refID := locatorIDForLine(t, output, "target.go:8 | Run()")
	refLoc, ok := reg.Resolve(refID)
	if !ok {
		t.Fatalf("expected reference locator %s to resolve", refID)
	}
	if refLoc.ResolvedPath != subdirTarget {
		t.Fatalf("expected reference locator to use %s, got %+v", subdirTarget, refLoc)
	}
	if refLoc.ResolvedPath == rootShadow {
		t.Fatalf("expected reference locator to avoid root shadow path, got %+v", refLoc)
	}
}

func TestCollectSymbolBundleAffectedFiles_IncludesRecommendedReadFiles(t *testing.T) {
	dir := t.TempDir()
	bundle := &SymbolBundle{
		Definition: SymbolBundleDefinition{
			File: "pkg/run.go",
			Line: 3,
		},
		Impact: &SymbolBundleImpact{
			RiskLevel: "medium",
			RecommendedReads: []SymbolBundleItem{
				{Kind: "references", File: "crosspkg/reader.go", Line: 8, Snippet: "_ = Run()"},
			},
		},
		Debug: SymbolBundleDebug{
			FileRootPath: dir,
		},
	}

	affected := collectSymbolBundleAffectedFiles(bundle, SearchOptions{ProjectMapRootPath: dir})
	wantDefinition := filepath.Join(dir, "pkg", "run.go")
	wantRecommended := filepath.Join(dir, "crosspkg", "reader.go")
	for _, want := range []string{wantDefinition, wantRecommended} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected affected files to include %s, got %v", want, affected)
		}
	}
}

func TestCollectSymbolBundleAffectedFiles_PrefersItemResolvedPath(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootShadow := filepath.Join(root, "target.go")
	subdirTarget := filepath.Join(subdir, "target.go")
	for _, path := range []string{rootShadow, subdirTarget} {
		if err := os.WriteFile(path, []byte("package pkg\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	bundle := &SymbolBundle{
		Definition: SymbolBundleDefinition{
			File: "pkg/run.go",
			Line: 3,
		},
		Sections: []SymbolBundleSection{
			{
				Kind:  "references",
				Title: "References",
				Items: []SymbolBundleItem{
					{Kind: "references", File: "target.go", ResolvedPath: subdirTarget, Line: 8, Snippet: "Run()"},
				},
			},
		},
		Debug: SymbolBundleDebug{
			FileRootPath: root,
		},
	}

	affected := collectSymbolBundleAffectedFiles(bundle, SearchOptions{ProjectMapRootPath: root, InvocationCWD: subdir})
	if !containsAffectedFile(affected, subdirTarget) {
		t.Fatalf("expected affected files to include %s, got %v", subdirTarget, affected)
	}
	if containsAffectedFile(affected, rootShadow) {
		t.Fatalf("did not expect affected files to include root shadow %s, got %v", rootShadow, affected)
	}
}

func TestBuildGoSymbolBundleLimitsImplementations(t *testing.T) {
	bundle := buildGoSymbolBundle("Closer", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Closer",
			Kind:    "interface",
			File:    "closer.go",
			Line:    5,
			EndLine: 7,
		},
		Body: []string{
			"5: type Closer interface {",
			"6: \tClose() error",
			"7: }",
		},
		Implementations: []navigation.ImplementationRef{
			{File: "agent.go", Line: 10, Name: "Agent"},
			{File: "service.go", Line: 20, Name: "Service"},
			{File: "worker.go", Line: 30, Name: "Worker"},
			{File: "job.go", Line: 40, Name: "Job"},
			{File: "task.go", Line: 50, Name: "Task"},
		},
	})

	var implSection *SymbolBundleSection
	for i := range bundle.Sections {
		if bundle.Sections[i].Kind == "implementations" {
			implSection = &bundle.Sections[i]
			break
		}
	}
	if implSection == nil {
		t.Fatal("expected implementations section")
	}
	if len(implSection.Items) != goImplementationLimit {
		t.Fatalf("expected %d implementation items, got %d", goImplementationLimit, len(implSection.Items))
	}
	if implSection.Total != 5 {
		t.Fatalf("expected Total=5, got %d", implSection.Total)
	}
	if !implSection.More {
		t.Fatal("expected More=true when implementations are truncated")
	}
}

func TestBuildGoSymbolBundleKeepsAllImplementationsWhenUnderLimit(t *testing.T) {
	bundle := buildGoSymbolBundle("Closer", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Closer",
			Kind:    "interface",
			File:    "closer.go",
			Line:    5,
			EndLine: 7,
		},
		Body: []string{
			"5: type Closer interface {",
			"6: \tClose() error",
			"7: }",
		},
		Implementations: []navigation.ImplementationRef{
			{File: "agent.go", Line: 10, Name: "Agent"},
			{File: "service.go", Line: 20, Name: "Service"},
		},
	})

	var implSection *SymbolBundleSection
	for i := range bundle.Sections {
		if bundle.Sections[i].Kind == "implementations" {
			implSection = &bundle.Sections[i]
			break
		}
	}
	if implSection == nil {
		t.Fatal("expected implementations section")
	}
	if len(implSection.Items) != 2 {
		t.Fatalf("expected 2 implementation items, got %d", len(implSection.Items))
	}
	if implSection.Total != 2 {
		t.Fatalf("expected Total=2, got %d", implSection.Total)
	}
	if implSection.More {
		t.Fatal("expected More=false when implementations are not truncated")
	}
}
