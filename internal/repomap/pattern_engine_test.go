package repomap

import "testing"

func TestCompilePatternDefinitions_AssignsPatternsPerExtension(t *testing.T) {
	definitions := []languagePattern{
		{
			Extensions: []string{".foo", ".bar"},
			Patterns:   []string{`^alpha`, `^beta`},
		},
	}

	compiled := compilePatternDefinitions(definitions)
	if len(compiled[".foo"]) != 2 {
		t.Fatalf("compiled .foo patterns = %d, want 2", len(compiled[".foo"]))
	}
	if len(compiled[".bar"]) != 2 {
		t.Fatalf("compiled .bar patterns = %d, want 2", len(compiled[".bar"]))
	}
	if compiled[".foo"][0].String() != `^alpha` {
		t.Fatalf("first .foo pattern = %q, want %q", compiled[".foo"][0].String(), `^alpha`)
	}
}

func TestLanguagePatternEngine_SupportsAndMatches(t *testing.T) {
	engine := newLanguagePatternEngine([]languagePattern{
		{
			Extensions: []string{".foo"},
			Patterns:   []string{`^alpha`},
		},
	})

	if !engine.supports("sample.foo") {
		t.Fatal("supports(sample.foo) = false, want true")
	}
	if engine.supports("sample.txt") {
		t.Fatal("supports(sample.txt) = true, want false")
	}
	if !engine.matches("sample.foo", "alphaSymbol") {
		t.Fatal("matches(sample.foo, alphaSymbol) = false, want true")
	}
	if engine.matches("sample.foo", "betaSymbol") {
		t.Fatal("matches(sample.foo, betaSymbol) = true, want false")
	}
}

func TestExtensionForPath_Dockerfile(t *testing.T) {
	if got := extensionForPath("Dockerfile"); got != "dockerfile" {
		t.Fatalf("extensionForPath(Dockerfile) = %q, want %q", got, "dockerfile")
	}
	if got := extensionForPath("app.go"); got != ".go" {
		t.Fatalf("extensionForPath(app.go) = %q, want %q", got, ".go")
	}
}
