package tools

// confirm.go - 共通の確認API（y/n/c + 複数行コメント + /paste対応）

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

// Confirm asks user for confirmation and optionally captures feedback.
// If interactive confirmation is enabled, it supports y/n/c and multi-line comments.
// Otherwise it falls back to legacy y/n confirm.
func Confirm(message string) ConfirmDecision {
	if !IsInteractiveModeEnabled() {
		approved := confirm(message)
		if approved {
			return ConfirmDecision{Action: ConfirmYes}
		}
		return ConfirmDecision{Action: ConfirmNo}
	}

	res := ConfirmInteractive(message)
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
