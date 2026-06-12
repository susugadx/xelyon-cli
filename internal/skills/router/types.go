package router

import skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"

const (
	// ScoreHighMin は high confidence の下限。
	ScoreHighMin = 80
	// ScoreMediumMin は medium confidence の下限。
	ScoreMediumMin = 50
	// ScoreLowMin は low confidence の下限。
	ScoreLowMin = 25
)

// CandidateCategory は router が判断した skill の turn 内カテゴリ。
type CandidateCategory string

const (
	CategoryPrimary    CandidateCategory = "primary"
	CategorySupporting CandidateCategory = "supporting"
	CategoryMaybe      CandidateCategory = "maybe"
	CategoryConflict   CandidateCategory = "conflict"
	CategoryNone       CandidateCategory = "none"
)

// ConfidenceBand は score を v1 の band に丸めた値。
type ConfidenceBand string

const (
	ConfidenceHigh   ConfidenceBand = "high"
	ConfidenceMedium ConfidenceBand = "medium"
	ConfidenceLow    ConfidenceBand = "low"
	ConfidenceNone   ConfidenceBand = "none"
)

// Input は pure router が使う正規化前の signal 入力。
type Input struct {
	TaskText              string
	Command               string
	RequestedMode         string
	ReadOnly              bool
	TouchedPaths          []string
	SignalDiagnostics     []string
	PromptCatalogMaxItems int
}

// Candidate は router が評価した skill 候補。
type Candidate struct {
	Name           string
	Description    string
	Source         skillcatalog.Source
	Role           skillcatalog.RoutingRole
	ReadOnly       bool
	Activation     skillcatalog.RoutingActivation
	Category       CandidateCategory
	Score          int
	Confidence     ConfidenceBand
	MatchedSignals []string
	Reason         string
	ConflictReason string
}

// Recommendation は full ranked list と runtime hint 用カテゴリを保持する。
type Recommendation struct {
	Ranked            []Candidate
	Primary           []Candidate
	Supporting        []Candidate
	Maybe             []Candidate
	Conflicts         []Candidate
	SignalDiagnostics []string
}

// HintLimits は runtime prompt hint に出すカテゴリ別上限。
type HintLimits struct {
	Primary    int
	Supporting int
	Conflict   int
	Maybe      int
}

// DefaultRuntimeHintLimits は v1 runtime hint のカテゴリ別上限。
func DefaultRuntimeHintLimits() HintLimits {
	return HintLimits{
		Primary:    2,
		Supporting: 5,
		Conflict:   5,
		Maybe:      0,
	}
}

// ConfidenceForScore は v1 score band を返す。
func ConfidenceForScore(score int) ConfidenceBand {
	switch {
	case score >= ScoreHighMin:
		return ConfidenceHigh
	case score >= ScoreMediumMin:
		return ConfidenceMedium
	case score >= ScoreLowMin:
		return ConfidenceLow
	default:
		return ConfidenceNone
	}
}
