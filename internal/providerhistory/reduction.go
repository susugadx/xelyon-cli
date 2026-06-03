package providerhistory

import "github.com/susugadx/xelyon-cli/internal/taskstate"

// Mode は provider-facing history reduction の動作を表す。
type Mode int

const (
	// Disabled は Phase 5b の no-op projection を維持する。
	Disabled Mode = iota
	// DryRun は provider payload を変えずに候補だけを記録する。
	DryRun
	// Apply は projection clone 上で安全な候補だけを置換する。
	Apply
	// Auto は現時点では dry-run 相当の安全側実効 mode。
	Auto
)

const (
	providerHistoryReplacementStatusNotImplemented = "not_implemented"
	providerHistoryReplacementStatusApply          = "apply"
	providerHistoryReplacementStatusPartialApply   = "partial_apply"
)

// Policy は provider-facing reduction の方針を選ぶ。
type Policy struct {
	Mode                                   Mode
	EvidencePointers                       []taskstate.EvidencePointer
	EvidenceReductionRequiresActiveContext bool
	ActiveContextTransportAvailable        bool
}

// ReductionCandidate は dry-run detector が評価した tool result を表す。
type ReductionCandidate struct {
	HistoryIndex             int
	Role                     string
	ToolName                 string
	ToolCallID               string
	EvidencePointers         []taskstate.EvidencePointer
	OriginalByteSize         int
	OriginalRuneSize         int
	Reason                   string
	SuggestedReplacementKind string
	SuggestedReplacementText string
	KeepReason               string
	ReplacementApplied       bool
}

func cloneReductionCandidate(candidate ReductionCandidate) ReductionCandidate {
	candidate.EvidencePointers = cloneProviderHistoryReductionEvidencePointers(candidate.EvidencePointers)
	return candidate
}

func cloneProviderHistoryReductionEvidencePointers(pointers []taskstate.EvidencePointer) []taskstate.EvidencePointer {
	if len(pointers) == 0 {
		return nil
	}
	cloned := make([]taskstate.EvidencePointer, len(pointers))
	copy(cloned, pointers)
	return cloned
}

// CommandEditDryRunCandidate は command/edit 系の将来置換候補診断を表す。
type CommandEditDryRunCandidate struct {
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

// CommandEditDryRunReport は command/edit 系 dry-run 診断を表す。
type CommandEditDryRunReport struct {
	ReplacementStatus                   string
	CommandCandidates                   int
	EditArgCandidates                   int
	CommandOriginalBytes                int
	EditArgOriginalBytes                int
	CommandEstimatedSavedBytes          int
	EditArgEstimatedSavedBytes          int
	CommandReplacedCount                int
	EditArgReplacedCount                int
	CommandReplacementSavedBytes        int
	EditArgReplacementSavedBytes        int
	ApproxCommandSavedTokens            int
	ApproxCommandReplacementSavedTokens int
	ApproxEditArgSavedTokens            int
	ApproxEditArgReplacementSavedTokens int
	CandidateReasonCounts               map[string]int
	KeptReasonCounts                    map[string]int
	Candidates                          []CommandEditDryRunCandidate
	Kept                                []CommandEditDryRunCandidate
}

// ProjectionReport は provider-facing projection の構築結果を要約する。
type ProjectionReport struct {
	Mode                                Mode
	OriginalMessageCount                int
	ProjectedMessageCount               int
	ToolResultCount                     int
	CandidateCount                      int
	KeptCount                           int
	ReplacedCount                       int
	OriginalBytes                       int
	ProjectedBytes                      int
	EstimatedSavedBytes                 int
	ApproxSavedTokens                   int
	ContentReplacementSavedBytes        int
	ApproxContentReplacementSavedTokens int
	ReplacementStatus                   string
	KeptReasonCounts                    map[string]int
	ResponsesChainDisabled              bool
	Candidates                          []ReductionCandidate
	Kept                                []ReductionCandidate
	CommandEditDryRun                   CommandEditDryRunReport
}

func normalizePolicy(policy Policy) Policy {
	switch policy.Mode {
	case DryRun, Apply:
	case Auto:
		policy.Mode = DryRun
	default:
		policy.Mode = Disabled
	}
	policy.EvidencePointers = cloneProviderHistoryReductionEvidencePointers(policy.EvidencePointers)
	return policy
}
