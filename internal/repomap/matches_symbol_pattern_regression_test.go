package repomap

import "testing"

func TestPatterns_MultiLanguageRegression(t *testing.T) {
	positive := []struct {
		name string
		path string
		line string
	}{
		{name: "Java interface", path: "src/Runner.java", line: "public interface Runner {"},
		{name: "Kotlin data class", path: "src/Runner.kt", line: "data class Runner(val id: String)"},
		{name: "Ruby module", path: "lib/runner.rb", line: "  module Runner"},
		{name: "PHP function", path: "web/runner.php", line: "protected static function buildMap() {"},
		{name: "Swift protocol", path: "ios/Runner.swift", line: "public protocol Runner {"},
		{name: "Scala case class", path: "src/Runner.scala", line: "case class Runner(id: String)"},
		{name: "Shell function", path: "scripts/build.sh", line: "function build_map"},
	}
	for _, tt := range positive {
		t.Run(tt.name, func(t *testing.T) {
			if !matchesSymbolPattern(tt.path, tt.line) {
				t.Fatalf("matchesSymbolPattern(%q, %q) = false, want true", tt.path, tt.line)
			}
		})
	}

	negative := []struct {
		name string
		path string
		line string
	}{
		{name: "Java statement", path: "src/Runner.java", line: "if (ready) {"},
		{name: "Kotlin assignment", path: "src/Runner.kt", line: "val runner = Runner()"},
		{name: "Ruby call", path: "lib/runner.rb", line: "puts runner"},
		{name: "PHP echo", path: "web/runner.php", line: "echo $runner;"},
		{name: "Swift call", path: "ios/Runner.swift", line: "print(runner)"},
		{name: "Scala call", path: "src/Runner.scala", line: "println(runner)"},
		{name: "Shell command", path: "scripts/build.sh", line: "echo build"},
	}
	for _, tt := range negative {
		t.Run(tt.name, func(t *testing.T) {
			if matchesSymbolPattern(tt.path, tt.line) {
				t.Fatalf("matchesSymbolPattern(%q, %q) = true, want false", tt.path, tt.line)
			}
		})
	}
}
