package uistyle

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestIsTrueColorCapable_FalseCases(t *testing.T) {
	originalNoColor := color.NoColor
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	color.NoColor = true
	if isTrueColorCapable(&bytes.Buffer{}) {
		t.Fatal("isTrueColorCapable() = true, want false when color is disabled")
	}

	color.NoColor = false
	t.Setenv("COLORTERM", "truecolor")
	if isTrueColorCapable(&bytes.Buffer{}) {
		t.Fatal("isTrueColorCapable() = true, want false for non-terminal writer")
	}
}

func TestShouldUseTrueColor(t *testing.T) {
	tests := []struct {
		name       string
		noColor    bool
		isTerminal bool
		colorTerm  string
		want       bool
	}{
		{name: "disabled when noColor", noColor: true, isTerminal: true, colorTerm: "truecolor", want: false},
		{name: "disabled when not terminal", noColor: false, isTerminal: false, colorTerm: "truecolor", want: false},
		{name: "enabled for truecolor", noColor: false, isTerminal: true, colorTerm: "truecolor", want: true},
		{name: "enabled for 24bit", noColor: false, isTerminal: true, colorTerm: "24bit", want: true},
		{name: "disabled for plain color term", noColor: false, isTerminal: true, colorTerm: "screen", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUseTrueColor(tt.noColor, tt.isTerminal, tt.colorTerm)
			if got != tt.want {
				t.Fatalf("shouldUseTrueColor(%v, %v, %q) = %v, want %v", tt.noColor, tt.isTerminal, tt.colorTerm, got, tt.want)
			}
		})
	}
}

func TestTrueColorWriters_EmitEscapeSequences(t *testing.T) {
	var buf bytes.Buffer

	writeTCFgBg(&buf, 1, 2, 3, 4, 5, 6, "hello")
	writeTCFg(&buf, 7, 8, 9, "world")

	got := buf.String()
	if !strings.Contains(got, "\x1b[38;2;1;2;3m\x1b[48;2;4;5;6mhello\x1b[0m") {
		t.Fatalf("writeTCFgBg output = %q, want truecolor escape", got)
	}
	if !strings.Contains(got, "\x1b[38;2;7;8;9mworld\x1b[0m") {
		t.Fatalf("writeTCFg output = %q, want truecolor escape", got)
	}
}

func TestNewFileOpPalette_SelectsPlainOr16Color(t *testing.T) {
	originalNoColor := color.NoColor
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	color.NoColor = true
	plain := NewFileOpPalette(&bytes.Buffer{})
	var plainBuf bytes.Buffer
	plain.Accent(&plainBuf, "plain")
	if plainBuf.String() != "plain" {
		t.Fatalf("plain palette output = %q, want plain text", plainBuf.String())
	}

	color.NoColor = false
	sixteen := NewFileOpPalette(&bytes.Buffer{})
	var sixteenBuf bytes.Buffer
	sixteen.Accent(&sixteenBuf, "color")
	if !strings.Contains(sixteenBuf.String(), "color") {
		t.Fatalf("16-color palette output = %q, want label text", sixteenBuf.String())
	}
}

func TestNewFileOpPaletteForCapabilities_SelectsAllModes(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		pal := newFileOpPaletteForCapabilities(true, false, "")
		var buf bytes.Buffer
		pal.Accent(&buf, "plain")
		if buf.String() != "plain" {
			t.Fatalf("plain palette output = %q, want plain text", buf.String())
		}
	})

	t.Run("truecolor", func(t *testing.T) {
		pal := newFileOpPaletteForCapabilities(false, true, "truecolor")
		var buf bytes.Buffer
		pal.Accent(&buf, "color")
		if !strings.Contains(buf.String(), "\x1b[38;2;") {
			t.Fatalf("truecolor palette output = %q, want ANSI truecolor sequence", buf.String())
		}
	})

	t.Run("16-color fallback", func(t *testing.T) {
		pal := newFileOpPaletteForCapabilities(false, true, "screen")
		var buf bytes.Buffer
		pal.Accent(&buf, "color")
		if !strings.Contains(buf.String(), "color") {
			t.Fatalf("16-color fallback output = %q, want label text", buf.String())
		}
	})
}

func TestNewTrueColorFileOpPalette_UsesANSI(t *testing.T) {
	pal := newTrueColorFileOpPalette()
	var buf bytes.Buffer

	pal.AddLine(&buf, "+1")
	pal.DelLine(&buf, "-1")
	pal.Hunk(&buf, "@@")
	pal.Accent(&buf, "path")
	pal.Muted(&buf, "muted")
	pal.Border(&buf, "border")
	pal.Context(&buf, "context")

	got := buf.String()
	if strings.Count(got, "\x1b[") < 7 {
		t.Fatalf("truecolor palette output = %q, want ANSI sequences", got)
	}
}
