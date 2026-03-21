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
