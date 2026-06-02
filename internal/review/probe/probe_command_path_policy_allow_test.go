package probe

import (
	"path/filepath"
	"testing"
)

func TestHostReadOnlyCommandPathPolicy_AllowsRepoPaths(t *testing.T) {
	repoRoot := t.TempDir()
	workDir := filepath.Join(repoRoot, "internal")

	tests := []struct {
		name    string
		command string
		args    []string
		workDir string
	}{
		{name: "cat file", command: "cat", args: []string{"file.txt"}},
		{name: "ls empty args", command: "ls", args: nil},
		{name: "ls internal", command: "ls", args: []string{"internal"}},
		{name: "find empty args treated as current dir", command: "find", args: nil},
		{name: "find current dir", command: "find", args: []string{".", "-name", "*.go"}},
		{name: "find internal dir", command: "find", args: []string{"internal", "-name", "*.go"}},
		{name: "find with option separator internal", command: "find", args: []string{"--", "internal", "-name", "*.go"}},
		{name: "find option separator with expression-like first path", command: "find", args: []string{"--", "-L", "internal", "-name", "*.go"}},
		{name: "git diff with path after separator", command: "git", args: []string{"diff", "--", "internal/review"}},
		{name: "git global option status", command: "git", args: []string{"--no-optional-locks", "status", "--short"}},
		{name: "go package path inside repo", command: "go", args: []string{"test", "./internal/review"}},
		{name: "npm path policy excluded", command: "npm", args: []string{"test"}},
		{name: "cargo path policy excluded", command: "cargo", args: []string{"test"}},
		{name: "sed file operand inside repo", command: "sed", args: []string{"-n", "1p", "internal/file.go"}},
		{name: "sed expression option with file operand inside repo", command: "sed", args: []string{"-n", "-e", "1p", "internal/file.go"}},
		{name: "cat relative to workdir", command: "cat", args: []string{"file.txt"}, workDir: workDir},
	}
	for _, sc := range hostReadOnlySearchPathPolicyAllowScenarios(repoRoot) {
		tests = append(tests, struct {
			name    string
			command string
			args    []string
			workDir string
		}{
			name:    sc.name,
			command: sc.command,
			args:    sc.args,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd := repoRoot
			if tt.workDir != "" {
				wd = tt.workDir
			}

			err := validateHostReadOnlyCommandPathPolicy(repoRoot, wd, tt.command, tt.args)
			if err != nil {
				t.Fatalf("validateHostReadOnlyCommandPathPolicy(%q, %q, %#v) error = %v", tt.command, wd, tt.args, err)
			}
		})
	}
}
