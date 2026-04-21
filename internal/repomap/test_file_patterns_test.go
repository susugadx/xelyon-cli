package repomap

import "testing"

func TestTestFileHelpersAndWrappers(t *testing.T) {
	testFiles := []string{
		"service_test.go",
		"tests/api/test_builder.py",
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

func TestTestSortBase_DoesNotTreatContestAsTest(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "contest.java", want: "contest.java"},
		{name: "Contest.kt", want: "contest.kt"},
		{name: "MyContest.cs", want: "mycontest.cs"},
		{name: "Contest.php", want: "contest.php"},
		{name: "UserServiceTest.java", want: "userservice.java"},
		{name: "UserServiceTests.kt", want: "userservice.kt"},
		{name: "user_service_test.php", want: "user_service.php"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testSortBase(tt.name); got != tt.want {
				t.Fatalf("testSortBase(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
