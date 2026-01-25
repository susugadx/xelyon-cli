package api

import "encoding/json"

// OpenAITool は OpenAI Chat Completions API 用のツール形式
// リクエストの tools[] フィールドに使用
type OpenAITool struct {
	Type     string              `json:"type"` // "function"
	Function *OpenAIToolFunction `json:"function"`
}

// OpenAIToolFunction はツール関数の定義
type OpenAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Strict      bool                   `json:"strict,omitempty"`
}

// OpenAIToolCall はレスポンスの tool_calls フィールド
// ストリーミングでは複数チャンクに分割されて送信される
type OpenAIToolCall struct {
	Index    int                    `json:"index"`    // ストリーミング用インデックス
	ID       string                 `json:"id"`       // ツール呼び出しID
	Type     string                 `json:"type"`     // "function"
	Function OpenAIToolCallFunction `json:"function"` // 関数呼び出し情報
}

// OpenAIToolCallFunction は関数呼び出しの詳細
type OpenAIToolCallFunction struct {
	Name      string `json:"name"`      // ツール名
	Arguments string `json:"arguments"` // 引数（JSON文字列）
}

// ConvertMCPToolToOpenAIFunction はMCPツールをOpenAI Function形式に変換
// MCPのInputSchemaはJSON Schema形式でOpenAIと互換性がある
func ConvertMCPToolToOpenAIFunction(name, description string, inputSchema []byte) OpenAIToolFunction {
	// スキーマが空またはnullの場合
	if len(inputSchema) == 0 || string(inputSchema) == "null" {
		return OpenAIToolFunction{
			Name:        name,
			Description: description,
			Parameters:  nil,
		}
	}

	// MCP InputSchema をそのまま map[string]interface{} として使用
	// OpenAI は JSON Schema 形式をそのまま受け入れる
	var params map[string]interface{}
	if err := json.Unmarshal(inputSchema, &params); err != nil {
		// パースエラー時は空のパラメータで続行
		return OpenAIToolFunction{
			Name:        name,
			Description: description,
			Parameters:  nil,
		}
	}

	return OpenAIToolFunction{
		Name:        name,
		Description: description,
		Parameters:  params,
	}
}
