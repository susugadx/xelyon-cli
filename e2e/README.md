# E2E Tests

実際のLLM APIを使用するEnd-to-Endテスト。

## 実行方法

```bash
# 環境変数を設定して実行
XELYON_E2E=1 OPENAI_API_KEY=sk-... go test ./e2e/ -v -timeout 300s

# Makefile経由
make e2e
```

## 前提条件

- `XELYON_E2E=1` 環境変数が必須（未設定時はスキップ）
- `OPENAI_API_KEY` 環境変数が必須（未設定時は個別テストがスキップ）

## テスト内容

| テスト | 内容 |
|--------|------|
| TestE2E_SearchCode | gather_context ツールで `func main` を検索 |
| TestE2E_ReadFile | read_file ツールで go.mod を読み込み |
| TestE2E_ReadFileBatch | read_file ツールで go.mod + Makefile を一括読み込み |
| TestE2E_SearchCodeSymbolResolve | gather_context ツールでシンボル解決 |
| TestE2E_SimpleQuery | 自然言語での調査タスク |

## コスト

- モデル: `gpt-5.4-nano`（最安）
- 1回あたり約 $0.006 以下
- 通常の `make ci-check` では実行されない

## CI

GitHub Actions で週1回（月曜 UTC 0:00）自動実行。手動実行も可能。
