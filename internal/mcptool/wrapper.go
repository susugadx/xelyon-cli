package mcptool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const defaultMCPToolCallTimeout = 600 * time.Second

var (
	// ErrApprovalDenied は MCP approval policy により実行が拒否されたことを表す。
	ErrApprovalDenied = errors.New("MCP tool execution denied by approval policy")
	// ErrApprovalRequired は headless 実行で MCP tool の承認確認が必要だったことを表す。
	ErrApprovalRequired = errors.New("approval_required")
)

// Definition は MCP server から取得した tool metadata を tools.Registry 用に表す。
type Definition struct {
	ServerName  string
	Name        string
	Description string
	InputSchema json.RawMessage
	CallTimeout time.Duration
	Approval    mcpapproval.Mode
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
		if mcpapproval.Effective(tool.Approval) == mcpapproval.ModeDeny {
			continue
		}
		wrapper := NewWrapper(WrapperOptions{
			Caller:      caller,
			ServerName:  tool.ServerName,
			ToolName:    tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			CallTimeout: tool.CallTimeout,
			Approval:    tool.Approval,
		})
		if registry.HasTool(wrapper.Name()) {
			continue
		}
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
	approval    mcpapproval.Mode
}

// WrapperOptions は Wrapper 作成時の依存と metadata をまとめる。
type WrapperOptions struct {
	Caller      ToolCaller
	ServerName  string
	ToolName    string
	Description string
	InputSchema json.RawMessage
	CallTimeout time.Duration
	Approval    mcpapproval.Mode
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
		approval:    mcpapproval.Effective(opts.Approval),
	}
}

// Name はツール名を返す（mcp_<server>_<tool> 形式、特殊文字を置換）
func (w *Wrapper) Name() string {
	return mcpnames.ExportedToolName(w.serverName, w.toolName)
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
	return api.MCPInputSchemaParameters(w.inputSchema)
}

// Run はツールを実行
func (w *Wrapper) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	out := execCtx.Output()
	toolName := w.Name()

	if w.approvalMode() == mcpapproval.ModeDeny {
		err := fmt.Errorf("%w: %s", ErrApprovalDenied, toolName)
		return "Error: " + err.Error(), nil, err
	}

	// 引数バリデーション（簡易版）
	if err := w.validateArgs(out.StdoutWriter(), args); err != nil {
		return fmt.Sprintf("Validation Error: %v", err), nil, err
	}

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

	switch w.approvalMode() {
	case mcpapproval.ModeConfirm:
		if execCtx.IsHeadless() {
			err := fmt.Errorf("%w: MCP tool execution requires approval by MCP approval policy: %s", ErrApprovalRequired, toolName)
			return "Error: " + err.Error(), nil, err
		}
		decision := common.ConfirmWithIO(execCtx.PromptIO(), message)
		switch decision.Action {
		case common.ConfirmNo:
			return "User rejected MCP tool execution", nil, nil
		case common.ConfirmComment:
			feedback := "User provided feedback: " + decision.Comment
			return feedback, nil, nil
		}
	case mcpapproval.ModeAuto:
		out.Green.Printf("Auto-approved (MCP config): %s\n", toolName)
	}

	// スキーマに基づいて型変換（string → number/integer/boolean）
	anyArgs := w.convertArgsWithSchema(args)

	callTimeout := w.callTimeoutDuration()
	parentCtx := execCtx.EffectiveContext()
	ctx, cancel := context.WithTimeout(parentCtx, callTimeout)
	defer cancel()

	result, err := w.manager.CallTool(ctx, w.serverName, w.toolName, anyArgs)
	if err != nil {
		switch {
		case errors.Is(parentCtx.Err(), context.Canceled):
			return "Error: Tool execution canceled by request context",
				nil,
				fmt.Errorf("tool execution canceled by request context: %w", parentCtx.Err())
		case errors.Is(parentCtx.Err(), context.DeadlineExceeded):
			return "Error: Tool execution stopped by request deadline",
				nil,
				fmt.Errorf("tool execution stopped by request deadline: %w", parentCtx.Err())
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return fmt.Sprintf("Error: Tool execution timed out after %s", formatTimeoutDuration(callTimeout)),
				nil,
				fmt.Errorf("tool execution timed out: %w", ctx.Err())
		case errors.Is(ctx.Err(), context.Canceled):
			return "Error: Tool execution canceled",
				nil,
				fmt.Errorf("tool execution canceled: %w", ctx.Err())
		}
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	// 結果をフォーマット
	formattedResult := w.formatResult(result)
	return formattedResult, nil, nil
}

func (w *Wrapper) approvalMode() mcpapproval.Mode {
	return mcpapproval.Effective(w.approval)
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
					case "array", "object":
						if structured, err := parseStructuredArg(propType, k, v); err == nil {
							anyArgs[k] = structured
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

	if properties, ok := schema["properties"].(map[string]any); ok && properties != nil {
		warnInvalidPropertySchemas(out, w.toolName, properties)

		if err := validateStructuredArgs(properties, args); err != nil {
			return err
		}
	}

	return nil
}

func warnInvalidPropertySchemas(out io.Writer, toolName string, properties map[string]any) {
	for propName, propInfo := range properties {
		propMap, ok := propInfo.(map[string]any)
		if !ok || propMap == nil {
			fmt.Fprintf(out, "⚠️  Warning: Invalid property schema for %s in tool %s\n", propName, toolName)
		}
	}
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

func validateStructuredArgs(properties map[string]any, args map[string]string) error {
	for argName, rawValue := range args {
		propMap, ok := properties[argName].(map[string]any)
		if !ok || propMap == nil {
			continue
		}
		propType, ok := propMap["type"].(string)
		if !ok || (propType != "array" && propType != "object") {
			continue
		}
		if _, err := parseStructuredArg(propType, argName, rawValue); err != nil {
			return err
		}
	}
	return nil
}

func parseStructuredArg(propType, argName, rawValue string) (any, error) {
	var decoded any
	if err := json.Unmarshal([]byte(rawValue), &decoded); err != nil {
		return nil, fmt.Errorf("argument '%s' must be valid JSON %s: %w", argName, propType, err)
	}

	switch propType {
	case "array":
		arrayValue, ok := decoded.([]any)
		if !ok {
			return nil, fmt.Errorf("argument '%s' must be a JSON array", argName)
		}
		return arrayValue, nil
	case "object":
		objectValue, ok := decoded.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("argument '%s' must be a JSON object", argName)
		}
		return objectValue, nil
	default:
		return rawValue, nil
	}
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
