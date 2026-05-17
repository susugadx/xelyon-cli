package api

import (
	"context"
	"testing"
)

func TestActiveContextBlocksContext_NilAndEmptyAreNoop(t *testing.T) {
	var nilCtx context.Context
	if got := ActiveContextBlocksFromContext(nilCtx); got != nil {
		t.Fatalf("ActiveContextBlocksFromContext(nil) = %#v, want nil", got)
	}

	ctx := WithActiveContextBlocks(nilCtx, nil)
	if ctx == nil {
		t.Fatal("WithActiveContextBlocks(nil, nil) returned nil context")
	}
	if got := ActiveContextBlocksFromContext(ctx); got != nil {
		t.Fatalf("ActiveContextBlocksFromContext(empty) = %#v, want nil", got)
	}

	base := context.Background()
	if got := WithActiveContextBlocks(base, []ActiveContextBlock{}); got != base {
		t.Fatal("WithActiveContextBlocks(empty) should return the original context")
	}
}

func TestActiveContextBlocksContext_ClonesBlocks(t *testing.T) {
	original := []ActiveContextBlock{{
		Name:    "current_task_state",
		Content: "snapshot",
	}}

	ctx := WithActiveContextBlocks(context.Background(), original)
	original[0].Name = "mutated"
	original[0].Content = "mutated"

	got := ActiveContextBlocksFromContext(ctx)
	if len(got) != 1 {
		t.Fatalf("len(ActiveContextBlocksFromContext()) = %d, want 1", len(got))
	}
	if got[0].Name != "current_task_state" || got[0].Content != "snapshot" {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want original block", got[0])
	}

	got[0].Content = "changed"
	gotAgain := ActiveContextBlocksFromContext(ctx)
	if gotAgain[0].Content != "snapshot" {
		t.Fatalf("second read Content = %q, want snapshot", gotAgain[0].Content)
	}
}

func TestActiveContextBlocksContext_ClearMasksInheritedBlocks(t *testing.T) {
	parent := WithActiveContextBlocks(context.Background(), []ActiveContextBlock{{
		Name:    "current_task_state",
		Content: "parent snapshot",
	}})

	child := WithoutActiveContextBlocks(parent)
	if got := ActiveContextBlocksFromContext(child); got != nil {
		t.Fatalf("ActiveContextBlocksFromContext(child) = %#v, want nil after explicit clear", got)
	}

	parentBlocks := ActiveContextBlocksFromContext(parent)
	if len(parentBlocks) != 1 || parentBlocks[0].Content != "parent snapshot" {
		t.Fatalf("ActiveContextBlocksFromContext(parent) = %#v, want inherited parent block unchanged", parentBlocks)
	}
}

func TestResponseIDChainDisabledContext(t *testing.T) {
	var nilCtx context.Context
	if ResponseIDChainDisabledFromContext(nilCtx) {
		t.Fatal("ResponseIDChainDisabledFromContext(nil) = true, want false")
	}

	ctx := WithResponseIDChainDisabled(nilCtx)
	if ctx == nil {
		t.Fatal("WithResponseIDChainDisabled(nil) returned nil context")
	}
	if !ResponseIDChainDisabledFromContext(ctx) {
		t.Fatal("ResponseIDChainDisabledFromContext(ctx) = false, want true")
	}
	if ResponseIDChainDisabledFromContext(context.Background()) {
		t.Fatal("ResponseIDChainDisabledFromContext(background) = true, want false")
	}
}
