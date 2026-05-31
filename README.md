# XELYON CLI

AI搭載のコーディングアシスタントCLI

[![CI](https://github.com/susugadx/xelyon-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/susugadx/xelyon-cli/actions/workflows/ci.yml)
[![E2E](https://github.com/susugadx/xelyon-cli/actions/workflows/e2e.yml/badge.svg)](https://github.com/susugadx/xelyon-cli/actions/workflows/e2e.yml)
[![codecov](https://codecov.io/gh/susugadx/xelyon-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/susugadx/xelyon-cli)
[![Go Version](https://img.shields.io/github/go-mod-go-version/susugadx/xelyon-cli)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/susugadx/xelyon-cli)](https://github.com/susugadx/xelyon-cli/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特徴

### 💬 自然言語でコーディング
コマンドを覚える必要なし。日本語で指示するだけ。
- 「このファイルのバグ直して」
- 「テスト書いて」
- 「リファクタリングして」
- 「git commitして」

**差分を見せて確認してから実行（編集・bash・gitなど）**

### 🌐 10種類のLLMプロバイダー
DeepSeek, Kimi, OpenAI, Azure OpenAI, Gemini, Claude, Ollama, Groq, OpenRouter, Bedrock をシームレスに切り替え。
ローカルLLM（Ollama）も対応で、オフラインでも使用可能。

**OpenAI / Azure OpenAI Responses API 対応**: `gpt-5.3-codex` などの Codex モデルを自動検出し、Azure OpenAI は API key と Microsoft Entra ID bearer token の両方に対応。
**プロバイダー診断**: `xelyon doctor <provider> --capabilities` は model / deployment 能力を live request なしで確認し、`--require-capability responses_api` などで必要な capability を gate できます。DeepSeek / Groq / OpenRouter などの Chat Completions 系も共通 capability DTO、request preview、smoke を確認できます。OpenRouter は Claude 系 model で Anthropic Skin route も表示します。`--smoke` は usage / cost を text / JSON に出力し、対応 provider では response ID も表示します。
**DeepSeek Reasoner 対応**: `reasoning_content`（思考内容）のストリーミング表示・ツール実行フローでの保持に対応。
**Kimi Native Provider 対応**: Moonshot の Chat Completions API で `kimi-k2.6` / `kimi-k2.5` の streaming、tool calls、thinking、`reasoning_content` に対応。
**プロバイダー別プロンプト最適化**: OpenAI / Gemini では短い実況を促し、Gemini など特定モデルのルール遵守を強化するプレフィックスを自動注入。
**Gemini FCリトライ**: FC失敗時にテキストモードではなくFCモードでリトライ（キャッシュ汚染防止）。idle timeout / thinking timeout / 一般エラーそれぞれで上限付きリトライ。
**FC rescue JSON修復**: テキストモードで抽出されたツールJSONに生制御文字（改行・タブ等）が含まれる場合、自動修復してパース成功させる。

### 🛠️ 組み込みツール
- **ファイル操作**: 編集ツールは provider/model に応じて自動切替。OpenAI / Azure OpenAI / Gemini 系と OpenRouter の OpenAI/Gemini family は Codex 互換の `apply_patch`、Kimi / Claude / Bedrock / DeepSeek / Groq / Ollama / unknown 系は旧 `str_replace` / `write_file` / `delete_file` を使います。`XELYON_EDIT_TOOL=str_replace` などの明示 override がある場合はそれを最優先します
- **コード検索**: `search_code` は language-aware router として動作し、`mode=auto` を既定に symbol-aware / literal / regex の各レーンを内部選択（複数パターン、結果分類、不正regex検出にも対応）
- **シンボル調査**: `search_code` は短い symbol query を優先し、対応言語では定義・caller・参照・関連テストをまとめて返却。Go は first-class に `Config.Build` / `(*Config).Build` や regex っぽい query の rescue も吸収し、`intent=impact` は Go、TypeScript `.ts` / `.d.ts`、対象を絞った TSX `.tsx`、JavaScript `.js` / JSX `.jsx` で構造化 impact を返却
- **サブエージェント委譲**: `spawn_agent` / `wait_agent` で探索タスクを別コンテキストの軽量モデルへ委譲し、親には最終レポートだけを返す
- **AST基盤**: `internal/ast` の Pure Go Tree-sitter（gotreesitter）ベース共通解析基盤を、Go の symbol-aware 検索、`read_file(symbol=...)`、legacy `str_replace` の書き込み前構文検証で利用
- **開発支援**: bash（git, テスト, フォーマット等すべて対応）
- **LSP連携**: シンボル検索（定義・参照・実装）

### 🔎 Runtime-owned investigation
XELYON は検索ツールをモデルへ渡すだけでなく、runtime 側で調査経路を組み立てます。`gather_context` を既定の調査入口にし、symbol-like query では `search_code(intent=impact)` の structured impact を使って definition / caller / reference / related test を `SymbolBundle` として束ねます。

Structured impact では `RecommendedReads` によって先に読むべき証拠を runtime が選び、`gather_context` が compact read として prefetch します。`resolved_by=lsp|ast|mixed|fallback`、`confidence`、`truncated`、`budget_limit_hit` などの diagnostics に応じて prefetch 件数も保守的に調整します。

```text
xelyon
> Button の影響範囲を tsx に絞って調べて
# runtime: gather_context query="Button" file_filter=tsx
# → definition / JSX usage or caller / related test / diagnostics / prefetched evidence
```

詳細は [Search optimization and structured impact](docs/search.md) を参照してください。

| language / target | file_filter / pattern | structured impact depth | notes |
| --- | --- | --- | --- |
| Go | `go`, `*.go`, `**/*.go`, direct `.go` path | definition, callers, references, tests, implementations | LSP / Go navigation を優先 |
| TypeScript | `ts`, `*.ts`, direct `.ts` path | definition, imports, callers, type refs, references, tests | `.tsx` は含めず対象を絞る |
| TypeScript declaration | `*.d.ts`, direct `.d.ts` path | declaration definition, type refs, imports, references | 実装 `.ts` / `.tsx` とは別 target |
| TSX | `tsx`, `*.tsx`, direct `.tsx` path | definition, JSX usage / callers, type refs, references, tests | React/JSX component 調査向け |
| JavaScript | `js`, `*.js`, direct `.js` path | definition, imports, callers, references, tests | `.mjs` / `.cjs` は structured impact 対象外 |
| JSX | `jsx`, `*.jsx`, direct `.jsx` path | definition, JSX usage / callers, references, tests | JavaScript family の JSX target |
| Broad TypeScript | `typescript` | fallback scope | `.ts` と `.tsx` を広く探すが structured impact target ではない |
| Broad JavaScript | `javascript` | fallback scope | `.js` / `.jsx` / `.mjs` / `.cjs` を広く探すが structured impact target ではない |

### 📋 確認ベースの安全設計
- 既定の `execution.mode: balanced` では、読み取り系は自動実行し、編集や通常 bash などは確認します
- `execution.mode: trusted` は workspace 内の通常編集を自動化し、高影響操作だけ確認します
- `execution.mode: full_auto` は原則自動実行し、`execution.always_confirm` に指定したカテゴリだけ確認します
- `--auto-approve`で信頼環境向け全ツール自動承認（SafetyLow含む）
- **既定編集フロー**: `gather_context` で文脈を集め、必要に応じて `read_file` / `search_code` で補い、active edit mode に応じて `apply_patch` または `str_replace` / `write_file` / `delete_file` で差分確認のうえ適用

### 📋 Plan Mode（オプショナル）
`/plan on` で有効化するとPlan Mode経由で処理されます。
1. **調査フェーズ**: コードベースを調査（SafetyHighツールを自動実行）
2. **単純なQ&A**: 調査のみで回答可能な場合はそのまま終了
3. **計画生成**: 実装が必要な場合、調査結果・根拠・制約と実装ステップを JSON で出力
4. **承認と実装**: ユーザーが計画を確認・承認すると、承認済み計画の要約情報だけを通常モードへ引き継いで実装を開始

デフォルトは通常モード（ツール個別確認）。軽いタスクにはオーバーヘッドなく即座に応答。

### 🔄 自動リトライ機能
ツール実行が失敗した場合、自動的にリトライして成功するまで試行します。
- 通常モードのツール実行と completion hook の再試行で有効
- リトライ中: `❌ Failed (retry 1/10)` → `🔄 Retrying...`
- 成功時: `✅ Succeeded (on retry 3)`
- 上限到達時は AI に修正を促すか、通常応答へ戻ります

### ⚡ Parallel Tool Execution
LLMが1回の応答で複数のread-onlyツールを返した場合、並列実行してレイテンシを削減します。
- **parallel-safe ツール**: `read_file`, `list_dir`, `search_code`, `web_search`, `git_status` 等
- **bash**: `ls`, `find`, `rg`, `grep`, `cat`, `git status`, `git diff`, `git log` 等のread-onlyコマンド（allowlist ベース）のみ並列化
- **sequential ツール**: `apply_patch`, `write_file`, `str_replace`, `delete_file`, MCP ツール等の副作用ありツールは従来通り順次実行
- mixed case: parallel-safe 群を先に並列実行 → sequential 群を順次実行 → 結果は元の tool call 順で配送
- 最大並列数: 4（セマフォ制御、固定値）
- 通常モード・Plan Mode 両方で同じポリシーを適用（`executeToolCallsWithParallel` 共通 executor）
- **安全性**: ループ検知・deprecated ツールフィルタリングは実行前（Phase 0）に評価。Investigation Phase は独自ループ（`executeToolOnly`）を使用し並列実行の対象外
- **stdout 制御**: 並列パスでは `ExecuteQuiet()` を使用し wrapper 層の出力（ヘッダー・引数・折りたたみ結果）を抑制。`Tool.Run()` 内部の直接 stdout 出力（例: `read_file` の `📄 Read: ...`）は抑制できない（Tool interface が io.Writer を受け取らないため）。parallel-safe ツールの内部出力は少量のステータス行のみであり、現時点で致命的ではないと判断
- **キャンセル（best effort）**: `context.Context` は goroutine 起動前・実行前にチェック。`Tool.Run()` は ctx を受け取らないため、実行開始後の中断は不可。cancel は best effort であり、fully cancellable ではない
- **ExecutionContext**: process-global 単一変数（`sync.RWMutex` で race-safe）。並列 batch 開始前に Set → goroutine は Read のみ → batch 完了後に Clear。ただし race-safe であることと multi-owner-safe であることは異なり、同一プロセス内で複数 Agent が同時に異なる ExecutionContext を必要とする設計には対応していない。single-agent CLI 前提で許容
- **FC / text-based 差異**: FC（tool_call_id あり）はループ中断時に role=tool で応答。text-based は role=user。text-based の後続ツールにはダミーメッセージを追加しない（intentional spec — 旧 sequential 実装との互換性を by design で維持）

### 🤖 サブエージェント委譲
探索・調査タスクは `spawn_agent` / `wait_agent` で軽量サブエージェントへ委譲できます。
- **コンテキスト分離**: 親に返るのはサブの最終レポートだけ。`read_file` 全文や `search_code` の中間結果は親コンテキストへ再注入されません
- **リアルタイム可視化**: `wait_agent` 実行中はサブエージェントのツール実行を親UIへ逐次表示し、編集系ツールの適用結果も追跡できます
- **既定モデル**: `sub_agent.default_model` が空ならメイン provider の最安モデルを自動選択します。明示設定するとそのモデルを優先します
- **推論強度**: 既定は off（`sub_agent.default_effort` で low / medium / high を指定可能）
- **同時実行数**: 既定 1（`sub_agent.max_concurrent`）
- **再帰禁止**: サブエージェント自身には `spawn_agent` / `wait_agent` を渡しません
- **コスト透明性**: `/status` で親セッションとサブエージェントのトークン使用量・コストを分離表示し、合算コストも確認できます

### 📋 複数行ペースト対応
コードを直接ペーストして処理。Bracketed Paste Mode で改行を含むテキストも1つの入力として認識。
- **そのままペースト（推奨）**: TUI/通常モードともに入力欄へ Cmd+V / Ctrl+V
- **` ``` ` モード**: 明示的に複数行入力を開始

> WSL等で問題がある場合: `XELYON_BRACKETED_PASTE=0` で無効化可能

### 📎 TUI添付（ファイル/画像）
- **添付（ファイル/画像）**: `/attach`・ドラッグ&ドロップ・`Ctrl+V` 画像の全経路で、1ドラフト最大12件
- **コマンド操作**: `/attach <path>` `/detach <index>` `/detach-all`
- **Ctrl+V画像貼り付け（Windows/WSL）**: クリップボードテキストが空なら画像貼り付けを試行
- **PDFプレビュー展開**: PDF は送信時に `Attached context` へテキスト抽出して展開（先頭 20 ページ / 30000 文字まで）
- **PDF抽出の失敗時**: 読み取り失敗時や抽出可能テキストがない場合は、その旨を context に明示
- **安全な一時ファイル管理**: クリップボード画像の一時ファイルは送信/削除/終了でcleanupし、古い残骸は起動時に自動GC

### 🖼️ マルチモーダル対応
画像ファイルを指定してUIデザインからコード生成。
エラースクリーンショットから原因分析も可能。

### 🔌 LSP連携（IDE並みのコード理解）
Language Server Protocol (LSP) を活用してIDE並みのコード理解を実現。
- **削除時参照チェック**: ファイル削除前に外部参照を自動検出し警告
- **完了検証**: AI が「完了」「done」と宣言した際、変更ファイルの LSP 診断を自動実行しエラー残存時は修正を続行
- **Final Checks**: `completed_with_changes` の完了候補でユーザー定義のシェルコマンド（`go test ./...` 等）を自動実行。失敗時は AI が修正を続行（`final_checks.commands` で設定）
- **自動検出**: プロジェクト内の言語を起動時に自動検出し、導入済み LSP サーバーを起動準備。未インストールのLSPサーバーは導入コマンドを提案（`lsp.skip_install_prompt` で非表示化）
- **サーバー設定**: TUI では `/config` の `lsp.servers` で管理。未導入サーバーは起動時案内の shell command で導入します
- **23言語対応**:
  - **メイン**: Go, TypeScript/JavaScript (React/JSX), Python, Rust
  - **バックエンド**: Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua
  - **フロントエンド**: CSS/SCSS, HTML, Vue, Svelte
  - **設定/スクリプト**: YAML, TOML, SQL, Bash, Markdown

### 📊 Context Window 管理
長時間の会話でもトークン上限を気にせず作業。
- **`/tokens`**: 現在のトークン使用量と上限を確認
- **Project Map 自動注入**: 起動時は root manifest 寄りの軽量マップだけを注入し、詳細シンボルは必要時に取得。大規模 repo でも固定コストを抑制
- **自動圧縮**: Context 100K または 80% 到達で自動的に履歴を圧縮（デフォルトON）
- **圧縮専用モデル**: OpenAI は GPT-5 Mini、Gemini は Flash-Lite、Claude/Bedrock(Claude) は Haiku で低コスト圧縮
- **手動圧縮**: `/compress [N]` で履歴を圧縮（最新N件を保持）
- **OpenAI Compact API**: `/compress --compact` でOpenAI独自の圧縮（ユーザーメッセージ保持）
- **80%/90%警告**: 上限接近時に自動で警告表示
- **トークン上限エラー時の提案**: エラー発生時に `/compress` または `/clear` を案内
- **Assistant narration clearing**: ツール呼び出し時に表示した短い実況テキストは History 保存時に自動削除し、入力トークンの再送を抑制
- **ツール結果の自動truncate**: 3ターン以上前のツール結果（50行超）を送信時に自動圧縮（先頭20行+末尾5行を保持）。元の履歴は保持され、API送信時にのみ適用
- **Claude系の server-side tool clearing**: Claude / Bedrock(Claude) / OpenRouter(Claude models) では `clear_tool_uses` により古い `tool_use` / `tool_result` ペア構造をサーバー側で削減し、compaction 発動前に入力トークンを節約
- **プロンプトキャッシュ最適化**: Claude/Bedrock(Claude) 利用時、安定区間の末尾userメッセージにBPを配置し、古い履歴のキャッシュHIT率を向上（`prompt_cache.enabled: true`で有効）。Opus 4.6の最低キャッシュトークン数（4096）に対応するため、system promptの最終ブロックにcache_controlを配置
- **Long Context 料金自動判定**: Claude/Gemini Pro で200Kトークン超のリクエスト時、long context 料金ティアを自動適用。キャッシュトークン（cache_read + cache_creation）も含めた総入力トークンでティア判定
- **Gemini service tier**: `gemini.service_tier` で Gemini BYOK の `standard` / `flex` / `priority` 同期 inference と料金表示を切り替え

### 🧠 履歴を消さずに、AIへ渡す文脈だけ軽くする
XELYON は raw history / session / audit / JSONL を保持したまま、provider に送る履歴だけを request ごとに projection できます。

- 古い `read_file` / `search_code` / `gather_context` 結果は、条件を満たすと evidence pointer placeholder に置き換えられます
- 成功した test / build / lint command output は、安全条件付きで summary 化できます
- 古い成功済み `write_file.content` も、raw storage を保ったまま provider-facing projection だけで省略できます
- 必要な evidence は現在ファイルから rehydrate し、request-local active context として provider 入力へ戻せます
- active context は OpenAI Responses、Chat Completions 系、Claude、Gemini、Bedrock など既存の provider transport に対応します
- raw storage は変更せず、rehydrated evidence も history へ保存しません

この provider history reduction / rehydrate context はまだ experimental dogfood 機能です。設定例と内部契約は [docs/dev/history-active-context.md](docs/dev/history-active-context.md) を参照してください。

### 📈 リアルタイムトークン表示
API実測値に基づくトークン使用量とコストをリアルタイム表示。
- **ステータスバー**: プロンプト直前に `● model │ Mode │ tokens/limit │ ~$cost` を表示
- **起動時コンテキスト表示**: ツリー形式で初期コンテキストの内訳を表示
- **ナビゲーション削減**: Project Map + `gather_context` の structured impact / prefetch routing により `read_file` の往復を減らし、編集に集中
- **リクエスト完了時**: `✓ In: 1,234 + Out: 567 = 1,801 tok (~$0.002)` で使用量を表示
- **Ollama対応**: ローカル実行時はコスト表示を非表示
- **圧縮閾値**: `compression.trigger_percent`（デフォルト80%）超過時に自動圧縮/警告

### 📝 プロジェクト設定（xelyon.yaml）
プロジェクト固有のルール・コンテキストを構造化 YAML で管理。`/init` でテンプレート作成。

```yaml
# xelyon.yaml（プロジェクトルートに配置）
context: "Go製CLIツール。Cobraベース。"
rules:
  - "変更後は make ci-check を実行"
  - "公開関数にはコメント必須"
conditional:
  - name: Agent internals
    paths:
      - "internal/agent/**/*.go"
    rules:
      - "公開関数・型には日本語コメント必須"
    context: "トークン推定は共通推定器を使う"
ignore:
  patterns:
    - "dist"
    - "*.min.js"
final_checks:             # config.yaml の final_checks を上書き
  commands:
    - "go vet ./... && go test ./..."
  timeout: 120
```

- **context**: AI に注入するプロジェクト説明
- **rules**: 番号付きで system prompt に注入される必須ルール
- **conditional**: `paths` に一致した時だけ注入する rules/context
- **ignore**: Project Map / `list_dir` / `search_code` で共有する ignore パターン
- **final_checks**: 明示完了時の final checks（`config.yaml` の final_checks より優先）
- **Project Map**: 起動時は軽量 manifest が自動生成されるため、`xelyon.yaml` にファイル一覧や関数目次は書かない

## インストール

```bash
# Homebrew (macOS)
brew install susugadx/tap/xelyon

# または GitHub Releases からダウンロード
# https://github.com/susugadx/xelyon-cli/releases
```

## クイックスタート

### 1. APIキーを設定

```bash
export DEEPSEEK_API_KEY="sk-..."  # または他のプロバイダー
```

### 2. 起動

通常の対話セッションは TUI が primary surface です。`--no-tui` は legacy classic REPL の fallback として残していますが、新しい対話型コマンドは TUI 側に追加します。

```bash
xelyon

🗺️  Project map loaded (42 files, 118 symbols)
📋 Context size: ~10.8k tok

> main.goを読んで、バグがあれば修正して
```

### 3. 基本コマンド

```bash
/           # TUI のコマンド候補を表示
/model      # 現在のモデル確認/切り替え
/use gemini # プロバイダー切り替え
/thinking high # Extended Thinking 有効化
/attach ./notes.txt # 現在の入力ドラフトへ添付（1ドラフト最大12件）
/detach 1   # 指定添付を外す
/config     # global config を対話式で編集
/project    # xelyon.yaml を対話式で編集
/init       # xelyon.yaml テンプレート作成
/review [instructions] # 現在の変更レビューを開く（引数は追加観点として即時実行）
/exit       # 終了
```

`/review` は現在の作業ツリー差分全体をレビューします。通常は現在の provider/model を使いますが、`review.provider` と `review.model` を設定すると `/review` だけ別の provider/model で実行できます。`review.provider` だけを設定した場合はその provider の既定モデルを使い、`review.model` を設定する場合は `review.provider` も必須です。review モデル呼び出しには通常会話履歴を渡しません。

preset の `Review current changes` は追加指示なしの通常レビューで、`Review current changes with custom focus` は同じ current changes 全体に追加の観点・重点項目を渡します。custom focus は対象ファイルや差分範囲を絞るものではありません。特定 finding だけの再検証や focused verification mode はまだ未実装です。

`/review` は `evidence -> probe plan -> probe results -> report -> saturation` の段階を持つ監査可能な review harness です。Go-first の evidence augmentation を使いつつ、共通 harness は全言語で雑な clean 判定を抑制します。`XELYON_REVIEW_RUN_ARTIFACTS=1` を設定すると、各段階の debug artifact を実行中はメモリに保持し、終了後にリポジトリ配下の `.xelyon/review-runs/<UTC timestamp>/` へ保存します。保存先 component が symlink の場合は repo 外へ書かず warning にします。artifact には evidence や probe output が含まれ得るため、必要な場合だけ明示的に有効化してください。

`review.web_search_evidence.enabled` を有効化した場合、raw web search results は discovery-only で、URL fetch 済みの `external_doc` snippet だけが citation-capable evidence になります。ただし `external_doc` は自動的に公式仕様とは扱わず、source domain / title / content が明確に支える場合だけ confirmed external spec として扱います。credibility 不明、fetch failed、truncated、inconclusive の場合は unverified / residual / blocked として分類します。

### 4. NAVモードでテキストをコピー

```text
Esc           入力欄が空なら NAV モードへ
count+j/k/h/l 数字プレフィックス付き移動
w / b / e     単語移動
0 / ^ / $     行頭 / 非空白先頭 / 行末
v             文字単位のビジュアル選択
V             行単位のビジュアル選択
y             選択範囲をコピー
yy            現在行をコピー
Tab           ツールブロックをフォーカス/折りたたみ
q / i / Esc   入力モードへ戻る
```

起動直後のヘッダーや AI 出力、ツール結果も NAV モードからコピーできます。コピー内容は ANSI 色を除いたプレーンテキストです。`3j`、`12G`、`vwe` のような Vim 風操作にも対応しています。

## よく使う機能

### プロバイダー切り替え

```bash
xelyon --provider gemini --model gemini-3.5-flash
xelyon --provider gemini --model gemini-2.5-flash

# または対話中に
> /use claude
```

### Web検索

`web_search` は Kimi / OpenAI / Gemini / Claude のネイティブ検索を使います。メインプロバイダーが Kimi の場合は Moonshot Chat Completions の built-in `$web_search` をそのまま使えます。DeepSeek / Groq / Ollama / OpenRouter / Bedrock などの検索非対応 provider では、`web_search.provider` で検索専用プロバイダーを指定してください。

```yaml
# ~/.xelyon/config.yaml
web_search:
  provider: gemini

```

Gemini native web search は API の `usageMetadata` が返る場合、通常の token usage / cost として表示します。Kimi で `$web_search` が起動すると、Moonshot の call fee（`$0.005 / invocation`）が token cost とは別に発生します。XELYON は call fee を外部固定費として観測し、検索結果 tokens は次の `prompt_tokens` に含まれるため token totals へ二重加算しません。

Gemini API キーは無料で取得できます: https://aistudio.google.com/apikey

### サブエージェント設定

`sub_agent.max_concurrent` の既定は `1` です。複数の調査タスクを並列させたいときだけ、例のように値を上げてください。

```yaml
# ~/.xelyon/config.yaml
sub_agent:
  enabled: true
  default_model: ""
  default_effort: off
  max_concurrent: 2  # 既定: 1
```

`sub_agent.default_model` が空の場合は、メイン provider に応じて OpenAI は `gpt-5.4-mini`、Claude は `claude-haiku-4-5-20251001`、Gemini は `gemini-3.1-flash-lite`、Kimi は `kimi-k2.5` などの低コストモデルを自動選択します。Azure OpenAI では hard-coded モデル名ではなく `provider_models.azure.default_model` の deployment 名を使います。親モデルの `default_model` や `thinking` 設定は直接上書きしません。

### 最大出力トークン数の設定

```yaml
# ~/.xelyon/config.yaml
provider_models:
  claude:
    default_model: claude-sonnet-4-6
    max_output_tokens: 64000   # デフォルト: 64000
  gemini:
    default_model: gemini-3.1-pro-preview-customtools
    max_output_tokens: 65536   # デフォルト: 65536
  deepseek:
    default_model: deepseek-v4-flash
    max_output_tokens: 16384   # デフォルト: 16384
  kimi:
    default_model: kimi-k2.6
    max_output_tokens: 32768   # デフォルト: 32768
  openai:
    default_model: corp-gpt-deployment
    catalog_model: gpt-5.4     # deployment/alias の料金・context 判定に使う既知モデル
  azure:
    default_model: my-gpt-5-deployment
    catalog_model: gpt-5.4     # Azure deployment 名と実モデル名が異なる場合に設定
```

| プロバイダー | デフォルト max_output_tokens |
|------------|---------------------------|
| claude     | 64000                     |
| bedrock    | 64000                     |
| gemini     | 65536                     |
| openai     | 16384                     |
| azure      | 16384                     |
| deepseek   | 16384                     |
| kimi       | 32768                     |
| groq       | 8192                      |
| ollama     | 4096                      |
| openrouter | 64000                     |

### 確認動作のカスタマイズ

```yaml
# ~/.xelyon/config.yaml
execution:
  mode: balanced             # balanced / trusted / full_auto
  always_confirm: []         # どのモードでも確認するカテゴリ
  safe_shell_commands: []    # verification / env 用に追加で自動許可する shell command
```

既存の `tool_confirm` は互換読み込みされますが、新規設定では `execution.mode` を使用してください。

### 中間出力の表示レベル

```yaml
# ~/.xelyon/config.yaml
output:
  assistant_updates: ""  # 空なら Normal=phase, Plan=verbose
```

`assistant_updates` は assistant prose の途中表示だけを制御します。`phase` は通常モードの逐次実況を短いフェーズ要約に寄せ、`off` はさらに抑制します。`apply_patch` の diff 表示やツール出力の折りたたみ挙動は変わりません。

```bash
# 全ツール自動承認（信頼できる環境向け、SafetyLow含む）
xelyon --auto-approve
```

### 差分表示設定

```yaml
# ~/.xelyon/config.yaml
diff:
  context_lines: 10    # 差分表示のコンテキスト行数（0で省略なし、デフォルト: 10）
  max_total_lines: 0   # 差分表示の最大行数（0で無制限、デフォルト: 0）
```

### MCP設定

```yaml
# ~/.xelyon/config.yaml
mcp:
  enabled: true    # MCP機能のON/OFF（デフォルト: true）
  headless: false  # ヘッドレスモードでMCPを使うか（デフォルト: false）
```

`enabled: false` にするとMCPサーバーへの接続をスキップし、トークン消費を削減できます。
`~/.xelyon/mcp.json` の設定はそのまま残るため、再度 `enabled: true` にすれば復活します。
- **ツール単位フィルタリング**: MCPサーバーのツールを `include`/`exclude` で制御。不要なツールを除外してトークン消費を最適化

### Final Checks

AI が `completed_with_changes` の完了候補に達すると、ここで定義したコマンドを順番に実行します。
コマンド失敗時は AI にエラー内容をフィードバックし修正を続行します。final checks が通るまで完了扱いにはなりません。
Makefile は不要です。普段使っているコマンドをそのまま書けます。

```yaml
# ~/.xelyon/config.yaml
final_checks:
  commands:
    # Go
    - "go vet ./... && go test ./..."
    # Node.js / TypeScript
    # - "npm test"
    # Python
    # - "pytest"
    # Rust
    # - "cargo test"
    # Makefile がある場合
    # - "make ci-check"
  timeout: 600         # コマンドタイムアウト秒（デフォルト: 600）
```

`xelyon.yaml` にも `final_checks` を定義でき、`config.yaml` より優先されます（プロジェクト固有の final checks 設定に便利）。
変更ファイルは `XELYON_CHANGED_FILES` 環境変数（スペース区切り）で参照できます。
`/plan` の調査フェーズでは実行されず、承認後に引き継がれる通常モード実装フェーズで動作します。

### 設定管理

```bash
> /config         # global config の対話式設定メニュー
> /config show    # global config を表示（デフォルトとの差分を ⚡ で表示）
> /project        # project config (xelyon.yaml) を対話式で編集
```

対話式メニューでは Provider & Model、Execution Mode、Compression、Project Map、LSP Servers、Web Search、MCP、Final Checks などをカテゴリ別に管理できます。

## ドキュメント

| ドキュメント | 内容 |
|------------|------|
| [コマンド一覧](docs/commands.md) | 対話型/CLIコマンド、フラグ、使用例 |
| [プロバイダー設定](docs/providers.md) | 各プロバイダーのAPIキー取得方法 |
| [設定リファレンス](docs/config.md) | config.yaml と環境変数 |
| [Search optimization](docs/search.md) | structured impact、RecommendedReads、diagnostics-aware prefetch |
| [MCP連携](docs/mcp.md) | 外部ツール追加 |
| [LSP連携](docs/lsp.md) | 言語サーバー連携（23言語対応） |
| [使い方詳細](docs/usage.md) | 複数行入力、画像入力、サブエージェント委譲など |

## 開発に参加する

xelyon-cli の開発に参加したい方向け：

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
make build
./xelyon
```

> ⚠️ このリポジトリの `xelyon.yaml` は xelyon-cli 開発用です。

## コントリビュート

PRやIssue歓迎です！詳細は [CONTRIBUTING.md](CONTRIBUTING.md) を参照してください。

## ライセンス

MIT
