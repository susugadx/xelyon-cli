package agent

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func setManagerToolsForTest(t *testing.T, manager *mcp.Manager, tools []mcp.MCPTool) {
	t.Helper()

	field := reflect.ValueOf(manager).Elem().FieldByName("tools")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(tools))
}

func TestPromptConfirmWithRuntime(t *testing.T) {
	t.Run("yes returns true", func(t *testing.T) {
		var out bytes.Buffer
		runtime := ui.NewRuntime(strings.NewReader("\n"), &out, &out)
		if !promptConfirmWithRuntime(runtime, "Continue?") {
			t.Fatal("promptConfirmWithRuntime() = false, want true")
		}
	})

	t.Run("comment is treated as cancel", func(t *testing.T) {
		var out bytes.Buffer
		runtime := ui.NewRuntime(strings.NewReader("c\n\n\n"), &out, &out)
		if promptConfirmWithRuntime(runtime, "Continue?") {
			t.Fatal("promptConfirmWithRuntime() = true, want false")
		}
		if !strings.Contains(out.String(), "Treating as cancel") {
			t.Fatalf("expected cancel warning, got %q", out.String())
		}
	})
}

func TestHandleHistoryCommand_PrintsPreviewAndTruncates(t *testing.T) {
	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	agent := &Agent{
		Runtime: runtime,
		History: []api.Message{
			{Role: "user", Content: "short"},
			{Role: "assistant", Content: strings.Repeat("a", config.HistoryPreviewLen+5)},
		},
	}

	handleHistoryCommand(agent)
	got := out.String()
	for _, fragment := range []string{
		"📜 2 messages in history",
		"1. 👤 short",
		"2. 🤖 ",
		"...",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("handleHistoryCommand() output missing %q:\n%s", fragment, got)
		}
	}
}

func TestPrintHelpToWriter_WritesGeneratedHelp(t *testing.T) {
	var out bytes.Buffer
	printHelpToWriter(&out, nil)
	got := out.String()
	if !strings.Contains(got, GeneratedHelpCommandsText) {
		t.Fatalf("printHelpToWriter() missing generated commands help:\n%s", got)
	}
	if !strings.Contains(got, GeneratedHelpTipsText) {
		t.Fatalf("printHelpToWriter() missing generated tips help:\n%s", got)
	}
	if strings.Contains(got, "/review") {
		t.Fatalf("classic /help should not advertise TUI-only /review:\n%s", got)
	}
	if strings.Contains(got, "/project") {
		t.Fatalf("classic /help should not advertise TUI-only /project:\n%s", got)
	}
}

func TestPrintHelpToWriterForSurface_TUICommands(t *testing.T) {
	var out bytes.Buffer
	printHelpToWriterForSurface(&out, nil, commandcatalog.CommandSurfaceTUI)
	got := out.String()
	if !strings.Contains(got, GeneratedTUIHelpCommandsText) {
		t.Fatalf("TUI help missing generated TUI commands help:\n%s", got)
	}
	if !strings.Contains(got, "/review") {
		t.Fatalf("TUI /help should advertise TUI-only /review:\n%s", got)
	}
	if !strings.Contains(got, "/init") {
		t.Fatalf("TUI /help should advertise /init:\n%s", got)
	}
	if !strings.Contains(got, "/project") {
		t.Fatalf("TUI /help should advertise /project:\n%s", got)
	}
	if !strings.Contains(got, GeneratedHelpTipsText) {
		t.Fatalf("TUI help missing generated tips help:\n%s", got)
	}
}

func TestBuildMCPToolsPromptAndGitHubHint(t *testing.T) {
	manager := mcp.NewManager()
	setManagerToolsForTest(t, manager, []mcp.MCPTool{
		{ServerName: "github", Name: "list_issues", Description: "List GitHub issues"},
		{ServerName: "slack", Name: "send_message", Description: "Send a Slack message"},
	})

	promptText := buildMCPToolsPrompt(manager)
	for _, fragment := range []string{
		"mcp_github_list_issues",
		"List GitHub issues",
		"GitHub MCP Usage Guide",
	} {
		if !strings.Contains(promptText, fragment) {
			t.Fatalf("buildMCPToolsPrompt() missing %q:\n%s", fragment, promptText)
		}
	}

	agent := &Agent{mcpManager: manager}
	if !agent.HasGitHubMCP() {
		t.Fatal("HasGitHubMCP() = false, want true")
	}

	hinted := agent.AddGitHubHint("Open PR status")
	if !strings.Contains(hinted, "SYSTEM HINT: Use MCP GitHub tools for this request") {
		t.Fatalf("AddGitHubHint() = %q, want GitHub system hint", hinted)
	}

	unrelated := agent.AddGitHubHint("Fix the formatter")
	if unrelated != "Fix the formatter" {
		t.Fatalf("AddGitHubHint() unrelated = %q, want unchanged input", unrelated)
	}
}

func TestBuildMCPToolsPrompt_EmptyAndHasGitHubMCP_False(t *testing.T) {
	manager := mcp.NewManager()
	if got := buildMCPToolsPrompt(manager); got != "" {
		t.Fatalf("buildMCPToolsPrompt() = %q, want empty string", got)
	}

	agent := &Agent{mcpManager: manager}
	if agent.HasGitHubMCP() {
		t.Fatal("HasGitHubMCP() = true, want false")
	}

	setManagerToolsForTest(t, manager, []mcp.MCPTool{
		{ServerName: "slack", Name: "send_message", Description: "Send a Slack message"},
	})
	if agent.HasGitHubMCP() {
		t.Fatal("HasGitHubMCP() = true, want false for non-github tools")
	}
}

func TestHandleExitCommand_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_AGENT_EXIT_HELPER") != "1" {
		return
	}

	handleExitCommand(&Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard),
		},
	})
}

func TestHandleExitCommand_ExitsWithCodeZero(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestHandleExitCommand_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_AGENT_EXIT_HELPER=1")

	err = cmd.Run()
	if err == nil {
		return
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 0 {
		t.Fatalf("exit code = %d, want 0", exitErr.ExitCode())
	}
}
