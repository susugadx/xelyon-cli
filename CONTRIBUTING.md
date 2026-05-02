# Contributing to XELYON CLI

XELYON CLIへの貢献ありがとうございます！

## 開発環境セットアップ

### 必要なもの
- Go 1.24以上
- Git

### セットアップ手順
```bash
# リポジトリをクローン
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli

# 依存関係のインストール
go mod tidy

# ビルド
go build -o xelyon

# 環境変数設定
export DEEPSEEK_API_KEY="your-api-key"
# Web検索を別プロバイダーで試す場合は以下も設定
# export GEMINI_API_KEY="your-gemini-api-key"
# export OPENAI_API_KEY="your-openai-api-key"
# export ANTHROPIC_API_KEY="your-anthropic-api-key"

# 実行
./xelyon
```

## コミットルール

### コミットメッセージ形式
```
<type>: <subject>

<body>
```

**Type**:
- `feat`: 新機能
- `fix`: バグ修正
- `docs`: ドキュメントのみの変更
- `refactor`: リファクタリング
- `test`: テストの追加・修正
- `chore`: ビルドプロセスやツールの変更

**例**:
```
feat: Web検索プロバイダー設定を追加

ネイティブWeb検索の実行先を設定できるようにした。
- web_search.provider を追加
- OpenAI / Gemini / Claude のネイティブ検索を選択可能
- 検索結果キャッシュを維持
```

### ドキュメント更新ルール（必須）
機能追加・変更時は**必ず**以下を同時に更新：
- **README.md**: 使い方、コマンド説明、バージョン履歴

ドキュメント更新なしのPRは受け付けません。

## コード品質

### プルリクエスト前に実行
```bash
# フォーマット
go fmt ./...

# ビルド確認
go build -o xelyon

# テスト（追加されている場合）
go test ./...

# Linter（推奨）
golangci-lint run
```

### Slash候補順序の更新フロー
`internal/commandcatalog` の `SortWeight` や discoverable 設定を変更した場合は、slash候補順序のゴールデンテストも確認してください。

```bash
go test ./internal/tui/slash -run TestSuggestions_GoldenOrderForRootPrefix -count=1
```

失敗した場合は、`internal/tui/slash/command_suggestions_golden_test.go` の `want` 順序を意図した並びに更新し、変更理由をPR説明に記載してください。

### `SplitStrict` fuzz のローカル確認（任意）
`commandruntime.SplitStrict` に変更を入れた場合は、短時間でも fuzz を回すことを推奨します。

```bash
go test ./internal/commandruntime -run '^$' -fuzz=FuzzSplitStrict -fuzztime=30s -count=1
```

### 設定を追加・変更した場合
設定オプションを追加・変更した場合は、以下を実行してドキュメントを更新してください：

```bash
make gen-all
```

これにより以下が自動生成されます：
- `config.yaml.example` - 設定例ファイル（コメント付き）
- `docs/config.md` 内の設定例セクション

### コーディング規約
- すべてのI/O操作でエラーチェック必須
- HTTPクライアントには必ずTimeout設定
- 危険なコマンド（rm -rf等）はブロック
- ファイル編集時は.bakバックアップ作成

## プルリクエストの流れ

1. Forkしてブランチ作成
   ```bash
   git checkout -b feature/my-feature
   ```

2. 変更を実装

3. コミット
   ```bash
   git add .
   git commit -m "feat: 新機能を追加"
   ```

4. プッシュ
   ```bash
   git push origin feature/my-feature
   ```

5. GitHub上でPull Request作成

## 質問・議論
- Issue: バグ報告、機能提案
- Discussions: 質問、アイデア共有

## 参考資料
- [README.md](README.md): ユーザー向けドキュメント
