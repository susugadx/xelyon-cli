package search

import "testing"

func TestNormalizeSearchMode_DefaultsToAuto(t *testing.T) {
	got, ok := normalizeSearchMode("", false, false)
	if !ok {
		t.Fatal("expected default mode to be valid")
	}
	if got != SearchModeAuto {
		t.Fatalf("normalizeSearchMode() = %q, want %q", got, SearchModeAuto)
	}
}

func TestNormalizeSearchMode_LegacyIsRegexMapping(t *testing.T) {
	tests := []struct {
		name             string
		mode             string
		legacyIsRegex    bool
		legacyIsRegexSet bool
		want             SearchMode
	}{
		{name: "legacy true", legacyIsRegex: true, legacyIsRegexSet: true, want: SearchModeRegex},
		{name: "legacy false", legacyIsRegex: false, legacyIsRegexSet: true, want: SearchModeLiteral},
		{name: "explicit mode wins", mode: "auto", legacyIsRegex: true, legacyIsRegexSet: true, want: SearchModeAuto},
		{name: "direct compat true", legacyIsRegex: true, want: SearchModeRegex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeSearchMode(tt.mode, tt.legacyIsRegex, tt.legacyIsRegexSet)
			if !ok {
				t.Fatal("expected valid normalization")
			}
			if got != tt.want {
				t.Fatalf("normalizeSearchMode(%q, %v, %v) = %q, want %q", tt.mode, tt.legacyIsRegex, tt.legacyIsRegexSet, got, tt.want)
			}
		})
	}
}

func TestNormalizeSearchOptions_ModeCanonicalization(t *testing.T) {
	tests := []struct {
		name      string
		opts      SearchOptions
		wantMode  string
		wantRegex bool
	}{
		{name: "auto wins over legacy regex", opts: SearchOptions{Mode: string(SearchModeAuto), IsRegex: true, LegacyIsRegexSet: true}, wantMode: string(SearchModeAuto), wantRegex: false},
		{name: "symbol clears regex", opts: SearchOptions{Mode: string(SearchModeSymbol), IsRegex: true, LegacyIsRegexSet: true}, wantMode: string(SearchModeSymbol), wantRegex: false},
		{name: "regex forces regex", opts: SearchOptions{Mode: string(SearchModeRegex), IsRegex: false, LegacyIsRegexSet: true}, wantMode: string(SearchModeRegex), wantRegex: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeSearchOptions(tt.opts)
			if !ok {
				t.Fatal("expected valid normalization")
			}
			if got.Mode != tt.wantMode {
				t.Fatalf("Mode = %q, want %q", got.Mode, tt.wantMode)
			}
			if got.IsRegex != tt.wantRegex {
				t.Fatalf("IsRegex = %v, want %v", got.IsRegex, tt.wantRegex)
			}
		})
	}
}

func TestNormalizeSearchMode_Invalid(t *testing.T) {
	if _, ok := normalizeSearchMode("invalid", false, false); ok {
		t.Fatal("expected invalid mode to fail normalization")
	}
}
