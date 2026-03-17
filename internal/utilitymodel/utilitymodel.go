package utilitymodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// TaskWebSearchCompaction は web_search 結果の圧縮タスク。
const TaskWebSearchCompaction = "web_search_compaction"

var newProvider = api.NewProvider

var defaultTasks = []string{
	TaskWebSearchCompaction,
}

// EnabledForTask は指定タスクで utility model を使える設定か判定する。
func EnabledForTask(cfg *config.Config, task string) bool {
	_, _, ok := resolveTaskTarget(cfg, task)
	return ok
}

// Run は utility model に単発タスクを実行させる。
func Run(ctx context.Context, cfg *config.Config, task, systemPrompt, userPrompt string) (string, error) {
	providerName, model, ok := resolveTaskTarget(cfg, task)
	if !ok {
		return "", fmt.Errorf("utility model is not configured for task %q", task)
	}

	provider, err := newProvider(providerName)
	if err != nil {
		return "", err
	}

	runtimeCfg := buildRuntimeConfig(cfg)
	api.ApplyRuntimeConfig(provider, runtimeCfg)

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = tools.WithRegistry(ctx, tools.NewRegistry())
	ctx = tools.WithConfig(ctx, runtimeCfg)

	return provider.ChatWithTools(ctx, systemPrompt, []api.Message{
		{Role: "user", Content: userPrompt},
	}, model)
}

func resolveTaskTarget(cfg *config.Config, task string) (providerName, model string, ok bool) {
	if cfg == nil || !cfg.UtilityModel.Enabled {
		return "", "", false
	}
	if !isAllowedTask(cfg.UtilityModel.Tasks, task) {
		return "", "", false
	}

	providerName = normalizeProviderName(cfg.UtilityModel.Provider)
	if providerName == "" {
		return "", "", false
	}

	model = strings.TrimSpace(cfg.UtilityModel.Model)
	if model == "" {
		model = cfg.GetModelForProvider(providerName)
	}
	if model == "" {
		return "", "", false
	}

	return providerName, model, true
}

func isAllowedTask(configured []string, task string) bool {
	task = strings.ToLower(strings.TrimSpace(task))
	if task == "" {
		return false
	}

	tasks := normalizeTasks(configured)
	for _, configuredTask := range tasks {
		if configuredTask == task {
			return true
		}
	}
	return false
}

func normalizeTasks(tasks []string) []string {
	if len(tasks) == 0 {
		return append([]string(nil), defaultTasks...)
	}

	result := make([]string, 0, len(tasks))
	for _, task := range tasks {
		task = strings.ToLower(strings.TrimSpace(task))
		if task == "" || contains(result, task) {
			continue
		}
		result = append(result, task)
	}
	if len(result) == 0 {
		return append([]string(nil), defaultTasks...)
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeProviderName(providerName string) string {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "anthropic":
		return "claude"
	default:
		return strings.ToLower(strings.TrimSpace(providerName))
	}
}

func buildRuntimeConfig(cfg *config.Config) *config.Config {
	runtimeCfg := config.CloneConfig(cfg)
	runtimeCfg.Thinking.Enabled = false
	runtimeCfg.PromptCache.Enabled = false
	runtimeCfg.UtilityModel.Enabled = false
	return runtimeCfg
}
