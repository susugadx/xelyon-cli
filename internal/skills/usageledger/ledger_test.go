package usageledger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreAppendSummaryAndPrivacy(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	store := NewStore(Options{
		StateHome:     t.TempDir(),
		ProjectRoot:   "/repo/project",
		Enabled:       true,
		RetentionDays: 30,
		Now:           func() time.Time { return now },
	})

	rawTask := "fix bug with SECRET_TOKEN=abc123"
	if err := store.Append(Record{
		Type: "recommendation",
		Recommended: []SkillSummary{{
			Name:       "bug-investigation",
			Category:   "primary",
			Score:      92,
			Confidence: "high",
			Activation: "hint",
		}},
		Policy: PolicySnapshot{Enabled: true, Activation: "hint", PrimaryLimit: 2, SupportingLimit: 5, ConflictLimit: 5},
	}); err != nil {
		t.Fatalf("Append(recommendation) error = %v", err)
	}
	if err := store.Append(Record{
		Type:      "activation",
		Activated: []string{"bug-investigation"},
	}); err != nil {
		t.Fatalf("Append(activation) error = %v", err)
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 2 {
		t.Fatalf("Records = %d, want 2", summary.Records)
	}
	if len(summary.Skills) != 1 || summary.Skills[0].RecommendedCount != 1 || summary.Skills[0].ActivatedCount != 1 {
		t.Fatalf("Skills = %#v", summary.Skills)
	}

	data, err := os.ReadFile(summary.Path)
	if err != nil {
		t.Fatalf("ReadFile(ledger) error = %v", err)
	}
	if strings.Contains(string(data), rawTask) || strings.Contains(string(data), "SECRET_TOKEN") {
		t.Fatalf("ledger leaked raw task/secret:\n%s", string(data))
	}
	if strings.Contains(summary.Path, "/repo/project") {
		t.Fatalf("ledger path exposes raw project path: %s", summary.Path)
	}
}

func TestStorePrunesByRetention(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	store := NewStore(Options{
		StateHome:     t.TempDir(),
		ProjectRoot:   "/repo/project",
		Enabled:       true,
		RetentionDays: 1,
		Now:           func() time.Time { return now },
	})

	if err := store.Append(Record{
		Timestamp:   now.AddDate(0, 0, -3),
		Type:        "recommendation",
		Recommended: []SkillSummary{{Name: "old", Score: 80}},
	}); err != nil {
		t.Fatalf("Append(old) error = %v", err)
	}
	if err := store.Append(Record{
		Timestamp:   now,
		Type:        "recommendation",
		Recommended: []SkillSummary{{Name: "new", Score: 80}},
	}); err != nil {
		t.Fatalf("Append(new) error = %v", err)
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 1 || len(summary.Skills) != 1 || summary.Skills[0].Name != "new" {
		t.Fatalf("summary after prune = %#v", summary)
	}
}

func TestStoreClearAndClearAll(t *testing.T) {
	stateHome := t.TempDir()
	storeA := NewStore(Options{StateHome: stateHome, ProjectRoot: "/repo/a", Enabled: true})
	storeB := NewStore(Options{StateHome: stateHome, ProjectRoot: "/repo/b", Enabled: true})
	if err := storeA.Append(Record{Type: "activation", Activated: []string{"a"}}); err != nil {
		t.Fatalf("Append(A) error = %v", err)
	}
	if err := storeB.Append(Record{Type: "activation", Activated: []string{"b"}}); err != nil {
		t.Fatalf("Append(B) error = %v", err)
	}
	if err := storeA.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if _, err := os.Stat(storeA.path()); !os.IsNotExist(err) {
		t.Fatalf("storeA path should be removed, err=%v", err)
	}
	if _, err := os.Stat(storeB.path()); err != nil {
		t.Fatalf("storeB path should remain, err=%v", err)
	}
	if err := storeA.ClearAll(); err != nil {
		t.Fatalf("ClearAll() error = %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(storeB.path()))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("ledger dir entries = %#v, want empty", entries)
	}
}

func TestDiagnosticsForRecommendedNeverActivated(t *testing.T) {
	summary := Summary{Skills: []SkillUsage{{
		Name:             "review",
		RecommendedCount: 3,
		ActivatedCount:   0,
	}}}

	diagnostics := Diagnostics(summary)
	if len(diagnostics) != 1 || diagnostics[0].Code != "usage_recommended_never_activated" {
		t.Fatalf("Diagnostics() = %#v", diagnostics)
	}
}

func TestSummaryDoesNotCountConflictAsActivationRecommendation(t *testing.T) {
	store := NewStore(Options{
		StateHome:   t.TempDir(),
		ProjectRoot: "/repo/project",
		Enabled:     true,
	})
	for i := 0; i < 3; i++ {
		if err := store.Append(Record{
			Type: "recommendation",
			Recommended: []SkillSummary{{
				Name:       "strict-diff-review",
				Category:   "conflict",
				Score:      82,
				Confidence: "high",
				Activation: "hint",
			}},
		}); err != nil {
			t.Fatalf("Append(conflict recommendation) error = %v", err)
		}
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if len(summary.Skills) != 0 {
		t.Fatalf("conflict recommendations should not count as activation recommendations: %#v", summary.Skills)
	}
	if diagnostics := Diagnostics(summary); len(diagnostics) != 0 {
		t.Fatalf("Diagnostics() = %#v, want none for conflict-only records", diagnostics)
	}
}

func TestFormatSummarySanitizesSkillNames(t *testing.T) {
	got := FormatSummary(Summary{
		RepoKey: "repo",
		Records: 1,
		Skills: []SkillUsage{{
			Name:             "demo\n- injected",
			RecommendedCount: 1,
		}},
	})
	if strings.Contains(got, "\n- injected") {
		t.Fatalf("FormatSummary leaked multiline skill name:\n%s", got)
	}
	if !strings.Contains(got, "- demo - injected: recommended 1, activated 0") {
		t.Fatalf("FormatSummary missing sanitized skill name:\n%s", got)
	}
}

func TestDisabledStoreDoesNotWrite(t *testing.T) {
	stateHome := t.TempDir()
	store := NewStore(Options{StateHome: stateHome, ProjectRoot: "/repo/project", Enabled: false})
	if err := store.Append(Record{Type: "activation", Activated: []string{"skill"}}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if _, err := os.Stat(store.path()); !os.IsNotExist(err) {
		t.Fatalf("disabled store should not write, stat err=%v", err)
	}
}
