package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogEntry は監査ログのエントリ
type LogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Tool        string            `json:"tool"`
	Args        map[string]string `json:"args"`
	Output      string            `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	FileChanged bool              `json:"file_changed"`
	Success     bool              `json:"success"`
}

// Logger は監査ログを記録
type Logger struct {
	mu       sync.Mutex
	filePath string
	enabled  bool
}

// ToolLogger はツール実行の監査ログ記録に必要な最小インターフェース。
type ToolLogger interface {
	// LogToolExecution はツール実行結果を記録する。
	LogToolExecution(tool string, args map[string]string, output string, err error, fileChanged bool)
}

var (
	globalLogger   *Logger
	globalLoggerMu sync.RWMutex
)

// NewDefaultLogger は標準の保存先を使う監査ロガーを作成する。
func NewDefaultLogger(enabled bool) (*Logger, error) {
	if !enabled {
		return NewDisabledLogger(), nil
	}
	logPath, err := defaultLogPath()
	if err != nil {
		return nil, err
	}
	return NewLoggerWithPath(logPath, true), nil
}

// NewLoggerWithPath は保存先を明示した監査ロガーを作成する。
func NewLoggerWithPath(filePath string, enabled bool) *Logger {
	return &Logger{
		filePath: filePath,
		enabled:  enabled,
	}
}

// NewDisabledLogger は無効状態の監査ロガーを返す。
func NewDisabledLogger() *Logger {
	return &Logger{enabled: false}
}

// SetGlobalLogger は互換用途のグローバル監査ロガーを差し替える。
func SetGlobalLogger(logger *Logger) {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	globalLogger = logger
}

// Init は監査ログを初期化
func Init(enabled bool) error {
	logger, err := NewDefaultLogger(enabled)
	if err != nil {
		return err
	}
	SetGlobalLogger(logger)
	return err
}

// GetLogger はグローバルロガーを取得
func GetLogger() *Logger {
	globalLoggerMu.RLock()
	defer globalLoggerMu.RUnlock()
	if globalLogger == nil {
		// デフォルトで無効状態のロガーを返す
		return NewDisabledLogger()
	}
	return globalLogger
}

func defaultLogPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	logDir := filepath.Join(homeDir, ".xelyon", "audit")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return "", err
	}

	logFileName := fmt.Sprintf("audit_%s.jsonl", time.Now().Format("20060102"))
	return filepath.Join(logDir, logFileName), nil
}

// LogToolExecution はツール実行をログに記録
func (l *Logger) LogToolExecution(tool string, args map[string]string, output string, err error, fileChanged bool) {
	if !l.enabled {
		return
	}

	entry := LogEntry{
		Timestamp:   time.Now(),
		Tool:        tool,
		Args:        sanitizeArgs(args),
		FileChanged: fileChanged,
		Success:     err == nil,
	}

	// 出力は最初の500文字のみ記録（ログサイズ削減）
	if len(output) > 500 {
		entry.Output = output[:500] + "... (truncated)"
	} else {
		entry.Output = output
	}

	if err != nil {
		entry.Error = err.Error()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// JSONLフォーマット（1行1エントリ）
	file, e := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if e != nil {
		// ログ記録失敗はサイレント（通常動作を妨げない）
		return
	}
	defer file.Close()

	data, e := json.Marshal(entry)
	if e != nil {
		return
	}

	if _, err := file.Write(data); err != nil {
		return
	}
	if _, err := file.WriteString("\n"); err != nil {
		return
	}
}

// sanitizeArgs は機密情報を含む可能性のある引数をサニタイズ
func sanitizeArgs(args map[string]string) map[string]string {
	sanitized := make(map[string]string)
	for k, v := range args {
		// パスワード、トークン、APIキーなどは記録しない
		if k == "password" || k == "token" || k == "api_key" || k == "secret" {
			sanitized[k] = "[REDACTED]"
		} else if len(v) > 200 {
			// 長すぎる値は切り詰め
			sanitized[k] = v[:200] + "... (truncated)"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}
