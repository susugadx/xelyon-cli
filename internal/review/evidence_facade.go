package review

import reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"

// ReviewEvidenceBundle は `/review current_changes` の Pass1 入力に渡す一次情報を表す。
type ReviewEvidenceBundle = reviewevidence.ReviewEvidenceBundle

// ReviewDiffEvidence は staged / unstaged それぞれの diff 一次情報を保持する。
type ReviewDiffEvidence = reviewevidence.ReviewDiffEvidence

// ReviewChangedFile は git diff --name-status 由来の変更 file を表す。
type ReviewChangedFile = reviewevidence.ReviewChangedFile

// ReviewContextFileEvidence は changed file と近傍 context file の安全に制限された snapshot を表す。
type ReviewContextFileEvidence = reviewevidence.ReviewContextFileEvidence

// ReviewRelatedSearchHit は changed file 由来の軽量 search term に一致した repo-local 行を表す。
type ReviewRelatedSearchHit = reviewevidence.ReviewRelatedSearchHit

// ReviewGenericImpactCandidates は言語非依存 heuristic で広げた review 用 impact 候補を表す。
type ReviewGenericImpactCandidates = reviewevidence.ReviewGenericImpactCandidates

// ReviewGenericImpactCandidate は generic impact expansion が検出した 1 候補を表す。
type ReviewGenericImpactCandidate = reviewevidence.ReviewGenericImpactCandidate

// ReviewUntrackedFile は untracked path の安全に制限された snapshot または symlink metadata を表す。
type ReviewUntrackedFile = reviewevidence.ReviewUntrackedFile

// ReviewRuleFileEvidence は review 方針に影響する repo-local rule file を表す。
type ReviewRuleFileEvidence = reviewevidence.ReviewRuleFileEvidence

// ReviewChangeInventory は変更 surface を分類した一覧を表す。
type ReviewChangeInventory = reviewevidence.ReviewChangeInventory

// ReviewEvidenceLimits は EvidenceBuilder の resource budget を表す。
type ReviewEvidenceLimits = reviewevidence.ReviewEvidenceLimits

// ReviewEvidenceBuilderOption は ReviewEvidenceBuilder の差し替え設定を表す。
type ReviewEvidenceBuilderOption = reviewevidence.ReviewEvidenceBuilderOption

// ReviewWebSearchEvidenceProvider は ReviewEvidenceBuilder が使う外部 Web 検索 evidence 境界。
type ReviewWebSearchEvidenceProvider = reviewevidence.ReviewWebSearchEvidenceProvider

// ReviewPostPass1WebSearchEvidenceProvider は Pass1 probe plan 後の追加 Web 検索 evidence 収集境界。
type ReviewPostPass1WebSearchEvidenceProvider = reviewevidence.ReviewPostPass1WebSearchEvidenceProvider

// ReviewEvidenceBuilder は current_changes の一次情報を git と filesystem から収集する。
type ReviewEvidenceBuilder = reviewevidence.ReviewEvidenceBuilder

// ReviewEvidenceCommandRunner は EvidenceBuilder が使う git command 実行境界を表す。
type ReviewEvidenceCommandRunner = reviewevidence.ReviewEvidenceCommandRunner

// ReviewEvidenceModelInput は ReviewEvidenceBundle を LLM 入力向けに正規化した DTO。
type ReviewEvidenceModelInput = reviewevidence.ReviewEvidenceModelInput

// ReviewEvidenceTextBlock は本文と truncation flag を組にした DTO。
type ReviewEvidenceTextBlock = reviewevidence.ReviewEvidenceTextBlock

// ReviewEvidenceChangedFileInput は changed file の path 表示を正規化した DTO。
type ReviewEvidenceChangedFileInput = reviewevidence.ReviewEvidenceChangedFileInput

// ReviewEvidenceContextFileInput は context file evidence を LLM 入力向けに表す。
type ReviewEvidenceContextFileInput = reviewevidence.ReviewEvidenceContextFileInput

// ReviewEvidenceRelatedSearchHitInput は search hit evidence を LLM 入力向けに表す。
type ReviewEvidenceRelatedSearchHitInput = reviewevidence.ReviewEvidenceRelatedSearchHitInput

