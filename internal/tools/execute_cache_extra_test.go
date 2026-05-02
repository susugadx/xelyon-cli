package tools

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type recordingToolCache struct {
	invalidatedFiles  []string
	invalidatedDirs   []string
	invalidatedSearch []string
	clearCount        int
}

func (c *recordingToolCache) GetFile(string) (string, bool)              { return "", false }
func (c *recordingToolCache) SetFile(string, string)                     {}
func (c *recordingToolCache) GetDir(string) (string, bool)               { return "", false }
func (c *recordingToolCache) SetDir(string, string)                      {}
func (c *recordingToolCache) GetSearch(string, string) (string, bool)    { return "", false }
func (c *recordingToolCache) SetSearch(string, string, string, []string) {}
func (c *recordingToolCache) ClearSearchCache()                          {}
func (c *recordingToolCache) InvalidateFile(path string) {
	c.invalidatedFiles = append(c.invalidatedFiles, path)
}
func (c *recordingToolCache) InvalidateDir(path string) {
	c.invalidatedDirs = append(c.invalidatedDirs, path)
}
func (c *recordingToolCache) Clear() { c.clearCount++ }
func (c *recordingToolCache) InvalidateSearchCacheForFile(path string) {
	c.invalidatedSearch = append(c.invalidatedSearch, path)
}

type captureArgsTool struct {
	name     string
	result   string
	gotArgs  map[string]string
	runCount int
}

func (t *captureArgsTool) Name() string { return t.name }

func (t *captureArgsTool) Description() string { return "capture args tool" }

func (t *captureArgsTool) Parameters() map[string]interface{} { return map[string]interface{}{} }

func (t *captureArgsTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	t.runCount++
	t.gotArgs = make(map[string]string, len(args))
	for k, v := range args {
		t.gotArgs[k] = v
	}
	return t.result, nil, nil
}

func TestExecuteCoreWithContext_DefaultsPathAndNormalizesEmptyOutput(t *testing.T) {
	registry := NewRegistry()
	tool := &captureArgsTool{name: "list_dir"}
	registry.Register(tool)

	tc := &ToolCall{Tool: "list_dir", Args: map[string]string{}}
	got, change, isError := executeCoreWithContext(ExecutionContext{
		Context:  context.Background(),
		Registry: registry,
		Stdout:   io.Discard,
		Stderr:   io.Discard,
	}, tc)

	if change != nil {
		t.Fatalf("executeCoreWithContext() returned unexpected change: %+v", change)
	}
	if got != "(no output)" {
		t.Fatalf("executeCoreWithContext() = %q, want %q", got, "(no output)")
	}
	if isError {
		t.Fatalf("executeCoreWithContext() isError = true, want false")
	}
	if tc.Args["path"] != "." {
		t.Fatalf("tc.Args[path] = %q, want %q", tc.Args["path"], ".")
	}
	if tool.gotArgs["path"] != "." {
		t.Fatalf("tool got path %q, want %q", tool.gotArgs["path"], ".")
	}
}

func TestInvalidateToolCache_ByToolKind(t *testing.T) {
	writeAbs, err := filepath.Abs("tmp/output.txt")
	if err != nil {
		t.Fatalf("filepath.Abs(write) error = %v", err)
	}
	deleteAbs, err := filepath.Abs("tmp/delete.txt")
	if err != nil {
		t.Fatalf("filepath.Abs(delete) error = %v", err)
	}
	copyAbs, err := filepath.Abs("tmp/copied.txt")
	if err != nil {
		t.Fatalf("filepath.Abs(copy) error = %v", err)
	}
	dirAbs, err := filepath.Abs("tmp/newdir")
	if err != nil {
		t.Fatalf("filepath.Abs(dir) error = %v", err)
	}

	tests := []struct {
		name           string
		tc             *ToolCall
		wantFiles      []string
		wantDirs       []string
		wantSearch     []string
		wantClearCount int
	}{
		{
			name:           "apply_patch clears all caches",
			tc:             &ToolCall{Tool: "apply_patch", Args: map[string]string{"patch": "*** Begin Patch"}},
			wantClearCount: 1,
		},
		{
			name:       "write_file invalidates file and search cache",
			tc:         &ToolCall{Tool: "write_file", Args: map[string]string{"path": "tmp/output.txt"}},
			wantFiles:  []string{writeAbs},
			wantSearch: []string{writeAbs},
		},
		{
			name:       "delete_file invalidates file dir and search cache",
			tc:         &ToolCall{Tool: "delete_file", Args: map[string]string{"path": "tmp/delete.txt"}},
			wantFiles:  []string{deleteAbs},
			wantDirs:   []string{filepath.Dir(deleteAbs)},
			wantSearch: []string{deleteAbs},
		},
		{
			name:       "copy_file invalidates destination dir and search cache",
			tc:         &ToolCall{Tool: "copy_file", Args: map[string]string{"dest": "tmp/copied.txt"}},
			wantDirs:   []string{filepath.Dir(copyAbs)},
			wantSearch: []string{copyAbs},
		},
		{
			name:     "create_dir invalidates parent dir only",
			tc:       &ToolCall{Tool: "create_dir", Args: map[string]string{"path": "tmp/newdir"}},
			wantDirs: []string{filepath.Dir(dirAbs)},
		},
		{
			name:           "git_checkout clears all caches",
			tc:             &ToolCall{Tool: "git_checkout", Args: map[string]string{}},
			wantClearCount: 1,
		},
		{
			name:           "bash write command clears all caches",
			tc:             &ToolCall{Tool: "bash", Args: map[string]string{"command": "go build ./..."}},
			wantClearCount: 1,
		},
		{
			name: "bash read only command keeps cache",
			tc:   &ToolCall{Tool: "bash", Args: map[string]string{"command": "go test ./..."}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &recordingToolCache{}
			invalidateToolCache(ExecutionContext{ToolCache: cache}, tt.tc)

			if got := strings.Join(cache.invalidatedFiles, ","); got != strings.Join(tt.wantFiles, ",") {
				t.Fatalf("invalidated files = %v, want %v", cache.invalidatedFiles, tt.wantFiles)
			}
			if got := strings.Join(cache.invalidatedDirs, ","); got != strings.Join(tt.wantDirs, ",") {
				t.Fatalf("invalidated dirs = %v, want %v", cache.invalidatedDirs, tt.wantDirs)
			}
			if got := strings.Join(cache.invalidatedSearch, ","); got != strings.Join(tt.wantSearch, ",") {
				t.Fatalf("invalidated search = %v, want %v", cache.invalidatedSearch, tt.wantSearch)
			}
			if cache.clearCount != tt.wantClearCount {
				t.Fatalf("clearCount = %d, want %d", cache.clearCount, tt.wantClearCount)
			}
		})
	}
}

