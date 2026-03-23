// colors はターミナルUI出力用の共有カラー定数を定義する。
package ui

import "github.com/fatih/color"

// 共通色定義
// 全パッケージで一貫した色使いを実現するための定義
var (
	// 基本色
	Cyan    = color.New(color.FgCyan)    // 情報・ヘッダー
	Yellow  = color.New(color.FgYellow)  // 警告・プロンプト
	Green   = color.New(color.FgGreen)   // 成功・追加
	Red     = color.New(color.FgRed)     // エラー・削除
	White   = color.New(color.FgWhite)   // 通常テキスト
	Blue    = color.New(color.FgBlue)    // リンク・参照
	Magenta = color.New(color.FgMagenta) // 特殊・ハイライト

	// Bold variants
	BoldCyan   = color.New(color.FgCyan, color.Bold)
	BoldYellow = color.New(color.FgYellow, color.Bold)
	BoldGreen  = color.New(color.FgGreen, color.Bold)
	BoldRed    = color.New(color.FgRed, color.Bold)
	BoldWhite  = color.New(color.FgWhite, color.Bold)

	// 特殊用途
	Dim   = color.New(color.Faint) // 薄い表示（コンテキスト行など）
	Bold  = color.New(color.Bold)  // 太字
	Faint = color.New(color.Faint) // 薄い表示
)

// 用途別エイリアス（セマンティックカラー）
var (
	Info    = Cyan     // 情報メッセージ
	Success = Green    // 成功メッセージ
	Warning = Yellow   // 警告メッセージ
	Error   = Red      // エラーメッセージ
	Prompt  = Yellow   // ユーザー入力プロンプト
	Header  = BoldCyan // セクションヘッダー
)
