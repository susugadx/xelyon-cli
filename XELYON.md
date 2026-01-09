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
│   │   └── verify.go      # 自動検証（go fmt, go test, rollback）
│   ├── mcp/               # MCP連携
│   │   ├── client.go      # MCPサーバー接続・ツール管理
│   │   └── integration.go # Tool Registry統合
│   ├── repomap/           # Repo Map（コード構造解析）
│   │   ├── parser.go      # 言語パーサー管理
│   │   ├── symbols.go     # シンボル定義
│   │   ├── extractor.go   # Tree-sitterでシンボル抽出
│   │   └── repomap.go     # Repo Map生成
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
- **コマンド処理**: `/save`, `/load`, `/sessions`, `/undo`, `/config`, `/model`, `/version`, `/clear`, `/history`, `/help`
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
- **確認UI改善**: 英語/日本語併記、ボックス囲みの見やすい差分表示
- **簡潔な表示**: ツール呼び出し時のJSON表示を廃止、人間が読める形式に
- **エラーヒント**: 複数マッチ時にファイルプレビュー（先頭50行）を表示
- **フレームワーク自動検出**: run_test/formatが言語・ツールを自動検出

#### 3. UI/UX (internal/ui/)
- **スピナー**: API呼び出し中のローディング表示（80ms更新）
- **ページング**: 100行を超える出力を自動的に分割表示
- **色付け**: cyan/green/yellow/redで情報を視覚的に区別

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
- **フォールバック実装**: 非ストリーミング時は自動で一括表示

#### 6. 設定管理 (internal/config/)
- **YAML形式**: `~/.xelyon/config.yaml`
- **プロバイダー設定**: default_provider, provider_models
- **デフォルトモデル**: default_model設定で起動時のモデル指定
- **プロバイダーごとの設定**: 各プロバイダーのデフォルトモデルと利用可能モデル一覧
- **自動作成**: 初回起動時にデフォルト設定を自動生成
- **バリデーション**: 無効なプロバイダー/モデル名を拒否
- **.env自動読み込み**: `godotenv`でプロジェクトディレクトリの`.env`を自動読み込み
  - 起動時に`main.go`で実行（ファイルが存在しなくてもエラーにならない）
  - プロジェクトごとに異なるAPIキーやプロバイダーを設定可能
  - `.env.example`でサンプルファイルを提供

#### 7. バージョン管理 (internal/version/)
- **一元管理**: `version.go`でバージョンを定数管理
- **複数表示**: 起動バナー、`/version`コマンド、`--version`フラグ
- **自動反映**: Version定数を変更するだけで全体に反映

#### 8. 自動検証システム (internal/agent/verify.go)
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
- **Tree-sitter解析**: AST解析で正確なシンボル抽出
- **複数言語対応**: Go, JavaScript, TypeScript, Python
- **シンボル抽出**: 関数、メソッド、構造体、クラス、インターフェース
- **トークン制限**: 大規模プロジェクトでも効率的にコンテキスト圧縮
- **自動生成**: 起動時にプロジェクトをスキャン
- **除外パターン**: node_modules, .git, vendor等は自動除外

## コーディングルール

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
  - Ctrl+C時に`agent.Cleanup()`呼び出し
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
- [ ] タイムスタンプ付き複数世代バックアップ
- [ ] `/undo all` - セッション中の全変更を一括取り消し
- [ ] `/changes` - 変更履歴一覧表示
- [ ] 永続的Undo - セッション終了後も復元可能
- [ ] .gitignoreへの.bak自動追加
- [ ] ループ検知の回数をカスタマイズ可能に
- [ ] APIリトライの回数・待機時間をカスタマイズ可能に
- [ ] 差分表示の省略行数をカスタマイズ可能に

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

