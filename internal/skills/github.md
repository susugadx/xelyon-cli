# GitHub (PR / Issue)

## PR 操作
```bash
# PR 一覧
gh pr list

# PR 作成
gh pr create --title "タイトル" --body "説明"

# 現在のブランチから PR 作成（インタラクティブ）
gh pr create

# PR の詳細確認
gh pr view <pr-number>

# PR をマージ
gh pr merge <pr-number>

# PR のレビュー状態確認
gh pr checks <pr-number>
```

## Issue 操作
```bash
# Issue 一覧
gh issue list

# Issue 作成
gh issue create --title "タイトル" --body "説明"

# Issue の詳細確認
gh issue view <issue-number>

# Issue をクローズ
gh issue close <issue-number>

# ラベル付きで Issue 作成
gh issue create --title "バグ" --label "bug"
```

## よくあるエラー
- 認証エラー → `gh auth login` を実行
- リポジトリが見つからない → `gh repo set-default` で設定

## PR レビュー
```bash
# レビューコメント追加
gh pr review <pr-number> --comment --body "コメント"

# 承認
gh pr review <pr-number> --approve

# 変更リクエスト
gh pr review <pr-number> --request-changes --body "理由"
```

## PR 差分確認
```bash
# PR の差分を見る
gh pr diff <pr-number>

# ファイル一覧
gh pr view <pr-number> --json files
```

## Issue 検索
```bash
# ラベルで絞り込み
gh issue list --label "bug"

# アサインされた Issue
gh issue list --assignee @me

# 状態で絞り込み
gh issue list --state closed
```

## コンフリクト解決
```bash
# マージ中のコンフリクト確認
git status

# コンフリクト解決後
git add <file>
git commit

# マージ中止
git merge --abort
```

## 特定コミットの操作
```bash
# コミットの詳細
git show <commit-hash>

# 特定コミットの変更を取り込む
git cherry-pick <commit-hash>
```

## リベース（注意）
```bash
# main の最新を取り込む
git rebase main

# コンフリクト解決後
git rebase --continue

# リベース中止
git rebase --abort
```

## タグ
```bash
# タグ一覧
git tag

# タグ作成
git tag v1.0.0

# タグをプッシュ
git push origin v1.0.0
```