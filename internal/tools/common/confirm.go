package common

import (
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ConfirmAction is the normalized action returned by confirmation prompts.
type ConfirmAction string

const (
	ConfirmYes     ConfirmAction = "yes"
	ConfirmNo      ConfirmAction = "no"
	ConfirmComment ConfirmAction = "comment"
)

// ConfirmDecision is the unified result for all confirmation prompts.
// Comment/Image are only set when Action==ConfirmComment.
type ConfirmDecision struct {
	Action  ConfirmAction
	Comment string
	Image   *ImageData
}

// ConfirmOptions は確認 UI に必要な実行時設定を束ねる。
type ConfirmOptions struct {
	AutoApprove bool
	Config      *config.Config
}

// SimpleConfirm はユーザーに確認を求める（テスト用にグローバル変数として定義）
// 空入力は無視してリトライする（AI実行中のEnter押下対策）
// ただしEOF時はfalseを返して終了する
// NOTE: テスト時は setupTestConfirm() でモックされる
var SimpleConfirmWithIO = func(promptIO ui.PromptIO, message string) bool {
	promptIO = ui.NormalizePromptIO(promptIO)
	reader := promptIO.BufioReader()
	out := NewOutput(promptIO.Out, promptIO.Err)

	for {
		out.Yellow.Printf("%s (y/n): ", message)

		response, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && response != "" {
				response = ui.StripBracketedPaste(response)
				response = strings.ToLower(strings.TrimSpace(response))
				return response == "y" || response == "yes" || response == "ｙ" || response == "はい"
			}
			// EOF または読み取りエラー時は終了
			return false
		}
		response = ui.StripBracketedPaste(response)
		response = strings.ToLower(strings.TrimSpace(response))

		// 空入力は無視してリトライ
		if response == "" {
			continue
		}

		return response == "y" || response == "yes" || response == "ｙ" || response == "はい"
	}
}

var SimpleConfirm = func(message string) bool {
	return SimpleConfirmWithIO(ui.DefaultPromptIO(), message)
}

// Confirm asks user for confirmation and optionally captures feedback.
// If interactive confirmation is enabled, it supports y/n/c and multi-line comments.
// Otherwise it falls back to legacy y/n confirm.
func Confirm(message string) ConfirmDecision {
	return ConfirmWithIO(ui.DefaultPromptIO(), message)
}

// ConfirmWithIO は入出力先を指定して確認を行う。
func ConfirmWithIO(promptIO ui.PromptIO, message string) ConfirmDecision {
	if !IsInteractiveModeEnabled() {
		approved := SimpleConfirmWithIO(promptIO, message)
		if approved {
			return ConfirmDecision{Action: ConfirmYes}
		}
		return ConfirmDecision{Action: ConfirmNo}
	}

	res := ConfirmInteractiveWithIO(promptIO, message)
	switch res.Action {
	case "yes":
		return ConfirmDecision{Action: ConfirmYes}
	case "comment":
		return ConfirmDecision{Action: ConfirmComment, Comment: res.Comment, Image: res.Image}
	default:
		return ConfirmDecision{Action: ConfirmNo}
	}
}

// ConfirmApproved is a compatibility helper that preserves the old bool-based API.
// NOTE: This drops comment/image information.
func ConfirmApproved(message string) bool {
	return Confirm(message).Action == ConfirmYes
}

// ConfirmWithAutoApproveDecisionAndOptions は実行時設定を指定して確認を行う。
func ConfirmWithAutoApproveDecisionAndOptions(promptIO ui.PromptIO, options ConfirmOptions, toolName, message string) ConfirmDecision {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := NewOutput(promptIO.Out, promptIO.Err)
	cfg := options.Config

	// --auto-approve が有効 かつ ツールが自動承認可能な場合
	if IsAutoApprovable(toolName, options.AutoApprove) {
		safety := GetToolSafety(toolName)
		out.Green.Printf("Auto-approved (%s): %s\n", GetSafetyDescription(safety), toolName)
		return ConfirmDecision{Action: ConfirmYes}
	}

	// SafetyHigh ツールの自動承認（設定で有効な場合）
	if cfg != nil && cfg.ToolConfirm.AutoApproveSafe && IsSafeToolAutoApprovable(toolName) {
		out.Green.Printf("Auto-approved (Safe read-only): %s\n", toolName)
		return ConfirmDecision{Action: ConfirmYes}
	}

	// SafetyMedium ツールの自動承認（設定で有効な場合）
	if cfg != nil && cfg.ToolConfirm.AutoApproveMedium && IsMediumToolAutoApprovable(toolName) {
		out.Green.Printf("Auto-approved (Medium write): %s\n", toolName)
		return ConfirmDecision{Action: ConfirmYes}
	}

	// それ以外は通常の確認プロンプト（対話モード時は y/n/c）
	return ConfirmWithIO(promptIO, message)
}

// ConfirmWithFeedback は対話的確認を行う（互換ラッパー）
// interactiveMode=true の場合は y/n/c、false の場合は従来の y/n
//
// NOTE: 新コードでは Confirm() / ConfirmDecision の利用を推奨。
func ConfirmWithFeedback(message string) (approved bool, comment string, image *ImageData) {
	dec := Confirm(message)
	switch dec.Action {
	case ConfirmYes:
		return true, "", nil
	case ConfirmComment:
		return false, dec.Comment, dec.Image
	default:
		return false, "", nil
	}
}
