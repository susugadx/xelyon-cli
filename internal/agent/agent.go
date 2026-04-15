package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"

	"github.com/susugadx/xelyon-cli/internal/i18n"

	// Subpackage imports - trigger init() for tool registration
	_ "github.com/susugadx/xelyon-cli/internal/tools/applypatch"
	toolsdev "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/gathercontext"
	_ "github.com/susugadx/xelyon-cli/internal/tools/navigation"
	_ "github.com/susugadx/xelyon-cli/internal/tools/planning"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// 色定義
var (
	cyan   = color.New(color.FgCyan, color.Bold)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	dim    = color.New(color.Faint)

	readFileOutlineFooterPattern = regexp.MustCompile(`\(\d+ lines total(?:\.[^)]*)?\)`)
)

type agentConversationState struct {
	session         *history.Session
	storage         *history.Storage
	lastOutputs     []string
	compactedItems  []api.InputItem
	isCompactedMode bool
}

type agentRequestState struct {
	cancelFunc           context.CancelFunc
	requestCtx           context.Context
	lastCancelReason     string
	strReplaceErrorCount int
	tokenLimitRetryCount int
	activeApprovedPlan   string
}

type agentWorkspaceState struct {
	changeStack      []tools.FileChange
	taskChangeOffset int
	changeStorage    *history.ChangeStorage
	taskTestResult   *bool
	taskTestCommand  string
	pendingLSPFiles  []string
}

type agentProjectPromptState struct {
	projectMapFileCount    int
	projectMapSymbolCount  int
	projectMap             *repomap.ProjectMap
	projectMapRootPath     string
	projectMapIgnoreKey    string
	projectMapStateKey     string
	projectMapWatchDirs    []string
	projectMapBaseSection  string
	projectMapFocusSection string
	projectMapSection      string
	projectMapBaseKey      string
	projectMapFocusKey     string
	projectMapDirty        bool
}

// Agent はCLIエージェント
type Agent struct {
	Model                           string // 初期モデル（後方互換性のため保持）
	CurrentModel                    string // 現在のモデル（再起動なしで切り替え可能）
	CurrentProvider                 api.Provider
	ProviderName                    string
	ProviderConfigKey               string
	Runtime                         *AgentRuntime
	History                         []api.Message
	SystemPrompt                    string
	mcpManager                      *mcp.Manager
	lspClient                       *lsp.Client       // LSPクライアント
	AutoApprove                     bool              // --auto-approve フラグ
	Stats                           *SessionStats     // セッション統計情報
	PlanModeEnabled                 bool              // Plan Mode ON/OFF（デフォルト: false）
	PendingApprovedPlan             string            // 次の通常ターンへ1回だけ handoff する承認済み plan
	PendingApprovedPlanHasChanges   bool              // 現在の承認済み plan で一度でも FileChange が記録されたか
	PendingApprovedPlanChangedFiles []string          // 現在の承認済み plan で記録された変更ファイル集合
	ToolCache                       *ToolCache        // ツール結果キャッシュ（read_file, list_dir）
	LocatorRegistry                 *locator.Registry // Locator ID レジストリ（セッション内追記のみ）
	status                          statusHolder

	agentConversationState
	agentRequestState
	agentWorkspaceState
	agentProjectPromptState

	// exitHook は os.Exit 前に呼ばれるフック（TUI モードのターミナル復旧等）
	exitHook func()

	// tuiToolResultCh は TUI モードでツール実行結果を構造化データとして送信するチャネル。
	// nil の場合は従来の stdout 出力を使用する。
	tuiToolResultCh     chan tools.ToolResultInfo
	tuiToolResultClosed atomic.Bool // TUI 終了後の send panic / deadlock 防止

	// 並列実行用ミューテックス
	historyMu     sync.Mutex
	changeStackMu sync.Mutex
	statsMu       sync.Mutex

	// per-turn observability for conservatively detecting serial single-pattern
	// search_code calls that could have been grouped into one multi-pattern call.
	searchCodeRecentSinglePatternByFamily map[string]string
	searchCodeMissedMultiCountedFamilies  map[string]struct{}
}

func (a *Agent) setPromptReader(reader *ui.MultilineReader) {
	if a == nil {
		return
	}
	a.ui().SetPromptReader(reader)
}

func (a *Agent) output() io.Writer {
	if a == nil {
		return ui.DefaultRuntime().Output()
	}
	return a.ui().Output()
}

