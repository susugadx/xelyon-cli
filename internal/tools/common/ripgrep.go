package common

import (
	"os/exec"
	"sync"
)

var (
	rgAvailable bool
	rgPath      string
	rgCheckOnce sync.Once
	rgLookPath  = exec.LookPath
)

// IsRipgrepAvailable は ripgrep (rg) がインストールされているかを返す。
// 初回呼び出し時に exec.LookPath で検索し、結果をキャッシュする。
func IsRipgrepAvailable() bool {
	rgCheckOnce.Do(resolveRipgrep)
	return rgAvailable
}

// RipgrepPath は ripgrep のフルパスを返す。未検出時は "rg" を返す。
func RipgrepPath() string {
	rgCheckOnce.Do(resolveRipgrep)
	if rgPath != "" {
		return rgPath
	}
	return "rg"
}

func resolveRipgrep() {
	path, err := rgLookPath("rg")
	if err == nil {
		rgAvailable = true
		rgPath = path
	}
}

// ResetRipgrepAvailabilityForTest はテスト用に ripgrep 検査キャッシュをリセットする。
func ResetRipgrepAvailabilityForTest() {
	rgAvailable = false
	rgPath = ""
	rgCheckOnce = sync.Once{}
	rgLookPath = exec.LookPath
}
