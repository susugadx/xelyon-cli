package config

import "time"

// HTTP Configuration
const (
	DefaultHTTPTimeout = 30 * time.Second
	SerperHTTPTimeout  = 10 * time.Second // Serper API専用
)

// Tool Execution Limits
const (
	MaxToolIterations = 10 // ツールループ最大回数
	MaxChangeStack    = 10 // Undo履歴最大保存数
)

// Output Display Limits
const (
	OutputTruncateLen   = 5000 // bash出力切り詰め長
	MaxDiffDisplayLines = 15   // diff表示最大行数
	MaxDiffIterations   = 20   // diff比較最大イテレーション
)
