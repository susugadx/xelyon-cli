package common

import (
	"io"
	"os"
	"path/filepath"
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
	Policy      *config.ExecutionPolicy // nil の場合は Config から自動解決
}

// ToolConfirmContext はツール実行時の判定材料。
// パス・ファイル数・move 先などから always_confirm カテゴリを判定する。
type ToolConfirmContext struct {
	TargetPath    string   // 対象ファイルパス（単一ツール用、空なら不明）
	TargetPaths   []string // 複数対象パス（apply_patch 等）。全パスを workspace 判定する。
	MovePath      string   // 移動先パス（rename/move 系のみ、空なら非 move）
	HasMove       bool     // move/rename を含むか（複数 hunk 対応）
	AffectedFiles int      // 変更対象ファイル数（apply_patch / batch 系）
}

// EffectivePolicy は実行ポリシーを返す。Policy が設定されていれば使い、
// なければ Config.Execution から解決する。
func (o ConfirmOptions) EffectivePolicy() config.ExecutionPolicy {
	if o.Policy != nil {
		return *o.Policy
	}
	if o.Config != nil {
		p := config.ResolveExecutionPolicy(o.Config.Execution)
		return p
	}
	return config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionBalanced)})
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

// ConfirmWithAutoApproveDecisionAndOptions は ToolConfirmContext なしの互換ラッパー。
func ConfirmWithAutoApproveDecisionAndOptions(promptIO ui.PromptIO, options ConfirmOptions, toolName, message string) ConfirmDecision {
	return ConfirmToolAction(promptIO, options, toolName, message, ToolConfirmContext{})
}

// MassChangeThreshold は mass_change と判定するファイル数の閾値。
const MassChangeThreshold = 5

// ConfirmToolAction は実行時設定とツールコンテキストに基づいて確認を行う。
//
// 判定順序:
//  1. always_confirm チェック（mode より優先、full_auto でも効く）
//  2. --auto-approve フラグ（headless 含む）
//  3. execution policy ベースの判定（workspace / trusted / full_auto）
//  4. フォールバック: 旧 tool_confirm 設定
//  5. 通常の確認プロンプト
func ConfirmToolAction(promptIO ui.PromptIO, options ConfirmOptions, toolName, message string, tc ToolConfirmContext) ConfirmDecision {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := NewOutput(promptIO.Out, promptIO.Err)
	policy := options.EffectivePolicy()

	// 1. always_confirm チェック（mode より優先）
	// ツール名ベースのカテゴリ
	if cat := ToolNameToConfirmCategory(toolName); cat != "" && policy.ShouldConfirm(cat) {
		return ConfirmWithIO(promptIO, message)
	}
	// パス/引数ベースのカテゴリ
	for _, cat := range ResolveContextCategories(toolName, tc) {
		if policy.ShouldConfirm(cat) {
			return ConfirmWithIO(promptIO, message)
		}
	}

	// 2. --auto-approve フラグ（headless 含む）
	if IsAutoApprovable(toolName, options.AutoApprove) {
		safety := GetToolSafety(toolName)
		out.Green.Printf("Auto-approved (%s): %s\n", GetSafetyDescription(safety), toolName)
		return ConfirmDecision{Action: ConfirmYes}
	}

	// 3. execution policy ベースの判定
	safety := GetToolSafety(toolName)

	// 3a. SafetyHigh — 全 mode で自動
	if safety == SafetyHigh && policy.AutoApproveRead {
		out.Green.Printf("Auto-approved (Safe read-only): %s\n", toolName)
		return ConfirmDecision{Action: ConfirmYes}
	}

	// 3b. SafetyMedium
	if safety == SafetyMedium {
		if policy.AutoApproveFullAuto {
			out.Green.Printf("Auto-approved (Full auto): %s\n", toolName)
			return ConfirmDecision{Action: ConfirmYes}
		}
		if policy.AutoApproveTrustedWrite && config.TrustedWriteTools[toolName] {
			// trusted: 全対象パスが workspace 内 かつ 通常編集のみ自動
			if AllPathsInsideWorkspace(tc) {
				out.Green.Printf("Auto-approved (Trusted write): %s\n", toolName)
				return ConfirmDecision{Action: ConfirmYes}
			}
		}
	}

	// 3c. SafetyLow — full_auto のみ自動（bash は別経路）
	if safety == SafetyLow && policy.AutoApproveFullAuto {
		out.Green.Printf("Auto-approved (Full auto): %s\n", toolName)
		return ConfirmDecision{Action: ConfirmYes}
	}

	// 4. フォールバック: 旧 tool_confirm 設定
	cfg := options.Config
	if cfg != nil {
		if cfg.ToolConfirm.AutoApproveSafe && safety == SafetyHigh {
			out.Green.Printf("Auto-approved (Safe read-only): %s\n", toolName)
			return ConfirmDecision{Action: ConfirmYes}
		}
		if cfg.ToolConfirm.AutoApproveMedium && safety == SafetyMedium {
			out.Green.Printf("Auto-approved (Medium write): %s\n", toolName)
			return ConfirmDecision{Action: ConfirmYes}
		}
	}

	// 5. 通常の確認プロンプト
	return ConfirmWithIO(promptIO, message)
}

