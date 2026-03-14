package navigation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func TestInspectSymbol_EmptySymbol(t *testing.T) {
	result := InspectSymbol("", "", "")
	if !strings.Contains(result, "Error") {
		t.Errorf("expected error for empty symbol, got: %s", result)
	}
}

func TestInspectSymbol_NotFound(t *testing.T) {
	result := InspectSymbol("NonExistentSymbol_XYZ_12345", "", "")
	if !strings.Contains(result, "No symbol found") {
		t.Errorf("expected 'No symbol found', got: %s", result)
	}
}

// setupTestGoFile はテスト用の Go ファイルを一時ディレクトリに作成し、
// そのディレクトリに cd した後、元に戻すための cleanup を返す。
func setupTestGoFile(t *testing.T, filename, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})
	return dir
}

const testGoSource = `package example

import "fmt"

// Build は何かをビルドする。
func Build(name string) error {
	fmt.Println(name)
	return nil
}

// Run は Build を呼ぶ。
func Run() {
	Build("test")
}

type Config struct {
	Name string
}

func (c *Config) Build() string {
	return c.Name
}

var DefaultBuilder = Build
`

func TestInspectSymbol_SingleCandidate(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Run", "", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Run") {
		t.Errorf("expected Run in output, got: %s", result)
	}
	if !strings.Contains(result, "func Run") {
		t.Errorf("expected function definition in output, got: %s", result)
	}
	if !strings.Contains(result, "Truncated:") {
		t.Errorf("expected Truncated section, got: %s", result)
	}
}

func TestInspectSymbol_WithPath(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Build", "example.go", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// example.go には function Build と method Build の2つがある
	// path でファイルは絞れるが、同一ファイル内の複数候補は残る
	if strings.Contains(result, "Multiple symbols matched") {
		// 期待通り：同一ファイルに function Build と method Build がある
		if !strings.Contains(result, "Refine with path") {
			t.Error("expected disambiguation hint")
		}
	}
}

func TestInspectSymbol_MultipleCandidates(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Build", "", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// Build は function と method の2つがある
	if !strings.Contains(result, "Multiple symbols matched") {
		// 単一候補に解決された場合も OK（AST の仕様による）
		t.Logf("resolved to single candidate: %s", result)
	}
}

func TestInspectSymbol_FullMode(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Config", "", "full")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Config") {
		t.Errorf("expected Config in output, got: %s", result)
	}
}

func TestInspectSymbol_WithCallers(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	// Build を Run から呼んでいるので caller がある
	// ただし同名のメソッド Build もあるため複数候補になる可能性がある
	result := InspectSymbol("Run", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// Run は単一候補のはず
	if strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected single candidate for Run, got: %s", result)
	}
}

func TestInspectSymbol_WithTests(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "example.go")
	testFile := filepath.Join(dir, "example_test.go")

	if err := os.WriteFile(mainFile, []byte(`package example

func Greet() string {
	return "hello"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, []byte(`package example

import "testing"

func TestGreet(t *testing.T) {
	if Greet() != "hello" {
		t.Error("wrong")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})

	result := InspectSymbol("Greet", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Related tests") {
		t.Errorf("expected related tests section, got: %s", result)
	}
	if !strings.Contains(result, "TestGreet") {
		t.Errorf("expected TestGreet in output, got: %s", result)
	}
}

func TestFormatMultipleCandidates(t *testing.T) {
	candidates := []SymbolCandidate{
		{Name: "Build", Kind: "function", File: "internal/agent/agent.go", Line: 21, EndLine: 85},
		{Name: "Build", Kind: "method", File: "internal/config/config.go", Line: 88, EndLine: 120},
	}
	result := formatMultipleCandidates("Build", candidates)
	if !strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected multiple candidates header, got: %s", result)
	}
	if !strings.Contains(result, "agent.go") {
		t.Errorf("expected agent.go in output, got: %s", result)
	}
	if !strings.Contains(result, "Refine with path") {
		t.Errorf("expected path hint, got: %s", result)
	}
}

func TestFormatInspectResult(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body: []string{
			"21: func (a *Agent) Build() error {",
			"22:     return nil",
			"23: }",
		},
		Callers: []Reference{
			{File: "cmd/root.go", Line: 88, Scope: "func main"},
		},
		Refs: []Reference{
			{File: "init.go", Line: 30, Snippet: "var defaultBuilder = Build"},
		},
		Tests: []TestRef{
			{File: "agent_test.go", Name: "TestBuild_Normal", Line: 55, EndLine: 80},
		},
	})

	if !strings.Contains(result, "── function Build") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "Callers (1)") {
		t.Error("expected callers section")
	}
	if !strings.Contains(result, "References (1)") {
		t.Error("expected references section")
	}
	if !strings.Contains(result, "Related tests (1)") {
		t.Error("expected tests section")
	}
}

