# XELYON CLI

> ⚠️ **このファイルは AI 用のコンテキストです。ドキュメントではありません。**
>
> - 詳細な仕様やドキュメントはここに書かない
> - 機能追加時もこのファイルは更新不要（README.md と docs/ を更新）
> - **AI が許可なくこのファイルを変更することを禁止**

## 概要

Go 製 AI コーディングアシスタント CLI。8 LLM プロバイダー対応（DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq, OpenRouter, Bedrock）。

## 技術スタック

- Go 1.24+
- Cobra（CLI）
- Tree-sitter（コード解析）

## 開発ルール

- コミット前は `make ci-check` 必須（go fmt → build → lint → test）
- 新機能・バグ修正時はテストも追加
- エラーハンドリング必須（I/O 操作、HTTP には Timeout 設定）
- コミットは日本語で機能単位で小さく、具体的に記述
- 新機能追加時は README.md と docs/ を更新（バグ修正は不要）
- 設定追加時は `make gen-all` を実行
- 未使用コードは作らない（呼び出し元を確認してから実装）
- 公開関数・型には日本語コメント必須（godoc 形式）
- 複雑なロジックには処理の意図をコメント

## ビルド＆テスト

```bash
go build -o xelyon && go test ./...
```

## 参照先

詳細は README.md と docs/ を参照
