package history

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestListResumeSessions_FiltersByWorkingDirAndIncludesLegacy(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	currentDir := filepath.Join(t.TempDir(), "current")
	otherDir := filepath.Join(t.TempDir(), "other")
	if err := saveResumeTestSession(storage, "current-model", currentDir, "current"); err != nil {
		t.Fatalf("save current session: %v", err)
	}
	if err := saveResumeTestSession(storage, "other-model", otherDir, "other"); err != nil {
		t.Fatalf("save other session: %v", err)
	}
	if err := saveResumeTestSession(storage, "legacy-model", "", "legacy"); err != nil {
		t.Fatalf("save legacy session: %v", err)
	}
	emptySession := NewSession("empty-model")
	emptySession.WorkingDir = currentDir
	if err := storage.Save(emptySession); err != nil {
		t.Fatalf("save empty session: %v", err)
	}

	filtered, err := storage.ListResumeSessions(ResumeListOptions{WorkingDir: currentDir})
	if err != nil {
		t.Fatalf("ListResumeSessions() error = %v", err)
	}
	if got := resumeSessionModels(filtered); !sameStringSet(got, []string{"legacy-model", "current-model"}) {
		t.Fatalf("filtered models = %#v, want legacy/current", got)
	}

	all, err := storage.ListResumeSessions(ResumeListOptions{WorkingDir: currentDir, All: true})
	if err != nil {
		t.Fatalf("ListResumeSessions(all) error = %v", err)
	}
	if got := resumeSessionModels(all); !sameStringSet(got, []string{"legacy-model", "other-model", "current-model"}) {
		t.Fatalf("all models = %#v, want all non-empty sessions", got)
	}
}

func TestGetLastResumeSession_NoSessionsUsesSentinelError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	_, err = storage.GetLastResumeSession(ResumeListOptions{})
	if !errors.Is(err, ErrNoResumeSessions) {
		t.Fatalf("GetLastResumeSession() error = %v, want ErrNoResumeSessions", err)
	}
}

func TestResumeListOptions_ExcludeSessionID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	currentDir := filepath.Join(t.TempDir(), "current")
	otherDir := filepath.Join(t.TempDir(), "other")
	baseTime := time.Now()
	excluded := newResumeTestSession("excluded-model", currentDir, "excluded", baseTime.Add(3*time.Second))
	included := newResumeTestSession("included-model", currentDir, "included", baseTime.Add(2*time.Second))
	other := newResumeTestSession("other-model", otherDir, "other", baseTime.Add(time.Second))
	for _, session := range []*Session{excluded, included, other} {
		if err := storage.Save(session); err != nil {
			t.Fatalf("Save(%s) error = %v", session.Model, err)
		}
	}

	filtered, err := storage.ListResumeSessions(ResumeListOptions{WorkingDir: currentDir, ExcludeSessionID: excluded.ID})
	if err != nil {
		t.Fatalf("ListResumeSessions() error = %v", err)
	}
	if got := resumeSessionModels(filtered); !sameStringSet(got, []string{"included-model"}) {
		t.Fatalf("filtered models = %#v, want only included-model", got)
	}

	all, err := storage.ListResumeSessions(ResumeListOptions{WorkingDir: currentDir, All: true, ExcludeSessionID: excluded.ID})
	if err != nil {
		t.Fatalf("ListResumeSessions(all) error = %v", err)
	}
	if got := resumeSessionModels(all); !sameStringSet(got, []string{"included-model", "other-model"}) {
		t.Fatalf("all models = %#v, want included/other", got)
	}

	lastID, err := storage.GetLastResumeSession(ResumeListOptions{WorkingDir: currentDir, All: true, ExcludeSessionID: excluded.ID})
	if err != nil {
		t.Fatalf("GetLastResumeSession() error = %v", err)
	}
	if lastID != included.ID {
		t.Fatalf("GetLastResumeSession() = %q, want included %q", lastID, included.ID)
	}
}

func saveResumeTestSession(storage *Storage, model, workingDir, content string) error {
	session := NewSession(model)
	session.WorkingDir = workingDir
	session.AddMessage("user", content, model)
	return storage.Save(session)
}

func newResumeTestSession(model, workingDir, content string, lastModified time.Time) *Session {
	session := NewSession(model)
	session.WorkingDir = workingDir
	session.AddMessage("user", content, model)
	session.LastModified = lastModified
	return session
}

func resumeSessionModels(sessions []SessionMetadata) []string {
	models := make([]string, 0, len(sessions))
	for _, session := range sessions {
		models = append(models, session.Model)
	}
	return models
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, value := range a {
		counts[value]++
	}
	for _, value := range b {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}
