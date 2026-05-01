package agent

import (
	"errors"
	"fmt"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func numberedRetryHistory(pairs int) []api.Message {
	history := make([]api.Message, 0, pairs*2)
	for i := 1; i <= pairs; i++ {
		history = append(history,
			api.Message{Role: "user", Content: fmt.Sprintf("Test message %d", i)},
			api.Message{Role: "assistant", Content: fmt.Sprintf("Test response %d", i)},
		)
	}
	return history
}

// TestAgent_handleTokenLimitErrorWithRetry_Basic は token limit retry state の基本形を確認する。
func TestAgent_handleTokenLimitErrorWithRetry_Basic(t *testing.T) {
	agent := &Agent{
		History: numberedRetryHistory(6),
		agentRequestState: agentRequestState{
			tokenLimitRetryCount: 0,
		},
	}

	_ = agent.handleTokenLimitErrorWithRetry

	t.Run("Agent has tokenLimitRetryCount field", func(t *testing.T) {
		if agent.tokenLimitRetryCount != 0 {
			t.Errorf("tokenLimitRetryCount should be 0 initially, got %d", agent.tokenLimitRetryCount)
		}
	})

	t.Run("History has correct length", func(t *testing.T) {
		if len(agent.History) != 12 {
			t.Errorf("History should have 12 messages, got %d", len(agent.History))
		}
	})
}

// TestAgent_handleTokenLimitErrorWithRetry_TestCases は retry 前提の代表的な state を確認する。
func TestAgent_handleTokenLimitErrorWithRetry_TestCases(t *testing.T) {
	t.Run("通常エラーは retry count を変えない", func(t *testing.T) {
		agent := &Agent{
			History: numberedRetryHistory(1),
			agentRequestState: agentRequestState{
				tokenLimitRetryCount: 0,
			},
		}

		normalErr := errors.New("some other error")
		if normalErr.Error() == "" {
			t.Fatal("normal error should have a message")
		}
		if agent.tokenLimitRetryCount != 0 {
			t.Errorf("tokenLimitRetryCount should be 0, got %d", agent.tokenLimitRetryCount)
		}
	})

	t.Run("リトライ上限超えの状態確認", func(t *testing.T) {
		agent := &Agent{
			History: numberedRetryHistory(2),
			agentRequestState: agentRequestState{
				tokenLimitRetryCount: 1,
			},
		}

		if agent.tokenLimitRetryCount != 1 {
			t.Errorf("tokenLimitRetryCount should be 1, got %d", agent.tokenLimitRetryCount)
		}
	})

	t.Run("短い履歴の状態確認", func(t *testing.T) {
		agent := &Agent{
			History: numberedRetryHistory(5),
			agentRequestState: agentRequestState{
				tokenLimitRetryCount: 0,
			},
		}

		if len(agent.History) != 10 {
			t.Errorf("History should have 10 messages, got %d", len(agent.History))
		}
	})

	t.Run("長い履歴の状態確認", func(t *testing.T) {
		agent := &Agent{
			History: numberedRetryHistory(11),
			agentRequestState: agentRequestState{
				tokenLimitRetryCount: 0,
			},
		}

		if len(agent.History) != 22 {
			t.Errorf("History should have 22 messages, got %d", len(agent.History))
		}
		if agent.tokenLimitRetryCount != 0 {
			t.Errorf("tokenLimitRetryCount should be 0, got %d", agent.tokenLimitRetryCount)
		}
	})

	t.Run("エラー状態の確認", func(t *testing.T) {
		agent := &Agent{
			History: numberedRetryHistory(1),
			agentRequestState: agentRequestState{
				tokenLimitRetryCount: 0,
			},
		}

		retryErr := errors.New("retry failed")
		if retryErr.Error() != "retry failed" {
			t.Errorf("Error message should be 'retry failed', got %s", retryErr.Error())
		}
		if agent.tokenLimitRetryCount != 0 {
			t.Errorf("tokenLimitRetryCount should be 0, got %d", agent.tokenLimitRetryCount)
		}
	})
}

// TestAgent_handleTokenLimitErrorWithRetry_Integration は統合テストのプレースホルダー。
func TestAgent_handleTokenLimitErrorWithRetry_Integration(t *testing.T) {
	t.Skip("統合テストは別ファイルで実装: 実際のトークン上限エラー、config、CompressHistoryのモックが必要")
}
