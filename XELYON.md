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
- **19種類のツール**:
  - **ファイル編集**: read_file, write_file, str_replace, append_file, prepend_file
  - **ファイル管理**: list_dir, create_dir
  - **Git操作**: git_status, git_diff, git_add, git_commit, git_push, git_log
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
