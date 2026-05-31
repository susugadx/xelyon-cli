package probe

import (
	"errors"
	"testing"
)

func TestHostReadOnlyCommandPathPolicy_BlocksOutsidePaths(t *testing.T) {
	repoRoot := t.TempDir()

	tests := []struct {
		name    string
		command string
		args    []string
		workDir string
	}{
		{name: "cat absolute outside", command: "cat", args: []string{"/etc/hosts"}},
		{name: "cat parent outside", command: "cat", args: []string{"../outside.txt"}, workDir: repoRoot},
		{name: "ls absolute outside", command: "ls", args: []string{"/etc"}},
		{name: "find absolute outside", command: "find", args: []string{"/etc", "-name", "hosts"}},
		{name: "find option separator absolute outside", command: "find", args: []string{"--", "/etc", "-name", "hosts"}},
		{name: "find option separator expression-like first path with outside path", command: "find", args: []string{"--", "-L", "/etc", "-name", "hosts"}},
		{name: "find parent outside", command: "find", args: []string{"../outside", "-name", "*.go"}, workDir: repoRoot},
		{name: "git diff absolute outside after separator", command: "git", args: []string{"diff", "--", "/etc"}},
		{name: "sed file operand absolute outside", command: "sed", args: []string{"-n", "1p", "/etc/passwd"}},
		{name: "sed expression option file operand absolute outside", command: "sed", args: []string{"-n", "-e", "1p", "/etc/passwd"}},
		{name: "go package absolute outside", command: "go", args: []string{"test", "/tmp/pkg"}},
	}
	for _, sc := range hostReadOnlySearchPathPolicyBlockedOutsideScenarios() {
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
			if err == nil {
				t.Fatalf("validateHostReadOnlyCommandPathPolicy(%q, %q, %#v) error = nil", tt.command, wd, tt.args)
			}
			if !errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
				t.Fatalf("validateHostReadOnlyCommandPathPolicy(%q, %q, %#v) error = %v, want ErrHostReadOnlyOutsideRepoPath", tt.command, wd, tt.args, err)
			}
		})
	}
}
