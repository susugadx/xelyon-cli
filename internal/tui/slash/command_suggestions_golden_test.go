package slash

import "testing"

func TestSuggestions_GoldenOrderForRootPrefix(t *testing.T) {
	got := Suggestions("/")
	gotNames := make([]string, 0, len(got))
	for _, cmd := range got {
		gotNames = append(gotNames, cmd.Name)
	}

	want := []string{
		"/model",
		"/use",
		"/providers",
		"/think",
		"/status",
		"/tokens",
		"/review",
		"/project",
		"/config",
		"/copy",
		"/attach",
		"/detach",
		"/detach-all",
		"/compress",
		"/plan",
		"/save",
		"/load",
		"/sessions",
		"/clear",
		"/history",
		"/init",
		"/exit",
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
