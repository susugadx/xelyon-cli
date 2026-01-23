package history

import (
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	model := "test-model"
	session := NewSession(model)

	if session == nil {
		t.Fatal("NewSession returned nil")
	}

	if session.Model != model {
		t.Errorf("Expected model %s, got %s", model, session.Model)
	}

	if session.ID == "" {
		t.Error("Expected non-empty session ID")
	}

	if len(session.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(session.Messages))
	}

	if session.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}

	if session.LastModified.IsZero() {
		t.Error("Expected non-zero last modified time")
	}
}

func TestSession_AddMessage(t *testing.T) {
	session := NewSession("test-model")
	initialModified := session.LastModified

	time.Sleep(10 * time.Millisecond) // Ensure time difference

	session.AddMessage("user", "Hello", "test-model")

	if len(session.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(session.Messages))
	}

	msg := session.Messages[0]
	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", msg.Role)
	}

	if msg.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", msg.Content)
	}

	if msg.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", msg.Model)
	}

	if msg.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if !session.LastModified.After(initialModified) {
		t.Error("Expected LastModified to be updated")
	}
}

func TestSession_AddMultipleMessages(t *testing.T) {
	session := NewSession("test-model")

	session.AddMessage("user", "Message 1", "test-model")
	session.AddMessage("assistant", "Response 1", "test-model")
	session.AddMessage("user", "Message 2", "test-model")

	if len(session.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(session.Messages))
	}

	// Check order
	if session.Messages[0].Content != "Message 1" {
		t.Error("First message should be 'Message 1'")
	}
	if session.Messages[1].Content != "Response 1" {
		t.Error("Second message should be 'Response 1'")
	}
	if session.Messages[2].Content != "Message 2" {
		t.Error("Third message should be 'Message 2'")
	}
}

func TestSession_ToAPIMessages(t *testing.T) {
	session := NewSession("test-model")

	session.AddMessage("user", "Hello", "test-model")
	session.AddMessage("assistant", "Hi there", "test-model")

	apiMsgs := session.ToAPIMessages()

	if len(apiMsgs) != 2 {
		t.Fatalf("Expected 2 API messages, got %d", len(apiMsgs))
	}

	// Check first message
	if apiMsgs[0].Role != "user" {
		t.Errorf("Expected first role 'user', got '%s'", apiMsgs[0].Role)
	}
	if apiMsgs[0].Content != "Hello" {
		t.Errorf("Expected first content 'Hello', got '%s'", apiMsgs[0].Content)
	}

	// Check second message
	if apiMsgs[1].Role != "assistant" {
		t.Errorf("Expected second role 'assistant', got '%s'", apiMsgs[1].Role)
	}
	if apiMsgs[1].Content != "Hi there" {
		t.Errorf("Expected second content 'Hi there', got '%s'", apiMsgs[1].Content)
	}

	// Verify type
	_ = apiMsgs
}

func TestSession_ToAPIMessages_Empty(t *testing.T) {
	session := NewSession("test-model")

	apiMsgs := session.ToAPIMessages()

	if len(apiMsgs) != 0 {
		t.Errorf("Expected empty API messages, got %d", len(apiMsgs))
	}
}
