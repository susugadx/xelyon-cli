package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestFormatSearchResults_WithLocatorID(t *testing.T) {
	reg := locator.NewRegistry()

	results := []SearchResult{
		{
			FilePath:   "internal/agent/agent.go",
			MatchCount: 2,
			Matches: []Match{
				{LineNum: 10, Line: "x := NewAgent()", IsMatch: true, Type: MatchTypeCall, Block: &BlockInfo{Name: "func main", StartLine: 5}},
				{LineNum: 20, Line: "y := agent.Run()", IsMatch: true, Type: MatchTypeCall, Block: nil},
			},
		},
	}

	out := formatSearchResults(results, false, 15000, reg)

	// ファイルヘッダーにLocator IDが付与される
	if !strings.Contains(out, "[L1]") {
		t.Errorf("expected Locator ID for file header, got:\n%s", out)
	}

	// ブロック情報にLocator IDが付与される
	if !strings.Contains(out, "[L2]") {
		t.Errorf("expected Locator ID for block info, got:\n%s", out)
	}
}

func TestFormatSearchResults_NilRegistryNoIDs(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:   "main.go",
			MatchCount: 1,
			Matches: []Match{
				{LineNum: 1, Line: "package main", IsMatch: true, Type: MatchTypeDefinition},
			},
		},
	}

	out := formatSearchResults(results, false, 15000, nil)

	if strings.Contains(out, "[L") {
		t.Errorf("expected no Locator IDs with nil registry, got:\n%s", out)
	}
}

func TestFormatManifestResults_WithLocatorID(t *testing.T) {
	reg := locator.NewRegistry()

	results := []SearchResult{
		{
			FilePath:   "a.go",
			MatchCount: 3,
			Matches: []Match{
				{LineNum: 1, Line: "x", IsMatch: true, Type: MatchTypeUsage},
			},
		},
	}

	out := formatManifestResults(results, reg)

	if !strings.Contains(out, "[L1]") {
		t.Errorf("expected Locator ID in manifest output, got:\n%s", out)
	}
}

func TestFormatGenericSymbolResult_WithLocatorID(t *testing.T) {
	reg := locator.NewRegistry()

	def := genericSymbolDef{
		Name:      "MyFunc",
		Kind:      "function",
		File:      "pkg/util.go",
		Line:      42,
		Signature: "def MyFunc():",
	}
	refs := []genericSymbolRef{
		{File: "main.go", Line: 10, Snippet: "MyFunc()"},
	}
	bundle := buildGenericSymbolBundle("python", "MyFunc", def, []string{
		"42: def MyFunc():",
	}, []symbolBundleSectionInput{
		{Kind: "references", Title: "References", Items: refs, Limit: genericRefLimit},
	})

	out := formatSymbolBundle(bundle, reg, nil)

	// 定義ヘッダーにLocator IDが付与される
	if !strings.Contains(out, "[L1]") {
		t.Errorf("expected Locator ID for definition header, got:\n%s", out)
	}

	// 参照行にLocator IDが付与される
	if !strings.Contains(out, "[L2]") {
		t.Errorf("expected Locator ID for reference, got:\n%s", out)
	}
}

func TestLocatorID_Idempotent_InSearchResults(t *testing.T) {
	reg := locator.NewRegistry()

	results := []SearchResult{
		{
			FilePath:   "same.go",
			MatchCount: 2,
			Matches: []Match{
				{LineNum: 10, Line: "a", IsMatch: true, Type: MatchTypeCall, Block: &BlockInfo{Name: "func X", StartLine: 5}},
				{LineNum: 20, Line: "b", IsMatch: true, Type: MatchTypeCall, Block: &BlockInfo{Name: "func X", StartLine: 5}},
			},
		},
	}

	out := formatSearchResults(results, false, 15000, reg)

	// same.go [L1] がファイルヘッダー、func X [L2] がブロック
	// ブロックが2回出現しても同じIDが返る（冪等性）
	count := strings.Count(out, "[L2]")
	if count != 2 {
		t.Errorf("expected [L2] to appear twice (idempotent), got %d times in:\n%s", count, out)
	}

	// [L3] は存在しないはず（重複登録されていない）
	if strings.Contains(out, "[L3]") {
		t.Errorf("unexpected [L3] - dedup should prevent duplicate registration:\n%s", out)
	}
}
