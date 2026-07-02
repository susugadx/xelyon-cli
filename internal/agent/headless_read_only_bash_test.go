package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadlessReadOnlyDeniesAllBashAndSummarizesCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		setup   func(t *testing.T, dir string)
		assert  func(t *testing.T, dir string)
	}{
		{
			name:    "read_only_like_command",
			command: "pwd",
		},
		{
			name:    "find_delete",
			command: "find . -delete",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			assert: func(t *testing.T, dir string) {
				t.Helper()
				if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
					t.Fatalf("keep.txt stat error = %v, want file preserved", err)
				}
			},
		},
		{
			name:    "command_substitution_touch",
			command: "echo $(touch target.txt)",
			assert: func(t *testing.T, dir string) {
				t.Helper()
				if _, err := os.Stat(filepath.Join(dir, "target.txt")); !os.IsNotExist(err) {
					t.Fatalf("target.txt stat error = %v, want absent file", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			dir := testSubDir(t)
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			provider := &sequenceMockProvider{
				name: "openai",
				responses: []string{
					headlessToolCallJSON(t, "bash", map[string]string{"command": tt.command}),
					"final response after denied bash",
				},
			}
			result := RunHeadlessWithConfigOptions(context.Background(), "attempt bash", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
				ReadOnly: true,
			})

			if result.Status != HeadlessStatusSuccess {
				t.Fatalf("Status = %q, want success without FailOnToolError: %+v", result.Status, result.Error)
			}
			if len(result.ToolCalls) != 1 || result.ToolCalls[0].Success {
				t.Fatalf("ToolCalls = %+v, want one failed bash call", result.ToolCalls)
			}
			if !strings.Contains(result.ToolCalls[0].Output, "read-only mode denied bash tool") {
				t.Fatalf("ToolCalls[0].Output = %q, want bash read-only denial", result.ToolCalls[0].Output)
			}
			if tt.assert != nil {
				tt.assert(t, dir)
			}
			if result.Summary == nil || len(result.Summary.Commands) != 1 {
				t.Fatalf("Summary = %+v, want one denied command", result.Summary)
			}
			summary := result.Summary.Commands[0]
			if summary.Command != tt.command || summary.Status != headlessSummaryStatusFailed || summary.ExitCode != -1 || summary.Source != headlessCommandSourceTool {
				t.Fatalf("command summary = %+v, want failed/-1 tool summary", summary)
			}
		})
	}
}
