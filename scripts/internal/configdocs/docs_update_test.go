package configdocs

import (
	"strings"
	"testing"
)

func TestReplaceConfigExampleBlock(t *testing.T) {
	content := "<!-- CONFIG-EXAMPLE-START -->\nold\n<!-- CONFIG-EXAMPLE-END -->"
	updated, err := ReplaceConfigExampleBlock(content, testConfigExampleFileHeader+"general:\n  ui_language: auto\n")
	if err != nil {
		t.Fatalf("ReplaceConfigExampleBlock returned error: %v", err)
	}
	if updated == content {
		t.Fatal("expected config example block to be replaced")
	}
	if contains := hasConfigDetailsMarkers(updated); contains {
		t.Fatal("unexpected details marker detection")
	}
}

func TestReplaceConfigExampleBlockKeepsSectionSpacing(t *testing.T) {
	example := testConfigExampleFileHeader + `# ============================================================
# 一般設定
# ============================================================
general:
  ui_language: auto
# ============================================================
# 会話履歴圧縮設定
# ============================================================
compression:
  enabled: true
`
	content := "<!-- CONFIG-EXAMPLE-START -->\nold\n<!-- CONFIG-EXAMPLE-END -->"

	updated, err := ReplaceConfigExampleBlock(content, example)
	if err != nil {
		t.Fatalf("ReplaceConfigExampleBlock returned error: %v", err)
	}

	if !strings.Contains(updated, "ui_language: auto\n\n# ============================================================") {
		t.Fatalf("expected blank line before next generated section, got %s", updated)
	}
}

func TestReplaceConfigExampleBlockMissingMarkers(t *testing.T) {
	if _, err := ReplaceConfigExampleBlock("no markers", "general:\n  ui_language: auto\n"); err == nil {
		t.Fatal("expected marker error")
	}
}

func TestConfigDetailsMarkerHelpers(t *testing.T) {
	content := "<!-- CONFIG-DETAILS-START -->\nold\n<!-- CONFIG-DETAILS-END -->"
	if !hasConfigDetailsMarkers(content) {
		t.Fatal("expected details markers to be detected")
	}
	updated := replaceConfigDetailsBlock(content, "new")
	if updated == content {
		t.Fatal("expected details block to be replaced")
	}
}
