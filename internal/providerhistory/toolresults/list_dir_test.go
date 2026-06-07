package toolresults

import (
	"strings"
	"testing"
)

func TestBuildStructuredReplacement_ListDirPathNormalization(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantText string
	}{
		{name: "dot", path: ".", wantText: "path=."},
		{name: "dot slash", path: "./", wantText: "path=."},
		{name: "relative with dot", path: "./internal", wantText: "path=internal"},
		{name: "repo relative", path: "internal/providerhistory", wantText: "path=internal/providerhistory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacement, reason, ok := BuildStructuredReplacement(NewReplacementRequest("list_dir", listDirArgs(tt.path), largeListDirResult()))
			if !ok {
				t.Fatalf("BuildStructuredReplacement() ok=false reason=%q", reason)
			}
			if !strings.Contains(replacement.Text(), tt.wantText) {
				t.Fatalf("replacement text = %q, want %q", replacement.Text(), tt.wantText)
			}
		})
	}
}

func TestBuildStructuredReplacement_ListDirRejectsUnsafePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "parent escape", path: "../outside"},
		{name: "absolute path", path: "/tmp/project"},
		{name: "windows absolute path", path: `C:\tmp\project`},
		{name: "empty path", path: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, reason, ok := BuildStructuredReplacement(NewReplacementRequest("list_dir", listDirArgs(tt.path), largeListDirResult()))
			if ok {
				t.Fatal("BuildStructuredReplacement() ok=true, want rejected unsafe path")
			}
			if tt.path == "" {
				if reason != "missing_list_dir_path_argument" {
					t.Fatalf("reason = %q, want missing_list_dir_path_argument", reason)
				}
				return
			}
			if reason != "unsafe_list_dir_path" {
				t.Fatalf("reason = %q, want unsafe_list_dir_path", reason)
			}
		})
	}
}

func TestBuildStructuredReplacement_ListDirParsesSummary(t *testing.T) {
	content := "📂 /abs/project/internal\nsummary: depth=2, dirs=3, files=4\n" + strings.Repeat("files: a.go (1 bytes)\n", 200)
	replacement, reason, ok := BuildStructuredReplacement(NewReplacementRequest("list_dir", listDirArgs("internal"), content))
	if !ok {
		t.Fatalf("BuildStructuredReplacement() ok=false reason=%q", reason)
	}
	for _, want := range []string{"path=internal", "entries=7", "depth=2"} {
		if !strings.Contains(replacement.Text(), want) {
			t.Fatalf("replacement text = %q, want %q", replacement.Text(), want)
		}
	}
	if strings.Contains(replacement.Text(), "/abs/project") {
		t.Fatalf("replacement leaked absolute result path: %q", replacement.Text())
	}
	if replacement.SavedBytes() <= 0 || replacement.SavedTokens() <= 0 {
		t.Fatalf("saved metrics = bytes %d tokens %d, want positive", replacement.SavedBytes(), replacement.SavedTokens())
	}
}

func TestBuildStructuredReplacement_ListDirRejectsUnsuccessfulOrMalformedResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		reason  string
	}{
		{name: "missing summary", content: "📂 /abs/project\nfiles: main.go (10 bytes)", reason: "list_dir_summary_unparseable"},
		{name: "malformed summary", content: "📂 /abs/project\nsummary: dirs=3, files=4", reason: "list_dir_summary_unparseable"},
		{name: "error prefix", content: "Error: /abs/missing is not a directory", reason: "list_dir_result_not_success"},
		{name: "read error after header", content: "📂 /abs/project\nsummary: depth=1, dirs=0, files=0\nError: failed to read directory", reason: "list_dir_result_not_success"},
		{name: "no header", content: "summary: depth=1, dirs=1, files=1", reason: "list_dir_result_not_success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, reason, ok := BuildStructuredReplacement(NewReplacementRequest("list_dir", listDirArgs("internal"), tt.content))
			if ok {
				t.Fatal("BuildStructuredReplacement() ok=true, want rejected result")
			}
			if reason != tt.reason {
				t.Fatalf("reason = %q, want %q", reason, tt.reason)
			}
		})
	}
}

func listDirArgs(path string) string {
	return `{"path":` + strconvQuote(path) + `,"depth":2}`
}

func strconvQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func largeListDirResult() string {
	return "📂 /abs/project\nsummary: depth=2, dirs=3, files=4\n" + strings.Repeat("files: main.go (10 bytes), README.md (20 bytes)\n", 200)
}
