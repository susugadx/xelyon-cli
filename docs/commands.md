# コマンド一覧

XELYON CLIで使用できる全コマンドのリファレンスです。

## 対話型コマンド

セッション中に `/` で始まるコマンドを入力できます。

### `/help`

利用可能なコマンド一覧を表示します。

```
> /help
```

### `/stats`

現在のセッション統計を表示します（ユーザーメッセージ数、アシスタントメッセージ数、ツール実行回数など）。

```
> /stats
```

### `/memory`

ユーザーメモリ機能を管理します。

#### メモリ追加
```
> /memory add <内容>
```

例:
```
> /memory add 私はReactとTypeScriptを使った開発を好みます
```

#### メモリ一覧
```
> /memory list
```

#### メモリ削除
```
> /memory delete <ID>
```

#### メモリクリア
```
> /memory clear          # 全メモリ削除
> /memory clear --project # プロジェクト別メモリのみ削除
```

### `/plan`

Plan Modeのオン/オフを切り替えます。Plan Modeでは、AIが実装前に計画を立てて確認を求めます。

```
> /plan         # トグル（オン⇔オフ）
> /plan on      # 明示的にオン
> /plan off     # 明示的にオフ
```

### `/copy`

最後のAI出力をクリップボードにコピーします。

```
> /copy
```

### `/compress`

会話履歴を圧縮してトークン数を削減します。

```
> /compress
```

### `/use <provider> [model]`

プロバイダーとモデルを動的に切り替えます。

```
> /use deepseek
> /use openai gpt-4
> /use gemini gemini-2.0-flash-exp
> /use claude claude-sonnet-4-5-20250514
> /use ollama qwen2.5-coder:7b
> /use groq meta-llama/llama-4-scout-17b-16e-instruct
```

### `/providers`

利用可能なプロバイダーとモデル一覧を表示します。

```
> /providers
```

### `/changes`

セッション中のファイル変更履歴を表示します。

```
> /changes
```

### `/undo`

最後のファイル変更を取り消します（バックアップから復元）。

```
> /undo
```

### `/exit`

セッションを終了します。`Ctrl+D`、`Ctrl+C`、または `exit` でも終了できます。

```
> /exit
```

## CLIフラグ

起動時に指定できるオプションです。

### プロバイダー/モデル指定

```bash
# プロバイダー指定
xelyon --provider deepseek
xelyon --provider openai
xelyon --provider gemini
xelyon --provider claude
xelyon --provider ollama
xelyon --provider groq

# モデル指定
xelyon --provider openai --model gpt-4
xelyon -m gemini-2.0-flash-exp
```

### 初期プロンプト

```bash
xelyon "main.goを読んで説明して"
```

### その他のオプション

```bash
# バージョン表示
xelyon --version

# ヘルプ表示
xelyon --help

# セッション再開
xelyon --resume <session-id>

# プロジェクト別セッション
xelyon --project

# 圧縮せずにセッション開始（デバッグ用）
xelyon --no-compress

# ページャーを無効化
xelyon --no-pager
```

## 環境変数

詳細は [config.md](config.md) を参照してください。

```bash
# API キー
export DEEPSEEK_API_KEY=sk-...
export OPENAI_API_KEY=sk-...
export GEMINI_API_KEY=...
export ANTHROPIC_API_KEY=sk-ant-...
export GROQ_API_KEY=gsk_...

# デバッグモード
export XELYON_DEBUG=1

# 対話的確認モード（ツール実行前に確認）
export XELYON_INTERACTIVE_CONFIRM=1
```

## 使用例

### 基本的な対話

```bash
xelyon
> main.goを読んで、バグがあれば修正して
```

### Plan Modeで実装

```bash
xelyon
> /plan on
> ユーザー認証機能を追加して
```

### プロバイダー切り替え

```bash
xelyon
> /use openai gpt-4
> この問題を詳しく分析して
> /use deepseek
> 同じ問題をもう一度分析して
```

### メモリ活用

```bash
xelyon
> /memory add プロジェクトはNext.js 14 + TypeScript
> /memory add Tailwind CSSを使用
> 新しいコンポーネントを作って
```

### 変更管理

```bash
xelyon
> ファイルを編集して
> /changes      # 変更履歴確認
> /undo         # 最後の変更を取り消し
```
