package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type stubExtractor struct {
	rules []string
	err   error
}

func (s *stubExtractor) Extract(ctx context.Context, history []api.Message, model string) ([]string, error) {
	return s.rules, s.err
}

func TestAppendRulesToXelyon_CreateSection(t *testing.T) {
	base := "# Title\n\nSome content\n"
	rules := []string{"エラーは必ずwrapする", "context.Contextを渡す"}

	updated, err := appendRulesToXelyon(base, rules)
	if err != nil {
		t.Fatalf("appendRulesToXelyon error: %v", err)
	}
	if !strings.Contains(updated, "## 学習したルール") {
		t.Fatalf("expected section header")
	}
	if !strings.Contains(updated, "- エラーは必ずwrapする") {
		t.Fatalf("expected rule appended")
	}
}

func TestAppendRulesToXelyon_InsertIntoExistingSection(t *testing.T) {
	base := "# Title\n\n## 学習したルール\n\n- 既存\n\n## Next\n"
	rules := []string{"新規ルール"}

	updated, err := appendRulesToXelyon(base, rules)
	if err != nil {
		t.Fatalf("appendRulesToXelyon error: %v", err)
	}
	// Should appear after header (before existing list)
	expectedOrder := "## 学習したルール\n\n- 新規ルール\n\n- 既存"
	if !strings.Contains(updated, expectedOrder) {
		t.Fatalf("expected insertion after header. got:\n%s", updated)
	}
}

func TestFilterNewRules_DedupByContent(t *testing.T) {
	xelyon := "## 学習したルール\n\n- エラーは必ずwrapする\n"
	rules := []string{"エラーは必ずwrapする", "新しい"}

	out := filterNewRules(xelyon, rules)
	if len(out) != 1 || out[0] != "新しい" {
		t.Fatalf("unexpected filter result: %#v", out)
	}
}

func TestNormalizeRules_TrimsAndDedups(t *testing.T) {
	in := []string{"  - エラーは必ずwrapする  ", "エラーは必ずwrapする", "\tcontext.Contextを渡す\n"}
	out := normalizeRules(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 rules, got %d: %#v", len(out), out)
	}
}

func TestProposeAndApplyLearning_NoRules(t *testing.T) {
	ag := &Agent{History: []api.Message{{Role: "user", Content: "hi"}}, CurrentModel: "test"}
	applied, rules, err := ag.ProposeAndApplyLearning(context.Background(), &stubExtractor{rules: []string{}}, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if applied {
		t.Fatalf("expected not applied")
	}
	if len(rules) != 0 {
		t.Fatalf("expected no rules")
	}
}

func TestProposeAndApplyLearning_ReadError(t *testing.T) {
	ag := &Agent{History: []api.Message{{Role: "user", Content: "hi"}}, CurrentModel: "test"}
	_, _, err := ag.ProposeAndApplyLearning(context.Background(), &stubExtractor{rules: []string{"rule"}}, "__no_such_file__")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProposeAndApplyLearning_FiltersExistingBeforePrompt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "XELYON.md")
	if err := os.WriteFile(path, []byte("## 学習したルール\n\n- 既存ルール\n"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	ag := &Agent{History: []api.Message{{Role: "user", Content: "hi"}}, CurrentModel: "test"}

	// If all rules are already present, it should not try to prompt, and should return applied=false.
	applied, gotRules, err := ag.ProposeAndApplyLearning(context.Background(), &stubExtractor{rules: []string{"既存ルール"}}, path)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if applied {
		t.Fatalf("expected not applied")
	}
	if len(gotRules) != 1 {
		t.Fatalf("expected original extracted rules returned, got %#v", gotRules)
	}
}
