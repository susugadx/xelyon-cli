package probe

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHostReadOnlyCommandPathPolicy_BlocksSymlinkEscape(t *testing.T) {
	repoRoot := t.TempDir()
	linkPath := filepath.Join(repoRoot, "hosts_link")
	if err := os.Symlink("/etc/hosts", linkPath); err != nil {
		t.Skipf("symlink is not supported: %v", err)
	}

	err := validateHostReadOnlyCommandPathPolicy(repoRoot, repoRoot, "cat", []string{"hosts_link"})
	if err == nil {
		t.Fatal("validateHostReadOnlyCommandPathPolicy(cat hosts_link) error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
		t.Fatalf("validateHostReadOnlyCommandPathPolicy(cat hosts_link) error = %v, want ErrHostReadOnlyOutsideRepoPath", err)
	}
}
