package agent

import (
	"bytes"
	"context"
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
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type promptConfirmTestPrompter struct {
	called bool
}

func (p *promptConfirmTestPrompter) Prompt(_ context.Context, _ uiprompt.PromptRequest) (uiprompt.PromptResponse, error) {
	p.called = true
	return uiprompt.PromptResponse{Action: uiprompt.PromptActionNo}, nil
}

type promptConfirmTestTUIPrompter struct {
	promptConfirmTestPrompter
}

func (*promptConfirmTestTUIPrompter) BypassCommandConfirm() {}

func setManagerToolsForTest(t *testing.T, manager *mcp.Manager, tools []mcp.MCPTool) {
	t.Helper()

	field := reflect.ValueOf(manager).Elem().FieldByName("tools")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(tools))
}

func TestPromptConfirmWithRuntime(t *testing.T) {
	t.Run("tui prompter returns true without prompting", func(t *testing.T) {
		var out bytes.Buffer
		prompter := &promptConfirmTestTUIPrompter{}
		runtime := uiruntime.NewRuntime(strings.NewReader("n\n"), &out, &out)
		runtime.SetPrompter(prompter)

		if !promptConfirmWithRuntime(runtime, "Continue?") {
			t.Fatal("promptConfirmWithRuntime() = false, want true")
		}
		if prompter.called {
			t.Fatal("TUI prompter should not be called for slash command confirmation")
		}
	})

	t.Run("yes returns true", func(t *testing.T) {
		var out bytes.Buffer
		runtime := uiruntime.NewRuntime(strings.NewReader("\n"), &out, &out)
		if !promptConfirmWithRuntime(runtime, "Continue?") {
			t.Fatal("promptConfirmWithRuntime() = false, want true")
		}
	})

	t.Run("comment is treated as cancel", func(t *testing.T) {
		var out bytes.Buffer
		runtime := uiruntime.NewRuntime(strings.NewReader("c\n\n\n"), &out, &out)
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
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, &out)

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
	if !strings.Contains(got, "Surface: classic legacy fallback (--no-tui)") {
		t.Fatalf("classic /help should identify the legacy fallback surface:\n%s", got)
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
	if !strings.Contains(got, "Surface: TUI primary interactive surface") {
		t.Fatalf("TUI /help should identify the primary surface:\n%s", got)
	}
	if !strings.Contains(got, "Command discovery: type / in the input field for candidates") {
		t.Fatalf("TUI /help should point command discovery to slash candidates:\n%s", got)
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

func TestBuildMCPToolsPromptWithoutGitHubSpecificHint(t *testing.T) {
	manager := mcp.NewManager()
	setManagerToolsForTest(t, manager, []mcp.MCPTool{
		{ServerName: "github", Name: "list_issues", Description: "List GitHub issues"},
		{ServerName: "slack", Name: "send_message", Description: "Send a Slack message"},
	})

	promptText := buildMCPToolsPrompt(manager)
	for _, fragment := range []string{
		"mcp_github_list_issues",
		"<mcp_tools_data>",
		"Some MCP tools may be available through the tool registry",
		"Trust the actual tool result for availability, authentication, and success",
	} {
		if !strings.Contains(promptText, fragment) {
			t.Fatalf("buildMCPToolsPrompt() missing %q:\n%s", fragment, promptText)
		}
	}
	for _, fragment := range []string{
		"GitHub MCP Usage Guide",
		"SYSTEM HINT",
		"Array arguments",
		"you CAN via these MCP tools",
		"I cannot access this service",
		"List GitHub issues",
		"Send a Slack message",
	} {
		if strings.Contains(promptText, fragment) {
			t.Fatalf("buildMCPToolsPrompt() should not include GitHub-specific hint %q:\n%s", fragment, promptText)
		}
	}
}

func TestBuildMCPToolsPrompt_Empty(t *testing.T) {
	manager := mcp.NewManager()
	if got := buildMCPToolsPrompt(manager); got != "" {
		t.Fatalf("buildMCPToolsPrompt() = %q, want empty string", got)
	}
}

func TestHandleExitCommand_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_AGENT_EXIT_HELPER") != "1" {
		return
	}

	handleExitCommand(&Agent{
		Runtime: &AgentRuntime{
			UI: uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard),
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
