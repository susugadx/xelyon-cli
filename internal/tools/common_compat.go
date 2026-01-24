package tools

// common_compat.go - 後方互換性のためのエイリアス
// サブパッケージへの移行中に既存コードが動作し続けるようにする

import (
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// 型エイリアス
type ImageData = common.ImageData
type ConfirmAction = common.ConfirmAction
type ConfirmDecision = common.ConfirmDecision
type ConfirmResult = common.ConfirmResult
type ToolSafety = common.ToolSafety
type PatternMatchResult = common.PatternMatchResult

// 定数エイリアス
const (
	ConfirmYes     = common.ConfirmYes
	ConfirmNo      = common.ConfirmNo
	ConfirmComment = common.ConfirmComment
	SafetyHigh     = common.SafetyHigh
	SafetyMedium   = common.SafetyMedium
	SafetyLow      = common.SafetyLow
)

// 関数エイリアス - Backup
var createBackup = common.CreateBackup
var cleanupOldBackups = common.CleanupOldBackups

// 関数エイリアス - Confirm
var confirmWithAutoApproveDecision = common.ConfirmWithAutoApproveDecision
var SetAutoApprove = common.SetAutoApprove
var Confirm = common.Confirm
var ConfirmApproved = common.ConfirmApproved
var ConfirmWithFeedback = common.ConfirmWithFeedback
var ConfirmInteractive = common.ConfirmInteractive
var IsInteractiveModeEnabled = common.IsInteractiveModeEnabled

// 関数エイリアス - Safety
var GetToolSafety = common.GetToolSafety
var IsAutoApprovable = common.IsAutoApprovable
var IsSafeToolAutoApprovable = common.IsSafeToolAutoApprovable
var IsMediumToolAutoApprovable = common.IsMediumToolAutoApprovable
var GetSafetyDescription = common.GetSafetyDescription

// 関数エイリアス - Validation
var ValidatePath = common.ValidatePath
var ValidatePathAllowParent = common.ValidatePathAllowParent

// 関数エイリアス - Diff
var showDiff = common.ShowDiff
var showImprovedDiff = common.ShowImprovedDiff
var showPreview = common.ShowPreview

// 関数エイリアス - Pattern
var FindPatternInLines = common.FindPatternInLines
var DisplayPatternNotFound = common.DisplayPatternNotFound
var DisplayMultipleMatches = common.DisplayMultipleMatches
var DisplayContextAround = common.DisplayContextAround
var DisplayContentToInsert = common.DisplayContentToInsert

// 関数エイリアス - Helpers
var normalizeLeadingWhitespace = common.NormalizeLeadingWhitespace
var findWithNormalizedWhitespace = common.FindWithNormalizedWhitespace
var truncate = common.Truncate

// confirm は旧来の y/n 確認（テスト用にオーバーライド可能）
var confirm = common.SimpleConfirm

// getCurrentTime は現在時刻を返す
var getCurrentTime = common.GetCurrentTime

// min, max は Go 1.21+ の組み込み関数と衝突するため小文字のまま維持
func min(a, b int) int { return common.Min(a, b) }
func max(a, b int) int { return common.Max(a, b) }

// 関数エイリアス - Gitignore
var ensureGitignore = common.EnsureGitignore
var findGitignorePath = common.FindGitignorePath
var findGitRoot = common.FindGitRoot

// 関数エイリアス - Image
var LoadImage = common.LoadImage
var FormatSize = common.FormatSize

// 色エイリアス
var yellow = common.Yellow
var green = common.Green
var red = common.Red
var cyan = common.Cyan
