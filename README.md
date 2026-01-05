cat > XELYON.md << 'EOF'
# XELYON CLI プロジェクト設定

## 概要
Go製のCLIツール。RAG検索とDeepSeek AIを使ったコーディングアシスタント。

## 技術スタック
- Go 1.22
- Cobra (CLI フレームワーク)
- DeepSeek API

## コーディングルール
- エラーは必ずハンドリングする
- 日本語コメント可
- fmt.Println で絵文字を使ってステータス表示

## ファイル構造
- main.go: エントリーポイント
- cmd/root.go: コマンド定義
- internal/api/: API連携（XELYON, DeepSeek）
- internal/file/: ファイル読み書き

## ビルド
go build -o xelyon

## 今後の予定
- エージェント機能
- Claude API 対応
- 対話モード
EOF