# XELYON CLI プロジェクト設定

> ⚠️ **注意**: このファイルは **xelyon-cli 自体の開発用** です。
> xelyon を使いたいだけなら [バイナリをダウンロード](https://github.com/susugadx/xelyon-cli/releases) してください。

## 概要
Go製のAI搭載コーディングアシスタントCLI。複数のLLMプロバイダー（DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq）に対応。

## 技術スタック
- **Go 1.22+**
- **Cobra** - CLIフレームワーク
- **Tree-sitter** - コード構造解析（Repo Map）
- **fatih/color** - ターミナル色付け

## 開発ルール

### コミット
- メッセージは日本語OK
- 具体的に書く（❌「修正」→ ✅「HTTPタイムアウト30秒を追加」）
- 機能単位で小さくコミット

### コード品質
変更後は必ず実行：
```bash
go fmt ./...
go mod tidy
go build -o xelyon
go test ./...
```

### エラーハンドリング
- すべてのI/O操作でエラーチェック必須
- HTTPクライアントには必ずTimeout設定
- context.Contextを使ってキャンセル可能に

## ディレクトリ構造

```
xelyon-cli/
├── main.go              # エントリーポイント
├── cmd/root.go          # Cobraコマンド定義
├── internal/
│   ├── agent/           # エージェント（対話ループ、Plan Mode、検証）
│   ├── api/             # LLMプロバイダー（Provider Pattern）
│   ├── tools/           # ツール（Registry方式、35種類）
│   ├── mcp/             # MCP連携（外部ツール統合）
│   ├── repomap/         # Repo Map（Tree-sitter、30言語対応）
│   ├── review/          # コードレビュー機能
│   ├── refactor/        # リファクタリング機能
│   ├── config/          # 設定管理
│   ├── history/         # セッション管理
│   ├── ui/              # スピナー、ページャー
│   ├── cache/           # プロンプトキャッシュ
│   ├── crypto/          # 暗号化
│   ├── audit/           # 監査ログ
│   └── version/         # バージョン管理
├── docs/                # ユーザー向けドキュメント
└── README.md            # ユーザー向け説明
```

## 主要インターフェース

### Provider（LLM）
```go
type Provider interface {
    Name() string
    ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error)
}
```

### Tool（ツール）
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() []ToolParameter
    Run(args map[string]string) (string, *FileChange, error)
    Safety() ToolSafety
}
```

## ツール安全レベル

| レベル | 説明 | 例 |
|-------|------|---|
| SafetyHigh | 読み取りのみ、自動承認可 | read_file, grep, list_files |
| SafetyMedium | 変更あり、確認推奨 | write_file, str_replace |
| SafetyLow | 危険、必ず確認 | bash, git_push, delete_file |

## SystemPromptルール

AIの振る舞いを定義（`internal/agent/prompts.go`）：

1. ツール呼び出しは1回につき1つ
2. 実行前に必ず説明を入れる
3. ファイル編集前に必ず読む
4. bash実行には細心の注意
5. エラー時は3回まで自動リトライ
6. ユーザーの確認なしに危険な操作をしない

## プロンプトキャッシュ

| プロバイダー | 方式 |
|-------------|------|
| Claude | `cache_control` で明示的キャッシュ |
| OpenAI/DeepSeek | API側で自動キャッシュ |
| Gemini | Implicit Caching（自動） |

## 新機能追加時のチェックリスト

1. [ ] `internal/` に実装
2. [ ] ツールなら `internal/tools/` に追加、Registry登録
3. [ ] コマンドなら `internal/agent/agent_commands.go` に追加
4. [ ] テスト追加
5. [ ] `go fmt && go test ./...` 通過
6. [ ] ドキュメント更新
   - ツール追加 → `docs/commands.md`
   - コマンド追加 → `docs/commands.md`
   - 設定追加 → `docs/config.md`
   - 大きな新機能 → README.md の特徴セクション

## ビルド＆テスト

```bash
# ビルド
go build -o xelyon

# テスト
go test ./...

# カバレッジ
go test -cover ./...

# リント
golangci-lint run
```

## リリース

```bash
# バージョン更新
# internal/version/version.go の Version を変更

# タグ作成＆プッシュ（GitHub Actionsが自動リリース）
git tag v0.X.0
git push origin v0.X.0
```
