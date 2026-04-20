package agent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// handleCopyCommand は最後のAI出力をクリップボードにコピー
func handleCopyCommand(agent *Agent, args []string) bool {
	out := agent.output()

	// 引数解析（ロック不要な部分を先に処理）
	codeOnly := false
	requestedN := 0 // 0 = デフォルト（最後の出力）
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "code":
			codeOnly = true
		case "-n":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil {
					red.Fprintf(out, "Invalid number: %s\n", args[i+1])
					return true
				}
				requestedN = n
				i++
			} else {
				red.Fprintln(out, "Missing value for -n flag")
				return true
			}
		default:
			yellow.Fprintf(out, "Unknown argument: %s\n", arg)
			yellow.Fprintln(out, "Usage: /copy [code] [-n <number>]")
			return true
		}
	}

	// lastOutputs を1回のロックでスナップショット取得（TOCTOU 回避）
	agent.historyMu.Lock()
	outputCount := len(agent.lastOutputs)
	if outputCount == 0 {
		agent.historyMu.Unlock()
		yellow.Fprintln(out, "No AI output to copy yet")
		return true
	}
	outputIndex := outputCount - 1
	if requestedN > 0 {
		if requestedN < 1 || requestedN > outputCount {
			agent.historyMu.Unlock()
			red.Fprintf(out, "Index out of range (1-%d): %d\n", outputCount, requestedN)
			return true
		}
		outputIndex = outputCount - requestedN
	}
	output := agent.lastOutputs[outputIndex]
	agent.historyMu.Unlock()

	// コードブロックのみ抽出
	if codeOnly {
		codeBlocks := extractCodeBlocks(output)
		if len(codeBlocks) == 0 {
			yellow.Fprintln(out, "No code blocks found in output")
			return true
		}
		output = strings.Join(codeBlocks, "\n\n")
	}

	// クリップボードにコピー
	if err := clipboardWriteAll(output); err != nil {
		red.Fprintf(out, "Failed to copy to clipboard: %v\n", err)
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "xclip") || strings.Contains(errText, "xsel") || strings.Contains(errText, "wl-copy") {
			yellow.Fprintln(out, "\nLinux requires a clipboard utility:")
			yellow.Fprintln(out, "  Ubuntu/Debian: sudo apt-get install xclip")
			yellow.Fprintln(out, "  Fedora/RHEL:   sudo dnf install xclip")
			yellow.Fprintln(out, "  Arch:          sudo pacman -S xclip")
		}
		return true
	}

	// 成功メッセージ
	lines := strings.Count(output, "\n") + 1
	chars := len(output)
	green.Fprintf(out, "✅ Copied to clipboard (%d lines, %d chars", lines, chars)
	if codeOnly {
		_, _ = fmt.Fprint(out, ", code blocks only")
	}
	_, _ = fmt.Fprintln(out, ")")

	return true
}

// extractCodeBlocks は ```で囲まれたコードブロックを抽出
func extractCodeBlocks(text string) []string {
	// 正規表現: ```language\n...```
	re := regexp.MustCompile("(?s)```\\w*\\n(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}

	return blocks
}
