package search

import "strings"

type patternInputKind int

const (
	patternInputUnset patternInputKind = iota
	patternInputDelimited
	patternInputLiteral
)

// PatternInput は SearchOptions.Pattern の分割と検索 lane の解釈方法を表す。
type PatternInput struct {
	kind  patternInputKind
	value string
}

// NewDelimitedPatternInput は既存互換の comma-separated pattern 入力を作る。
func NewDelimitedPatternInput(pattern string) PatternInput {
	return PatternInput{kind: patternInputDelimited, value: pattern}
}

// NewLiteralPatternInput は単一の literal pattern 入力を作る。
func NewLiteralPatternInput(pattern string) PatternInput {
	return PatternInput{kind: patternInputLiteral, value: pattern}
}

func (input PatternInput) isSet() bool {
	return input.kind != patternInputUnset
}

func (input PatternInput) isLiteral() bool {
	return input.kind == patternInputLiteral
}

func (input PatternInput) pattern() string {
	return input.value
}

func applyPatternInput(opts SearchOptions) SearchOptions {
	if !opts.PatternInput.isSet() {
		return opts
	}

	opts.Pattern = opts.PatternInput.pattern()
	if opts.PatternInput.isLiteral() {
		opts.Mode = string(SearchModeLiteral)
		opts.IsRegex = false
		opts.LegacyIsRegexSet = false
		opts.Intent = ""
	}
	return opts
}

func effectiveSearchPatternList(opts SearchOptions) []string {
	if opts.PatternInput.isLiteral() {
		pattern := strings.TrimSpace(opts.Pattern)
		if pattern == "" {
			return nil
		}
		return []string{pattern}
	}
	return splitPatterns(opts.Pattern)
}
