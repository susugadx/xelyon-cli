package agent

import (
	"fmt"
	"strings"
)

// HeadlessFailureReason は headless JSON の CI 向け失敗分類を表す。
type HeadlessFailureReason string

const (
	// HeadlessFailureReasonUsageError は CLI の使い方や入力 validation の失敗を表す。
	HeadlessFailureReasonUsageError HeadlessFailureReason = "usage_error"
	// HeadlessFailureReasonConfigError は設定や初期化の失敗を表す。
	HeadlessFailureReasonConfigError HeadlessFailureReason = "config_error"
	// HeadlessFailureReasonProviderSetupRequired は provider credential setup 未完了を表す。
	HeadlessFailureReasonProviderSetupRequired HeadlessFailureReason = "provider_setup_required"
	// HeadlessFailureReasonToolError は tool 実行失敗を表す。
	HeadlessFailureReasonToolError HeadlessFailureReason = "tool_error"
	// HeadlessFailureReasonFinalCheckFailed は final check 失敗を表す。
	HeadlessFailureReasonFinalCheckFailed HeadlessFailureReason = "final_check_failed"
	// HeadlessFailureReasonAPIError は provider API request 失敗を表す。
	HeadlessFailureReasonAPIError HeadlessFailureReason = "api_error"
	// HeadlessFailureReasonCancelled は context cancel / timeout を表す。
	HeadlessFailureReasonCancelled HeadlessFailureReason = "cancelled"
	// HeadlessFailureReasonReadOnlyViolation は read-only 違反を表す。
	HeadlessFailureReasonReadOnlyViolation HeadlessFailureReason = "read_only_violation"
	// HeadlessFailureReasonUnsupportedCapability は未対応 capability を表す。
	HeadlessFailureReasonUnsupportedCapability HeadlessFailureReason = "unsupported_capability"
	// HeadlessFailureReasonToolLoopLimit は tool loop limit 到達を表す。
	HeadlessFailureReasonToolLoopLimit HeadlessFailureReason = "tool_loop_limit"
	// HeadlessFailureReasonUnknownError は未分類の失敗を表す。
	HeadlessFailureReasonUnknownError HeadlessFailureReason = "unknown_error"
)

// HeadlessExitPolicy は headless JSON の推奨 exit code policy を表す。
type HeadlessExitPolicy string

const (
	// HeadlessExitPolicyLegacy は既存互換の non-zero error code policy。
	HeadlessExitPolicyLegacy HeadlessExitPolicy = "legacy"
	// HeadlessExitPolicyCI は CI 向けの詳細 exit code policy。
	HeadlessExitPolicyCI HeadlessExitPolicy = "ci"
)

// ParseHeadlessExitPolicy は CLI flag 値を HeadlessExitPolicy に変換する。
func ParseHeadlessExitPolicy(value string) (HeadlessExitPolicy, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return HeadlessExitPolicyLegacy, nil
	}
	switch HeadlessExitPolicy(normalized) {
	case HeadlessExitPolicyLegacy:
		return HeadlessExitPolicyLegacy, nil
	case HeadlessExitPolicyCI:
		return HeadlessExitPolicyCI, nil
	default:
		return "", fmt.Errorf("invalid --exit-code-policy %q (want legacy or ci)", value)
	}
}

// HeadlessFailureReasonForErrorType は既存互換の error.type から CI 向け失敗分類を返す。
func HeadlessFailureReasonForErrorType(errType string) HeadlessFailureReason {
	switch errType {
	case HeadlessErrorTypeConfig:
		return HeadlessFailureReasonConfigError
	case HeadlessErrorTypeProviderSetupRequired:
		return HeadlessFailureReasonProviderSetupRequired
	case HeadlessErrorTypeAPI:
		return HeadlessFailureReasonAPIError
	case HeadlessErrorTypeCancelled:
		return HeadlessFailureReasonCancelled
	case HeadlessErrorTypeToolLoopLimit:
		return HeadlessFailureReasonToolLoopLimit
	case HeadlessErrorTypeToolError:
		return HeadlessFailureReasonToolError
	case HeadlessErrorTypeFinalCheckFailed:
		return HeadlessFailureReasonFinalCheckFailed
	case HeadlessErrorTypeReadOnlyViolation:
		return HeadlessFailureReasonReadOnlyViolation
	case string(HeadlessFailureReasonUsageError):
		return HeadlessFailureReasonUsageError
	case HeadlessErrorTypeUnsupportedCapability:
		return HeadlessFailureReasonUnsupportedCapability
	default:
		return HeadlessFailureReasonUnknownError
	}
}

// RecommendedHeadlessExitCode は headless result の推奨 process exit code を返す。
func RecommendedHeadlessExitCode(status string, reason HeadlessFailureReason, policy HeadlessExitPolicy) int {
	if status != HeadlessStatusError {
		return 0
	}
	if policy != HeadlessExitPolicyCI {
		return 1
	}

	switch reason {
	case HeadlessFailureReasonUsageError:
		return 2
	case HeadlessFailureReasonConfigError, HeadlessFailureReasonProviderSetupRequired:
		return 3
	case HeadlessFailureReasonToolError:
		return 4
	case HeadlessFailureReasonFinalCheckFailed:
		return 5
	case HeadlessFailureReasonAPIError:
		return 6
	case HeadlessFailureReasonCancelled:
		return 7
	case HeadlessFailureReasonReadOnlyViolation:
		return 8
	case HeadlessFailureReasonUnsupportedCapability:
		return 9
	case HeadlessFailureReasonToolLoopLimit, HeadlessFailureReasonUnknownError:
		return 1
	default:
		return 1
	}
}

func (r HeadlessResult) normalizedFailureReason() HeadlessFailureReason {
	if r.Status != HeadlessStatusError {
		return ""
	}
	if r.FailureReason != "" {
		return r.FailureReason
	}
	if r.Error == nil {
		return HeadlessFailureReasonUnknownError
	}
	return HeadlessFailureReasonForErrorType(r.Error.Type)
}

func (r *HeadlessResult) setExitPolicy(policy HeadlessExitPolicy) {
	if r == nil {
		return
	}
	r.ExitPolicy = policy
	r.FailureReason = r.normalizedFailureReason()
	r.RecommendedExitCode = RecommendedHeadlessExitCode(r.Status, r.FailureReason, policy)
}
