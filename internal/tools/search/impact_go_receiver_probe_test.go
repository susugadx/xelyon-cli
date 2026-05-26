package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQualifiedReceiverInspectAutoOptionsScopesFallbackSearch(t *testing.T) {
	got := qualifiedReceiverInspectAutoOptions(SearchOptions{
		ProjectMapStateKey: "state-key",
		InvocationCWD:      "/repo/app",
	}, "/repo", "/repo/sdk")

	if got.ProjectMapRootPath != "/repo" {
		t.Fatalf("ProjectMapRootPath = %q, want /repo", got.ProjectMapRootPath)
	}
	if got.FallbackReferenceSearchPath != "/repo/sdk" {
		t.Fatalf("FallbackReferenceSearchPath = %q, want /repo/sdk", got.FallbackReferenceSearchPath)
	}
	if got.ProjectMapStateKey != "state-key" {
		t.Fatalf("ProjectMapStateKey = %q, want state-key", got.ProjectMapStateKey)
	}
	if got.InvocationCWD != "/repo/app" {
		t.Fatalf("InvocationCWD = %q, want /repo/app", got.InvocationCWD)
	}
}

func TestQualifiedReceiverLocalDirFastPathClassifiesRoleAndDirectMethod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "builder.go"), []byte(`package sdk

type Builder struct{}

func (Builder) Build() string { return "" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	role, complete := qualifiedReceiverRoleFromLocalDir("Builder", dir)
	if !complete {
		t.Fatal("qualifiedReceiverRoleFromLocalDir complete = false, want true")
	}
	if role != methodProbeReceiverRoleConcrete {
		t.Fatalf("qualifiedReceiverRoleFromLocalDir = %q, want concrete", role)
	}

	direct, complete := qualifiedReceiverDirectMethodFromLocalDir("Builder", "Build", dir)
	if !complete {
		t.Fatal("qualifiedReceiverDirectMethodFromLocalDir complete = false, want true")
	}
	if !direct {
		t.Fatal("qualifiedReceiverDirectMethodFromLocalDir = false, want true")
	}
}

func TestQualifiedReceiverLocalDirFastPathClassifiesInterface(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "builder.go"), []byte(`package sdk

type Builder interface {
	Build() string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	role, complete := qualifiedReceiverRoleFromLocalDir("Builder", dir)
	if !complete {
		t.Fatal("qualifiedReceiverRoleFromLocalDir complete = false, want true")
	}
	if role != methodProbeReceiverRoleInterface {
		t.Fatalf("qualifiedReceiverRoleFromLocalDir = %q, want interface", role)
	}

	direct, complete := qualifiedReceiverDirectMethodFromLocalDir("Builder", "Build", dir)
	if !complete {
		t.Fatal("qualifiedReceiverDirectMethodFromLocalDir complete = false, want true")
	}
	if direct {
		t.Fatal("qualifiedReceiverDirectMethodFromLocalDir = true, want false for interface method")
	}
}
