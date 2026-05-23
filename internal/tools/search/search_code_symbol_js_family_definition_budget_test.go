package search

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldDeferIncompleteJSFamilyDefinitions(t *testing.T) {
	tests := []struct {
		name       string
		incomplete bool
		want       bool
	}{
		{name: "complete"},
		{name: "incomplete", incomplete: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDeferIncompleteJSFamilyDefinitions(tt.incomplete); got != tt.want {
				t.Fatalf("shouldDeferIncompleteJSFamilyDefinitions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSFamilyDefinitionCandidateMatchesWithinLimit(t *testing.T) {
	matches := make([]genericSymbolMatch, jsFamilyDefinitionCandidateMatchLimit+1)
	within, overLimit := jsFamilyDefinitionCandidateMatchesWithinLimit(matches[:jsFamilyDefinitionCandidateMatchLimit])
	if overLimit {
		t.Fatal("overLimit = true, want false for exact budget")
	}
	if len(within) != jsFamilyDefinitionCandidateMatchLimit {
		t.Fatalf("within len = %d, want %d", len(within), jsFamilyDefinitionCandidateMatchLimit)
	}

	within, overLimit = jsFamilyDefinitionCandidateMatchesWithinLimit(matches)
	if !overLimit {
		t.Fatal("overLimit = false, want true after observing one extra match")
	}
	if len(within) != jsFamilyDefinitionCandidateMatchLimit {
		t.Fatalf("within len = %d, want %d after trim", len(within), jsFamilyDefinitionCandidateMatchLimit)
	}
}

func TestResolveJSSymbol_ExactBudgetMultipleDefinitionsKeepsStructuredMultiple(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": jsFamilyDefinitionSource("buildUser", jsFamilyDefinitionCandidateMatchLimit),
	})

	result := resolveJSSymbol("buildUser", SearchOptions{
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
	})

	if result.Status != genericSymbolMultiple {
		t.Fatalf("status = %s, want %s for exact budget definition candidates", result.Status, genericSymbolMultiple)
	}
	if !strings.Contains(result.Output, `Multiple definitions found for "buildUser"`) {
		t.Fatalf("output = %q, want multiple definitions output", result.Output)
	}
	if len(result.AffectedFiles) != 1 {
		t.Fatalf("AffectedFiles len = %d, want 1 for exact budget definitions in one file", len(result.AffectedFiles))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactExactBudgetMultipleDefinitionsKeepsAmbiguous(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": jsFamilyDefinitionSource("buildUser", jsFamilyDefinitionCandidateMatchLimit),
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:       "buildUser",
		Intent:        "impact",
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
	})

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want true; output:\n%s", artifact.Rendered)
	}
	if !artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = false, want true for exact-budget complete multiple definitions; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil for ambiguous exact-budget definitions", artifact.Metadata.Bundle)
	}
	if !strings.Contains(artifact.Rendered, `Multiple definitions found for "buildUser"`) {
		t.Fatalf("output missing multiple definitions message:\n%s", artifact.Rendered)
	}
	if len(artifact.Metadata.AffectedFiles) != 1 {
		t.Fatalf("AffectedFiles len = %d, want 1 for exact budget definitions in one file", len(artifact.Metadata.AffectedFiles))
	}
}

func TestResolveJSSymbol_IncompleteMultipleDefinitionCandidatesDefersToFallback(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": jsFamilyDefinitionSource("buildUser", jsFamilyDefinitionCandidateMatchLimit+1),
	})

	result := resolveJSSymbol("buildUser", SearchOptions{
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
	})

	if result.Status != genericSymbolNone {
		t.Fatalf("status = %s, want %s for incomplete definition candidates", result.Status, genericSymbolNone)
	}
	if result.Output != "" {
		t.Fatalf("output = %q, want empty output so text fallback can run", result.Output)
	}
}

func TestResolveJSSymbol_DirectFileCappedMatchesKeepStructuredDefinition(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		fileType   string
		definition string
	}{
		{name: "javascript", file: "src/build.js", fileType: "js", definition: "function buildUser() { return 'ok' }\n"},
		{name: "typescript", file: "src/build.ts", fileType: "ts", definition: "export function buildUser() { return 'ok' }\n"},
		{name: "tsx", file: "src/build.tsx", fileType: "tsx", definition: "export function buildUser() { return <button /> }\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				tt.file: jsFamilyCappedDirectFileSource("buildUser", tt.definition),
			})

			result := resolveJSSymbol("buildUser", SearchOptions{
				Path:               filepath.Join(dir, filepath.FromSlash(tt.file)),
				FileType:           tt.fileType,
				InvocationCWD:      dir,
				ProjectMapRootPath: dir,
			})

			if result.Status != genericSymbolSingle {
				t.Fatalf("status = %s, want %s for direct file capped matches", result.Status, genericSymbolSingle)
			}
			if result.Bundle == nil {
				t.Fatal("bundle = nil, want structured bundle")
			}
			if got := result.Bundle.Definition.File; got != tt.file {
				t.Fatalf("definition file = %q, want %s", got, tt.file)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JSFamilyStructuredImpactDirectFileCappedMatchesKeepStructuredDefinition(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		fileType   string
		definition string
	}{
		{name: "javascript", file: "src/build.js", fileType: "js", definition: "function buildUser() { return 'ok' }\n"},
		{name: "typescript", file: "src/build.ts", fileType: "ts", definition: "export function buildUser() { return 'ok' }\n"},
		{name: "tsx", file: "src/build.tsx", fileType: "tsx", definition: "export function buildUser() { return <button /> }\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				tt.file: jsFamilyCappedDirectFileSource("buildUser", tt.definition),
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
				Pattern:            "buildUser",
				Intent:             "impact",
				Path:               filepath.Join(dir, filepath.FromSlash(tt.file)),
				FileType:           tt.fileType,
				InvocationCWD:      dir,
				ProjectMapRootPath: dir,
			})

			if !artifact.Metadata.StructuredImpact {
				t.Fatalf("StructuredImpact = false, want true; output:\n%s", artifact.Rendered)
			}
			if artifact.Metadata.Bundle == nil {
				t.Fatalf("Bundle = nil, want structured bundle; output:\n%s", artifact.Rendered)
			}
			if got := artifact.Metadata.Bundle.Definition.File; got != tt.file {
				t.Fatalf("definition file = %q, want %s", got, tt.file)
			}
			if artifact.Metadata.Ambiguous {
				t.Fatalf("Ambiguous = true, want false; output:\n%s", artifact.Rendered)
			}
		})
	}
}

func jsFamilyCappedDirectFileSource(symbol string, definition string) string {
	var source strings.Builder
	for i := 0; i <= jsFamilyDefinitionCandidateMatchLimit; i++ {
		source.WriteString(symbol)
		source.WriteString("()\n")
	}
	source.WriteString(definition)
	return source.String()
}

func jsFamilyDefinitionSource(symbol string, count int) string {
	var source strings.Builder
	for i := 0; i < count; i++ {
		source.WriteString("function ")
		source.WriteString(symbol)
		source.WriteString("() { return 'ok' }\n")
	}
	return source.String()
}
