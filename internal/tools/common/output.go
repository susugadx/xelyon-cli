package common

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// QuietColor は quiet mode 中の標準出力を抑制する色付き出力ラッパー。
type QuietColor struct {
	base *color.Color
}

// Print は色付きで出力する。
func (c QuietColor) Print(args ...interface{}) {
	if IsQuietMode() {
		return
	}
	c.base.Print(args...)
}

// Printf は色付きでフォーマット出力する。
func (c QuietColor) Printf(format string, args ...interface{}) {
	if IsQuietMode() {
		return
	}
	c.base.Printf(format, args...)
}

// Println は色付きで1行出力する。
func (c QuietColor) Println(args ...interface{}) {
	if IsQuietMode() {
		return
	}
	c.base.Println(args...)
}

// Sprint は文字列を返すため quiet mode でも抑制しない。
func (c QuietColor) Sprint(args ...interface{}) string {
	return c.base.Sprint(args...)
}

// Sprintf は文字列を返すため quiet mode でも抑制しない。
func (c QuietColor) Sprintf(format string, args ...interface{}) string {
	return c.base.Sprintf(format, args...)
}

// Print は quiet mode 中の標準出力を抑制する。
func Print(args ...interface{}) {
	if IsQuietMode() {
		return
	}
	fmt.Print(args...)
}

// Printf は quiet mode 中の標準出力を抑制する。
func Printf(format string, args ...interface{}) {
	if IsQuietMode() {
		return
	}
	fmt.Printf(format, args...)
}

// Println は quiet mode 中の標準出力を抑制する。
func Println(args ...interface{}) {
	if IsQuietMode() {
		return
	}
	fmt.Println(args...)
}

// Colors - ui パッケージの共通色を quiet-aware にしたラッパー。
var (
	Yellow = QuietColor{base: ui.Yellow}
	Green  = QuietColor{base: ui.Green}
	Red    = QuietColor{base: ui.Red}
	Cyan   = QuietColor{base: ui.Cyan}
	Dim    = QuietColor{base: ui.Dim}
)
