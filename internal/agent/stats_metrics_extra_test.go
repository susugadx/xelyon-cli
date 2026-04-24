package agent

import "testing"

// --- FormatTokens tests ---

func TestFormatTokens_AllRanges(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1500, "1.5k"},
		{1500000, "1.5M"},
		{100, "100"},
		{10000, "10.0k"},
	}

	for _, tt := range tests {
		got := FormatTokens(tt.input)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- FormatNumber tests ---

func TestFormatNumber_AllRanges(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1234, "1,234"},
		{10000, "10,000"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		got := FormatNumber(tt.input)
		if got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- FormatFileSize tests ---

func TestFormatFileSize_AllRanges(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tt := range tests {
		got := FormatFileSize(tt.input)
		if got != tt.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- sessionCacheHitRate tests ---

func TestSessionCacheHitRate_Various(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		cached   int
		wantRate float64
	}{
		{name: "zero input", input: 0, cached: 0, wantRate: 0},
		{name: "negative input", input: -1, cached: 0, wantRate: 0},
		{name: "50% hit", input: 200, cached: 100, wantRate: 50.0},
		{name: "100% hit", input: 100, cached: 100, wantRate: 100.0},
		{name: "0% hit", input: 100, cached: 0, wantRate: 0},
		{name: "25% hit", input: 200, cached: 50, wantRate: 25.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &SessionStats{
				InputTokens:       tt.input,
				CachedInputTokens: tt.cached,
			}
			got := sessionCacheHitRate(stats)
			if got != tt.wantRate {
				t.Errorf("sessionCacheHitRate() = %f, want %f", got, tt.wantRate)
			}
		})
	}
}

// --- NewSessionStats tests ---

func TestNewSessionStats_WithModel(t *testing.T) {
	stats := NewSessionStats("claude", "claude-opus-4-6")
	if stats.Provider != "claude" {
		t.Errorf("Provider = %q, want %q", stats.Provider, "claude")
	}
	if stats.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", stats.Model, "claude-opus-4-6")
	}
	if stats.ToolExecutions == nil {
		t.Error("ToolExecutions should be initialized")
	}
	if stats.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestNewSessionStats_WithoutModel(t *testing.T) {
	stats := NewSessionStats("openai")
	if stats.Model != "" {
		t.Errorf("Model = %q, want empty", stats.Model)
	}
}

// --- SavingsMetrics tests ---

func TestSavingsMetrics_AddAndHasAny(t *testing.T) {
	m := SavingsMetrics{}
	if m.hasAny() {
		t.Error("empty SavingsMetrics.hasAny() should be false")
	}

	m.add(SavingsMetrics{SavedCalls: 1})
	if !m.hasAny() {
		t.Error("SavingsMetrics with SavedCalls=1 should hasAny()")
	}
	if m.SavedCalls != 1 {
		t.Errorf("SavedCalls = %d, want 1", m.SavedCalls)
	}

	m.add(SavingsMetrics{EstimatedInputTokensSaved: 500, EstimatedCostSaved: 0.05})
	if m.EstimatedInputTokensSaved != 500 {
		t.Errorf("EstimatedInputTokensSaved = %d, want 500", m.EstimatedInputTokensSaved)
	}
	if m.EstimatedCostSaved < 0.049 || m.EstimatedCostSaved > 0.051 {
		t.Errorf("EstimatedCostSaved = %f, want ~0.05", m.EstimatedCostSaved)
	}
}

func TestSavingsMetrics_HasAny_EachField(t *testing.T) {
	tests := []struct {
		name    string
		metrics SavingsMetrics
		want    bool
	}{
		{name: "empty", metrics: SavingsMetrics{}, want: false},
		{name: "saved calls only", metrics: SavingsMetrics{SavedCalls: 1}, want: true},
		{name: "tokens saved only", metrics: SavingsMetrics{EstimatedInputTokensSaved: 1}, want: true},
		{name: "cost saved only", metrics: SavingsMetrics{EstimatedCostSaved: 0.01}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metrics.hasAny(); got != tt.want {
				t.Errorf("hasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- OptimizationMetrics tests ---

func TestOptimizationMetrics_AddAndHasAny(t *testing.T) {
	m := OptimizationMetrics{}
	if m.hasAny() {
		t.Error("empty OptimizationMetrics.hasAny() should be false")
	}

	m.add(OptimizationMetrics{NegativeCacheHits: 3, CompactionCount: 2})
	if !m.hasAny() {
		t.Error("OptimizationMetrics with values should hasAny()")
	}
	if m.NegativeCacheHits != 3 {
		t.Errorf("NegativeCacheHits = %d, want 3", m.NegativeCacheHits)
	}
	if m.CompactionCount != 2 {
		t.Errorf("CompactionCount = %d, want 2", m.CompactionCount)
	}
}

func TestOptimizationMetrics_HasAny_EachField(t *testing.T) {
	tests := []struct {
		name    string
		metrics OptimizationMetrics
		want    bool
	}{
		{name: "empty", metrics: OptimizationMetrics{}, want: false},
		{name: "NegativeCacheHits", metrics: OptimizationMetrics{NegativeCacheHits: 1}, want: true},
		{name: "ErrorCompressions", metrics: OptimizationMetrics{ErrorCompressions: 1}, want: true},
		{name: "FailedPairCompressions", metrics: OptimizationMetrics{FailedPairCompressions: 1}, want: true},
		{name: "OutlineFirstCount", metrics: OptimizationMetrics{OutlineFirstCount: 1}, want: true},
		{name: "CompactionCount", metrics: OptimizationMetrics{CompactionCount: 1}, want: true},
		{name: "CostAwareCompressions", metrics: OptimizationMetrics{CostAwareCompressions: 1}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metrics.hasAny(); got != tt.want {
				t.Errorf("hasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
