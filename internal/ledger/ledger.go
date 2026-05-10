package ledger

import (
	"sync"

	"github.com/susugadx/xelyon-cli/internal/tools"
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

func (g *Evidence) record(text, source string) {
	if g == nil || text == "" {
		return
	}
	g.items = append(g.items, evidenceFact{text: text, source: source})
}

type evidenceFact struct {
	text   string
	source string
}

// Text は evidence の本文を返す。
func (f evidenceFact) Text() string {
	return f.text
}

// Source は evidence の出所を返す。
func (f evidenceFact) Source() string {
	return f.source
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
	if g == nil || path == "" || g.contains(path) {
		return
	}
	g.items = append(g.items, recommendedReadFact{path: path, reason: reason})
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
	path   string
	reason string
}

// Path は推奨されたファイルパスを返す。
func (f recommendedReadFact) Path() string {
	return f.path
}

// Reason は推奨理由を返す。
func (f recommendedReadFact) Reason() string {
	return f.reason
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
	g.results = cloneTestResults(results)
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
	g.results = cloneTestResults(results)
}

// TestResult はテスト実行結果の最小 fact。
type TestResult struct {
	command string
	output  string
	status  string
}

// NewTestResult はテスト実行結果 fact を作る。
func NewTestResult(command, output, status string) TestResult {
	return TestResult{command: command, output: output, status: status}
}

// Command は実行コマンドを返す。
func (r TestResult) Command() string {
	return r.command
}

// Output はテスト出力を返す。
func (r TestResult) Output() string {
	return r.output
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

// Store は RuntimeTaskState を mutex 付きで保持する。
type Store struct {
	mu    sync.Mutex
	state RuntimeTaskState
}

// NewStore は空の runtime task ledger を返す。
func NewStore() *Store {
	return &Store{}
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
// この段階では FileChange 由来の ChangedFiles だけを記録する。
func (r *Recorder) RecordToolObservation(change *tools.FileChange) {
	if r == nil || r.store == nil || change == nil {
		return
	}
	paths := changedPathsFromFileChange(*change)
	r.mutate(func(state *RuntimeTaskState) {
		for _, path := range paths {
			state.ChangedFiles.recordPath(path)
		}
	})
}

// RecordChangedFile は ChangedFiles へファイルパスを記録する。
func (r *Recorder) RecordChangedFile(path string) {
	r.mutate(func(state *RuntimeTaskState) {
		state.ChangedFiles.recordPath(path)
	})
}

// RecordTouchedFile は TouchedFiles へファイルパスを記録する。
func (r *Recorder) RecordTouchedFile(path string) {
	r.mutate(func(state *RuntimeTaskState) {
		state.TouchedFiles.recordPath(path)
	})
}

// RecordEvidence は Evidence へ根拠を記録する。
func (r *Recorder) RecordEvidence(text, source string) {
	r.mutate(func(state *RuntimeTaskState) {
		state.Evidence.record(text, source)
	})
}

// RecordRecommendedRead は RecommendedReads へファイルパスと理由を記録する。
func (r *Recorder) RecordRecommendedRead(path, reason string) {
	r.mutate(func(state *RuntimeTaskState) {
		state.RecommendedReads.record(path, reason)
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
