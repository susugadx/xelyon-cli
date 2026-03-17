package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
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
// 入力内容に一致した rules/context のみを注入する。
func injectProjectConfig(systemPrompt string, pc *config.ProjectConfig, input string) string {
	systemPrompt = prompt.StripProjectConfigSections(systemPrompt)
	if pc == nil {
		return systemPrompt
	}

	selection := config.SelectProjectPromptSelection(pc, input)
	projectBlock := prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts)
	return prompt.InjectProjectConfigBlock(systemPrompt, projectBlock)
}

// applyProjectConfig はプロジェクト設定をエージェントに適用する統一ヘルパー。
// SystemPrompt 注入 + hooks 解決 + UI 表示を行う。
func applyProjectConfig(agent *Agent, pc *config.ProjectConfig) {
	if pc == nil {
		return
	}

	// 1. System prompt 注入
	agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc, "")

	// 2. hooks 解決（xelyon.yaml 優先、config.yaml フォールバック）
	if resolved := config.ResolveHooks(agent.cfg(), pc); resolved != nil {
		cfg := agent.cfg()
		cfg.Hooks = *resolved
	}

	// 3. UI 表示
	green.Fprintln(agent.output(), "📋 xelyon.yaml loaded")
}

// injectProjectMap はプロジェクト構造マップをシステムプロンプトに注入する。
func injectProjectMap(agent *Agent, input string) {
	if agent == nil {
		return
	}

	agent.SystemPrompt = stripProjectMapSection(agent.SystemPrompt)
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

	pc := loadProjectConfig()
	rootPath := cwd
	if pc != nil && strings.TrimSpace(pc.FilePath) != "" {
		rootPath = filepath.Dir(pc.FilePath)
	}
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, pc)
	ignoreKey := strings.Join(ignorePatterns, "\x00")
	priorityPaths := config.ExtractReferencedProjectPathsForRoot(input, rootPath)

	pm, rebuilt := ensureProjectMapManifest(agent, rootPath, ignorePatterns, ignoreKey)
	if pm == nil {
		return
	}
	pm.MaxTokens = calcProjectMapBudget(agent, cfg, pm.GetFileCount(), pm.GetSymbolCount())

	if !rebuilt && agent.projectMapSection != "" && slices.Equal(agent.projectMapPriority, priorityPaths) && token.EstimateTokenCount(agent.projectMapSection) <= pm.MaxTokens {
		agent.SystemPrompt += "\n\n" + agent.projectMapSection
		agent.projectMapFileCount = pm.GetFileCount()
		agent.projectMapSymbolCount = pm.GetSymbolCount()
		agent.projectMapDirty = false
		return
	}
	mapStr := pm.GenerateManifest(priorityPaths)
	if mapStr == "" {
		agent.projectMapSection = ""
		agent.projectMapPriority = nil
		agent.projectMapDirty = false
		return
	}

	agent.SystemPrompt += "\n\n" + mapStr
	agent.projectMapFileCount = pm.GetFileCount()
	agent.projectMapSymbolCount = pm.GetSymbolCount()
	agent.projectMapSection = mapStr
	agent.projectMapPriority = append([]string(nil), priorityPaths...)
	agent.projectMapDirty = false

	if rebuilt {
		green.Fprintf(agent.output(), "🗺️  Project map loaded (manifest from %d files)\n", agent.projectMapFileCount)
	}
}

func ensureProjectMapManifest(agent *Agent, rootPath string, ignorePatterns []string, ignoreKey string) (*repomap.ProjectMap, bool) {
	if agent == nil {
		return nil, false
	}

	if !agent.projectMapDirty &&
		agent.projectMapManifest != nil &&
		agent.projectMapRootPath == rootPath &&
		agent.projectMapIgnoreKey == ignoreKey {
		if stateKey := currentProjectMapStateKey(agent, rootPath); stateKey != "" && agent.projectMapStateKey == stateKey {
			return agent.projectMapManifest, false
		}
	}

	pm := repomap.NewProjectMap(rootPath, 0, ignorePatterns...)
	if err := pm.BuildManifest(); err != nil {
		yellow.Fprintf(agent.output(), "⚠️ ProjectMap build failed: %v\n", err)
		return nil, false
	}

	agent.projectMapManifest = pm
	agent.projectMapRootPath = rootPath
	agent.projectMapIgnoreKey = ignoreKey
	agent.projectMapWatchDirs = nil
	if !isGitProjectMapAvailable(rootPath) {
		agent.projectMapWatchDirs = collectProjectMapWatchDirs(rootPath, ignorePatterns)
	}
	agent.projectMapStateKey = currentProjectMapStateKey(agent, rootPath)
	return pm, true
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
		systemPrompt = injectProjectConfig(systemPrompt, pc, "")
	}

	if hadPlanPrompt {
		systemPrompt += api.SystemPromptCacheBoundary + planningPrompt
	}

	a.SystemPrompt = systemPrompt
	injectProjectMap(a, "")
}

