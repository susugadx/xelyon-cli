package mcpapproval

import "strings"

// Mode は MCP tool 実行時の承認ポリシーを表す。
type Mode string

const (
	// ModeConfirm は MCP tool 実行前にユーザー確認を要求する。
	ModeConfirm Mode = "confirm"
	// ModeAuto は MCP tool を確認なしで実行する。
	ModeAuto Mode = "auto"
	// ModeDeny は MCP tool を公開せず、実行も拒否する。
	ModeDeny Mode = "deny"
)

// Normalize は config 由来の文字列を MCP approval mode に正規化する。
// 空文字は未指定として confirm を返す。不正値は confirm と valid=false を返す。
func Normalize(raw string) (mode Mode, valid bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ModeConfirm, true
	}
	switch Mode(value) {
	case ModeConfirm, ModeAuto, ModeDeny:
		return Mode(value), true
	default:
		return ModeConfirm, false
	}
}

// Effective は zero value や不正な mode を安全側の confirm に正規化する。
func Effective(mode Mode) Mode {
	if mode == "" {
		return ModeConfirm
	}
	switch mode {
	case ModeConfirm, ModeAuto, ModeDeny:
		return mode
	default:
		return ModeConfirm
	}
}

// String は mode の config 表現を返す。
func (m Mode) String() string {
	return string(Effective(m))
}
