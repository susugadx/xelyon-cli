package navigation

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func TestLSPAdapter_ConvertLocations(t *testing.T) {
	adapter := NewLSPAdapter(&lsp.Client{}, "/workspace")
	locations := adapter.convertLocations([]lsp.Location{
		{
			URI: "file:///workspace/internal/pkg/file.go",
			Range: lsp.Range{
				Start: lsp.Position{Line: 9, Character: 3},
				End:   lsp.Position{Line: 9, Character: 8},
			},
		},
	})

	if len(locations) != 1 {
		t.Fatalf("len(locations) = %d, want 1", len(locations))
	}
	if locations[0].File != "internal/pkg/file.go" {
		t.Fatalf("file = %q, want relative path", locations[0].File)
	}
	if locations[0].Line != 10 || locations[0].Character != 4 {
		t.Fatalf("unexpected 1-indexed conversion: %+v", locations[0])
	}
}
