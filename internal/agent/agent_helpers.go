package agent

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// parseImageInputWithWriter は入力から画像パスを抽出する。
// 形式: "image:/path/to/file.png こんにちは" または "こんにちは image:/path/to/file.png"
func parseImageInputWithWriter(out io.Writer, input string) (text string, image *api.ImageData) {
	// image:プレフィックスを探す
	imagePrefix := "image:"

	// 正規表現的な簡易パース
	parts := strings.Fields(input)
	var textParts []string
	var imagePath string

	for _, part := range parts {
		if strings.HasPrefix(part, imagePrefix) {
			imagePath = strings.TrimPrefix(part, imagePrefix)
		} else {
			textParts = append(textParts, part)
		}
	}

	// 画像パスがない場合
	if imagePath == "" {
		return input, nil
	}

	// テキスト部分を結合
	text = strings.Join(textParts, " ")
	if text == "" {
		text = "Please analyze this image." // デフォルトメッセージ
	}

	// 画像読み込み
	img, err := api.LoadImage(imagePath)
	if err != nil {
		red.Fprintf(out, "Failed to load image: %v\n", err)
		return input, nil
	}

	green.Fprintf(out, "🖼️  Image loaded: %s (%s)\n", img.Path, api.FormatImageSize(img.Size))
	return text, img
}

// ANSI color codes for gradient (blue -> cyan)
const (
	colorBlue1 = "\033[38;5;27m" // Deep blue
	colorBlue2 = "\033[38;5;33m" // Blue
	colorCyan1 = "\033[38;5;39m" // Light blue
	colorCyan2 = "\033[38;5;45m" // Cyan
	colorCyan3 = "\033[38;5;51m" // Bright cyan
	colorReset = "\033[0m"
	colorDim   = "\033[2m"
)

// printHeaderToWriter はセッション開始時のヘッダーを表示
func printHeaderToWriter(out io.Writer, model string, provider api.Provider) {
	// ASCII logo with info on the right side
	// Logo lines paired with info text
	type lineInfo struct {
		color string
		logo  string
		info  string
	}

	lines := []lineInfo{
		{colorBlue1, `██╗  ██╗`, ""},
		{colorBlue1, `╚██╗██╔╝`, fmt.Sprintf("%sXELYON%s v%s", colorCyan2, colorReset, version.GetVersion())},
		{colorBlue2, ` ╚███╔╝ `, fmt.Sprintf("%sAI-powered coding agent%s", colorDim, colorReset)},
		{colorCyan1, ` ██╔██╗ `, ""},
		{colorCyan2, `██╔╝ ██╗`, fmt.Sprintf("Provider: %s", provider.Name())},
		{colorCyan3, `╚═╝  ╚═╝`, fmt.Sprintf("Model: %s", modelDisplayName(model))},
	}

	// Print logo with info
	_, _ = fmt.Fprintln(out)
	for _, l := range lines {
		if l.info == "" {
			_, _ = fmt.Fprintf(out, "  %s%s%s\n", l.color, l.logo, colorReset)
		} else {
			_, _ = fmt.Fprintf(out, "  %s%s%s   %s\n", l.color, l.logo, colorReset, l.info)
		}
	}
	_, _ = fmt.Fprintln(out)
}

func printModeInfoToWriter(out io.Writer, autoApprove, dryRun bool) {
	var modes []string
	if autoApprove {
		modes = append(modes, "Auto-approve")
	}
	if dryRun {
		modes = append(modes, "Dry-run")
	}

	// 特殊モードのときだけ表示
	if len(modes) > 0 {
		yellow.Fprintf(out, "  Mode: %s\n\n", strings.Join(modes, ", "))
	}

	cyan.Fprintln(out, "  ─────────────────────────────────────────")
	yellow.Fprintln(out, "  Type /help for commands, /exit to quit")
}

// modelDisplayName はモデル名を表示用にフォーマット
func modelDisplayName(model string) string {
	switch model {
	case "deepseek-chat":
		return "DeepSeek V3 (balanced)"
	case "deepseek-coder":
		return "DeepSeek Coder (code-focused)"
	case "deepseek-reasoner":
		return "DeepSeek R1 (reasoning)"
	case "claude":
		return "Claude (Vertex AI)"
	default:
		return model
	}
}

// loadProjectConfig はプロジェクト設定をロード（xelyon.yaml）
func loadProjectConfig() *config.ProjectConfig {
	return config.LoadProjectConfig()
}

// injectProjectConfig は ProjectConfig を SystemPrompt に注入する。
// Rules → InjectProjectRules, Context → 末尾に追加
func injectProjectConfig(systemPrompt string, pc *config.ProjectConfig) string {
	if pc == nil {
		return systemPrompt
	}

	rulesBlock := prompt.BuildRulesBlockFromList(pc.Rules)
	if rulesBlock != "" {
		systemPrompt = prompt.InjectProjectRules(systemPrompt, rulesBlock)
	}
	if pc.Context != "" {
		systemPrompt += "\n\n## Project Context:\n" + pc.Context
	}

	return systemPrompt
}

// applyProjectConfig はプロジェクト設定をエージェントに適用する統一ヘルパー。
// SystemPrompt 注入 + hooks 解決 + UI 表示を行う。
func applyProjectConfig(agent *Agent, pc *config.ProjectConfig) {
	if pc == nil {
		return
	}

	// 1. System prompt 注入
	agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc)

	// 2. hooks 解決（xelyon.yaml 優先、config.yaml フォールバック）
	if resolved := config.ResolveHooks(agent.cfg(), pc); resolved != nil {
		cfg := agent.cfg()
		cfg.Hooks = *resolved
	}

	// 3. UI 表示
	green.Fprintln(agent.output(), "📋 xelyon.yaml loaded")
}

