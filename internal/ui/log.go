package ui

import (
	"io"
	"os"

	"github.com/fatih/color"
)

// LogLevel はログメッセージの重要度を表す。
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

var currentLogLevel = LogInfo

func init() {
	if os.Getenv("XELYON_DEBUG") == "1" {
		currentLogLevel = LogDebug
	}
}

// SetLogLevel は現在の global ログレベルを設定する。
func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

// GetLogLevel は現在の global ログレベルを返す。
func GetLogLevel() LogLevel {
	return currentLogLevel
}

// --- WithRuntime 系（runtime のログレベルを参照） ---

// DebugWithRuntime は runtime のログレベルに従いデバッグログを出力する。
func DebugWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	if runtime.LogLevel() <= LogDebug {
		logMessage(runtime.ErrorOutput(), []color.Attribute{color.Faint}, "[DEBUG] ", format, args...)
	}
}

// InfoLogWithRuntime は runtime のログレベルに従い情報ログを出力する。
func InfoLogWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	if runtime.LogLevel() <= LogInfo {
		logMessage(runtime.Output(), []color.Attribute{color.FgCyan}, "", format, args...)
	}
}

// WarnWithRuntime は runtime のログレベルに従い警告ログを出力する。
func WarnWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	if runtime.LogLevel() <= LogWarn {
		logMessage(runtime.Output(), []color.Attribute{color.FgYellow}, "Warning: ", format, args...)
	}
}

// WarnWithoutEmojiWithRuntime は runtime のログレベルに従い接頭辞なし警告ログを出力する。
func WarnWithoutEmojiWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	if runtime.LogLevel() <= LogWarn {
		logMessage(runtime.Output(), []color.Attribute{color.FgYellow}, "", format, args...)
	}
}

// ErrorLogWithRuntime は runtime のログレベルに従いエラーログを出力する。
func ErrorLogWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	if runtime.LogLevel() <= LogError {
		logMessage(runtime.ErrorOutput(), []color.Attribute{color.FgRed}, "Error: ", format, args...)
	}
}

// ErrorLogWithoutEmojiWithRuntime は runtime のログレベルに従い接頭辞なしエラーログを出力する。
func ErrorLogWithoutEmojiWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	if runtime.LogLevel() <= LogError {
		logMessage(runtime.ErrorOutput(), []color.Attribute{color.FgRed}, "", format, args...)
	}
}

// SuccessLogWithRuntime は runtime の出力先に成功ログを出力する。
func SuccessLogWithRuntime(runtime *Runtime, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	logMessage(runtime.Output(), []color.Attribute{color.FgGreen}, "", format, args...)
}

// SuccessLogWithEmojiWithRuntime は runtime の出力先に絵文字付き成功ログを出力する。
func SuccessLogWithEmojiWithRuntime(runtime *Runtime, emoji, format string, args ...any) {
	runtime = runtimeOrDefault(runtime)
	logMessage(runtime.Output(), []color.Attribute{color.FgGreen}, emoji+" ", format, args...)
}

// --- compat wrappers（DefaultRuntime + global logLevel） ---

// Debug はデバッグログを出力する。
func Debug(format string, args ...any) {
	DebugWithRuntime(DefaultRuntime(), format, args...)
}

// DebugToWriter は指定 writer にデバッグログを出力する。
func DebugToWriter(w io.Writer, format string, args ...any) {
	if DefaultRuntime().LogLevel() <= LogDebug {
		logMessage(w, []color.Attribute{color.Faint}, "[DEBUG] ", format, args...)
	}
}

// InfoLog は情報ログを出力する。
func InfoLog(format string, args ...any) {
	InfoLogWithRuntime(DefaultRuntime(), format, args...)
}

// InfoLogToWriter は指定 writer に情報ログを出力する。
func InfoLogToWriter(w io.Writer, format string, args ...any) {
	if DefaultRuntime().LogLevel() <= LogInfo {
		logMessage(w, []color.Attribute{color.FgCyan}, "", format, args...)
	}
}

