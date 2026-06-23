package prompt

import (
	"strings"
	"testing"
)

func TestEffectivePromptComposeSeparatesStaticAndDynamic(t *testing.T) {
	effective, err := NewEffectivePrompt(
		StaticText("base", AuthorityConstitution, "base prompt"),
		DynamicText("project-map", AuthorityData, "project map", map[string]string{"source": "runtime"}),
		StaticText("repo", AuthorityRepoInstruction, "repo rules"),
	)
	if err != nil {
		t.Fatalf("NewEffectivePrompt() error = %v", err)
	}

	got := effective.Compose("\n---split---\n")
	want := "base prompt\n\nrepo rules\n---split---\nproject map"
	if got != want {
		t.Fatalf("Compose() = %q, want %q", got, want)
	}
}

func TestEffectivePromptRejectsInvalidZeroSection(t *testing.T) {
	if _, err := NewEffectivePrompt(PromptSection{}); err == nil {
		t.Fatal("NewEffectivePrompt() error = nil, want zero section rejection")
	}
}

func TestEffectivePromptRejectsDuplicateIDs(t *testing.T) {
	_, err := NewEffectivePrompt(
		StaticText("dup", AuthorityConstitution, "a"),
		DynamicText("dup", AuthorityData, "b", nil),
	)
	if err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("NewEffectivePrompt() error = %v, want duplicate ID error", err)
	}
}

func TestEffectivePromptFingerprintIncludesDynamicMetadata(t *testing.T) {
	base := StaticText("base", AuthorityConstitution, "base prompt")
	first, err := NewEffectivePrompt(base, DynamicText("dynamic", AuthorityData, "same", map[string]string{"version": "1"}))
	if err != nil {
		t.Fatalf("NewEffectivePrompt(first) error = %v", err)
	}
	second, err := NewEffectivePrompt(base, DynamicText("dynamic", AuthorityData, "same", map[string]string{"version": "2"}))
	if err != nil {
		t.Fatalf("NewEffectivePrompt(second) error = %v", err)
	}

	firstFingerprint, err := first.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint(first) error = %v", err)
	}
	secondFingerprint, err := second.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint(second) error = %v", err)
	}
	if firstFingerprint == secondFingerprint {
		t.Fatalf("Fingerprint() should change when dynamic metadata changes: %s", firstFingerprint)
	}
}

func TestBuildProjectMapPromptSection(t *testing.T) {
	section, ok := BuildProjectMapPromptSection("## Project Map\n- a.go")
	if !ok {
		t.Fatal("BuildProjectMapPromptSection() ok = false")
	}
	if section.ID() != "xelyon.project_map" || section.Authority() != AuthorityData || !section.Dynamic() {
		t.Fatalf("section metadata = id:%q authority:%q dynamic:%t", section.ID(), section.Authority(), section.Dynamic())
	}
	if strings.Contains(section.Content(), "\x00") {
		t.Fatalf("section content contains unexpected null byte")
	}
}
