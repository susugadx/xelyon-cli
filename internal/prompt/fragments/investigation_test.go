package fragments

import (
	"strings"
	"testing"
)

func TestBuildInvestigationToolingBlock_DefaultLines(t *testing.T) {
	block := BuildInvestigationToolingBlock(InvestigationToolingOptions{})

	for _, want := range []string{
		"gather_context: default investigation tool",
		"comma-separated gather_context item is an exact file or range",
		`gather_context(query="Makefile")`,
		`gather_context(query="./Makefile")`,
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("expected tooling block to contain %q, got:\n%s", want, block)
		}
	}
	if strings.Contains(block, "search_code: code discovery tool") {
		t.Fatalf("default tooling block should not inject override lines without options, got:\n%s", block)
	}
}

func TestBuildInvestigationToolingBlock_OptionalOverrides(t *testing.T) {
	block := BuildInvestigationToolingBlock(InvestigationToolingOptions{
		AllowLowLevelOverrides: true,
		SearchOverrideLabel:    "a low-level override",
		SearchOverrideExtra:    "Prefer mode=auto.",
		ReadOverrideExtra:      "Use exact ranges only when needed.",
		IncludeBatchRead:       true,
		BatchReadOverrideLabel: "an expert override",
		IncludeMultiPattern:    true,
		MultiPatternExtra:      "Combine related patterns.",
	})

	for _, want := range []string{
		"search_code: code discovery tool for a low-level override",
		"Prefer mode=auto.",
		"read_file: low-level exact-content reader for expert override",
		"Use exact ranges only when needed.",
		"read_file can batch them as an expert override",
		"prefer one gather_context query or one search_code call with comma-separated patterns",
		"Combine related patterns.",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("expected tooling block to contain %q, got:\n%s", want, block)
		}
	}
}

func TestReviewInvestigationSentence_DefaultSurfaceAvoidsHiddenToolNames(t *testing.T) {
	sentence := ReviewInvestigationSentence(false)
	if strings.Contains(sentence, "search_code") || strings.Contains(sentence, "read_file") {
		t.Fatalf("default review sentence should avoid hidden low-level tool names, got %q", sentence)
	}
}

func TestLegacyEditToolRulesBlock_SharedRules(t *testing.T) {
	block := LegacyEditToolRulesBlock()

	for _, want := range []string{
		"Prefer str_replace for partial edits after targeted reads or searches.",
		"Use write_file for full-file creation or replacement.",
		"Use delete_file only for intentional removals.",
		"actual gather_context, read_file, or search_code output",
		"Do not loop read-fail-read-fail.",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("expected legacy block to contain %q, got:\n%s", want, block)
		}
	}
}

func TestInvestigationModeAwareLines(t *testing.T) {
	if strings.Contains(InvestigationFollowUpLine(false, ""), "read_file") {
		t.Fatal("default follow-up line should avoid hidden read_file guidance")
	}
	if !strings.Contains(InvestigationFollowUpLine(true, ""), "read_file") {
		t.Fatal("legacy follow-up line should keep read_file guidance")
	}
	if strings.Contains(InvestigationMultiPatternLine(false, ""), "search_code") {
		t.Fatal("default multi-pattern line should avoid hidden search_code guidance")
	}
	if !strings.Contains(InvestigationMultiPatternLine(true, ""), "search_code") {
		t.Fatal("legacy multi-pattern line should keep search_code guidance")
	}
	if strings.Contains(InvestigationContextSourceLine(false), "read_file") || strings.Contains(InvestigationContextSourceLine(false), "search_code") {
		t.Fatal("default context-source line should avoid hidden low-level tool names")
	}
	if !strings.Contains(InvestigationContextSourceLine(true), "read_file") || !strings.Contains(InvestigationContextSourceLine(true), "search_code") {
		t.Fatal("legacy context-source line should keep low-level tool names")
	}
}

func TestInvestigationAllowedToolsLine_ModeVariants(t *testing.T) {
	defaultLine := InvestigationAllowedToolsLine(false)
	if strings.Contains(defaultLine, "search_code") || strings.Contains(defaultLine, "read_file") {
		t.Fatalf("default allowed tools should stay gather_context-first, got %q", defaultLine)
	}

	legacyLine := InvestigationAllowedToolsLine(true)
	for _, want := range []string{"gather_context", "search_code", "read_file", "web_search", "bash"} {
		if !strings.Contains(legacyLine, want) {
			t.Fatalf("legacy allowed tools should contain %q, got %q", want, legacyLine)
		}
	}
}
