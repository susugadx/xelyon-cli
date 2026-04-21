package search

import "testing"

func TestBuildRipgrepSearchArgs_ReflectsSearchOptions(t *testing.T) {
	opts := SearchOptions{
		FileType:       "go",
		IsRegex:        false,
		Multiline:      true,
		IncludeHidden:  true,
		IncludeIgnored: true,
		CtxLines:       3,
		ignoreGlobs:    []string{"!vendor/**"},
	}

	got := buildRipgrepSearchArgs("targetPattern", opts, ".")
	want := []string{
		"--json",
		"-n",
		"--context", "3",
		"--glob", "*.go",
		"--fixed-strings",
		"--multiline",
		"--hidden",
		"--no-ignore",
		"--glob", "!vendor/**",
		"targetPattern",
		".",
	}
	if len(got) != len(want) {
		t.Fatalf("buildRipgrepSearchArgs() len = %d, want %d\n got=%q\nwant=%q", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("buildRipgrepSearchArgs()[%d] = %q, want %q\n got=%q\nwant=%q", i, got[i], want[i], got, want)
		}
	}
}

func TestBuildGrepSearchArgs_GNUGrepHiddenExcludeAndMultilineWarning(t *testing.T) {
	opts := SearchOptions{
		FileType: "go",
		IsRegex:  true,
		CtxLines: 2,
	}
	opts.Multiline = true

	args, warnings := buildGrepSearchArgs("targetPattern", opts, ".", true)

	for _, mustHave := range []string{
		"-rn",
		"-I",
		"-E",
		"--exclude=.[!.]*",
		"--exclude=..?*",
		"--exclude-dir=.[!.]*",
		"--exclude-dir=..?*",
		"--include=*.go",
		"-C",
		"2",
		"targetPattern",
		".",
	} {
		if !testSearchArgsContains(args, mustHave) {
			t.Fatalf("buildGrepSearchArgs() missing arg %q in %q", mustHave, args)
		}
	}

	if len(warnings) != 1 || warnings[0] != "Warning: multiline search is not supported in grep fallback mode (rg not found)" {
		t.Fatalf("buildGrepSearchArgs() warnings = %q, want only multiline warning", warnings)
	}
}

func TestBuildGrepSearchArgs_NonGNUGrepWarnsHiddenExclusionLimit(t *testing.T) {
	opts := SearchOptions{
		IsRegex: false,
	}

	args, warnings := buildGrepSearchArgs("targetPattern", opts, ".", false)

	if !testSearchArgsContains(args, "-F") {
		t.Fatalf("buildGrepSearchArgs() should use fixed-string mode with is_regex=false, got %q", args)
	}
	for _, hiddenArg := range []string{
		"--exclude=.[!.]*",
		"--exclude=..?*",
		"--exclude-dir=.[!.]*",
		"--exclude-dir=..?*",
	} {
		if testSearchArgsContains(args, hiddenArg) {
			t.Fatalf("buildGrepSearchArgs() should not include %q for non-GNU grep, got %q", hiddenArg, args)
		}
	}
	if len(warnings) != 1 || warnings[0] != "Warning: hidden-file exclusion is not fully supported in grep fallback mode on non-GNU grep" {
		t.Fatalf("buildGrepSearchArgs() warnings = %q, want hidden exclusion warning", warnings)
	}
}

func TestBuildGrepSearchArgs_IncludeHiddenAddsPartialSupportWarning(t *testing.T) {
	opts := SearchOptions{
		IncludeHidden: true,
		Multiline:     true,
	}

	_, warnings := buildGrepSearchArgs("targetPattern", opts, ".", true)
	if len(warnings) != 2 {
		t.Fatalf("buildGrepSearchArgs() warning count = %d, want 2 (%q)", len(warnings), warnings)
	}
	if warnings[0] != "Warning: include_hidden is partially supported in grep fallback mode" {
		t.Fatalf("buildGrepSearchArgs() first warning = %q, want include_hidden warning", warnings[0])
	}
	if warnings[1] != "Warning: multiline search is not supported in grep fallback mode (rg not found)" {
		t.Fatalf("buildGrepSearchArgs() second warning = %q, want multiline warning", warnings[1])
	}
}

func testSearchArgsContains(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
