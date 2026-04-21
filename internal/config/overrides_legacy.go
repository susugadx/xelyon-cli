package config

import (
	"fmt"
	"os"
	"strconv"
)

func applyLegacyEnvironmentOverrides(c *Config) {
	applyEnvInt("XELYON_LOOP_THRESHOLD", func(n int) bool { return n > 0 },
		func(n int) { c.LoopDetection.Threshold = n }, "positive integer")
	applyEnvInt("XELYON_DIFF_CONTEXT_LINES", func(n int) bool { return n >= 0 },
		func(n int) { c.Diff.ContextLines = n }, "non-negative integer")
}

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

func applyLegacyFlagOverrides(c *Config, loopThreshold, diffLines *int) {
	if loopThreshold != nil && *loopThreshold > 0 {
		c.LoopDetection.Threshold = *loopThreshold
	}
	if diffLines != nil && *diffLines >= 0 {
		c.Diff.ContextLines = *diffLines
	}
}
