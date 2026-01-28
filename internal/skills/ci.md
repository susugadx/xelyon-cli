# CI (Continuous Integration)

## ログ確認
**重要**: GitHub Actions のログは gh CLI でしか取得できない。MCP では取得不可。
```bash
# 最近の実行一覧
gh run list --limit 5

# 失敗したログを確認
gh run view <run-id> --log-failed

# 全ログを確認
gh run view <run-id> --log

# 特定ワークフローの実行一覧
gh run list --workflow=ci.yml
```

## CI 失敗時の対処
1. `gh run list --limit 5` で失敗を特定
2. `gh run view <run-id> --log-failed` でエラー確認
3. エラー内容に基づいて修正
4. push して再実行

## よくあるエラー
- lint エラー → `golangci-lint run` でローカル確認
- test エラー → `go test ./...` でローカル確認
- build エラー → `go build ./...` でローカル確認

## ワークフロー再実行
```bash
# 失敗したジョブだけ再実行
gh run rerun <run-id> --failed

# 全部再実行
gh run rerun <run-id>
```

## 特定ブランチの確認
```bash
# 特定ブランチの実行一覧
gh run list --branch main

# PR の CI 状態確認
gh pr checks
```

## キャッシュ確認
```bash
# キャッシュ一覧
gh cache list

# キャッシュ削除
gh cache delete <cache-id>
```