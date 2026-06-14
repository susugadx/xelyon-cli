package mcptool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const defaultMCPToolCallTimeout = 30 * time.Second

// Definition は MCP server から取得した tool metadata を tools.Registry 用に表す。
type Definition struct {
	ServerName  string
	Name        string
	Description string
	InputSchema json.RawMessage
}

// ToolCaller は integration 層が必要とする最小契約。
// serverName/toolName/args を指定して実行し、テキスト結果またはエラーを返す。
type ToolCaller interface {
	CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (string, error)
}

// RegisterToRegistry は MCP tool metadata を tools.Registry に登録する。
func RegisterToRegistry(registry *tools.Registry, caller ToolCaller, definitions []Definition) {
	for _, definition := range definitions {
		tool := definition
		wrapper := NewWrapper(WrapperOptions{
			Caller:      caller,
			ServerName:  tool.ServerName,
			ToolName:    tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})

		registry.Register(wrapper)
	}
}

// Wrapper はMCPツールをTool interfaceにラップ
type Wrapper struct {
	manager     ToolCaller
	serverName  string
	toolName    string
	desc        string
	inputSchema json.RawMessage // JSONスキーマ情報
	callTimeout time.Duration
}

// WrapperOptions は Wrapper 作成時の依存と metadata をまとめる。
type WrapperOptions struct {
	Caller      ToolCaller
	ServerName  string
	ToolName    string
	Description string
	InputSchema json.RawMessage
	CallTimeout time.Duration
}

// NewWrapper は MCP tool metadata から tools.Tool 実装を作る。
func NewWrapper(opts WrapperOptions) *Wrapper {
	return &Wrapper{
		manager:     opts.Caller,
		serverName:  opts.ServerName,
		toolName:    opts.ToolName,
		desc:        opts.Description,
		inputSchema: opts.InputSchema,
		callTimeout: opts.CallTimeout,
	}
}

// Name はツール名を返す（mcp_<server>_<tool> 形式、特殊文字を置換）
func (w *Wrapper) Name() string {
	// 特殊文字をアンダースコアに置換
	safeServer := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, w.serverName)

	safeTool := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, w.toolName)

	return fmt.Sprintf("mcp_%s_%s", safeServer, safeTool)
}

// Description はツールの説明を返す
func (w *Wrapper) Description() string {
	if w.desc != "" {
		return w.desc
	}
	return fmt.Sprintf("MCP tool: %s from server %s", w.toolName, w.serverName)
}

// Parameters はツールのパラメータ定義を返す
func (w *Wrapper) Parameters() map[string]interface{} {
	// inputSchemaをそのままmap[string]interface{}に変換
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		return map[string]interface{}{
			"type":                 "object",
			"properties":           map[string]interface{}{},
			"additionalProperties": false,
		}
	}

	var params map[string]interface{}
	if err := json.Unmarshal(w.inputSchema, &params); err != nil {
		return map[string]interface{}{
			"type":                 "object",
			"properties":           map[string]interface{}{},
			"additionalProperties": false,
		}
	}
	return params
}

// Run はツールを実行
func (w *Wrapper) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	out := execCtx.Output()

	// 引数バリデーション（簡易版）
	if err := w.validateArgs(out.StdoutWriter(), args); err != nil {
		return fmt.Sprintf("Validation Error: %v", err), nil, err
	}

	// MCP tool は動的登録され SafetyLow として扱われる。
	// 通常 mode では確認し、full_auto や --auto-approve では実行ポリシーに従って自動承認される。
	toolName := w.Name()
	message := fmt.Sprintf("Execute MCP tool: %s (server: %s)", w.toolName, w.serverName)

	// 引数がある場合は表示
	if len(args) > 0 {
		argsDisplay := make([]string, 0, len(args))
		for k, v := range args {
			// 値が長い場合は省略
			displayVal := v
			if len(displayVal) > 50 {
				displayVal = displayVal[:47] + "..."
			}
			argsDisplay = append(argsDisplay, fmt.Sprintf("%s=%q", k, displayVal))
		}
		message = fmt.Sprintf("Execute MCP tool: %s\n  Server: %s\n  Args: %s",
			w.toolName, w.serverName, strings.Join(argsDisplay, ", "))
	}

	decision := common.ConfirmWithAutoApproveDecisionAndOptions(execCtx.PromptIO(), execCtx.ConfirmOptions(), toolName, message)
	switch decision.Action {
	case common.ConfirmNo:
		return "User rejected MCP tool execution", nil, nil
	case common.ConfirmComment:
		// コメントがある場合はフィードバックとして返す
		feedback := "User provided feedback: " + decision.Comment
		return feedback, nil, nil
	}

	// スキーマに基づいて型変換（string → number/integer/boolean）
	anyArgs := w.convertArgsWithSchema(args)

	callTimeout := w.callTimeoutDuration()
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	result, err := w.manager.CallTool(ctx, w.serverName, w.toolName, anyArgs)
	if err != nil {
		// タイムアウトエラーの場合
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("Error: Tool execution timed out after %s", formatTimeoutDuration(callTimeout)),
				nil,
				fmt.Errorf("tool execution timed out")
		}
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	// 結果をフォーマット
	formattedResult := w.formatResult(result)
	return formattedResult, nil, nil
}

