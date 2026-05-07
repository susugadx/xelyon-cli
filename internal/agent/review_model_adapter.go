package agent

// ReviewModel adapter の owner は internal/agent に置く。
// real adapter は CurrentProvider、CurrentModel、requestContext、config/UI/registry runtime を読むため、
// internal/review や TUI ではなく Agent 境界で provider call へ変換する。
// 次の接続では review.ReviewModelRequest.Prompt を完全な provider prompt として渡し、
// Phase は telemetry/status 用の境界として扱う。