// injectProjectMap はプロジェクト構造マップをシステムプロンプトに注入する。
func injectProjectMap(agent *Agent) {
	if agent == nil {
		return
	}

	agent.projectMapFileCount = 0
	agent.projectMapSymbolCount = 0

	cfg := agent.cfg()
	if !cfg.ProjectMap.Enabled {
		return
	}
	if !common.IsRipgrepAvailable() {
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	pm := repomap.NewProjectMap(cwd, 0, cfg.ProjectMap.AdditionalIgnoreDirs...)
	if err := pm.Build(); err != nil {
		yellow.Fprintf(agent.output(), "⚠️ ProjectMap build failed: %v\n", err)
		return
	}
	pm.MaxTokens = calcProjectMapBudget(agent, cfg, pm.GetFileCount(), pm.GetSymbolCount())

	mapStr := pm.Generate()
	if mapStr == "" {
		return
	}

	agent.SystemPrompt += "\n\n" + mapStr
	agent.projectMapFileCount = pm.GetFileCount()
	agent.projectMapSymbolCount = pm.GetSymbolCount()

	green.Fprintf(agent.output(), "🗺️  Project map loaded (%d symbols from %d files)\n", agent.projectMapSymbolCount, agent.projectMapFileCount)
}

func calcProjectMapBudget(agent *Agent, cfg *config.Config, fileCount, symbolCount int) int {
	// コンテキストウィンドウサイズを取得
	contextWindow := token.GetModelTokenLimit(agent.CurrentModel)
	if contextWindow <= 0 {
		contextWindow = 128000 // フォールバック
	}

	ratio := effectiveProjectMapContextRatio(cfg.ProjectMap.ContextRatio, fileCount, symbolCount)
	budgetCap := int(float64(contextWindow) * ratio)

	if budgetCap < 1 {
		return 1
	}

	return budgetCap
}

func effectiveProjectMapContextRatio(baseRatio float64, fileCount, symbolCount int) float64 {
	ratio := config.NormalizeProjectMapContextRatio(baseRatio)

	switch {
	case fileCount >= 400 || symbolCount >= 4000:
		if ratio < 0.04 {
			return 0.04
		}
	case fileCount >= 200 || symbolCount >= 2000:
		if ratio < 0.03 {
			return 0.03
		}
	}

	return ratio
}

// rebuildSystemPromptForCurrentProvider は現在の provider/model に合わせて
// SystemPrompt をベースから再構築する。
func (a *Agent) rebuildSystemPromptForCurrentProvider() {
	if a == nil || a.CurrentProvider == nil {
		return
	}

	planningPrompt := promptplan.BuildPlanningPrompt()
	hadPlanPrompt := strings.Contains(a.SystemPrompt, planningPrompt)

	systemPrompt := prompt.SystemPrompt
	if a.mcpManager != nil && len(a.mcpManager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(a.mcpManager)
	}
	systemPrompt = prompt.BuildProviderSystemPromptWithConfig(systemPrompt, a.CurrentProvider.Name(), a.CurrentModel, a.cfg())

	if pc := loadProjectConfig(); pc != nil {
		systemPrompt = injectProjectConfig(systemPrompt, pc)
	}

	if hadPlanPrompt {
		systemPrompt += api.SystemPromptCacheBoundary + planningPrompt
	}

	a.SystemPrompt = systemPrompt
	injectProjectMap(a)
}

func estimateProjectConfigTokens(pc *config.ProjectConfig) int {
	if pc == nil {
		return 0
	}

	total := 0
	if rulesBlock := prompt.BuildRulesBlockFromList(pc.Rules); rulesBlock != "" {
		total += token.EstimateTokenCount(rulesBlock)
	}
	if pc.Context != "" {
		total += token.EstimateTokenCount("\n\n## Project Context:\n" + pc.Context)
	}
	return total
}

func (a *Agent) syncSessionModel() {
	if a == nil || a.session == nil {
		return
	}
	a.session.Model = a.CurrentModel
}

func summarizeStatusError(err error) string {
	if err == nil {
		return "Request failed"
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "Request failed"
	}

	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		msg = strings.TrimSpace(msg[:idx])
	}

	const maxReasonLen = 120
	if len(msg) > maxReasonLen {
		msg = msg[:maxReasonLen-3] + "..."
	}

	return msg
}

func cancelDebugEnabled() bool {
	return os.Getenv("XELYON_DEBUG_CANCEL") == "1"
}

func (a *Agent) debugCancelf(format string, args ...any) {
	if a == nil || !cancelDebugEnabled() {
		return
	}
	_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Cancel] "+format+"\n", args...)
}

func (a *Agent) cancelActiveRequest(reason string) {
	if a == nil {
		return
	}

	if reason != "" {
		a.lastCancelReason = reason
	}

	if a.cancelFunc == nil {
		a.debugCancelf("cancel requested without active request (reason=%q)", reason)
		return
	}

	a.debugCancelf("canceling active request (reason=%q)", reason)
	a.cancelFunc()
}

func (a *Agent) statusReasonForError(err error) string {
	reason := summarizeStatusError(err)
	if a == nil {
		return reason
	}

	if reason != "Request failed" && strings.TrimSpace(a.lastCancelReason) != "" && strings.Contains(reason, "context canceled") {
		return reason + " [" + a.lastCancelReason + "]"
	}

	return reason
}
