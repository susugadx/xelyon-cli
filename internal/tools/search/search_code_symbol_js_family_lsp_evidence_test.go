package search

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

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
	collector := newJSFamilyLSPReferenceCollector("buildUser", genericSymbolDef{Name: "buildUser"}, lspOpts, 5)
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

func TestJSFamilyLSPReferenceCollectorBudgetsEvidenceButKeepsSummary(t *testing.T) {
	files := map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n",
	}
	locations := make([]navigation.LSPLocation, 0, jsFamilyLSPReferenceEvidenceLimit+3)
	for i := 0; i < jsFamilyLSPReferenceEvidenceLimit+2; i++ {
		file := fmt.Sprintf("src/caller%02d.ts", i)
		line := fmt.Sprintf("buildUser('caller-%02d')", i)
		files[file] = line + "\n"
		start, end := testLSPRangeForSearchToken(line, "buildUser")
		locations = append(locations, navigation.LSPLocation{File: file, Line: 1, Character: start, EndLine: 1, EndChar: end})
	}
	testLine := "buildUser('test caller')"
	files["src/build.test.ts"] = testLine + "\n"
	start, end := testLSPRangeForSearchToken(testLine, "buildUser")
	locations = append(locations, navigation.LSPLocation{File: "src/build.test.ts", Line: 1, Character: start, EndLine: 1, EndChar: end})

	dir := setupMultiLangDir(t, files)
	opts := SearchOptions{
		Path:          filepath.Join(dir, "src"),
		FileType:      "ts",
		InvocationCWD: dir,
	}
	lspOpts := newJSFamilyLSPReferenceOptions(opts, opts, opts)
	collection := collectJSFamilyLSPReferences("buildUser", genericSymbolDef{
		Name: "buildUser",
		File: "src/build.ts",
		Line: 1,
	}, locations, lspOpts)

	if len(collection.summaryRefs) != len(locations) {
		t.Fatalf("summary refs len = %d, want all %d accepted locations", len(collection.summaryRefs), len(locations))
	}
	if len(collection.refs) != jsFamilyLSPReferenceEvidenceLimit {
		t.Fatalf("evidence refs len = %d, want budget %d", len(collection.refs), jsFamilyLSPReferenceEvidenceLimit)
	}
	for _, ref := range collection.summaryRefs {
		if ref.Snippet != "" {
			t.Fatalf("summary ref = %+v, want no snippet payload", ref)
		}
	}
	for _, ref := range collection.refs {
		if ref.Snippet == "" {
			t.Fatalf("evidence ref = %+v, want snippet payload", ref)
		}
	}
	if !genericRefsContainSnippet(collection.refs, "test caller") {
		t.Fatalf("evidence refs = %+v, want test reference selected even when it is beyond raw budget order", collection.refs)
	}
}
