# XELYON CLI プロジェクト設定

## 概要
Go製のAI搭載コーディングアシスタントCLI。複数のLLMプロバイダー（DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq）に対応した対話型エージェントで、ツールを使って実際にコード編集・Git操作を実行します。

## 技術スタック
- **Go 1.22+**
- **Cobra** - CLIフレームワーク
- **godotenv** - .env環境変数の自動読み込み
- **LLM APIs** - DeepSeek, OpenAI, Gemini, Claude (Anthropic), Ollama, Groq
- **Tree-sitter** - コード構造解析
- **fatih/color** - ターミナル色付け

## 開発ルール（必ず守ること）

### ドキュメント更新
機能追加・変更時は**必ず**以下を同時に更新：
- **README.md**: 使い方、コマンド説明、バージョン履歴
- **XELYON.md**: アーキテクチャ、内部設計、SystemPromptルール

ドキュメント更新なしのコミットは禁止。

### コミットルール
- メッセージは日本語OK
- 具体的に書く（❌「修正」→ ✅「MCPリトライロジック追加、タイムアウト30秒設定」）
- 機能単位で小さくコミット

### コード品質
変更後は必ず実行：
```bash
go fmt ./...
go mod tidy
go build -o xelyon
```

可能なら：
```bash
go test ./...
```

### エラーハンドリング
- すべてのI/O操作でエラーチェック必須
- HTTPクライアントには必ずTimeout設定
- context.Contextを使ってキャンセル可能に

---

## アーキテクチャ

### ディレクトリ構造
```
xelyon-cli/
├── main.go                 # エントリーポイント
├── cmd/
│   └── root.go            # Cobraコマンド定義
├── internal/
│   ├── agent/             # エージェントロジック
│   │   ├── agent.go       # 対話ループ、コマンド処理、Undo管理
│   │   ├── agent_plan.go  # Plan Mode実装（計画生成→承認→自律実行）
│   │   ├── parallel.go    # 並列実行エンジン（errgroup使用）
│   │   ├── plan.go        # 実行計画の構造体と依存関係解決
│   │   └── verify.go      # 自動検証（go fmt, go test, rollback）
│   ├── mcp/               # MCP連携
│   │   ├── client.go      # MCPサーバー接続・ツール管理
│   │   └── integration.go # Tool Registry統合
│   ├── repomap/           # Repo Map（コード構造解析）
│   │   ├── parser.go      # 言語パーサー管理
│   │   ├── symbols.go     # シンボル定義
│   │   ├── extractor.go   # Tree-sitterでシンボル抽出
│   │   └── repomap.go     # Repo Map生成
│   ├── review/            # コードレビュー機能
│   │   ├── types.go       # 共通型定義（Target, Issue, Severity等）
│   │   ├── review.go      # Orchestrator（レビューパイプライン）
│   │   ├── scanner.go     # ファイル変更スキャン（changeStack/git diff）
│   │   ├── analyzer.go    # ルールベース解析（セキュリティ/一般/テスト）
│   │   ├── fixer.go       # 修正提案生成（suggestion-only）
│   │   └── reporter.go    # Markdownレポート出力
│   ├── api/               # API クライアント（Provider Pattern）
│   │   ├── provider.go    # Provider interface定義
│   │   ├── client.go      # Client struct（タイムアウト管理）
│   │   ├── deepseek.go    # DeepSeekProvider実装
│   │   ├── openai.go      # OpenAIProvider実装
│   │   ├── gemini.go      # GeminiProvider実装
│   │   ├── claude.go      # ClaudeProvider実装
│   │   ├── ollama.go      # OllamaProvider実装
│   │   ├── groq.go        # GroqProvider実装
│   │   ├── serper.go      # Serper API（Web検索）
│   │   └── xelyon.go      # RAG検索API
│   ├── config/            # 設定管理
│   │   └── config.go      # 設定ファイル読み書き
│   ├── tools/             # ツール実行エンジン（Tool Registry Pattern）
│   │   ├── registry.go    # Tool interface + Registry
│   │   ├── builtin.go     # 組み込みツールのRegistry登録
│   │   └── tools.go       # ツール実装、バックアップ管理
│   ├── ui/                # UI コンポーネント
│   │   ├── spinner.go     # ローディングスピナー
│   │   └── pager.go       # 自動ページング（100行超え）
│   ├── history/           # セッション管理
│   │   ├── session.go     # セッション構造
│   │   └── storage.go     # JSONL永続化
│   └── file/              # ファイルI/O
│       ├── reader.go
│       └── writer.go
└── XELYON.md              # このファイル（自動読み込み）
```

### コア機能

#### 1. エージェントシステム (internal/agent/)
- **対話ループ**: ユーザー入力 → AI推論 → ツール実行 → 結果表示
- **Plan Mode**: 自律実行 + 並列処理（v0.31.0）
  - **agent_plan.go**: `RunPlanMode()` - 計画生成→承認→自律実行
  - **parallel.go**: `ParallelExecutor` - golang.org/x/sync/errgroupで並列実行
  - **plan.go**: `Plan`, `PlanStep` - 実行計画のデータ構造と依存関係解決
  - **ワークフロー**:
    1. AI が JSON形式で実行計画を生成
    2. 計画を表示（ステップ、ツール、依存関係、並列実行可否）
    3. y/n/c で承認・拒否・フィードバック
    4. 承認後、依存関係順に自律実行（並列ステップは同時実行）
    5. SafetyLow 操作のみ個別確認
  - **起動方法**: `--plan` フラグ、または `/plan on` コマンド
- **コマンド処理**: `/save`, `/load`, `/sessions`, `/undo`, `/config`, `/model`, `/plan`, `/version`, `/clear`, `/history`, `/help`
- **変更履歴管理**: 最大10件のファイル変更を追跡、Undo機能
- **セッション管理**: 会話履歴の自動保存・復元
- **ループ検知**: 同じツール呼び出しが3回繰り返されると自動中断
- **APIリトライ**: エラー時に最大2回自動リトライ（指数バックオフ）

#### 2. ツールシステム (internal/tools/)
- **25種類のツール**:
  - **ファイル編集**: read_file, write_file, str_replace, append_file, prepend_file, insert_after, insert_before, copy_file
  - **ファイル管理**: list_dir, create_dir
  - **Git操作**: git_status, git_diff, git_add, git_commit, git_push, git_log, git_branch, git_checkout, git_stash
  - **開発支援**: run_test, format
  - **検索**: search_code, search_file, web_search
  - **シェル**: bash
- **自動バックアップ**: write_file/str_replace/append_file/prepend_fileで.bakファイル作成
- **安全性**: 危険なコマンド（rm -rf, sed -i等）をブロック
- **FileChange追跡**: ファイル変更をメタデータと共に記録
- **空白正規化**: str_replaceで行頭インデント違いを吸収（完全一致優先、フォールバック）
- **行レンジ置換**: old_str が空 かつ start_line/end_line 指定時に、1-indexed inclusive の行レンジ置換モードで編集可能
- **確認UI改善**: 英語/日本語併記、ボックス囲みの見やすい差分表示
- **簡潔な表示**: ツール呼び出し時のJSON表示を廃止、人間が読める形式に
- **エラーヒント**: 複数マッチ時にファイルプレビュー（先頭50行）を表示（Candidates/Next actions/IMPORTANTも含む）
- **フレームワーク自動検出**: run_test/formatが言語・ツールを自動検出

#### 3. UI/UX (internal/ui/)
- **スピナー**: API呼び出し中のローディング表示（80ms更新）
  - `run_test` 等の長時間実行ツールでも、実行中に stderr へスピナー表示して「停止して見える」問題を軽減