func (a *Agent) errorOutput() io.Writer {
	if a == nil {
		return ui.DefaultRuntime().ErrorOutput()
	}
	return a.ui().ErrorOutput()
}

func (a *Agent) appendSessionMessage(role, content, model string) {
	if a == nil || a.session == nil {
		return
	}
	a.session.AddMessage(role, content, model)
	a.persistSession()
}

func (a *Agent) appendSessionMessageFromAPI(msg api.Message, model string) {
	if a == nil || a.session == nil {
		return
	}
	a.session.AddMessageFromAPI(msg, model)
	a.persistSession()
}

func (a *Agent) appendSessionToolExecution(toolCall *tools.ToolCall, result string) {
	if a == nil || a.session == nil || toolCall == nil {
		return
	}
	success := !strings.HasPrefix(strings.TrimSpace(result), "Error:")
	a.session.AddToolExecution(toolCall.Tool, toolCall.Args, result, success, a.CurrentModel)
	a.persistSession()
}

func (a *Agent) persistSession() {
	if a == nil || a.session == nil || a.storage == nil {
		return
	}
	a.syncApprovedPlanStateToSession()
	a.syncResponseIDToSession()
	if err := a.storage.Save(a.session); err != nil {
		yellow.Fprintf(a.output(), "⚠️  Warning: Failed to save session: %v\n", err)
	}
}

// NewAgent は新しいAgentを作成
func NewAgent(model string, provider api.Provider, headless bool) *Agent {
	return NewAgentWithRuntime(model, provider, headless, NewAgentRuntimeWithConfig(config.DefaultConfig()))
}

// NewAgentWithRuntime は runtime を指定して新しい Agent を作成する。
func NewAgentWithRuntime(model string, provider api.Provider, headless bool, runtime *AgentRuntime) *Agent {
	runtime = normalizeAgentRuntime(runtime)
	api.ApplyRuntimeConfig(provider, runtime.effectiveConfig())
	runtimeUI := runtime.effectiveUI()
	api.ApplyUIRuntime(provider, runtimeUI)
	out := runtimeUI.Output()
	errOut := runtimeUI.ErrorOutput()

	// 言語設定を適用
	cfg := runtime.effectiveConfig()
	if cfg.General.UILanguage != "" && cfg.General.UILanguage != "auto" {
		i18n.SetLang(cfg.General.UILanguage)
	}

	// 古いartifactファイルを削除
	toolsdev.CleanupArtifactsWithWriter(errOut)

	storage, err := history.NewStorage()
	if err != nil {
		red.Fprintf(out, "Warning: Failed to initialize history storage: %v\n", err)
		storage = nil
	}

	// MCP初期化（設定と環境変数で制御）
	mcpManager := mcp.NewManager()
	mcpManager.SetOutput(errOut)
	if cfg.MCP.Enabled && (!headless || cfg.MCP.Headless) && os.Getenv("XELYON_DISABLE_MCP") != "1" {
		if err := mcpManager.LoadConfig(); err != nil {
			yellow.Fprintf(out, "Warning: Failed to load MCP config: %v\n", err)
		}

		ctx := context.Background()
		if err := mcpManager.Connect(ctx); err != nil {
			yellow.Fprintf(out, "Warning: MCP connection error: %v\n", err)
		}

		// MCPツールをTool Registryに登録
		if len(mcpManager.GetTools()) > 0 {
			mcpManager.RegisterToToolRegistry(runtime.effectiveRegistry())
		}
	}

	toolVisibility := resolveToolVisibilityPolicy(provider.Name(), model, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})
	systemPrompt := prompt.GetSystemPromptForProvider(provider.Name(), model)

	// MCPツールをSystemPromptに追加
	if len(mcpManager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(mcpManager)
	}

	// 変更履歴ストレージ初期化
	changeStorage, err := history.NewChangeStorage()
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize change storage: %v\n", err)
		changeStorage = nil
	}

	// LSP初期化
	var lspClient *lsp.Client
	cfg = runtime.effectiveConfig()
	if cfg.LSP.Enabled {
		cwd, err := os.Getwd()
		if err == nil {
			lspClient = lsp.NewClient(cwd)
			lspClient.SetOutput(errOut)
			lspClient.SetErrorOutput(errOut)
			// Config形式からLSP形式に変換
			servers := make(map[string]lsp.ServerConfig)
			for lang, serverCfg := range cfg.LSP.Servers {
				servers[lang] = lsp.ServerConfig{
					Command:  serverCfg.Command,
					Args:     serverCfg.Args,
					Disabled: serverCfg.Disabled,
				}
			}
			lspClient.SetConfigs(servers)

			if lspClient != nil && !shouldSkipLSPWarmup() {
				go func() {
					warmCtx, warmCancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer warmCancel()
					if _, err := lspClient.GetServer(warmCtx, "go"); err != nil {
						fmt.Fprintf(errOut, "LSP warm-up: gopls not available (%v)\n", err)
					}
				}()
			}

			// LSPツールはinit()で自動登録済み
			// LSPドキュメントはSystemPromptのWorkflow Rulesに統合済み
		}
	}

	// MCPProviderインターフェースを実装するプロバイダーにMCPツールを設定
	// （Function Calling経由で呼び出し可能にする）
	configureMCPTools(provider, mcpManager.GetTools(), errOut)

	runtime.effectiveRegistry().SetExcludedTools(toolVisibility.excluded())

	// プロバイダー別プレフィックスを Workflow Rules の直前に注入
	systemPrompt = prompt.BuildProviderSystemPromptWithConfig(systemPrompt, provider.Name(), model, runtime.effectiveConfig())

	// ToolCache 初期化（ディスクから復元）
	toolCache := runtime.effectiveToolCache()

	// Agent を作成
	agent := &Agent{
		Model:             model,
		CurrentModel:      model,
		CurrentProvider:   provider,
		ProviderName:      strings.ToLower(provider.Name()),
		ProviderConfigKey: providerConfigKeyFromProvider(provider),
		Runtime:           runtime,
		History:           []api.Message{},
		mcpManager:        mcpManager,
		lspClient:         lspClient,
		SystemPrompt:      systemPrompt,
		Stats:             NewSessionStats(strings.ToLower(provider.Name()), model),
		ToolCache:         toolCache,
		LocatorRegistry:   locator.NewRegistry(),
		status:            statusHolder{status: defaultStatus()},
		agentConversationState: agentConversationState{
			session:     history.NewSession(model),
			storage:     storage,
			lastOutputs: []string{},
		},
		agentWorkspaceState: agentWorkspaceState{
			changeStack:   []tools.FileChange{},
			changeStorage: changeStorage,
		},
	}

	// Usage callback を設定（プロバイダーがサポートしている場合）
	if reporter, ok := provider.(api.UsageReporter); ok {
		reporter.SetUsageCallback(func(u api.Usage) {
			agent.statsMu.Lock()
			defer agent.statsMu.Unlock()
			agent.Stats.AddUsage(u)
		})
	}

	return agent
}

