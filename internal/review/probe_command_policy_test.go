package review

import (
	"errors"
	"strings"
	"testing"
)

func TestHostReadOnlyCommandPolicy_AllowsKnownReadOnlyCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{name: "git status", command: "git", args: []string{"status", "--short"}},
		{name: "git global option + status", command: "git", args: []string{"--no-optional-locks", "status", "--short"}},
		{name: "git diff", command: "git", args: []string{"diff"}},
		{name: "git diff path separator", command: "git", args: []string{"diff", "--", "path/to/file"}},
		{name: "rg pattern", command: "rg", args: []string{"pattern"}},
		{name: "cat file", command: "cat", args: []string{"file.txt"}},
		{name: "go test", command: "go", args: []string{"test", "./..."}},
		{name: "go test json", command: "go", args: []string{"test", "-json", "./..."}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHostReadOnlyCommandPolicy(tt.command, tt.args); err != nil {
				t.Fatalf("validateHostReadOnlyCommandPolicy(%q, %#v) error = %v", tt.command, tt.args, err)
			}
		})
	}
}

func TestHostReadOnlyCommandPolicy_BlocksDangerousCommandsAndArgs(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		args          []string
		errorContains string
	}{
		{
			name:          "find delete",
			command:       "find",
			args:          []string{".", "-delete"},
			errorContains: "find argument -delete",
		},
		{
			name:          "find exec",
			command:       "find",
			args:          []string{".", "-exec", "sh", "-c", "true", ";"},
			errorContains: "find argument -exec",
		},
		{
			name:          "find fprint0",
			command:       "find",
			args:          []string{".", "-fprint0", "out.txt"},
			errorContains: "find argument -fprint0",
		},
		{
			name:          "git ext diff",
			command:       "git",
			args:          []string{"diff", "--ext-diff"},
			errorContains: "git argument --ext-diff",
		},
		{
			name:          "git global separator before subcommand",
			command:       "git",
			args:          []string{"--", "diff"},
			errorContains: "git global separator -- is not allowed before subcommand",
		},
		{
			name:          "git config override",
			command:       "git",
			args:          []string{"-c", "core.pager=cat", "diff"},
			errorContains: "git config override",
		},
		{
			name:          "rg pre",
			command:       "rg",
			args:          []string{"--pre", "cat", "pattern"},
			errorContains: "rg argument --pre",
		},
		{
			name:          "git output equals",
			command:       "git",
			args:          []string{"diff", "--output=out.diff"},
			errorContains: "git argument --output=out.diff",
		},
		{
			name:          "git output separated",
			command:       "git",
			args:          []string{"show", "--output", "show.out"},
			errorContains: "git argument --output",
		},
		{
			name:          "go coverprofile",
			command:       "go",
			args:          []string{"test", "-coverprofile=cover.out", "./probe"},
			errorContains: "go argument -coverprofile=cover.out",
		},
		{
			name:          "go vettool",
			command:       "go",
			args:          []string{"vet", "-vettool=./tool", "./probe"},
			errorContains: "go argument -vettool=./tool",
		},
		{
			name:          "go toolexec",
			command:       "go",
			args:          []string{"test", "-toolexec=./wrap.sh", "./probe"},
			errorContains: "go argument -toolexec=./wrap.sh",
		},
		{
			name:          "command path",
			command:       "./git",
			args:          []string{"status", "--short"},
			errorContains: "command path is not allowed",
		},
		{
			name:          "cat option",
			command:       "cat",
			args:          []string{"-n", "file.txt"},
			errorContains: "cat option -n",
		},
		{
			name:          "cat stdin",
			command:       "cat",
			args:          []string{"-"},
			errorContains: "cat argument -",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostReadOnlyCommandPolicy(tt.command, tt.args)
			if err == nil {
				t.Fatalf("validateHostReadOnlyCommandPolicy(%q, %#v) error = nil", tt.command, tt.args)
			}
			if !errors.Is(err, ErrHostReadOnlyBlocked) {
				t.Fatalf("validateHostReadOnlyCommandPolicy(%q, %#v) error = %v, want ErrHostReadOnlyBlocked", tt.command, tt.args, err)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("validateHostReadOnlyCommandPolicy(%q, %#v) error = %q, want to contain %q", tt.command, tt.args, err.Error(), tt.errorContains)
			}
		})
	}
}
