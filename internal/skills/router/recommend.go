package router

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

var tokenRe = regexp.MustCompile(`[[:alnum:]_/-]+`)

type taskProfile struct {
	text              string
	terms             map[string]struct{}
	intents           map[string]struct{}
	modes             map[string]struct{}
	implementation    bool
	review            bool
	planning          bool
	authoring         bool
	readOnly          bool
	pathHints         []string
	signalDiagnostics []string
}

type skillConflictProfile struct {
	readOnly  bool
	intents   []string
	modes     []string
	conflicts []string
}

// Recommend は catalog 内の全 skill を deterministic に score/rank する。
func Recommend(catalog skillcatalog.SkillCatalog, input Input) Recommendation {
	profile := analyzeInput(input)
	ranked := make([]Candidate, 0, len(catalog.Skills))
	for _, skill := range catalog.Skills {
		ranked = append(ranked, scoreSkill(skill, profile))
	}
	sortCandidates(ranked)

	rec := Recommendation{
		Ranked:            ranked,
		SignalDiagnostics: append([]string(nil), input.SignalDiagnostics...),
	}
	for _, candidate := range ranked {
		switch candidate.Category {
		case CategoryPrimary:
			rec.Primary = append(rec.Primary, candidate)
		case CategorySupporting:
			rec.Supporting = append(rec.Supporting, candidate)
		case CategoryMaybe:
			rec.Maybe = append(rec.Maybe, candidate)
		case CategoryConflict:
			rec.Conflicts = append(rec.Conflicts, candidate)
		}
	}
	return rec
}

func analyzeInput(input Input) taskProfile {
	requestedMode := strings.ToLower(strings.TrimSpace(input.RequestedMode))
	command := strings.ToLower(strings.TrimSpace(input.Command))
	planMode := requestedMode == "plan" || command == "/plan"
	text := strings.ToLower(strings.TrimSpace(input.TaskText + " " + input.Command + " " + input.RequestedMode))
	profile := taskProfile{
		text:              text,
		terms:             tokenSet(text),
		intents:           map[string]struct{}{},
		modes:             map[string]struct{}{},
		readOnly:          input.ReadOnly,
		signalDiagnostics: append([]string(nil), input.SignalDiagnostics...),
	}
	addIntent := func(intent string) {
		profile.intents[intent] = struct{}{}
	}
	addMode := func(mode string) {
		profile.modes[mode] = struct{}{}
	}

	reviewCue := hasReviewCue(text)
	planningCue := hasPlanningCue(text)
	investigationOnlyCue := hasInvestigationOnlyCue(text)
	contextualReadOnlyCue := reviewCue || investigationOnlyCue || (planningCue && !planMode)
	mutationCue := hasImplementationMutationCue(text, input.ReadOnly, contextualReadOnlyCue)

	if reviewCue {
		profile.review = true
		profile.readOnly = true
		addIntent("code-review")
		addMode("review")
	}
	if planMode {
		profile.planning = true
		addIntent("planning")
		addMode("planning")
		addMode("investigation")
	}
	if investigationOnlyCue {
		profile.readOnly = true
	}
	if mutationCue {
		profile.implementation = true
		profile.readOnly = input.ReadOnly
		addIntent("implementation")
		addMode("implementation")
	}
	if containsAny(text, "bug", "不具合", "バグ", "原因", "investigate", "調査") {
		addIntent("bug-investigation")
		addMode("investigation")
	}
	if containsAny(text, "risk", "リスク", "scan", "監査") {
		addIntent("risk-scan")
	}
	if containsAny(text, "refactor", "リファクタ", "整理") {
		addIntent("refactor")
		addMode("refactor")
	}
	if containsAny(text, "cleanup", "dead code", "未使用", "削除") {
		addIntent("cleanup")
		addMode("cleanup")
	}
	if containsAny(text, "test", "テスト", "coverage", "カバレッジ") {
		addIntent("test-coverage")
		addMode("test")
	}
	if containsAny(text, "config", "設定", "yaml", "toml") {
		addIntent("config")
		addMode("config")
	}
	if containsAny(text, "provider", "runtime", "model", "token", "pricing", "プロバイダ", "モデル") {
		addIntent("provider-runtime")
	}
	if containsAny(text, "state", "cache", "ledger", "history", "lifecycle", "状態", "履歴") {
		addIntent("state-lifecycle")
	}
	if containsAny(text, "security", "secret", "path traversal", "sandbox", "セキュリティ") {
		addIntent("security-boundary")
	}
	if containsAny(text, "package", "boundary", "境界") {
		addIntent("package-boundary")
	}
	if containsAny(text, "skill", "skills", "スキル", "スキルズ") {
		profile.authoring = true
		addIntent("skill-authoring")
		addMode("authoring")
	}
	if containsAny(text, "docs", "document", "ドキュメント", "説明") {
		addIntent("docs-authoring")
		addMode("docs")
	}
	if planningCue {
		profile.planning = true
		if !mutationCue {
			profile.readOnly = true
		}
		addIntent("planning")
		addMode("planning")
	}

	for _, path := range input.TouchedPaths {
		applyPathHints(filepath.ToSlash(path), &profile)
	}
	return profile
}

