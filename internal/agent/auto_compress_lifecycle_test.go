package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestMaybeAutoCompressAfterTurnRecordsSuccessfulCompression(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.TokenThreshold = 1

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()
	state := newAutoCompressionTurnState()

	if !agent.maybeAutoCompressAfterTurn(state) {
		t.Fatal("maybeAutoCompressAfterTurn() = false, want true")
	}
	if !state.attemptedThisTurn() {
		t.Fatal("attemptedThisTurn() = false, want true")
	}
	if !state.compressedThisTurn() {
		t.Fatal("compressedThisTurn() = false, want true")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestMaybeAutoCompressAfterTurnRecordsFailedAttempt(t *testing.T) {
	provider := &compressionTestProvider{
		name:    "openai",
		chatErr: errors.New("summary failed"),
	}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.TokenThreshold = 1

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()
	state := newAutoCompressionTurnState()

	if agent.maybeAutoCompressAfterTurn(state) {
		t.Fatal("maybeAutoCompressAfterTurn() = true, want false on failed compression")
	}
	if !state.attemptedThisTurn() {
		t.Fatal("attemptedThisTurn() = false, want true")
	}
	if state.compressedThisTurn() {
		t.Fatal("compressedThisTurn() = true, want false")
	}
	if provider.chatCalls != 1 {
		t.Fatalf("ChatWithTools call count = %d, want 1", provider.chatCalls)
	}
}

func TestMaybeAutoCompressAfterTurnSkipsWhenAlreadyCompressed(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.TokenThreshold = 1

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()
	state := newAutoCompressionTurnState()
	state.recordAttempt(true)

	if agent.maybeAutoCompressAfterTurn(state) {
		t.Fatal("maybeAutoCompressAfterTurn() = true, want false when turn already compressed")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}

func TestMaybeAutoCompressAfterTurnSkipsWhenAlreadyAttemptedAndFailed(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.TokenThreshold = 1

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()
	state := newAutoCompressionTurnState()
	state.recordAttempt(false)

	if agent.maybeAutoCompressAfterTurn(state) {
		t.Fatal("maybeAutoCompressAfterTurn() = true, want false when turn already attempted compression")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}

func TestMaybeAutoCompressDuringTurnDoesNotRecordAttemptWithoutCompressibleHistory(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "compressed summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.KeepRecent = 1
	cfg.Compression.PreferCompactAPI = false
	cfg.Compression.TokenThreshold = 1

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = oversizedCompressionHistory()
	state := newAutoCompressionTurnState()

	result := agent.maybeAutoCompressDuringTurn(context.Background(), 0, state)
	if result.compressed {
		t.Fatal("maybeAutoCompressDuringTurn().compressed = true, want false without pre-turn history")
	}
	if result.attempted {
		t.Fatal("maybeAutoCompressDuringTurn().attempted = true, want false without pre-turn history")
	}
	if result.requestErr != nil {
		t.Fatalf("maybeAutoCompressDuringTurn().requestErr = %v, want nil without attempted compression", result.requestErr)
	}
	if state.attemptedThisTurn() {
		t.Fatal("attemptedThisTurn() = true, want false when summary generation did not start")
	}
	if provider.chatCalls != 0 {
		t.Fatalf("ChatWithTools call count = %d, want 0", provider.chatCalls)
	}
}
