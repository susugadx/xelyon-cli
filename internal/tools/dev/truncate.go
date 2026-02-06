package dev

import (
	"fmt"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// TruncateWithFile は長い出力をファイルに保存し、先頭と末尾を返す。
// OutputTruncateLen（20000文字）以下の場合はそのまま返す。
// 超過時は /tmp/xelyon_output_<unixnano>.txt に全文を保存し、
// 先頭1000文字 + 末尾500文字 + ファイルパスを返す。
func TruncateWithFile(output string) string {
	if len(output) <= config.OutputTruncateLen {
		return output
	}

	// ファイルに全文保存
	filename := fmt.Sprintf("/tmp/xelyon_output_%d.txt", time.Now().UnixNano())
	if err := os.WriteFile(filename, []byte(output), 0600); err != nil {
		// ファイル保存に失敗した場合はフォールバック（従来方式）
		return output[:config.OutputTruncateLen] + "\n... (truncated)"
	}

	head := output[:1000]
	tail := output[len(output)-500:]

	return fmt.Sprintf("%s\n\n... (truncated %d chars, full output saved to %s)\n\n%s",
		head, len(output), filename, tail)
}
