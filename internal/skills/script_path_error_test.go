package skills

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveScriptPath_ReturnsTypedErrors(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "typed")
	mustWriteSkill(t, skillDir, validSkill("typed", "desc", "# body"))
	skill := ParsedSkill{Directory: skillDir}

	tests := []struct {
		name   string
		path   string
		kind   ScriptPathErrorKind
		ensure func(t *testing.T)
	}{
		{
			name: "required",
			path: "   ",
			kind: ScriptPathErrorRequired,
		},
		{
			name: "absolute",
			path: filepath.Join(root, "x.sh"),
			kind: ScriptPathErrorAbsolute,
		},
		{
			name: "traversal",
			path: "../escape.sh",
			kind: ScriptPathErrorEscapesScripts,
		},
		{
			name: "not_found",
			path: "missing.sh",
			kind: ScriptPathErrorNotFound,
		},
		{
			name: "directory",
			path: "dir",
			kind: ScriptPathErrorDirectory,
			ensure: func(t *testing.T) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(skillDir, "scripts", "dir"), 0o755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ensure != nil {
				tt.ensure(t)
			}
			_, err := ResolveScriptPath(skill, tt.path)
			if err == nil {
				t.Fatal("ResolveScriptPath() error = nil, want typed error")
			}
			var pathErr *ScriptPathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("ResolveScriptPath() error = %T, want *ScriptPathError (%v)", err, err)
			}
			if pathErr.Kind != tt.kind {
				t.Fatalf("ScriptPathError.Kind = %q, want %q", pathErr.Kind, tt.kind)
			}
		})
	}
}
