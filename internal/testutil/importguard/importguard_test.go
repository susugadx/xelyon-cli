package importguard

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestWalkPackageBoundaryFilesProductionFileSets(t *testing.T) {
	root := t.TempDir()
	writeImportGuardTestFile(t, root, "root.go")
	writeImportGuardTestFile(t, root, "root_test.go")
	writeImportGuardTestFile(t, root, filepath.Join("sub", "child.go"))
	writeImportGuardTestFile(t, root, filepath.Join("sub", "child_test.go"))
	writeImportGuardTestFile(t, root, filepath.Join("testdata", "ignored.go"))

	tests := []struct {
		name    string
		fileSet PackageBoundaryFileSet
		want    []string
	}{
		{
			name:    "production files are recursive",
			fileSet: PackageBoundaryProductionGoFiles,
			want: []string{
				"root.go",
				filepath.Join("sub", "child.go"),
			},
		},
		{
			name:    "root production files exclude subpackages",
			fileSet: PackageBoundaryRootProductionGoFiles,
			want: []string{
				"root.go",
			},
		},
		{
			name:    "all files include tests recursively",
			fileSet: PackageBoundaryAllGoFiles,
			want: []string{
				"root.go",
				"root_test.go",
				filepath.Join("sub", "child.go"),
				filepath.Join("sub", "child_test.go"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectImportGuardPackageBoundaryFiles(t, root, tt.fileSet)
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("files = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("files = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func collectImportGuardPackageBoundaryFiles(t *testing.T, root string, fileSet PackageBoundaryFileSet) []string {
	t.Helper()
	var files []string
	if err := walkPackageBoundaryFiles(root, fileSet, func(path string) error {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		t.Fatalf("walk package boundary files: %v", err)
	}
	return files
}

func writeImportGuardTestFile(t *testing.T, root, relPath string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("package sample\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
