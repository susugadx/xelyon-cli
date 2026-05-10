package ledger

import (
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	maxRecordedFiles     = 200
	maxEvidenceItems     = 200
	maxRecommendedReads  = 50
	maxFailedTestResults = 5
	maxPassedTestResults = 5
	maxTestExcerptBytes  = 2000
	maxFactExcerptBytes  = maxTestExcerptBytes
)

// RuntimeTaskState はセッション中だけ保持するタスク台帳のスナップショット。
// prompt/history/provider request への接続はこの package の外側で明示的に行う。
type RuntimeTaskState struct {
	ChangedFiles     ChangedFiles
	TouchedFiles     TouchedFiles
	Evidence         Evidence
	RecommendedReads RecommendedReads
	LastFailedTests  LastFailedTests
	LastPassedTests  LastPassedTests
}

func (s RuntimeTaskState) clone() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles:     s.ChangedFiles.clone(),
		TouchedFiles:     s.TouchedFiles.clone(),
		Evidence:         s.Evidence.clone(),
		RecommendedReads: s.RecommendedReads.clone(),
		LastFailedTests:  s.LastFailedTests.clone(),
		LastPassedTests:  s.LastPassedTests.clone(),
	}
}

// ChangedFiles は変更が観測されたファイルを初出順で保持する。
type ChangedFiles struct {
	files []fileFact
}

func (g ChangedFiles) clone() ChangedFiles {
	return ChangedFiles{files: cloneFileFacts(g.files)}
}

// Paths は記録済みファイルパスを防御コピーで返す。
func (g ChangedFiles) Paths() []string {
	return fileFactPaths(g.files)
}

// Len は記録済みファイルパス数を返す。
func (g ChangedFiles) Len() int {
	return len(g.files)
}

func (g *ChangedFiles) recordPath(path string) {
	if g == nil || path == "" || g.contains(path) {
		return
	}
	if len(g.files) >= maxRecordedFiles {
		return
	}
	g.files = append(g.files, fileFact{path: path})
}

func (g ChangedFiles) contains(path string) bool {
	for _, file := range g.files {
		if file.path == path {
			return true
		}
	}
	return false
}

// TouchedFiles は読み取りや参照で触れたファイルを初出順で保持する。
type TouchedFiles struct {
	files []fileFact
}

func (g TouchedFiles) clone() TouchedFiles {
	return TouchedFiles{files: cloneFileFacts(g.files)}
}

// Paths は記録済みファイルパスを防御コピーで返す。
func (g TouchedFiles) Paths() []string {
	return fileFactPaths(g.files)
}

// Len は記録済みファイルパス数を返す。
func (g TouchedFiles) Len() int {
	return len(g.files)
}

func (g *TouchedFiles) recordPath(path string) {
	if g == nil || path == "" || g.contains(path) {
		return
	}
	if len(g.files) >= maxRecordedFiles {
		return
	}
	g.files = append(g.files, fileFact{path: path})
}

func (g TouchedFiles) contains(path string) bool {
	for _, file := range g.files {
		if file.path == path {
			return true
		}
	}
	return false
}

type fileFact struct {
	path string
}

// Path は記録されたファイルパスを返す。
func (f fileFact) Path() string {
	return f.path
}

func cloneFileFacts(files []fileFact) []fileFact {
	if len(files) == 0 {
		return nil
	}
	cloned := make([]fileFact, len(files))
	copy(cloned, files)
	return cloned
}

func fileFactPaths(files []fileFact) []string {
	if len(files) == 0 {
		return nil
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.path)
	}
	return paths
}

// Evidence はタスク判断の根拠テキストを保持する。
type Evidence struct {
	items []evidenceFact
}

func (g Evidence) clone() Evidence {
	if len(g.items) == 0 {
		return Evidence{}
	}
	items := make([]evidenceFact, len(g.items))
	copy(items, g.items)
	return Evidence{items: items}
}

// Items は記録済み根拠を防御コピーで返す。
func (g Evidence) Items() []evidenceFact {
	if len(g.items) == 0 {
		return nil
	}
	items := make([]evidenceFact, len(g.items))
	copy(items, g.items)
	return items
}

// Len は記録済み根拠数を返す。
func (g Evidence) Len() int {
	return len(g.items)
}

func (g *Evidence) record(fact evidenceFact) {
	if g == nil || fact.excerpt == "" || g.contains(fact) {
		return
	}
	if len(g.items) >= maxEvidenceItems {
		return
	}
	g.items = append(g.items, fact)
}

