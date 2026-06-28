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

type subAgentRuntimeContextSpawner interface {
	SpawnWithRuntimeContext(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config, runtimeCtx SpawnRuntimeContext) (string, error)
}

type subAgentWaiter interface {
	Wait(ids []string, timeoutMs int) WaitResponse
}

const (
	// SpawnAgentToolName は sub-agent 起動 tool 名です。
	SpawnAgentToolName = "spawn_agent"
	// WaitAgentToolName は sub-agent 待機 tool 名です。
	WaitAgentToolName = "wait_agent"
)

// SpawnAgentTool は spawn_agent ツールです。
type SpawnAgentTool struct {
	manager subAgentSpawner
}

// NewSpawnAgentTool は spawn_agent ツールを作成します。
func NewSpawnAgentTool(manager subAgentSpawner) *SpawnAgentTool {
	return &SpawnAgentTool{manager: manager}
}

func (t *SpawnAgentTool) Name() string { return SpawnAgentToolName }

func (t *SpawnAgentTool) Description() string {
	return tools.ToolDescription(t.Name())
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
			"model": map[string]interface{}{
				"type":        "string",
				"description": "Optional sub-agent model override. Omit or use sub_agent.default_model to use the configured/default sub-agent model.",
			},
			"reasoning_effort": map[string]interface{}{
				"type":        "string",
				"enum":        reasoningEffortSchemaEnum(),
				"description": "Optional per-agent reasoning effort override: off, low, medium, high, or xhigh.",
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

	requestCtx := execCtx.EffectiveContext()
	cfg := execCtx.EffectiveConfig()
	model := args["model"]
	reasoningEffort := args["reasoning_effort"]
	var (
		id  string
		err error
	)
	if spawner, ok := t.manager.(subAgentRuntimeContextSpawner); ok {
		id, err = spawner.SpawnWithRuntimeContext(
			requestCtx,
			message,
			taskType,
			model,
			reasoningEffort,
			provider,
			cfg,
			SpawnRuntimeContext{CurrentModel: execCtx.Model},
		)
	} else {
		id, err = t.manager.Spawn(
			requestCtx,
			message,
			taskType,
			model,
			reasoningEffort,
			provider,
			cfg,
		)
	}
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

func (t *WaitAgentTool) Name() string { return WaitAgentToolName }

func (t *WaitAgentTool) Description() string {
	return tools.ToolDescription(t.Name())
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
			"timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Optional wait timeout in milliseconds. Values below 60000 are ignored and wait without a timeout.",
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
