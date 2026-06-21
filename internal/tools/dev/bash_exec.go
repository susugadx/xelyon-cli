package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type bashExecutionRequest struct {
	promptIO uiruntime.PromptIO
	out      common.Output
	cfg      *config.Config
	command  string
}

func newBashExecutionRequest(promptIO uiruntime.PromptIO, cfg *config.Config, command string) (bashExecutionRequest, string, bool) {
	normalizedPromptIO := uiruntime.NormalizePromptIO(promptIO)
	out := common.NewOutput(normalizedPromptIO.Out, normalizedPromptIO.Err)
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if command == "" {
		return bashExecutionRequest{}, "Error: command is empty", false
	}
	return bashExecutionRequest{
		promptIO: normalizedPromptIO,
		out:      out,
		cfg:      cfg,
		command:  command,
	}, "", true
}

func configureBashCommandDirectory(out common.Output, cmd *exec.Cmd) {
	cwd, err := os.Getwd()
	if err == nil {
		cmd.Dir = cwd
		return
	}
	out.Yellow.Printf("Warning: Could not get current directory: %v\n", err)
	cmd.Dir = "."
}

func runBashCommand(out common.Output, cmd *exec.Cmd, streamOutput bool) (string, error) {
	if streamOutput {
		return executeBashWithStreaming(out, cmd)
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runBashCommandWithContext(ctx context.Context, out common.Output, cmd *exec.Cmd, streamOutput bool) (string, error) {
	if streamOutput {
		return executeBashWithStreamingAndContext(ctx, out, cmd)
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func formatBashCommandResult(result string, cmdErr error) string {
	if cmdErr != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", cmdErr, result)
	}
	if strings.TrimSpace(result) == "" {
		result = "(no output)"
	}
	return TruncateWithFile(result)
}

func formatInterruptedBashResult(out common.Output, cmd *exec.Cmd, result string) string {
	out.Yellow.Println("\n⚠️  Command interrupted. Partial output returned.")
	killProcessGroup(cmd)
	return fmt.Sprintf("Command interrupted.\nPartial output:\n%s", result)
}

// AutoApproveReason は bash 自動承認時の理由ラベル。
type AutoApproveReason string

const (
	approveNone             AutoApproveReason = ""
	approveVerificationBash AutoApproveReason = "Safe bash"
	approveSafeShellCmd     AutoApproveReason = "Safe bash"
	approveSafeBuiltin      AutoApproveReason = "Safe bash"
)

// ExecuteBash はシェルコマンドを実行する。
func ExecuteBash(command string) string {
	return ExecuteBashWithOutput(common.DefaultOutput(), command)
}

// ExecuteBashWithOutput は出力先を指定してシェルコマンドを実行する。
func ExecuteBashWithOutput(out common.Output, command string) string {
	return ExecuteBashWithPromptIOAndConfig(uiruntime.NewPromptIO(nil, out.StdoutWriter(), out.StderrWriter(), nil), config.DefaultConfig(), command)
}

// ExecuteBashWithPromptIOAndConfig は設定と入出力を指定してシェルコマンドを実行する。
func ExecuteBashWithPromptIOAndConfig(promptIO uiruntime.PromptIO, cfg *config.Config, command string) string {
	req, msg, ok := newBashExecutionRequest(promptIO, cfg, command)
	if !ok {
		return msg
	}

	reason, msg, ok := checkAndConfirmBash(req.promptIO, req.cfg, req.command)
	if !ok {
		return msg
	}

	printRunning(req.out, req.command, reason)
	cmd := exec.Command("bash", "-c", req.command)
	configureBashCommandDirectory(req.out, cmd)

	result, cmdErr := runBashCommand(req.out, cmd, req.cfg.Streaming.StreamBashOutput)
	return formatBashCommandResult(result, cmdErr)
}

// ExecuteBashWithContext は Context 対応でシェルコマンドを実行する。
// コンテキストがキャンセルされた場合、部分結果を返す。
func ExecuteBashWithContext(ctx context.Context, command string) string {
	return ExecuteBashWithContextAndOutput(ctx, common.DefaultOutput(), command)
}

// ExecuteBashWithContextAndOutput は Context 対応で出力先を指定してシェルコマンドを実行する。
func ExecuteBashWithContextAndOutput(ctx context.Context, out common.Output, command string) string {
	return ExecuteBashWithContextAndPromptIOAndConfig(ctx, uiruntime.NewPromptIO(nil, out.StdoutWriter(), out.StderrWriter(), nil), config.DefaultConfig(), command)
}

// ExecuteBashWithContextAndPromptIOAndConfig は Context・設定・入出力を指定してシェルコマンドを実行する。
func ExecuteBashWithContextAndPromptIOAndConfig(ctx context.Context, promptIO uiruntime.PromptIO, cfg *config.Config, command string) string {
	req, msg, ok := newBashExecutionRequest(promptIO, cfg, command)
	if !ok {
		return msg
	}

	reason, msg, ok := checkAndConfirmBash(req.promptIO, req.cfg, req.command)
	if !ok {
		return msg
	}

	printRunning(req.out, req.command, reason)
	cmd := exec.CommandContext(ctx, "bash", "-c", req.command)
	configureBashCommandDirectory(req.out, cmd)
	setSysProcAttr(cmd)

	result, cmdErr := runBashCommandWithContext(ctx, req.out, cmd, req.cfg.Streaming.StreamBashOutput)
	if ctx.Err() != nil {
		return formatInterruptedBashResult(req.out, cmd, result)
	}
	return formatBashCommandResult(result, cmdErr)
}

// printRunning は bash 実行開始メッセージを表示する。
// auto-approved の場合は理由を統合して 1 行で表示。
// ユーザー確認済み（reason が空）の場合は Running のみ。
func printRunning(out common.Output, command string, reason AutoApproveReason) {
	if reason != "" {
		out.Green.Printf("▶ Running (auto-approved: %s): %s\n", reason, command)
	} else {
		out.Green.Printf("▶ Running: %s\n", command)
	}
}
