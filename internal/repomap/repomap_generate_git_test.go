package repomap

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGenerate_WithGitStatus(t *testing.T) {
	requireRipgrep(t)

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init")
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nfunc Build() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	output := pm.Generate()
	if !strings.Contains(output, "## Uncommitted Changes") {
		t.Fatalf("expected git status section:\n%s", output)
	}
	if !strings.Contains(output, "?? main.go") {
		t.Fatalf("expected untracked file in git status:\n%s", output)
	}
}
