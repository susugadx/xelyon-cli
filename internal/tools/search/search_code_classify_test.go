package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalast "github.com/susugadx/xelyon-cli/internal/ast"
)

func TestClassifyMatch(t *testing.T) {
	tests := []struct {
		line     string
		expected MatchType
	}{
		{"func hello() {}", MatchTypeDefinition},
		{"type Config struct {", MatchTypeDefinition},
		{"class MyClass:", MatchTypeDefinition},
		{"def process(data):", MatchTypeDefinition},
		{"    def method(self):", MatchTypeDefinition},
		{"    class InnerClass:", MatchTypeDefinition},
		{"        func nested() {", MatchTypeDefinition},
		{"    var localVar = 1", MatchTypeDefinition},
		{"import fmt", MatchTypeImport},
		{"from os import path", MatchTypeImport},
		{`const { useState } = require('react')`, MatchTypeImport},
		{"x := 42", MatchTypeAssignment},
		{"self.value = data", MatchTypeAssignment},
		{"if x == 1 {", MatchTypeUsage},
		{"fmt.Println(target)", MatchTypeUsage},
		{`fmt.Println("hello=world")`, MatchTypeUsage},
		{`url := "https://example.com?key=value"`, MatchTypeAssignment},
		{"\tName string `json:\"name\"`", MatchTypeUsage},
		{"\tValue int `yaml:\"val=default\"`", MatchTypeUsage},
		{"async def process():", MatchTypeDefinition},
		{"export function doSomething() {", MatchTypeDefinition},
		{"export default function doSomething() {", MatchTypeDefinition},
		{"async export const getUser = () => {", MatchTypeDefinition},
		{"impl Builder {", MatchTypeDefinition},
		{"static void main()", MatchTypeDefinition},
		{"public int getValue()", MatchTypeDefinition},
		{"private string ToString()", MatchTypeDefinition},
		{"if something() {", MatchTypeUsage},
		{"for i := range items {", MatchTypeUsage},
		{"return getValue()", MatchTypeUsage},
		{"switch getType() {", MatchTypeUsage},
		{"while hasNext() {", MatchTypeUsage},
		{"go func() {", MatchTypeUsage},
		{"defer func() {", MatchTypeUsage},
		{"// comment", MatchTypeComment},
		{"# python comment", MatchTypeComment},
	}
	for _, tt := range tests {
		got := classifyMatch(tt.line)
		if got != tt.expected {
			t.Errorf("classifyMatch(%q) = %d, want %d", tt.line, got, tt.expected)
		}
	}
}

func TestReclassifyWithAST_GoFile(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "build.go")
	content := `package main

// Build は廃止予定
func Build() {}

func callBuild() {
	Build()
}

func logBuild() {
	fmt.Println("Build failed")
}

func refBuild() {
	var handler = Build
	_ = handler
}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "Build",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     false,
		Multiline:   false,
	})

	for _, want := range []string{"[def]", "[call]", "[string]", "[comment]", "[ref]"} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected %s in result, got:\n%s", want, result)
		}
	}
	if strings.Contains(result, "[assign]") {
		t.Fatalf("did not expect [assign] after AST reclassification, got:\n%s", result)
	}
	if !strings.Contains(result, "── in func callBuild") {
		t.Fatalf("expected AST scope annotation for func callBuild, got:\n%s", result)
	}
	if strings.Contains(result, "(L") {
		t.Fatalf("did not expect regex block line numbers for AST-derived scope, got:\n%s", result)
	}
}

func TestReclassifyWithAST_NonGoFile(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "build.py")
	content := "def wrapper():\n    print(\"Build failed\")\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "Build",
		Path:        dir,
		FilePattern: "*.py",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     false,
		Multiline:   false,
	})

	if !strings.Contains(result, "[ref]") {
		t.Fatalf("expected regex fallback classification to remain [ref], got:\n%s", result)
	}
	if strings.Contains(result, "[string]") || strings.Contains(result, "[call]") {
		t.Fatalf("did not expect AST-only tags for non-Go file, got:\n%s", result)
	}
}

func TestSearchCode_MultiPatternReclassifyWithAST_UsesEachPattern(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "build.go")
	content := `package main

func Build() {}

func callBuild() {
	Build()
}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "Build,callBuild",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     false,
		Multiline:   false,
	})

	sectionFor := func(header string) string {
		start := strings.Index(result, header)
		if start < 0 {
			return ""
		}
		rest := result[start+len(header):]
		next := strings.Index(rest, "━━ Pattern ")
		if next < 0 {
			return rest
		}
		return rest[:next]
	}

	buildHeader := "━━ Pattern 1/2: \"Build\" ━━"
	callBuildHeader := "━━ Pattern 2/2: \"callBuild\" ━━"
	buildSection := sectionFor(buildHeader)
	if buildSection == "" {
		t.Fatalf("expected %s in result, got:\n%s", buildHeader, result)
	}
	if !strings.Contains(buildSection, "[call]") {
		t.Fatalf("expected Build section to keep AST call classification, got:\n%s", buildSection)
	}

	callBuildSection := sectionFor(callBuildHeader)
	if callBuildSection == "" {
		t.Fatalf("expected %s in result, got:\n%s", callBuildHeader, result)
	}
	if !strings.Contains(callBuildSection, "[def]") {
		t.Fatalf("expected callBuild section to be classified as definition, got:\n%s", callBuildSection)
	}
	if strings.Contains(callBuildSection, "[ref]") {
		t.Fatalf("did not expect callBuild section to fall back to [ref], got:\n%s", callBuildSection)
	}
}

func TestExtractTargetName(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		isRegex bool
		want    string
	}{
		{name: "fixed string", pattern: "Build", isRegex: false, want: "Build"},
		{name: "regex without meta", pattern: "Build", isRegex: true, want: "Build"},
		{name: "regex with meta", pattern: "Build.*", isRegex: true, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractTargetName(tt.pattern, tt.isRegex); got != tt.want {
				t.Fatalf("extractTargetName(%q, %v) = %q, want %q", tt.pattern, tt.isRegex, got, tt.want)
			}
		})
	}
}

func TestMatchClassToType(t *testing.T) {
	tests := []struct {
		name  string
		class internalast.MatchClass
		want  MatchType
	}{
		{name: "def", class: internalast.ClassDef, want: MatchTypeDefinition},
		{name: "import", class: internalast.ClassImport, want: MatchTypeImport},
		{name: "call", class: internalast.ClassCall, want: MatchTypeCall},
		{name: "comment", class: internalast.ClassComment, want: MatchTypeComment},
		{name: "string", class: internalast.ClassString, want: MatchTypeString},
		{name: "ref", class: internalast.ClassRef, want: MatchTypeRef},
		{name: "unknown", class: internalast.ClassUnknown, want: MatchTypeRef},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchClassToType(tt.class); got != tt.want {
				t.Fatalf("matchClassToType(%s) = %v, want %v", tt.class, got, tt.want)
			}
		})
	}
}
