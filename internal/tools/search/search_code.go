package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// MatchType はマッチ行の種別（ソート順序を定義）
type MatchType int

const (
	MatchTypeDefinition MatchType = iota // 0: func/type/class 等の定義
	MatchTypeImport                      // 1: import/require/use 等
	MatchTypeCall                        // 2: 関数/メソッド呼び出し
	MatchTypeAssignment                  // 3: := や = による代入
	MatchTypeRef                         // 4: その他の参照
	MatchTypeComment                     // 5: コメント行
	MatchTypeString                      // 6: 文字列リテラル
)

const MatchTypeUsage MatchType = MatchTypeRef // 後方互換: 旧名称

// lineRangeHint は search_code 結果末尾に付与する編集ヒント。
// UI には表示されず、LLM の tool result にのみ含まれる。
const lineRangeHint = "\n\nTip: Use the active edit tool with the matched lines plus surrounding context to make exact edits."

// matchTypeTag はマッチ種別の表示タグ
var matchTypeTag = [7]string{"[def]", "[import]", "[call]", "[assign]", "[ref]", "[comment]", "[string]"}

// BlockInfo はマッチが所属するブロック（関数/クラス）の情報
type BlockInfo struct {
	Name      string // "func handleSSEResponse", "class MyClass" 等
	StartLine int
}

// SearchResult はファイルごとの検索結果
type SearchResult struct {
	FilePath   string
	Matches    []Match
	MatchCount int // マッチ行のみのカウント
}

// Match はマッチ行またはコンテキスト行
type Match struct {
	LineNum int
	Line    string
	IsMatch bool       // true=マッチ行, false=コンテキスト行
	Type    MatchType  // マッチ種別（ソート用）
	Block   *BlockInfo // マッチが所属するブロック（nil=トップレベル）
}

// SearchOptions はコード検索のオプション
type SearchOptions struct {
	Pattern          string
	Intent           string
	Mode             string
	Path             string
	FilePattern      string // file_filter から自動判定。glob 文字を含む場合に設定。
	FileType         string // file_filter から自動判定。glob 文字を含まない場合に設定。
	CtxLines         int    // ツール経路では内部固定値（3）。外部パラメータは廃止。
	TokenBudget      int    // 内部固定値（15000）。外部パラメータは廃止。
	IsRegex          bool
	Multiline        bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	IncludeHidden    bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	IncludeIgnored   bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	OutputMode       string // 内部専用。外部パラメータは廃止。
	LegacyIsRegexSet bool

	LocatorRegistry    *locator.Registry // Locator ID レジストリ（nilの場合はID付与しない）
	LSPClient          navigation.LSPClient
	ProjectMap         *repomap.ProjectMap
	ProjectMapRootPath string
	ProjectMapStateKey string
	InvocationCWD      string

	ignoreMatcher *pathmatch.Matcher
	ignoreGlobs   []string
	ignoreKey     string
}

// ExecuteSearchCode はコード検索を実行し、フォーマット済み結果を返す
func ExecuteSearchCode(opts SearchOptions) string {
	return ExecuteSearchCodeWithConfig(nil, nil, opts)
}

// ExecuteSearchCodeWithCache はキャッシュを指定してコード検索を実行する。
func ExecuteSearchCodeWithCache(cache tools.ToolCacheInterface, opts SearchOptions) string {
	return ExecuteSearchCodeWithConfig(nil, cache, opts)
}

// ExecuteSearchCodeWithConfig は設定とキャッシュを指定してコード検索を実行する。
func ExecuteSearchCodeWithConfig(cfg *config.Config, cache tools.ToolCacheInterface, opts SearchOptions) string {
	if opts.Pattern == "" {
		return "Error: pattern is required"
	}
	var ok bool
	opts, ok = normalizeSearchOptions(opts)
	if !ok {
		return "Error: invalid mode (expected auto, symbol, literal, or regex)"
	}
	if opts.Path == "" {
		opts.Path = "."
	}

	// context_lines is fixed internally.
	opts.CtxLines = 3

	// token_budget is fixed internally.
	// 15000 tokens cover almost all normal searches without truncation.
	// Only abnormal wide searches should hit this safety valve.
	opts.TokenBudget = 15000

	if !opts.IncludeIgnored {
		projectCfg := config.LoadProjectConfig()
		ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, projectCfg)
		opts.ignoreMatcher = pathmatch.NewMatcher(ignorePatterns)
		opts.ignoreGlobs = pathmatch.BuildRGIgnoreGlobs(ignorePatterns)
		opts.ignoreKey = strings.Join(ignorePatterns, ",")
	}

	if shouldExecuteImpactSearch(opts) {
		return executeImpactSearch(cache, opts)
	}
	patterns := effectiveSearchPatterns(opts)
	return executeSearchPatterns(cache, patterns, opts)
}
