package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	// HeadlessSchemaVersion は headless JSON contract の schema version。
	HeadlessSchemaVersion = "xelyon.headless.v1"

	// HeadlessStatusSuccess は headless JSON の成功 status。
	HeadlessStatusSuccess = "success"
	// HeadlessStatusError は headless JSON の失敗 status。
	HeadlessStatusError = "error"

	// HeadlessErrorTypeConfig は CLI/config/input validation 系の headless error type。
	HeadlessErrorTypeConfig = "config_error"
	// HeadlessErrorTypeCancelled は context cancel / timeout 系の headless error type。
	HeadlessErrorTypeCancelled = "cancelled"
	// HeadlessErrorTypeAPI は provider request 失敗の headless error type。
	HeadlessErrorTypeAPI = "api_error"
	// HeadlessErrorTypeProviderSetupRequired は provider credential setup 未完了の headless error type。
	HeadlessErrorTypeProviderSetupRequired = "provider_setup_required"
	// HeadlessErrorTypeToolLoopLimit は tool loop limit 到達時の headless error type。
	HeadlessErrorTypeToolLoopLimit = "tool_loop_limit"
	// HeadlessErrorTypeToolError は strict mode で tool 実行失敗を全体失敗へ昇格した headless error type。
	HeadlessErrorTypeToolError = "tool_error"
	// HeadlessErrorTypeFinalCheckFailed は headless final check 失敗の error type。
	HeadlessErrorTypeFinalCheckFailed = "final_check_failed"
	// HeadlessErrorTypeReadOnlyViolation は read-only mode の no-write 違反の error type。
	HeadlessErrorTypeReadOnlyViolation = "read_only_violation"
	// HeadlessErrorTypeUnsupportedCapability は要求 capability が provider/runtime で未対応な場合の error type。
	HeadlessErrorTypeUnsupportedCapability = "unsupported_capability"

	// HeadlessInputSourceArgs は positional args 由来の prompt input source。
	HeadlessInputSourceArgs HeadlessInputSource = "args"
	// HeadlessInputSourcePromptFile は --prompt-file 由来の prompt input source。
	HeadlessInputSourcePromptFile HeadlessInputSource = "prompt_file"
	// HeadlessInputSourceStdin は stdin 由来の prompt input source。
	HeadlessInputSourceStdin HeadlessInputSource = "stdin"
)

// HeadlessInputSource は headless prompt の入力元を表す。
type HeadlessInputSource string

// HeadlessInput は headless JSON に出す prompt input metadata を表す。
type HeadlessInput struct {
	Source     HeadlessInputSource `json:"source"`                // args, prompt_file, stdin
	PromptFile string              `json:"prompt_file,omitempty"` // --prompt-file の指定 path
	Bytes      int                 `json:"bytes"`                 // prompt body の byte 数
	Image      *HeadlessInputImage `json:"image,omitempty"`       // --image の metadata
}

// HeadlessInputImage は headless JSON に出す画像入力 metadata を表す。
type HeadlessInputImage struct {
	Path              string `json:"path"`                // --image の指定 path
	MIMEType          string `json:"mime_type,omitempty"` // image/png など
	Bytes             int64  `json:"bytes,omitempty"`     // 画像 file size
	ProviderSupported bool   `json:"provider_supported"`  // 選択 provider が画像入力対応か
}

// HeadlessRunOptions は headless 実行時の追加ポリシーを表す。
type HeadlessRunOptions struct {
	// FailOnToolError は tool_calls[].success=false を全体 failure に昇格する。
	FailOnToolError bool
	// ReadOnly は headless 実行で workspace mutation を拒否する no-write mode。
	ReadOnly bool
	// Image は初回 request に添付する画像入力。nil の場合は text-only。
	Image *api.ImageData
}

// HeadlessResult はHeadlessモードの実行結果
type HeadlessResult struct {
	SchemaVersion       string                `json:"schema_version"`                // headless JSON schema version
	Status              string                `json:"status"`                        // HeadlessStatusSuccess or HeadlessStatusError
	Provider            string                `json:"provider"`                      // LLMプロバイダー名
	Model               string                `json:"model"`                         // モデル名
	Response            string                `json:"response"`                      // AIの最終回答
	Input               *HeadlessInput        `json:"input,omitempty"`               // prompt input metadata
	FailureReason       HeadlessFailureReason `json:"failure_reason,omitempty"`      // CI 向け失敗分類
	ExitPolicy          HeadlessExitPolicy    `json:"exit_policy"`                   // exit code policy
	RecommendedExitCode int                   `json:"recommended_exit_code"`         // 推奨 process exit code
	Summary             *HeadlessSummary      `json:"summary,omitempty"`             // CI 向け runtime summary
	ToolCalls           []ToolCallResult      `json:"tool_calls,omitempty"`          // 実行されたツール呼び出し
	Tokens              *TokenUsage           `json:"tokens,omitempty"`              // トークン使用量
	WebSearch           *WebSearchUsage       `json:"web_search,omitempty"`          // ネイティブ Web 検索の固定料金観測
	DurationMs          int64                 `json:"duration_ms"`                   // 実行時間（ミリ秒）
	Timestamp           string                `json:"timestamp"`                     // タイムスタンプ（RFC3339）
	Error               *ErrorInfo            `json:"error,omitempty"`               // エラー情報
	Cost                float64               `json:"cost"`                          // 推定コスト（USD）
	PricingUnavailable  bool                  `json:"pricing_unavailable,omitempty"` // 既知の料金表がない場合 true
}

