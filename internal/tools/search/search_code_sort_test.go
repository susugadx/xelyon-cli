package search

import "testing"

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", false},
		{"main_test.go", true},
		{"src/handler.go", false},
		{"src/handler_test.go", true},
		{"app.test.js", true},
		{"app.test.ts", true},
		{"app.spec.js", true},
		{"app.spec.ts", true},
		{"test_helper.py", true},
		{"utils.py", false},
		// Python
		{"models_test.py", true},
		{"conftest.py", true},
		{"tests/test_models.py", true},
		// Rust
		{"tests/integration.rs", true},
		{"src/lib.rs", false},
		// Java/Kotlin
		{"UserServiceTest.java", true},
		{"UserServiceTests.java", true},
		{"ConfigSpec.kt", true},
		// Java false positives
		{"contest.java", false},
		{"protest.java", false},
		{"manifest.java", false},
		// C#
		{"OrderServiceTests.cs", true},
		// PHP
		{"UserRepositoryTest.php", true},
		// Ruby
		{"user_spec.rb", true},
		{"user_test.rb", true},
		{"user.rb", false},
		// Swift
		{"ConfigTests.swift", true},
		// Scala
		{"ConfigSpec.scala", true},
		// Elixir
		{"user_test.exs", true},
		// Lua
		{"utils_test.lua", true},
		// C/C++
		{"config_test.cpp", true},
		{"tests/main.cpp", true},
		{"src/main.cpp", false},
	}
	for _, tt := range tests {
		got := isTestFile(tt.path)
		if got != tt.expected {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestSortResultsByPriority(t *testing.T) {
	results := []SearchResult{
		{FilePath: "handler_test.go", Matches: []Match{{IsMatch: true, Type: MatchTypeUsage}}},
		{FilePath: "handler.go", Matches: []Match{{IsMatch: true, Type: MatchTypeUsage}}},
	}
	sortResultsByPriority(results)

	if results[0].FilePath != "handler.go" {
		t.Errorf("Expected non-test file first, got %s", results[0].FilePath)
	}
	if results[1].FilePath != "handler_test.go" {
		t.Errorf("Expected test file second, got %s", results[1].FilePath)
	}
}

func TestSortResultsByPriority_DefinitionFirst(t *testing.T) {
	results := []SearchResult{
		{FilePath: "caller.go", Matches: []Match{{IsMatch: true, Type: MatchTypeUsage}}},
		{FilePath: "define.go", Matches: []Match{{IsMatch: true, Type: MatchTypeDefinition}}},
	}
	sortResultsByPriority(results)

	if results[0].FilePath != "define.go" {
		t.Errorf("Expected definition file first, got %s", results[0].FilePath)
	}
	if results[1].FilePath != "caller.go" {
		t.Errorf("Expected usage-only file second, got %s", results[1].FilePath)
	}
}
