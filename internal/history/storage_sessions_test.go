package history

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStorage_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// 複数のセッションを作成して保存
	// 注意: Session IDは秒単位のUnixタイムスタンプなので、1秒以上空ける必要がある
	var sessionIDs []string
	for i := 1; i <= 3; i++ {
		session := NewSession(fmt.Sprintf("model-%d", i))
		session.AddMessage("user", fmt.Sprintf("Message %d", i), session.Model)

		if err := storage.Save(session); err != nil {
			t.Fatalf("Save session%d failed: %v", i, err)
		}
		sessionIDs = append(sessionIDs, session.ID)
	}

	// リスト取得
	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	// 3つのセッションが返されるべき
	if len(sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(sessions))
	}

	seen := make(map[string]struct{}, len(sessions))
	for _, s := range sessions {
		seen[s.ID] = struct{}{}
	}
	for _, id := range sessionIDs {
		if _, ok := seen[id]; !ok {
			t.Fatalf("Session ID %s was not returned by ListSessions", id)
		}
	}
}

func TestStorage_ListSessions_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// リスト取得（セッションなし）
	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected empty sessions, got %d", len(sessions))
	}
}

func TestStorage_GetLastSession(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// セッション作成
	session1 := NewSession("model-1")
	session1.AddMessage("user", "Old session", "model-1")
	if err := storage.Save(session1); err != nil {
		t.Fatalf("Failed to save session1: %v", err)
	}

	session2 := NewSession("model-2")
	session2.AddMessage("user", "New session", "model-2")
	if err := storage.Save(session2); err != nil {
		t.Fatalf("Failed to save session2: %v", err)
	}

	// 最新セッション取得
	lastID, err := storage.GetLastSession()
	if err != nil {
		t.Fatalf("GetLastSession failed: %v", err)
	}

	if lastID == "" {
		t.Fatal("GetLastSession returned empty ID")
	}
	if lastID != session1.ID && lastID != session2.ID {
		t.Fatalf("Expected last session ID to be one of created sessions, got %s", lastID)
	}
}

func TestStorage_GetLastSession_NoSessions(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// セッションなしでGetLastSession
	_, err = storage.GetLastSession()
	if err == nil {
		t.Error("Expected error when no sessions exist")
	}

	if !strings.Contains(err.Error(), "no sessions found") {
		t.Errorf("Expected 'no sessions found' error, got: %v", err)
	}
}

func TestStorage_ListSessions_SortsByLastModifiedDescending(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session1 := NewSession("model-1")
	session1.AddMessage("user", "first", "model-1")
	if err := storage.Save(session1); err != nil {
		t.Fatalf("save session1 failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	session2 := NewSession("model-2")
	session2.AddMessage("user", "second", "model-2")
	if err := storage.Save(session2); err != nil {
		t.Fatalf("save session2 failed: %v", err)
	}

	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(ListSessions()) = %d, want 2", len(sessions))
	}
	if !sessions[0].LastModified.After(sessions[1].LastModified) && !sessions[0].LastModified.Equal(sessions[1].LastModified) {
		t.Fatalf("sessions are not sorted by LastModified desc: %#v", sessions)
	}
}
