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
		{name: "find empty args", command: "find", args: nil},
		{name: "find current dir", command: "find", args: []string{".", "-name", "*.go"}},
		{name: "find internal dir", command: "find", args: []string{"internal", "-name", "*.go"}},
		{name: "find option separator internal", command: "find", args: []string{"--", "internal", "-name", "*.go"}},
		{name: "find option separator with expression-like first path", command: "find", args: []string{"--", "-L", "internal", "-name", "*.go"}},
		{name: "find expression operand starts with dash", command: "find", args: []string{"-name", "-L"}},
		{name: "rg pattern", command: "rg", args: []string{"pattern"}},
		{name: "rg absolute-like positional pattern", command: "rg", args: []string{"/etc"}},
		{name: "rg pattern with repo path", command: "rg", args: []string{"pattern", "internal"}},
		{name: "rg pattern with separator and repo path", command: "rg", args: []string{"pattern", "--", "internal"}},
		{name: "rg regexp option with absolute-like pattern", command: "rg", args: []string{"-e", "/etc", "--", "internal"}},
		{name: "rg regexp equals with absolute-like pattern", command: "rg", args: []string{"--regexp=/etc", "--", "internal"}},
		{name: "grep pattern with separator and repo path", command: "grep", args: []string{"pattern", "--", "internal/file.go"}},
		{name: "grep traversal-like positional pattern", command: "grep", args: []string{"../todo", "internal/file.go"}},
		{name: "grep regexp option with absolute-like pattern", command: "grep", args: []string{"-e", "/etc", "--", "internal/file.go"}},
		{name: "grep regexp equals with absolute-like pattern", command: "grep", args: []string{"--regexp=/etc", "--", "internal/file.go"}},
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
			name:          "find leading option L",
			command:       "find",
			args:          []string{"-L", "/etc", "-name", "hosts"},
			errorContains: "find leading option -L",
		},
		{
			name:          "find leading option H",
			command:       "find",
			args:          []string{"-H", "/etc", "-name", "hosts"},
			errorContains: "find leading option -H",
		},
		{
			name:          "find leading option P",
			command:       "find",
			args:          []string{"-P", "/etc", "-name", "hosts"},
			errorContains: "find leading option -P",
		},
		{
			name:          "find leading option O",
			command:       "find",
			args:          []string{"-O3", "/etc", "-name", "hosts"},
			errorContains: "find leading option -O3",
		},
		{
			name:          "find leading option D",
			command:       "find",
			args:          []string{"-D", "tree", "/etc", "-name", "hosts"},
			errorContains: "find leading option -D",
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
