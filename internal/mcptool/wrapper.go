package mcptool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"strings"
	"time"
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
