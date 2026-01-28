# スキルズ

XELYON はタスクに応じて関連スキルを自動ロードします。

## 仕組み

ユーザーの入力からキーワードを検出し、関連するスキルファイルを読み込んでAIのコンテキストに追加します。
```
> CI通して
�� Skill loaded: ci
🔧 Tool: bash
   Command: gh run list --limit 5
```
## スキルズの役割

スキルズは AI にコマンドを教えるだけでなく、**どのツールを優先すべきか**を指示します。

### 例: CI ログ確認

**スキルなしの場合:**
- AI は MCP で GitHub API を使おうとする
- しかし MCP では Actions のログは取得できない

**スキルありの場合:**
- 「GitHub Actions のログは gh CLI でしか取得できない」と指示
- AI は `gh run list` → `gh run view --log-failed` を使う

### スキルズに書くべきこと

1. **優先度** - 「MCP じゃなく gh CLI を使え」
2. **手順** - 「まず list → 次に view --log-failed」
3. **トラブルシュート** - 「このエラーはこう対処」

## 組み込みスキル

| スキル | キーワード例 | 内容 |
|--------|-------------|------|
| `ci` | CI, Actions, workflow, ビルド | GitHub Actions ログ確認 |
| `github` | PR, Issue, プルリク, レビュー | PR / Issue 操作 |
| `git` | git, commit, push, ブランチ | Git 基本操作 |
| `testing` | test, テスト, coverage | テスト実行（Go/JS/Python等） |
| `docker` | docker, container, コンテナ | Docker 操作 |

## カスタムスキル

`.xelyon/skills/` にスキルファイルを追加できます。組み込みスキルより優先されます。

### 作成方法
```bash
mkdir -p .xelyon/skills
```

`.xelyon/skills/deploy.md`:
```markdown
# デプロイ

## 本番デプロイ
ssh prod "cd /app && git pull && systemctl restart app"

## ステージングデプロイ
ssh staging "cd /app && git pull"
```

### キーワード

ファイル名がキーワードになります。`deploy.md` は「deploy」「デプロイ」で検出されます。

## /skill コマンド（予定）
```bash
/skill           # スキル一覧を表示
/skill ci        # 手動でスキルをロード
```