// ReviewEvidenceGenericImpactInput は generic impact candidates の LLM 入力表現。
type ReviewEvidenceGenericImpactInput = reviewevidence.ReviewEvidenceGenericImpactInput

// ReviewEvidenceGenericImpactCandidateInput は generic impact candidate の LLM 入力表現。
type ReviewEvidenceGenericImpactCandidateInput = reviewevidence.ReviewEvidenceGenericImpactCandidateInput

// ReviewEvidenceChangeInventoryInput は変更 surface の一覧を LLM 入力向けに表す。
type ReviewEvidenceChangeInventoryInput = reviewevidence.ReviewEvidenceChangeInventoryInput

// ReviewEvidenceRuleFileInput は rule file evidence を LLM 入力向けに表す。
type ReviewEvidenceRuleFileInput = reviewevidence.ReviewEvidenceRuleFileInput

// ReviewEvidenceDiffInput は diff evidence を LLM 入力向けに表す。
type ReviewEvidenceDiffInput = reviewevidence.ReviewEvidenceDiffInput

// ReviewEvidenceUntrackedFileInput は untracked snapshot を LLM 入力向けに表す。
type ReviewEvidenceUntrackedFileInput = reviewevidence.ReviewEvidenceUntrackedFileInput

// ReviewEvidenceLimitsInput は resource budget を JSON 安定表現にした DTO。
type ReviewEvidenceLimitsInput = reviewevidence.ReviewEvidenceLimitsInput

// ReviewEvidenceTruncationFlagsInput は bundle 全体の truncation 状態を固定順序で表す。
type ReviewEvidenceTruncationFlagsInput = reviewevidence.ReviewEvidenceTruncationFlagsInput

// ReviewEvidenceDiffTruncationInput は diff ごとの truncation 状態を表す。
type ReviewEvidenceDiffTruncationInput = reviewevidence.ReviewEvidenceDiffTruncationInput

// ReviewEvidencePathTruncationInput は path 付き evidence の truncation 状態を表す。
type ReviewEvidencePathTruncationInput = reviewevidence.ReviewEvidencePathTruncationInput

// ReviewWebSearchEvidenceCollectorOptions は外部 Web 検索 evidence collector の設定。
type ReviewWebSearchEvidenceCollectorOptions = reviewevidence.ReviewWebSearchEvidenceCollectorOptions

// ReviewWebSearchQueryRunner は review 用の非対話 Web 検索境界。
type ReviewWebSearchQueryRunner = reviewevidence.ReviewWebSearchQueryRunner

// ReviewWebSearchEvidenceCollector は /review 用の外部 Web 検索 evidence を収集する。
type ReviewWebSearchEvidenceCollector = reviewevidence.ReviewWebSearchEvidenceCollector

const (
	ReviewGenericImpactRoleSameStemTestOrSpec       = reviewevidence.ReviewGenericImpactRoleSameStemTestOrSpec
	ReviewGenericImpactRoleNearbyTestOrTestsDir     = reviewevidence.ReviewGenericImpactRoleNearbyTestOrTestsDir
	ReviewGenericImpactRoleNearbyProjectConfig      = reviewevidence.ReviewGenericImpactRoleNearbyProjectConfig
	ReviewGenericImpactRoleDocsReference            = reviewevidence.ReviewGenericImpactRoleDocsReference
	ReviewGenericImpactRoleTextualReference         = reviewevidence.ReviewGenericImpactRoleTextualReference
	ReviewGenericImpactRoleChangedPathStemReference = reviewevidence.ReviewGenericImpactRoleChangedPathStemReference

	reviewEvidenceRepoRootPathDisplay    = reviewevidence.RepoRootPathDisplay
	reviewEvidenceOutsideRepoPathDisplay = reviewevidence.OutsideRepoPathDisplay
)

// WithReviewEvidenceLimits は EvidenceBuilder の resource budget を差し替える。
func WithReviewEvidenceLimits(limits ReviewEvidenceLimits) ReviewEvidenceBuilderOption {
	return reviewevidence.WithReviewEvidenceLimits(limits)
}

