package openaisubscription

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

// DiagnosticStatus は subscription doctor check の結果を表します。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

const DiagnosticRouteResponsesStreaming = "responses_streaming"

// DiagnosticCheck は subscription 設定診断の 1 項目です。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticCapabilities は provider capability snapshot の表示 contract です。
type DiagnosticCapabilities = providerdiag.DiagnosticCapabilities

// CompactRequest は subscription Compact API request を表します。
type CompactRequest struct {
	Model        string          `json:"model"`
	Input        []api.InputItem `json:"input"`
	Instructions string          `json:"instructions,omitempty"`
}