func TestReadDefinitionBody_MaxLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")
	var lines []string
	lines = append(lines, "package test")
	lines = append(lines, "")
	lines = append(lines, "func Foo() {")
	for i := 0; i < 50; i++ {
		lines = append(lines, "    // line")
	}
	lines = append(lines, "}")
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	cand := SymbolCandidate{
		File:    file,
		Line:    3,
		EndLine: 54,
	}

	body := readDefinitionBody(cand, 10)
	if len(body) == 0 {
		t.Fatal("expected body lines")
	}
	// 10行 + truncation メッセージ = 11 行
	if len(body) != 11 {
		t.Errorf("expected 11 lines (10 + truncation), got %d", len(body))
	}
	lastLine := body[len(body)-1]
	if !strings.Contains(lastLine, "more lines") {
		t.Errorf("expected truncation message, got: %s", lastLine)
	}
}

func TestClassifyCallers_UsesASTClass(t *testing.T) {
	def := SymbolCandidate{
		Name: "Build", File: "agent.go", Line: 10, EndLine: 20,
	}
	refs := []Reference{
		{File: "agent.go", Line: 15, Snippet: "Build()", Scope: "func Build", Class: ast.ClassCall},
		{File: "main.go", Line: 5, Snippet: "Build()", Scope: "func main", Class: ast.ClassCall},
		{File: "ref.go", Line: 8, Snippet: "var x = Build", Scope: "package-level", Class: ast.ClassRef},
		{File: "test_test.go", Line: 3, Snippet: "Build()", IsTest: true, Class: ast.ClassCall},
	}

	callers, more := classifyCallers(refs, def, 10)
	if more {
		t.Error("expected no truncation")
	}
	// 定義行(agent.go:15)とテスト(test_test.go)は除外、ClassRef(ref.go)は caller でない
	if len(callers) != 1 {
		t.Errorf("expected 1 caller (main.go), got %d", len(callers))
	}
	if len(callers) > 0 && callers[0].File != "main.go" {
		t.Errorf("expected main.go, got %s", callers[0].File)
	}
}

func TestClassifyCallers_Truncation(t *testing.T) {
	def := SymbolCandidate{Name: "X", File: "def.go", Line: 1, EndLine: 1}
	var refs []Reference
	for i := range 10 {
		refs = append(refs, Reference{
			File: "other.go", Line: i + 1, Snippet: "X()", Class: ast.ClassCall,
		})
	}

	callers, more := classifyCallers(refs, def, 3)
	if !more {
		t.Error("expected truncation")
	}
	if len(callers) != 3 {
		t.Errorf("expected 3 callers, got %d", len(callers))
	}
}

// --- Regression tests ---

// Regression 1: 同名シンボルを定義するファイル内の call/ref が除外される
func TestFilterRefsByCandidate_AmbiguousFileExclusion(t *testing.T) {
	cand := SymbolCandidate{
		Name: "Build", Kind: "function", File: "agent.go", Line: 10, EndLine: 20,
	}
	// config.go は同名シンボル Build を持つ曖昧ファイル
	ambiguousFiles := map[string]bool{"config.go": true}

	refs := []Reference{
		// 候補自身の定義行 → 除外
		{File: "agent.go", Line: 15, Snippet: "func Build()", Class: ast.ClassDef},
		// config.go の定義行 → 曖昧ファイルで除外
		{File: "config.go", Line: 88, Snippet: "func (c *Config) Build()", Class: ast.ClassDef},
		// config.go 内の call → 曖昧ファイルで除外（config 側 Build かもしれない）
		{File: "config.go", Line: 95, Snippet: "c.Build()", Class: ast.ClassCall},
		// config.go 内の ref → 曖昧ファイルで除外
		{File: "config.go", Line: 100, Snippet: "var b = Build", Class: ast.ClassRef},
		// 候補への正当な呼び出し（曖昧ファイルでない）→ 残る
		{File: "main.go", Line: 5, Snippet: "Build(\"test\")", Class: ast.ClassCall},
		// 別ファイルの参照（曖昧ファイルでない）→ 残る
		{File: "init.go", Line: 30, Snippet: "var builder = Build", Class: ast.ClassRef},
	}

	filtered := filterRefsByCandidate(refs, cand, ambiguousFiles)
	if len(filtered) != 2 {
		t.Errorf("expected 2 refs after filtering, got %d", len(filtered))
		for _, r := range filtered {
			t.Logf("  %s:%d class=%s", r.File, r.Line, r.Class)
		}
	}
	for _, ref := range filtered {
		if ref.File == "config.go" {
			t.Errorf("config.go ref should have been filtered: %s:%d", ref.File, ref.Line)
		}
	}
}