// WithReviewEvidenceCommandRunner は EvidenceBuilder の git command runner を差し替える。
func WithReviewEvidenceCommandRunner(runner ReviewEvidenceCommandRunner) ReviewEvidenceBuilderOption {
	return reviewevidence.WithReviewEvidenceCommandRunner(runner)
}

// WithReviewWebSearchEvidenceProvider は外部 Web 検索 evidence 収集境界を差し替える。
func WithReviewWebSearchEvidenceProvider(provider ReviewWebSearchEvidenceProvider) ReviewEvidenceBuilderOption {
	return reviewevidence.WithReviewWebSearchEvidenceProvider(provider)
}

// NewReviewEvidenceBuilder は repo root と /review 起動 cwd を基準に EvidenceBuilder を構築する。
func NewReviewEvidenceBuilder(repoRoot, cwd string, opts ...ReviewEvidenceBuilderOption) *ReviewEvidenceBuilder {
	return reviewevidence.NewReviewEvidenceBuilder(repoRoot, cwd, opts...)
}

// DefaultReviewEvidenceLimits は EvidenceBuilder の既定 resource budget を返す。
func DefaultReviewEvidenceLimits() ReviewEvidenceLimits {
	return reviewevidence.DefaultReviewEvidenceLimits()
}

// BuildReviewGenericImpactCandidates は言語非依存 heuristic で review 用 impact 候補を作る。
func BuildReviewGenericImpactCandidates(bundle ReviewEvidenceBundle) ReviewGenericImpactCandidates {
	return reviewevidence.BuildReviewGenericImpactCandidates(bundle)
}

// BuildReviewEvidenceModelInput は ReviewEvidenceBundle を LLM 入力 DTO に変換する。
func BuildReviewEvidenceModelInput(bundle ReviewEvidenceBundle) ReviewEvidenceModelInput {
	return reviewevidence.BuildReviewEvidenceModelInput(bundle)
}

// RenderReviewEvidenceJSON は ReviewEvidenceBundle を LLM 入力 JSON に変換する。
func RenderReviewEvidenceJSON(bundle ReviewEvidenceBundle) ([]byte, error) {
	return reviewevidence.RenderReviewEvidenceJSON(bundle)
}

// RenderReviewEvidenceMarkdown は ReviewEvidenceBundle を LLM 入力 Markdown に変換する。
func RenderReviewEvidenceMarkdown(bundle ReviewEvidenceBundle) string {
	return reviewevidence.RenderReviewEvidenceMarkdown(bundle)
}

// NewReviewWebSearchEvidenceCollector は外部 Web 検索 evidence collector を構築する。
func NewReviewWebSearchEvidenceCollector(opts ReviewWebSearchEvidenceCollectorOptions) *ReviewWebSearchEvidenceCollector {
	return reviewevidence.NewReviewWebSearchEvidenceCollector(opts)
}

func formatReviewEvidencePathDisplay(repoRoot, candidate string) string {
	return reviewevidence.FormatReviewEvidencePathDisplay(repoRoot, candidate)
}

func isReviewEvidenceWindowsAbsolutePath(candidate string) bool {
	return reviewevidence.IsReviewEvidenceWindowsAbsolutePath(candidate)
}

func reviewEvidencePathReplacementVariants(path string) []string {
	return reviewevidence.ReviewEvidencePathReplacementVariants(path)
}

func resolveReviewEvidenceDir(candidate, fallback string) (string, error) {
	return reviewevidence.ResolveReviewEvidenceDir(candidate, fallback)
}

func buildReviewEvidenceGitArgs(repoRoot string, commandArgs []string) []string {
	return reviewevidence.BuildReviewEvidenceGitArgs(repoRoot, commandArgs)
}

func buildReviewEvidenceGitEnv(environ []string) []string {
	return reviewevidence.BuildReviewEvidenceGitEnv(environ)
}

func combineReviewEvidenceGitDiagnostics(stderr, stdout string) string {
	return reviewevidence.CombineReviewEvidenceGitDiagnostics(stderr, stdout)
}
