package taskstate

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