// toolNameToConfirmCategory はツール名から直接マッピングできるカテゴリを返す。
func ToolNameToConfirmCategory(toolName string) config.ConfirmCategory {
	switch toolName {
	case "delete_file":
		return config.ConfirmDeleteFile
	case "git_push", "git_force_push", "git_reset_hard":
		return config.ConfirmBashDestructive
	default:
		return ""
	}
}

// resolveContextCategories はパス・引数・ファイル数から追加のカテゴリを導出する。
func ResolveContextCategories(toolName string, tc ToolConfirmContext) []config.ConfirmCategory {
	var cats []config.ConfirmCategory

	// move_file: MovePath または HasMove が設定されている場合
	if tc.MovePath != "" || tc.HasMove {
		cats = append(cats, config.ConfirmMoveFile)
	}

	// workspace_outside_write: いずれかの対象パスが workspace 外ならカテゴリ追加
	if hasOutsideWorkspacePath(tc) {
		cats = append(cats, config.ConfirmWorkspaceOutside)
	}

	// mass_change: 変更対象ファイル数が閾値以上
	if tc.AffectedFiles >= MassChangeThreshold {
		cats = append(cats, config.ConfirmMassChange)
	}

	// destructive_write: apply_patch は常に destructive_write 候補
	if toolName == "apply_patch" {
		cats = append(cats, config.ConfirmDestructiveWrite)
	}

	return cats
}

// hasOutsideWorkspacePath は ToolConfirmContext 内のいずれかのパスが workspace 外かを判定する。
// 複数パス（TargetPaths）がある場合は全パスを走査する。
func hasOutsideWorkspacePath(tc ToolConfirmContext) bool {
	// 単一パス
	if tc.TargetPath != "" && !isInsideWorkspace(tc.TargetPath) {
		return true
	}
	// MovePath
	if tc.MovePath != "" && !isInsideWorkspace(tc.MovePath) {
		return true
	}
	// 複数パス（apply_patch 等）
	for _, p := range tc.TargetPaths {
		if p != "" && !isInsideWorkspace(p) {
			return true
		}
	}
	return false
}

// isInsideWorkspace は対象パスが CWD（workspace）配下かを判定する。
// target と base の両方を実体パスに解決してから比較し、
// symlink 経由 CWD でも symlink escape でも正しく判定する。
// パスが空の場合は workspace 内として扱う（パス不明時は安全側に倒さない）。
func isInsideWorkspace(path string) bool {
	return isInsideWorkspaceWithOps(path, defaultWorkspacePathOps)
}

type workspacePathOps struct {
	abs          func(string) (string, error)
	getwd        func() (string, error)
	evalSymlinks func(string) (string, error)
	lstat        func(string) (os.FileInfo, error)
}

var defaultWorkspacePathOps = workspacePathOps{
	abs:          filepath.Abs,
	getwd:        os.Getwd,
	evalSymlinks: filepath.EvalSymlinks,
	lstat:        os.Lstat,
}

func isInsideWorkspaceWithOps(path string, ops workspacePathOps) bool {
	if path == "" {
		return true // パス不明 = 判定不能 → workspace 内扱い
	}
	absPath, err := ops.abs(path)
	if err != nil {
		return true
	}
	cwd, err := ops.getwd()
	if err != nil {
		return true
	}
	allowedDir, err := ops.abs(cwd)
	if err != nil {
		return true
	}

	// base（CWD）を実体パスに解決
	realAllowed, err := ops.evalSymlinks(allowedDir)
	if err != nil {
		realAllowed = allowedDir
	}

	// target を実体パスに解決（存在する場合）
	if _, statErr := ops.lstat(absPath); statErr == nil {
		realPath, err := ops.evalSymlinks(absPath)
		if err != nil {
			return false // symlink 解決失敗 → 安全側（workspace 外扱い）
		}
		return isSubPathCheck(realPath, realAllowed)
	}

	// ファイル未存在 → 親ディレクトリを実体解決して判定
	parentDir := filepath.Dir(absPath)
	if _, statErr := ops.lstat(parentDir); statErr == nil {
		realParent, err := ops.evalSymlinks(parentDir)
		if err != nil {
			return false
		}
		return isSubPathCheck(realParent, realAllowed)
	}

	// 親も存在しない → Abs パス同士で比較（フォールバック）
	return isSubPathCheck(absPath, realAllowed)
}

// allPathsInsideWorkspace は ToolConfirmContext 内の全パスが workspace 内かを判定する。
// 1 つでも workspace 外のパスがあれば false。パスが全て空なら true（不明 = workspace 内扱い）。
func AllPathsInsideWorkspace(tc ToolConfirmContext) bool {
	if tc.TargetPath != "" && !isInsideWorkspace(tc.TargetPath) {
		return false
	}
	if tc.MovePath != "" && !isInsideWorkspace(tc.MovePath) {
		return false
	}
	for _, p := range tc.TargetPaths {
		if p != "" && !isInsideWorkspace(p) {
			return false
		}
	}
	return true
}

// isSubPathCheck は target が base ディレクトリ配下かどうか判定する。
func isSubPathCheck(target, base string) bool {
	if target == base {
		return true
	}
	return strings.HasPrefix(target, base+string(filepath.Separator))
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
