package tools

import (
	"fmt"
	"sort"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Tool はツールの共通インターフェース
type Tool interface {
	// Name はツール名を返す
	Name() string

	// Description はツールの説明を返す
	Description() string

	// Parameters はツールのパラメータ定義を返す（OpenAI形式）
	Parameters() map[string]interface{}

	// Run はツールを実行
	// output: ツールの実行結果
	// change: ファイル変更情報（変更がない場合はnil）
	// err: エラー（エラーがない場合はnil）
	Run(execCtx ExecutionContext, args map[string]string) (output string, change *FileChange, err error)
}

// ToolDefinition はツール定義（プロバイダー変換用）
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// Registry はツールの登録・管理を行う
type Registry struct {
	mu            sync.RWMutex // 並行アクセス保護（MCP動的登録対応）
	tools         map[string]Tool
	excludedTools map[string]bool // 現在の surface から除外し、実行も拒否するツール名セット
}

// NewRegistry は新しいRegistryを作成
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register はツールを登録（スレッドセーフ）
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
	r.syncDefaultToolDefinitionsLocked()
}

// ExecuteWithContext は実行コンテキスト付きでツール呼び出しを実行する。
func (r *Registry) ExecuteWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	r.mu.RLock()
	tool, ok := r.tools[tc.Tool]
	excluded := r.excludedTools[tc.Tool]
	r.mu.RUnlock()

	if !ok {
		return fmt.Sprintf("Unknown tool: %s", tc.Tool), nil
	}
	if excluded {
		return fmt.Sprintf("Error: tool not available in current mode: %s", tc.Tool), nil
	}

	output, change, err := tool.Run(normalizeExecutionContext(execCtx), tc.Args)

	// 監査ログ記録（失敗しても処理続行）
	if logger := normalizeExecutionContext(execCtx).EffectiveAuditLogger(); logger != nil {
		logger.LogToolExecution(tc.Tool, tc.Args, output, err, change != nil)
	}

	if err != nil {
		return fmt.Sprintf("Error: %v", err), change
	}

	return output, change
}

// GetTool は登録されたツールを取得（スレッドセーフ）
// 主にテスト用途
func (r *Registry) GetTool(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// Clone は現在の Registry をコピーした新しい Registry を返す。
func (r *Registry) Clone() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cloned := &Registry{
		tools:         make(map[string]Tool, len(r.tools)),
		excludedTools: make(map[string]bool, len(r.excludedTools)),
	}
	for name, tool := range r.tools {
		cloned.tools[name] = tool
	}
	for name, excluded := range r.excludedTools {
		cloned.excludedTools[name] = excluded
	}
	if len(cloned.excludedTools) == 0 {
		cloned.excludedTools = nil
	}
	return cloned
}

// HasTool は指定名のツールが登録されているかを返す（スレッドセーフ）
func (r *Registry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// SetExcludedTools は現在の surface から除外するツール名を設定する。
// 除外されたツールは定義から消え、直接実行も拒否される。
func (r *Registry) SetExcludedTools(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.excludedTools = make(map[string]bool, len(names))
	for _, n := range names {
		r.excludedTools[n] = true
	}
	r.syncDefaultToolDefinitionsLocked()
}

// GetExcludedTools は現在の除外ツール名リストを返す
func (r *Registry) GetExcludedTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.excludedTools))
	for n := range r.excludedTools {
		names = append(names, n)
	}
	return names
}

// ClearExcludedTools は除外ツール設定をクリア
func (r *Registry) ClearExcludedTools() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.excludedTools = nil
	r.syncDefaultToolDefinitionsLocked()
}

// GetToolDefinitions は登録済みツールの定義を返す（除外設定を適用）
func (r *Registry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.toolDefinitionsLocked()
}

func (r *Registry) toolDefinitionsLocked() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		if r.excludedTools[tool.Name()] {
			continue
		}
		defs = append(defs, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

// GetAPIToolDefinitions は provider へ渡す API 形式のツール定義を返す。
func (r *Registry) GetAPIToolDefinitions() []api.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return apiToolDefinitionsFromToolDefinitions(r.toolDefinitionsLocked())
}

func (r *Registry) syncDefaultToolDefinitionsLocked() {
	if DefaultRegistry == nil || r != DefaultRegistry {
		return
	}
	api.SetDefaultToolDefinitions(apiToolDefinitionsFromToolDefinitions(r.toolDefinitionsLocked()))
}

func apiToolDefinitionsFromToolDefinitions(defs []ToolDefinition) []api.ToolDefinition {
	apiDefs := make([]api.ToolDefinition, 0, len(defs))
	for _, def := range defs {
		apiDefs = append(apiDefs, api.ToolDefinition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}
	return apiDefs
}

// DefaultRegistry はデフォルトのツールレジストリ
var DefaultRegistry = NewRegistry()
