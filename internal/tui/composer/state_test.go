package composer

import (
	"strings"
	"testing"
)

func TestStateHasDraftAndSubmittableContent(t *testing.T) {
	t.Run("empty state is not draft or submittable", func(t *testing.T) {
		var state State

		if state.HasDraft(" \n\t ") {
			t.Fatal("HasDraft() = true, want false for whitespace-only empty state")
		}
		if state.HasSubmittableContent(" \n\t ") {
			t.Fatal("HasSubmittableContent() = true, want false for whitespace-only empty state")
		}
	})

	t.Run("whitespace text part is draft but not submittable", func(t *testing.T) {
		state := State{Parts: []Part{{Kind: PartText, Text: "  \n"}}}

		if !state.HasDraft("") {
			t.Fatal("HasDraft() = false, want true when folded state has a text part")
		}
		if state.HasSubmittableContent("") {
			t.Fatal("HasSubmittableContent() = true, want false for whitespace-only text part")
		}
	})

	t.Run("paste block content is submittable without input", func(t *testing.T) {
		var state State
		state.AppendPasteBlock("pasted content")

		if !state.HasSubmittableContent("") {
			t.Fatal("HasSubmittableContent() = false, want true for paste block content")
		}
	})

	t.Run("missing paste block reference is not submittable", func(t *testing.T) {
		state := State{Parts: []Part{{Kind: PartPaste, PasteUID: 99}}}

		if state.HasSubmittableContent("") {
			t.Fatal("HasSubmittableContent() = true, want false for missing paste block reference")
		}
	})
}

func TestStateBuildPayloadPreservesPartOrder(t *testing.T) {
	var state State
	state.AppendText("before\n")
	state.AppendPasteBlock("pasted\nbody")
	state.AppendText("\nafter")

	got := state.BuildPayload("\ninput")
	want := "before\npasted\nbody\nafter\ninput"
	if got != want {
		t.Fatalf("BuildPayload() = %q, want %q", got, want)
	}
}

func TestStateAppendPasteBlockRecordsMetadata(t *testing.T) {
	var state State
	content := "first line\n日本語"

	state.AppendPasteBlock(content)

	block, ok := state.FindPasteBlock(1)
	if !ok {
		t.Fatal("FindPasteBlock(1) ok = false, want true")
	}
	if block.Content != content {
		t.Fatalf("Content = %q, want %q", block.Content, content)
	}
	if block.CharCount != 14 {
		t.Fatalf("CharCount = %d, want 14", block.CharCount)
	}
	if block.LineCount != 2 {
		t.Fatalf("LineCount = %d, want 2", block.LineCount)
	}
	if state.NextPasteUID != 1 {
		t.Fatalf("NextPasteUID = %d, want 1", state.NextPasteUID)
	}
}

func TestStateAppendTextMergesAdjacentTextParts(t *testing.T) {
	var state State
	state.AppendText("hello")
	state.AppendText(" world")

	if len(state.Parts) != 1 {
		t.Fatalf("parts = %d, want 1 merged text part", len(state.Parts))
	}
	if state.Parts[0].Text != "hello world" {
		t.Fatalf("merged text = %q, want %q", state.Parts[0].Text, "hello world")
	}

	state.AppendPasteBlock("paste")
	state.AppendText(" tail")
	if len(state.Parts) != 3 {
		t.Fatalf("parts after paste = %d, want 3", len(state.Parts))
	}
}

func TestStateRemoveLastPasteBlockReturnsAllTrailingText(t *testing.T) {
	var state State
	state.AppendText("prefix")
	state.AppendPasteBlock("first paste")
	state.AppendText(" after first")
	state.AppendPasteBlock("second paste")
	state.AppendText(" tail")

	trailing, ok := state.RemoveLastPasteBlock()
	if !ok {
		t.Fatal("RemoveLastPasteBlock() ok = false, want true")
	}
	if trailing != " after first tail" {
		t.Fatalf("trailing text = %q, want %q", trailing, " after first tail")
	}
	if _, ok := state.FindPasteBlock(2); ok {
		t.Fatal("FindPasteBlock(2) ok = true, want false after removing last paste block")
	}

	got := state.BuildPayload("")
	want := "prefixfirst paste"
	if got != want {
		t.Fatalf("BuildPayload() after remove = %q, want %q", got, want)
	}
}

func TestStatePopTrailingTextPreservesOriginalOrder(t *testing.T) {
	state := State{
		Parts: []Part{
			{Kind: PartText, Text: "head"},
			{Kind: PartPaste, PasteUID: 1},
			{Kind: PartText, Text: " tail-1"},
			{Kind: PartText, Text: " tail-2"},
		},
		PasteBlocks: []PasteBlock{{UID: 1, Content: "paste"}},
	}

	got := state.PopTrailingText()
	if got != " tail-1 tail-2" {
		t.Fatalf("PopTrailingText() = %q, want original-order trailing text", got)
	}
	if payload := state.BuildPayload(""); payload != "headpaste" {
		t.Fatalf("BuildPayload() after pop = %q, want %q", payload, "headpaste")
	}
}

func TestStateVisibleRowsReturnsLastRowsWithPasteNumbers(t *testing.T) {
	var state State
	state.AppendText("one")
	state.AppendPasteBlock("paste one")
	state.AppendText("two")
	state.AppendPasteBlock("paste two")

	rows := state.VisibleRows(3)
	if len(rows) != 3 {
		t.Fatalf("VisibleRows(3) length = %d, want 3", len(rows))
	}
	if rows[0].Kind != PartPaste || rows[0].PasteBlock.Number != 1 {
		t.Fatalf("row[0] = %#v, want first paste row with number 1", rows[0])
	}
	if rows[1].Kind != PartText || rows[1].Text != "two" {
		t.Fatalf("row[1] = %#v, want text row two", rows[1])
	}
	if rows[2].Kind != PartPaste || rows[2].PasteBlock.Number != 2 {
		t.Fatalf("row[2] = %#v, want second paste row with number 2", rows[2])
	}

	if got := state.VisibleRows(0); got != nil {
		t.Fatalf("VisibleRows(0) = %#v, want nil", got)
	}
}

func TestStateClearResetsFoldedState(t *testing.T) {
	var state State
	state.AppendText("text")
	state.AppendPasteBlock(strings.Repeat("x", PasteBlockFoldThreshold+1))

	state.Clear()

	if !state.IsPlainInput() {
		t.Fatal("IsPlainInput() = false, want true after Clear")
	}
	if state.NextPasteUID != 0 {
		t.Fatalf("NextPasteUID = %d, want 0 after Clear", state.NextPasteUID)
	}
	if state.BuildPayload("input") != "input" {
		t.Fatalf("BuildPayload() after Clear = %q, want input", state.BuildPayload("input"))
	}
}
