// Package ui は質問表示 UI を提供する
package uiruntime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

// Questionnaire はユーザーへの質問を表示・回答収集する
type Questionnaire struct {
	Question     string   // 質問文
	QuestionType string   // single_choice, multi_choice, free_text
	Options      []string // 選択肢（choice系の場合）
	Default      string   // デフォルト値
}

// QuestionnaireAnswer は質問への回答
type QuestionnaireAnswer struct {
	Value      string   // 単一回答
	Values     []string // 複数回答（multi_choice用）
	IsMultiple bool     // 複数回答かどうか
}

// Ask は質問を表示し、回答を収集
func (q *Questionnaire) Ask() (*QuestionnaireAnswer, error) {
	return q.AskWithIO(DefaultPromptIO())
}

// AskWithIO は入出力先を指定して質問を表示し、回答を収集する。
func (q *Questionnaire) AskWithIO(promptIO PromptIO) (*QuestionnaireAnswer, error) {
	promptIO = normalizePromptIO(promptIO)
	if prompter := promptIO.Prompter(); prompter != nil {
		return q.askWithPrompter(promptIO, prompter)
	}
	stopSpinnerForPromptIO(promptIO)
	_, _ = fmt.Fprint(promptIO.Out, "\033[?25h") // カーソル表示

	switch q.QuestionType {
	case "single_choice":
		return q.askSingleChoice(promptIO)
	case "multi_choice":
		return q.askMultiChoice(promptIO)
	case "free_text":
		return q.askFreeText(promptIO)
	default:
		return nil, fmt.Errorf("unknown question type: %s", q.QuestionType)
	}
}

func (q *Questionnaire) askWithPrompter(promptIO PromptIO, prompter uiprompt.Prompter) (*QuestionnaireAnswer, error) {
	options := make([]uiprompt.PromptOption, 0, len(q.Options))
	for _, option := range q.Options {
		options = append(options, uiprompt.PromptOption{
			Label: option,
			Value: option,
		})
	}

	req := uiprompt.PromptRequest{
		Message:      q.Question,
		Options:      options,
		DefaultValue: q.Default,
	}
	switch q.QuestionType {
	case "single_choice":
		req.Kind = uiprompt.PromptKindSingleChoice
	case "multi_choice":
		req.Kind = uiprompt.PromptKindMultiChoice
	case "free_text":
		req.Kind = uiprompt.PromptKindText
	default:
		return nil, fmt.Errorf("unknown question type: %s", q.QuestionType)
	}

	resp, err := prompter.Prompt(promptIO.PromptContext(), req)
	if err != nil {
		return nil, err
	}
	if resp.Cancelled {
		return nil, fmt.Errorf("question cancelled")
	}

	switch q.QuestionType {
	case "single_choice":
		return &QuestionnaireAnswer{Value: resp.Value}, nil
	case "multi_choice":
		return &QuestionnaireAnswer{Values: resp.Values, IsMultiple: true}, nil
	case "free_text":
		text := resp.Text
		if text == "" && q.Default != "" {
			text = q.Default
		}
		return &QuestionnaireAnswer{Value: text}, nil
	default:
		return nil, fmt.Errorf("unknown question type: %s", q.QuestionType)
	}
}

