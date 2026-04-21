package config

import "os"

// ApplyEnvironmentOverrides は環境変数で設定を上書き
func (c *Config) ApplyEnvironmentOverrides() {
	applyBracketedPasteOverride(c)
	applyLegacyEnvironmentOverrides(c)
}

func applyBracketedPasteOverride(c *Config) {
	// Bracketed Paste Mode の制御（XELYON_BRACKETED_PASTE=0 で無効化）
	if val := os.Getenv("XELYON_BRACKETED_PASTE"); val == "0" || val == "false" {
		c.Paste.BracketedPaste = false
	}
}

// ApplyFlagOverrides はCLIフラグで設定を上書き（内部設定）
func (c *Config) ApplyFlagOverrides(loopThreshold, diffLines *int) {
	applyLegacyFlagOverrides(c, loopThreshold, diffLines)
}
