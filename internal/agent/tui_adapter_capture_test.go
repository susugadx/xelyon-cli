package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tui"
)

func TestTUIAdapter_ChatFlushesCapturedOutput(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	var messages []tui.AppendMessageMsg
	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	agent := newChatRequestTestAgent(t, provider, &out)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, func(msg tui.AppendMessageMsg) {
		messages = append(messages, msg)
	})
	adapter.SetOutputCapture()
	adapter.Chat("hello from tui")

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if adapter.IsProcessing() {
		t.Fatal("Chat() should reset processing flag")
	}
	if len(messages) == 0 {
		t.Fatal("Chat() should relay captured output to TUI")
	}
}

func TestTUIAdapter_ChatWithImageUsesImageProvider(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	imagePath := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(imagePath, []byte("fake png data for testing"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	var messages []tui.AppendMessageMsg
	provider := &imageOnceProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, func(msg tui.AppendMessageMsg) {
		messages = append(messages, msg)
	})
	adapter.SetOutputCapture()
	adapter.Chat("describe image:" + imagePath)

	if provider.imageCalls != 1 {
		t.Fatalf("provider.imageCalls = %d, want 1", provider.imageCalls)
	}
	if adapter.IsProcessing() {
		t.Fatal("Chat() should reset processing flag after image request")
	}
	if len(messages) == 0 {
		t.Fatal("Chat() with image should relay captured output to TUI")
	}
}

func TestTUICaptureWriter_Flush(t *testing.T) {
	var batches []string
	writer := newTUICaptureWriter(func(text string) {
		batches = append(batches, text)
	})

	if _, err := writer.Write([]byte("partial output")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	writer.Flush()

	if len(batches) != 1 || strings.TrimSpace(batches[0]) != "partial output" {
		t.Fatalf("Flush() batches = %v, want [partial output]", batches)
	}
}