// askSingleChoice は単一選択質問を処理
func (q *Questionnaire) askSingleChoice(promptIO PromptIO) (*QuestionnaireAnswer, error) {
	_, _ = fmt.Fprintf(promptIO.Out, "\n%s?%s %s\n\n", colorCyan, colorReset, q.Question)

	defaultIdx := -1
	for i, opt := range q.Options {
		marker := "  "
		if opt == q.Default || (q.Default == "" && i == 0) {
			marker = "▶ "
			defaultIdx = i
		}
		_, _ = fmt.Fprintf(promptIO.Out, "  %s%d. %s%s\n", marker, i+1, opt, colorReset)
	}

	if defaultIdx >= 0 {
		_, _ = fmt.Fprintf(promptIO.Out, "\n%s  %s%s\n", colorDim, i18n.T("q.default_hint", q.Options[defaultIdx]), colorReset)
	}
	_, _ = fmt.Fprintf(promptIO.Out, "%s%s:%s ", colorCyan, i18n.T("q.choice_prompt"), colorReset)

	input, err := q.readInput(promptIO)
	if err != nil {
		return nil, err
	}
	input = strings.TrimSpace(input)

	// 空入力 = デフォルト
	if input == "" && defaultIdx >= 0 {
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, q.Options[defaultIdx], colorReset)
		return &QuestionnaireAnswer{Value: q.Options[defaultIdx]}, nil
	}

	// 数字入力
	if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(q.Options) {
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, q.Options[num-1], colorReset)
		return &QuestionnaireAnswer{Value: q.Options[num-1]}, nil
	}

	// テキストマッチ
	for _, opt := range q.Options {
		if strings.EqualFold(input, opt) {
			_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, opt, colorReset)
			return &QuestionnaireAnswer{Value: opt}, nil
		}
	}

	// 無効な入力 → 最初の選択肢
	_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s (default)%s\n", colorGreen, q.Options[0], colorReset)
	return &QuestionnaireAnswer{Value: q.Options[0]}, nil
}

// askMultiChoice は複数選択質問を処理
func (q *Questionnaire) askMultiChoice(promptIO PromptIO) (*QuestionnaireAnswer, error) {
	_, _ = fmt.Fprintf(promptIO.Out, "\n%s?%s %s\n", colorCyan, colorReset, q.Question)
	_, _ = fmt.Fprintf(promptIO.Out, "%s  %s%s\n\n", colorDim, i18n.T("q.multi_prompt"), colorReset)

	for i, opt := range q.Options {
		_, _ = fmt.Fprintf(promptIO.Out, "  %d. %s\n", i+1, opt)
	}

	_, _ = fmt.Fprintf(promptIO.Out, "\n%s%s:%s ", colorCyan, i18n.T("q.choice_prompt"), colorReset)

	input, err := q.readInput(promptIO)
	if err != nil {
		return nil, err
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return &QuestionnaireAnswer{Values: []string{}, IsMultiple: true}, nil
	}

	// カンマ区切りで解析
	parts := strings.Split(input, ",")
	var selected []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if num, err := strconv.Atoi(part); err == nil && num >= 1 && num <= len(q.Options) {
			selected = append(selected, q.Options[num-1])
		}
	}

	// 重複削除
	seen := make(map[string]bool)
	var unique []string
	for _, s := range selected {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}

	_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, i18n.T("q.selected", strings.Join(unique, ", ")), colorReset)
	return &QuestionnaireAnswer{Values: unique, IsMultiple: true}, nil
}

// askFreeText はフリーテキスト質問を処理
func (q *Questionnaire) askFreeText(promptIO PromptIO) (*QuestionnaireAnswer, error) {
	_, _ = fmt.Fprintf(promptIO.Out, "\n%s?%s %s\n", colorCyan, colorReset, q.Question)

	if q.Default != "" {
		_, _ = fmt.Fprintf(promptIO.Out, "%s  %s%s\n", colorDim, i18n.T("q.default_hint", q.Default), colorReset)
	}
	_, _ = fmt.Fprintf(promptIO.Out, "%s%s:%s ", colorCyan, i18n.T("q.text_prompt"), colorReset)

	input, err := q.readInput(promptIO)
	if err != nil {
		return nil, err
	}
	input = strings.TrimSpace(input)

	if input == "" && q.Default != "" {
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, q.Default, colorReset)
		return &QuestionnaireAnswer{Value: q.Default}, nil
	}

	_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, input, colorReset)
	return &QuestionnaireAnswer{Value: input}, nil
}

// readInput は入力を読み取る
func (q *Questionnaire) readInput(promptIO PromptIO) (string, error) {
	return promptIO.ReadSimpleLine()
}
