package agent

import (
	"testing"
)

// --- Level 1: checkMissingWriteTools ---

func TestCheckMissingWriteTools_HasMissing(t *testing.T) {
	stepTools := []string{"read_file", "str_replace"}
	executed := map[string]bool{"read_file": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 1 || missing[0] != "str_replace" {
		t.Errorf("expected [str_replace], got %v", missing)
	}
}

func TestCheckMissingWriteTools_AllExecuted(t *testing.T) {
	stepTools := []string{"read_file", "str_replace", "write_file"}
	executed := map[string]bool{"read_file": true, "str_replace": true, "write_file": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 0 {
		t.Errorf("expected empty, got %v", missing)
	}
}

func TestCheckMissingWriteTools_ReadOnly(t *testing.T) {
	stepTools := []string{"read_file", "list_dir", "bash"}
	executed := map[string]bool{"read_file": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 0 {
		t.Errorf("expected empty (no write tools in plan), got %v", missing)
	}
}

func TestCheckMissingWriteTools_MultipleMissing(t *testing.T) {
	stepTools := []string{"str_replace", "write_file", "delete_file"}
	executed := map[string]bool{}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 3 {
		t.Errorf("expected 3 missing, got %v", missing)
	}
}

func TestCheckMissingWriteTools_EmptyPlan(t *testing.T) {
	stepTools := []string{}
	executed := map[string]bool{"str_replace": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 0 {
		t.Errorf("expected empty, got %v", missing)
	}
}

// --- Level 1 エスケープ: containsAlreadyAppliedDeclaration ---

func TestContainsAlreadyAppliedDeclaration_English(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"This change was already applied in step 1.", true},
		{"The modification has already been implemented.", true},
		{"This was handled in a previous step.", true},
		{"The file was already modified by an earlier step.", true},
		{"I'll now apply the str_replace.", false},
	}
	for _, tt := range tests {
		got := containsAlreadyAppliedDeclaration(tt.input)
		if got != tt.want {
			t.Errorf("containsAlreadyAppliedDeclaration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestContainsAlreadyAppliedDeclaration_Japanese(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"この変更は前のステップで既に適用されています。", true},
		{"修正済みのため、このステップでは変更不要です。", true},
		{"既に変更が反映されています。", true},
		{"str_replaceを実行します。", false},
	}
	for _, tt := range tests {
		got := containsAlreadyAppliedDeclaration(tt.input)
		if got != tt.want {
			t.Errorf("containsAlreadyAppliedDeclaration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestContainsAlreadyAppliedDeclaration_NoMatch(t *testing.T) {
	inputs := []string{
		"I will now make the changes.",
		"Let me apply the fix.",
		"ファイルを修正します。",
		"",
	}
	for _, input := range inputs {
		if containsAlreadyAppliedDeclaration(input) {
			t.Errorf("containsAlreadyAppliedDeclaration(%q) should be false", input)
		}
	}
}

// --- diffContainsFileChanges ---

func TestDiffContainsFileChanges_EmptyFiles(t *testing.T) {
	// 空のファイルリストでは常に false
	if diffContainsFileChanges(nil) {
		t.Error("diffContainsFileChanges(nil) should be false")
	}
	if diffContainsFileChanges([]string{}) {
		t.Error("diffContainsFileChanges([]) should be false")
	}
}

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
