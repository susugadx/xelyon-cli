package repomap

import "testing"

func TestTestFileHelpersAndWrappers(t *testing.T) {
	testFiles := []string{
		"service_test.go",
		"tests/api/test_builder.py",
		"src/__tests__/builder.test.ts",
		"pkg/user.spec.ts",
		"pkg/UserServiceTest.java",
		"pkg/user_service_test.php",
		"pkg/MySpec.swift",
		"pkg/specs/user_test.exs",
		"pkg/test/run_spec.lua",
	}
	for _, path := range testFiles {
		if !isTestFile(path) {
			t.Fatalf("isTestFile(%q) = false, want true", path)
		}
		if !IsTestFile(path) {
			t.Fatalf("IsTestFile(%q) = false, want true", path)
		}
	}
	if isTestFile("pkg/contest.java") {
		t.Fatal("isTestFile(\"pkg/contest.java\") = true, want false")
	}

	if !isTestSuffixName("UserServiceTests.kt") {
		t.Fatal("isTestSuffixName() should detect PascalCase Tests suffix")
	}
	if !isTestSuffixName("user_service_spec.php") {
		t.Fatal("isTestSuffixName() should detect snake_case spec suffix")
	}
	if isTestSuffixName("contest.java") {
		t.Fatal("isTestSuffixName() should not treat contest as test suffix")
	}
}
