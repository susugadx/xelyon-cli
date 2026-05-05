package review

import "testing"

func TestHostReadOnlyCommandPolicy_AllowsKnownReadOnlyCommands(t *testing.T) {
	type testCase struct {
		name    string
		command string
		args    []string
	}

	tests := []testCase{
		{name: "git status", command: "git", args: []string{"status", "--short"}},
		{name: "git global option + status", command: "git", args: []string{"--no-optional-locks", "status", "--short"}},
		{name: "git diff", command: "git", args: []string{"diff"}},
		{name: "git diff path separator", command: "git", args: []string{"diff", "--", "path/to/file"}},
		{name: "find empty args", command: "find", args: nil},
		{name: "find current dir", command: "find", args: []string{".", "-name", "*.go"}},
		{name: "find internal dir", command: "find", args: []string{"internal", "-name", "*.go"}},
		{name: "find option separator internal", command: "find", args: []string{"--", "internal", "-name", "*.go"}},
		{name: "find option separator with expression-like first path", command: "find", args: []string{"--", "-L", "internal", "-name", "*.go"}},
		{name: "find expression operand starts with dash", command: "find", args: []string{"-name", "-L"}},
		{name: "cat file", command: "cat", args: []string{"file.txt"}},
		{name: "go test", command: "go", args: []string{"test", "./..."}},
		{name: "go test json", command: "go", args: []string{"test", "-json", "./..."}},
	}
	for _, sc := range hostReadOnlySearchCommandPolicyAllowScenarios() {
		tests = append(tests, testCase(sc))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHostReadOnlyCommandPolicy(tt.command, tt.args); err != nil {
				t.Fatalf("validateHostReadOnlyCommandPolicy(%q, %#v) error = %v", tt.command, tt.args, err)
			}
		})
	}
}