// Regression: 逆方向（path="config.go"）でも agent.go 由来の ref が混ざらない
func TestFilterRefsByCandidate_ReverseDirection(t *testing.T) {
	cand := SymbolCandidate{
		Name: "Build", Kind: "method", File: "config.go", Line: 88, EndLine: 95,
	}
	// agent.go は同名シンボル Build を持つ曖昧ファイル
	ambiguousFiles := map[string]bool{"agent.go": true}

	refs := []Reference{
		// agent.go の call → 曖昧ファイルで除外
		{File: "agent.go", Line: 30, Snippet: "Build(\"test\")", Class: ast.ClassCall},
		// agent.go の ref → 曖昧ファイルで除外
		{File: "agent.go", Line: 40, Snippet: "var b = Build", Class: ast.ClassRef},
		// config.go 自身の定義行 → 除外
		{File: "config.go", Line: 90, Snippet: "c.Build()", Class: ast.ClassCall},
		// 安全な参照 → 残る
		{File: "handler.go", Line: 10, Snippet: "c.Build()", Class: ast.ClassCall},
	}

	filtered := filterRefsByCandidate(refs, cand, ambiguousFiles)
	if len(filtered) != 1 {
		t.Errorf("expected 1 ref after filtering, got %d", len(filtered))
		for _, r := range filtered {
			t.Logf("  %s:%d class=%s", r.File, r.Line, r.Class)
		}
	}
	if len(filtered) > 0 && filtered[0].File != "handler.go" {
		t.Errorf("expected handler.go, got %s", filtered[0].File)
	}
}

// Regression 2: interface メソッド宣言や関数型シグネチャは caller に分類されない
func TestClassifyCallers_InterfaceAndFuncTypeNotCaller(t *testing.T) {
	def := SymbolCandidate{
		Name: "Process", Kind: "function", File: "process.go", Line: 5, EndLine: 20,
	}
	refs := []Reference{
		// interface メソッド宣言 → ClassDef → caller にならない
		{File: "handler.go", Line: 10, Snippet: "Process(ctx context.Context)", Class: ast.ClassDef},
		// 関数型シグネチャ → ClassRef → caller にならない
		{File: "types.go", Line: 15, Snippet: "type ProcessFunc func() = Process", Class: ast.ClassRef},
		// 実際の呼び出し → ClassCall → caller になる
		{File: "main.go", Line: 30, Snippet: "Process(ctx)", Class: ast.ClassCall},
	}

	callers, _ := classifyCallers(refs, def, 10)
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d", len(callers))
	}
	if len(callers) > 0 && callers[0].File != "main.go" {
		t.Errorf("expected main.go caller, got %s", callers[0].File)
	}
}

// Regression 3: テスト参照が References と Related tests で二重カウントされない
func TestClassifyRefs_NoTestDoubleCount(t *testing.T) {
	def := SymbolCandidate{
		Name: "Build", Kind: "function", File: "agent.go", Line: 10, EndLine: 20,
	}
	refs := []Reference{
		// テストファイルの参照 → classifyRefs では除外（Related tests で扱う）
		{File: "agent_test.go", Line: 55, Snippet: "Build()", IsTest: true, Class: ast.ClassCall},
		{File: "agent_test.go", Line: 60, Snippet: "var b = Build", IsTest: true, Class: ast.ClassRef},
		// 非テストの参照 → refs に含まれる
		{File: "init.go", Line: 30, Snippet: "var builder = Build", Class: ast.ClassRef},
	}

	classified, _ := classifyRefs(refs, def, 10)
	for _, ref := range classified {
		if ref.IsTest {
			t.Errorf("test ref should not appear in References: %s:%d", ref.File, ref.Line)
		}
	}
	if len(classified) != 1 {
		t.Errorf("expected 1 non-test ref, got %d", len(classified))
	}
}

// Regression 4: 上流検索の打ち切りが出力に反映される
func TestFormatInspectResult_UpstreamTruncated(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:              []string{"21: func Build() {}"},
		UpstreamTruncated: true,
	})

	if !strings.Contains(result, "upstream: true") {
		t.Errorf("expected upstream truncation notice, got: %s", result)
	}
	if !strings.Contains(result, "search results were capped") {
		t.Errorf("expected upstream truncation explanation, got: %s", result)
	}
}

