package agent

import "github.com/susugadx/xelyon-cli/internal/providerhistory"

// ProviderHistoryReductionMode は provider-facing history reduction の動作を表す。
type ProviderHistoryReductionMode = providerhistory.Mode

const (
	// ProviderHistoryReductionDisabled は provider-facing history reduction を無効化する。
	ProviderHistoryReductionDisabled = providerhistory.Disabled
	// ProviderHistoryReductionDryRun は provider payload を変えずに候補だけを記録する。
	ProviderHistoryReductionDryRun = providerhistory.DryRun
	// ProviderHistoryReductionApply は projection clone 上で安全な候補だけを置換する。
	ProviderHistoryReductionApply = providerhistory.Apply
	// ProviderHistoryReductionAuto は現時点では dry-run 相当の安全側実効 mode。
	ProviderHistoryReductionAuto = providerhistory.Auto
)

// ProviderHistoryReductionPolicy は provider-facing reduction の方針を選ぶ。
type ProviderHistoryReductionPolicy = providerhistory.Policy

// ProviderHistoryReductionCandidate は dry-run detector が評価した tool result を表す。
type ProviderHistoryReductionCandidate = providerhistory.ReductionCandidate

// ProviderHistoryCommandEditDryRunCandidate は command/edit 系の将来置換候補診断を表す。
type ProviderHistoryCommandEditDryRunCandidate = providerhistory.CommandEditDryRunCandidate

// ProviderHistoryCommandEditDryRunReport は command/edit 系 dry-run 診断を表す。
type ProviderHistoryCommandEditDryRunReport = providerhistory.CommandEditDryRunReport

// ProviderHistoryProjectionReport は provider-facing projection の構築結果を要約する。
type ProviderHistoryProjectionReport = providerhistory.ProjectionReport

const (
	providerHistoryCommandEditReplacementStatusNotImplemented = "not_implemented"
	providerHistoryReplacementStatusApply                     = "apply"
	providerHistoryCommandEditReplacementStatusPartialApply   = "partial_apply"
)

func normalizeProviderHistoryReductionPolicy(policy ProviderHistoryReductionPolicy) ProviderHistoryReductionPolicy {
	switch policy.Mode {
	case ProviderHistoryReductionDryRun, ProviderHistoryReductionApply:
	case ProviderHistoryReductionAuto:
		policy.Mode = ProviderHistoryReductionDryRun
	default:
		policy.Mode = ProviderHistoryReductionDisabled
	}
	return policy
}

func newProviderHistoryCommandEditDryRunReport() ProviderHistoryCommandEditDryRunReport {
	return ProviderHistoryCommandEditDryRunReport{
		ReplacementStatus: providerHistoryCommandEditReplacementStatusNotImplemented,
	}
}
