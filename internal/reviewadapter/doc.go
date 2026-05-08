// Package reviewadapter は /review runner の外側依存を組み立てる adapter 境界を提供する。
//
// internal/review は domain contract と orchestration の owner として維持し、
// concrete な evidence/probe runner の選択はこの package で閉じる。
// provider、agent、TUI には依存しない。
package reviewadapter
