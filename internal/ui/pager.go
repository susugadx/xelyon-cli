package ui

import (
	"fmt"
	"strings"
)

const defaultPageSize = 100

// Pager は長い出力をページング表示
type Pager struct {
	pageSize int
	Runtime  *Runtime
}

// NewPager は新しいPagerを作成
func NewPager() *Pager {
	return NewPagerWithRuntime(DefaultRuntime())
}

// NewPagerWithRuntime は UI runtime を指定して新しい Pager を作成する。
func NewPagerWithRuntime(runtime *Runtime) *Pager {
	return &Pager{
		pageSize: defaultPageSize,
		Runtime:  runtimeOrDefault(runtime),
	}
}

// Display はコンテンツをページング表示
func (p *Pager) Display(content string) {
	promptIO := runtimeOrDefault(p.Runtime).PromptIO()
	out := promptIO.Out
	lines := strings.Split(content, "\n")

	if len(lines) <= p.pageSize {
		// ページング不要
		_, _ = fmt.Fprintln(out, content)
		return
	}

	// ページングで表示
	for i := 0; i < len(lines); i += p.pageSize {
		end := i + p.pageSize
		if end > len(lines) {
			end = len(lines)
		}

		// このページを表示
		for _, line := range lines[i:end] {
			_, _ = fmt.Fprintln(out, line)
		}

		// まだ続きがある場合はプロンプト
		if end < len(lines) {
			if !p.promptContinue(&promptIO) {
				return
			}
		}
	}
}

// promptContinue は継続確認プロンプトを表示
func (p *Pager) promptContinue(promptIO *PromptIO) bool {
	if promptIO == nil {
		defaultPromptIO := DefaultPromptIO()
		promptIO = &defaultPromptIO
	}
	_, _ = fmt.Fprint(promptIO.Out, Yellow.Sprint("--more-- (Press Enter to continue, q to quit): "))

	input, err := promptIO.ReadSimpleLine()
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input != "q" && input != "quit"
}
