package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const defaultPageSize = 100

// Pager は長い出力をページング表示
type Pager struct {
	pageSize int
}

// NewPager は新しいPagerを作成
func NewPager() *Pager {
	return &Pager{
		pageSize: defaultPageSize,
	}
}

// Display はコンテンツをページング表示
func (p *Pager) Display(content string) {
	lines := strings.Split(content, "\n")

	if len(lines) <= p.pageSize {
		// ページング不要
		fmt.Println(content)
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
			fmt.Println(line)
		}

		// まだ続きがある場合はプロンプト
		if end < len(lines) {
			if !p.promptContinue() {
				return
			}
		}
	}
}

// promptContinue は継続確認プロンプトを表示
func (p *Pager) promptContinue() bool {
	Yellow.Print("--more-- (Press Enter to continue, q to quit): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input != "q" && input != "quit"
}
