package ui

import "context"

// PromptKind は UI runtime に依頼できる prompt の種類を表す。
type PromptKind string

const (
	PromptKindConfirm      PromptKind = "confirm"
	PromptKindSingleChoice PromptKind = "single_choice"
	PromptKindMultiChoice  PromptKind = "multi_choice"
	PromptKindText         PromptKind = "text"
)

// PromptAction は confirm prompt の回答種別を表す。
type PromptAction string

const (
	PromptActionYes     PromptAction = "yes"
	PromptActionNo      PromptAction = "no"
	PromptActionComment PromptAction = "comment"
)

// PromptConfirmSubmitPolicy は confirm prompt で Enter submit を許可する条件を表す。
type PromptConfirmSubmitPolicy string

const (
	// PromptConfirmSubmitSelected は現在選択されている confirm option を Enter で送信する。
	PromptConfirmSubmitSelected PromptConfirmSubmitPolicy = ""
	// PromptConfirmSubmitExplicit は明示的な shortcut または選択操作後の Enter だけを送信する。
	PromptConfirmSubmitExplicit PromptConfirmSubmitPolicy = "explicit"
)

// PromptOption は single/multi choice prompt の選択肢を表す。
type PromptOption struct {
	Label       string
	Description string
	Value       string // confirm では "yes" / "no" / "comment" の action 値
}

// PromptRequest は TUI などの UI 実装へ渡す prompt 契約。
type PromptRequest struct {
	Kind                PromptKind
	Title               string
	Message             string
	Options             []PromptOption // confirm では任意の表示ラベル/順序を指定できる
	DefaultValue        string
	DefaultValues       []string
	AllowComment        bool
	ConfirmSubmitPolicy PromptConfirmSubmitPolicy
	Placeholder         string
}

// PromptResponse は prompt の正規化済み回答を表す。
type PromptResponse struct {
	Action    PromptAction
	Value     string
	Values    []string
	Text      string
	Cancelled bool
}

// Prompter は runtime に紐づく対話 prompt 実装の境界。
type Prompter interface {
	Prompt(ctx context.Context, req PromptRequest) (PromptResponse, error)
}
