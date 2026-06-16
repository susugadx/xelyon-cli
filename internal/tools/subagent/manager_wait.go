package subagent

import "time"

// Wait は指定されたサブエージェントの完了を待ちます。
func (m *Manager) Wait(ids []string, timeoutMs int) WaitResponse {
	results := make([]WaitResult, len(ids))
	status := "completed"

	var deadline <-chan time.Time
	if timeoutMs > 0 {
		deadline = time.After(time.Duration(timeoutMs) * time.Millisecond)
	}

	timedOut := false
	for i, id := range ids {
		sub, ok := m.getAgent(id)
		if !ok {
			results[i] = WaitResult{AgentID: id, Status: "error", Output: "agent not found"}
			status = "error"
			continue
		}

		if timedOut {
			results[i] = m.snapshotOrTimeout(sub, true)
			if results[i].Status == "error" {
				status = "error"
			}
			continue
		}

		if deadline == nil {
			<-sub.done
			results[i] = m.snapshotOrTimeout(sub, false)
		} else {
			select {
			case <-sub.done:
				results[i] = m.snapshotOrTimeout(sub, false)
			case <-deadline:
				timedOut = true
				results[i] = m.snapshotOrTimeout(sub, true)
			}
		}

		switch results[i].Status {
		case "error":
			status = "error"
		case "timeout":
			if status != "error" {
				status = "timeout"
			}
		}
	}

	return WaitResponse{
		Results: results,
		Status:  status,
	}
}

func (m *Manager) snapshotOrTimeout(sub *managedSubAgent, timeout bool) WaitResult {
	if sub == nil {
		return WaitResult{Status: "error", Output: "agent not found"}
	}

	if timeout {
		select {
		case <-sub.done:
		default:
			return WaitResult{
				AgentID: sub.id,
				Status:  "timeout",
				Output:  "",
			}
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	output := ""
	if sub.result != nil {
		switch {
		case sub.result.Status == "error" && sub.result.ErrorMessage != "":
			output = sub.result.ErrorMessage
		case sub.result.Response != "":
			output = sub.result.Response
		case sub.result.ErrorMessage != "":
			output = sub.result.ErrorMessage
		}
	}

	return WaitResult{
		AgentID: sub.id,
		Status:  sub.status,
		Output:  output,
	}
}
