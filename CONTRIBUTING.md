# Contributing to XELYON CLI

XELYON CLIへの貢献ありがとうございます！

## 開発環境セットアップ

### 必要なもの
- Go 1.23以上
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
export SERPER_API_KEY="your-serper-api-key"  # Web検索用（オプション）

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
feat: Web検索機能を追加

Serper APIを使ったリアルタイムWeb検索を実装。
- web_searchツール追加
- 上位5件の検索結果を取得
```

### ドキュメント更新ルール（必須）
機能追加・変更時は**必ず**以下を同時に更新：
- **README.md**: 使い方、コマンド説明、バージョン履歴
- **XELYON.md**: アーキテクチャ、内部設計、SystemPromptルール

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
- [XELYON.md](XELYON.md): プロジェクト詳細設計
