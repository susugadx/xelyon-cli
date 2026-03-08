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

// 関数エイリアス - Confirm
var SetAutoApprove = common.SetAutoApprove
var Confirm = common.Confirm
var ConfirmApproved = common.ConfirmApproved
var ConfirmWithFeedback = common.ConfirmWithFeedback
var ConfirmInteractive = common.ConfirmInteractive
var ConfirmInteractiveWithIO = common.ConfirmInteractiveWithIO
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

// 関数エイリアス - Pattern
var FindPatternInLines = common.FindPatternInLines
var DisplayPatternNotFound = common.DisplayPatternNotFound
var DisplayMultipleMatches = common.DisplayMultipleMatches
var DisplayContextAround = common.DisplayContextAround
var DisplayContentToInsert = common.DisplayContentToInsert

// 関数エイリアス - Helpers
var truncate = common.Truncate

// getCurrentTime は現在時刻を返す
var getCurrentTime = common.GetCurrentTime

// 関数エイリアス - Image
var LoadImage = common.LoadImage
var FormatSize = common.FormatSize

// 色エイリアス
var cyan = common.Cyan
var dim = common.Dim
