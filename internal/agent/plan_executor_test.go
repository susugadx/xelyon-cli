package agent

import (
	"testing"
)

// --- Level 2: getGitDiffHash ---

func TestGetGitDiffHash_Deterministic(t *testing.T) {
	// 同じ状態で2回呼び出したら同じハッシュを返す
	h1 := getGitDiffHash()
	h2 := getGitDiffHash()
	if h1 == "" {
		t.Skip("git not available")
	}
	if h1 != h2 {
		t.Errorf("expected same hash, got %q vs %q", h1, h2)
	}
}

func TestGetGitDiffHash_NonEmpty(t *testing.T) {
	h := getGitDiffHash()
	if h == "" {
		t.Skip("git not available")
	}
	// SHA256 ハッシュは 64 文字の hex 文字列
	if len(h) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars: %q", len(h), h)
	}
}
