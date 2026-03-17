# 使い方詳細

基本的な使い方は [README](../README.md) を参照してください。このドキュメントでは詳細な機能を説明します。

## 目次

- [複数行入力](#複数行入力)
- [画像入力（マルチモーダル）](#画像入力マルチモーダル)
- [プロジェクト設定ファイル（xelyon.yaml）](#プロジェクト設定ファイルxelyonyaml)
- [コードレビュー](#コードレビュー)
- [リファクタリング](#リファクタリング)
- [確認UI（y/n/c）](#確認uiync)
- [状態表示](#状態表示)
- [Dry Run](#dry-run)

---

## 複数行入力

プロンプトやコードを複数行で入力できます。

### 方法1: 直接ペースト（推奨）

**Bracketed Paste Mode** が有効な場合、複数行のテキストをそのままペーストできます。

```bash
> func main() {
      fmt.Println("Hello")
  }
# → 複数行が1つの入力として認識される
```

- コードエディタやブラウザから複数行をコピー＆ペースト
- 自動的に複数行として認識される
- 対応: iTerm2, Terminal.app, GNOME Terminal, Windows Terminal, tmux, kitty

**WSL環境での問題**: WSL等でエスケープシーケンスが文字として表示される場合は、以下で無効化できます：
```bash
# 環境変数で無効化
export XELYON_BRACKETED_PASTE=0

# または config.yaml で
paste:
  bracketed_paste: false
```

### 方法2: ``` マーカー

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

### 方法3: /paste コマンド

Bracketed Paste非対応環境向け。

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

## プロジェクト設定ファイル（xelyon.yaml）

プロジェクトルートに `xelyon.yaml` を置くと、起動時に自動で読み込まれ、AIのコンテキストとして使用されます。

### 作成

```bash
> /init
```

### xelyon.yaml について

xelyon.yaml は AI 用の構造化設定ファイルです。

**書くべきこと:**
- `context`: プロジェクトの概要（1-2行）
- `rules`: 開発ルール・コーディング規約
- `conditional`: 特定パスにだけ適用したい rules/context
- `ignore`: Project Map / `list_dir` / `search_code` で共有したい ignore パターン
- `hooks`: 完了時フック・ステップ完了時フック（変更後に実行する検証コマンド）

**書かないこと:**
- ディレクトリ構造やコードマップの詳細
- 詳細なドキュメント

> 起動時は軽量な Project Map manifest が自動生成されるため、`xelyon.yaml` にファイル一覧や関数目次を書く必要はありません。

### xelyon.yaml の例

```yaml
# my-project - Project Configuration
# AI 用コンテキスト。ドキュメントではありません。
# AI が許可なくこのファイルを肥大化させることを禁止します。

context: |
  Webアプリケーションのバックエンドサーバー

rules:
  - "変数名はキャメルケース"
  - "関数コメント必須"
  - "コミットメッセージは日本語で"

conditional:
  - name: API handlers
    paths:
      - "internal/handlers/**/*.go"
      - "internal/api/**/*.go"
    rules:
      - "公開関数・型には日本語コメント必須"
    context: |
      HTTP handler は timeout と error handling を必須にします。

ignore:
  patterns:
    - "coverage"
    - "*.generated.go"

hooks:
  on_completion:
    - "npm run lint && npm run build && npm test"
  on_step_complete:
    - "git add -A && git commit -m 'Step {{step_id}}: {{step_description}}'"
  timeout: 60
  max_retry: 3
```

> **Note:** `hooks.on_completion` を定義すると、AI がコード変更後に自動でそのコマンドを実行します。`hooks.on_step_complete` を定義すると、Plan Mode の各ステップ完了時にコマンドを実行します（テンプレート変数: `{{step_id}}`, `{{step_description}}`, `{{step_status}}`）。省略時は `config.yaml` の hooks 設定が使われます。

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
```

### フラグ一覧

| フラグ | 短縮形 | 説明 |
|--------|--------|------|
| `--all` | `-a` | すべてのgit変更をレビュー |
| `--security` | `-s` | セキュリティフォーカス |
| `--test` | `-t` | テストカバレッジフォーカス |
| `--max-issues N` | | 表示する最大Issue数 |

### 検出ルール

- **セキュリティ**: コマンドインジェクション、パストラバーサル、弱い暗号化
- **一般**: 大きな差分、TODO/FIXME追加、ドキュメント不足
- **テスト**: テストカバレッジ不足、アサーション不足

レポートは `~/.xelyon/reviews/` に保存されます。

---

## リファクタリング

`/refactor` コマンドでコードのリファクタリング分析を行います（静的解析のみ）。

```bash
# 基本分析
> /refactor

# 特定タイプのみ
> /refactor --type split-file

# 閾値カスタマイズ
> /refactor --max-file-lines 300 --max-func-lines 50
```

### フラグ一覧

| フラグ | 説明 |
|--------|------|
| `--type` | フィルタ（split-file, extract-method, dry, rename） |
| `--max-file-lines N` | ファイル行数上限 |
| `--max-func-lines N` | 関数行数上限 |

---

## 確認UI（y/n/c）

すべての確認プロンプトは `y/n/c`（yes/no/comment）をサポートします。

- `y`: 実行
- `n`: キャンセル
- `c`: コメントを入力してAIに再提案させる

### 対象例

- ファイル変更（`write_file`, `str_replace`）
- Git操作（`bash: git commit`）
- テスト失敗時のロールバック

### 設定

無効化: `XELYON_INTERACTIVE_CONFIRM=0`

---

## 状態表示

各ターンの最後に現在の状態を表示します。

### 起動時の表示例

```bash
$ xelyon
📋 xelyon.yaml loaded
🗺️  Project map loaded (manifest from 42 files)
📋 Context size: ~8.1k tok
   ├── Base prompt: ~3.9k
   ├── Tools (9): ~2.7k
   ├── Project map manifest (42 files): ~0.9k
   └── xelyon.yaml: ~0.6k
```

- `running`: 実行中
- `waiting_input`: 入力待ち
- `waiting_approval`: 承認待ち
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
> ユーザー認証機能を追加して
# → 自然言語で指示するだけでAIが実装
# → ファイル編集前には差分を表示して確認
> /exit
```

### コードレビュー（静的解析）

```bash
> /review --security
> /review **/*.go --test
```

### 日常的な開発

```bash
$ xelyon
> バグ #123 を修正して
> /review
> /exit
```

---

## キャッシュについて