// Warn は警告ログを出力する。
func Warn(format string, args ...any) {
	WarnWithRuntime(DefaultRuntime(), format, args...)
}

// WarnToWriter は指定 writer に警告ログを出力する。
func WarnToWriter(w io.Writer, format string, args ...any) {
	if DefaultRuntime().LogLevel() <= LogWarn {
		logMessage(w, []color.Attribute{color.FgYellow}, "Warning: ", format, args...)
	}
}

// WarnWithoutEmoji は接頭辞なしの警告ログを出力する。
func WarnWithoutEmoji(format string, args ...any) {
	WarnWithoutEmojiWithRuntime(DefaultRuntime(), format, args...)
}

// WarnWithoutEmojiToWriter は指定 writer に接頭辞なしの警告ログを出力する。
func WarnWithoutEmojiToWriter(w io.Writer, format string, args ...any) {
	if DefaultRuntime().LogLevel() <= LogWarn {
		logMessage(w, []color.Attribute{color.FgYellow}, "", format, args...)
	}
}

// ErrorLog はエラーログを出力する。
func ErrorLog(format string, args ...any) {
	ErrorLogWithRuntime(DefaultRuntime(), format, args...)
}

// ErrorLogToWriter は指定 writer にエラーログを出力する。
func ErrorLogToWriter(w io.Writer, format string, args ...any) {
	if DefaultRuntime().LogLevel() <= LogError {
		logMessage(w, []color.Attribute{color.FgRed}, "Error: ", format, args...)
	}
}

// ErrorLogWithoutEmoji は接頭辞なしのエラーログを出力する。
func ErrorLogWithoutEmoji(format string, args ...any) {
	ErrorLogWithoutEmojiWithRuntime(DefaultRuntime(), format, args...)
}

// ErrorLogWithoutEmojiToWriter は指定 writer に接頭辞なしのエラーログを出力する。
func ErrorLogWithoutEmojiToWriter(w io.Writer, format string, args ...any) {
	if DefaultRuntime().LogLevel() <= LogError {
		logMessage(w, []color.Attribute{color.FgRed}, "", format, args...)
	}
}

// SuccessLog は成功ログを出力する。
func SuccessLog(format string, args ...any) {
	SuccessLogWithRuntime(DefaultRuntime(), format, args...)
}

// SuccessLogToWriter は指定 writer に成功ログを出力する。
func SuccessLogToWriter(w io.Writer, format string, args ...any) {
	logMessage(w, []color.Attribute{color.FgGreen}, "", format, args...)
}

// SuccessLogWithEmoji は絵文字付きの成功ログを出力する。
func SuccessLogWithEmoji(emoji, format string, args ...any) {
	SuccessLogWithEmojiWithRuntime(DefaultRuntime(), emoji, format, args...)
}

// SuccessLogWithEmojiToWriter は指定 writer に絵文字付き成功ログを出力する。
func SuccessLogWithEmojiToWriter(w io.Writer, emoji, format string, args ...any) {
	logMessage(w, []color.Attribute{color.FgGreen}, emoji+" ", format, args...)
}

// Fatal はエラーログを出力して終了する。
func Fatal(format string, args ...any) {
	ErrorLogWithRuntime(DefaultRuntime(), format, args...)
	os.Exit(1)
}

// Fatalf は整形済みエラーログを出力して終了する。
func Fatalf(format string, args ...any) {
	ErrorLogWithRuntime(DefaultRuntime(), format, args...)
	os.Exit(1)
}

func logMessage(w io.Writer, attrs []color.Attribute, prefix, format string, args ...any) {
	if w == nil {
		w = DefaultRuntime().Output()
	}
	_, _ = color.New(attrs...).Fprintf(w, prefix+format+"\n", args...)
}
