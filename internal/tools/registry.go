package tools

import "fmt"

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
	tools map[string]Tool
}

// NewRegistry は新しいRegistryを作成
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register はツールを登録
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Execute はツール呼び出しを実行
func (r *Registry) Execute(tc *ToolCall) (string, *FileChange) {
	tool, ok := r.tools[tc.Tool]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", tc.Tool), nil
	}

	output, change, err := tool.Run(tc.Args)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), change
	}

	return output, change
}

// DefaultRegistry はデフォルトのツールレジストリ
var DefaultRegistry = NewRegistry()
