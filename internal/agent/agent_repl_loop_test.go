package agent

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newREPLLoopTestAgent(t *testing.T, provider api.Provider, input io.Reader, out *bytes.Buffer) (*Agent, *ui.MultilineReader) {
	t.Helper()

	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.UI = ui.NewRuntime(input, out, out)

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	agent.setAutoApprove(true)

	mlReader := ui.NewMultilineReaderWithRuntime(runtime.UI)
	agent.setPromptReader(mlReader)
	return agent, mlReader
}

func TestRunREPLLoop_BlankInputAndSpecialCommandThenChat(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	pr, pw := io.Pipe()
	agent, mlReader := newREPLLoopTestAgent(t, provider, pr, &out)
	agent.History = []api.Message{{Role: "user", Content: "previous"}}

	go func() {
		defer pw.Close()
		for _, line := range []string{"\n", "/clear\n", "hello\n"} {
			_, _ = io.WriteString(pw, line)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	runREPLLoop(agent, mlReader)

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if !strings.Contains(out.String(), "History cleared") {
		t.Fatalf("runREPLLoop() output = %q, want clear confirmation", out.String())
	}
}

func TestRunREPLLoop_ImageInputBranches(t *testing.T) {
	disableColors(t)

	t.Run("valid image triggers image chat", func(t *testing.T) {
		dir := t.TempDir()
		imagePath := filepath.Join(dir, "test.png")
		if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		var out bytes.Buffer
		provider := &imageCapableChatProvider{name: "openai"}
		agent, mlReader := newREPLLoopTestAgent(t, provider, strings.NewReader("image:"+imagePath+"\n"), &out)

		runREPLLoop(agent, mlReader)

		if provider.imageCalls != 1 {
			t.Fatalf("provider.imageCalls = %d, want 1", provider.imageCalls)
		}
		if !strings.Contains(provider.lastMessage, "Please analyze this image.") {
			t.Fatalf("provider.lastMessage = %q, want default image prompt", provider.lastMessage)
		}
		if !strings.Contains(out.String(), "Image loaded:") {
			t.Fatalf("runREPLLoop() output = %q, want image load log", out.String())
		}
	})

	t.Run("invalid image falls back to text chat", func(t *testing.T) {
		var out bytes.Buffer
		provider := &scriptedChatProvider{name: "openai", functionCalling: true}
		agent, mlReader := newREPLLoopTestAgent(t, provider, strings.NewReader("review image:/missing.png\n"), &out)

		runREPLLoop(agent, mlReader)

		if provider.callCount != 1 {
			t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
		}
		if !strings.Contains(out.String(), "Failed to load image") {
			t.Fatalf("runREPLLoop() output = %q, want image load failure", out.String())
		}
	})
}