func (g Evidence) contains(fact evidenceFact) bool {
	for _, item := range g.items {
		if item.path == fact.path &&
			item.startLine == fact.startLine &&
			item.endLine == fact.endLine &&
			item.source == fact.source &&
			item.toolCallID == fact.toolCallID &&
			item.excerpt == fact.excerpt {
			return true
		}
	}
	return false
}

type evidenceFact struct {
	path       string
	startLine  int
	endLine    int
	source     string
	toolCallID string
	fileHash   string
	stale      bool
	excerpt    string
}

// Path は evidence の対象ファイルパスを返す。
func (f evidenceFact) Path() string {
	return f.path
}

// StartLine は evidence の開始行を返す。
func (f evidenceFact) StartLine() int {
	return f.startLine
}

// EndLine は evidence の終了行を返す。
func (f evidenceFact) EndLine() int {
	return f.endLine
}

// Text は evidence の本文を返す。
func (f evidenceFact) Text() string {
	return f.excerpt
}

// Excerpt は evidence の短い抜粋を返す。
func (f evidenceFact) Excerpt() string {
	return f.excerpt
}

// Source は evidence の出所を返す。
func (f evidenceFact) Source() string {
	return f.source
}

// ToolCallID は evidence の元 tool call id を返す。
func (f evidenceFact) ToolCallID() string {
	return f.toolCallID
}

// FileHash は evidence の対象ファイル hash を返す。P0a では空値のまま保持する。
func (f evidenceFact) FileHash() string {
	return f.fileHash
}

// Stale は evidence が古い可能性を返す。P0a では常に false。
func (f evidenceFact) Stale() bool {
	return f.stale
}

// RecommendedReads は後続で読むべきファイルを初出順で保持する。
type RecommendedReads struct {
	items []recommendedReadFact
}

func (g RecommendedReads) clone() RecommendedReads {
	if len(g.items) == 0 {
		return RecommendedReads{}
	}
	items := make([]recommendedReadFact, len(g.items))
	copy(items, g.items)
	return RecommendedReads{items: items}
}

// Items は記録済み推奨 read を防御コピーで返す。
func (g RecommendedReads) Items() []recommendedReadFact {
	if len(g.items) == 0 {
		return nil
	}
	items := make([]recommendedReadFact, len(g.items))
	copy(items, g.items)
	return items
}

// Len は記録済み推奨 read 数を返す。
func (g RecommendedReads) Len() int {
	return len(g.items)
}

func (g *RecommendedReads) record(path, reason string) {
	g.recordFact(recommendedReadFact{path: path, reason: reason})
}

func (g *RecommendedReads) recordFact(fact recommendedReadFact) {
	if g == nil || fact.path == "" || g.contains(fact.path) {
		return
	}
	if len(g.items) >= maxRecommendedReads {
		return
	}
	g.items = append(g.items, fact)
}

func (g RecommendedReads) contains(path string) bool {
	for _, item := range g.items {
		if item.path == path {
			return true
		}
	}
	return false
}

type recommendedReadFact struct {
	path       string
	reason     string
	source     string
	toolCallID string
}

// Path は推奨されたファイルパスを返す。
func (f recommendedReadFact) Path() string {
	return f.path
}

// Reason は推奨理由を返す。
func (f recommendedReadFact) Reason() string {
	return f.reason
}

// Source は推奨 read の出所を返す。
func (f recommendedReadFact) Source() string {
	return f.source
}

// ToolCallID は推奨 read の元 tool call id を返す。
func (f recommendedReadFact) ToolCallID() string {
	return f.toolCallID
}

// LastFailedTests は最後に失敗したテスト結果を保持する。
type LastFailedTests struct {
	results []TestResult
}

func (g LastFailedTests) clone() LastFailedTests {
	return LastFailedTests{results: cloneTestResults(g.results)}
}

// Results は記録済みテスト結果を防御コピーで返す。
func (g LastFailedTests) Results() []TestResult {
	return cloneTestResults(g.results)
}

// Len は記録済みテスト結果数を返す。
func (g LastFailedTests) Len() int {
	return len(g.results)
}

func (g *LastFailedTests) replace(results []TestResult) {
	if g == nil {
		return
	}
	g.results = capTestResults(cloneTestResults(results), maxFailedTestResults)
}

func (g *LastFailedTests) append(result TestResult) {
	if g == nil || result.command == "" {
		return
	}
	g.results = append(g.results, result)
	g.results = capTestResults(g.results, maxFailedTestResults)
}