// HeadlessSummary は headless JSON に出す CI 向け runtime summary を表す。
type HeadlessSummary struct {
	ChangedFiles []string                    `json:"changed_files,omitempty"` // 変更が観測されたファイル
	Commands     []HeadlessCommandSummary    `json:"commands,omitempty"`      // tool 経由で実行されたコマンド
	FinalChecks  []HeadlessFinalCheckSummary `json:"final_checks,omitempty"`  // final_checks.commands の実行結果
}

// HeadlessCommandSummary は bash tool などで実行されたコマンドの要約を表す。
type HeadlessCommandSummary struct {
	Command  string `json:"command"`   // 実行されたコマンド
	ExitCode int    `json:"exit_code"` // 終了コード。不明な失敗は -1
	Status   string `json:"status"`    // passed or failed
	Source   string `json:"source"`    // tool
}

// HeadlessFinalCheckSummary は final_checks.commands の実行結果要約を表す。
type HeadlessFinalCheckSummary struct {
	Command  string `json:"command"`   // 実行された final check コマンド
	ExitCode int    `json:"exit_code"` // 終了コード。不明な失敗は -1
	Status   string `json:"status"`    // passed or failed
}

// ToolCallResult は個別のツール呼び出し結果
type ToolCallResult struct {
	Tool    string            `json:"tool"`    // ツール名
	Args    map[string]string `json:"args"`    // 引数
	Output  string            `json:"output"`  // 出力
	Success bool              `json:"success"` // 成功フラグ
}

// TokenUsage はトークン使用量
type TokenUsage struct {
	Input    int `json:"input"`    // 入力トークン数
	Cached   int `json:"cached"`   // キャッシュヒット入力トークン数
	Output   int `json:"output"`   // 出力トークン数
	Thinking int `json:"thinking"` // Thinking トークン数
	Total    int `json:"total"`    // 合計トークン数
}

// WebSearchUsage はネイティブ Web 検索の call fee と検索結果 token 観測を表す。
type WebSearchUsage struct {
	Calls        int     `json:"calls"`                   // built-in web search 呼び出し回数
	FeeEstimate  float64 `json:"fee_estimate"`            // 推定 call fee（USD）
	ResultTokens int     `json:"result_tokens,omitempty"` // provider が返した検索結果 token 観測値
}

// ErrorInfo はエラー情報
type ErrorInfo struct {
	Type    string `json:"type"`           // エラータイプ（api_error, tool_error, etc.）
	Message string `json:"message"`        // エラーメッセージ
	Code    int    `json:"code,omitempty"` // エラーコード（HTTPステータスなど）
}

// NewHeadlessInput は headless prompt input metadata を生成する。
func NewHeadlessInput(source HeadlessInputSource, promptFile string, byteCount int) HeadlessInput {
	if byteCount < 0 {
		byteCount = 0
	}
	if source != HeadlessInputSourcePromptFile {
		promptFile = ""
	}
	return HeadlessInput{
		Source:     source,
		PromptFile: promptFile,
		Bytes:      byteCount,
	}
}

// NewHeadlessInputImage は headless image input metadata を生成する。
func NewHeadlessInputImage(path string, mimeType string, byteCount int64, providerSupported bool) HeadlessInputImage {
	if byteCount < 0 {
		byteCount = 0
	}
	return HeadlessInputImage{
		Path:              path,
		MIMEType:          mimeType,
		Bytes:             byteCount,
		ProviderSupported: providerSupported,
	}
}

// NewHeadlessInputImageFromData は読み込み済み画像から headless image metadata を生成する。
func NewHeadlessInputImageFromData(image *api.ImageData, providerSupported bool) HeadlessInputImage {
	if image == nil {
		return NewHeadlessInputImage("", "", 0, providerSupported)
	}
	return NewHeadlessInputImage(image.Path, image.MediaType, image.Size, providerSupported)
}

// WithImage は HeadlessInput に画像入力 metadata を付与する。
func (i HeadlessInput) WithImage(image HeadlessInputImage) HeadlessInput {
	i.Image = &image
	return i
}

// WithInput は HeadlessResult に prompt input metadata を付与する。
func (r *HeadlessResult) WithInput(input HeadlessInput) *HeadlessResult {
	if r == nil {
		return nil
	}
	r.Input = &input
	return r
}

