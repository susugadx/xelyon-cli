package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestStringMapEditor_Run_AddEditDeleteAndCancel(t *testing.T) {
	t.Run("add edit delete save", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("a\nbeta\ntwo\ne\n1\nONE\nd\n2\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringMapEditorWithRuntime("command_aliases", map[string]string{"alpha": "one"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		if len(got) != 1 || got["alpha"] != "ONE" {
			t.Fatalf("Run() value = %#v, want alpha=ONE only", got)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringMapEditorWithRuntime("command_aliases", map[string]string{"alpha": "one"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if got != nil || changed {
			t.Fatalf("Run() = %#v, %v; want nil, false", got, changed)
		}
	})
}
