package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func providerHistoryWriteFileReplacementHistory(t *testing.T, callID, args, result string) []api.Message {
	t.Helper()
	return []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments(callID, "write_file", args)),
		providerHistoryToolResult(callID, "write_file", result),
		api.Message{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}
}

func providerHistoryLargeWriteFileContent() string {
	return strings.Repeat("package main\n\nfunc generated() string { return \"x\" }\n", 260)
}

func providerHistoryWriteFileArguments(t *testing.T, path, content string) string {
	t.Helper()
	return providerHistoryJSONArguments(t, map[string]string{"path": path, "content": content})
}

func providerHistoryWriteFileSuccess(content, path string) string {
	return fmt.Sprintf("Successfully wrote %d bytes (%d lines) to %s", len(content), strings.Count(content, "\n")+1, path)
}

func assertProviderHistoryWriteFileContentReplacement(t *testing.T, args, path, originalContent string) string {
	t.Helper()
	var fields map[string]string
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		t.Fatalf("projected write_file arguments are not valid JSON: %v\nargs=%s", err, args)
	}
	if fields["path"] != path {
		t.Fatalf("projected path = %q, want %q in args %s", fields["path"], path, args)
	}
	replacement := fields["content"]
	if replacement == "" || replacement == originalContent || !strings.HasPrefix(replacement, "[omitted old write_file.content; path="+path+"]") {
		t.Fatalf("projected content = %q, want write_file.content placeholder", replacement)
	}
	if strings.ContainsAny(replacement, "\n\t\"") {
		t.Fatalf("projected content = %q, want single-line quote-free placeholder", replacement)
	}
	return replacement
}
