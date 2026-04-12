package agent

import "testing"

func TestPromptManager_DebugString(t *testing.T) {
	if got := (&PromptManager{}).DebugString(); got != "<nil>" {
		t.Fatalf("DebugString() = %q, want %q", got, "<nil>")
	}

	manager := &PromptManager{agent: &Agent{
		CurrentModel: "gpt-5.4",
		agentProjectPromptState: agentProjectPromptState{
			projectMapDirty:       true,
			projectMapFileCount:   7,
			projectMapSymbolCount: 13,
		},
	}}
	want := "PromptManager(model=gpt-5.4, dirty=true, files=7, symbols=13)"
	if got := manager.DebugString(); got != want {
		t.Fatalf("DebugString() = %q, want %q", got, want)
	}
}
