package agent

import (
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// runREPLLoop は legacy classic REPL ループを実行する。
func runREPLLoop(agent *Agent, mlReader *ui.MultilineReader) {
	var lastInterrupt time.Time

	for {
		// AI出力後に溜まった入力をクリア（出力中のEnter押下を無視）
		mlReader.FlushInput()

		// Status / 状態表示（常にプロンプト直前に表示）
		agent.PrintStatusFooter()

		// 緑色のプロンプト
		greenPrompt := green.Sprint(">")
		input, err := mlReader.ReadInput("\n" + greenPrompt + " ")
		if err != nil {
			if handleREPLReadError(agent, err, &lastInterrupt) {
				continue
			}
			// Other errors (like EOF): exit loop
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 特殊コマンド
		if handleSpecialCommand(input, agent) {
			continue
		}

		// 画像入力チェック: image:/path/to/file.png 形式を検出
		if strings.Contains(input, "image:") {
			textPart, image := parseImageInputWithWriter(agent.output(), input)
			if image != nil {
				_ = agent.chatWithImage(textPart, image)
				continue
			}
		}

		// AIに送信
		_ = agent.chat(input)
	}
}
