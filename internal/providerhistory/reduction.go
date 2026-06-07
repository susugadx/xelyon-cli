package providerhistory

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

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

// RawOutputArtifactsMode は data-bearing raw output artifact-backed compact の providerhistory policy mode。
type RawOutputArtifactsMode int

const (
	// RawOutputArtifactsDisabled は raw output artifact-backed 候補生成を無効化する。
	RawOutputArtifactsDisabled RawOutputArtifactsMode = iota
	// RawOutputArtifactsDryRun は raw output artifact-backed 候補を report し payload は変えない。
	RawOutputArtifactsDryRun
	// RawOutputArtifactsApply は全 gate 成立時だけ artifact-backed replacement を許可する。
	RawOutputArtifactsApply
)

// ProjectionSideEffectMode は projection 中に副作用を許可するかを表す。
type ProjectionSideEffectMode int

const (
	// ProjectionSideEffectsAllow は provider request / review apply 用に必要な副作用を許可する。
	ProjectionSideEffectsAllow ProjectionSideEffectMode = iota
	// ProjectionSideEffectsReadOnly は token estimate / status などの read-only 経路で副作用を禁止する。
	ProjectionSideEffectsReadOnly
)

// RawOutputArtifactStore は providerhistory が使う rawoutputs store の最小 surface。
type RawOutputArtifactStore interface {
	Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error)
	Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error)
}

const (
	providerHistoryReplacementStatusNotImplemented  = "not_implemented"
	providerHistoryReplacementStatusApply           = "apply"
	providerHistoryReplacementStatusPartialApply    = "partial_apply"
	providerHistoryContentReplacementMinSavedTokens = 128
)

// Policy は provider-facing reduction の方針を選ぶ。
type Policy struct {
	Mode                                   Mode
	EvidencePointers                       []taskstate.EvidencePointer
	EvidenceReductionRequiresActiveContext bool
	ActiveContextTransportAvailable        bool
	RawOutputArtifactsMode                 RawOutputArtifactsMode
	RawOutputArtifactStore                 RawOutputArtifactStore
	SessionID                              string
	RawOutputRehydrateContextEnabled       bool
	RawOutputApplyDisabledReason           string
	SideEffects                            ProjectionSideEffectMode
}

// ReductionCandidate は dry-run detector が評価した tool result を表す。
type ReductionCandidate struct {
	HistoryIndex                          int
	Role                                  string
	ToolName                              string
	ToolCallID                            string
	CandidateOnly                         bool
	FutureApplyCandidate                  bool
	EvidencePointers                      []taskstate.EvidencePointer
	OriginalByteSize                      int
	OriginalRuneSize                      int
	Reason                                string
	SuggestedReplacementKind              string
	SuggestedReplacementText              string
	EstimatedSavedBytes                   int
	ApproxEstimatedSavedTokens            int
	KeepReason                            string
	ReplacementApplied                    bool
	RawOutputRefID                        string
	ArtifactBackedCandidate               bool
	ArtifactBackedApplyEligible           bool
	ArtifactGateStatus                    string
	RehydrateGateStatus                   string
	ThresholdStatus                       string
	SafetyStatus                          string
	FailClosedReason                      string
	ArtifactBackedActualSavedBytes        int
	ApproxArtifactBackedActualSavedTokens int
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
	HistoryIndex                int
	Role                        string
	ToolName                    string
	ToolCallID                  string
	Kind                        string
	OriginalByteSize            int
	OriginalRuneSize            int
	ApproxOriginalTokens        int
	Reason                      string
	Classifier                  string
	SuggestedReplacementKind    string
	SuggestedReplacementText    string
	EstimatedSavedBytes         int
	ApproxEstimatedSavedTokens  int
	ReplacementEligible         bool
	ReplacementApplied          bool
	KeepReason                  string
	RawOutputRefID              string
	ArtifactBackedCandidate     bool
	ArtifactBackedApplyEligible bool
	ArtifactGateStatus          string
	RehydrateGateStatus         string
	ThresholdStatus             string
	FreshnessStatus             string
	SafetyStatus                string
	FailClosedReason            string
}

// CommandEditDryRunReport は command/edit 系 dry-run 診断を表す。
type CommandEditDryRunReport struct {
	ReplacementStatus                                     string
	CommandCandidates                                     int
	EditArgCandidates                                     int
	CommandOriginalBytes                                  int
	EditArgOriginalBytes                                  int
	CommandEstimatedSavedBytes                            int
	EditArgEstimatedSavedBytes                            int
	CommandReplacedCount                                  int
	EditArgReplacedCount                                  int
	CommandReplacementSavedBytes                          int
	EditArgReplacementSavedBytes                          int
	ApproxCommandSavedTokens                              int
	ApproxCommandReplacementSavedTokens                   int
	ApproxEditArgSavedTokens                              int
	ApproxEditArgReplacementSavedTokens                   int
	CandidateReasonCounts                                 map[string]int
	CommandReplacementClassifierCounts                    map[string]int
	KeptReasonCounts                                      map[string]int
	RawOutputRefs                                         []rawoutputs.RawOutputRef
	ArtifactBackedCommandCandidates                       int
	ArtifactBackedCommandApplyEligible                    int
	ArtifactBackedCommandReplacedCount                    int
	ArtifactBackedCommandDryRunEstimatedSavedBytes        int
	ApproxArtifactBackedCommandDryRunEstimatedSavedTokens int
	ArtifactBackedCommandReplacementSavedBytes            int
	ApproxArtifactBackedCommandReplacementSavedTokens     int
	ArtifactBackedKeptReasonCounts                        map[string]int
	Candidates                                            []CommandEditDryRunCandidate
	Kept                                                  []CommandEditDryRunCandidate
}

// ProjectionReport は provider-facing projection の構築結果を要約する。
type ProjectionReport struct {
	Mode                                     Mode
	OriginalMessageCount                     int
	ProjectedMessageCount                    int
	ToolResultCount                          int
	CandidateCount                           int
	KeptCount                                int
	ReplacedCount                            int
	OriginalBytes                            int
	ProjectedBytes                           int
	EstimatedSavedBytes                      int
	ApproxSavedTokens                        int
	ContentReplacementSavedBytes             int
	ApproxContentReplacementSavedTokens      int
	ContentReplacementToolCounts             map[string]int
	SkillReplacementToolCounts               map[string]int
	FutureFamilyCandidateCounts              map[string]int
	FutureFamilyKeptReasonCounts             map[string]int
	ReplacementStatus                        string
	KeptReasonCounts                         map[string]int
	ResponsesChainDisabled                   bool
	RawOutputRefs                            []rawoutputs.RawOutputRef
	RawOutputRefCount                        int
	RawOutputArtifactCount                   int
	DataBearingCandidateCount                int
	ArtifactBackedEstimatedSavedBytes        int
	ApproxArtifactBackedEstimatedSavedTokens int
	ArtifactBackedActualSavedBytes           int
	ApproxArtifactBackedActualSavedTokens    int
	Candidates                               []ReductionCandidate
	Kept                                     []ReductionCandidate
	CommandEditDryRun                        CommandEditDryRunReport
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
	if policy.Mode == Disabled {
		policy.RawOutputArtifactsMode = RawOutputArtifactsDisabled
	}
	switch policy.SideEffects {
	case ProjectionSideEffectsReadOnly:
	default:
		policy.SideEffects = ProjectionSideEffectsAllow
	}
	return policy
}

func rawOutputArtifactMaterializationAllowed(policy Policy) bool {
	return policy.SideEffects != ProjectionSideEffectsReadOnly
}
