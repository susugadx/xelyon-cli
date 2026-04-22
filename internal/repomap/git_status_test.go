package repomap

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseGitStatusOutput(t *testing.T) {
	output := " M internal/agent/agent.go\n" +
		"A  internal/repomap/new_file.go\n" +
		"R  old/name.go -> internal/repomap/name.go\n" +
		"?? internal/repomap/untracked.go\n" +
		"\n" +
		"X \n" +
		"bad"

	got := parseGitStatusOutput(output)
	want := []GitChange{
		{Status: "M", Path: filepath.ToSlash("internal/agent/agent.go")},
		{Status: "A", Path: filepath.ToSlash("internal/repomap/new_file.go")},
		{Status: "R", Path: filepath.ToSlash("internal/repomap/name.go")},
		{Status: "??", Path: filepath.ToSlash("internal/repomap/untracked.go")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseGitStatusOutput() = %#v, want %#v", got, want)
	}
}