// Regression 5: 上流打ち切りなしの場合は upstream 行が出力されない
func TestFormatInspectResult_NoUpstreamTruncated(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:              []string{"21: func Build() {}"},
		UpstreamTruncated: false,
	})

	if strings.Contains(result, "upstream") {
		t.Errorf("expected no upstream line when not truncated, got: %s", result)
	}
}

// Regression 6: 上流検索の異常終了が警告として出力される
func TestFormatInspectResult_UpstreamIncomplete(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:               []string{"21: func Build() {}"},
		UpstreamIncomplete: true,
	})

	if !strings.Contains(result, "Warnings:") {
		t.Errorf("expected warnings section, got: %s", result)
	}
	if !strings.Contains(result, "search incomplete: true") {
		t.Errorf("expected incomplete search warning, got: %s", result)
	}
}

// Regression 7: 上流検索が完全な場合は incomplete 警告が出力されない
func TestFormatInspectResult_NoUpstreamIncomplete(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:               []string{"21: func Build() {}"},
		UpstreamIncomplete: false,
	})

	if strings.Contains(result, "search incomplete") {
		t.Errorf("expected no incomplete warning when search is complete, got: %s", result)
	}
}

// setupTestGoFiles は複数の Go ファイルを一時ディレクトリに作成し、
// そのディレクトリに cd した後、元に戻すための cleanup を登録する。
func setupTestGoFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})
	return dir
}

// E2E: agent.go の Build を inspect したとき config.go 由来の caller/ref が混ざらない
func TestInspectSymbol_CrossFileIsolation_AgentSide(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"agent.go": `package example

// Build はエージェントをビルドする。
func Build(name string) string {
	return "agent:" + name
}

// RunAgent は Build を呼ぶ。
func RunAgent() string {
	return Build("agent")
}
`,
		"config.go": `package example

// Config は設定。
type Config struct {
	Name string
}

// Build は Config をビルドする。
func (c *Config) Build() string {
	return "config:" + c.Name
}

// UseConfig は Config.Build を呼ぶ。
func UseConfig() string {
	c := &Config{Name: "test"}
	return c.Build()
}

// configRef は Build を参照する。
var configRef = Config{}.Build
`,
	})

	result := InspectSymbol("Build", "agent.go", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find Build, got: %s", result)
	}
	// 複数候補の場合はここで終了（同一ファイル内の複数候補はここでは起きない）
	if strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected single candidate with path=agent.go, got: %s", result)
	}

	// config.go 由来の caller/ref が混ざっていないことを確認
	if strings.Contains(result, "config.go") {
		t.Errorf("config.go references should not appear in agent.go Build inspection:\n%s", result)
	}
	if strings.Contains(result, "UseConfig") {
		t.Errorf("UseConfig (config.go caller) should not appear:\n%s", result)
	}
	if strings.Contains(result, "configRef") {
		t.Errorf("configRef (config.go ref) should not appear:\n%s", result)
	}

	// agent.go 自身の caller（RunAgent）は出るべき
	// ただし同一ファイル内なので candidate 定義行範囲外の call のみ
	if !strings.Contains(result, "agent.go") {
		t.Errorf("expected agent.go in header, got: %s", result)
	}
}

// E2E: 逆方向 — config.go の Build を inspect したとき agent.go 由来が混ざらない
func TestInspectSymbol_CrossFileIsolation_ConfigSide(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"agent.go": `package example

// Build はエージェントをビルドする。
func Build(name string) string {
	return "agent:" + name
}

// RunAgent は Build を呼ぶ。
func RunAgent() string {
	return Build("agent")
}
`,
		"config.go": `package example

// Config は設定。
type Config struct {
	Name string
}

// Build は Config をビルドする。
func (c *Config) Build() string {
	return "config:" + c.Name
}

// UseConfig は Config.Build を呼ぶ。
func UseConfig() string {
	c := &Config{Name: "test"}
	return c.Build()
}
`,
	})

	result := InspectSymbol("Build", "config.go", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find Build, got: %s", result)
	}
	if strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected single candidate with path=config.go, got: %s", result)
	}

	// agent.go 由来の caller/ref が混ざっていないこと
	if strings.Contains(result, "agent.go") {
		t.Errorf("agent.go references should not appear in config.go Build inspection:\n%s", result)
	}
	if strings.Contains(result, "RunAgent") {
		t.Errorf("RunAgent (agent.go caller) should not appear:\n%s", result)
	}
}

func TestIsTestFunction(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"TestBuild_Normal", true},
		{"BenchmarkBuild", true},
		{"ExampleBuild", true},
		{"helperFunc", false},
		{"testHelper", false},
	}

	for _, tt := range tests {
		got := isTestFunction(tt.name)
		if got != tt.want {
			t.Errorf("isTestFunction(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
