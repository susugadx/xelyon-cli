package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func withChangeStorageHooks(t *testing.T) {
	t.Helper()

	oldUserHomeDir := userHomeDirForChanges
	t.Cleanup(func() {
		userHomeDirForChanges = oldUserHomeDir
	})
}

func TestNewChangeStorage_MkdirError(t *testing.T) {
	homeFile := filepath.Join(t.TempDir(), "home-file")
	if err := os.WriteFile(homeFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("HOME", homeFile)

	if _, err := NewChangeStorage(); err == nil || !strings.Contains(err.Error(), "failed to create changes directory") {
		t.Fatalf("NewChangeStorage() error = %v, want create changes directory error", err)
	}
}

func TestNewChangeStorage_UserHomeDirError(t *testing.T) {
	withChangeStorageHooks(t)
	userHomeDirForChanges = func() (string, error) {
		return "", errors.New("home lookup failed")
	}

	if _, err := NewChangeStorage(); err == nil || !strings.Contains(err.Error(), "failed to get home directory") {
		t.Fatalf("NewChangeStorage() error = %v, want get home directory error", err)
	}
}

func TestAppendChange_OpenError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cs := &ChangeStorage{changesPath: blocker}
	err := cs.AppendChange("session-open-error", tools.FileChange{
		FilePath:    "/tmp/test.txt",
		Timestamp:   time.Now(),
		Tool:        "write_file",
		Description: "open should fail",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to open changes file") {
		t.Fatalf("AppendChange() error = %v, want open changes file error", err)
	}
}

func TestLoadSessionChanges_OpenAndScanErrors(t *testing.T) {
	t.Run("open error is returned", func(t *testing.T) {
		changesDir := t.TempDir()
		sessionPath := filepath.Join(changesDir, "changes_session-open.jsonl")
		if err := os.MkdirAll(sessionPath, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		cs := &ChangeStorage{changesPath: changesDir}
		if _, err := cs.LoadSessionChanges("session-open"); err == nil || !strings.Contains(err.Error(), "failed to read changes file") {
			t.Fatalf("LoadSessionChanges() error = %v, want read changes file error", err)
		}
	})

	t.Run("scanner error is returned", func(t *testing.T) {
		changesDir := t.TempDir()
		cs := &ChangeStorage{changesPath: changesDir}
		sessionID := "session-scan"
		filename := filepath.Join(changesDir, "changes_"+sessionID+".jsonl")
		longLine := strings.Repeat("x", 70*1024)
		if err := os.WriteFile(filename, []byte(longLine), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		if _, err := cs.LoadSessionChanges(sessionID); err == nil || !strings.Contains(err.Error(), "failed to read changes file") {
			t.Fatalf("LoadSessionChanges() error = %v, want read changes file error", err)
		}
	})
}

func TestListSessionsAndCleanupOldChanges_ErrorTolerantPaths(t *testing.T) {
	t.Run("list sessions skips unreadable or invalid change files", func(t *testing.T) {
		changesDir := t.TempDir()
		cs := &ChangeStorage{changesPath: changesDir}

		validSession := "valid"
		valid := tools.FileChange{
			FilePath:    "/tmp/test.txt",
			Timestamp:   time.Now(),
			Tool:        "write_file",
			Description: "valid",
		}
		if err := cs.AppendChange(validSession, valid); err != nil {
			t.Fatalf("AppendChange(valid) error = %v", err)
		}

		badDir := filepath.Join(changesDir, "changes_baddir.jsonl")
		if err := os.MkdirAll(badDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(badDir) error = %v", err)
		}

		emptyFile := filepath.Join(changesDir, "changes_empty.jsonl")
		if err := os.WriteFile(emptyFile, nil, 0o600); err != nil {
			t.Fatalf("WriteFile(empty) error = %v", err)
		}

		sessions, err := cs.ListSessions()
		if err != nil {
			t.Fatalf("ListSessions() error = %v", err)
		}
		if len(sessions) != 1 || sessions[0].SessionID != validSession {
			t.Fatalf("ListSessions() = %#v, want only valid session", sessions)
		}
	})

	t.Run("cleanup skips stat and remove failures", func(t *testing.T) {
		changesDir := t.TempDir()
		cs := &ChangeStorage{changesPath: changesDir}

		oldDir := filepath.Join(changesDir, "changes_olddir.jsonl")
		if err := os.MkdirAll(oldDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(oldDir) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(oldDir, "child"), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile(oldDir/child) error = %v", err)
		}
		oldTime := time.Now().AddDate(0, 0, -40)
		if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
			t.Fatalf("Chtimes(oldDir) error = %v", err)
		}

		staleLink := filepath.Join(changesDir, "changes_broken.jsonl")
		if err := os.Symlink(filepath.Join(changesDir, "missing-target"), staleLink); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}

		recentFile := filepath.Join(changesDir, "changes_recent.jsonl")
		if err := os.WriteFile(recentFile, []byte("{}\n"), 0o600); err != nil {
			t.Fatalf("WriteFile(recent) error = %v", err)
		}

		deleted, err := cs.CleanupOldChanges(30)
		if err != nil {
			t.Fatalf("CleanupOldChanges() error = %v", err)
		}
		if deleted != 0 {
			t.Fatalf("CleanupOldChanges() deleted %d files, want 0 when only failing removals are old", deleted)
		}
		if _, err := os.Stat(recentFile); err != nil {
			t.Fatalf("recent file should remain, stat error = %v", err)
		}
	})
}

func TestCleanupOldChanges_DefaultDaysFallback(t *testing.T) {
	changesDir := t.TempDir()
	cs := &ChangeStorage{changesPath: changesDir}

	oldFile := filepath.Join(changesDir, "changes_old.jsonl")
	if err := os.WriteFile(oldFile, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(old) error = %v", err)
	}
	oldTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(old) error = %v", err)
	}

	deleted, err := cs.CleanupOldChanges(0)
	if err != nil {
		t.Fatalf("CleanupOldChanges(0) error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("CleanupOldChanges(0) deleted %d files, want 1 using default retention", deleted)
	}
}