// convertArgsWithSchema はスキーマに基づいて引数の型を変換する
func (w *Wrapper) convertArgsWithSchema(args map[string]string) map[string]any {
	anyArgs := make(map[string]any)

	// スキーマが空の場合は文字列のまま返す
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		for k, v := range args {
			anyArgs[k] = v
		}
		return anyArgs
	}

	// JSONスキーマをパース
	var schema map[string]any
	if err := json.Unmarshal(w.inputSchema, &schema); err != nil {
		// パースエラーの場合は文字列のまま返す
		for k, v := range args {
			anyArgs[k] = v
		}
		return anyArgs
	}

	// プロパティ情報を取得
	properties, _ := schema["properties"].(map[string]any)

	for k, v := range args {
		converted := false
		// スキーマに型情報があれば変換
		if properties != nil {
			if propInfo, ok := properties[k].(map[string]any); ok {
				if propType, ok := propInfo["type"].(string); ok {
					switch propType {
					case "integer":
						if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
							anyArgs[k] = intVal
							converted = true
						}
					case "number":
						if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
							anyArgs[k] = floatVal
							converted = true
						}
					case "boolean":
						if boolVal, err := strconv.ParseBool(v); err == nil {
							anyArgs[k] = boolVal
							converted = true
						}
					}
				}
			}
		}
		// 変換できない場合は文字列のまま
		if !converted {
			anyArgs[k] = v
		}
	}

	return anyArgs
}

// ConvertArgsWithSchema は schema に基づく引数変換を実行する。
func (w *Wrapper) ConvertArgsWithSchema(args map[string]string) map[string]any {
	return w.convertArgsWithSchema(args)
}

func (w *Wrapper) callTimeoutDuration() time.Duration {
	if w.callTimeout > 0 {
		return w.callTimeout
	}
	return defaultMCPToolCallTimeout
}

func formatTimeoutDuration(d time.Duration) string {
	if d > 0 && d%time.Second == 0 {
		return fmt.Sprintf("%d seconds", int(d/time.Second))
	}
	return d.String()
}

// validateArgs は引数を検証する（簡易版）
func (w *Wrapper) validateArgs(out io.Writer, args map[string]string) error {
	if out == nil {
		out = io.Discard
	}

	// 空のスキーマの場合はスキップ
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		return nil
	}

	// JSONスキーマをパース（簡易実装）
	var schema map[string]any
	if err := json.Unmarshal(w.inputSchema, &schema); err != nil {
		// パースエラーは警告のみ
		fmt.Fprintf(out, "⚠️  Failed to parse input schema for tool %s: %v\n", w.toolName, err)
		return nil
	}

	if err := validateTopLevelRequiredArgs(out, w.toolName, schema, args); err != nil {
		return err
	}

	// 旧互換: properties.<name>.required=true も必須パラメータとして扱う。
	if properties, ok := schema["properties"].(map[string]any); ok && properties != nil {
		for propName, propInfo := range properties {
			propMap, ok := propInfo.(map[string]any)
			if !ok || propMap == nil {
				// プロパティスキーマが不正な場合は警告してスキップ
				fmt.Fprintf(out, "⚠️  Warning: Invalid property schema for %s in tool %s\n", propName, w.toolName)
				continue
			}
			if required, ok := propMap["required"].(bool); ok && required {
				if _, hasArg := args[propName]; !hasArg {
					return fmt.Errorf("required argument '%s' is missing", propName)
				}
			}
		}
	}

	return nil
}

func validateTopLevelRequiredArgs(out io.Writer, toolName string, schema map[string]any, args map[string]string) error {
	requiredRaw, ok := schema["required"]
	if !ok {
		return nil
	}

	requiredList, ok := requiredRaw.([]any)
	if !ok {
		fmt.Fprintf(out, "⚠️  Warning: Invalid required schema for tool %s\n", toolName)
		return nil
	}

	for _, requiredArg := range requiredList {
		argName, ok := requiredArg.(string)
		if !ok || argName == "" {
			fmt.Fprintf(out, "⚠️  Warning: Invalid required argument entry for tool %s\n", toolName)
			continue
		}
		if _, hasArg := args[argName]; !hasArg {
			return fmt.Errorf("required argument '%s' is missing", argName)
		}
	}

	return nil
}

// ValidateArgs は MCP tool 実行前の簡易 argument validation を行う。
func (w *Wrapper) ValidateArgs(out io.Writer, args map[string]string) error {
	return w.validateArgs(out, args)
}

// formatResult は結果をフォーマットする
// NOTE: 出力の切り詰めはtoken_guard.goで一元管理するため、ここでは行わない
func (w *Wrapper) formatResult(result string) string {
	// 結果が空の場合
	if result == "" {
		return "Tool executed successfully (no output)"
	}

	return result
}

// FormatResult は MCP tool のテキスト結果を表示用に整える。
func (w *Wrapper) FormatResult(result string) string {
	return w.formatResult(result)
}
