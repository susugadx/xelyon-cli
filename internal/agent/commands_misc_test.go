package agent

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestIsCodexModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"gpt-5.2-codex", true},
		{"gpt-5.1-codex", true},
		{"gpt-5.1-codex-max", true},
		{"gpt-5-codex", true},
		{"GPT-5.2-CODEX", true}, // case insensitive
		{"gpt-5.2-Codex", true}, // mixed case
		{"gpt-4o", false},
		{"gpt-5.2", false},
		{"claude-3-opus", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := isCodexModel(tt.model)
			if got != tt.want {
				t.Errorf("isCodexModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "single code block with language",
			text: "```go\nfunc main() {}\n```",
			want: []string{"func main() {}"},
		},
		{
			name: "single code block without language",
			text: "```\ncode here\n```",
			want: []string{"code here"},
		},
		{
			name: "multiple code blocks",
			text: "```go\nfunc foo() {}\n```\n\nSome text\n\n```python\ndef bar():\n    pass\n```",
			want: []string{"func foo() {}", "def bar():\n    pass"},
		},
		{
			name: "no code blocks",
			text: "Just some regular text without any code blocks",
			want: []string{},
		},
		{
			name: "empty string",
			text: "",
			want: []string{},
		},
		{
			name: "code block with extra whitespace",
			text: "```js\n  const x = 1;  \n```",
			want: []string{"const x = 1;"},
		},
		{
			name: "nested backticks in text",
			text: "Use `inline code` like this\n```go\nfunc test() {}\n```",
			want: []string{"func test() {}"},
		},
		{
			name: "code block with multiline content",
			text: "```typescript\ninterface User {\n  name: string;\n  age: number;\n}\n```",
			want: []string{"interface User {\n  name: string;\n  age: number;\n}"},
		},
		{
			name: "unclosed code block",
			text: "```go\nfunc incomplete()",
			want: []string{},
		},
		{
			name: "code block with different languages",
			text: "```rust\nfn main() {}\n```\n```cpp\nint main() {}\n```",
			want: []string{"fn main() {}", "int main() {}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeBlocks(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("extractCodeBlocks() returned %d blocks, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractCodeBlocks()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRequestCacheMode(t *testing.T) {
	tests := []struct {
		name  string
		usage api.Usage
		want  string
	}{
		{name: "none", usage: api.Usage{}, want: "none"},
		{name: "read", usage: api.Usage{InputTokens: 100, CachedInputTokens: 40}, want: "read"},
		{name: "create", usage: api.Usage{InputTokens: 100, CacheCreationTokens: 40}, want: "create"},
		{name: "read and create", usage: api.Usage{InputTokens: 100, CachedInputTokens: 20, CacheCreationTokens: 30}, want: "read + create"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requestCacheMode(tt.usage); got != tt.want {
				t.Fatalf("requestCacheMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildLastRequestTable_UsesCostOverride(t *testing.T) {
	usage := &api.Usage{
		InputTokens:       1000,
		CachedInputTokens: 600,
		OutputTokens:      200,
	}
	costEstimate := cost.CostEstimate{Cost: 0.1234}

	table := buildLastRequestTable(nil, "openai", "gpt-5.4", usage, &costEstimate)
	if table == nil {
		t.Fatal("buildLastRequestTable() = nil, want table")
	}

	output := table.RenderCompact()
	for _, want := range []string{"Cache Mode", "read", "Cached", "Hit Rate", "$0.1234 USD"} {
		if !strings.Contains(output, want) {
			t.Fatalf("buildLastRequestTable() output missing %q:\n%s", want, output)
		}
	}
}

func TestBuildLastRequestTable_ShowsPricingUnavailable(t *testing.T) {
	usage := &api.Usage{InputTokens: 1000, OutputTokens: 200}
	table := buildLastRequestTable(nil, "bedrock", "amazon.nova-pro-v1:0", usage, nil)
	if table == nil {
		t.Fatal("buildLastRequestTable() = nil, want table")
	}

	output := table.RenderCompact()
	if !strings.Contains(output, "N/A (pricing unavailable)") {
		t.Fatalf("buildLastRequestTable() output should show unavailable pricing:\n%s", output)
	}
}

func TestBuildLastRequestTable_UsesCatalogModelForPricing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
	})
	usage := &api.Usage{InputTokens: 1000, OutputTokens: 200}

	table := buildLastRequestTable(cfg, "openai", "corp-gpt-deployment", usage, nil)
	if table == nil {
		t.Fatal("buildLastRequestTable() = nil, want table")
	}

	output := table.RenderCompact()
	if strings.Contains(output, "N/A (pricing unavailable)") {
		t.Fatalf("buildLastRequestTable() ignored catalog_model pricing:\n%s", output)
	}
	if !strings.Contains(output, "$0.0055 USD") {
		t.Fatalf("buildLastRequestTable() output missing catalog_model cost:\n%s", output)
	}
}

func TestLastRequestUsageForStatus_PrefersLastTurnUsage(t *testing.T) {
	stats := NewSessionStats("openai", "gpt-5.4")
	stats.LastUsage = &api.Usage{InputTokens: 100, OutputTokens: 10}
	stats.LastTurnUsage = &api.Usage{
		InputTokens:       200,
		CachedInputTokens: 120,
		OutputTokens:      20,
	}
	stats.LastTurnCost = 0.0456

	usage, cost := lastRequestUsageForStatus(stats)
	if usage != stats.LastTurnUsage {
		t.Fatal("expected last turn usage to be returned")
	}
	if cost == nil || cost.Cost != 0.0456 {
		t.Fatalf("cost override = %v, want 0.0456", cost)
	}
}

func TestRequestCacheHitRate(t *testing.T) {
	usage := api.Usage{InputTokens: 200, CachedInputTokens: 50}
	if got := requestCacheHitRate(usage); got != 25.0 {
		t.Fatalf("requestCacheHitRate() = %.1f, want 25.0", got)
	}
}

func TestPrintSessionSections_Optimizations(t *testing.T) {
	stats := NewSessionStats("test")
	stats.ToolExecutions["read_file"] = 2
	stats.Optimizations = OptimizationMetrics{
		NegativeCacheHits:      3,
		ErrorCompressions:      4,
		FailedPairCompressions: 5,
		OutlineFirstCount:      7,
		CompactionCount:        10,
		CostAwareCompressions:  4,
	}

	var out bytes.Buffer
	agent := &Agent{
		Stats: stats,
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	printSessionSections(agent)
	output := out.String()

	for _, want := range []string{
		"⚡ Optimizations",
		"Negative cache",
		"Error compression",
		"Failed-pair compression",
		"Outline-first mode",
		"Auto-compress",
		"Cost-aware auto-compress",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("printSessionSections() output missing %q:\n%s", want, output)
		}
	}
}

func TestPrintSessionSections_NoOptimizations(t *testing.T) {
	stats := NewSessionStats("test")
	stats.ToolExecutions = map[string]int{}

	var out bytes.Buffer
	agent := &Agent{
		Stats: stats,
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	printSessionSections(agent)
	output := out.String()

	if !strings.Contains(output, "No optimizations triggered yet") {
		t.Fatalf("printSessionSections() output missing empty optimization message:\n%s", output)
	}
	if strings.Contains(output, "🤖 Sub-agents") {
		t.Fatalf("printSessionSections() should not show sub-agent section without spawned agents:\n%s", output)
	}
}

func TestBuildSessionTokenTable_WithSubAgentCosts(t *testing.T) {
	stats := NewSessionStats("openai", "gpt-5.4")
	stats.InputTokens = 1200
	stats.CachedInputTokens = 300
	stats.OutputTokens = 400
	stats.ThinkingTokens = 50
	stats.AccumulatedCost = 0.0900

	agent := &Agent{
		CurrentModel: "gpt-5.4",
		ProviderName: "openai",
	}
	summary := &subagent.SubAgentSummary{
		TotalSpawned:  2,
		TotalInput:    41855,
		TotalCached:   31972,
		TotalOutput:   2144,
		TotalThinking: 17,
		TotalCost:     0.0052,
	}

	table := buildSessionTokenTable(agent, stats, summary)
	if table == nil {
		t.Fatal("buildSessionTokenTable() = nil, want table")
	}

	output := table.RenderCompact()
	for _, want := range []string{
		"Parent Context",
		"Parent Input",
		"Parent Cached",
		"Parent Hit Rate",
		"Parent Output",
		"Parent Thinking",
		"Parent Total",
		"Sub-agent Input",
		"Sub-agent Cached",
		"Sub-agent Hit Rate",
		"Sub-agent Output",
		"Sub-agent Thinking",
		"Sub-agent Total",
		"Parent Cost",
		"Sub-agent Cost",
		"Total Cost",
		"$0.0900 USD",
		"$0.0052 USD",
		"$0.0952 USD",
		"41,855 tokens",
		"31,972 tokens",
		"2,144 tokens",
		"17 tokens",
		"44,016 tokens",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("buildSessionTokenTable() output missing %q:\n%s", want, output)
		}
	}
}

func TestBuildSessionTokenTable_UnknownParentCost(t *testing.T) {
	stats := NewSessionStats("bedrock", "amazon.nova-pro-v1:0")
	stats.AddTokens(1000, 200)

	agent := &Agent{
		CurrentModel: "amazon.nova-pro-v1:0",
		ProviderName: "bedrock",
	}

	table := buildSessionTokenTable(agent, stats, nil)
	if table == nil {
		t.Fatal("buildSessionTokenTable() = nil, want table")
	}
	output := table.RenderCompact()
	if !strings.Contains(output, "N/A (pricing unavailable)") {
		t.Fatalf("buildSessionTokenTable() output should show unavailable pricing:\n%s", output)
	}
}

func TestPrintSubAgentStats_ErrorShowsMessage(t *testing.T) {
	summary := subagent.SubAgentSummary{
		Agents: []subagent.SubAgentStats{
			{
				ID:             "sub-001",
				Model:          "gpt-5.4-nano",
				Status:         "completed",
				InputTokens:    20555,
				CachedTokens:   15872,
				OutputTokens:   1164,
				ThinkingTokens: 0,
				Cost:           0.0027,
				ToolExecutions: 1,
			},
			{
				ID:           "sub-002",
				Model:        "gpt-5.4-nano",
				Status:       "error",
				ErrorMessage: "provider timeout while fetching result\nwith extra details",
			},
		},
		TotalSpawned:   2,
		TotalCompleted: 1,
		TotalErrors:    1,
		TotalInput:     20555,
		TotalCached:    15872,
		TotalOutput:    1164,
		TotalCost:      0.0027,
		TotalTools:     1,
	}

	var out bytes.Buffer
	printSubAgentStats(&out, summary)
	output := out.String()

	if !strings.Contains(output, "🤖 Sub-agents") {
		t.Fatalf("printSubAgentStats() output missing header:\n%s", output)
	}
	if !strings.Contains(output, "sub-001") || !strings.Contains(output, "sub-002") {
		t.Fatalf("printSubAgentStats() output missing agent ids:\n%s", output)
	}
	if !strings.Contains(output, "$0.0027") {
		t.Fatalf("printSubAgentStats() output missing cost:\n%s", output)
	}
	errorRow := regexp.MustCompile(`error\s*│\s*0\s*│\s*0\s*│\s*0\s*│\s*0\s*│\s*\$0\.0000\s*│\s*0\s*│\s*provider timeout while fetching result`)
	if !errorRow.MatchString(output) {
		t.Fatalf("printSubAgentStats() error row should show message:\n%s", output)
	}
	if !strings.Contains(output, "Error") {
		t.Fatalf("printSubAgentStats() output missing error column:\n%s", output)
	}
}

func TestPrintSubAgentStats_RunningUsesDash(t *testing.T) {
	summary := subagent.SubAgentSummary{
		Agents: []subagent.SubAgentStats{
			{
				ID:     "sub-001",
				Model:  "gpt-5.4-nano",
				Status: "running",
			},
		},
		TotalSpawned: 1,
		TotalRunning: 1,
	}

	var out bytes.Buffer
	printSubAgentStats(&out, summary)
	output := out.String()

	runningRow := regexp.MustCompile(`running\s*│\s*-\s*│\s*-\s*│\s*-\s*│\s*-\s*│\s*-\s*│\s*-\s*│`)
	if !runningRow.MatchString(output) {
		t.Fatalf("printSubAgentStats() running row should use '-':\n%s", output)
	}
}

func TestHandleTokensCommand_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		CurrentModel: "gpt-5.2",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Token Usage / トークン使用量") {
		t.Fatalf("expected runtime output to contain token header, got %q", output)
	}
	if !strings.Contains(output, "Current:") {
		t.Fatalf("expected runtime output to contain current token line, got %q", output)
	}
}

func TestGetSessionFileInfo_UsesHistoryJSONLPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.AddMessage("user", "hello", session.Model)
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	agent := &Agent{
		agentConversationState: agentConversationState{
			session: session,
			storage: storage,
		},
	}

	gotPath, gotSize := getSessionFileInfo(agent)
	wantPath := fmt.Sprintf("~/.xelyon/history/%s.jsonl", session.ID)
	if gotPath != wantPath {
		t.Fatalf("getSessionFileInfo() path = %q, want %q", gotPath, wantPath)
	}
	if gotSize <= 0 {
		t.Fatalf("getSessionFileInfo() size = %d, want > 0", gotSize)
	}
}
