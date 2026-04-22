package gemini

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestBuildPartAction_ThoughtTakesPriority(t *testing.T) {
	functionCall := &api.GeminiFunctionCall{Name: "read_file"}
	action := buildPartAction(GeminiFunctionPart{
		Thought:          true,
		Text:             "hidden",
		ThoughtSignature: "sig-thought",
		FunctionCall:     functionCall,
	})

	if !action.collectThought {
		t.Fatal("buildPartAction() collectThought = false, want true")
	}
	if action.collectSignature {
		t.Fatal("buildPartAction() collectSignature = true, want false")
	}
	if action.text != "" {
		t.Fatalf("buildPartAction() text = %q, want empty", action.text)
	}
	if action.functionCall != functionCall {
		t.Fatal("buildPartAction() functionCall pointer changed")
	}
}

func TestBuildPartAction_ThoughtSignatureSuppressesTextPath(t *testing.T) {
	functionCall := &api.GeminiFunctionCall{Name: "read_file"}
	action := buildPartAction(GeminiFunctionPart{
		Text:             "signature hidden text",
		ThoughtSignature: "sig-1",
		FunctionCall:     functionCall,
	})

	if !action.collectSignature {
		t.Fatal("buildPartAction() collectSignature = false, want true")
	}
	if action.text != "" {
		t.Fatalf("buildPartAction() text = %q, want empty", action.text)
	}
	if action.functionCall != functionCall {
		t.Fatal("buildPartAction() functionCall pointer changed")
	}
	if action.thoughtSignature != "sig-1" {
		t.Fatalf("buildPartAction() thoughtSignature = %q, want %q", action.thoughtSignature, "sig-1")
	}
}

func TestBuildPartAction_TextAndFunctionCallRemainInDefaultPath(t *testing.T) {
	functionCall := &api.GeminiFunctionCall{Name: "read_file"}
	action := buildPartAction(GeminiFunctionPart{
		Text:         "answer",
		FunctionCall: functionCall,
	})

	if action.collectThought || action.collectSignature {
		t.Fatal("buildPartAction() should not mark thought/signature in default path")
	}
	if action.text != "answer" {
		t.Fatalf("buildPartAction() text = %q, want %q", action.text, "answer")
	}
	if action.functionCall != functionCall {
		t.Fatal("buildPartAction() functionCall pointer changed")
	}
}

func TestInterpretTextPart_SuppressesSplitToolJSON(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)

	first := state.interpretTextPart(`{"tool":"read`)
	if first.responseText != `{"tool":"read` {
		t.Fatalf("first responseText = %q, want first chunk", first.responseText)
	}
	if first.shouldDisplay {
		t.Fatal("first shouldDisplay = true, want false for tool JSON suppression")
	}
	if !state.suppressingToolJSON {
		t.Fatal("state.suppressingToolJSON = false, want true after first chunk")
	}

	second := state.interpretTextPart(`_file","args":{"path":"/tmp/demo.txt"}}`)
	if second.responseText != `_file","args":{"path":"/tmp/demo.txt"}}` {
		t.Fatalf("second responseText = %q, want second chunk", second.responseText)
	}
	if second.shouldDisplay {
		t.Fatal("second shouldDisplay = true, want false for suppressed JSON continuation")
	}
	if state.suppressingToolJSON {
		t.Fatal("state.suppressingToolJSON = true, want false after JSON close")
	}
}

func TestInterpretTextPart_CodeBlockToolJSONSeparatesDisplayAndRescue(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)
	input := "Before\n```json\n{\"tool\":\"read_file\",\"args\":{\"path\":\"/tmp/demo.txt\"}}\n```\nAfter"

	action := state.interpretTextPart(input)
	if len(action.rescuedToolJSONs) != 1 {
		t.Fatalf("rescuedToolJSONs len = %d, want 1", len(action.rescuedToolJSONs))
	}
	if !strings.Contains(action.rescuedToolJSONs[0], `"tool":"read_file"`) {
		t.Fatalf("rescuedToolJSON = %q, want read_file", action.rescuedToolJSONs[0])
	}
	if !action.shouldDisplay {
		t.Fatal("shouldDisplay = false, want true for surrounding prose")
	}
	if !strings.Contains(action.displayText, "Before") || !strings.Contains(action.displayText, "After") {
		t.Fatalf("displayText = %q, want surrounding prose", action.displayText)
	}
	if strings.Contains(action.responseText, "```") {
		t.Fatalf("responseText = %q, want code block markers removed", action.responseText)
	}
}

func TestInterpretTextPart_CodeBlockOnlySkipsDisplay(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)

	action := state.interpretTextPart("```json\n{\"tool\":\"bash\",\"args\":{}}\n```")
	if len(action.rescuedToolJSONs) != 1 {
		t.Fatalf("rescuedToolJSONs len = %d, want 1", len(action.rescuedToolJSONs))
	}
	if action.shouldDisplay {
		t.Fatal("shouldDisplay = true, want false when only tool JSON code block exists")
	}
	if action.responseText != "" {
		t.Fatalf("responseText = %q, want empty after code block removal", action.responseText)
	}
}

func TestApplyTextAction_UpdatesResponseAndRescueState(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)

	state.applyTextAction(ctx, sseTextAction{
		responseText:     "Hello",
		rescuedToolJSONs: []string{`{"tool":"read_file"}`},
		shouldDisplay:    false,
	})
	if state.response() != "Hello" {
		t.Fatalf("response = %q, want %q", state.response(), "Hello")
	}
	if len(state.rescuedToolJSONs) != 1 {
		t.Fatalf("rescuedToolJSONs len = %d, want 1", len(state.rescuedToolJSONs))
	}
}

func TestApplyTextAction_DisplayMissingDoesNotPanic(t *testing.T) {
	ctx, _, _ := newGeminiResponseContext()
	state := newSSEInterpretState(ctx, nil, "", false)
	state.display = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("applyTextAction() panicked with nil display: %v", r)
		}
	}()

	state.applyTextAction(ctx, sseTextAction{
		responseText:  "Visible text",
		displayText:   "Visible text",
		shouldDisplay: true,
	})
	if state.response() != "Visible text" {
		t.Fatalf("response = %q, want %q", state.response(), "Visible text")
	}
}
