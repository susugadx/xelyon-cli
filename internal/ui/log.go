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

// SetLogLevel は現在のログレベルを設定する。
func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

// GetLogLevel は現在のログレベルを返す。
func GetLogLevel() LogLevel {
	return currentLogLevel
}

// Debug はデバッグログを出力する。
func Debug(format string, args ...interface{}) {
	DebugToWriter(DefaultRuntime().ErrorOutput(), format, args...)
}

// DebugToWriter は指定 writer にデバッグログを出力する。
func DebugToWriter(w io.Writer, format string, args ...interface{}) {
	if currentLogLevel <= LogDebug {
		logMessage(w, []color.Attribute{color.Faint}, "[DEBUG] ", format, args...)
	}
}

// InfoLog は情報ログを出力する。
func InfoLog(format string, args ...interface{}) {
	InfoLogToWriter(DefaultRuntime().Output(), format, args...)
}

// InfoLogToWriter は指定 writer に情報ログを出力する。
func InfoLogToWriter(w io.Writer, format string, args ...interface{}) {
	if currentLogLevel <= LogInfo {
		logMessage(w, []color.Attribute{color.FgCyan}, "", format, args...)
	}
}

// Warn は警告ログを出力する。
func Warn(format string, args ...interface{}) {
	WarnToWriter(DefaultRuntime().Output(), format, args...)
}

// WarnToWriter は指定 writer に警告ログを出力する。
func WarnToWriter(w io.Writer, format string, args ...interface{}) {
	if currentLogLevel <= LogWarn {
		logMessage(w, []color.Attribute{color.FgYellow}, "Warning: ", format, args...)
	}
}

// WarnWithoutEmoji は接頭辞なしの警告ログを出力する。
func WarnWithoutEmoji(format string, args ...interface{}) {
	WarnWithoutEmojiToWriter(DefaultRuntime().Output(), format, args...)
}

// WarnWithoutEmojiToWriter は指定 writer に接頭辞なしの警告ログを出力する。
func WarnWithoutEmojiToWriter(w io.Writer, format string, args ...interface{}) {
	if currentLogLevel <= LogWarn {
		logMessage(w, []color.Attribute{color.FgYellow}, "", format, args...)
	}
}

// ErrorLog はエラーログを出力する。
func ErrorLog(format string, args ...interface{}) {
	ErrorLogToWriter(DefaultRuntime().ErrorOutput(), format, args...)
}

// ErrorLogToWriter は指定 writer にエラーログを出力する。
func ErrorLogToWriter(w io.Writer, format string, args ...interface{}) {
	if currentLogLevel <= LogError {
		logMessage(w, []color.Attribute{color.FgRed}, "Error: ", format, args...)
	}
}

// ErrorLogWithoutEmoji は接頭辞なしのエラーログを出力する。
func ErrorLogWithoutEmoji(format string, args ...interface{}) {
	ErrorLogWithoutEmojiToWriter(DefaultRuntime().ErrorOutput(), format, args...)
}

// ErrorLogWithoutEmojiToWriter は指定 writer に接頭辞なしのエラーログを出力する。
func ErrorLogWithoutEmojiToWriter(w io.Writer, format string, args ...interface{}) {
	if currentLogLevel <= LogError {
		logMessage(w, []color.Attribute{color.FgRed}, "", format, args...)
	}
}

// SuccessLog は成功ログを出力する。
func SuccessLog(format string, args ...interface{}) {
	SuccessLogToWriter(DefaultRuntime().Output(), format, args...)
}

// SuccessLogToWriter は指定 writer に成功ログを出力する。
func SuccessLogToWriter(w io.Writer, format string, args ...interface{}) {
	logMessage(w, []color.Attribute{color.FgGreen}, "", format, args...)
}

// SuccessLogWithEmoji は絵文字付きの成功ログを出力する。
func SuccessLogWithEmoji(emoji, format string, args ...interface{}) {
	SuccessLogWithEmojiToWriter(DefaultRuntime().Output(), emoji, format, args...)
}

// SuccessLogWithEmojiToWriter は指定 writer に絵文字付き成功ログを出力する。
func SuccessLogWithEmojiToWriter(w io.Writer, emoji, format string, args ...interface{}) {
	logMessage(w, []color.Attribute{color.FgGreen}, emoji+" ", format, args...)
}

// Fatal はエラーログを出力して終了する。
func Fatal(format string, args ...interface{}) {
	ErrorLogToWriter(DefaultRuntime().ErrorOutput(), format, args...)
	os.Exit(1)
}

// Fatalf は整形済みエラーログを出力して終了する。
func Fatalf(format string, args ...interface{}) {
	ErrorLogToWriter(DefaultRuntime().ErrorOutput(), format, args...)
	os.Exit(1)
}

func logMessage(w io.Writer, attrs []color.Attribute, prefix, format string, args ...interface{}) {
	if w == nil {
		w = DefaultRuntime().Output()
	}
	_, _ = color.New(attrs...).Fprintf(w, prefix+format+"\n", args...)
}
