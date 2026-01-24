package ui

import (
	"fmt"
	"os"
)

// LogLevel represents the severity of a log message
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

// SetLogLevel sets the current logging level
func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

// GetLogLevel returns the current logging level
func GetLogLevel() LogLevel {
	return currentLogLevel
}

// Debug logs a debug message (only when XELYON_DEBUG=1)
func Debug(format string, args ...interface{}) {
	if currentLogLevel <= LogDebug {
		Dim.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// InfoLog logs an info message
func InfoLog(format string, args ...interface{}) {
	if currentLogLevel <= LogInfo {
		Info.Printf(format+"\n", args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if currentLogLevel <= LogWarn {
		Warning.Printf("Warning: "+format+"\n", args...)
	}
}

// WarnWithoutEmoji logs a warning message without prefix
func WarnWithoutEmoji(format string, args ...interface{}) {
	if currentLogLevel <= LogWarn {
		Warning.Printf(format+"\n", args...)
	}
}

// ErrorLog logs an error message
func ErrorLog(format string, args ...interface{}) {
	if currentLogLevel <= LogError {
		Error.Printf("Error: "+format+"\n", args...)
	}
}

// ErrorLogWithoutEmoji logs an error message without prefix
func ErrorLogWithoutEmoji(format string, args ...interface{}) {
	if currentLogLevel <= LogError {
		Error.Printf(format+"\n", args...)
	}
}

// SuccessLog logs a success message
func SuccessLog(format string, args ...interface{}) {
	Success.Printf(format+"\n", args...)
}

// SuccessLogWithEmoji logs a success message with custom emoji
func SuccessLogWithEmoji(emoji, format string, args ...interface{}) {
	Success.Printf(emoji+" "+format+"\n", args...)
}

// Fatal logs an error message and exits with code 1
func Fatal(format string, args ...interface{}) {
	Error.Printf("Error: "+format+"\n", args...)
	os.Exit(1)
}

// Fatalf logs an error message to stderr and exits with code 1
func Fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