- **ページング**: 100行を超える出力を自動的に分割表示
- **色付け**: cyan/green/yellow/redで情報を視覚的に区別
- **複数行入力** (v0.32.0):
  - **Bracketed Paste Mode**: ターミナルペースト時に自動検出（ESC[200~...ESC[201~）
    - `EnableBracketedPaste()` でターミナルにモード有効化シーケンスを送信
    - `isTerminal()` でターミナル判定（テスト環境では無効化）
    - 対応ターミナル: iTerm2, GNOME Terminal, Konsole, Windows Terminal, tmux等
  - **``` マーカー**: 明示的な複数行モード（行番号付きエディタ表示）
  - **multiline.go**: `MultilineReader.ReadInput()` で両方式を統一処理
  - **agent.go**: 起動時に `EnableBracketedPaste()`, 終了時に `DisableBracketedPaste()`

  - **Paste Mode**（WSL等のBracketed Paste非対応環境向け）:
    - `internal/ui/paste_mode.go`: `PasteMode.Capture()` で「空行2回」等のルールで複数行入力を安全に取り込み
    - `/paste` コマンド（`internal/agent/paste.go`）と、確認プロンプトのコメント入力（`internal/tools/confirm_interactive.go` / Plan承認の `confirmPlan`）から共通利用
    - コメント入力中に `/paste`（エイリアス: `/p`）を入力すると Paste Mode を起動し、終了後の内容をコメントに挿入


#### 4. 履歴管理 (internal/history/)
- **JSONL形式**: ストリーミング対応、1行1メッセージ
- **メタデータ分離**: session_id, model, timestamp, preview等
- **保存先**: `~/.xelyon/history/`

#### 5. マルチプロバイダーシステム (internal/api/)
- **Provider Interface**: 統一されたAPI（`ChatWithTools()`）
- **6種類のプロバイダー**:
  - **DeepSeek**: deepseek-chat, deepseek-coder, deepseek-reasoner
  - **OpenAI**: gpt-4o, gpt-4o-mini, gpt-4-turbo
  - **Gemini**: gemini-2.0-flash-exp, gemini-1.5-pro, gemini-1.5-flash
  - **Claude**: claude-sonnet-4-20250514, claude-opus-4, claude-haiku-3-5
  - **Ollama**: llama3, codellama, mistral等（ローカル）
  - **Groq**: llama3-70b-8192, llama3-8b-8192, mixtral-8x7b
- **切り替え方法**:
  - 環境変数 `XELYON_PROVIDER`
  - CLIフラグ `--provider <name>`
  - 設定ファイル `default_provider`
- **ストリーミング対応**: 全プロバイダーでリアルタイム出力
- **アイドルタイムアウト**: データ受信がない状態がN秒続くとタイムアウト（長時間出力でもタイムアウトしない）
- **フォールバック実装**: 非ストリーミング時は自動で一括表示

#### 6. 設定管理 (internal/config/)
- **YAML形式**: `~/.xelyon/config.yaml`
- **プロバイダー設定**: default_provider, provider_models
- **デフォルトモデル**: default_model設定で起動時のモデル指定
- **プロバイダーごとの設定**: 各プロバイダーのデフォルトモデルと利用可能モデル一覧
- **自動作成**: 初回起動時にデフォルト設定を自動生成
- **バリデーション**: 無効なプロバイダー/モデル名を拒否
- **プロンプトキャッシュ設定**: `prompt_cache`（enabled / max_entries / ttl_seconds）
- **.env自動読み込み**: `godotenv`でプロジェクトディレクトリの`.env`を自動読み込み
  - 起動時に`main.go`で実行（ファイルが存在しなくてもエラーにならない）
  - プロジェクトごとに異なるAPIキーやプロバイダーを設定可能
  - `.env.example`でサンプルファイルを提供

#### 7. プロンプトキャッシュ基盤 (internal/cache/)
- **目的**: system prompt / Repo Map など生成コストが高い文字列を再利用できるようにするためのキャッシュ基盤。
- **実装**: `internal/cache/cache.go`
  - in-memory の **TTL付き LRU**（最大エントリ数で追い出し）
  - **スレッドセーフ**（Mutexで保護）
  - **Clock抽象化**（テストで時間を制御）
  - 無効化時は no-op（Getは not found）
  - 最小メトリクス（hit/miss）
- **設定**: `prompt_cache`（`internal/config/config.go`）
  - `enabled`: 有効化
  - `max_entries`: 最大エントリ数
  - `ttl_seconds`: デフォルトTTL（秒）
- **注意**: Claude の `cache_control` 付与や OpenAI Prompt Caching API への統合は、別途 provider 実装側での対応が必要（この変更では基盤と設定のみ）。
- **Provider統合状況**:
  - **Claude**: `prompt_cache.enabled` のとき、Messages API の `system` を content blocks に変換し `cache_control` を付与（Prompt Caching 用）
  - **OpenAI**: 現状の `/v1/chat/completions` 実装は維持（API側 Prompt Caching は将来対応）
  - **Gemini**: Context Caching は別API（cached content 管理）が必要なため将来対応



#### 7. バージョン管理 (internal/version/)
- **一元管理**: `version.go`でバージョンを定数管理
- **複数表示**: 起動バナー、`/version`コマンド、`--version`フラグ
- **自動反映**: Version定数を変更するだけで全体に反映

#### 8. 自動検証システム (internal/agent/verify.go)

- **ロールバック確認の改善**:
  - `go test` 失敗時の "Rollback the change?" 確認は `y/n/c`（comment）に対応
  - `c` の場合は、コメントを `internal/agent/confirm.go` の `confirmOrCommentToAI()` 経由で AI に送り、次の提案（修正案）を生成して継続
  - ロールバックはユーザーが `y` で明示承認した場合のみ実行

- **実装**:
  - `internal/agent/agent_verify.go`: rollback提案フロー
  - `internal/agent/confirm.go`: agent側確認（y/n/c）+ コメント時のAI再提案

- **対象**: Goファイル（.go）の変更時
- **go fmt**: コードフォーマット自動実行
- **go test**: 該当パッケージのテスト実行
- **rollback提案**: テスト失敗時に変更を取り消すか確認
- **スキップ可能**: 検証を実行しない選択も可能
- **プロジェクト検出**: go.mod の存在を確認

#### 9. MCP連携 (internal/mcp/)
- **MCPクライアント**: 外部MCPサーバーに接続
- **ツール自動登録**: 接続時にツール一覧を取得、Tool Registryに登録
- **設定ファイル**: `~/.xelyon/mcp.json` でサーバー定義
- **stdioトランスポート**: コマンド実行でサーバー起動
- **動的SystemPrompt**: MCPツールを自動的にAIに提示
- **エラーハンドリング**: 接続失敗しても続行、警告表示のみ

#### 10. Repo Map (internal/repomap/)

- **起動高速化（永続キャッシュ）**:
  - Repo Map の生成結果（`rm.Generate()`）は、プロジェクトパスをキーとして **ディスクに永続キャッシュ**されます。
  - **無効化条件**: プロジェクト配下のファイル更新（fingerprint=max modTime）が検知されるとキャッシュは無効化され、再生成されます。
  - **保存先**: `~/.xelyon/cache/repomap/<sha256(projectPath)>.json`
  - **除外ディレクトリ**: `.git`, `node_modules`, `vendor`, `.xelyon`, `.idea`, `.vscode`
  - **実装**:
    - `internal/cache/repomap_persist.go`（fingerprint計算 + load/save）
    - `internal/agent/repomap_cache.go`（agent起動時の統合）

- **Tree-sitter解析**: AST解析で正確なシンボル抽出
- **複数言語対応**: Go, JavaScript, TypeScript, Python
- **シンボル抽出**: 関数、メソッド、構造体、クラス、インターフェース
- **トークン制限**: 大規模プロジェクトでも効率的にコンテキスト圧縮
- **自動生成**: 起動時にプロジェクトをスキャン
- **除外パターン**: node_modules, .git, vendor等は自動除外

#### 11. コードレビュー (internal/review/)

- **`/review` コマンド**: セッション中の変更をルールベースでレビュー
- **パイプライン**: Scanner → Analyzer → Fixer → Reporter
- **検出ルール**:
  | カテゴリ | ルールID | 説明 | 重大度 |
  |---------|----------|------|--------|
  | General | large-diff | 500行以上の追加 | warning |
  | General | todo-added | TODO/FIXME追加検出 | info |
  | General | go-export-missing-doc | エクスポート関数のドキュメント不足 | info |
  | General | sensitive-file | 機密ファイル変更 | warning |
  | Security | cmd-injection | コマンドインジェクション | error |
  | Security | weak-crypto | 弱い暗号化 | warning |
  | Security | http-no-timeout | HTTPタイムアウト未設定 | warning |
  | Security | path-traversal | パストラバーサル | error |
  | Test | test-coverage | 新規ファイルのテスト不足 | info |
  | Test | assertion-missing | テスト関数のアサーション不足 | warning |
- **フラグ**:
  - `--all, -a`: セッションではなくgit diff全体をレビュー
  - `--security, -s`: セキュリティルールを有効化
  - `--test, -t`: テストルールを有効化
  - `--fix, -f`: 修正提案を生成（suggestion-only、ファイル変更なし）
  - `--yes, -y`: 確認プロンプトをスキップ
- **出力**: Markdownレポート（`~/.xelyon/reviews/YYYYMMDD_HHMMSS.md`）

## コーディングルール


### 会話学習（Phase 3）: 学習したルールの提案

セッション終了時（`/exit`）に、そのセッション中の会話から「今後も守るべきコーディング指示・ルール」をLLMで抽出し、`XELYON.md` に追記する提案を行います。

- 抽出されたルール候補がある場合のみ、`XELYON.md に追加しますか？ (y/n)` を表示
- `y` の場合、`## 学習したルール` セクションへ箇条書きで追記（存在しない場合は新規作成）
- 既に同じ文面が `XELYON.md` 内に存在する場合は重複追加しません
- 抽出に失敗してもCLI終了は継続します（Warning表示のみ）

例: 「エラーは必ずwrapして」という指示が会話中に出た場合、`- エラーは必ずwrapする` のように提案されます。


### エラーハンドリング
- すべてのI/O操作でエラーチェック必須
- ユーザー体験を妨げない範囲でサイレント失敗を許容
- 重要なエラーは`red.Printf()`で明示

### ツール設計原則
1. **冪等性**: 同じ入力で同じ結果
2. **確認プロンプト**: 破壊的操作（write_file, str_replace）は必ず確認
3. **diff表示**: 変更内容を視覚化
4. **バックアップ**: 既存ファイル編集時は.bakを作成

### Undo機能の仕組み
```go
// FileChange構造体
type FileChange struct {
    FilePath    string    // 変更したファイル
    BackupPath  string    // .bakファイルパス
    Timestamp   time.Time // 変更時刻
    Tool        string    // 使用ツール
    Description string    // 変更内容
}

// Agent.changeStack: 最大10件保持（FIFO）
// /undo: スタックからpop → .bakから復元
```

---

## セキュリティ対策（v0.24.0）

### Phase 1: CRITICAL対策（完了）

#### コマンドインジェクション防止
- **対象**: `internal/tools/bash.go`
- **実装**: コマンド連結文字の検出（`;`, `&&`, `||`, `|`, `` ` ``, `$(`, `>`, `>>`, `<`）
- **動作**: safeCommands以外でセパレータ検出時にブロック＋警告表示

#### パストラバーサル防止
- **対象**: 全ファイル操作ツール（read_file, write_file, delete_file, etc.）
- **実装**: `internal/tools/validation.go`の`ValidatePath()`
- **動作**: カレントディレクトリ外へのアクセスを検出・拒否

#### APIキー露出防止
- **Gemini**: URLパラメータ→`x-goog-api-key`ヘッダーに変更
- **エラーサニタイズ**: `internal/api/provider.go`の`sanitizeErrorMessage()`
  - 正規表現でAPIキーパターンを`[REDACTED]`に置換
  - OpenAI (`sk-...`)、Google (`AIza...`)、AWS (`AKIA...`)対応

#### MCP任意コード実行防止
- **対象**: `internal/mcp/client.go`
- **実装**:
  - コマンドホワイトリスト: `npx`, `node`, `python`, `python3`, `uvx`, `docker`
  - パストラバーサル検出: コマンドに`..`や`/`を含む場合拒否
  - 環境変数サニタイズ: `KEY`, `TOKEN`, `SECRET`を含む変数を除外
  - 安全な環境変数のみ継承: `PATH`, `HOME`, `USER`, etc.

#### グレースフルシャットダウン
- **対象**: `internal/agent/agent.go`
- **実装**: SIGINT/SIGTERM シグナルハンドラ
- **動作**:
  - Ctrl+C 1回目: AI応答中断（`context.Cancel`経由）、メッセージ表示
  - Ctrl+C 2回目（3秒以内）: `agent.Cleanup()`呼び出し後アプリ終了
  - MCPサーバー自動クローズ
  - セッション自動保存

### Phase 2: HIGH対策（完了）

#### HTTPクライアント再利用
- **対象**: 全6プロバイダー（DeepSeek, OpenAI, Claude, Groq, Gemini, Ollama）
- **効果**: 20MB+メモリ節約（100リクエストあたり）、コネクションプーリング有効化

#### ファイルパーミッション修正
- **対象**: セッション履歴、設定ファイル、バックアップ
- **変更**: `0644` → `0600`（ユーザーのみ読み書き可能）

#### レート制限ハンドリング
- **対象**: `internal/api/provider.go`の`handleRateLimit()`
- **実装**: HTTP 429検出＋`Retry-After`ヘッダー解析（秒数/HTTP-date両対応）

#### 競合状態修正
- **Spinner**: `internal/ui/spinner.go`
  - `stopChan`のnil check追加
  - ローカル変数コピーで競合回避
- **ToolRegistry**: `internal/tools/registry.go`
  - `sync.RWMutex`追加
  - Register（書き込みロック）、Execute（読み取りロック）

#### APIエンドポイント設定可能化
- **実装**: 環境変数でURLオーバーライド
- **対応変数**: `DEEPSEEK_API_URL`, `OPENAI_API_URL`, `ANTHROPIC_API_URL`, `GROQ_API_URL`
- **用途**: テスト環境、プロキシ経由、企業内API

### Phase 3: MEDIUM対策（完了）

#### エラー検出強化
- **JSONパースエラー**: Groq/Ollamaでストリーミング中のパースエラーを警告表示
- **I/Oエラー**: Gemini/Ollama/Groqで`scanner.Err()`チェック追加
- **セッション保存エラー**: `internal/agent/agent_chat.go`で保存失敗時に警告表示

#### RepoMap文字列連結最適化
- **対象**: `internal/repomap/repomap.go`
- **変更**: `+=` → `strings.Builder`
- **効果**: O(n²) → O(n)改善

### Phase 4: LOW対策（完了）

#### Contextタイムアウト設定
- **対象**: `internal/agent/agent_chat.go`
- **実装**: `context.WithTimeout(context.Background(), 3*time.Minute)`
- **効果**: API呼び出しの無限待機防止、リソースリーク対策

#### MCPバージョン一元化
- **対象**: `internal/mcp/client.go`
- **変更**: `"0.12.0"` → `version.Version`
- **効果**: バージョン更新時の修正漏れ防止

---

### SystemPromptルール（重要）
**v0.9.0で大幅改善**: 性格定義 + 16個の体系化ルール（4フェーズ構成）

#### Core Identity（性格定義）
- 正直で嘘をつかない
- 限界を認める（わからない時は「わかりません」と言う）
- 推測と事実を明確に区別
- プロフェッショナルな開発者マインド
- 日本語に堪能

#### Workflow Rules（16個のルール）

**Phase 1: Planning & Understanding（計画と理解）**
1. 行動前にコンテキスト理解（list_dir, read_file, search_code使用）
2. **v0.9.0追加**: 複雑なタスクは実行前に計画立案
3. **v0.9.0追加**: ツール使用理由を説明
4. **v0.9.0追加**: 重要な変更前にユーザー確認

**Phase 2: Execution（実行）**
5. 適切なツールを使用（search_code, search_file, str_replace, write_file）
6. write_fileは完全なファイル内容を含む
7. str_replace安全ルール:
   - old_str が非空の場合は「文字列置換モード」。start_line/end_line が指定されていても無視される
   - 行レンジ置換は old_str を空にし、start_line と end_line を両方指定した場合のみ有効（1-indexed inclusive）
   - 行レンジ置換では start_line/end_line の片方欠落、数値変換不可、start > end、範囲外は明確なエラー（old_strへのフォールバック無し）
   - old_str が複数回マッチする場合は失敗し、Candidates（最大5件）+ 前後2行スニペット + Next actions + IMPORTANT を提示する
   - 複数マッチ時はコンテキスト含む
   - 長い編集は分割（~10行ずつ）
   - 連続編集時はread_fileで確認
8. JSON形式でツール呼び出し

**Phase 3: Verification（検証）**
9. ツール実行後の分析と次ステップ決定
10. **v0.9.0追加**: コード品質チェック（go fmt, go test）
11. 完了時はツール呼び出しなし

**Phase 4: Error Handling（エラーハンドリング）**
12. **v0.9.0追加**: エラー時は代替アプローチを試行
13. **v0.9.0追加**: キャンセル時は別のアプローチを提案

**General Guidelines**
14. ユーザーの言語で応答（日本語/英語）
15. 簡潔で有用な回答
16. **v0.9.0追加**: 思考プロセスを明示

## 開発ガイド

### 新機能追加時のチェックリスト
- [ ] ツール追加 → `tools.go`のExecute()に分岐追加
- [ ] コマンド追加 → `agent.go`のhandleSpecialCommand()に追加
- [ ] ヘルプ更新 → `printHelp()`を更新
- [ ] README更新 → バージョン履歴に追記

### ツール追加方法

新しいツールを追加する場合の手順:

1. **`internal/tools/tools.go`のExecute関数にcase追加**
```go
case "new_tool":
    result = executeNewTool(tc.Args["arg1"], tc.Args["arg2"])
```

2. **実装関数を追加**
```go
func executeNewTool(arg1, arg2 string) string {
    // バリデーション
    if arg1 == "" {
        return "Error: arg1 is required"
    }

    // 処理実装
    result := // ...

    return result
}
```

3. **`internal/agent/agent.go`のSystemPromptに追記**
```
- new_tool: Description of the tool. Args: {"arg1": "...", "arg2": "..."}
```

4. **`printHelp()`にも追加**

### コマンド追加方法

新しい対話コマンドを追加する場合:

1. **`internal/agent/agent.go`の`handleSpecialCommand`に追加**
```go
case "/newcommand":
    return handleNewCommand(agent, args)
```

2. **対応するハンドラーを実装**
```go
func handleNewCommand(agent *Agent, args []string) bool {
    // コマンド処理
    green.Println("Command executed!")
    return true
}
```

3. **`printHelp()`を更新**

### 会話履歴の構造

**セッションファイル（JSONL）**: `~/.xelyon/history/{session_id}.jsonl`
```jsonl
{"timestamp":"2026-01-06T10:00:00Z","role":"user","content":"hello","model":"deepseek-chat"}
{"timestamp":"2026-01-06T10:00:05Z","role":"assistant","content":"Hi!","model":"deepseek-chat"}
```

**メタデータファイル（JSON）**: `~/.xelyon/history/metadata/{session_id}.json`
```json
{
  "session_id": "1704567890",
  "model": "deepseek-chat",
  "start_time": "2026-01-06T10:00:00Z",
  "last_modified": "2026-01-06T10:00:05Z",
  "message_count": 2,
  "preview": "hello"
}
```

### デバッグ方法

**ログ出力**（開発中の一時的なデバッグ）:
```go
fmt.Printf("DEBUG: variable = %+v\n", variable)
```

**エラーメッセージは詳細に**:
```go
if err != nil {
    return fmt.Errorf("failed to process file %s: %w", path, err)
}
```

**手動テスト**:
```bash
# ビルド
go build -o xelyon

# 対話モードでテスト
./xelyon

# ワンショットでテスト
./xelyon "test query"

# フラグのテスト
./xelyon --resume
./xelyon --coder
./xelyon --think
```

### ビルド＆テスト
```bash
# ビルド
go build -o xelyon

# フォーマット
go fmt ./...

# テスト
go test ./...

# 依存関係の更新
go mod tidy

# クリーンビルド
go clean
go build -o xelyon
```

## トラブルシューティング

### 履歴が保存されない
- `~/.xelyon/history/`ディレクトリの権限確認
- `storage.Save()`のエラーログ確認（サイレント失敗）

### スピナーが止まらない
- API応答の`firstChunk`フラグ確認
- `spinner.Stop()`呼び出し漏れチェック

### Undoできない
- `changeStack`が空 → 編集操作をまだ実行していない
- `.bak`ファイルなし → 新規ファイル作成（バックアップ不要）

### ビルドエラー
- 依存関係の問題 → `go mod tidy`実行
- クリーンビルド → `go clean && go build -o xelyon`

### 会話履歴が読めない
- `~/.xelyon/history/`ディレクトリの権限確認
- JSONLの妥当性チェック: `cat ~/.xelyon/history/*.jsonl | jq .`

### 設定ファイルが読めない
- `~/.xelyon/config.yaml`の権限確認
- YAMLの妥当性チェック: `cat ~/.xelyon/config.yaml`
- 削除して再起動でデフォルト設定が再作成される

## 既知の制約

1. **ストリーミング**: DeepSeek APIはストリーミングレスポンスのみサポート
2. **履歴サイズ**: 大きなセッション（1000+メッセージ）は読み込みが遅くなる可能性
3. **ページング**: AIの応答にはページング適用されない（ツール出力のみ）
4. **並行実行**: 現在は1つのツールを順次実行（並列実行は未対応）
5. **バックアップ**: .bakファイルは1世代のみ保持（複数世代バックアップ未対応）
6. **ループ検知**: 3回同じツール呼び出しで中断（v0.8.0で追加）
7. **APIリトライ**: 最大2回までリトライ（v0.8.0で追加）

## 今後の拡張案

以下の機能拡張は GitHub Issues に移行されました:
- タイムスタンプ付き複数世代バックアップ → [Issue #11](https://github.com/susugadx/xelyon-cli/issues/11)
- `/undo all` コマンド → [Issue #12](https://github.com/susugadx/xelyon-cli/issues/12)
- `/changes` コマンド → [Issue #13](https://github.com/susugadx/xelyon-cli/issues/13)
- 永続的Undo機能 → [Issue #14](https://github.com/susugadx/xelyon-cli/issues/14)
- .gitignoreへの.bak自動追加 → [Issue #15](https://github.com/susugadx/xelyon-cli/issues/15)
- ループ検知のカスタマイズ → [Issue #16](https://github.com/susugadx/xelyon-cli/issues/16)
- APIリトライのカスタマイズ → [Issue #17](https://github.com/susugadx/xelyon-cli/issues/17)
- 差分表示のカスタマイズ → [Issue #18](https://github.com/susugadx/xelyon-cli/issues/18)

## v0.34.0 機能追加

## v0.35.0 機能追加（Issue #39）

### コマンドエイリアス（ショートカット）

**概要**: 対話コマンドにエイリアス（ショートカット）を追加しました。例: `/h` → `/help`, `/m` → `/memory`。

#### 設定（config.yaml）

`~/.xelyon/config.yaml` に `command_aliases` を追加することで、エイリアスの追加・上書きができます。

```yaml
command_aliases:
  /h: /help
  /m: /memory
  /hh: /help
```

#### 仕様
- エイリアスはコマンド解釈の入口で解決されます（`handleSpecialCommand`）。
- `command_aliases` は組み込みエイリアスより優先されます。
- 多段エイリアス（例: `/a`→`/b`→`/help`）に対応。
- 循環参照（例: `/a`→`/b`→`/a`）は無限ループ防止のため途中で停止し、元のコマンドを採用します。

#### 実装ファイル
- `internal/agent/command_aliases.go`
- `internal/agent/agent_commands.go`（入口に解決を追加）
- `internal/agent/command_aliases_test.go`


### タイムスタンプ付きバックアップ & .gitignore自動管理 (Issue #11, #15)

**概要**: バックアップファイルにタイムスタンプを付与し、複数世代を保持する機能を実装。また、初回バックアップ作成時に `.gitignore` へ `*.bak*` パターンを自動追加する機能を実装。

#### 実装ファイル
- `internal/config/config.go`: `BackupConfig` 構造体追加
- `internal/tools/common.go`: `createBackup()`, `cleanupOldBackups()` 関数を修正/追加
- `internal/tools/gitignore.go`: `.gitignore` 管理機能（新規作成）

#### 主要機能

##### 1. タイムスタンプ付きバックアップ

**バックアップ形式**:
```
元ファイル: config.yaml
バックアップ: config.yaml.bak.20260110_153045
```

**実装** (`internal/tools/common.go`):
```go
func createBackup(filePath string) (string, error) {
	// ファイルが存在しない場合はスキップ（新規作成）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil
	}

	// .gitignore自動追加（初回のみ）
	if err := ensureGitignore(filepath.Dir(filePath)); err != nil {
		yellow.Printf("Warning: Failed to update .gitignore: %v\n", err)
	}

	// タイムスタンプ付きバックアップパス生成
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.bak.%s", filePath, timestamp)

	// バックアップ作成
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// 古いバックアップを削除（maxGenerationsを超えたもの）
	if err := cleanupOldBackups(filePath); err != nil {
		yellow.Printf("Warning: Failed to cleanup old backups: %v\n", err)
	}

	return backupPath, nil
}
```

**ポイント**:
- タイムスタンプ形式: `YYYYMMdd_HHmmss`
- ファイル名のソートで古い順に並ぶ
- 既存の `.bak` 形式から完全移行

##### 2. 古いバックアップの自動削除

**実装** (`internal/tools/common.go`):
```go
func cleanupOldBackups(filePath string) error {
	// 設定から最大世代数を取得
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	maxGenerations := cfg.Backup.MaxGenerations
	if maxGenerations <= 0 {
		maxGenerations = 5 // デフォルト
	}

	// 同じファイルのバックアップを検索
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)
	pattern := fmt.Sprintf("%s.bak.*", baseName)

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	// バックアップが最大世代数以下なら削除不要
	if len(matches) <= maxGenerations {
		return nil
	}

	// タイムスタンプでソート（古い順）
	sort.Strings(matches)

	// 古いものから削除
	deleteCount := len(matches) - maxGenerations
	for i := 0; i < deleteCount; i++ {
		if err := os.Remove(matches[i]); err != nil {
			yellow.Printf("Warning: Failed to delete old backup %s: %v\n", matches[i], err)
		}
	}

	return nil
}
```

**ポイント**:
- `filepath.Glob()` で同じファイルのバックアップのみ検索
- `sort.Strings()` でタイムスタンプ順にソート
- 最大世代数を超えた古いファイルを削除
- 個別の削除失敗は続行

##### 3. .gitignore自動管理

**実装** (`internal/tools/gitignore.go`):
```go
func ensureGitignore(dir string) error {
	gitignoreCheckedLock.Lock()
	defer gitignoreCheckedLock.Unlock()

	// すでにこのディレクトリで確認済みの場合はスキップ
	if gitignoreAddedFlag[dir] {
		return nil
	}

	// .gitignore のパスを決定（リポジトリルートまたはカレントディレクトリ）
	gitignorePath := findGitignorePath(dir)
	if gitignorePath == "" {
		gitignoreAddedFlag[dir] = true
		return nil
	}

	// .gitignore にすでに *.bak* パターンが含まれているかチェック
	exists, _ := fileExists(gitignorePath)
	if exists {
		hasPattern, _ := gitignoreHasBackupPattern(gitignorePath)
		if hasPattern {
			gitignoreAddedFlag[dir] = true
			return nil
		}
	}

	// ユーザーに確認
	yellow.Printf("📝 .gitignore にバックアップファイルを追加\n")
	yellow.Print("Add to .gitignore? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		yellow.Println("Skipped")
		gitignoreAddedFlag[dir] = true
		return nil
	}

	// .gitignore に追加
	if err := addBackupPatternsToGitignore(gitignorePath); err != nil {
		return err
	}

	green.Printf("✅ .gitignore に *.bak* パターンを追加しました\n")
	gitignoreAddedFlag[dir] = true
	return nil
}
```

**追加内容** (`.gitignore`):
```
# XELYON CLI backup files
*.bak
*.bak.*
```

**ポイント**:
- **初回のみ確認**: `gitignoreAddedFlag` でディレクトリごとに追加済みフラグを管理
- **リポジトリルート検出**: `.git` ディレクトリを探してルートの `.gitignore` を使用
- **既存パターンチェック**: すでに `*.bak` パターンがあれば追加しない
- **スレッドセーフ**: `sync.Mutex` で並行アクセスを保護

##### 4. 設定ファイル拡張

**Config構造体** (`internal/config/config.go`):
```go
type Config struct {
	DefaultProvider string
	DefaultModel    string
	ProviderModels  map[string]ProviderModelConfig
	Compression     CompressionConfig
	Backup          BackupConfig  // 追加
}

type BackupConfig struct {
	MaxGenerations int `yaml:"max_generations"` // 保持する世代数（デフォルト5）
}
```

**デフォルト設定**:
```go
Backup: BackupConfig{
	MaxGenerations: 5,
}
```

**設定例** (`~/.xelyon/config.yaml`):
```yaml
backup:
  max_generations: 10  # 10世代保持
```

#### 技術的詳細

**バックアップ作成タイミング**:
- `write_file`: 既存ファイルの上書き時
- `str_replace`: ファイル編集時
- `append_file`, `prepend_file`, `insert_after`, `insert_before`: ファイル追加時
- `delete_file`: ファイル削除前
- `move_file`: ファイル移動前
- `delete_lines`: 行削除前

**削除ロジック**:
1. 同じファイルのバックアップを検索 (`*.bak.*` パターン)
2. タイムスタンプでソート（ファイル名の辞書順 = 時系列順）
3. 最大世代数を超えた古いファイルを削除

**Undo機能との互換性**:
- `/undo` コマンドは最新のバックアップ (`FileChange.BackupPath`) を使用
- タイムスタンプ付きバックアップなので、複数世代が保持される
- 手動で古い世代に戻すことも可能

#### 使用例

##### バックアップ作成の流れ
```
# 1回目の編集（初回）
> "config.yaml を編集して"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 .gitignore にバックアップファイルを追加
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
バックアップファイル (*.bak*) を .gitignore に追加しますか？
場所: /path/to/project/.gitignore

Add to .gitignore? (y/n): y
✅ .gitignore に *.bak* パターンを追加しました
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

→ config.yaml.bak.20260110_143000 作成

# 2回目の編集
> "config.yaml を再編集"
→ config.yaml.bak.20260110_143500 作成

# 6回目の編集（最大世代数5を超える）
> "config.yaml を修正"
→ config.yaml.bak.20260110_145000 作成
→ 最も古いバックアップ (config.yaml.bak.20260110_143000) を自動削除
```

#### 活用シーン
- 複数世代のバックアップから過去の状態に戻したい
- Git管理外にバックアップを置いてコミット履歴を汚さない
- 手動で古い世代を確認・復元したい
- バックアップファイルのディスク使用量を制限したい

詳細は [Issue #11](https://github.com/susugadx/xelyon-cli/issues/11), [Issue #15](https://github.com/susugadx/xelyon-cli/issues/15) を参照。

---

## v0.33.0 機能追加

### /changes と /undo all コマンド (Issue #12, #13)

**概要**: セッション中のファイル変更履歴を表示する `/changes` コマンドと、すべての変更を一括で取り消す `/undo all` コマンドを実装。

#### 実装ファイル
- `internal/agent/agent_commands.go`: `handleChangesCommand()`, `handleUndoAll()`, `getChangeTypeJP()` 関数
- `internal/tools/types.go`: `FileChange` 構造体（既存）

#### 主要機能

##### 1. 変更履歴の表示 (`/changes`)

```go
func handleChangesCommand(agent *Agent) bool {
	if len(agent.changeStack) == 0 {
		yellow.Println("変更履歴はありません")
		return true
	}

	for i, change := range agent.changeStack {
		// 変更種別を日本語で表示
		changeType := getChangeTypeJP(change.Tool)

		// Undo可能かチェック（バックアップファイルの存在確認）
		canUndo := "❌"
		if change.BackupPath != "" {
			if _, err := os.Stat(change.BackupPath); err == nil {
				canUndo = "✅"
			}
		}

		// 表示
		fmt.Printf("  [%d] %s %s\n", i+1, changeType, change.FilePath)
		fmt.Printf("      時刻: %s | ツール: %s | Undo: %s\n",
			change.Timestamp.Format("15:04:05"), change.Tool, canUndo)
	}
}
```

**変更種別アイコン**:
- 📝 作成 (write_file)
- ✏️  編集 (str_replace, append_file, prepend_file, insert_after, insert_before)
- 🗑️  削除 (delete_file)
- 📦 移動 (move_file)
- 📋 コピー (copy_file)
- ✂️  行削除 (delete_lines)

**ポイント**:
- バックアップファイルの存在を自動確認
- Undoステータスを視覚的に表示（✅/❌）
- タイムスタンプと説明を含む詳細情報

##### 2. すべての変更を取り消し (`/undo all`)

```go
func handleUndoAll(agent *Agent) bool {
	totalChanges := len(agent.changeStack)

	// 確認プロンプト
	fmt.Printf("取り消す変更数: %d 件\n", totalChanges)
	yellow.Println("\n⚠️  Warning: すべてのファイルがバックアップから復元されます")

	// ユーザー確認
	if !confirmAction() {
		yellow.Println("Cancelled")
		return true
	}

	// 逆順で処理（新しい変更から古い変更へ）
	successCount := 0
	failCount := 0

	for i := len(agent.changeStack) - 1; i >= 0; i-- {
		change := agent.changeStack[i]

		// バックアップから復元
		if err := restoreFromBackup(change); err != nil {
			red.Printf("  ❌ [%d/%d] %s - 復元失敗: %v\n",
				totalChanges-i, totalChanges, change.FilePath, err)
			failCount++
			continue
		}

		green.Printf("  ✅ [%d/%d] %s\n", totalChanges-i, totalChanges, change.FilePath)
		successCount++
	}

	// スタックをクリア
	agent.changeStack = []tools.FileChange{}

	// 結果表示
	green.Printf("✅ 成功: %d 件\n", successCount)
	if failCount > 0 {
		yellow.Printf("⚠️  失敗/スキップ: %d 件\n", failCount)
	}
}
```

**ポイント**:
- **逆順処理**: 新しい変更から古い変更へ復元（依存関係を考慮）
- **エラーハンドリング**: 個別のファイル復元が失敗しても処理を継続
- **確認プロンプト**: 実行前に変更数を表示して確認
- **詳細な進捗表示**: `[1/5]` 形式で進捗状況を表示
- **結果サマリー**: 成功/失敗の件数を最後に表示

##### 3. /undo コマンドの拡張

```go
func handleUndoCommand(agent *Agent, args []string) bool {
	// /undo all の場合
	if len(args) > 0 && args[0] == "all" {
		return handleUndoAll(agent)
	}

	// 既存の単一undo処理
	// ...
}
```

**変更点**:
- 引数 `args []string` を追加
- `args[0] == "all"` の場合は `handleUndoAll()` を呼び出し
- 既存の単一undo機能はそのまま維持

#### 使用例

##### 変更履歴の確認
```
> /changes
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 Change History / 変更履歴 (5 件)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [1] 📝 作成 internal/api/new_feature.go
      時刻: 14:23:15 | ツール: write_file | Undo: ✅

  [2] ✏️  編集 internal/api/client.go
      時刻: 14:25:30 | ツール: str_replace | Undo: ✅
      説明: タイムアウト設定を30秒に変更

  [3] 🗑️  削除 tmp/old_file.go
      時刻: 14:27:10 | ツール: delete_file | Undo: ✅
```

##### すべての変更を取り消し
```
> /undo all
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  Undo All Changes / すべての変更を取り消し
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
取り消す変更数: 4 件

⚠️  Warning: すべてのファイルがバックアップから復元されます

Continue? (y/n): y

Restoring files...
  ✅ [1/4] internal/api/utils.go
  ✅ [2/4] tmp/old_file.go
  ✅ [3/4] internal/api/client.go
  ✅ [4/4] internal/api/new_feature.go

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ 成功: 4 件
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 活用シーン
- AIが意図しない変更を加えた場合のロールバック
- 試行錯誤の結果、変更前の状態に戻したい場合
- セッション終了前に変更内容を確認したい場合
- 複数の変更を一度にロールバックしたい場合

#### 技術的詳細

**FileChange構造体** (internal/tools/types.go):
```go
type FileChange struct {
	FilePath    string
	BackupPath  string    // .bakファイルのパス
	Timestamp   time.Time
	Tool        string    // ツール名（write_file, str_replace, etc.）
	Description string    // 変更の説明
}
```

**changeStack管理**:
- `agent.changeStack []tools.FileChange` に変更履歴を格納
- 最大10件まで保持（`MaxChangeStackSize = 10`）
- 新しい変更は末尾に追加（LIFO: Last In, First Out）

詳細は [Issue #12](https://github.com/susugadx/xelyon-cli/issues/12), [Issue #13](https://github.com/susugadx/xelyon-cli/issues/13) を参照。

---

## v0.32.0 機能追加

### /use と /providers コマンド (Issue #8)

**概要**: 会話中にLLMプロバイダーを切り替える `/use` コマンドと、利用可能なプロバイダー一覧を表示する `/providers` コマンドを実装。

#### 実装ファイル
- `internal/agent/agent.go`: `SwitchProvider()` メソッドと `IsAPIKeyAvailable()` 関数
- `internal/agent/agent_commands.go`: `handleUseCommand()` と `handleProvidersCommand()` 関数

#### 主要機能

##### 1. プロバイダー切り替え (`/use <provider>`)

```go
func (a *Agent) SwitchProvider(providerName string) error {
	// API キー存在チェック
	if !IsAPIKeyAvailable(providerName) {
		return fmt.Errorf("%s のAPIキーが設定されていません", providerName)
	}

	// プロバイダーインスタンス作成
	provider, err := api.NewProvider(providerName)
	if err != nil {
		return fmt.Errorf("プロバイダーの初期化に失敗しました: %w", err)
	}

	// プロバイダー切り替え
	oldProvider := a.ProviderName
	a.CurrentProvider = provider
	a.ProviderName = providerName

	// 統計情報のプロバイダー名も更新
	if a.Stats != nil {
		a.Stats.Provider = providerName
	}

	green.Printf("✅ Provider: %s → %s\n", oldProvider, providerName)
	return nil
}
```

**ポイント**:
- 会話履歴 (`a.History`) はそのまま保持 → プロバイダー切り替え後も会話継続可能
- 統計情報 (`a.Stats`) のプロバイダー名も自動更新
- APIキー未設定の場合はエラーメッセージと設定方法を表示

##### 2. APIキー存在チェック

```go
func IsAPIKeyAvailable(provider string) bool {
	switch provider {
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY") != ""
	case "openai":
		return os.Getenv("OPENAI_API_KEY") != ""
	case "claude":
		return os.Getenv("ANTHROPIC_API_KEY") != ""
	case "gemini":
		return os.Getenv("GEMINI_API_KEY") != ""
	case "groq":
		return os.Getenv("GROQ_API_KEY") != ""
	case "ollama":
		return true // Ollama はローカルなのでキー不要
	default:
		return false
	}
}
```

##### 3. プロバイダー一覧表示 (`/providers`)

```go
func handleProvidersCommand(agent *Agent) bool {
	providers := []string{"deepseek", "claude", "openai", "gemini", "groq", "ollama"}

	for _, provider := range providers {
		isCurrent := agent.ProviderName == provider
		hasAPIKey := IsAPIKeyAvailable(provider)

		icon := "  "
		if isCurrent {
			icon = "✓ "
		}

		status := ""
		if provider == "ollama" {
			status = "(ローカル)"
		} else if hasAPIKey {
			status = "(API key設定済み)"
		} else {
			status = "(API key未設定)"
		}

		// 色付け表示
		if isCurrent {
			green.Printf("%s%-12s %s\n", icon, provider, status)
		} else {
			fmt.Printf("%s%-12s %s\n", icon, provider, status)
		}
	}
}
```

**表示例**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📡 利用可能なプロバイダー / Available Providers
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ deepseek     (現在使用中)
✓ claude       (API key設定済み)
  openai       (API key未設定)
✓ gemini       (API key設定済み)
✓ ollama       (ローカル)
  groq         (API key未設定)

使い方: /use <provider>
例: /use claude
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 活用シーン
- 推論タスクで Claude に切り替え
- コスト節約でローカル Ollama に切り替え
- 異なるLLMの出力を比較
- APIレート制限に達したら別プロバイダーに切り替え

詳細は [Issue #8](https://github.com/susugadx/xelyon-cli/issues/8) を参照。

---

## v0.31.0 機能追加

### /compress コマンド (Issue #6)

**概要**: 会話履歴をLLMでサマリー化してトークン数を削減する `/compress` コマンドを実装。

#### 実装ファイル
- `internal/agent/compress.go`: 圧縮ロジックとサマリー生成
- `internal/agent/agent_commands.go`: `handleCompressCommand()` 関数
- `internal/config/config.go`: `CompressionConfig` 構造体追加

#### 圧縮ロジック

1. **保持対象の選択**: 最新N件（デフォルト10件）を保持、それ以前を圧縮
2. **サマリー生成**: LLMに圧縮対象を送信、箇条書きで5-10項目にサマリー化
3. **履歴再構築**: サマリーを先頭に挿入、元メッセージを削除

#### トークン数計算

```go
func estimateTokens(messages []api.Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	// 簡易計算: 平均3文字/token
	return totalChars / 3
}
```

#### 設定 (config.yaml)

```yaml
compression:
  auto_compress: false      # 自動圧縮（将来実装）
  threshold_tokens: 40000   # 自動圧縮閾値
  keep_recent: 10           # 保持メッセージ数
```

詳細は [Issue #6](https://github.com/susugadx/xelyon-cli/issues/6) を参照。

---

## v0.30.0 機能追加

### /copy コマンド (Issue #5)

**概要**: 最後のAI出力をクリップボードにコピーする `/copy` コマンドを実装。

#### 実装ファイル
- `internal/agent/agent.go`: Agent構造体に `lastOutputs []string` フィールド追加
- `internal/agent/agent_chat.go`: `handleNormalResponse()` で出力を記録
- `internal/agent/agent_commands.go`: `handleCopyCommand()` と `extractCodeBlocks()` 関数

#### 依存パッケージ
```bash
go get github.com/atotto/clipboard
```

- クロスプラットフォーム対応（Windows/macOS/Linux）
- Linux では `xclip` または `xsel` が必要

#### 使用例

```bash
# 基本的な使い方
> /copy              # 最後の出力全体
> /copy code         # コードブロックのみ
> /copy -n 2         # 2つ前の出力
> /copy code -n 3    # 3つ前のコードブロック
```

#### 機能詳細

1. **出力履歴管理**
   - 最後の10件のAI出力を `lastOutputs` に保持
   - `handleNormalResponse()` で自動記録
   - リングバッファ形式（古い出力から削除）

2. **コードブロック抽出**
   - 正規表現: `(?s)` + "```\\w*\\n(.*?)```"
   - Markdown コードブロック（\`\`\`language\n...\`\`\`）を検出
   - 複数ブロックは空行2つで区切って連結

3. **引数解析**
   - `code`: コードブロックのみ抽出
   - `-n N`: N番目前の出力を指定（1-indexed）
   - 組み合わせ可能: `/copy code -n 2`

4. **エラーハンドリング**
   - Linux環境で xclip/xsel 未インストールの場合、インストール方法を案内
   - 出力履歴が空の場合: "No AI output to copy yet"
   - コードブロックなしの場合: "No code blocks found in output"

#### コードブロック抽出ロジック

```go
func extractCodeBlocks(text string) []string {
	// 正規表現: ```language\n...```
	re := regexp.MustCompile("(?s)```\\w*\\n(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}

	return blocks
}
```

#### Linux環境のセットアップ

```bash
# Ubuntu/Debian
sudo apt-get install xclip

# Fedora/RHEL
sudo dnf install xclip

# Arch Linux
sudo pacman -S xclip
```

詳細は [Issue #5](https://github.com/susugadx/xelyon-cli/issues/5) を参照。

---

## v0.29.0 機能追加

### /stats コマンド (Issue #4)

**概要**: セッション統計情報を表示する `/stats` コマンドを実装。

#### 実装ファイル
- `internal/agent/stats.go`: SessionStats構造体とコスト計算ロジック
- `internal/agent/agent.go`: Agent構造体にStatsフィールド追加
- `internal/agent/agent_chat.go`: メッセージ・ツール実行のカウント処理
- `internal/agent/agent_commands.go`: handleStatsCommand関数

#### 表示内容
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Session Statistics / セッション統計
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⏱️  Time / 経過時間
  Elapsed: 15m 30s

💬 Messages / メッセージ数
  User:      10
  Assistant: 12
  Total:     22

🔧 Tool Executions / ツール実行回数
  Total: 25
  Breakdown:
    - read_file    : 8
    - write_file   : 5
    - git_commit   : 2

🤖 Provider / プロバイダー
  Name: DeepSeek
  Model: deepseek-coder

💰 Token Usage & Cost / トークン使用量とコスト
  Input:  12,500 tokens
  Output: 3,200 tokens
  Total:  15,700 tokens
  Estimated Cost: $0.0032 USD

📁 Session File / セッションファイル
  Path: ~/.xelyon/sessions/abc123.json
  Size: 45.2 KB
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### SessionStats構造体
```go
type SessionStats struct {
	StartTime         time.Time
	UserMessages      int
	AssistantMessages int
	ToolExecutions    map[string]int // ツール名 -> 実行回数
	InputTokens       int
	OutputTokens      int
	Provider          string
}
```

#### コスト計算（$/1M tokens）
```go
switch s.Provider {
case "deepseek":
	inputCost = 0.14
	outputCost = 0.28
case "openai":
	inputCost = 2.50
	outputCost = 10.00
case "claude":
	inputCost = 3.00
	outputCost = 15.00
case "gemini":
	inputCost = 0.075
	outputCost = 0.30
case "groq":
	inputCost = 0.10
	outputCost = 0.10
case "ollama":
	return 0.0 // ローカル実行
}
```

#### 統計情報の記録タイミング
1. **セッション開始時**: `NewAgent()` で `SessionStats` 初期化、StartTimeを記録
2. **Userメッセージ送信時**: `chat()` 関数で `Stats.UserMessages++`
3. **Assistantレスポンス時**: `executeToolCall()` / `handleNormalResponse()` で `Stats.AssistantMessages++`
4. **ツール実行時**: `executeToolCall()` で `Stats.AddToolExecution(toolName)`
5. **トークン使用時**: 将来の実装（APIレスポンスから取得）

#### 今後の拡張
- トークン数の自動取得（API usage フィールド対応）
- セッションごとのコスト累計グラフ
- プロバイダー間のコスト比較機能

詳細は [Issue #4](https://github.com/susugadx/xelyon-cli/issues/4) を参照。

---

## v0.28.0 機能拡張

### --auto-approve フラグ (Issue #3)

**概要**: 安全・中レベルの操作を自動承認する `-y` / `--auto-approve` フラグを実装。

#### 実装ファイル
- `internal/tools/safety.go`: ツール危険度分類（SafetyHigh/Medium/Low）
- `internal/tools/common.go`: `confirmWithAutoApprove()` 関数
- `cmd/root.go`: `-y` / `--auto-approve` フラグ
- `internal/agent/agent.go`: AutoApprove フィールド
- `internal/agent/agent.go`: DryRunMode フィールド
- `internal/agent/agent_chat.go`: executeToolCall() で Dry Run 分岐（tools.Execute() を呼ばずに結果をシミュレート）


#### 危険度分類
```go
// SafetyHigh: 読み取り専用（常に自動承認OK）
read_file, list_dir, git_status, git_log, lint, test, web_search

// SafetyMedium: 書き込み（--auto-approve で承認）
write_file, str_replace, append_file, prepend_file, insert_after, insert_before,
copy_file, create_dir, git_add, git_commit

// SafetyLow: 破壊的操作（常に確認必須）
delete_file, delete_lines, move_file, bash,
git_push, git_checkout, git_branch, git_stash
```

#### 動作
1. `--auto-approve` なし: 全て確認プロンプト表示
2. `--auto-approve` あり:
   - SafetyHigh / SafetyMedium → 自動承認（"✓ Auto-approved" 表示）
   - SafetyLow → 確認プロンプト表示（安全性重視）

#### テスト
- `internal/tools/safety_test.go`: 危険度分類・自動承認ロジックのテスト

詳細は [Issue #3](https://github.com/susugadx/xelyon-cli/issues/3) を参照。

---

### 対話的修正機能 Phase 1 (Issue #1)

**概要**: ツール実行前の確認プロンプトを拡張しました。

### 実装詳細

#### 1. 拡張確認プロンプト (y/n/c)
**ファイル**: `internal/tools/confirm_interactive.go`

従来の `y/n` 確認を `y/n/c` に拡張:
- `y` (yes) - ツールを実行
- `n` (no) - キャンセル
- `c` (comment) - コメント（修正指示）

**内部実装（ConfirmDecision API）**
- `internal/tools/confirm_interactive.go` の `Confirm()` が `ConfirmDecision`（`ConfirmYes`/`ConfirmNo`/`ConfirmComment`）を返します
- ツール側で `ConfirmComment` が選ばれた場合は、ツールは処理を実行せずに `[COMMENT] ...` 形式のメッセージを返し、エージェントがコメントをコンテキストにして再提案します
- 適用例: `copy_file`（上書き）, `git_add`, `git_branch`（未コミット変更ありの切り替え）, `git_commit`, `lint`（auto-fix）, `move_file`（上書き）

#### 2. 複数行コメント入力
`c` を選択すると複数行コメント入力モードに切り替わります:
```
💬 Enter your comment (press Enter twice to finish):
> この実装だとエラーハンドリングがない
> もっと堅牢にして
>
```

空行2回で入力終了。

#### 3. 修正ループアーキテクチャ
**ファイル**: `internal/agent/interactive_confirm.go`

```go
// executeToolCallInteractive はツール実行を対話的確認付きで実行
// 最大3回まで修正可能（無限ループ防止）
func (a *Agent) executeToolCallInteractive(response string, toolCall *tools.ToolCall)
```

**フロー**:
1. ツール実行前に確認プロンプト表示
2. ユーザーが `c` でコメント入力
3. コメントをAIに送信
4. AIが修正案を返す
5. 再度確認プロンプト（最大3回まで）

#### 4. データ構造
```go
// ConfirmResult は確認結果
type ConfirmResult struct {
	Action  string // "yes", "no", "comment"
	Comment string // "comment" の場合のコメント内容
}
```

### 制限事項（Phase 1）
- Phase 1では基本機能のみ実装
- 画像入力対応は Phase 2 で実装予定
- マルチモーダルAPI対応が必要（Claude 3, GPT-4o等）

### 今後の拡張（Phase 2, 3）
- Phase 2: 画像入力対応（`c image:/path/to/screenshot.png`）
- Phase 3: コメント履歴表示、修正前後の差分表示

詳細は [Issue #1](https://github.com/susugadx/xelyon-cli/issues/1) を参照。

---

## v0.10.0 アーキテクチャ改善

### Provider Pattern (internal/api/)
**目的**: 複数のLLMプロバイダー（DeepSeek, Claude, OpenAI等）を統一インターフェースで扱う

**実装**:
- `Provider` interface: 各プロバイダーの共通インターフェース
- `Client` struct: タイムアウト管理とcontext対応
- `DeepSeekProvider`: DeepSeek APIの実装
- 環境変数 `XELYON_PROVIDER` で切り替え可能（将来の拡張用）

**後方互換性**: 既存の`ChatWithTools`関数は内部でProviderを使用

### Tool Registry Pattern (internal/tools/)
**目的**: ツールを動的に登録・実行し、外部ツール（MCP等）の統合を容易にする

**実装**:
- `Tool` interface: 各ツールの共通インターフェース
- `Registry` struct: ツールの登録・管理
- `DefaultRegistry`: デフォルトのレジストリ（16個の組み込みツール登録済み）
- `builtin.go`: 既存ツールのWrapper実装

**後方互換性**: `Execute`関数は内部でDefaultRegistryを使用

**拡張性**:
- 新しいツールは`Tool` interfaceを実装してRegistryに登録
- MCP（Model Context Protocol）対応の準備完了

## v0.17.0 Phase 1 ツール詳細リファレンス

### append_file
**目的**: ファイル末尾に追加

**引数**:
- `path`: ファイルパス
- `content`: 追加する内容

**特徴**:
- 非破壊的操作（確認プロンプトなし）
- 既存ファイルのみバックアップ作成
- プレビュー表示（既存ファイルの最終10行 + 追加コンテンツ）
- 新規ファイルの場合は作成

**使用例**:
```
ユーザー: "todo.txtにTODO項目を追加して"
AI: append_fileツールを使用します
```

---

### prepend_file
**目的**: ファイル先頭に追加

**引数**:
- `path`: ファイルパス
- `content`: 追加する内容

**特徴**:
- 非破壊的操作（確認プロンプトなし）
- 既存ファイルのみバックアップ作成
- プレビュー表示（追加コンテンツ + 既存ファイルの最初10行）
- 新規ファイルの場合は作成

**使用例**:
```
ユーザー: "main.goにheaderコメントを追加して"
AI: prepend_fileツールを使用します
```

---

### create_dir
**目的**: ディレクトリ作成（親ディレクトリも含む）

**引数**:
- `path`: ディレクトリパス

**特徴**:
- 冪等的操作（すでに存在する場合は成功メッセージ）
- 確認プロンプトなし
- バックアップなし
- `os.MkdirAll()`で親ディレクトリも自動作成
- パーミッション: 0755

**使用例**:
```
ユーザー: "src/utils/helpersディレクトリを作成して"
AI: create_dirツールを使用します
```

---

### run_test
**目的**: テストフレームワークを自動検出して実行

**引数**:
- `path`: テスト対象ディレクトリ（オプション、デフォルト: "."）

**特徴**:
- フレームワーク自動検出:
  1. `go.mod` → `go test ./...`
  2. `package.json` → `npm test`（yarn検出時は`yarn test`）
  3. `pytest.ini` → `pytest`
  4. `Cargo.toml` → `cargo test`
- テスト結果を完全表示（切り詰めなし）
- 終了コードを含む
- 読み取り専用操作（バックアップなし、確認プロンプトなし）
- 検出失敗時は該当するテストコマンドをbashで実行することを提案

**使用例**:
```
ユーザー: "テストを実行して"
AI: run_testツールでテストフレームワークを自動検出して実行します

ユーザー: "internal/toolsのテストを実行して"
AI: run_testツールでinternal/toolsディレクトリのテストを実行します
```

---

### format
**目的**: フォーマッターを自動検出して実行

**引数**:
- `path`: フォーマット対象ディレクトリ（オプション、デフォルト: "."）

**特徴**:
- フォーマッター自動検出:
  1. `*.go` → `go fmt`
  2. `*.js/*.ts` + `.prettierrc*` → `prettier --write`
  3. `*.py` → `black`または`autopep8`
  4. `*.rs` → `rustfmt`
- 複数ファイルの一括フォーマット
- フォーマット済みファイル一覧を表示
- 確認プロンプトなし（フォーマッターは安全かつ可逆的）
- バックアップはGitに依存（フォーマット前にコミット推奨）

**使用例**:
```
ユーザー: "すべてのGoファイルをフォーマットして"
AI: formatツールでgofmtを実行します

ユーザー: "src/ディレクトリをフォーマットして"
AI: formatツールでsrc/ディレクトリをフォーマットします
```

---

### Phase 1 ツールの設計原則

1. **低リスク**: 破壊的操作なし（確認プロンプト不要）
2. **高価値**: 頻繁に使用される操作
3. **明確なユースケース**: 用途が明確で理解しやすい
4. **自動検出**: run_test/formatはフレームワークを自動検出
5. **プレビュー**: append_file/prepend_fileは変更箇所をプレビュー表示
6. **冪等性**: create_dirはすでに存在しても成功

---

## v0.18.0 Phase 3 ツール詳細リファレンス

### insert_after
**目的**: パターンマッチした行の後に内容を挿入

**引数**:
- `path`: ファイルパス（必須）
- `pattern`: マッチさせる行の内容（必須）
- `content`: 挿入する内容（必須）

**特徴**:
- **2段階パターンマッチング**:
  1. Tier 1: 厳密な文字列マッチ（行全体が完全一致）
  2. Tier 2: 正規化ホワイトスペースマッチ（タブ→スペース変換、先頭空白除去後に比較）
- **コンテキスト表示**: マッチした行の前後5行を表示
- **複数マッチエラー**: パターンが複数行にマッチした場合、全マッチ場所を表示してエラー
- **パターン未検出エラー**: パターンが見つからない場合、ファイルの最初50行をプレビュー表示
- **バックアップ作成**: 挿入前に自動的に.bakファイル作成
- **Undo対応**: FileChange追跡により/undoコマンドで復元可能

**使用例**:
```go
// 例: import文の後にコメントを追加
insert_after {
  "path": "main.go",
  "pattern": "import (",
  "content": "\t// Additional imports"
}
```

**リスク評価**: 低（非破壊的、append類似、バックアップあり）

---

### insert_before
**目的**: パターンマッチした行の前に内容を挿入

**引数**:
- `path`: ファイルパス（必須）
- `pattern`: マッチさせる行の内容（必須）
- `content`: 挿入する内容（必須）

**特徴**:
- **insert_afterと同じ機能**: 挿入位置のみ異なる（before vs after）
- **2段階パターンマッチング**: Tier 1 (厳密) → Tier 2 (正規化)
- **コンテキスト表示**: マッチ行の前後5行表示
- **複数マッチエラー**: 全マッチ場所を前後2行付きで表示
- **バックアップ作成**: 挿入前に自動作成
- **Undo対応**: FileChange追跡

**使用例**:
```go
// 例: 関数定義の前にコメントを追加
insert_before {
  "path": "handlers.go",
  "pattern": "func HandleRequest(w http.ResponseWriter, r *http.Request) {",
  "content": "// HandleRequest processes incoming HTTP requests"
}
```

**リスク評価**: 低（非破壊的、prepend類似、バックアップあり）

---

### copy_file
**目的**: ファイルをコピー（パーミッション保持）

**引数**:
- `src`: コピー元ファイルパス（必須）
- `dest`: コピー先ファイルパス（必須）

**特徴**:
- **効率的コピー**: `io.Copy()`を使用してメモリ効率的にコピー
- **パーミッション保持**: `os.Chmod()`でソースファイルのパーミッションを保持
- **条件付き確認**:
  - コピー先が存在しない → 確認なし、即座にコピー
  - コピー先が存在する → 確認プロンプト表示、バックアップ作成
- **ディレクトリ除外**: ソースがディレクトリの場合はエラー（ファイルのみ対応）
- **Undo対応**: 上書き時のみFileChange追跡（バックアップから復元可能）

**使用例**:
```go
// 例: 設定ファイルのバックアップ
copy_file {
  "src": "config.yaml",
  "dest": "config.yaml.backup"
}

// 例: テンプレートファイルのコピー
copy_file {
  "src": "template.go",
  "dest": "handlers/new_handler.go"
}
```

**確認UI（上書き時）**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Copy File / ファイルコピー
📂 Source / コピー元: config.yaml
📂 Destination / コピー先: config.yaml.backup
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  Warning: Destination file already exists / 警告: コピー先ファイルが既に存在します
Overwrite? / 上書きしますか？ (y/n):
```

**リスク評価**: 低（上書き時のみ確認、バックアップあり）

---

### Phase 3 実装の特徴

1. **パターンマッチングの信頼性**:
   - 2段階マッチング戦略（厳密 → 正規化）により、インデントが多少異なっても検出可能
   - `normalizeLeadingWhitespace()`を再利用（str_replaceと同じアルゴリズム）

2. **ユーザーフレンドリーなエラー処理**:
   - パターン未検出: ファイル内容をプレビュー表示（最初50行）
   - 複数マッチ: すべてのマッチ場所を前後コンテキスト付きで表示
   - ユーザーがパターンを修正しやすい情報を提供

3. **確認プロンプト戦略**:
   - insert_after/insert_before: 確認なし（append/prepend類似の非破壊的操作）
   - copy_file: 上書き時のみ確認（新規作成は即座に実行）

4. **コード再利用**:
   - `normalizeLeadingWhitespace()`: 既存関数を再利用
   - `createBackup()`: 既存関数を再利用
   - パターンマッチロジック: executeInsertAfter/Beforeで共通（DRY原則）

5. **Undo完全対応**:
   - すべてのツールがFileChange構造体を返す
   - バックアップパスを追跡
   - `/undo`コマンドで完全復元可能

---

## v0.17.0 Phase 2 ツール詳細リファレンス

### git_branch
**目的**: ブランチ管理（一覧・作成・切り替え）

**引数**:
- `action`: "list" | "create" | "switch" (デフォルト: "list")
- `branch_name`: create/switchで必須

**特徴**:
- **list**: すべてのブランチを表示（`git branch -a`でローカル+リモート）
- **create**: 新しいブランチを作成（確認なし、非破壊的）
- **switch**: ブランチ切り替え
  - 未コミット変更がある場合: 確認プロンプト表示、変更内容をプレビュー
  - 未コミット変更がない場合: 即座に切り替え
- エラーハンドリング: Git本体のエラーをそのまま返す

**使用例**:
```
ユーザー: "List all branches"
AI: git_branch { action: "list" }

ユーザー: "Create a new branch called feature/new-tool"
AI: git_branch { action: "create", branch_name: "feature/new-tool" }

ユーザー: "Switch to main branch"
AI: git_branch { action: "switch", branch_name: "main" }
[未コミット変更がある場合は確認UI表示]
```

---

### git_checkout
**目的**: ファイル復元（HEADから）またはブランチチェックアウト

**引数**:
- `target`: ファイルパス or ブランチ名（必須）

**特徴**:
- **ターゲット判定**: ファイルの存在確認、パスに`/`や`.`が含まれるかで判定
- **ファイル復元**（破壊的操作）:
  - HEADから現在の内容を取得してdiff表示
  - バックアップ自動作成（`.bak`）
  - 赤色警告メッセージ表示
  - 確認プロンプト必須
  - FileChange追跡（Undo可能）
- **ブランチチェックアウト**:
  - 未コミット変更がある場合は確認プロンプト
  - `git_branch`の`switch`と同等の動作
  - Tip: `git_branch`の使用を推奨メッセージ表示

**使用例**:
```
ユーザー: "Discard my changes to main.go"
AI: git_checkout { target: "main.go" }
[確認UI + Diff表示]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  Git Checkout File / ファイル復元
📂 File / ファイル: main.go
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  DESTRUCTIVE: This will discard all local changes!
Restore from HEAD? (y/n): y
📦 Backup created: main.go.bak
✅ Restored from HEAD: main.go
```

---

### git_stash
**目的**: 未コミット変更を一時退避・復元・削除

**引数**:
- `action`: "save" | "list" | "pop" | "apply" | "drop" (デフォルト: "save")
- `message`: saveで任意メッセージ、pop/apply/dropでスタッシュインデックス（"0", "1"等）

**特徴**:
- **save**: 変更をスタッシュ
  - 変更がない場合は警告を表示
  - メッセージ任意（`git stash push -m "message"`）
  - 確認なし（非破壊的、復元可能）
- **list**: スタッシュ一覧表示（`git stash list`）
- **pop**: スタッシュ適用して削除
  - スタッシュプレビュー表示（最初20行）
  - マージコンフリクト警告
  - 確認プロンプト必須
- **apply**: スタッシュ適用（削除しない）
  - popと同様だが、スタッシュは保持される
  - 「スタッシュは保持されます」メッセージ表示
- **drop**: スタッシュ削除
  - 破壊的操作、赤色警告
  - 確認プロンプト必須

**使用例**:
```
ユーザー: "Stash my changes"
AI: git_stash { action: "save", message: "WIP: testing" }
📦 Stashing changes
Changes to stash:
 M main.go
✅ Stashed changes

ユーザー: "List stashes"
AI: git_stash { action: "list" }

ユーザー: "Apply the most recent stash"
AI: git_stash { action: "apply", message: "0" }
[確認UI + プレビュー表示]
```

---

### Phase 2 ツールの設計原則

1. **安全性優先**: 破壊的操作には必ず確認プロンプト
2. **プレビュー表示**: 変更内容やdiffを事前に表示
3. **バックアップ**: ファイル復元時は自動的に`.bak`作成
4. **二言語対応**: すべてのUIで英語/日本語併記
5. **Git管理**: Gitコマンドのエラーはそのまま返す（ユーザーが理解しやすい）
6. **FileChange追跡**: git_checkoutのファイル復元はUndo可能

---

## v0.19.0 Phase 4 ツール詳細リファレンス

### delete_lines
**目的**: ファイルから指定した行範囲を削除

**引数**:
- `path`: ファイルパス（必須）
- `start_line`: 開始行番号（文字列、1-indexed、必須）
- `end_line`: 終了行番号（文字列、1-indexed、必須）

**動作**:
1. 引数をstrconv.Atoiで数値に変換
2. 範囲検証（startLine >= 1, endLine >= startLine）
3. ファイル読み込み＆行分割
4. endLineがファイル長を超える場合は自動的にクランプ（エラーにしない）
5. 削除される行数を計算
6. **確認UI表示**:
   - RED警告表示（破壊的操作を明示）
   - 削除される行範囲とコンテキスト（前後5行）を表示
   - 削除される行は赤色で強調表示
   - 必須確認プロンプト
7. バックアップ作成（.bak）
8. 新しい内容を構築（削除行を除外）
9. ファイル書き込み

**特徴**:
- **Graceful clamping**: endLineが範囲外の場合はファイル末尾までクランプ（ユーザーフレンドリー）
- **RED警告**: 破壊的操作を強調
- **コンテキスト表示**: 削除前に前後5行を表示して誤操作を防止
- **Undo対応**: FileChange追跡で復元可能

**エッジケース**:
- endLine > ファイル行数 → ファイル末尾までクランプ
- startLine == endLine → 単一行削除（正常動作）
- 1-N（全行） → 空ファイル作成（許可）
- 空ファイル → エラー

**実装**: `tools.go:executeDeleteLines()` (約100行)

---

### delete_file
**目的**: ファイルを完全削除（Undoでファイル復元可能）

**引数**:
- `path`: ファイルパス（必須）

**動作**:
1. ファイル存在確認＆ディレクトリでないことを確認
2. ファイル内容読み込み
3. **確認UI表示**:
   - ファイル情報（サイズ、行数）
   - RED警告表示（破壊的操作を明示）
   - ファイルプレビュー（最初20行）
   - 必須確認プロンプト
4. **バックアップ作成（削除前に必須）**:
   - バックアップ失敗時は削除を中止（安全第一）
5. ファイル削除（os.Remove）

**特徴**:
- **バックアップ失敗時は削除中止**: 復元不可能な状態を防ぐ
- **ファイルプレビュー**: 削除前に内容を確認（最初20行）
- **Undo対応**: バックアップからファイルを復元可能

**Undo戦略**:
- FileChange構造体で削除されたファイルの元のパスとバックアップパスを追跡
- Undo時はバックアップファイルを元のパスにコピーしてファイル再作成

**エッジケース**:
- 空ファイル → バックアップ作成（空の.bak）、削除可能
- 読み取り権限なし → バックアップ失敗、削除中止
- シンボリックリンク → リンク自体を削除（対象ファイルは残る）
- ディレクトリ → エラー（ファイルのみ対応）

**実装**: `tools.go:executeDeleteFile()` (約70行)

---

### move_file
**目的**: ファイルを移動・リネーム（アトミック操作）

**引数**:
- `src`: 移動元ファイルパス（必須）
- `dest`: 移動先ファイルパス（必須）

**動作**:
1. ソースファイル確認（存在、ファイルであること）
2. 同一ファイルチェック（no-op）
3. 移動先の親ディレクトリ存在確認
4. **移動先の衝突処理**:
   - 移動先が既に存在する場合のみ確認プロンプト表示
   - 移動先のバックアップ作成（移動元ではない！）
5. **アトミック移動**:
   - os.Rename()でアトミックに移動
   - クロスファイルシステムエラーの場合はコピー＋削除にフォールバック
6. コピー＋削除フォールバック時:
   - パーミッション保持
   - ソース削除失敗時は警告＋手動クリーンアップ指示

**特徴**:
- **アトミック操作**: os.Rename()でファイルシステムレベルのアトミック保証
- **クロスファイルシステム対応**: 自動的にフォールバック
- **移動先のみバックアップ**: 移動元はアトミックに移動されるため不要

**FileChange戦略**:
- 移動先が上書きされた場合のみFileChange作成
- FilePath: 移動先（dest）
- BackupPath: 移動先のバックアップ
- Undo時は移動先をバックアップで上書き（移動元は復元しない）

**エッジケース**:
- 同一ファイル → no-op、即座にreturn
- 親ディレクトリなし → エラー（自動作成しない）
- クロスファイルシステム → コピー＋削除にフォールバック
- コピー成功・削除失敗 → 警告、手動クリーンアップ要求

**実装**: `tools.go:executeMoveFile()` (約110行)

---

### lint
**目的**: リンター実行＆自動修正（プロジェクト自動検出）

**引数**:
- `path`: 対象パス（ファイルまたはディレクトリ、オプション、デフォルト: "."）
- `auto_fix`: 自動修正フラグ（"true"/"false"、オプション、デフォルト: "false"）

**動作**:
1. **リンター検出**（detectLinter ヘルパー関数）:
   - **Go**: go.mod存在 → golangci-lint（優先） or go vet
   - **JavaScript/TypeScript**: package.json + .eslintrc* 存在 → eslint
   - **Python**: *.py存在 → ruff（優先） or pylint
   - **Rust**: Cargo.toml存在 → clippy
2. **Phase 1: チェック実行**（自動修正なし）:
   - リンターのチェックコマンドを実行
   - 出力を表示
   - 問題が検出されたか判定
3. **Phase 2: 自動修正**（オプション）:
   - auto_fix == "true" かつ fixCmd が存在する場合のみ実行
   - 確認プロンプト表示（警告：ファイルが変更される）
   - バックアップ作成（単一ファイルの場合のみ）
   - 自動修正コマンド実行
   - 結果表示

**特徴**:
- **自動検出**: プロジェクトタイプを自動判定
- **2段階実行**: チェック → 自動修正（オプション）
- **確認プロンプト必須**: 自動修正は必ず確認
- **バックアップ制限あり**: 単一ファイルのみバックアップ（ディレクトリ全体は制限として明示）

**制限事項**:
- **ディレクトリ全体の自動修正**: バックアップなし（複数ファイル追跡は200行以上の実装が必要なため）
- 軽減策: 確認UIで制限を明示、Gitコミット後の使用を推奨

**サポートされるリンター**:
| 言語 | リンター | チェックコマンド | 修正コマンド |
|------|---------|------------------|--------------|
| Go | golangci-lint | golangci-lint run | golangci-lint run --fix |
| Go | go vet | go vet ./... | （なし） |
| JavaScript/TypeScript | eslint | eslint . | eslint . --fix |
| Python | ruff | ruff check . | ruff check . --fix |
| Python | pylint | pylint . | （なし） |
| Rust | clippy | cargo clippy | cargo clippy --fix --allow-dirty |

**エッジケース**:
- リンター未インストール → エラー、サポートされているリンター一覧表示
- 自動修正非対応（go vet, pylint） → チェックのみ実行、警告表示
- 問題なし → "No issues found"、自動修正スキップ
- 自動修正失敗 → エラー出力表示、backupPathを返す

**実装**:
- `tools.go:detectLinter()` (約60行)
- `tools.go:executeLint()` (約100行)
- ヘルパー関数: fileExists, commandExists, hasGlobMatches (約20行)

---

### Phase 4 実装の特徴

1. **破壊的操作の安全性**:
   - すべての破壊的操作にRED警告＋必須確認プロンプト
   - バックアップ失敗時は操作を中止（delete_file）
   - コンテキスト表示で誤操作を防止（delete_lines）
   - ファイルプレビュー表示（delete_file）

2. **Undo対応**:
   - すべてのツールがFileChange追跡によりUndo可能
   - delete_file: バックアップからファイル復元
   - move_file: 移動先が上書きされた場合のみUndo可能
   - delete_lines: バックアップからファイル復元

3. **アトミック操作**:
   - move_file: os.Rename()でファイルシステムレベルのアトミック保証
   - クロスファイルシステム時は自動的にフォールバック

4. **リンター統合**:
   - 4言語（Go/JS/Python/Rust）のリンターを自動検出
   - 2段階実行（チェック → 自動修正）
   - プロジェクトタイプを自動判定

5. **二言語対応**:
   - すべてのUI表示で英語/日本語併記
   - 確認プロンプト、警告、エラーメッセージすべて対応

6. **設計原則の継承**:
   - Phase 1-3の設計原則を継承
   - バックアップ作成（.bak）
   - 確認プロンプト
   - FileChange追跡
   - エラーハンドリング

---

## v0.8.0 で実装済み
- [x] ループ検知機能（3回で中断）
- [x] APIリトライ機能（最大2回）
- [x] 長い差分表示の省略（10行超え）
- [x] キャンセル時のメッセージ改善
- [x] 複数マッチ時のファイルプレビュー
- [x] バージョン管理の一元化
- [x] `/model`コマンドで再起動なしに切り替え
- [x] `/version`コマンド
- [x] `--version`フラグ

---

## セキュリティ機能（v0.24.0-v0.25.0で実装）

### 1. 監査ログ機能（v0.25.0）

**目的**: ツール実行履歴を記録し、セキュリティ監査やトラブルシューティングを支援

**有効化方法**:
```bash
export XELYON_AUDIT_LOG=1
./xelyon
```

**ログフォーマット（JSONL）**:
```json
{"timestamp":"2026-01-09T15:04:05Z","tool":"read_file","args":{"path":"main.go"},"output":"package main...(truncated)","success":true,"file_changed":false}
```

**保存場所**: `~/.xelyon/audit/audit_YYYYMMDD.jsonl`

**機密情報保護**:
- `password`, `token`, `api_key`, `secret` フィールドは自動的に `[REDACTED]` に置換
- 出力は最初の500文字のみ記録（ログサイズ削減）

**実装ファイル**:
- `internal/audit/logger.go` - ロガー本体
- `internal/tools/registry.go` - Execute()メソッドでログ記録

---

### 2. セッション履歴の暗号化（v0.25.0）

**目的**: 会話履歴に含まれる機密情報（APIキー、パスワード、プロジェクトコード）を保護

**有効化方法**:
```bash
export XELYON_ENCRYPT_HISTORY=1
./xelyon
```

**暗号化方式**:
- **アルゴリズム**: AES-256-GCM（認証付き暗号）
- **鍵導出**: PBKDF2（100,000回イテレーション、SHA-256）
- **ソルト**: 128-bit ランダム生成（ファイルごとに異なる）
- **ノンス**: 96-bit ランダム生成（暗号化ごとに異なる）

**暗号化キー管理**:
- キーファイル: `~/.xelyon/.session_key`
- パーミッション: `0600`（ユーザーのみ読み取り可能）
- 初回実行時に自動生成、以降は再利用

**互換性**:
- 既存の非暗号化セッションとの互換性を保持
- 復号化失敗時は該当行をスキップ（エラーで停止しない）

**実装ファイル**:
- `internal/crypto/encryption.go` - 暗号化/復号化ロジック
- `internal/history/storage.go` - Save()/Load()で暗号化統合

**セキュリティ考慮事項**:
- 暗号化キーは平文で保存（ユーザーのホームディレクトリに限定）
- より高度な保護が必要な場合は、OS のキーチェーン（Keychain/SecretService）統合を検討

---

### 3. APIレスポンス検証（v0.25.0）

**目的**: 不正なAPIレスポンスによる予期しない動作を防止

**実装機能**:
1. **ストリーミングレスポンス検証** (`ValidateStreamResponse`)
   - `choices`配列の存在チェック
   - `delta`または`message`フィールドの検証
   - 各choiceがオブジェクトであることを確認

2. **通常レスポンス検証** (`ValidateChatResponse`)
   - `choices[].message.content`の存在チェック

3. **ツール呼び出し検証** (`ValidateToolCall`)
   - `function.name`と`function.arguments`の必須フィールドチェック

4. **エラーレスポンス安全パース** (`ValidateErrorResponse`)
   - `error.message`の抽出
   - JSONパース失敗時のフォールバック

**実装ファイル**:
- `internal/api/validator.go` - 検証ロジック
- `internal/api/deepseek.go` - DeepSeekプロバイダーで適用（サンプル）

**拡張性**:
- 他のプロバイダー（OpenAI, Claude等）にも同じパターンで適用可能

---

### 4. セキュリティ対策一覧（v0.24.0で実装）

| 対策 | 実装内容 | ファイル |
|------|---------|---------|
| **コマンドインジェクション防止** | 連結文字（`;`, `&&`, `||`等）検出＋ブロック | `internal/tools/bash.go` |
| **パストラバーサル防止** | `ValidatePath()`でプロジェクト外アクセス禁止 | `internal/tools/validation.go` |
| **APIキー露出防止** | エラーメッセージから正規表現でAPIキーを削除 | `internal/api/provider.go` |
| **MCP環境変数サニタイズ** | KEY/TOKEN/SECRET変数をMCPサーバーに渡さない | `internal/mcp/client.go` |
| **グレースフルシャットダウン** | SIGINT/SIGTERM対応、MCPサーバー自動クローズ | `internal/agent/agent.go` |
| **ファイルパーミッション** | 機密ファイル（セッション、設定）を0600に変更 | `internal/history/storage.go`, `internal/config/config.go` |
| **レート制限ハンドリング** | HTTP 429検出＋Retry-Afterヘッダー解析 | `internal/api/provider.go` |
| **HTTPクライアント再利用** | コネクションプーリング、メモリ効率向上 | 全APIプロバイダー |

---

### 5. セキュリティ設定の推奨事項

**本番環境での推奨設定**:
```bash
# 監査ログを有効化
export XELYON_AUDIT_LOG=1

# セッション履歴を暗号化
export XELYON_ENCRYPT_HISTORY=1

# APIエンドポイントをプロキシ経由に設定（必要に応じて）
export DEEPSEEK_API_URL="https://proxy.example.com/v1/chat/completions"

# XELYON CLIを実行
./xelyon --provider deepseek
```

**監査ログローテーション**:
- ログファイルは日付ごとに分割（`audit_20260109.jsonl`）
- 古いログは定期的に削除または別ストレージに移動を推奨

**暗号化キーバックアップ**:
- `~/.xelyon/.session_key`を安全な場所にバックアップ
- キー紛失時は既存の暗号化セッションを復号化できない

---


## テスト実装（v0.26.0-v0.27.0）

### 1. テストアーキテクチャ

**テストヘルパー**: `internal/testutil/testutil.go`
- `CreateTempFile()` - 一時ファイル作成
- `AssertFileExists()` - ファイル存在確認
- `AssertFileContent()` - ファイル内容検証
- `SetupTempHome()` - HOME環境変数モック
- `ReadFile()` - ファイル読み込み

**モックパターン**:
```go
// confirm関数のモック
func setupTestConfirm(t *testing.T, result bool) {
    original := confirm
    confirm = func(prompt string) bool {
        return result
    }
    t.Cleanup(func() {
        confirm = original
    })
}

// ValidatePath関数のモック
func setupTestMocks(t *testing.T) {
    originalValidate := ValidatePath
    ValidatePath = func(path string) (string, error) {
        return filepath.Abs(path)
    }
    t.Cleanup(func() {
        ValidatePath = originalValidate
    })
}
```

---

### 2. テストカバレッジ（v0.27.0時点）

| パッケージ | テスト数 | カバレッジ | 状態 |
|-----------|---------|-----------|------|
| **internal/crypto** | 8 | 81.5% | ✅ 優秀 |
| **internal/audit** | 7 | 86.4% | ✅ 優秀 |
| **internal/tools** | 50 | 23.1% | ⚠️ 対象関数は95%+ |
| **internal/api** | 33 | 13.0% | ⚠️ 対象関数は95%+ |
| **プロジェクト全体** | 95 | 14.7% | - |

**注**: 全体カバレッジが低いのは、`cmd/`, `internal/agent/`, `internal/ui/` 等が未テストのため。テストした関数のカバレッジは平均95%以上。

---

### 3. テスト項目詳細

#### 3.1 暗号化テスト（internal/crypto）
- ✅ AES-256-GCM 暗号化/復号化ラウンドトリップ
- ✅ ソルト・ノンスのランダム性検証
- ✅ 認証タグ改ざん検出
- ✅ PBKDF2 鍵導出の一貫性
- ✅ パスフレーズファイル作成（0600パーミッション）

#### 3.2 監査ログテスト（internal/audit）
- ✅ JSONL形式記録
- ✅ 機密情報サニタイズ（password, token, api_key → [REDACTED]）
- ✅ 長い値の切り詰め（200文字, 500文字制限）
- ✅ 並行アクセス安全性（100 goroutines）
- ✅ ログ記録失敗時のサイレント動作

#### 3.3 ファイル操作テスト（internal/tools）
- ✅ read_file: ファイル読み込み、長いファイル切り詰め（100% coverage）
- ✅ write_file: 新規作成、上書き、ディレクトリ自動作成（90.9% coverage）
- ✅ str_replace: 厳密マッチ、複数マッチエラー、バックアップ作成（72.2% coverage）
- ✅ delete_file: 削除前バックアップ、Undo機能
- ✅ delete_lines: 行範囲削除、範囲外クランプ
- ✅ move_file: ファイル移動、上書き確認、クロスファイルシステム対応

#### 3.4 API検証テスト（internal/api）
- ✅ ValidateStreamResponse: choices配列、delta/messageフィールド検証（100% coverage）
- ✅ ValidateChatResponse: message.content検証（100% coverage）
- ✅ ValidateToolCall: function.name/arguments検証（100% coverage）
- ✅ sanitizeErrorMessage: APIキー削除（OpenAI, Google, Bearer token）（100% coverage）
- ✅ handleRateLimit: Retry-Afterヘッダー解析（100% coverage）

---

### 4. テスト実行方法

```bash
# 全テスト実行
go test ./...

# カバレッジ付き
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 特定パッケージのみ
go test ./internal/crypto/ -v
go test ./internal/audit/ -v
go test ./internal/tools/ -v
go test ./internal/api/ -v

# レースコンディション検出
go test ./... -race
```

---

## リリース自動化（v0.28.0-v0.28.3）

### 1. GoReleaser設定（.goreleaser.yml）

**基本設定**:
```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go fmt ./...
    - go test ./...
```

**ビルド設定**:
```yaml
builds:
  - id: xelyon
    binary: xelyon
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    tags: [norepomap]  # tree-sitter依存を除外
    ldflags:
      - -s -w
      - -X github.com/susugadx/xelyon-cli/internal/version.Version={{.Version}}
      - -X github.com/susugadx/xelyon-cli/internal/version.Commit={{.Commit}}
      - -X github.com/susugadx/xelyon-cli/internal/version.Date={{.Date}}
```

**アーカイブ設定**:
```yaml
archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    files: [LICENSE, README.md, XELYON.md]
```

**Changelog設定**:
```yaml
changelog:
  groups:
    - title: Features
      regexp: '^feat:'
    - title: Bug Fixes
      regexp: '^fix:'
```

**Homebrew Tap設定**:
```yaml
brews:
  - name: xelyon
    repository:
      owner: susugadx
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    install: |
      bin.install "xelyon"
```

---

### 2. GitHub Actions設定（.github/workflows/release.yml）

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'

    - name: Run tests
      run: go test -v ./...

    - name: Run GoReleaser
      uses: goreleaser/goreleaser-action@v6
      with:
        version: '~> v2'
        args: release --clean
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

---

### 3. CGO不要ビルド（v0.28.2）

**問題**: go-tree-sitterがCGOを必要とし、クロスコンパイルが複雑化

**解決策**: ビルドタグによる条件付きコンパイル

**実装**:
1. **既存ファイルにタグ追加**: `// +build !norepomap`
   - `internal/repomap/repomap.go`
   - `internal/repomap/parser.go`
   - `internal/repomap/extractor.go`
   - `internal/repomap/symbols.go`

2. **スタブ実装**: `internal/repomap/stub.go` (`// +build norepomap`)
   ```go
   type RepoMap struct {
       RootPath  string
       Files     []*FileSymbols
       MaxTokens int
   }
   
   func (rm *RepoMap) Generate() string {
       return "⚠️ Repo Map feature is disabled (requires CGO)"
   }
   ```

**ビルドパターン**:
```bash
# 開発用（CGO有効・Repo Map完全動作）
go build -o xelyon

# リリース用（CGO無効・Repo Map無効化）
go build -tags norepomap -o xelyon
```

---

### 4. リリース手順

```bash
# 1. バージョンタグ作成
git tag v0.28.3

# 2. GitHubにpush
git push origin main
git push origin v0.28.3

# 3. GitHub Actionsが自動実行
# ✅ go test ./...
# ✅ クロスコンパイル（Linux/macOS/Windows）
# ✅ GitHub Releasesにアップロード
# ✅ Homebrew Tap更新
```

**生成されるファイル**:
- `xelyon_0.28.3_linux_amd64.tar.gz`
- `xelyon_0.28.3_linux_arm64.tar.gz`
- `xelyon_0.28.3_darwin_amd64.tar.gz`
- `xelyon_0.28.3_darwin_arm64.tar.gz`
- `xelyon_0.28.3_windows_amd64.zip`
- `checksums.txt`

---

### 5. サポート環境

| OS | アーキテクチャ | 状態 |
|----|--------------|------|
| **Linux** | amd64 | ✅ |
| **Linux** | arm64 | ✅ |
| **macOS** | amd64 (Intel) | ✅ |
| **macOS** | arm64 (Apple Silicon) | ✅ |
| **Windows** | amd64 | ✅ |
| **Windows** | arm64 | ❌ 現状サポート外 |

---
