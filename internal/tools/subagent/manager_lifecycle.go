package subagent

import (
	"context"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (m *Manager) runAgent(ctx context.Context, sub *managedSubAgent, message, model string, provider api.Provider, cfg *config.Config) {
	runCtx := WithEventChannel(ctx, m.eventCh)
	runCtx = WithAgentID(runCtx, sub.id)

	defer close(sub.done)
	defer func() {
		status, result := m.agentOutcome(sub)
		EmitCompletionEvent(runCtx, status, result)
	}()
	defer func() {
		if recovered := recover(); recovered != nil {
			m.markAgentPanic(sub, recovered)
		}
	}()

	result := normalizeAgentResult(m.runHeadless(runCtx, message, model, provider, cfg))
	m.setAgentResult(sub, result)
}

func normalizeAgentResult(result *RunResult) *RunResult {
	if result == nil {
		return &RunResult{
			Status:       "error",
			ErrorMessage: "sub-agent runner returned nil result",
		}
	}
	if result.Status == "running" {
		return &RunResult{
			Status:       "error",
			ErrorMessage: "sub-agent runner returned invalid running status",
		}
	}
	return result
}

func (m *Manager) allocateRunningAgent(model, taskType string, cfg *config.Config) (*managedSubAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	limit := maxConcurrent(cfg)
	if m.runningCountLocked() >= limit {
		return nil, fmt.Errorf("max concurrent sub-agents reached (%d)", limit)
	}

	id := fmt.Sprintf("sub-%03d", m.counter.Add(1))
	sub := &managedSubAgent{
		id:        id,
		model:     model,
		taskType:  taskType,
		status:    "running",
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
	m.agents[id] = sub
	return sub, nil
}

func (m *Manager) removeAgent(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, id)
}

func (m *Manager) markAgentPanic(sub *managedSubAgent, recovered interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub.status = "error"
	sub.result = &RunResult{
		Status:       "error",
		ErrorMessage: fmt.Sprintf("panic: %v", recovered),
	}
}

func (m *Manager) setAgentResult(sub *managedSubAgent, result *RunResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub.result = result
	if result.Status == "completed" {
		sub.status = "completed"
		return
	}
	sub.status = "error"
}

func (m *Manager) agentOutcome(sub *managedSubAgent) (string, *RunResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return sub.status, sub.result
}

func (m *Manager) getAgent(id string) (*managedSubAgent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.agents[id]
	return sub, ok
}

func (m *Manager) runningCountLocked() int {
	count := 0
	for _, sub := range m.agents {
		if sub.status == "running" {
			count++
		}
	}
	return count
}

func maxConcurrent(cfg *config.Config) int {
	if cfg == nil || cfg.SubAgent.MaxConcurrent <= 0 {
		return defaultSubAgentMaxConcurrent
	}
	return cfg.SubAgent.MaxConcurrent
}
