package agent

import "github.com/susugadx/xelyon-cli/internal/ledger"

// ProviderHistoryReductionMode は provider-facing history reduction の動作を表す。
type ProviderHistoryReductionMode int

const (
	// ProviderHistoryReductionDisabled は Phase 5b の no-op projection を維持する。
	ProviderHistoryReductionDisabled ProviderHistoryReductionMode = iota
	// ProviderHistoryReductionDryRun は provider payload を変えずに候補だけを記録する。
	ProviderHistoryReductionDryRun
	// ProviderHistoryReductionApply は projection clone 上で安全な候補だけを置換する。
	ProviderHistoryReductionApply
	// ProviderHistoryReductionAuto は現時点では dry-run 相当の安全側実効 mode。
	ProviderHistoryReductionAuto
)

// ProviderHistoryReductionPolicy は provider-facing reduction の方針を選ぶ。
type ProviderHistoryReductionPolicy struct {
	Mode ProviderHistoryReductionMode
}

// ProviderHistoryReductionCandidate は dry-run detector が評価した tool result を表す。
type ProviderHistoryReductionCandidate struct {
	HistoryIndex             int
	Role                     string
	ToolName                 string
	ToolCallID               string
	EvidencePointers         []ledger.EvidencePointer
	OriginalByteSize         int
	OriginalRuneSize         int
	Reason                   string
	SuggestedReplacementKind string
	SuggestedReplacementText string
	KeepReason               string
	ReplacementApplied       bool
}

func cloneProviderHistoryReductionCandidate(candidate ProviderHistoryReductionCandidate) ProviderHistoryReductionCandidate {
	candidate.EvidencePointers = cloneProviderHistoryReductionEvidencePointers(candidate.EvidencePointers)
	return candidate
}

func cloneProviderHistoryReductionEvidencePointers(pointers []ledger.EvidencePointer) []ledger.EvidencePointer {
	if len(pointers) == 0 {
		return nil
	}
	cloned := make([]ledger.EvidencePointer, len(pointers))
	copy(cloned, pointers)
	return cloned
}

// ProviderHistoryCommandEditDryRunCandidate は command/edit 系の将来置換候補診断を表す。
type ProviderHistoryCommandEditDryRunCandidate struct {
	HistoryIndex         int
	Role                 string
	ToolName             string
	ToolCallID           string
	Kind                 string
	OriginalByteSize     int
	OriginalRuneSize     int
	ApproxOriginalTokens int
	Reason               string
	KeepReason           string
}

// ProviderHistoryCommandEditDryRunReport は command/edit 系 dry-run 診断を表す。
type ProviderHistoryCommandEditDryRunReport struct {
	ReplacementStatus                   string
	CommandCandidates                   int
	EditArgCandidates                   int
	CommandOriginalBytes                int
	EditArgOriginalBytes                int
	CommandReplacedCount                int
	CommandReplacementSavedBytes        int
	ApproxCommandSavedTokens            int
	ApproxCommandReplacementSavedTokens int
	ApproxEditArgSavedTokens            int
	CandidateReasonCounts               map[string]int
	KeptReasonCounts                    map[string]int
	Candidates                          []ProviderHistoryCommandEditDryRunCandidate
	Kept                                []ProviderHistoryCommandEditDryRunCandidate
}

// ProviderHistoryProjectionReport は provider-facing projection の構築結果を要約する。
type ProviderHistoryProjectionReport struct {
	Mode                   ProviderHistoryReductionMode
	OriginalMessageCount   int
	ProjectedMessageCount  int
	ToolResultCount        int
	CandidateCount         int
	KeptCount              int
	ReplacedCount          int
	OriginalBytes          int
	ProjectedBytes         int
	EstimatedSavedBytes    int
	ApproxSavedTokens      int
	KeptReasonCounts       map[string]int
	ResponsesChainDisabled bool
	Candidates             []ProviderHistoryReductionCandidate
	Kept                   []ProviderHistoryReductionCandidate
	CommandEditDryRun      ProviderHistoryCommandEditDryRunReport
}

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
