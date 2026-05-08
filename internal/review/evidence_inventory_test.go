package review

import "testing"

func TestReviewEvidenceInventoryClassifiesSurfacesAndStatuses(t *testing.T) {
	inventory := buildReviewChangeInventory([]ReviewChangedFile{
		{Path: "internal/app.go", Status: "M"},
		{Path: "internal/app_test.go", Status: "M"},
		{Path: "internal/tui/testdata/focus.json", Status: "M"},
		{Path: "tests/fixture.json", Status: "M"},
		{Path: "docs/review.md", Status: "M"},
		{Path: "xelyon.yaml", Status: "M"},
		{Path: "internal/registry_generated.go", Status: "M"},
		{Path: "generated/schema.go", Status: "M"},
		{Path: "new.go", Status: "A"},
		{Path: "old.go", Status: "D"},
		{Path: "new-name.go", OldPath: "old-name.go", Status: "R100"},
		{Path: "internal/foo.go", OldPath: "internal/foo_test.go", Status: "R100"},
	}, []string{"scratch.txt"})

	assertStringSlice(t, inventory.Generated, []string{"generated/schema.go", "internal/registry_generated.go"})
	assertStringSlice(t, inventory.Tests, []string{"internal/app_test.go", "internal/foo_test.go", "internal/tui/testdata/focus.json", "tests/fixture.json"})
	assertStringSlice(t, inventory.Docs, []string{"docs/review.md"})
	assertStringSlice(t, inventory.Config, []string{"xelyon.yaml"})
	assertStringSlice(t, inventory.Production, []string{"internal/app.go", "internal/foo.go", "new-name.go", "new.go", "old-name.go", "old.go", "scratch.txt"})
	assertStringSlice(t, inventory.NewFiles, []string{"new.go"})
	assertStringSlice(t, inventory.DeletedFiles, []string{"old.go"})
	assertStringSlice(t, inventory.RenamedFiles, []string{"internal/foo.go", "new-name.go"})
	assertStringSlice(t, inventory.Untracked, []string{"scratch.txt"})
}
