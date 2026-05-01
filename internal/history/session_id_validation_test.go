package history

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateSessionID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		validIDs := []string{
			"session-1",
			"abc_123",
			"A.B-C",
			strings.Repeat("x", maxSessionIDLength),
		}
		for _, id := range validIDs {
			if err := validateSessionID(id); err != nil {
				t.Fatalf("validateSessionID(%q) error = %v, want nil", id, err)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		invalidIDs := []string{
			"",
			" ",
			".",
			"..",
			" leading",
			"trailing ",
			"a/b",
			`a\b`,
			strings.Repeat("x", maxSessionIDLength+1),
		}
		for _, id := range invalidIDs {
			if err := validateSessionID(id); err == nil {
				t.Fatalf("validateSessionID(%q) error = nil, want error", id)
			}
		}
	})
}

func TestStorage_RejectsInvalidSessionID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	invalidIDs := []string{
		"",
		".",
		"..",
		"bad/id",
		`bad\id`,
		" bad",
		"bad ",
		strings.Repeat("x", maxSessionIDLength+1),
	}

	for i, sessionID := range invalidIDs {
		t.Run(fmt.Sprintf("%d:%q", i, sessionID), func(t *testing.T) {
			session := NewSession("test-model")
			session.ID = sessionID
			session.AddMessage("user", "hello", session.Model)

			if err := storage.Save(session); err == nil || !strings.Contains(err.Error(), "invalid session ID") {
				t.Fatalf("Save() error = %v, want invalid session ID error", err)
			}
			if err := storage.Rewrite(session); err == nil || !strings.Contains(err.Error(), "invalid session ID") {
				t.Fatalf("Rewrite() error = %v, want invalid session ID error", err)
			}
			if _, err := storage.Load(sessionID); err == nil || !strings.Contains(err.Error(), "invalid session ID") {
				t.Fatalf("Load() error = %v, want invalid session ID error", err)
			}
		})
	}
}