// ToJSON は HeadlessResult を JSON 文字列に変換
func (r *HeadlessResult) ToJSON() (string, error) {
	normalized, err := r.normalizedForJSON()
	if err != nil {
		return "", err
	}
	bytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r *HeadlessResult) normalizedForJSON() (HeadlessResult, error) {
	if r == nil {
		result := HeadlessResult{SchemaVersion: HeadlessSchemaVersion}
		result.setExitPolicy(HeadlessExitPolicyLegacy)
		return result, nil
	}
	normalized := *r
	if normalized.SchemaVersion == "" {
		normalized.SchemaVersion = HeadlessSchemaVersion
	}
	policy, err := ParseHeadlessExitPolicy(string(normalized.ExitPolicy))
	if err != nil {
		return HeadlessResult{}, err
	}
	normalized.setExitPolicy(policy)
	return normalized, nil
}

// NewSuccessResult は成功結果を生成
func NewSuccessResult(provider, model, response string, toolCalls []ToolCallResult, durationMs int64) *HeadlessResult {
	result := &HeadlessResult{
		SchemaVersion: HeadlessSchemaVersion,
		Status:        HeadlessStatusSuccess,
		Provider:      provider,
		Model:         model,
		Response:      response,
		ToolCalls:     toolCalls,
		DurationMs:    durationMs,
		Timestamp:     time.Now().Format(time.RFC3339),
		ExitPolicy:    HeadlessExitPolicyLegacy,
	}
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

// NewErrorResult はエラー結果を生成
func NewErrorResult(provider, model string, errType, errMsg string, durationMs int64) *HeadlessResult {
	result := &HeadlessResult{
		SchemaVersion: HeadlessSchemaVersion,
		Status:        HeadlessStatusError,
		Provider:      provider,
		Model:         model,
		Response:      "",
		DurationMs:    durationMs,
		Timestamp:     time.Now().Format(time.RFC3339),
		Error: &ErrorInfo{
			Type:    errType,
			Message: errMsg,
		},
		ExitPolicy: HeadlessExitPolicyLegacy,
	}
	result.FailureReason = result.normalizedFailureReason()
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

func promoteHeadlessToolErrorResult(result *HeadlessResult) *HeadlessResult {
	if result == nil {
		return nil
	}
	result.Status = HeadlessStatusError
	result.Error = &ErrorInfo{
		Type:    HeadlessErrorTypeToolError,
		Message: "one or more tool calls failed",
	}
	result.FailureReason = HeadlessFailureReasonToolError
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

func promoteHeadlessCancelledResult(result *HeadlessResult, err error) *HeadlessResult {
	if result == nil {
		return nil
	}
	message := "context canceled"
	if err != nil {
		message = err.Error()
	}
	result.Status = HeadlessStatusError
	result.Error = &ErrorInfo{
		Type:    HeadlessErrorTypeCancelled,
		Message: message,
	}
	result.FailureReason = HeadlessFailureReasonCancelled
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

func promoteHeadlessFinalCheckFailedResult(result *HeadlessResult) *HeadlessResult {
	if result == nil {
		return nil
	}
	result.Status = HeadlessStatusError
	result.Error = &ErrorInfo{
		Type:    HeadlessErrorTypeFinalCheckFailed,
		Message: "one or more final check commands failed",
	}
	result.FailureReason = HeadlessFailureReasonFinalCheckFailed
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

func promoteHeadlessReadOnlyViolationResult(result *HeadlessResult) *HeadlessResult {
	if result == nil {
		return nil
	}
	result.Status = HeadlessStatusError
	result.Error = &ErrorInfo{
		Type:    HeadlessErrorTypeReadOnlyViolation,
		Message: "one or more tool calls were denied by read-only mode",
	}
	result.FailureReason = HeadlessFailureReasonReadOnlyViolation
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

// NewUsageErrorResult は CLI 入力 validation 用の headless JSON error を生成する。
func NewUsageErrorResult(provider, model, errMsg string, durationMs int64) *HeadlessResult {
	result := NewErrorResult(provider, model, HeadlessErrorTypeConfig, errMsg, durationMs)
	result.FailureReason = HeadlessFailureReasonUsageError
	result.RecommendedExitCode = RecommendedHeadlessExitCode(result.Status, result.FailureReason, result.ExitPolicy)
	return result
}

// ApplyHeadlessExitPolicy は HeadlessResult に exit policy と推奨 exit code を反映する。
func ApplyHeadlessExitPolicy(result *HeadlessResult, policy HeadlessExitPolicy) (*HeadlessResult, error) {
	normalizedPolicy, err := ParseHeadlessExitPolicy(string(policy))
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &HeadlessResult{SchemaVersion: HeadlessSchemaVersion}
	}
	result.setExitPolicy(normalizedPolicy)
	return result, nil
}

// NewToolLoopLimitResult は headless tool loop limit 到達時の結果を生成する。
func NewToolLoopLimitResult(provider, model string, limit int, toolCalls []ToolCallResult, durationMs int64) *HeadlessResult {
	result := NewErrorResult(provider, model, HeadlessErrorTypeToolLoopLimit, HeadlessToolLoopLimitMessage(limit), durationMs)
	result.ToolCalls = toolCalls
	return result
}

// HeadlessToolLoopLimitMessage は tool loop limit 到達時のユーザー向け error message を返す。
func HeadlessToolLoopLimitMessage(limit int) string {
	return fmt.Sprintf("tool loop limit reached (%d iterations)", limit)
}
