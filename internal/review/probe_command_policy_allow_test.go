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
		{name: "sed numeric print", command: "sed", args: []string{"-n", "1,120p", "internal/file.go"}},
		{name: "go test", command: "go", args: []string{"test", "./..."}},
		{name: "go test package path", command: "go", args: []string{"test", "./internal/review"}},
		{name: "go test json", command: "go", args: []string{"test", "-json", "./..."}},
		{name: "npm test", command: "npm", args: []string{"test"}},
		{name: "npm run lint", command: "npm", args: []string{"run", "lint"}},
		{name: "npm test script args after separator", command: "npm", args: []string{"test", "--", "--prefix", "/tmp/project"}},
		{name: "cargo test", command: "cargo", args: []string{"test"}},
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
