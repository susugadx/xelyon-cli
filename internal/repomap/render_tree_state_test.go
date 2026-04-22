package repomap

import (
	"reflect"
	"testing"
)

func TestBuildRenderTreeState_SortsDirectoriesAndFiles(t *testing.T) {
	files := []*FileEntry{
		{Path: "b/main.go"},
		{Path: "a/z.go"},
		{Path: "root.go"},
		{Path: "a/a.go"},
	}
	options := []renderOption{
		{include: true},
		{include: true},
		{include: true},
		{include: true},
	}

	state := buildRenderTreeState(files, options)
	if !reflect.DeepEqual(state.dirs, []string{"./", "a/", "b/"}) {
		t.Fatalf("dirs = %#v, want [./ a/ b/]", state.dirs)
	}

	gotA := []string{state.grouped["a/"][0].Path, state.grouped["a/"][1].Path}
	if !reflect.DeepEqual(gotA, []string{"a/a.go", "a/z.go"}) {
		t.Fatalf("a/ files = %#v, want [a/a.go a/z.go]", gotA)
	}

	if idx, ok := state.pathIndex["root.go"]; !ok || idx != 2 {
		t.Fatalf("pathIndex[root.go] = (%d, %v), want (2, true)", idx, ok)
	}
}

func TestRenderDirectoryName(t *testing.T) {
	if got := renderDirectoryName("root.go"); got != "./" {
		t.Fatalf("renderDirectoryName(root.go) = %q, want ./", got)
	}
	if got := renderDirectoryName("pkg/main.go"); got != "pkg/" {
		t.Fatalf("renderDirectoryName(pkg/main.go) = %q, want pkg/", got)
	}
}
