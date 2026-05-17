package search

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptSymbolFiltersLSPReferencesByPathScope(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
		"src/app.ts":   "buildUser('src')\n",
		"other/app.ts": "buildUser('other')\n",
	})
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{
			{File: "src/app.ts", Line: 1, Character: 1},
			{File: "other/app.ts", Line: 1, Character: 1},
		},
	}
	opts := SearchOptions{
		Pattern:       "buildUser",
		Mode:          string(SearchModeSymbol),
		Path:          filepath.Join(dir, "src"),
		FileType:      "ts",
		InvocationCWD: dir,
		LSPClient:     lspClient,
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want TypeScript symbol bundle; output:\n%s", artifact.Rendered)
	}
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.ts") {
		t.Fatalf("callers = %+v, want in-scope LSP caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "other/app.ts") || symbolBundleItemsContainSnippet(callers, "other") {
		t.Fatalf("callers = %+v, did not want out-of-scope LSP caller", callers)
	}
}

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

func TestJSFamilyLSPReferenceCollectorFiltersBeforeLoadingEvidence(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/app.ts":       "buildUser('accepted')\n",
		"src/generated.ts": "buildUser('generated')\n",
		"src/app.jsx":      "buildUser('jsx')\n",
		"other/app.ts":     "buildUser('other')\n",
	})
	opts := SearchOptions{
		Path:          filepath.Join(dir, "src"),
		FileType:      "ts",
		InvocationCWD: dir,
		ignoreMatcher: pathmatch.NewMatcher([]string{"src/generated.ts"}),
	}
	lspOpts := newJSFamilyLSPReferenceOptions(opts, opts, opts)
	collector := newJSFamilyLSPReferenceCollector("buildUser", lspOpts, 5)
	defer collector.Close()

	var loaded []string
	acceptedPath := filepath.Join(dir, "src", "app.ts")
	collector.builder.loadFile = func(absPath string) *jsFamilyLSPReferenceFile {
		loaded = append(loaded, absPath)
		return &jsFamilyLSPReferenceFile{lines: []string{"buildUser('accepted')"}}
	}

	tests := []struct {
		name string
		loc  navigation.LSPLocation
		want bool
	}{
		{name: "accepted", loc: navigation.LSPLocation{File: "src/app.ts", Line: 1, Character: 1}, want: true},
		{name: "ignored", loc: navigation.LSPLocation{File: "src/generated.ts", Line: 1, Character: 1}},
		{name: "wrong file type", loc: navigation.LSPLocation{File: "src/app.jsx", Line: 1, Character: 1}},
		{name: "out of scope", loc: navigation.LSPLocation{File: "other/app.ts", Line: 1, Character: 1}},
		{name: "missing line", loc: navigation.LSPLocation{File: "src/app.ts", Character: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collector.AddLocation(tt.loc); got != tt.want {
				t.Fatalf("AddLocation() = %v, want %v", got, tt.want)
			}
		})
	}

	collection := collector.Result()
	if len(collection.refs) != 1 {
		t.Fatalf("refs = %+v, want one accepted LSP ref", collection.refs)
	}
	if got := collection.refs[0].File; got != "src/app.ts" {
		t.Fatalf("ref file = %q, want src/app.ts", got)
	}
	if got := collection.refs[0].Snippet; got != "buildUser('accepted')" {
		t.Fatalf("snippet = %q, want accepted snippet", got)
	}
	if len(loaded) != 1 || loaded[0] != acceptedPath {
		t.Fatalf("loaded paths = %+v, want only %q", loaded, acceptedPath)
	}
}

func TestJSFamilyLSPReferenceBuilderLoadsEachFileOnce(t *testing.T) {
	absPath := filepath.Join(t.TempDir(), "src", "app.ts")
	loads := 0
	builder := &jsFamilyLSPReferenceBuilder{
		symbol: "buildUser",
		files:  make(map[string]*jsFamilyLSPReferenceFile),
		loadFile: func(gotPath string) *jsFamilyLSPReferenceFile {
			loads++
			if gotPath != absPath {
				t.Fatalf("load path = %q, want %q", gotPath, absPath)
			}
			return &jsFamilyLSPReferenceFile{lines: []string{
				"buildUser('one')",
				"buildUser('two')",
			}}
		},
	}
	defer builder.Close()

	first := builder.Ref(jsFamilyLSPReferenceCandidate{
		displayPath: "src/app.ts",
		absPath:     absPath,
		loc:         navigation.LSPLocation{Line: 1},
	})
	second := builder.Ref(jsFamilyLSPReferenceCandidate{
		displayPath: "src/app.ts",
		absPath:     absPath,
		loc:         navigation.LSPLocation{Line: 2},
	})

	if loads != 1 {
		t.Fatalf("loads = %d, want one load for repeated LSP locations in the same file", loads)
	}
	if first.Snippet != "buildUser('one')" || second.Snippet != "buildUser('two')" {
		t.Fatalf("snippets = (%q, %q), want cached file snippets", first.Snippet, second.Snippet)
	}
}
