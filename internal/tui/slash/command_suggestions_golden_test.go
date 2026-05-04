package slash

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

func TestSuggestions_GoldenOrderForRootPrefix(t *testing.T) {
	got := Suggestions("/")
	gotNames := make([]string, 0, len(got))
	for _, cmd := range got {
		gotNames = append(gotNames, cmd.Name)
	}

	wantCommands := commandcatalog.DiscoverableCommandsForSurface(commandcatalog.CommandSurfaceTUI)
	want := make([]string, 0, len(wantCommands))
	for _, cmd := range wantCommands {
		want = append(want, cmd.Name)
	}

	if len(gotNames) != len(want) {
		t.Fatalf("len(Suggestions(/)) = %d, want %d\n got=%#v", len(gotNames), len(want), gotNames)
	}
	for i := range want {
		if gotNames[i] != want[i] {
			t.Fatalf("Suggestions(/)[%d] = %q, want %q\n got=%#v", i, gotNames[i], want[i], gotNames)
		}
	}
}