func hasReviewCue(text string) bool {
	if containsAny(text, "/review", "diff review", "code review", "差分レビュー", "レビュー") {
		return true
	}
	return startsWithSignal(text, "review") ||
		containsAny(text, "please review", "can you review", "could you review")
}

func hasPlanningCue(text string) bool {
	return containsAny(text, "plan", "計画", "方針", "相談")
}

func hasInvestigationOnlyCue(text string) bool {
	return containsAny(
		text,
		"investigate only",
		"analysis only",
		"do not fix",
		"don't fix",
		"no fix",
		"調査だけ",
		"原因調査だけ",
		"修正はまだしない",
		"修正しない",
		"直さない",
		"なおさない",
	)
}

func hasImplementationMutationCue(text string, inputReadOnly bool, contextualReadOnlyCue bool) bool {
	if inputReadOnly {
		return false
	}
	if contextualReadOnlyCue && containsAny(text, "how to implement", "how to fix", "実装方法", "修正方法", "直し方") {
		return false
	}
	if containsAny(text,
		"apply patch",
		"apply changes",
		"edit file",
		"edit files",
		"update file",
		"update files",
		"modify file",
		"modify files",
		"make changes",
		"write code",
		"fix review findings",
		"fix findings",
		"fix the",
		"fix this",
		"fix it",
		"fix bug",
		"fix bugs",
		"implement the",
		"implement this",
		"implement it",
		"and implement",
		"then implement",
		"and fix",
		"then fix",
		"修正して",
		"直して",
		"なおして",
		"実装して",
		"変更して",
		"更新して",
	) {
		return true
	}
	if contextualReadOnlyCue {
		return false
	}
	return containsAny(text, "implement", "fix", "修正", "実装", "apply", "edit")
}

func scoreSkill(skill skillcatalog.ParsedSkill, profile taskProfile) Candidate {
	metadata := skill.Routing
	activation := skillcatalog.RoutingActivationHint
	role := skillcatalog.RoutingRole("")
	readOnly := false
	intents := []string(nil)
	modes := []string(nil)
	triggers := []string(nil)
	conflicts := []string(nil)
	if metadata != nil {
		activation = metadata.Activation
		role = metadata.Role
		readOnly = metadata.ReadOnly
		intents = metadata.Intents
		modes = metadata.Modes
		triggers = metadata.Triggers
		conflicts = metadata.Conflicts
	}

	score := 0
	var matched []string
	var reasons []string

	if skillNameMentioned(profile.text, skill.Name) {
		score += 85
		matched = append(matched, "explicit skill name")
		reasons = append(reasons, fmt.Sprintf("task explicitly mentions %s", skill.Name))
	}

	triggerScore := 0
	for _, trigger := range triggers {
		if trigger != "" && containsSignal(profile.text, trigger) {
			triggerScore += 30
			matched = append(matched, "trigger:"+trigger)
		}
	}
	if triggerScore > 50 {
		triggerScore = 50
	}
	if triggerScore > 0 {
		score += triggerScore
		reasons = append(reasons, "trigger phrase matched")
	}

	intentMatches := intersectSet(intents, profile.intents)
	if len(intentMatches) > 0 {
		score += min(35, 22+8*len(intentMatches))
		for _, intent := range intentMatches {
			matched = append(matched, "intent:"+intent)
		}
		reasons = append(reasons, "routing intent matches task")
	}

	modeMatches := intersectSet(modes, profile.modes)
	if len(modeMatches) > 0 {
		score += min(25, 16+5*len(modeMatches))
		for _, mode := range modeMatches {
			matched = append(matched, "mode:"+mode)
		}
	}

	overlap := descriptionOverlap(skill.Description, profile.terms)
	if overlap > 0 {
		add := min(30, 8+overlap*5)
		score += add
		matched = append(matched, fmt.Sprintf("description:%d", overlap))
		if len(reasons) < 2 {
			reasons = append(reasons, "description overlaps task terms")
		}
	}

	if role == skillcatalog.RoutingRolePrimary && score > 0 {
		score += 6
	} else if role == skillcatalog.RoutingRoleSupporting || role == skillcatalog.RoutingRoleGuardrail {
		if score > 0 {
			score += 4
		}
	} else if role == skillcatalog.RoutingRoleAuthoring && profile.authoring {
		score += 12
	}

	switch skill.Source {
	case skillcatalog.SourceProject:
		if score > 0 {
			score += 5
		}
	case skillcatalog.SourceHome:
		if score > 0 {
			score += 3
		}
	case skillcatalog.SourceXelyon:
		if score > 0 {
			score += 4
		}
	}

	explicit := slicesContains(matched, "explicit skill name")
	if (activation == skillcatalog.RoutingActivationManual || activation == skillcatalog.RoutingActivationNever) && !explicit {
		score = 0
	}
	score = clamp(score, 0, 100)

	conflictReason := resolveConflictReason(skillConflictProfile{
		readOnly:  readOnly,
		intents:   intents,
		modes:     modes,
		conflicts: conflicts,
	}, profile)
	category := classifyCandidate(score, role, activation, conflictReason, profile)
	if conflictReason != "" {
		reasons = append([]string{conflictReason}, reasons...)
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "no strong routing signals matched")
	}

	return Candidate{
		Name:           skill.Name,
		Description:    skill.Description,
		Source:         skill.Source,
		Role:           role,
		ReadOnly:       readOnly,
		Activation:     activation,
		Category:       category,
		Score:          score,
		Confidence:     ConfidenceForScore(score),
		MatchedSignals: dedupeStrings(matched),
		Reason:         strings.Join(dedupeStrings(reasons), "; "),
		ConflictReason: conflictReason,
	}
}

