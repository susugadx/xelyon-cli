package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
	for i, msg := range messages {
		if msg.Message.Role != tui.ChatRoleAssistantChunk {
			t.Fatalf("messages[%d].Role = %q, want assistant chunk role", i, msg.Message.Role)
		}
	}
}

func TestTUIAdapter_ChatReturnsProviderError(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockErrorProvider{}, &out)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, nil)
	err := adapter.Chat("please fail")

	if err == nil || !strings.Contains(err.Error(), "mock error") {
		t.Fatalf("Chat() error = %v, want mock error", err)
	}
	if adapter.IsProcessing() {
		t.Fatal("Chat() should reset processing flag after error")
	}
}

func TestTUIAdapter_ChatWaitsForTUIEventFlushBeforeReturning(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	agent := newChatRequestTestAgent(t, provider, &out)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, nil)
	flushStarted := make(chan struct{})
	releaseFlush := make(chan struct{})
	returned := make(chan struct{})

	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseFlush) })
	adapter.setTUIEventFlush(func() {
		close(flushStarted)
		<-releaseFlush
	})

	go func() {
		adapter.Chat("hello from tui")
		close(returned)
	}()

	select {
	case <-flushStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("Chat() did not reach TUI event flush")
	}

	select {
	case <-returned:
		t.Fatal("Chat() returned before TUI event flush completed")
	case <-time.After(20 * time.Millisecond):
	}

	releaseOnce.Do(func() { close(releaseFlush) })

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("Chat() did not return after TUI event flush completed")
	}
}

func TestTUIAdapter_ChatWithImagePathReturnsLoadError(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &imageOnceProvider{}, &out)
	t.Cleanup(agent.Cleanup)

	adapter := NewTUIAdapter(agent, nil)
	err := adapter.ChatWithImagePath("describe", filepath.Join(t.TempDir(), "missing.png"))

	if err == nil || !strings.Contains(err.Error(), "failed to load image") {
		t.Fatalf("ChatWithImagePath() error = %v, want image load error", err)
	}
	if got := tui.AgentErrorKindFromError(err, tui.AgentErrorProvider); got != tui.AgentErrorValidation {
		t.Fatalf("ChatWithImagePath() error kind = %q, want %q", got, tui.AgentErrorValidation)
	}
	if adapter.IsProcessing() {
		t.Fatal("ChatWithImagePath() should reset processing flag after load error")
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
