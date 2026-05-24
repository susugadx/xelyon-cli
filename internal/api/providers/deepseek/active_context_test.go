package deepseek

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
)

func TestBuildChatCompletionsRequest_IncludesActiveContextFromContext(t *testing.T) {
	evidence := deepSeekTestRehydratedEvidence()
	ctx := api.WithActiveContextBlocks(api.WithToolUseDisabled(context.Background()), []api.ActiveContextBlock{{
		Name:    "provider_history_rehydrated_evidence",
		Content: evidence,
	}})

	req, _ := New("test-key").buildChatCompletionsRequest(
		ctx,
		"System",
		[]api.Message{{Role: "user", Content: "Hello"}},
		"deepseek-v4-flash",
	)

	if len(req.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want system + active context + history", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || req.Messages[0].Content != "System" {
		t.Fatalf("Messages[0] = %#v, want system message", req.Messages[0])
	}
	if req.Messages[1].Role != "system" || req.Messages[1].Content != evidence {
		t.Fatalf("Messages[1] = %#v, want rehydrated evidence active context", req.Messages[1])
	}
	if req.Messages[2].Role != "user" || req.Messages[2].Content != "Hello" {
		t.Fatalf("Messages[2] = %#v, want original history message", req.Messages[2])
	}
}

func deepSeekTestRehydratedEvidence() string {
	return ledger.RenderRehydratedEvidenceBlock(ledger.RehydratedEvidenceBlock{Items: []ledger.RehydratedEvidenceItem{{
		Path:       "README.md",
		StartLine:  1,
		EndLine:    2,
		Source:     "read_file",
		Reason:     ledger.RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_read",
		Content:    "line one\nline two",
	}}})
}
