// colors はターミナルUI出力用の共有カラー定数を定義する。
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
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

// fileOpPalette はファイル操作 diff 表示用の色関数を束ねる。
// 各フィールドは writer にテキストを色付きで書き出す（改行は含めない）。
type fileOpPalette struct {
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
	if color.NoColor {
		return false
	}
	if !isFileTerminal(w) {
		return false
	}
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit"
}

// writeTCFgBg は 24-bit truecolor の前景+背景でテキストを書き出す。
func writeTCFgBg(w io.Writer, fr, fg, fb, br, bg, bb byte, text string) {
	_, _ = fmt.Fprintf(w, "\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm%s\033[0m", fr, fg, fb, br, bg, bb, text)
}

// writeTCFg は 24-bit truecolor の前景のみでテキストを書き出す。
func writeTCFg(w io.Writer, r, g, b byte, text string) {
	_, _ = fmt.Fprintf(w, "\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, text)
}

// newFileOpPalette は writer の端末能力に応じたパレットを返す。
func newFileOpPalette(w io.Writer) fileOpPalette {
	if color.NoColor {
		return newPlainFileOpPalette()
	}
	if isTrueColorCapable(w) {
		return newTrueColorFileOpPalette()
	}
	return new16ColorFileOpPalette()
}

func newPlainFileOpPalette() fileOpPalette {
	plain := func(w io.Writer, text string) { _, _ = fmt.Fprint(w, text) }
	return fileOpPalette{
		AddLine: plain,
		DelLine: plain,
		Hunk:    plain,
		Accent:  plain,
		Muted:   plain,
		Border:  plain,
		Context: plain,
	}
}

func new16ColorFileOpPalette() fileOpPalette {
	wrap := func(attrs ...color.Attribute) func(io.Writer, string) {
		c := color.New(attrs...)
		return func(w io.Writer, text string) { _, _ = c.Fprint(w, text) }
	}
	return fileOpPalette{
		AddLine: wrap(color.FgGreen),
		DelLine: wrap(color.FgRed),
		Hunk:    wrap(color.FgCyan),
		Accent:  wrap(color.FgMagenta),
		Muted:   wrap(color.Faint),
		Border:  wrap(color.Faint),
		Context: wrap(color.FgWhite),
	}
}

func newTrueColorFileOpPalette() fileOpPalette {
	return fileOpPalette{
		AddLine: func(w io.Writer, text string) { writeTCFgBg(w, 184, 211, 240, 26, 47, 69, text) },
		DelLine: func(w io.Writer, text string) { writeTCFgBg(w, 227, 182, 175, 69, 38, 35, text) },
		Hunk:    func(w io.Writer, text string) { writeTCFg(w, 137, 168, 255, text) },
		Accent:  func(w io.Writer, text string) { writeTCFg(w, 183, 156, 255, text) },
		Muted:   func(w io.Writer, text string) { writeTCFg(w, 126, 135, 148, text) },
		Border:  func(w io.Writer, text string) { writeTCFg(w, 42, 49, 64, text) },
		Context: func(w io.Writer, text string) { writeTCFg(w, 216, 222, 233, text) },
	}
}

// ---------------------------------------------------------------------------
// ファイル操作 UI 用エクスポートヘルパー
// ---------------------------------------------------------------------------

// FileOpHeader はファイル操作ツールのコンパクトヘッダーを出力する。
//
//	── tool  detail ──
func FileOpHeader(w io.Writer, toolLabel, detail string) {
	if w == nil {
		return
	}
	pal := newFileOpPalette(w)
	pal.Border(w, "── ")
	pal.Accent(w, toolLabel)
	if detail != "" {
		_, _ = fmt.Fprint(w, "  ")
		pal.Muted(w, detail)
	}
	pal.Border(w, " ──")
	_, _ = fmt.Fprintln(w)
}

// FileOpDivider はファイル操作用の薄い区切り線を出力する。
func FileOpDivider(w io.Writer, width int) {
	if w == nil {
		return
	}
	pal := newFileOpPalette(w)
	pal.Border(w, strings.Repeat("─", width))
	_, _ = fmt.Fprintln(w)
}

// FileOpStatsLine はファイル操作の変更統計を 1 行で出力する。
//
//	-5 / +8 (net +3)
func FileOpStatsLine(w io.Writer, removed, added int) {
	if w == nil {
		return
	}
	pal := newFileOpPalette(w)
	_, _ = fmt.Fprint(w, "  ")
	pal.DelLine(w, fmt.Sprintf("-%d", removed))
	_, _ = fmt.Fprint(w, " / ")
	pal.AddLine(w, fmt.Sprintf("+%d", added))
	net := added - removed
	switch {
	case net > 0:
		_, _ = fmt.Fprint(w, " (net ")
		pal.AddLine(w, fmt.Sprintf("+%d", net))
		_, _ = fmt.Fprintln(w, ")")
	case net < 0:
		_, _ = fmt.Fprint(w, " (net ")
		pal.DelLine(w, fmt.Sprintf("%d", net))
		_, _ = fmt.Fprintln(w, ")")
	default:
		_, _ = fmt.Fprintln(w, " (net 0)")
	}
}

// FileOpPathLine はファイル操作のパス行を出力する。
//
//	M path (+3, -2)
func FileOpPathLine(w io.Writer, action, path, counts string) {
	if w == nil {
		return
	}
	pal := newFileOpPalette(w)
	_, _ = fmt.Fprint(w, "  ")
	pal.Muted(w, action)
	_, _ = fmt.Fprint(w, " ")
	pal.Accent(w, path)
	if counts != "" {
		_, _ = fmt.Fprint(w, " ")
		pal.Muted(w, counts)
	}
	_, _ = fmt.Fprintln(w)
}
