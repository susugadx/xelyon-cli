package review

import (
	"errors"
	"strings"
	"testing"
)

func TestHostReadOnlyCommandPolicy_BlocksDangerousCommandsAndArgs(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		args          []string
		errorContains string
	}{
		{name: "find delete", command: "find", args: []string{".", "-delete"}, errorContains: "find argument -delete"},
		{name: "find exec", command: "find", args: []string{".", "-exec", "sh", "-c", "true", ";"}, errorContains: "find argument -exec"},
		{name: "find fprint0", command: "find", args: []string{".", "-fprint0", "out.txt"}, errorContains: "find argument -fprint0"},
		{name: "find files0-from", command: "find", args: []string{"-files0-from", "/etc/passwd", "-maxdepth", "0"}, errorContains: "find argument -files0-from"},
		{name: "find leading option L", command: "find", args: []string{"-L", "/etc", "-name", "hosts"}, errorContains: "find leading option -L"},
		{name: "find leading option H", command: "find", args: []string{"-H", "/etc", "-name", "hosts"}, errorContains: "find leading option -H"},
		{name: "find leading option P", command: "find", args: []string{"-P", "/etc", "-name", "hosts"}, errorContains: "find leading option -P"},
		{name: "find leading option O", command: "find", args: []string{"-O3", "/etc", "-name", "hosts"}, errorContains: "find leading option -O3"},
		{name: "find leading option D", command: "find", args: []string{"-D", "tree", "/etc", "-name", "hosts"}, errorContains: "find leading option -D"},
		{name: "find follow expression", command: "find", args: []string{".", "-follow", "-name", "*.go"}, errorContains: "find argument -follow"},
		{name: "find newer reference", command: "find", args: []string{".", "-newer", "/etc/passwd"}, errorContains: "find argument -newer"},
		{name: "find samefile reference", command: "find", args: []string{".", "-samefile", "/etc/passwd"}, errorContains: "find argument -samefile"},
		{name: "git ext diff", command: "git", args: []string{"diff", "--ext-diff"}, errorContains: "git argument --ext-diff"},
		{name: "git diff no-index", command: "git", args: []string{"diff", "--no-index", "/etc/passwd", "/etc/group"}, errorContains: "git argument --no-index"},
		{name: "git global separator before subcommand", command: "git", args: []string{"--", "diff"}, errorContains: "git global separator -- is not allowed before subcommand"},
		{name: "git config override", command: "git", args: []string{"-c", "core.pager=cat", "diff"}, errorContains: "git config override"},
		{name: "git grep open in pager", command: "git", args: []string{"grep", "-O", "pattern"}, errorContains: "git argument -O"},
		{name: "git grep open in pager short cluster", command: "git", args: []string{"grep", "-nOcat", "pattern"}, errorContains: "git argument -nOcat"},
		{name: "git grep pattern file", command: "git", args: []string{"grep", "-f", "/etc/patterns"}, errorContains: "git argument -f"},
		{name: "git grep pattern file short cluster", command: "git", args: []string{"grep", "-nf/etc/patterns", "pattern"}, errorContains: "git argument -nf/etc/patterns"},
		{name: "git textconv", command: "git", args: []string{"grep", "--textconv", "pattern"}, errorContains: "git argument --textconv"},
		{name: "git recurse submodules", command: "git", args: []string{"grep", "--recurse-submodules", "pattern"}, errorContains: "git argument --recurse-submodules"},
		{name: "rg pre", command: "rg", args: []string{"--pre", "cat", "pattern"}, errorContains: "rg argument --pre"},
		{name: "rg follow", command: "rg", args: []string{"--follow", "pattern"}, errorContains: "rg argument --follow"},
		{name: "rg short follow", command: "rg", args: []string{"-L", "pattern"}, errorContains: "rg argument -L"},
		{name: "rg short follow cluster", command: "rg", args: []string{"-nL", "pattern", "internal"}, errorContains: "rg argument -nL"},
		{name: "grep dereference recursive", command: "grep", args: []string{"-R", "pattern", "internal"}, errorContains: "grep argument -R"},
		{name: "grep long dereference recursive", command: "grep", args: []string{"--dereference-recursive", "pattern", "internal"}, errorContains: "grep argument --dereference-recursive"},
		{name: "ls symlink dereference", command: "ls", args: []string{"-L", "."}, errorContains: "ls argument -L"},
		{name: "git output equals", command: "git", args: []string{"diff", "--output=out.diff"}, errorContains: "git argument --output=out.diff"},
		{name: "git output separated", command: "git", args: []string{"show", "--output", "show.out"}, errorContains: "git argument --output"},
		{name: "sed script file", command: "sed", args: []string{"-n", "-f", "script.sed", "internal/file.go"}, errorContains: "sed argument -f"},
		{name: "sed write command", command: "sed", args: []string{"-n", "1w out.txt", "internal/file.go"}, errorContains: "read-only print command"},
		{name: "sed substitution execute flag", command: "sed", args: []string{"-n", "s/foo/bar/e", "internal/file.go"}, errorContains: "read-only print command"},
		{name: "go coverprofile", command: "go", args: []string{"test", "-coverprofile=cover.out", "./probe"}, errorContains: "go argument -coverprofile=cover.out"},
		{name: "go double dash coverprofile", command: "go", args: []string{"test", "--coverprofile=cover.out", "./probe"}, errorContains: "go argument --coverprofile=cover.out"},
		{name: "go double dash exec attached", command: "go", args: []string{"test", "--exec=/bin/echo", "./probe"}, errorContains: "go argument --exec=/bin/echo"},
		{name: "go double dash exec detached", command: "go", args: []string{"test", "--exec", "/bin/echo", "./probe"}, errorContains: "go argument --exec"},
		{name: "go vettool", command: "go", args: []string{"vet", "-vettool=./tool", "./probe"}, errorContains: "go argument -vettool=./tool"},
		{name: "go toolexec", command: "go", args: []string{"test", "-toolexec=./wrap.sh", "./probe"}, errorContains: "go argument -toolexec=./wrap.sh"},
		{name: "go double dash toolexec attached", command: "go", args: []string{"test", "--toolexec=./wrap.sh", "./probe"}, errorContains: "go argument --toolexec=./wrap.sh"},
		{name: "go double dash toolexec detached", command: "go", args: []string{"test", "--toolexec", "./wrap.sh", "./probe"}, errorContains: "go argument --toolexec"},
		{name: "go double dash vettool attached", command: "go", args: []string{"vet", "--vettool=./tool", "./probe"}, errorContains: "go argument --vettool=./tool"},
		{name: "go overlay", command: "go", args: []string{"test", "-overlay=/tmp/overlay.json", "./probe"}, errorContains: "go argument -overlay=/tmp/overlay.json"},
		{name: "go double dash overlay", command: "go", args: []string{"test", "--overlay=/tmp/overlay.json", "./probe"}, errorContains: "go argument --overlay=/tmp/overlay.json"},
		{name: "npm prefix", command: "npm", args: []string{"test", "--prefix", "/tmp/project"}, errorContains: "npm argument --prefix"},
		{name: "npm script shell", command: "npm", args: []string{"run", "lint", "--script-shell=/tmp/sh"}, errorContains: "npm argument --script-shell=/tmp/sh"},
		{name: "cargo manifest path", command: "cargo", args: []string{"test", "--manifest-path", "/tmp/Cargo.toml"}, errorContains: "cargo argument --manifest-path"},
		{name: "cargo config", command: "cargo", args: []string{"clippy", "--config", "build.rustc-wrapper=\"/tmp/wrapper\""}, errorContains: "cargo argument --config"},
		{name: "command path", command: "./git", args: []string{"status", "--short"}, errorContains: "command path is not allowed"},
		{name: "cat option", command: "cat", args: []string{"-n", "file.txt"}, errorContains: "cat option -n"},
		{name: "cat stdin", command: "cat", args: []string{"-"}, errorContains: "cat argument -"},
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
