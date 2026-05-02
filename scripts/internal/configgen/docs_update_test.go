package configgen

import "testing"

func TestReplaceConfigExampleBlock(t *testing.T) {
	content := "<!-- CONFIG-EXAMPLE-START -->\nold\n<!-- CONFIG-EXAMPLE-END -->"
	updated, err := ReplaceConfigExampleBlock(content, configExampleFileHeader+"general:\n  ui_language: auto\n")
	if err != nil {
		t.Fatalf("ReplaceConfigExampleBlock returned error: %v", err)
	}
	if updated == content {
		t.Fatal("expected config example block to be replaced")
	}
	if contains := HasConfigDetailsMarkers(updated); contains {
		t.Fatal("unexpected details marker detection")
	}
}

func TestReplaceConfigExampleBlockMissingMarkers(t *testing.T) {
	if _, err := ReplaceConfigExampleBlock("no markers", "general:\n  ui_language: auto\n"); err == nil {
		t.Fatal("expected marker error")
	}
}

func TestConfigDetailsMarkerHelpers(t *testing.T) {
	content := "<!-- CONFIG-DETAILS-START -->\nold\n<!-- CONFIG-DETAILS-END -->"
	if !HasConfigDetailsMarkers(content) {
		t.Fatal("expected details markers to be detected")
	}
	updated := ReplaceConfigDetailsBlock(content, "new")
	if updated == content {
		t.Fatal("expected details block to be replaced")
	}
}
