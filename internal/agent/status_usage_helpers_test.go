package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestLastUsageForStatusSeparatesChatTurnAndReview(t *testing.T) {
	stats := NewSessionStats("openai", "gpt-5.4")
	stats.LastUsage = &api.Usage{InputTokens: 100, OutputTokens: 10}
	stats.LastTurnUsage = &api.Usage{
		InputTokens:       200,
		CachedInputTokens: 120,
		OutputTokens:      20,
	}
	stats.LastTurnCost = 0.0456
	reviewUsage := api.Usage{InputTokens: 300, OutputTokens: 30}
	stats.LastReviewUsage = &reviewUsage
	stats.LastReviewCost = 0.0789

	chatUsage, chatCost := lastChatTurnUsageForStatus(stats)
	if chatUsage != stats.LastTurnUsage {
		t.Fatal("lastChatTurnUsageForStatus() should return the chat turn usage")
	}
	if chatCost == nil || chatCost.Cost != 0.0456 {
		t.Fatalf("chat cost override = %v, want 0.0456", chatCost)
	}

	gotReviewUsage, reviewCost := lastReviewUsageForStatus(stats)
	if gotReviewUsage != stats.LastReviewUsage {
		t.Fatal("lastReviewUsageForStatus() should return the review usage")
	}
	if reviewCost == nil || reviewCost.Cost != 0.0789 {
		t.Fatalf("review cost override = %v, want 0.0789", reviewCost)
	}
}

func TestLastUsageForStatusIgnoresLegacyLastUsage(t *testing.T) {
	stats := NewSessionStats("openai", "gpt-5.4")
	stats.LastUsage = &api.Usage{InputTokens: 100, OutputTokens: 10}

	chatUsage, chatCost := lastChatTurnUsageForStatus(stats)
	if chatUsage != nil || chatCost != nil {
		t.Fatalf("chat usage = %v cost = %v, want nil when no chat turn usage exists", chatUsage, chatCost)
	}

	reviewUsage, reviewCost := lastReviewUsageForStatus(stats)
	if reviewUsage != nil || reviewCost != nil {
		t.Fatalf("review usage = %v cost = %v, want nil when no review usage exists", reviewUsage, reviewCost)
	}
}
