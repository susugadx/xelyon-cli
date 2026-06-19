package prompt

import (
	"strings"
	"testing"
)

func TestBuildProjectMapSection(t *testing.T) {
	got := BuildProjectMapSection("## Project Map\n\n- a.go")
	if !strings.Contains(got, ProjectMapStartMarker) || !strings.Contains(got, ProjectMapEndMarker) {
		t.Fatalf("BuildProjectMapSection() should include markers:\n%s", got)
	}
	if !strings.Contains(got, projectMapDataStartTag) || !strings.Contains(got, projectMapDataEndTag) {
		t.Fatalf("BuildProjectMapSection() should include data wrapper:\n%s", got)
	}
	if !strings.Contains(got, "## Project Map") {
		t.Fatalf("BuildProjectMapSection() should include section body:\n%s", got)
	}
}

func TestInjectProjectMapSection_ReplacesExisting(t *testing.T) {
	first := InjectProjectMapSection("base", "## Project Map\n\n- a.go")
	second := InjectProjectMapSection(first, "## Project Map\n\n- b.go")

	if strings.Count(second, ProjectMapStartMarker) != 1 {
		t.Fatalf("marker block should remain single:\n%s", second)
	}
	if strings.Contains(second, "- a.go") {
		t.Fatalf("old project map should be replaced:\n%s", second)
	}
	if !strings.Contains(second, "- b.go") {
		t.Fatalf("new project map should be injected:\n%s", second)
	}
}

func TestExtractProjectMapSection_MarkerBlock(t *testing.T) {
	prompt := "base\n\n" + BuildProjectMapSection("## Project Map\n\n- a.go")
	got := ExtractProjectMapSection(prompt)
	if !strings.Contains(got, "- a.go") {
		t.Fatalf("ExtractProjectMapSection() should return marker body:\n%s", got)
	}
	if strings.Contains(got, ProjectMapStartMarker) || strings.Contains(got, ProjectMapEndMarker) {
		t.Fatalf("ExtractProjectMapSection() should not include markers:\n%s", got)
	}
	if strings.Contains(got, projectMapDataStartTag) || strings.Contains(got, projectMapDataEndTag) {
		t.Fatalf("ExtractProjectMapSection() should not include data wrapper:\n%s", got)
	}
}

func TestExtractProjectMapSection_NoLegacyFallback(t *testing.T) {
	prompt := "base\n\n## Project Map\n- a.go\n\n## Project Context:\nctx"
	got := ExtractProjectMapSection(prompt)
	if got != "" {
		t.Fatalf("ExtractProjectMapSection() should ignore legacy block by default, got:\n%s", got)
	}
}

func TestExtractProjectMapSectionCompat_LegacyFallback(t *testing.T) {
	prompt := "base\n\n## Project Map\n- a.go\n\n## Project Context:\nctx"
	got := ExtractProjectMapSectionCompat(prompt)
	if strings.Contains(got, "Project Context") {
		t.Fatalf("legacy extraction should stop before next section:\n%s", got)
	}
	if !strings.Contains(got, "- a.go") {
		t.Fatalf("legacy extraction should include project map body:\n%s", got)
	}
}

func TestStripProjectMapSection_MarkerBlock(t *testing.T) {
	prompt := "base\n\n" + BuildProjectMapSection("## Project Map\n\n- a.go")
	got := StripProjectMapSection(prompt)
	if got != "base" {
		t.Fatalf("StripProjectMapSection() = %q, want base", got)
	}
}

func TestBuildProjectMapSection_NeutralizesDelimiterCollisions(t *testing.T) {
	body := strings.Join([]string{
		"## Project Map",
		ProjectMapStartMarker,
		ProjectMapEndMarker,
		projectMapDataStartTag,
		projectMapDataEndTag,
		"- a.go",
	}, "\n")

	got := BuildProjectMapSection(body)
	if strings.Count(got, ProjectMapStartMarker) != 1 || strings.Count(got, ProjectMapEndMarker) != 1 {
		t.Fatalf("BuildProjectMapSection() should keep only structural markers:\n%s", got)
	}
	if strings.Count(got, projectMapDataStartTag) != 1 || strings.Count(got, projectMapDataEndTag) != 1 {
		t.Fatalf("BuildProjectMapSection() should keep only structural data wrapper:\n%s", got)
	}

	extracted := ExtractProjectMapSection(got)
	for _, forbidden := range []string{
		ProjectMapStartMarker,
		ProjectMapEndMarker,
		projectMapDataStartTag,
		projectMapDataEndTag,
	} {
		if strings.Contains(extracted, forbidden) {
			t.Fatalf("ExtractProjectMapSection() should return neutralized body without raw delimiter %q:\n%s", forbidden, extracted)
		}
	}
	if !strings.Contains(extracted, "- a.go") {
		t.Fatalf("ExtractProjectMapSection() lost project map body:\n%s", extracted)
	}
}

func TestStripProjectMapSection_NoLegacyFallback(t *testing.T) {
	prompt := "base\n\n## Project Map\n- a.go"
	got := StripProjectMapSection(prompt)
	if got != prompt {
		t.Fatalf("StripProjectMapSection() should keep legacy block by default, got=%q", got)
	}
}

func TestStripProjectMapSectionCompat_LegacyFallback(t *testing.T) {
	prompt := "base\n\n## Project Map\n- a.go"
	got := StripProjectMapSectionCompat(prompt)
	if got != "base" {
		t.Fatalf("legacy StripProjectMapSection() = %q, want base", got)
	}
}
