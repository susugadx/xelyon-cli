package api

import (
	"context"
	"testing"
)

func TestWithToolUseDisabledMarksRequestContext(t *testing.T) {
	if IsToolUseDisabled(context.Background()) {
		t.Fatal("IsToolUseDisabled() = true for default context, want false")
	}

	ctx := WithToolUseDisabled(context.Background())
	if !IsToolUseDisabled(ctx) {
		t.Fatal("IsToolUseDisabled() = false after WithToolUseDisabled, want true")
	}
}

func TestShouldSendToolPayloadCombinesProviderCapabilityAndRequestMode(t *testing.T) {
	if !ShouldSendToolPayload(context.Background(), true) {
		t.Fatal("ShouldSendToolPayload(default, true) = false, want true")
	}
	if ShouldSendToolPayload(context.Background(), false) {
		t.Fatal("ShouldSendToolPayload(default, false) = true, want false")
	}
	if ShouldSendToolPayload(WithToolUseDisabled(context.Background()), true) {
		t.Fatal("ShouldSendToolPayload(tool disabled, true) = true, want false")
	}
}