func TestPreviewReadFilePathsAndFormatReadFilePreviewArg(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]string
		wantPaths []string
		wantLabel string
	}{
		{
			name:      "uses JSON paths array",
			args:      map[string]string{"paths": `["a.txt","b.txt"]`},
			wantPaths: []string{"a.txt", "b.txt"},
			wantLabel: "Files: 2",
		},
		{
			name:      "falls back to singular path when JSON is invalid",
			args:      map[string]string{"paths": `not-json`, "path": "main.go"},
			wantPaths: []string{"main.go"},
			wantLabel: "File: main.go",
		},
		{
			name:      "returns none when no path is present",
			args:      map[string]string{},
			wantPaths: nil,
			wantLabel: "Files: (none)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := previewReadFilePaths(tt.args); strings.Join(got, ",") != strings.Join(tt.wantPaths, ",") {
				t.Fatalf("previewReadFilePaths() = %v, want %v", got, tt.wantPaths)
			}
			if got := formatReadFilePreviewArg(tt.args); got != tt.wantLabel {
				t.Fatalf("formatReadFilePreviewArg() = %q, want %q", got, tt.wantLabel)
			}
		})
	}
}

func TestPrintToolArgs_CoversSpecializedBranches(t *testing.T) {
	tests := []struct {
		name string
		tc   *ToolCall
		want []string
	}{
		{
			name: "search_code prefers file_pattern fallback",
			tc: &ToolCall{Tool: "search_code", Args: map[string]string{
				"pattern":      "Handle",
				"path":         "internal",
				"file_pattern": "*.go",
			}},
			want: []string{"Pattern: Handle", "Path: internal", "File pattern: *.go"},
		},
		{
			name: "lint shows default path and auto fix",
			tc: &ToolCall{Tool: "lint", Args: map[string]string{
				"auto_fix": "true",
			}},
			want: []string{"Path: .", "Auto-fix: enabled"},
		},
		{
			name: "copy_file shows source and destination",
			tc: &ToolCall{Tool: "copy_file", Args: map[string]string{
				"src":  "a.txt",
				"dest": "b.txt",
			}},
			want: []string{"Source: a.txt", "Destination: b.txt"},
		},
		{
			name: "delete_file shows target file",
			tc:   &ToolCall{Tool: "delete_file", Args: map[string]string{"path": "old.txt"}},
			want: []string{"File: old.txt"},
		},
		{
			name: "web_search shows query",
			tc:   &ToolCall{Tool: "web_search", Args: map[string]string{"query": "golang coverage"}},
			want: []string{"Query: golang coverage"},
		},
		{
			name: "default branch prints arbitrary args",
			tc:   &ToolCall{Tool: "custom_tool", Args: map[string]string{"id": "42"}},
			want: []string{"id: 42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printToolArgs(&buf, tt.tc)

			for _, want := range tt.want {
				if !strings.Contains(buf.String(), want) {
					t.Fatalf("printToolArgs() output missing %q in %q", want, buf.String())
				}
			}
		})
	}
}

func TestDisplayCollapsedOutput_HandlesShortStatusAndMultilineBody(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	t.Run("single line status stays inline", func(t *testing.T) {
		var buf bytes.Buffer
		displayCollapsedOutput(&buf, "Successfully wrote file", config.DefaultConfig())
		if got := buf.String(); !strings.Contains(got, "⎿  Successfully wrote file") {
			t.Fatalf("displayCollapsedOutput() = %q, want inline status", got)
		}
	})

	t.Run("multiline output is formatted", func(t *testing.T) {
		var buf bytes.Buffer
		displayCollapsedOutput(&buf, "line1\nline2\nline3", config.DefaultConfig())
		got := buf.String()
		if !strings.Contains(got, "line1") || !strings.Contains(got, "line2") {
			t.Fatalf("displayCollapsedOutput() should keep multiline content, got %q", got)
		}
	})
}
