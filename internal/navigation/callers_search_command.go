package navigation

import (
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const ripgrepReferenceTimeout = 10 * time.Second

// findReferences は ripgrep でシンボル名を検索し、全参照を返す。
// truncated が true の場合、上流の検索結果が上限を超えたことを示す。
// incomplete が true の場合、読み取り失敗や異常終了により結果が不完全であることを示す。
func findReferences(symbol string) (refs []Reference, truncated bool, incomplete bool) {
	if !common.IsRipgrepAvailable() {
		return nil, false, false
	}

	stdout, cancel, wait, err := startReferenceSearchCommand(symbol)
	if err != nil {
		return nil, false, true
	}
	defer cancel()

	return runReferenceSearch(stdout, symbol, cancel, wait)
}

func startReferenceSearchCommand(symbol string) (stdout io.Reader, cancel context.CancelFunc, wait func() error, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), ripgrepReferenceTimeout)
	cmd := exec.CommandContext(ctx, common.RipgrepPath(), ripgrepReferenceSearchArgs(symbol)...)

	pipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		cancelFn()
		return nil, nil, nil, pipeErr
	}
	if startErr := cmd.Start(); startErr != nil {
		cancelFn()
		return nil, nil, nil, startErr
	}

	return pipe, cancelFn, cmd.Wait, nil
}

func ripgrepReferenceSearchArgs(symbol string) []string {
	return []string{
		"-n",
		"--no-heading",
		"--color", "never",
		"-w", // 単語境界
		"--type", "go",
		"--glob", "!vendor/",
		symbol,
		".",
	}
}
