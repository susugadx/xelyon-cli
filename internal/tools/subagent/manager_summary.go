package subagent

import "sort"

// GetSummary は全サブエージェントの統計サマリーを返します。
func (m *Manager) GetSummary() SubAgentSummary {
	m.mu.Lock()
	defer m.mu.Unlock()

	summary := SubAgentSummary{
		Agents:       make([]SubAgentStats, 0, len(m.agents)),
		TotalSpawned: len(m.agents),
	}
	if len(m.agents) == 0 {
		return summary
	}

	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		sub := m.agents[id]
		stats := SubAgentStats{
			ID:       sub.id,
			Model:    sub.model,
			TaskType: sub.taskType,
			Status:   sub.status,
		}
		if sub.result != nil {
			if sub.result.Model != "" {
				stats.Model = sub.result.Model
			}
			stats.ErrorMessage = sub.result.ErrorMessage
			stats.InputTokens = sub.result.InputTokens
			stats.CachedTokens = sub.result.CachedTokens
			stats.OutputTokens = sub.result.OutputTokens
			stats.ThinkingTokens = sub.result.ThinkingTokens
			stats.Cost = sub.result.Cost
			stats.PricingUnavailable = sub.result.PricingUnavailable
			stats.ToolExecutions = sub.result.ToolExecutions
			stats.ToolBreakdown = sub.result.ToolBreakdown
			stats.DurationMs = sub.result.DurationMs

			summary.TotalInput += stats.InputTokens
			summary.TotalCached += stats.CachedTokens
			summary.TotalOutput += stats.OutputTokens
			summary.TotalThinking += stats.ThinkingTokens
			summary.TotalCost += stats.Cost
			if stats.PricingUnavailable {
				summary.PricingUnavailable = true
			}
			summary.TotalTools += stats.ToolExecutions
		}

		switch sub.status {
		case "completed":
			summary.TotalCompleted++
		case "error":
			summary.TotalErrors++
		case "running":
			summary.TotalRunning++
		}

		summary.Agents = append(summary.Agents, stats)
	}

	return summary
}
