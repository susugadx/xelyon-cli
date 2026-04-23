package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type subAgentSpawner interface {
	Spawn(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config) (string, error)
}

type subAgentWaiter interface {
	Wait(ids []string, timeoutMs int) WaitResponse
}

// SpawnAgentTool は spawn_agent ツールです。
type SpawnAgentTool struct {
	manager subAgentSpawner
}

// NewSpawnAgentTool は spawn_agent ツールを作成します。
func NewSpawnAgentTool(manager subAgentSpawner) *SpawnAgentTool {
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

	return runningAgentResponse(id), nil, nil
}

// WaitAgentTool は wait_agent ツールです。
type WaitAgentTool struct {
	manager subAgentWaiter
}

// NewWaitAgentTool は wait_agent ツールを作成します。
func NewWaitAgentTool(manager subAgentWaiter) *WaitAgentTool {
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

	ids, err := parseWaitAgentIDs(args["ids"])
	if err != nil {
		return fmt.Sprintf("Error: invalid ids: %v", err), nil, nil
	}
	if len(ids) == 0 {
		return "Error: ids is empty", nil, nil
	}

	timeoutMs, err := parseWaitTimeoutMs(args["timeout_ms"])
	if err != nil {
		return fmt.Sprintf("Error: invalid timeout_ms: %v", err), nil, nil
	}

	response := t.manager.Wait(ids, timeoutMs)
	return waitResponseJSON(response), nil, nil
}

func parseWaitAgentIDs(raw string) ([]string, error) {
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func parseWaitTimeoutMs(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value > 0 && value < 60000 {
		return 0, nil
	}
	return value, nil
}

func runningAgentResponse(agentID string) string {
	bytes, _ := json.Marshal(map[string]string{
		"agent_id": agentID,
		"status":   "running",
	})
	return string(bytes)
}

func waitResponseJSON(response WaitResponse) string {
	bytes, _ := json.Marshal(response)
	return string(bytes)
}
