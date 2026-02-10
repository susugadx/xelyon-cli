package agent

import (
	"fmt"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// CostTracker は Plan Mode 実行時のトークン使用量を追跡する
type CostTracker struct {
	mu             sync.Mutex
	supervisorCost TokenCost
	workerCosts    []TokenCost
}

// TokenCost はモデルごとのトークン使用量
type TokenCost struct {
	Model        string
	InputTokens  int
	OutputTokens int
}

// NewCostTracker は新しい CostTracker を作成
func NewCostTracker() *CostTracker {
	return &CostTracker{}
}

// AddSupervisorCost は Supervisor のコストを加算
func (ct *CostTracker) AddSupervisorCost(model string, input, output int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.supervisorCost.Model = model
	ct.supervisorCost.InputTokens += input
	ct.supervisorCost.OutputTokens += output
}

// AddWorkerCost は Worker のコストを記録
func (ct *CostTracker) AddWorkerCost(model string, input, output int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.workerCosts = append(ct.workerCosts, TokenCost{
		Model:        model,
		InputTokens:  input,
		OutputTokens: output,
	})
}

// PrintSummary はコストサマリーを表示
func (ct *CostTracker) PrintSummary() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// トークン使用がなければ何も表示しない
	if ct.supervisorCost.InputTokens == 0 && ct.supervisorCost.OutputTokens == 0 && len(ct.workerCosts) == 0 {
		return
	}

	fmt.Printf("\n--- Plan execution cost summary ---\n")
	if ct.supervisorCost.Model != "" {
		fmt.Printf("  Supervisor (%s): %d input + %d output tokens\n",
			ct.supervisorCost.Model, ct.supervisorCost.InputTokens, ct.supervisorCost.OutputTokens)
	}

	totalInput, totalOutput := 0, 0
	workerModel := ""
	for _, wc := range ct.workerCosts {
		totalInput += wc.InputTokens
		totalOutput += wc.OutputTokens
		if workerModel == "" {
			workerModel = wc.Model
		}
	}
	if len(ct.workerCosts) > 0 {
		fmt.Printf("  Workers (%s) x%d calls: %d input + %d output tokens\n",
			workerModel, len(ct.workerCosts), totalInput, totalOutput)
	}
}

// SetupUsageCallbacks は Provider に UsageCallback を設定する
func (ct *CostTracker) SetupUsageCallbacks(supervisorProvider api.Provider, supervisorModel string, workerProvider api.Provider, workerModel string) {
	if reporter, ok := supervisorProvider.(api.UsageReporter); ok {
		reporter.SetUsageCallback(func(u api.Usage) {
			ct.AddSupervisorCost(supervisorModel, u.InputTokens, u.OutputTokens)
		})
	}
	// 同一プロバイダーの場合、Worker コストは Supervisor と同じ callback で収集される
	// （区別できないため、全て Supervisor コストとして記録）
	if workerProvider != supervisorProvider {
		if reporter, ok := workerProvider.(api.UsageReporter); ok {
			reporter.SetUsageCallback(func(u api.Usage) {
				ct.AddWorkerCost(workerModel, u.InputTokens, u.OutputTokens)
			})
		}
	}
}
