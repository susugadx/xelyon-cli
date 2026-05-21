package agent

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
	OriginalByteSize         int
	OriginalRuneSize         int
	Reason                   string
	SuggestedReplacementKind string
	SuggestedReplacementText string
	KeepReason               string
	ReplacementApplied       bool
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
