package history

import "testing"

func TestSession_ToAPIMessages_SkipsToolExecutionEntries(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.AddToolExecution("wait_agent", map[string]string{"ids": `["sub-001"]`}, "done", true, "test-model")
	session.AddMessage("assistant", "world", "test-model")

	msgs := session.ToAPIMessages()
	if len(msgs) != 2 {
		t.Fatalf("len(ToAPIMessages()) = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("unexpected roles: %#v", msgs)
	}
}

func TestSession_TruncateMessages(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.AddMessage("assistant", "world", "test-model")
	session.markPersisted()
	session.AddMessage("user", "next", "test-model")

	if !session.TruncateMessages(1) {
		t.Fatal("TruncateMessages() = false, want true")
	}
	if len(session.Messages) != 1 {
		t.Fatalf("len(session.Messages) = %d, want 1", len(session.Messages))
	}
	if session.persistedCount != 1 {
		t.Fatalf("persistedCount = %d, want 1", session.persistedCount)
	}
}

func TestSession_ResetConversation(t *testing.T) {
	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	session.PendingApprovedPlan = "Implementation Plan\n1. Step"
	session.PendingApprovedPlanChangedFiles = []string{"foo.go"}
	session.CompactedItems = []CompactedItem{{Type: "compacted", Data: "compressed"}}
	session.IsCompactedMode = true
	session.ResponseID = "resp_123"
	session.markPersisted()

	session.ResetConversation()

	if len(session.Messages) != 0 {
		t.Fatalf("len(session.Messages) = %d, want 0", len(session.Messages))
	}
	if session.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty", session.PendingApprovedPlan)
	}
	if session.PendingApprovedPlanHasChanges {
		t.Fatal("PendingApprovedPlanHasChanges = true, want false")
	}
	if len(session.PendingApprovedPlanChangedFiles) != 0 {
		t.Fatalf("PendingApprovedPlanChangedFiles = %v, want empty", session.PendingApprovedPlanChangedFiles)
	}
	if len(session.CompactedItems) != 0 {
		t.Fatalf("len(session.CompactedItems) = %d, want 0", len(session.CompactedItems))
	}
	if session.IsCompactedMode {
		t.Fatal("IsCompactedMode = true, want false")
	}
	if session.ResponseID != "" {
		t.Fatalf("ResponseID = %q, want empty", session.ResponseID)
	}
	if session.persistedCount != 0 {
		t.Fatalf("persistedCount = %d, want 0", session.persistedCount)
	}
}
