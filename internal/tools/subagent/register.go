package subagent

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// SpawnAgentTool は spawn_agent ツールです。
type SpawnAgentTool struct {
	manager *Manager
}

// NewSpawnAgentTool は spawn_agent ツールを作成します。
func NewSpawnAgentTool(manager *Manager) *SpawnAgentTool {
	return &SpawnAgentTool{manager: manager}
}

func (t *SpawnAgentTool) Name() string { return "spawn_agent" }

func (t *SpawnAgentTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *SpawnAgentTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Task instruction for the sub-agent. Include file paths and specific requirements.",
			},
			"task_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"explore", "edit", "verify"},
				"description": "Task type: explore (read-only investigation, default), edit (file modifications), verify (run build/test/lint).",
			},
		},
		"required":             []string{"message"},
		"additionalProperties": false,
	}
}

func (t *SpawnAgentTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	message := args["message"]
	if message == "" {
		return "Error: message is required", nil, nil
	}
	if t.manager == nil {
		return "Error: sub-agent manager is not configured", nil, nil
	}

	provider := execCtx.EffectiveProvider()
	if provider == nil {
		return "Error: provider is not available", nil, nil
	}

	taskType := args["task_type"]
	if taskType == "" {
		taskType = "explore"
	}

	id, err := t.manager.Spawn(
		execCtx.EffectiveContext(),
		message,
		taskType,
		args["model"],
		args["reasoning_effort"],
		provider,
		execCtx.EffectiveConfig(),
	)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, nil
	}

	bytes, _ := json.Marshal(map[string]string{
		"agent_id": id,
		"status":   "running",
	})
	return string(bytes), nil, nil
}

// WaitAgentTool は wait_agent ツールです。
type WaitAgentTool struct {
	manager *Manager
}

// NewWaitAgentTool は wait_agent ツールを作成します。
func NewWaitAgentTool(manager *Manager) *WaitAgentTool {
	return &WaitAgentTool{manager: manager}
}

func (t *WaitAgentTool) Name() string { return "wait_agent" }

func (t *WaitAgentTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *WaitAgentTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ids": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Agent IDs to wait for (from spawn_agent).",
			},
		},
		"required":             []string{"ids"},
		"additionalProperties": false,
	}
}

func (t *WaitAgentTool) Run(_ tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	if t.manager == nil {
		return "Error: sub-agent manager is not configured", nil, nil
	}

	var ids []string
	if err := json.Unmarshal([]byte(args["ids"]), &ids); err != nil {
		return fmt.Sprintf("Error: invalid ids: %v", err), nil, nil
	}
	if len(ids) == 0 {
		return "Error: ids is empty", nil, nil
	}

	timeoutMs := 0
	if raw := args["timeout_ms"]; raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Sprintf("Error: invalid timeout_ms: %v", err), nil, nil
		}
		if value > 0 && value < 60000 {
			value = 0
		}
		timeoutMs = value
	}

	response := t.manager.Wait(ids, timeoutMs)
	bytes, _ := json.Marshal(response)
	return string(bytes), nil, nil
}