// LastPassedTests は最後に成功したテスト結果を保持する。
type LastPassedTests struct {
	results []TestResult
}

func (g LastPassedTests) clone() LastPassedTests {
	return LastPassedTests{results: cloneTestResults(g.results)}
}

// Results は記録済みテスト結果を防御コピーで返す。
func (g LastPassedTests) Results() []TestResult {
	return cloneTestResults(g.results)
}

// Len は記録済みテスト結果数を返す。
func (g LastPassedTests) Len() int {
	return len(g.results)
}

func (g *LastPassedTests) replace(results []TestResult) {
	if g == nil {
		return
	}
	g.results = capTestResults(cloneTestResults(results), maxPassedTestResults)
}

func (g *LastPassedTests) append(result TestResult) {
	if g == nil || result.command == "" {
		return
	}
	g.results = append(g.results, result)
	g.results = capTestResults(g.results, maxPassedTestResults)
}

// TestResult はテスト実行結果の最小 fact。
type TestResult struct {
	command  string
	exitCode int
	status   string
	excerpt  string
}

// NewTestResult はテスト実行結果 fact を作る。
func NewTestResult(command, output, status string) TestResult {
	exitCode := 0
	if normalizeTestStatus(status, 0) == "failed" {
		exitCode = -1
	}
	return NewTestResultWithExitCode(command, exitCode, status, output)
}

// NewTestResultWithExitCode は exit code 付きのテスト結果 fact を作る。
func NewTestResultWithExitCode(command string, exitCode int, status string, excerpt string) TestResult {
	status = normalizeTestStatus(status, exitCode)
	return TestResult{
		command:  command,
		exitCode: exitCode,
		status:   status,
		excerpt:  truncateBytes(excerpt, maxTestExcerptBytes),
	}
}

// Command は実行コマンドを返す。
func (r TestResult) Command() string {
	return r.command
}

// ExitCode はテストコマンドの exit code を返す。
func (r TestResult) ExitCode() int {
	return r.exitCode
}

// Output はテスト出力の短い抜粋を返す。
func (r TestResult) Output() string {
	return r.excerpt
}

// Excerpt はテスト出力の短い抜粋を返す。
func (r TestResult) Excerpt() string {
	return r.excerpt
}

// Status はテスト状態を返す。
func (r TestResult) Status() string {
	return r.status
}

func cloneTestResults(results []TestResult) []TestResult {
	if len(results) == 0 {
		return nil
	}
	cloned := make([]TestResult, len(results))
	copy(cloned, results)
	return cloned
}

func capTestResults(results []TestResult, limit int) []TestResult {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	capped := make([]TestResult, limit)
	copy(capped, results[len(results)-limit:])
	return capped
}

// Store は RuntimeTaskState を mutex 付きで保持する。
type Store struct {
	mu            sync.Mutex
	state         RuntimeTaskState
	repoRoot      string
	invocationCWD string
}

// NewStore は空の runtime task ledger を返す。
func NewStore() *Store {
	return NewStoreForInvocationCWD(defaultInvocationCWD())
}

// NewStoreWithRoot は repo root を明示した runtime task ledger を返す。
func NewStoreWithRoot(root string) *Store {
	return NewStoreWithWorkspace(root, root)
}

// NewStoreForInvocationCWD は起動 cwd から repo root を推定した runtime task ledger を返す。
func NewStoreForInvocationCWD(cwd string) *Store {
	cwd = normalizeRepoRoot(cwd)
	if cwd == "" {
		cwd = normalizeRepoRoot(defaultInvocationCWD())
	}
	if cwd == "" {
		return NewStoreWithWorkspace("", "")
	}
	return NewStoreWithWorkspace(repoRootForInvocationCWD(cwd), cwd)
}

// NewStoreWithWorkspace は repo root と起動 cwd を明示した runtime task ledger を返す。
func NewStoreWithWorkspace(root, invocationCWD string) *Store {
	root = normalizeRepoRoot(root)
	invocationCWD = normalizeRepoRoot(invocationCWD)
	if invocationCWD == "" {
		invocationCWD = root
	}
	if root == "" {
		root = invocationCWD
	}
	return &Store{repoRoot: root, invocationCWD: invocationCWD}
}