func resolveConflictReason(skill skillConflictProfile, profile taskProfile) string {
	conflictSet := stringSet(skill.conflicts)
	if profile.implementation && (skill.readOnly || hasAnyConflict(conflictSet, skillcatalog.RoutingConflictImplementation, skillcatalog.RoutingConflictFileEdit)) {
		return "read-only skill conflicts with implementation or file editing in this turn"
	}
	if profile.readOnly && hasAnyConflict(conflictSet, skillcatalog.RoutingConflictReadOnly) {
		return "skill declares a read-only conflict for this read-only turn"
	}
	if profile.readOnly && skill.hasImplementationGuidance() {
		return "implementation skill conflicts with read-only review or analysis in this turn"
	}
	if profile.review && hasAnyConflict(conflictSet, skillcatalog.RoutingConflictReview) && !profile.implementation {
		return "skill declares a review conflict for this read-only review turn"
	}
	if profile.planning && hasAnyConflict(conflictSet, skillcatalog.RoutingConflictPlanningOnly) && profile.implementation {
		return "planning-only skill conflicts with implementation"
	}
	return ""
}

func (skill skillConflictProfile) hasImplementationGuidance() bool {
	return hasAnyString(skill.intents, "implementation", "file-edit") ||
		hasAnyString(skill.modes, "implementation", "file-edit")
}

func classifyCandidate(score int, role skillcatalog.RoutingRole, activation skillcatalog.RoutingActivation, conflictReason string, profile taskProfile) CandidateCategory {
	if conflictReason != "" && score >= ScoreLowMin {
		return CategoryConflict
	}
	if activation == skillcatalog.RoutingActivationNever && score < ScoreHighMin {
		return CategoryNone
	}
	if score < ScoreLowMin {
		return CategoryNone
	}
	if score < ScoreMediumMin {
		return CategoryMaybe
	}
	if role == skillcatalog.RoutingRoleSupporting || role == skillcatalog.RoutingRoleGuardrail {
		return CategorySupporting
	}
	if role == skillcatalog.RoutingRoleAuthoring && !profile.authoring {
		return CategorySupporting
	}
	return CategoryPrimary
}

func sortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if categoryRank(left.Category) != categoryRank(right.Category) {
			return categoryRank(left.Category) < categoryRank(right.Category)
		}
		leftName := strings.ToLower(left.Name)
		rightName := strings.ToLower(right.Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return left.Name < right.Name
	})
}

func categoryRank(category CandidateCategory) int {
	switch category {
	case CategoryPrimary:
		return 0
	case CategorySupporting:
		return 1
	case CategoryConflict:
		return 2
	case CategoryMaybe:
		return 3
	default:
		return 4
	}
}

