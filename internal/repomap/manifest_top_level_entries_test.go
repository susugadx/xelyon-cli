package repomap

import (
	"reflect"
	"testing"
)

func TestCollectTopLevelEntries_SplitsDirsAndFiles(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			nil,
			{Path: ""},
			{Path: "README.md"},
			{Path: "main.go"},
			{Path: "internal/agent/compress.go"},
			{Path: "internal/config/project.go"},
			{Path: "cmd/xelyon/main.go"},
		},
	}

	dirs, files := pm.collectTopLevelEntries()

	if !reflect.DeepEqual(dirs, []string{"cmd/", "internal/"}) {
		t.Fatalf("collectTopLevelEntries() dirs = %v, want [cmd/ internal/]", dirs)
	}
	if !reflect.DeepEqual(files, []string{"README.md", "main.go"}) {
		t.Fatalf("collectTopLevelEntries() files = %v, want [README.md main.go]", files)
	}
}
