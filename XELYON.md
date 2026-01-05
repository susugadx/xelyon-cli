# XELYON CLI プロジェクト設定

## 概要
Go製のCLIツール。RAG検索とDeepSeek AIを使ったコーディングアシスタント。

## 技術スタック
- Go 1.22
- Cobra (CLI フレームワーク)
- DeepSeek API

## ファイル構造
- main.go: エントリーポイント
- cmd/root.go: コマンド定義
- internal/api/: API連携
- internal/file/: ファイル操作