func applyPathHints(path string, profile *taskProfile) {
	if profile == nil {
		return
	}
	addIntent := func(intent string) {
		profile.intents[intent] = struct{}{}
	}
	addMode := func(mode string) {
		profile.modes[mode] = struct{}{}
	}
	appendHint := func(hint string) {
		profile.pathHints = append(profile.pathHints, hint)
	}

	lower := strings.ToLower(path)
	ext := strings.ToLower(filepath.Ext(lower))
	if strings.Contains(lower, "internal/skills") || strings.Contains(lower, ".agents/skills") {
		addIntent("skill-authoring")
		addMode("authoring")
		profile.authoring = true
		appendHint("skill-path")
	}
	if strings.Contains(lower, "config") || ext == ".yaml" || ext == ".yml" || ext == ".toml" {
		addIntent("config")
		addMode("config")
		appendHint("config-path")
	}
	if strings.Contains(lower, "provider") || strings.Contains(lower, "token") || strings.Contains(lower, "pricing") {
		addIntent("provider-runtime")
		appendHint("provider-path")
	}
	if strings.HasSuffix(lower, "_test.go") || strings.Contains(lower, "/test") || strings.Contains(lower, "fixture") {
		addIntent("test-coverage")
		addMode("test")
		appendHint("test-path")
	}
	if strings.Contains(lower, "docs/") || ext == ".md" {
		addIntent("docs-authoring")
		addMode("docs")
		appendHint("docs-path")
	}
}

func tokenSet(text string) map[string]struct{} {
	terms := map[string]struct{}{}
	for _, token := range tokenRe.FindAllString(strings.ToLower(text), -1) {
		token = strings.Trim(token, "_-/")
		if len([]rune(token)) < 3 {
			continue
		}
		terms[token] = struct{}{}
	}
	return terms
}

func descriptionOverlap(description string, terms map[string]struct{}) int {
	count := 0
	for token := range tokenSet(description) {
		if _, ok := terms[token]; ok {
			count++
		}
	}
	return count
}

func skillNameMentioned(text, name string) bool {
	return containsSignal(text, name)
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if containsSignal(text, value) {
			return true
		}
	}
	return false
}

func startsWithSignal(text, value string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	value = strings.ToLower(strings.TrimSpace(value))
	if text == "" || value == "" {
		return false
	}
	if !isASCIISignal(value) {
		return strings.HasPrefix(text, value)
	}
	text = normalizeSignalWhitespace(text)
	value = normalizeSignalWhitespace(value)
	if !strings.HasPrefix(text, value) {
		return false
	}
	return hasSignalBoundary(text, len(value))
}

func containsSignal(text, value string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	value = strings.ToLower(strings.TrimSpace(value))
	if text == "" || value == "" {
		return false
	}
	if !isASCIISignal(value) {
		return strings.Contains(text, value)
	}
	return containsBoundaryPhrase(text, value)
}

func containsBoundaryPhrase(text, phrase string) bool {
	text = normalizeSignalWhitespace(text)
	phrase = normalizeSignalWhitespace(phrase)
	if text == "" || phrase == "" {
		return false
	}
	offset := 0
	for {
		idx := strings.Index(text[offset:], phrase)
		if idx < 0 {
			return false
		}
		idx += offset
		end := idx + len(phrase)
		if hasSignalBoundary(text, idx-1) && hasSignalBoundary(text, end) {
			return true
		}
		offset = idx + 1
	}
}

func normalizeSignalWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func hasSignalBoundary(value string, idx int) bool {
	if idx < 0 || idx >= len(value) {
		return true
	}
	return !isASCIIIdentifierByte(value[idx])
}

func isASCIISignal(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > 0x7f {
			return false
		}
	}
	return true
}

func isASCIIIdentifierByte(value byte) bool {
	return value == '_' || ('a' <= value && value <= 'z') || ('0' <= value && value <= '9')
}

func intersectSet(values []string, set map[string]struct{}) []string {
	var out []string
	for _, value := range values {
		if _, ok := set[value]; ok {
			out = append(out, value)
		}
	}
	return out
}

func hasAnyConflict(conflicts map[string]struct{}, values ...string) bool {
	for _, value := range values {
		if _, ok := conflicts[value]; ok {
			return true
		}
	}
	return false
}

func hasAnyString(values []string, wants ...string) bool {
	wantSet := stringSet(wants)
	for _, value := range values {
		if _, ok := wantSet[value]; ok {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func dedupeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func slicesContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
