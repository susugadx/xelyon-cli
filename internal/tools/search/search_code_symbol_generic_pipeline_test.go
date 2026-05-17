package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func TestParseGenericSymbolMatchLine(t *testing.T) {
	match, ok := parseGenericSymbolMatchLine("pkg/service.go:42:  func Run() error {  ", SearchOptions{})
	if !ok {
		t.Fatal("expected parseGenericSymbolMatchLine to parse valid rg output")
	}
	if match.File != "pkg/service.go" {
		t.Fatalf("file = %q, want %q", match.File, "pkg/service.go")
	}
	if match.Line != 42 {
		t.Fatalf("line = %d, want %d", match.Line, 42)
	}
	if match.Content != "func Run() error {" {
		t.Fatalf("content = %q, want %q", match.Content, "func Run() error {")
	}
}

func TestCollectGenericSymbolMatches_AppliesFilterAndLimit(t *testing.T) {
	output := strings.Join([]string{
		"pkg/service.go:10:func Run(){}",
		"pkg/service.py:11:def Run():",
		"pkg/runner.go:12:Run()",
	}, "\n")

	matches := collectGenericSymbolMatches(strings.NewReader(output), SearchOptions{FileType: "go"}, 1).matches
	if len(matches) != 1 {
		t.Fatalf("matches len = %d, want %d", len(matches), 1)
	}
	if matches[0].File != "pkg/service.go" || matches[0].Line != 10 {
		t.Fatalf("unexpected first match: %+v", matches[0])
	}
}

func TestCollectGenericSymbolMatches_AppliesIgnoreMatcher(t *testing.T) {
	output := strings.Join([]string{
		"generated/service.ts:10:export function buildUser() {}",
		"src/service.ts:11:export function buildUser() {}",
	}, "\n")

	matches := collectGenericSymbolMatches(strings.NewReader(output), SearchOptions{
		FileType:      "ts",
		ignoreMatcher: pathmatch.NewMatcher([]string{"generated"}),
	}, 0).matches
	if len(matches) != 1 {
		t.Fatalf("matches len = %d, want 1: %+v", len(matches), matches)
	}
	if matches[0].File != "src/service.ts" {
		t.Fatalf("match file = %q, want src/service.ts", matches[0].File)
	}
}

func TestBuildGenericRgArgs_HonorsVisibilityAndIgnoreContract(t *testing.T) {
	args, _ := buildGenericRgArgs("buildUser", SearchOptions{
		FileType:       "ts",
		IncludeHidden:  true,
		IncludeIgnored: true,
		ignoreGlobs:    []string{"!generated/**"},
	})
	joined := " " + strings.Join(args, " ") + " "
	for _, want := range []string{" --glob *.ts ", " --hidden ", " --no-ignore ", " --glob !generated/** "} {
		if !strings.Contains(joined, want) {
			t.Fatalf("buildGenericRgArgs() = %v, want token %q", args, want)
		}
	}
}
