package review

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectSearchCommandPathCandidates_RG(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "positional pattern only",
			args: []string{"/etc"},
			want: nil,
		},
		{
			name: "pattern and path",
			args: []string{"pattern", "internal"},
			want: []string{"internal"},
		},
		{
			name: "explicit regexp and path",
			args: []string{"-e", "pattern", "internal"},
			want: []string{"internal"},
		},
		{
			name: "regexp equals and separator path",
			args: []string{"--regexp=/etc", "--", "internal"},
			want: []string{"internal"},
		},
		{
			name: "path after separator",
			args: []string{"pattern", "--", "/etc"},
			want: []string{"/etc"},
		},
		{
			name: "pattern file with attached short option",
			args: []string{"-f/etc/patterns", "internal"},
			want: []string{filepath.FromSlash("/etc/patterns"), "internal"},
		},
		{
			name: "pattern file inside short option cluster",
			args: []string{"-nf/usr/share/patterns", "needle", "internal"},
			want: []string{filepath.FromSlash("/usr/share/patterns"), "needle", "internal"},
		},
		{
			name: "consume iglob operand before explicit pattern",
			args: []string{"--iglob", "*.go", "-e", "needle", "/etc"},
			want: []string{"/etc"},
		},
		{
			name: "ignore-file separated contributes path arg",
			args: []string{"--ignore-file", "/etc/ignore", "pattern"},
			want: []string{"/etc/ignore"},
		},
		{
			name: "ignore-file equals contributes path arg",
			args: []string{"--ignore-file=/etc/ignore", "pattern"},
			want: []string{"/etc/ignore"},
		},
		{
			name: "files mode treats positional as path",
			args: []string{"--files", "/etc"},
			want: []string{"/etc"},
		},
		{
			name: "post-separator regexp-like token is treated as positional path candidate",
			args: []string{"--", "--regexp", "/etc"},
			want: []string{"/etc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectSearchCommandPathCandidates("rg", tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("collectSearchCommandPathCandidates(rg, %#v) = %#v, want %#v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCollectSearchCommandPathCandidates_Grep(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "positional pattern should not be treated as path",
			args: []string{"../todo", "internal/file.go"},
			want: []string{"internal/file.go"},
		},
		{
			name: "explicit regexp with absolute path",
			args: []string{"-e", "todo", "/etc/passwd"},
			want: []string{"/etc/passwd"},
		},
		{
			name: "separator with first positional as pattern",
			args: []string{"--", "/etc", "internal/file.go"},
			want: []string{"internal/file.go"},
		},
		{
			name: "recursive with explicit pattern after path",
			args: []string{"-R", "/etc", "-e", "needle"},
			want: []string{"/etc"},
		},
		{
			name: "recursive keeps first positional as pattern",
			args: []string{"-R", "needle", "/etc"},
			want: []string{"/etc"},
		},
		{
			name: "recursive absolute-like pattern without path",
			args: []string{"-R", "/etc"},
			want: nil,
		},
		{
			name: "directories recurse long option keeps first positional as pattern",
			args: []string{"--directories=recurse", "needle", "/etc"},
			want: []string{"/etc"},
		},
		{
			name: "directories recurse short attached option keeps first positional as pattern",
			args: []string{"-drecurse", "needle", "/etc"},
			want: []string{"/etc"},
		},
		{
			name: "exclude-from separated contributes path arg",
			args: []string{"--exclude-from", "/etc/patterns", "needle"},
			want: []string{"/etc/patterns"},
		},
		{
			name: "exclude-from equals contributes path arg",
			args: []string{"--exclude-from=/etc/patterns", "needle"},
			want: []string{"/etc/patterns"},
		},
		{
			name: "pattern file inside short option cluster",
			args: []string{"-nf/usr/share/patterns", "needle", "internal"},
			want: []string{filepath.FromSlash("/usr/share/patterns"), "needle", "internal"},
		},
		{
			name: "post-separator short regexp-like token is treated as positional path candidate",
			args: []string{"--", "-e", "/etc"},
			want: []string{"/etc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectSearchCommandPathCandidates("grep", tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("collectSearchCommandPathCandidates(grep, %#v) = %#v, want %#v", tt.args, got, tt.want)
			}
		})
	}
}
