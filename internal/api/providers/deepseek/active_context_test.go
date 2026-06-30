package deepseek

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
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

func TestBuildChatCompletionsRequest_KeepsImageHistoryTextOnly(t *testing.T) {
	req, _ := New("test-key").buildChatCompletionsRequest(
		api.WithToolUseDisabled(context.Background()),
		"System",
		[]api.Message{
			api.NewUserImageMessage("inspect", &api.ImageData{MediaType: "image/png", Base64: "aW1hZ2U="}),
		},
		"deepseek-v4-flash",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want system + image history", body["messages"])
	}
	imageMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("image message = %#v, want object", messages[1])
	}
	if imageMessage["content"] != "inspect" {
		t.Fatalf("image message content = %#v, want text-only content", imageMessage["content"])
	}
}

func deepSeekTestRehydratedEvidence() string {
	return taskstate.RenderRehydratedEvidenceBlock(taskstate.RehydratedEvidenceBlock{Items: []taskstate.RehydratedEvidenceItem{{
		Path:       "README.md",
		StartLine:  1,
		EndLine:    2,
		Source:     "read_file",
		Reason:     taskstate.RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_read",
		Content:    "line one\nline two",
	}}})
}
