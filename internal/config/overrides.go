package config

import "os"

const reviewWebSearchEvidenceEnv = "XELYON_REVIEW_WEB_SEARCH"

// ApplyEnvironmentOverrides は環境変数で設定を上書き
func (c *Config) ApplyEnvironmentOverrides() {
	applyBracketedPasteOverride(c)
	applyReviewWebSearchEvidenceOverride(c)
	applyLegacyEnvironmentOverrides(c)
}

func applyBracketedPasteOverride(c *Config) {
	// Bracketed Paste Mode の制御（XELYON_BRACKETED_PASTE=0 で無効化）
	if val := os.Getenv("XELYON_BRACKETED_PASTE"); val == "0" || val == "false" {
		c.Paste.BracketedPaste = false
	}
}

func applyReviewWebSearchEvidenceOverride(c *Config) {
	if c == nil {
		return
	}
	switch os.Getenv(reviewWebSearchEvidenceEnv) {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		c.Review.WebSearchEvidence.Enabled = true
	}
}

// ApplyFlagOverrides はCLIフラグで設定を上書き（内部設定）
func (c *Config) ApplyFlagOverrides(loopThreshold, diffLines *int) {
	applyLegacyFlagOverrides(c, loopThreshold, diffLines)
}