// Snapshot は現在の RuntimeTaskState を防御コピーで返す。
func (s *Store) Snapshot() RuntimeTaskState {
	if s == nil {
		return RuntimeTaskState{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.clone()
}

// Reset は runtime task ledger を空に戻す。
func (s *Store) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = RuntimeTaskState{}
}

// Recorder は Store への書き込み入口を返す。
func (s *Store) Recorder() *Recorder {
	if s == nil {
		return nil
	}
	return &Recorder{store: s}
}

// Recorder は RuntimeTaskState の mutation を集約する。
type Recorder struct {
	store *Store
}

// RecordToolObservation はツール観測から台帳 fact を更新する。
func (r *Recorder) RecordToolObservation(observation ToolObservation) {
	if r == nil || r.store == nil {
		return
	}
	invocationCWD := r.store.invocationCWD
	if strings.TrimSpace(observation.InvocationCWD) != "" {
		invocationCWD = observation.InvocationCWD
	}
	facts := collectToolObservationFacts(r.store.repoRoot, invocationCWD, observation)
	r.mutate(func(state *RuntimeTaskState) {
		for _, path := range facts.changedFiles {
			state.ChangedFiles.recordPath(path)
		}
		for _, path := range facts.touchedFiles {
			state.TouchedFiles.recordPath(path)
		}
		for _, fact := range facts.evidence {
			state.Evidence.record(fact)
		}
		for _, fact := range facts.recommendedReads {
			state.RecommendedReads.recordFact(fact)
		}
		for _, result := range facts.failedTests {
			state.LastFailedTests.append(result)
		}
		for _, result := range facts.passedTests {
			state.LastPassedTests.append(result)
		}
	})
}

// RecordChangedFile は ChangedFiles へファイルパスを記録する。
func (r *Recorder) RecordChangedFile(path string) {
	normalized, ok := normalizeLedgerPath(r.repoRoot(), r.invocationCWD(), path)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.ChangedFiles.recordPath(normalized)
	})
}

// RecordTouchedFile は TouchedFiles へファイルパスを記録する。
func (r *Recorder) RecordTouchedFile(path string) {
	normalized, ok := normalizeLedgerPath(r.repoRoot(), r.invocationCWD(), path)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.TouchedFiles.recordPath(normalized)
	})
}

// RecordEvidence は Evidence へ根拠を記録する。
func (r *Recorder) RecordEvidence(text, source string) {
	excerpt := strings.TrimSpace(truncateBytes(text, maxFactExcerptBytes))
	if excerpt == "" {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.Evidence.record(evidenceFact{excerpt: excerpt, source: source})
	})
}

// RecordRecommendedRead は RecommendedReads へファイルパスと理由を記録する。
func (r *Recorder) RecordRecommendedRead(path, reason string) {
	normalized, ok := normalizeLedgerPath(r.repoRoot(), r.invocationCWD(), path)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.RecommendedReads.record(normalized, strings.TrimSpace(reason))
	})
}

// RecordTestObservation はテスト実行観測から LastPassedTests / LastFailedTests を更新する。
func (r *Recorder) RecordTestObservation(observation TestObservation) {
	result, ok := testResultFromObservation(observation)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		switch result.status {
		case "passed":
			state.LastPassedTests.append(result)
		case "failed":
			state.LastFailedTests.append(result)
		}
	})
}

// SetLastFailedTests は LastFailedTests を指定値で置き換える。
func (r *Recorder) SetLastFailedTests(results []TestResult) {
	r.mutate(func(state *RuntimeTaskState) {
		state.LastFailedTests.replace(results)
	})
}

// SetLastPassedTests は LastPassedTests を指定値で置き換える。
func (r *Recorder) SetLastPassedTests(results []TestResult) {
	r.mutate(func(state *RuntimeTaskState) {
		state.LastPassedTests.replace(results)
	})
}

func (r *Recorder) mutate(fn func(*RuntimeTaskState)) {
	if r == nil || r.store == nil || fn == nil {
		return
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	fn(&r.store.state)
}

func (r *Recorder) repoRoot() string {
	if r == nil || r.store == nil {
		return ""
	}
	return r.store.repoRoot
}

func (r *Recorder) invocationCWD() string {
	if r == nil || r.store == nil {
		return ""
	}
	return r.store.invocationCWD
}

func changedPathsFromFileChange(change tools.FileChange) []string {
	if len(change.Details) > 0 {
		paths := make([]string, 0, len(change.Details))
		for _, detail := range change.Details {
			paths = append(paths, detail.FilePath)
		}
		return paths
	}
	return []string{change.FilePath}
}
