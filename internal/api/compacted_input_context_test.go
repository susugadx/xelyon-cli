package api

import (
	"context"
	"testing"
)

func TestCompactedInputItemsContext_ClonesItems(t *testing.T) {
	original := []InputItem{{
		Type: "message",
		Role: "user",
		Content: []InputContentPart{
			{Type: "input_text", Text: "hello"},
		},
		ThoughtParts: []map[string]any{{"signature": "sig"}},
	}}

	ctx := WithCompactedInputItems(context.Background(), original)
	original[0].Role = "mutated"
	original[0].Content.([]InputContentPart)[0].Text = "mutated"
	original[0].ThoughtParts[0]["signature"] = "mutated"

	got := CompactedInputItemsFromContext(ctx)
	if len(got) != 1 {
		t.Fatalf("len(CompactedInputItemsFromContext()) = %d, want 1", len(got))
	}
	if got[0].Role != "user" {
		t.Fatalf("Role = %q, want user", got[0].Role)
	}
	parts, ok := got[0].Content.([]InputContentPart)
	if !ok || len(parts) != 1 || parts[0].Text != "hello" {
		t.Fatalf("Content = %#v, want cloned input_text hello", got[0].Content)
	}
	if got[0].ThoughtParts[0]["signature"] != "sig" {
		t.Fatalf("ThoughtParts = %#v, want cloned signature", got[0].ThoughtParts)
	}

	got[0].Role = "changed"
	gotAgain := CompactedInputItemsFromContext(ctx)
	if gotAgain[0].Role != "user" {
		t.Fatalf("second read Role = %q, want user", gotAgain[0].Role)
	}
}
