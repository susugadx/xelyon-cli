package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestStringSliceEditor_Run_DeleteAndCancel(t *testing.T) {
	t.Run("delete and save", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("d\n1\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringSliceEditorWithRuntime("hooks.on_completion", []string{"one", "two"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		if len(got) != 1 || got[0] != "two" {
			t.Fatalf("Run() value = %#v, want [two]", got)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringSliceEditorWithRuntime("hooks.on_completion", []string{"one"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if got != nil || changed {
			t.Fatalf("Run() = %#v, %v; want nil, false", got, changed)
		}
	})
}
