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

	// Run はツールを実行
	// output: ツールの実行結果
	// change: ファイル変更情報（変更がない場合はnil）
	// err: エラー（エラーがない場合はnil）
	Run(args map[string]string) (output string, change *FileChange, err error)
}

// Registry はツールの登録・管理を行う
type Registry struct {
	mu    sync.RWMutex // 並行アクセス保護（MCP動的登録対応）
	tools map[string]Tool
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

// DefaultRegistry はデフォルトのツールレジストリ
var DefaultRegistry = NewRegistry()
