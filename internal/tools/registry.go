package tools

import (
	"fmt"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/audit"
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
	Run(args map[string]string) (output string, change *FileChange, err error)
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
	excludedTools map[string]bool // GetToolDefinitions から除外するツール名セット
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
}

// Execute はツール呼び出しを実行（スレッドセーフ + 監査ログ記録）
func (r *Registry) Execute(tc *ToolCall) (string, *FileChange) {
	r.mu.RLock()
	tool, ok := r.tools[tc.Tool]
	r.mu.RUnlock()

	if !ok {
		return fmt.Sprintf("Unknown tool: %s", tc.Tool), nil
	}

	output, change, err := tool.Run(tc.Args)

	// 監査ログ記録（失敗しても処理続行）
	logger := audit.GetLogger()
	logger.LogToolExecution(tc.Tool, tc.Args, output, err, change != nil)

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

// HasTool は指定名のツールが登録されているかを返す（スレッドセーフ）
func (r *Registry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// SetExcludedTools は GetToolDefinitions から除外するツール名を設定
// Normal Mode で planning 系ツールを非表示にするために使用
func (r *Registry) SetExcludedTools(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.excludedTools = make(map[string]bool, len(names))
	for _, n := range names {
		r.excludedTools[n] = true
	}
}

// ClearExcludedTools は除外ツール設定をクリア
func (r *Registry) ClearExcludedTools() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.excludedTools = nil
}

// GetToolDefinitions は登録済みツールの定義を返す（除外設定を適用）
func (r *Registry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
	return defs
}

// DefaultRegistry はデフォルトのツールレジストリ
var DefaultRegistry = NewRegistry()