func estimateProjectConfigTokens(pc *config.ProjectConfig) int {
	if pc == nil {
		return 0
	}

	selection := config.SelectProjectPromptSelection(pc, "")
	return token.EstimateTokenCount(prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts))
}

func (a *Agent) refreshProjectPrompt(input string) {
	if a == nil {
		return
	}

	systemPrompt := stripProjectMapSection(prompt.StripProjectConfigSections(a.SystemPrompt))
	if pc := loadProjectConfig(); pc != nil {
		systemPrompt = injectProjectConfig(systemPrompt, pc, input)
	}
	a.SystemPrompt = systemPrompt
	injectProjectMap(a, input)
}

func (a *Agent) refreshProjectPromptIfDirty(input string) {
	if a == nil || !a.projectMapDirty {
		return
	}
	a.refreshProjectPrompt(input)
}

func (a *Agent) invalidateProjectMapManifest() {
	if a == nil {
		return
	}

	a.projectMapManifest = nil
	a.projectMapRootPath = ""
	a.projectMapIgnoreKey = ""
	a.projectMapStateKey = ""
	a.projectMapWatchDirs = nil
	a.projectMapSection = ""
	a.projectMapPriority = nil
	a.projectMapDirty = true
}

func currentProjectMapStateKey(agent *Agent, rootPath string) string {
	head := gitProjectMapHEAD(rootPath)
	status := gitProjectMapStatusDigest(rootPath)
	if head != "" || status != "" {
		return head + ":" + status
	}

	if digest := nonGitProjectMapWatchDigest(rootPath, projectMapWatchDirs(agent), projectMapIgnorePatterns(agent)); digest != "" {
		return "dirs:" + digest
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("fs:%d", info.ModTime().UTC().UnixNano())
}

func gitProjectMapHEAD(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"rev-parse", "HEAD"})
}

func gitProjectMapStatusDigest(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"status", "--porcelain"})
}

func gitProjectMapCommandDigest(rootPath string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = rootPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}

	sum := sha256.Sum256(bytes.TrimSpace(stdout.Bytes()))
	return hex.EncodeToString(sum[:])
}

func isGitProjectMapAvailable(rootPath string) bool {
	return gitProjectMapHEAD(rootPath) != "" || gitProjectMapStatusDigest(rootPath) != ""
}

func projectMapWatchDirs(agent *Agent) []string {
	if agent == nil || len(agent.projectMapWatchDirs) == 0 {
		return []string{"."}
	}

	dirs := make([]string, len(agent.projectMapWatchDirs))
	copy(dirs, agent.projectMapWatchDirs)
	return dirs
}

func projectMapIgnorePatterns(agent *Agent) []string {
	if agent == nil || agent.projectMapIgnoreKey == "" {
		return nil
	}
	return pathmatch.NormalizePatterns(strings.Split(agent.projectMapIgnoreKey, "\x00"))
}

func nonGitProjectMapWatchDigest(rootPath string, watchDirs []string, ignorePatterns []string) string {
	if len(watchDirs) == 0 {
		return ""
	}

	matcher := pathmatch.NewMatcher(ignorePatterns)
	var state strings.Builder
	for _, relDir := range watchDirs {
		relDir = filepath.Clean(filepath.ToSlash(strings.TrimSpace(relDir)))
		if relDir == "" {
			relDir = "."
		}

		absDir := rootPath
		if relDir != "." {
			absDir = filepath.Join(rootPath, relDir)
		}

		entries, err := os.ReadDir(absDir)
		switch {
		case err != nil:
			state.WriteString(relDir)
			state.WriteString(":missing\n")
		default:
			filtered := 0
			var entryState strings.Builder
			for _, entry := range entries {
				entryRelPath := entry.Name()
				if relDir != "." {
					entryRelPath = filepath.ToSlash(filepath.Join(relDir, entry.Name()))
				}
				if matcher.Match(entryRelPath, entry.IsDir()) {
					continue
				}
				filtered++
				entryState.WriteString(entry.Name())
				if entry.IsDir() {
					entryState.WriteByte('/')
				}
				entryState.WriteByte('\n')
			}
			_, _ = fmt.Fprintf(&state, "%s:%d\n", relDir, filtered)
			state.WriteString(entryState.String())
		}
	}

	sum := sha256.Sum256([]byte(state.String()))
	return hex.EncodeToString(sum[:])
}

func collectProjectMapWatchDirs(rootPath string, ignorePatterns []string) []string {
	matcher := pathmatch.NewMatcher(ignorePatterns)
	dirs := []string{"."}

	_ = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		relPath, relErr := filepath.Rel(rootPath, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.Clean(filepath.ToSlash(relPath))
		if relPath == "." {
			return nil
		}
		if matcher.Match(relPath, true) {
			return filepath.SkipDir
		}

		dirs = append(dirs, relPath)
		return nil
	})

	slices.Sort(dirs)
	return slices.Compact(dirs)
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
