// colors はターミナルUI出力用の共有カラー定数を定義する。
package uistyle

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
)

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

// ---------------------------------------------------------------------------
// ファイル操作 UI 用パレット
// ---------------------------------------------------------------------------

// FileOpPalette はファイル操作 diff 表示用の色関数を束ねる。
// 各フィールドは writer にテキストを色付きで書き出す（改行は含めない）。
type FileOpPalette struct {
	AddLine func(io.Writer, string) // 追加行（AddFg on AddBg）
	DelLine func(io.Writer, string) // 削除行（DelFg on DelBg）
	Hunk    func(io.Writer, string) // @@ ハンクヘッダー
	Accent  func(io.Writer, string) // パス名・ツールラベル
	Muted   func(io.Writer, string) // 薄いテキスト
	Border  func(io.Writer, string) // 区切り線
	Context func(io.Writer, string) // 変更なし行
}

// isTrueColorCapable は writer が truecolor 対応端末かどうかを返す。
func isTrueColorCapable(w io.Writer) bool {
	return shouldUseTrueColor(color.NoColor, isFileTerminal(w), os.Getenv("COLORTERM"))
}

func isFileTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func shouldUseTrueColor(noColor bool, isTerminal bool, colorTerm string) bool {
	if noColor || !isTerminal {
		return false
	}
	return colorTerm == "truecolor" || colorTerm == "24bit"
}

// writeTCFgBg は 24-bit truecolor の前景+背景でテキストを書き出す。
func writeTCFgBg(w io.Writer, fr, fg, fb, br, bg, bb byte, text string) {
	_, _ = fmt.Fprintf(w, "\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm%s\033[0m", fr, fg, fb, br, bg, bb, text)
}

// writeTCFg は 24-bit truecolor の前景のみでテキストを書き出す。
func writeTCFg(w io.Writer, r, g, b byte, text string) {
	_, _ = fmt.Fprintf(w, "\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, text)
}

// NewFileOpPalette は writer の端末能力に応じたパレットを返す。
func NewFileOpPalette(w io.Writer) FileOpPalette {
	return newFileOpPaletteForCapabilities(color.NoColor, isFileTerminal(w), os.Getenv("COLORTERM"))
}

func newFileOpPaletteForCapabilities(noColor bool, isTerminal bool, colorTerm string) FileOpPalette {
	if noColor {
		return newPlainFileOpPalette()
	}
	if shouldUseTrueColor(false, isTerminal, colorTerm) {
		return newTrueColorFileOpPalette()
	}
	return new16ColorFileOpPalette()
}

func newPlainFileOpPalette() FileOpPalette {
	plain := func(w io.Writer, text string) { _, _ = fmt.Fprint(w, text) }
	return FileOpPalette{
		AddLine: plain,
		DelLine: plain,
		Hunk:    plain,
		Accent:  plain,
		Muted:   plain,
		Border:  plain,
		Context: plain,
	}
}

func new16ColorFileOpPalette() FileOpPalette {
	wrap := func(attrs ...color.Attribute) func(io.Writer, string) {
		c := color.New(attrs...)
		return func(w io.Writer, text string) { _, _ = c.Fprint(w, text) }
	}
	return FileOpPalette{
		AddLine: wrap(color.FgGreen),
		DelLine: wrap(color.FgRed),
		Hunk:    wrap(color.FgCyan),
		Accent:  wrap(color.FgMagenta),
		Muted:   wrap(color.Faint),
		Border:  wrap(color.Faint),
		Context: wrap(color.FgWhite),
	}
}

func newTrueColorFileOpPalette() FileOpPalette {
	return FileOpPalette{
		AddLine: func(w io.Writer, text string) { writeTCFgBg(w, 184, 211, 240, 26, 47, 69, text) },
		DelLine: func(w io.Writer, text string) { writeTCFgBg(w, 227, 182, 175, 69, 38, 35, text) },
		Hunk:    func(w io.Writer, text string) { writeTCFg(w, 137, 168, 255, text) },
		Accent:  func(w io.Writer, text string) { writeTCFg(w, 183, 156, 255, text) },
		Muted:   func(w io.Writer, text string) { writeTCFg(w, 126, 135, 148, text) },
		Border:  func(w io.Writer, text string) { writeTCFg(w, 42, 49, 64, text) },
		Context: func(w io.Writer, text string) { writeTCFg(w, 216, 222, 233, text) },
	}
}
