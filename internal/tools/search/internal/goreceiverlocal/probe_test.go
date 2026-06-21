package goreceiverlocal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoleFromDirAndHasDirectMethod_ClassifiesConcreteReceiver(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "builder.go"), []byte(`package sdk

type Builder struct{}

func (Builder) Build() string { return "" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	role, complete := RoleFromDir("Builder", dir)
	if !complete {
		t.Fatal("RoleFromDir complete = false, want true")
	}
	if role != RoleConcrete {
		t.Fatalf("RoleFromDir = %q, want concrete", role)
	}

	direct, complete := HasDirectMethod("Builder", "Build", dir)
	if !complete {
		t.Fatal("HasDirectMethod complete = false, want true")
	}
	if !direct {
		t.Fatal("HasDirectMethod = false, want true")
	}
}

func TestRoleFromDirAndHasDirectMethod_ClassifiesInterfaceReceiver(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "builder.go"), []byte(`package sdk

type Builder interface {
	Build() string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	role, complete := RoleFromDir("Builder", dir)
	if !complete {
		t.Fatal("RoleFromDir complete = false, want true")
	}
	if role != RoleInterface {
		t.Fatalf("RoleFromDir = %q, want interface", role)
	}

	direct, complete := HasDirectMethod("Builder", "Build", dir)
	if !complete {
		t.Fatal("HasDirectMethod complete = false, want true")
	}
	if direct {
		t.Fatal("HasDirectMethod = true, want false for interface method")
	}
}