// shouldSkipLSPWarmup は環境変数指定時に warm up を無効化する。
func shouldSkipLSPWarmup() bool {
	return os.Getenv("XELYON_DISABLE_LSP_WARMUP") == "1"
}

// cleanupHook はテスト用フック（非nil時にCleanupから呼ばれる）
var cleanupHook func()

// syncResponseIDToSession はプロバイダーの ResponseID をセッションに同期する（保存前に呼ぶ）
func (a *Agent) syncResponseIDToSession() {
	if a.session == nil {
		return
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		if ridProvider.HasCachedResponseID() {
			a.session.ResponseID = ridProvider.GetResponseID()
		}
	}
}

// Cleanup はエージェントのリソースをクリーンアップ
func (a *Agent) Cleanup() {
	if cleanupHook != nil {
		cleanupHook()
	}
	if a.mcpManager != nil {
		a.mcpManager.Close()
	}
	// LSPクリーンアップ
	if a.lspClient != nil {
		a.lspClient.Close()
	}
	// ToolCache 永続化
	if a.ToolCache != nil {
		if err := a.ToolCache.Save(); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to save tool cache: %v\n", err)
		}
	}
	// セッション保存
	if a.storage != nil && a.session != nil {
		a.syncApprovedPlanStateToSession()
		a.syncResponseIDToSession()
		if err := a.storage.Save(a.session); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to save session: %v\n", err)
		}
	}
}

// GetLSPClient はLSPクライアントを返す（コマンド用）
func (a *Agent) GetLSPClient() *lsp.Client {
	return a.lspClient
}

// appendHistory は History へスレッドセーフに追加（並列実行時用）
func (a *Agent) appendHistory(msg api.Message) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	a.History = append(a.History, msg)
}

// getHistorySnapshot は History のスナップショットをスレッドセーフに取得（並列実行時用）
func (a *Agent) getHistorySnapshot() []api.Message {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	snapshot := make([]api.Message, len(a.History))
	copy(snapshot, a.History)
	return snapshot
}

