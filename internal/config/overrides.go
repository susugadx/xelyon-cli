package config

import (
	"fmt"
	"os"
	"strconv"
)

// ApplyEnvironmentOverrides は環境変数で設定を上書き
func (c *Config) ApplyEnvironmentOverrides() {
	// Bracketed Paste Mode の制御（XELYON_BRACKETED_PASTE=0 で無効化）
	if val := os.Getenv("XELYON_BRACKETED_PASTE"); val == "0" || val == "false" {
		c.Paste.BracketedPaste = false
	}
	// 内部設定の環境変数オーバーライド（後方互換）
	applyEnvInt("XELYON_LOOP_THRESHOLD", func(n int) bool { return n > 0 },
		func(n int) { c.LoopDetection.Threshold = n }, "positive integer")
	applyEnvInt("XELYON_DIFF_CONTEXT_LINES", func(n int) bool { return n >= 0 },
		func(n int) { c.Diff.ContextLines = n }, "non-negative integer")
}

// applyEnvInt は環境変数を整数として読み込み、バリデーション後に適用する。
// 不正な値の場合は stderr に警告を出力する。
func applyEnvInt(envKey string, valid func(int) bool, apply func(int), expect string) {
	val := os.Getenv(envKey)
	if val == "" {
		return
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: invalid %s=%q (expected %s)\n", envKey, val, expect)
		return
	}
	if !valid(n) {
		fmt.Fprintf(os.Stderr, "Warning: invalid %s=%q (expected %s)\n", envKey, val, expect)
		return
	}
	apply(n)
}

// ApplyFlagOverrides はCLIフラグで設定を上書き（内部設定）
func (c *Config) ApplyFlagOverrides(loopThreshold, diffLines *int) {
	if loopThreshold != nil && *loopThreshold > 0 {
		c.LoopDetection.Threshold = *loopThreshold
	}
	if diffLines != nil && *diffLines >= 0 {
		c.Diff.ContextLines = *diffLines
	}
}
