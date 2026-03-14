package navigation

import (
	"path/filepath"
	"testing"
)

func TestFindRelatedTests_DeterministicRepresentativeInSummary(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"a_test.go": `package example

import "testing"

func TestBuildAlpha(t *testing.T) {
	_ = 1
}
`,
		"z_test.go": `package example

import "testing"

func TestBuildZeta(t *testing.T) {
	_ = 2
}
`,
	})

	refs := []Reference{
		{File: filepath.Join(dir, "z_test.go"), IsTest: true},
		{File: filepath.Join(dir, "a_test.go"), IsTest: true},
	}

	got, total, more := findRelatedTests("Build", refs, 1)
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if !more {
		t.Fatalf("expected more=true when limit is smaller than total")
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 representative test, got %d", len(got))
	}
	if got[0].File != "a_test.go" {
		t.Fatalf("expected deterministic first survivor a_test.go, got %+v", got[0])
	}
	if got[0].Name != "TestBuildAlpha" {
		t.Fatalf("expected TestBuildAlpha as representative survivor, got %+v", got[0])
	}
}

func TestFindRelatedTests_ExcludesUnrelatedTests(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"build_test.go": `package example

import "testing"

func TestBuildMain(t *testing.T) {
	_ = 1
}
`,
		"misc_test.go": `package example

import "testing"

func TestHealth(t *testing.T) {
	_ = 2
}
`,
	})

	refs := []Reference{
		{File: filepath.Join(dir, "build_test.go"), IsTest: true},
		{File: filepath.Join(dir, "misc_test.go"), IsTest: true},
	}

	got, total, more := findRelatedTests("Build", refs, 10)
	if total != 1 {
		t.Fatalf("expected total=1 related test, got %d (%+v)", total, got)
	}
	if more {
		t.Fatalf("expected more=false when all related tests are returned")
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 related test, got %d", len(got))
	}
	if got[0].File != "build_test.go" || got[0].Name != "TestBuildMain" {
		t.Fatalf("expected only build_test.go/TestBuildMain survivor, got %+v", got[0])
	}
}
