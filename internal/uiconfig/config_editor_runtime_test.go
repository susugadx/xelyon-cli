package uiconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestNewEditors_UseDefaultRuntime(t *testing.T) {
	if editor := NewStringSliceEditor("hooks.on_completion", nil); editor.Runtime == nil {
		t.Fatal("NewStringSliceEditor() Runtime should not be nil")
	}
	if editor := NewStringMapEditor("command_aliases", nil); editor.Runtime == nil {
		t.Fatal("NewStringMapEditor() Runtime should not be nil")
	}
	if editor := NewStructMapEditor("lsp.servers", config.FieldTypeStructMap); editor.Runtime == nil {
		t.Fatal("NewStructMapEditor() Runtime should not be nil")
	}
}

func TestReadLineWithIO_StripsBracketedPasteAndWhitespace(t *testing.T) {
	runtime := NewRuntime(strings.NewReader("\x1b[200~  hello world  \x1b[201~\n"), &bytes.Buffer{}, &bytes.Buffer{})
	promptIO := runtime.PromptIO()

	if got := readLineWithIO(&promptIO); got != "hello world" {
		t.Fatalf("readLineWithIO() = %q, want %q", got, "hello world")
	}
}
