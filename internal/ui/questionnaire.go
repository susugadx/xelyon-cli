// Package ui は質問表示 UI を提供する
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/i18n"
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
	StopGlobalSpinner()
	fmt.Print("\033[?25h") // カーソル表示

	switch q.QuestionType {
	case "single_choice":
		return q.askSingleChoice()
	case "multi_choice":
		return q.askMultiChoice()
	case "free_text":
		return q.askFreeText()
	default:
		return nil, fmt.Errorf("unknown question type: %s", q.QuestionType)
	}
}

// askSingleChoice は単一選択質問を処理
func (q *Questionnaire) askSingleChoice() (*QuestionnaireAnswer, error) {
	fmt.Printf("\n%s?%s %s\n\n", colorCyan, colorReset, q.Question)

	defaultIdx := -1
	for i, opt := range q.Options {
		marker := "  "
		if opt == q.Default || (q.Default == "" && i == 0) {
			marker = "▶ "
			defaultIdx = i
		}
		fmt.Printf("  %s%d. %s%s\n", marker, i+1, opt, colorReset)
	}

	if defaultIdx >= 0 {
		fmt.Printf("\n%s  %s%s\n", colorDim, i18n.T("q.default_hint", q.Options[defaultIdx]), colorReset)
	}
	fmt.Printf("%s%s:%s ", colorCyan, i18n.T("q.choice_prompt"), colorReset)

	input := q.readInput()
	input = strings.TrimSpace(input)

	// 空入力 = デフォルト
	if input == "" && defaultIdx >= 0 {
		fmt.Printf("%s✓ %s%s\n", colorGreen, q.Options[defaultIdx], colorReset)
		return &QuestionnaireAnswer{Value: q.Options[defaultIdx]}, nil
	}

	// 数字入力
	if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(q.Options) {
		fmt.Printf("%s✓ %s%s\n", colorGreen, q.Options[num-1], colorReset)
		return &QuestionnaireAnswer{Value: q.Options[num-1]}, nil
	}

	// テキストマッチ
	for _, opt := range q.Options {
		if strings.EqualFold(input, opt) {
			fmt.Printf("%s✓ %s%s\n", colorGreen, opt, colorReset)
			return &QuestionnaireAnswer{Value: opt}, nil
		}
	}

	// 無効な入力 → 最初の選択肢
	fmt.Printf("%s✓ %s (default)%s\n", colorGreen, q.Options[0], colorReset)
	return &QuestionnaireAnswer{Value: q.Options[0]}, nil
}

// askMultiChoice は複数選択質問を処理
func (q *Questionnaire) askMultiChoice() (*QuestionnaireAnswer, error) {
	fmt.Printf("\n%s?%s %s\n", colorCyan, colorReset, q.Question)
	fmt.Printf("%s  %s%s\n\n", colorDim, i18n.T("q.multi_prompt"), colorReset)

	for i, opt := range q.Options {
		fmt.Printf("  %d. %s\n", i+1, opt)
	}

	fmt.Printf("\n%s%s:%s ", colorCyan, i18n.T("q.choice_prompt"), colorReset)

	input := q.readInput()
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

	fmt.Printf("%s✓ %s%s\n", colorGreen, i18n.T("q.selected", strings.Join(unique, ", ")), colorReset)
	return &QuestionnaireAnswer{Values: unique, IsMultiple: true}, nil
}

// askFreeText はフリーテキスト質問を処理
func (q *Questionnaire) askFreeText() (*QuestionnaireAnswer, error) {
	fmt.Printf("\n%s?%s %s\n", colorCyan, colorReset, q.Question)

	if q.Default != "" {
		fmt.Printf("%s  %s%s\n", colorDim, i18n.T("q.default_hint", q.Default), colorReset)
	}
	fmt.Printf("%s%s:%s ", colorCyan, i18n.T("q.text_prompt"), colorReset)

	input := q.readInput()
	input = strings.TrimSpace(input)

	if input == "" && q.Default != "" {
		fmt.Printf("%s✓ %s%s\n", colorGreen, q.Default, colorReset)
		return &QuestionnaireAnswer{Value: q.Default}, nil
	}

	fmt.Printf("%s✓ %s%s\n", colorGreen, input, colorReset)
	return &QuestionnaireAnswer{Value: input}, nil
}

// readInput は入力を読み取る
func (q *Questionnaire) readInput() string {
	if reader := GetGlobalReader(); reader != nil {
		line, err := reader.ReadSimpleLine()
		if err == nil {
			return line
		}
	}
	return readLineFromStdin()
}