// appendChange は changeStack へスレッドセーフに追加（並列実行時用）
func (a *Agent) appendChange(change tools.FileChange) {
	a.changeStackMu.Lock()
	defer a.changeStackMu.Unlock()
	a.changeStack = append(a.changeStack, change)
	if len(a.changeStack) > config.MaxChangeStack {
		a.changeStack = a.changeStack[1:]
	}
}

// incrementToolExecution は Stats をスレッドセーフに更新（並列実行時用）
func (a *Agent) incrementToolExecution(toolName string) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.AddToolExecution(toolName)
	}
}

// incrementAssistantMessages は Stats をスレッドセーフに更新（並列実行時用）
func (a *Agent) incrementAssistantMessages() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}
}

func (a *Agent) addOptimizationMetrics(metrics OptimizationMetrics) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.Optimizations.add(metrics)
	}
}

func (a *Agent) addCompactionMetrics(metrics CompactionMetrics) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.Optimizations.addCompaction(metrics)
	}
}

func isReadFileOutlineResult(result string) bool {
	return readFileOutlineFooterPattern.MatchString(result)
}

func (a *Agent) recordToolResultOptimizations(tc *tools.ToolCall, result string) {
	if tc.Tool == "read_file" && isReadFileOutlineResult(result) {
		a.addOptimizationMetrics(OptimizationMetrics{OutlineFirstCount: 1})
	}
	a.recordToolObservability(tc.Tool, tc.RawArgs, tc.Args, result)
}

// recordToolObservability はツール実行のobservabilityメトリクスを記録する。
// rawArgs は FC 経由の場合に map[string]any、XML rescue 経由では nil になるため、
// stringArgs（ToolCall.Args）をフォールバックとして参照する。
func (a *Agent) recordToolObservability(toolName string, rawArgs map[string]any, stringArgs map[string]string, result string) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats == nil {
		return
	}
	obs := &a.Stats.ToolObs

	switch toolName {
	case "read_file":
		// batch read: paths 引数が2つ以上の有効パスを持つ場合
		if pathsVal, ok := rawArgs["paths"]; ok {
			if isBatchPaths(pathsVal) {
				obs.ReadFileBatchCalls++
			}
		} else if pathsStr, ok := stringArgs["paths"]; ok {
			// XML rescue フォールバック: Args["paths"] は JSON 文字列
			if isBatchPaths(pathsStr) {
				obs.ReadFileBatchCalls++
			}
		}
		if hasReadFileTargets(rawArgs, stringArgs) {
			obs.ReadFileTargetCalls++
		}
		// empty-path error: canonical error string を検出
		if result == "Error: paths is empty" {
			obs.ReadFileEmptyPathsErrors++
		}
	case "search_code":
		if isSearchCodeImpact(rawArgs, stringArgs) {
			obs.SearchCodeImpactCalls++
			return
		}
		// explicit multi-pattern only. impact は独立メトリクスとして扱う。
		if isObservedSearchCodeExplicitMulti(rawArgs, stringArgs) {
			obs.SearchCodeExplicitMultiCalls++
			obs.SearchCodeMultiPatternCalls++
			return
		}
		a.recordSearchCodeMissedMultiLocked(rawArgs, stringArgs)
	}
}

func (a *Agent) resetSearchCodeTurnObservability() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	a.searchCodeRecentSinglePatternByFamily = make(map[string]string)
	a.searchCodeMissedMultiCountedFamilies = make(map[string]struct{})
}

func (a *Agent) recordSearchCodeMissedMultiLocked(rawArgs map[string]any, stringArgs map[string]string) {
	args := buildSearchCodeObservabilityArgs(rawArgs, stringArgs)
	pattern := strings.TrimSpace(args["pattern"])
	if pattern == "" || isMultiPatternArg(pattern) {
		return
	}

	exactPattern := normalizeSearchCodeObservedPattern(pattern, false)
	familyPattern := normalizeSearchCodeObservedPattern(pattern, true)
	if exactPattern == "" || familyPattern == "" {
		return
	}

	optionsKey := searchCodeOptionsKey(&tools.ToolCall{
		Tool: "search_code",
		Args: args,
	})
	if optionsKey == "" {
		return
	}

	if a.searchCodeRecentSinglePatternByFamily == nil {
		a.searchCodeRecentSinglePatternByFamily = make(map[string]string)
	}
	if a.searchCodeMissedMultiCountedFamilies == nil {
		a.searchCodeMissedMultiCountedFamilies = make(map[string]struct{})
	}

	familyKey := optionsKey + "|family=" + familyPattern
	if _, counted := a.searchCodeMissedMultiCountedFamilies[familyKey]; counted {
		return
	}

	prevPattern, seen := a.searchCodeRecentSinglePatternByFamily[familyKey]
	if !seen {
		a.searchCodeRecentSinglePatternByFamily[familyKey] = exactPattern
		return
	}
	if prevPattern == exactPattern {
		return
	}

	a.Stats.ToolObs.SearchCodeMissedMultiPattern++
	a.searchCodeMissedMultiCountedFamilies[familyKey] = struct{}{}
}

