# 使い方詳細

基本的な使い方は [README](../README.md) を参照してください。このドキュメントでは詳細な機能を説明します。

## 目次

- [複数行入力](#複数行入力)
- [画像入力（マルチモーダル）](#画像入力マルチモーダル)
- [プロジェクト設定ファイル（XELYON.md）](#プロジェクト設定ファイルxelyonmd)
- [コードレビュー](#コードレビュー)
- [リファクタリング](#リファクタリング)
- [確認UI（y/n/c）](#確認uiync)
- [状態表示](#状態表示)
- [Dry Run](#dry-run)

---

## 複数行入力

プロンプトやコードを複数行で入力できます。

### 方法1: ``` マーカー

```bash
> ```
📝 Multiline input mode (end with ``` on a new line)
  1 | 以下のコードをレビューして
  2 |
  3 | func main() {
  4 |     fmt.Println("Hello")
  5 | }
  6 | ```
✅ Captured 5 lines
```

### 方法2: ペースト（Bracketed Paste Mode）

- コードエディタやブラウザから複数行をコピー＆ペースト
- 自動的に複数行として認識される
- 対応: iTerm2, Terminal.app, GNOME Terminal, Windows Terminal, tmux

### 方法3: /paste コマンド

WSL等のBracketed Paste非対応環境向け。

```bash
> /paste
📝 Paste Mode
[長文をペースト]
[空行]
[空行]
✅ Captured N lines
```

- 終了: 空行2回、`END`、Ctrl+D
- キャンセル: `/cancel`
- エイリアス: `/p`

---

## 画像入力（マルチモーダル）

画像ファイルを指定して、画像を見ながらコード生成・分析ができます。

```bash
# ワイヤーフレームからReactコンポーネントを生成
xelyon -i wireframe.png "この画面をReactで実装して"

# エラースクリーンショットから原因を分析
xelyon --image error.png --provider gemini "このエラーを修正して"

# 対話モード中にも使用可能（image:プレフィックス）
> image:screenshot.png このUIの問題点を教えて
```

**対応プロバイダー**: Gemini, Claude, OpenAI（DeepSeek, Ollama, Groqは非対応）

---

## プロジェクト設定ファイル（XELYON.md）

プロジェクトルートに `XELYON.md` を置くと、起動時に自動で読み込まれ、AIのコンテキストとして使用されます。

### 作成・更新

```bash
# 初回: 新規作成（対話式）
> /init

# 以降: 差分更新（カスタムルールは保持）
> /sync
```

| コマンド | 動作 | 用途 |
|---------|------|------|
| `/init` | 新規作成（既存があれば上書き確認） | 初回セットアップ |
| `/sync` | 差分更新（カスタムルールを保持） | コード変更後の同期 |

### XELYON.md の例

```markdown
## コーディングルール
- 変数名はキャメルケース
- 関数コメント必須

## 技術スタック
- React 18 + TypeScript
- Tailwind CSS

## AI への指示
- コード変更後は必ずテストを提案して
- コミットメッセージは日本語で
```

### ルール学習

終了時に `/exit` すると、会話から抽出したルールを XELYON.md に追記する提案が表示されます。

例: 「テストは必ず書いて」と何度か言った場合、終了時に「テストを必ず書く」というルールを追記するか提案されます。

---

## コードレビュー

`/review` コマンドでセッション中の変更をAIがレビューします。

```bash
# セッション中の変更をレビュー
> /review

# 特定のファイル/ディレクトリ
> /review internal/api/

# globパターン
> /review **/*.go --security

# 修正提案を生成
> /review --fix

# 並列モードで高速修正
> /review --fix --parallel
```

### フラグ一覧

| フラグ | 短縮形 | 説明 |
|--------|--------|------|
| `--all` | `-a` | すべてのgit変更をレビュー |
| `--security` | `-s` | セキュリティフォーカス |
| `--test` | `-t` | テストカバレッジフォーカス |
| `--fix` | `-f` | 修正提案を生成 |
| `--yes` | `-y` | 確認をスキップ |
| `--parallel` | `-p` | 並列モードで修正を適用 |
| `--workers N` | `-w N` | 並列ワーカー数（デフォルト: 4） |

### 検出ルール

- **セキュリティ**: コマンドインジェクション、パストラバーサル、弱い暗号化
- **一般**: 大きな差分、TODO/FIXME追加、ドキュメント不足
- **テスト**: テストカバレッジ不足、アサーション不足

レポートは `~/.xelyon/reviews/` に保存されます。

---

## リファクタリング

`/refactor` コマンドでコードのリファクタリング分析を行います。

```bash
# 基本分析
> /refactor

# AI分析付き
> /refactor --ai

# 自動修正まで実行
> /refactor --ai --fix --yes

# 閾値カスタマイズ
> /refactor --max-file-lines 300 --max-func-lines 50
```

### フラグ一覧

| フラグ | 説明 |
|--------|------|
| `--ai` | AI分析を有効化 |
| `--fix` | 修正を適用 |
| `--yes` | 確認をスキップ |
| `--type` | フィルタ（long_file, long_function等） |
| `--max-file-lines N` | ファイル行数上限 |
| `--max-func-lines N` | 関数行数上限 |

---

## 確認UI（y/n/c）

すべての確認プロンプトは `y/n/c`（yes/no/comment）をサポートします。

- `y`: 実行
- `n`: キャンセル
- `c`: コメントを入力してAIに再提案させる

### 対象例

- ファイル上書き（`copy_file`, `move_file`）
- Git操作（`git_add`, `git_commit`）
- テスト失敗時のロールバック

### 設定

無効化: `XELYON_INTERACTIVE_CONFIRM=0`

---

## 状態表示

各ターンの最後に現在の状態を表示します。

- `running`: 実行中
- `waiting_input`: 入力待ち
- `waiting_approval`: 承認待ち（Plan Mode）
- `aborted`: 中断

---

## Dry Run

`--dry-run` でツール実行をシミュレートします。

```bash
xelyon --dry-run
```

- ツール実行は行われない
- 変更履歴（Undo対象）も作られない
- 「まず安全に確認したい」場合に使用

対話中の切り替え: `/dryrun on` / `/dryrun off`

---

## ワークフロー例

### 新機能実装

```bash
> /plan on
> ユーザー認証機能を追加して
# → 計画表示 → y で承認 → 自動実行
> /review --fix
> /exit
```

### コードレビュー + 自動修正

```bash
> /review --security --fix
> /review --fix --parallel --workers 8
```

### 日常的な開発

```bash
$ xelyon
> バグ #123 を修正して
> /review
> /undo  # 問題があれば戻す
> /exit
```
