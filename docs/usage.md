# 使い方詳細

基本的な使い方は [README](../README.md) を参照してください。このドキュメントでは詳細な機能を説明します。

## 目次

- [複数行入力](#複数行入力)
- [添付（ファイル・画像）](#添付ファイル画像)
- [NAVモード](#navモード)
- [画像入力（マルチモーダル）](#画像入力マルチモーダル)
- [プロジェクト設定ファイル（xelyon.yaml）](#プロジェクト設定ファイルxelyonyaml)
- [サブエージェント委譲](#サブエージェント委譲)
- [確認UI（y/n/c）](#確認uiync)
- [状態表示](#状態表示)

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

---

## 添付（ファイル・画像）

TUI の入力ドラフトには、テキストだけでなくファイル・画像を添付できます。
添付は `/attach`・ドラッグ&ドロップ・`Ctrl+V` 画像を合算して 1 ドラフト最大 12 件です。

### 追加方法

1. ドラッグ&ドロップ
   ファイルパスを入力欄へドロップすると自動添付されます。
2. コマンド
   - `/attach <path>`: 1件添付
   - `/detach <index>`: 指定番号を1件外す
   - `/detach-all`: すべて外す
3. `Ctrl+V`（Windows/WSL）
   クリップボードがテキストなら通常貼り付け、空なら画像貼り付けを試みます。

### 操作の補足

- 添付行には `#<n>` が表示され、`/detach` の番号指定に使えます。
- 入力カーソルが先頭で `Backspace` を押すと最後の添付を外します。
- 画像添付はマルチモーダル入力として送信され、ファイル添付は `Attached context` として本文へ展開されます。
- PDF 添付は `Attached context` へテキスト抽出して展開されます（先頭 20 ページ / 30000 文字まで）。
- PDF の読み取りに失敗した場合、または抽出可能テキストがない場合は、その旨を `Attached context` に明示します。
- クリップボード画像の一時ファイルは、送信・detach・終了時に削除されます。異常終了で残った古い一時ディレクトリは起動時に自動GCされます（24時間超）。

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
xelyon --image screenshot.png --provider kimi "この画面を説明して"

# 対話モード中にも使用可能（image:プレフィックス）
> image:screenshot.png このUIの問題点を教えて
```

**対応プロバイダー**: Kimi, Gemini, Claude, OpenAI, Azure OpenAI（DeepSeek, Ollama, Groqは非対応）

---

## プロジェクト guidance と repo 設定

標準の project guidance は repo 直下の `AGENTS.md` に書きます。`/init` は `AGENTS.md` を作成します。

`xelyon.yaml` は repo-local の XELYON 設定ファイルです。ignore pattern や final checks など、XELYON の実行設定を project ごとに変えたい場合に使います。

### 作成

```bash
> /init
```

### AGENTS.md について

`AGENTS.md` は agent が読む repo guidance です。既存の `CLAUDE.md` / `.claude/CLAUDE.md` を併用したい場合は `/config` の Agent Instructions で選択できます。

### xelyon.yaml について

xelyon.yaml は XELYON 用の repo 設定ファイルです。

**書くべきこと:**
- `ignore`: Project Map / `list_dir` / `search_code` で共有したい ignore パターン
- `final_checks`: 明示完了時の final checks（変更後に実行する検証コマンド）

**書かないこと:**
- agent guidance の本文（`AGENTS.md` に書く）
- ディレクトリ構造やコードマップの詳細
- 詳細なドキュメント

> 起動時は軽量な Project Map manifest が自動生成されるため、`xelyon.yaml` にファイル一覧や関数目次を書く必要はありません。

### xelyon.yaml の例

```yaml
# XELYON repo config

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

既存 `xelyon.yaml` の `context` / `rules` / `conditional` は互換のため読み込みますが、新規 project guidance は `AGENTS.md` を推奨します。

---

## サブエージェント委譲

XELYON は探索・調査タスクを `spawn_agent` / `wait_agent` でサブエージェントへ委譲できます。

- 親に返るのはサブの最終レポートだけで、中間の `read_file` / `search_code` 出力は親コンテキストへ残りません
- `wait_agent` 実行中はサブエージェントのツール実行が親UIへ逐次表示され、`str_replace` は色付き diff で確認できます
- `sub_agent.default_model` が空ならメイン provider の最安モデルを自動選択します。Azure OpenAI では `provider_models.azure.default_model` の deployment 名を使います。推論強度は off、同時実行数の既定は 1 です
- サブエージェント自身には `spawn_agent` / `wait_agent` を渡さないため、再帰的な spawn は行いません

設定例:

`sub_agent.max_concurrent` の既定は `1` です。複数の調査タスクを並列させたいときだけ、例のように値を上げてください。

```yaml
# ~/.xelyon/config.yaml
sub_agent:
  enabled: true
  default_model: ""
  default_effort: off
  max_concurrent: 2  # 既定: 1
```

サブエージェントは内部ツールなので、通常は自然言語の指示から自動で使われます。探索対象のファイルパスや観点を明示すると、より短い往復で結果が返ります。

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

## ワークフロー例

### 新機能実装

```bash
> ユーザー認証機能を追加して
# → 自然言語で指示するだけでAIが実装
# → ファイル編集前には差分を表示して確認
> /exit
```

### 日常的な開発

```bash
$ xelyon
> バグ #123 を修正して
> /exit
```

---

## キャッシュについて
