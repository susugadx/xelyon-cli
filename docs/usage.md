# 使い方詳細

基本的な使い方は [README](../README.md) を参照してください。このドキュメントでは詳細な機能を説明します。

## 目次

- [複数行入力](#複数行入力)
- [NAVモード](#navモード)
- [画像入力（マルチモーダル）](#画像入力マルチモーダル)
- [プロジェクト設定ファイル（xelyon.yaml）](#プロジェクト設定ファイルxelyonyaml)
- [サブエージェント委譲](#サブエージェント委譲)
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

## NAVモード

入力欄が空の状態で `Esc` を押すと NAV モードに入ります。起動直後のヘッダー、AI の回答、ツールブロックを含む表示中テキストをキーボードだけで移動・選択・コピーできます。

### 基本操作

```text
Esc           NAVモードに入る / 選択解除 / 入力モードへ戻る
j / k         行移動
d / u         半ページ移動
gg / G        先頭 / 末尾へ移動
Tab           ツールブロックをフォーカス、または折りたたみ切り替え
q / i         入力モードへ戻る
```

NAV モードに入った直後のカーソルは、現在の viewport 先頭行に置かれます。マウスホイールは viewport だけをスクロールし、カーソル位置は変わりません。

### ビジュアル選択

```text
count+j/k/h/l 数字プレフィックス付き移動
w / b / e     単語移動
0 / ^ / $     行頭 / 非空白先頭 / 行末
v             文字単位のビジュアル選択を開始
V             行単位のビジュアル選択を開始
y             選択範囲をコピー
yy            現在行をコピー
```

- `count` は `3j`、`10l`、`12G`、`3gg` のように主要な移動キーへ適用できます
- `v` は Vim の visual 相当で、複数行にまたがる文字選択にも対応します
- `V` は行単位で範囲を広げます
- `w` / `b` / `e`、`0` / `^` / `$` は通常時だけでなく Visual 中の範囲拡張にも使えます
- コピー時は ANSI 色を除去したプレーンテキストをクリップボードへ送ります
- ツールブロックを `Tab` でフォーカスしている間の `y` は、従来どおりそのブロック詳細をコピーします

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
- `final_checks`: 明示完了時の final checks（変更後に実行する検証コマンド）

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

final_checks:
  commands:
    - "npm run lint && npm run build && npm test"
  timeout: 600
```

> **Note:** `final_checks.commands` を定義すると、AI が `completed_with_changes` の完了候補で自動実行します。省略時は `config.yaml` の final_checks 設定が使われます。旧 `verification` も互換入力として読み取られます。

---

## サブエージェント委譲

XELYON は探索・調査タスクを `spawn_agent` / `wait_agent` でサブエージェントへ委譲できます。

- 親に返るのはサブの最終レポートだけで、中間の `read_file` / `search_code` 出力は親コンテキストへ残りません
- `wait_agent` 実行中はサブエージェントのツール実行が親UIへ逐次表示され、`str_replace` は色付き diff で確認できます
- `sub_agent.default_model` が空ならメイン provider の最安モデルを自動選択し、推論強度は off、同時実行数は 5 です
- サブエージェント自身には `spawn_agent` / `wait_agent` を渡さないため、再帰的な spawn は行いません

設定例:

```yaml
# ~/.xelyon/config.yaml
sub_agent:
  enabled: true
  default_model: ""
  default_effort: off
  max_concurrent: 5
```

サブエージェントは内部ツールなので、通常は自然言語の指示から自動で使われます。探索対象のファイルパスや観点を明示すると、より短い往復で結果が返ります。

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
🗺️  Project map loaded (42 files, 118 symbols)
📋 Context size: ~10.8k tok
   ├── Base prompt: ~3.9k
   ├── Tools (9): ~2.7k
   ├── Project map (118 symbols, 42 files): ~3.6k
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