func buildSearchCodeObservabilityArgs(rawArgs map[string]any, stringArgs map[string]string) map[string]string {
	args := make(map[string]string, len(stringArgs)+len(rawArgs))
	for k, v := range stringArgs {
		args[k] = v
	}
	for k, v := range rawArgs {
		if s, ok := stringifySearchCodeArg(v); ok {
			args[k] = s
		}
	}
	return args
}

func isSearchCodeImpact(rawArgs map[string]any, stringArgs map[string]string) bool {
	args := buildSearchCodeObservabilityArgs(rawArgs, stringArgs)
	return strings.EqualFold(strings.TrimSpace(args["intent"]), "impact")
}

func isObservedSearchCodeExplicitMulti(rawArgs map[string]any, stringArgs map[string]string) bool {
	args := buildSearchCodeObservabilityArgs(rawArgs, stringArgs)
	pattern := strings.TrimSpace(args["pattern"])
	if pattern == "" {
		return false
	}
	return isMultiPatternArg(pattern)
}

func stringifySearchCodeArg(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case bool:
		if val {
			return "true", true
		}
		return "false", true
	case int:
		return fmt.Sprintf("%d", val), true
	case int64:
		return fmt.Sprintf("%d", val), true
	case float64:
		return fmt.Sprintf("%g", val), true
	default:
		return "", false
	}
}

func normalizeSearchCodeObservedPattern(pattern string, stripFamilyNoise bool) string {
	tokens := tokenizeObservedSearchCodePattern(pattern)
	if len(tokens) == 0 {
		return ""
	}
	if !stripFamilyNoise {
		return strings.Join(tokens, " ")
	}

	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, noise := searchCodeFamilyNoiseTerms[token]; noise {
			continue
		}
		filtered = append(filtered, token)
	}
	if len(filtered) == 0 {
		return ""
	}
	return strings.Join(filtered, " ")
}

func tokenizeObservedSearchCodePattern(pattern string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(pattern)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

var searchCodeFamilyNoiseTerms = map[string]struct{}{
	"definition":     {},
	"definitions":    {},
	"caller":         {},
	"callers":        {},
	"ref":            {},
	"refs":           {},
	"reference":      {},
	"references":     {},
	"test":           {},
	"tests":          {},
	"impl":           {},
	"implementation": {},
}

// isBatchPaths は paths 引数が実質的な batch（2パス以上）か判定する。
// XML rescue 経由ではタグ内に前後空白・改行が含まれるため TrimSpace してから判定する。
func isBatchPaths(pathsVal any) bool {
	switch v := pathsVal.(type) {
	case []any:
		return len(v) >= 2
	case string:
		// JSON 文字列の場合: "[" で始まりカンマを含む → 2要素以上
		s := strings.TrimSpace(v)
		return len(s) > 2 && s[0] == '[' && strings.Contains(s, ",")
	}
	return false
}

func hasReadFileTargets(rawArgs map[string]any, stringArgs map[string]string) bool {
	if targetsVal, ok := rawArgs["targets"]; ok {
		if targets, ok := stringifySearchCodeArg(targetsVal); ok {
			return strings.TrimSpace(targets) != ""
		}
	}
	if targets, ok := stringArgs["targets"]; ok {
		return strings.TrimSpace(targets) != ""
	}
	return false
}

// isMultiPatternArg は pattern 引数が multi-pattern（カンマ区切り2パターン以上）か判定する。
// search_code の splitPatterns と同じロジック: \, はリテラルカンマとして除外。
func isMultiPatternArg(patternVal any) bool {
	s, ok := patternVal.(string)
	if !ok || s == "" {
		return false
	}
	// \, をプレースホルダーに置換してからカンマでsplit
	replaced := strings.ReplaceAll(s, `\,`, "\x00")
	parts := strings.Split(replaced, ",")
	count := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	return count >= 2
}
