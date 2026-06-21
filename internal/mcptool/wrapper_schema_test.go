package mcptool

import (
	"context"
	"encoding/json"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"testing"
)

func TestWrapperRunPassesSchemaStructuredArgs(t *testing.T) {
	caller := &argsRecordingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "create_issue",
		Approval:   mcpapproval.ModeAuto,
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"labels":{"type":"array"},
				"metadata":{"type":"object"},
				"count":{"type":"integer"},
				"title":{"type":"string"}
			}
		}`),
	})

	result, fileChange, err := wrapper.Run(newAutoApprovedExecutionContext(context.Background()), map[string]string{
		"labels":   `["bug","urgent"]`,
		"metadata": `{"issue":123}`,
		"count":    "2",
		"title":    "Fix MCP",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if result != "called" {
		t.Fatalf("result = %q, want called", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	labels, ok := caller.args["labels"].([]any)
	if !ok || len(labels) != 2 || labels[0] != "bug" || labels[1] != "urgent" {
		t.Fatalf("labels = %#v, want []any with bug/urgent", caller.args["labels"])
	}
	metadata, ok := caller.args["metadata"].(map[string]any)
	if !ok || metadata["issue"] != float64(123) {
		t.Fatalf("metadata = %#v, want map with issue 123", caller.args["metadata"])
	}
	if caller.args["count"] != int64(2) {
		t.Fatalf("count = %#v, want int64(2)", caller.args["count"])
	}
	if caller.args["title"] != "Fix MCP" {
		t.Fatalf("title = %#v, want original string", caller.args["title"])
	}
}

func TestWrapperConvertArgsKeepsScalarWhenSchemaParseFails(t *testing.T) {
	wrapper := NewWrapper(WrapperOptions{
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"count":{"type":"integer"},
				"enabled":{"type":"boolean"},
				"ratio":{"type":"number"}
			}
		}`),
	})

	got := wrapper.ConvertArgsWithSchema(map[string]string{
		"count":   "not-an-int",
		"enabled": "not-a-bool",
		"ratio":   "not-a-number",
	})

	for key, want := range map[string]string{
		"count":   "not-an-int",
		"enabled": "not-a-bool",
		"ratio":   "not-a-number",
	} {
		if got[key] != want {
			t.Fatalf("converted[%s] = %#v, want scalar fallback %q", key, got[key], want)
		}
	}
}
