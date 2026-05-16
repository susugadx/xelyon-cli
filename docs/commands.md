# コマンド一覧

XELYON CLIで使用できる全コマンドのリファレンスです。

## CLI 診断コマンド

### `xelyon doctor kimi`

Kimi native provider の `MOONSHOT_API_KEY`、`KIMI_API_URL`、provider 登録、model config、未対応機能、`prompt_cache_key` request shape を確認します。`--smoke` を付けると live Chat Completions request を送信し、streaming、thinking on/off、同一 session の prompt cache key、usage callback を確認します。画像入力の実 API 受理を確認する場合は `--image-smoke` を使い、1x1 PNG を base64 image request として送信します。function calling まで確認する場合は `--tool-smoke`、built-in `$web_search` まで確認する場合は `--web-search-smoke` を使います。web search smoke は `web_search_call_count`、`web_search_call_fee_estimate`、`web_search_usage_observed`、`cached_input_tokens`、検索結果 token 観測値を text / JSON に出します。

`--smoke` / `--image-smoke` / `--tool-smoke` / `--web-search-smoke` は live API request を送るため、通常 CI では使いません。`cached_tokens` は Moonshot API が返した場合だけ観測され、0 でも smoke は成功扱いです。`--web-search-smoke` は実検索 call fee が発生し、`$web_search` tool call が 1 件以上観測された場合だけ成功扱いになります。通常の `stop` response で tool call がない場合、request 自体が返っていても smoke は fail します。call fee は token cost とは別料金で、検索結果 tokens は次 request の `prompt_tokens` に含まれるため二重加算しません。endpoint の token usage が返らない場合は `usage` check が warn になり、web search の fee / call count 観測だけでは token usage 観測済みとは扱いません。

```bash
xelyon doctor kimi
xelyon doctor kimi --model kimi-k2.6
xelyon doctor kimi --smoke
xelyon doctor kimi --image-smoke
xelyon doctor kimi --tool-smoke
xelyon doctor kimi --web-search-smoke
xelyon doctor kimi --json
```

手元で実 Kimi 環境の回帰確認を走らせる場合は、`MOONSHOT_API_KEY` を設定して `make kimi-smoke` を実行します。画像入力だけ確認する場合は `make kimi-image-smoke`、tool calling も含める場合は `make kimi-tool-smoke`、built-in web search は `make kimi-web-search-smoke` を使います。

### `xelyon doctor openai`

OpenAI provider の `OPENAI_API_KEY`、`OPENAI_API_URL`、`OPENAI_RESPONSES_URL`、provider 登録、model / `catalog_model` 解決、Responses / Chat Completions route と判定理由、function calling 設定、token / pricing metadata、Responses retention 設定を確認します。`--capabilities` を付けると live request を送らず、Responses API / streaming / function calling / image input / `previous_response_id` / server compaction / context window / max output / pricing の解決結果を `capabilities` に表示します。`--smoke` を付けると live text request を送信し、Responses route では response ID、usage、概算 cost を表示します。Chat Completions route では response ID は返らないため、usage と概算 cost を確認します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。Responses API の `previous_response_id` chain まで確認する場合は `--retention-smoke` を使い、`responses.store=true` の initial / followup request を連続実行します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。複数 smoke を指定した場合は request を別々に実行または preview し、JSON では `smoke.requests[]` / `request_preview.requests[]` に request 単位の結果を出します。

`OPENAI_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、tool payload / `tool_choice` は送信しません。`--retention-smoke` は Responses API route 専用で、Chat Completions route では live request を送らず fail します。`--model` は実 request に送るモデル名または alias、`--catalog-model` は alias の underlying OpenAI model として token / pricing / route 判定に使います。`--smoke` / `--tool-smoke` / `--retention-smoke` は live API request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor openai
xelyon doctor openai --model gpt-5.4
xelyon doctor openai --model corp-openai-deployment --catalog-model gpt-5.4
xelyon doctor openai --smoke
xelyon doctor openai --tool-smoke
xelyon doctor openai --retention-smoke
xelyon doctor openai --capabilities
xelyon doctor openai --print-request
xelyon doctor openai --tool-smoke --print-request
xelyon doctor openai --retention-smoke --print-request
xelyon doctor openai --smoke --tool-smoke
xelyon doctor openai --smoke --tool-smoke --retention-smoke
xelyon doctor openai --json
```

手元で doctor 経路だけを実 OpenAI 環境で確認する場合は、`OPENAI_API_KEY` を設定して `make openai-doctor-smoke` を実行します。既定では `gpt-5.4` で text / tool / retention smoke をまとめて実行し、必要なら `OPENAI_DOCTOR_SMOKE_MODEL` で変更できます。

### `xelyon doctor azure`

Azure OpenAI の base URL、認証、Entra ID token command、deployment 解決、`catalog_model`、Responses route と判定理由、function calling 設定、Responses retention 設定を確認します。`--capabilities` を付けると live request を送らず、Responses API / streaming / function calling / image input / `previous_response_id` / server compaction / context window / max output / pricing の解決結果を `capabilities` に表示します。`--smoke` を付けると、設定済み deployment に `responses.store=false` の最小 Responses API リクエストを送信し、response ID、usage、概算 cost も表示します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。Responses API の `previous_response_id` chain まで確認する場合は `--retention-smoke` を使い、`responses.store=true` の initial / followup request を連続実行します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。

`--deployment` は Azure 側の deployment 名、`--catalog-model` はその deployment の実モデル名です。`--print-config` を付けると、この 2 つから `~/.xelyon/config.yaml` に貼れる YAML 断片だけを出力します。`--smoke` / `--tool-smoke` / `--retention-smoke` は live API request を送るため、設定確認だけなら付けないでください。response ID や usage が返らない場合、または pricing catalog に該当モデルがない場合は warn になりますが、smoke 成功自体は失敗にしません。

```bash
xelyon doctor azure
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --print-config
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --smoke
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --tool-smoke
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --retention-smoke
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --capabilities
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --print-request
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --retention-smoke --print-request
xelyon doctor azure --json
```

手元で実 Azure 環境の回帰確認をまとめて走らせる場合は、`AZURE_OPENAI_BASE_URL`、`AZURE_OPENAI_DEPLOYMENT`、`AZURE_OPENAI_CATALOG_MODEL`、認証情報を設定して `make azure-smoke` を実行します。

Azure API error では 401/403/404/429 と tool payload rejected の原因候補を補足します。

### `xelyon doctor bedrock`

AWS Bedrock の region、AWS 認証チェーン、provider 登録、model / `catalog_model` 解決、Claude Messages / ConverseStream route、function calling 設定、token / pricing metadata を確認します。`--smoke` は text smoke、`--tool-smoke` は dummy tool call、`--image-smoke` は tiny PNG 画像入力、`--thinking-smoke` は Extended Thinking request を明示実行します。

Smoke の JSON では AWS SDK `ResultMetadata` 由来の `request_id`、request 単位の usage、概算 cost を `smoke.requests[]` に出します。summary usage / cost は request 単位の観測値を合算します。request ID、usage、pricing が返らない場合は warn ですが、API smoke 自体が成功していれば fail にはしません。複数 request のうち 1 件でも usage が返らない場合、summary cost は部分値を確定値として表示せず usage unavailable とします。Bedrock では Azure の `response_id` alias は出しません。`BEDROCK_FUNCTION_CALLING=0` の場合、`--tool-smoke` は function calling 無効として warn skip します。ConverseStream route で `--image-smoke` / `--thinking-smoke` を指定した場合は、未対応 request shape として warn skip します。

```bash
xelyon doctor bedrock
xelyon doctor bedrock --model global.anthropic.claude-sonnet-4-6
xelyon doctor bedrock --model corp-bedrock-sonnet --catalog-model global.anthropic.claude-sonnet-4-6
xelyon doctor bedrock --smoke
xelyon doctor bedrock --tool-smoke
xelyon doctor bedrock --image-smoke
xelyon doctor bedrock --thinking-smoke
xelyon doctor bedrock --json
```

手元で doctor 経路だけを実 AWS 環境で確認する場合は、AWS 認証チェーンを設定して `make bedrock-doctor-smoke` を実行します。runtime supported モデル全体の継続確認には `make bedrock-smoke` / `make bedrock-smoke-matrix` を使います。

## 対話型コマンド

セッション中に `/` で始まるコマンドを入力できます。
TUI が primary surface です。`--no-tui` の classic REPL は deprecated legacy fallback として残しており、新しい対話型 UI コマンドは TUI 側だけに追加します。
TUI では入力欄で `/` または `/r` のような prefix を入力すると command 候補が表示され、Enter で選択中の command を実行し、Tab で入力欄へ補完できます。

設定系コマンドの責務は分かれています。`/config` は global config (`~/.xelyon/config.yaml`) の編集、`/project` は project config (`xelyon.yaml`) の編集、`/init` は `xelyon.yaml` テンプレート作成だけを担当します。TUI で `xelyon.yaml` を管理する通常導線は `/project` です。

### `/help`

利用可能なコマンド一覧を表示します。TUI では `/` 候補が primary の command discovery です。

```
> /help
```

### `/status`, `/stats`

現在状態、直近リクエスト、セッション統計をまとめて表示します。`/stats` は互換エイリアスです。
表示には現在の command surface も含まれます。TUI では `TUI primary`、`--no-tui` では `classic legacy fallback` として表示し、classic 側では新しい対話型 UI コマンドを追加しない方針を明示します。
サブエージェントを使ったセッションでは、親とサブのコストを分離表示し、`🤖 Sub-agents` セクションで各サブのトークン使用量・コスト・ツール実行回数も確認できます。
Kimi `$web_search` を使った直近リクエストでは、`WebSearchCalls > 0` の場合だけ、token usage とは別に Web Search Calls、Web Search Fee、観測できた Search Result Tokens が表示されます。

```
> /status
> /stats   # alias
```

### `/save`

現在のセッションを保存します。

```
> /save
```

### `/load`

保存されたセッションを読み込みます。引数なしで最後のセッションを読み込みます。

```
> /load              # 最後のセッションを読み込み
> /load <session-id> # 指定したセッションを読み込み
```

### `/sessions`

保存されたセッション一覧を表示します（最新10件）。

```
> /sessions
```

### `/clear`

会話履歴をクリアします。

```
> /clear
```

### `/history`

現在の会話履歴を表示します。

```
> /history
```

### `/model`

現在のモデルを表示、または切り替えます。

```
> /model              # 現在のモデルを表示
> /model <model-name> # モデルを切り替え
```

例:
```
> /model gpt-4o
> /model gpt-5.5
> /model gpt-5.5-pro
> /model gpt-5.3-codex
> /model claude-sonnet-4-5-20250514
> /model gemini-2.0-flash-exp
```

`gpt-5.5` は OpenAI Responses API の streaming 経路、`gpt-5.5-pro` は streaming unsupported のため non-streaming 経路を使用します。GPT-5.5 Pro は応答に数分かかる場合があります。

### `/copy`

最後のAI出力をクリップボードにコピーします。

```
> /copy
```

### `/attach`

TUI で現在の入力ドラフトにファイルまたは画像を 1 件添付します。パスに空白がある場合は引用符で囲みます。添付は `/attach`・ドラッグ&ドロップ・`Ctrl+V` 画像を合算して 1 ドラフト最大 12 件です。

```
> /attach ./notes.txt
> /attach "screenshots/error shot.png"
```

### `/detach`

添付を番号指定で 1 件外します。番号は入力欄上部の添付行に表示される `#<n>` です。

```
> /detach 2
> /detach #2
```

### `/detach-all`

現在の入力ドラフトにある添付をすべて外します。

```
> /detach-all
```

### `/review`

TUIモードで、現在の変更レビュー用の preset 画面を開きます。
引数を付けた場合は、そのテキストを追加指示として現在の変更レビューを即時実行します。

```
> /review
> /review focus on regressions
```

### `/tokens`

トークン使用量とコンテキストウィンドウの状態を表示します。

```
> /tokens
```

**出力例:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Token Usage / トークン使用量
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [████████░░░░░░░░░░░░░░░░░░░░░░] 25.3%

  Current: 50,600 / 200,000 tokens

📋 Breakdown:
    System Prompt: 8,500 tokens (4.3%)
    History:       42,100 tokens (21.1%)  [38 messages]

🤖 Model:
    claude-sonnet-4-20250514 (context: 200,000 tokens)

⚙️  Auto-compress:
    ON (threshold: 80%)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**表示内容:**
- 使用率バー（色分け: 緑 < 80% < 黄 < 90% < 赤）
- 現在のトークン数 / モデルの上限
- 内訳（System Prompt、会話履歴）
- 使用モデルとコンテキストサイズ
- 自動圧縮の状態

### `/compress`

会話履歴を圧縮してトークン数を削減します。

```
> /compress           # LLMサマリーで圧縮（最新10件保持）
> /compress 20        # 最新20件を保持して圧縮
> /compress --compact # OpenAI Compact API で圧縮
> /compress -c        # 同上（短縮形）
```

**動作（デフォルト）:**
1. 古いメッセージをAIで要約
2. 要約を先頭に配置
3. 最新N件のメッセージを保持（デフォルト: 10件）

**Compact API モード (`--compact` / `-c`):**
- OpenAI Responses API の `/responses/compact` エンドポイントを使用
- ユーザーメッセージはそのまま保持（verbatim）
- アシスタント応答は暗号化された圧縮データに置換
- **制限**: OpenAI プロバイダーかつ Responses API 対応モデルのみ

**自動圧縮:** デフォルトで有効（Context 100K または 80% 到達時に自動実行）
- `prefer_compact_api: true` の場合、Compact API を優先使用
- `model: ""` の場合、圧縮時はプロバイダー別の低コストモデルを自動選択

### `/provider <provider> [model]`, `/use <provider> [model]`

プロバイダーとモデルを動的に切り替えます。TUI で引数なしの `/provider` を実行すると provider picker を開きます。`/use` は legacy alias です。

```
> /provider
> /provider deepseek
> /provider openai gpt-5.4
> /provider kimi kimi-k2.6
> /provider gemini gemini-2.0-flash-exp
> /provider claude claude-sonnet-4-5-20250514
> /provider ollama qwen2.5-coder:7b
> /provider groq meta-llama/llama-4-scout-17b-16e-instruct
> /use openai gpt-5.4   # legacy alias
```

### `/providers`

利用可能なプロバイダーとモデル一覧を表示します。

```
> /providers
```

### `/config`

global config (`~/.xelyon/config.yaml`) を確認・変更します。TUI の対話式メニューで設定項目をカテゴリ別に管理できます。project 固有の `xelyon.yaml` は `/project` で編集します。

```
> /config               # 対話式設定メニューを起動
> /config show          # 全設定を表示（デフォルトとの差分を ⚡ で表示）
> /config model <name>  # デフォルトモデルを変更
```

**対話式メニュー:**

`/config` を引数なしで実行すると、以下のような対話式メニューが表示されます：

```
┌─ Configuration ───────────────────────────┐
│ [1] 🤖 Provider & Model                   │
│ [2] 🛡️ Execution Mode                     │
│ [3] 📦 Compression                        │
│ [4] 🗺️ Project Map                        │
│ [5] 🔧 LSP Servers                        │
│ [6] 🔍 Web Search                         │
│ [7] 🧪 Final Checks                       │
│                                           │
│ [q] Cancel                                │
└───────────────────────────────────────────┘
```

**サポートする設定型:**

| 型 | 編集方法 | 例 |
|----|----------|-----|
| bool | y/n で切り替え | `thinking.enabled` |
| int | 数値入力 | `api_retry.count` |
| string | テキスト入力 | `default_model` |
| select | 番号選択 | `default_provider` (deepseek/claude/openai/...) |
| []string | 項目追加/削除 | `execution.safe_shell_commands` |
| map[string]struct | サブメニューで編集 | `provider_models`, `lsp.servers` |

**主なカテゴリ:** Provider & Model, Execution Mode, Compression, Paste Mode, Project Map, Agent Instructions, LSP Servers, Output, Web Search, Sub-agent, MCP Servers, Final Checks など

**変更は即座に保存:** `~/.xelyon/config.yaml` に自動保存されます。

### `/skills`

Agent Skills のカタログを一覧・確認・診断します。`/skills list` は検出済み skill 名を一覧表示し、`/skills show <name>` は対象の `SKILL.md` 本文と resource 一覧を表示します。`/skills doctor` は parse error や重複名などの診断を表示します。

```
> /skills list
> /skills show imagegen
> /skills doctor
```

### `/init`

project config (`xelyon.yaml`) のテンプレートを作成します。既存ファイルの編集は `/project` が通常導線です。

```
> /init
```

**テンプレートに含まれるフィールド:**
- `context` — プロジェクトの概要・背景情報
- `rules` — 必須ルール（AI が必ず従うルール）
- `conditional` — `paths` に一致した時だけ注入する rules/context
- `ignore` — Project Map / `list_dir` / `search_code` で共有する ignore パターン
- `final_checks` — 明示完了時の final checks（省略時は config.yaml の final_checks を使用）

**注意:**
- コード構造の詳細な記載は不要
- Project Map は起動時に軽量 manifest を自動注入するため、ファイル一覧や関数目次は書かない
- `final_checks.commands` を定義すると、AIが `completed_with_changes` の完了候補で必ず実行します

### `/project`

TUI で project config (`xelyon.yaml`) の編集画面を開きます。global config (`~/.xelyon/config.yaml`) は `/config` で編集します。

```
> /project
```

**TUI で編集できる項目:**
- `context`
- `rules`
- `ignore.patterns`
- `final_checks.commands`
- `final_checks.timeout`（`final_checks.commands` がある場合）

`conditional` は現時点では preview のみです。`xelyon.yaml` がない場合は、画面内でテンプレートを作成できます。

## TUI 添付の補足

- ドラッグ&ドロップでファイルパスを貼り付けると自動添付されます（最大 12 件）。
- `Ctrl+V` は通常テキスト貼り付けですが、クリップボードテキストが空の場合は画像貼り付けを試みます（Windows/WSL）。
- 送信時、画像はマルチモーダル入力として送信し、ファイルは `Attached context` として本文へ展開されます。
- PDF 添付は `Attached context` へテキスト抽出して展開されます（先頭 20 ページ / 30000 文字まで）。
- PDF の読み取りに失敗した場合、または抽出可能テキストがない場合は、その旨を `Attached context` に明示します。
- クリップボード画像の一時ファイルは、送信・detach・終了時に削除されます。異常終了で残った古い一時ディレクトリは起動時に自動GCされます（24時間超）。

### `/plan`

Plan Modeを切り替えます。有効にすると、リクエストが「調査→計画→承認（planning only）」のフローで処理されます。

```
> /plan           # 現在のモード表示
> /plan on        # Plan Mode 有効化
> /plan off       # 通常モードに戻る
> /plan toggle    # Plan Mode を切り替え
> /plan status    # ステータス表示
```

**デフォルト:** OFF（通常モード）

**通常モード（OFF）:**
- `execution.mode` に従ってツールを自動実行または確認しながら実行
- 軽いタスクにはオーバーヘッドなく即座に応答

**Plan Mode（ON）:**
1. **調査フェーズ**: SafetyHighツール（read_file, list_dir, search_code等）を自由に実行
2. **計画生成**: 実装が必要な場合、ステップをJSONで出力
3. **承認**: ユーザーが計画を確認・承認して Plan Mode を終了

`/plan` 自体は実装を開始しません。実装は次の通常モードのターンで行います。

**ステータス表示:**
```
[Status] waiting_input | Mode: Normal | Ready / 入力待ち
[Status] waiting_input | Mode: 📋 Plan | Ready / 入力待ち
```

### `/thinking`, `/think`

Extended Thinking（推論モード）を切り替えます。複雑なタスクでより深い推論を行う際に使用します。
`/think` は互換 alias です。

```
> /thinking              # 現在の状態表示
> /thinking on           # 有効化（現在のレベルで）
> /thinking off          # 無効化
> /thinking low          # 低レベルで有効化
> /thinking medium       # 中レベルで有効化（デフォルト）
> /thinking high         # 高レベルで有効化
> /thinking xhigh        # 最高レベルで有効化（max）
```

**対応プロバイダー:**

| プロバイダー | 対応 | 動作 |
|-------------|------|------|
| Claude | ✅ | Opus 4.7 / Opus 4.6 / Sonnet 4.6: adaptive thinking + effort / 4.5以前: budget_tokens |
| OpenAI | ✅ | reasoning_effort パラメータ |
| Gemini | ✅ | thinkingConfig.thinkingBudget |
| DeepSeek | ✅ | V4 `thinking` field + `reasoning_effort` |
| Groq | ❌ | 警告表示（非対応） |
| Ollama | ⚠️ | モデル依存（R1/QwQ推奨） |

**対応モデル:**
- **Claude**: Sonnet 4 以降（Opus 4.7 / Opus 4.6 / Sonnet 4.6 は adaptive thinking。Opus 4.7 の `/thinking xhigh` は `xhigh` effort）
- **OpenAI**: gpt-5.2 系
- **Gemini**: 2.5 Pro 系（Flash は非対応）
- **DeepSeek**: モデル名は維持し、`/thinking off` は `thinking.disabled`、`/thinking on` は `thinking.enabled` + `reasoning_effort` を送ります。`/thinking xhigh` は DeepSeek では `max` に変換されます。`reasoning_content` は💭で表示し、ツール実行フローでも保持します。

**注意**: Extended Thinking はトークン消費量が増加します。

### `/lsp`

LSPサーバーのステータスを表示・管理します。

`/lsp` は legacy classic (`--no-tui`) 用の診断コマンドです。TUI では候補と `/help` には表示せず、LSP 設定は `/config` から編集します。

```
xelyon --no-tui
> /lsp              # ステータス表示
> /lsp status       # 同上
> /lsp detect       # プロジェクト内の言語を検出
> /lsp install go   # 指定言語のLSPサーバーをインストール
> /lsp install all  # 未インストールの全サーバーをインストール
```

**出力例:**
```
LSP Server Status / LSPサーバー状態
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✅ Go: gopls (running)
  ✅ TypeScript: vtsls (running)
  ⏸️ Python: pyright (installed, idle)
  ❌ Rust: rust-analyzer (not installed)
```

**サブコマンド:**

| コマンド | 説明 |
|---------|------|
| `/lsp` または `/lsp status` | 現在のLSPサーバー状態を表示 |
| `/lsp detect` | プロジェクト内の言語を検出して表示 |
| `/lsp install <言語>` | 指定言語のLSPサーバーをインストール |
| `/lsp install all` | 未インストールの全サーバーをインストール |

**対応言語:** Go, TypeScript/JavaScript, Python, Rust, Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua, CSS/SCSS, HTML, Vue, Svelte, YAML, TOML, SQL, Bash, Markdown（23言語）

詳細は [LSP連携ガイド](lsp.md) を参照してください。

### `/version`

XELYONのバージョン情報を表示します。

```
> /version
```

**出力例:**
```
XELYON CLI v0.28.3
```

### `/exit`, `/quit`, `/q`

セッションを終了します。`Ctrl+D`、`Ctrl+C`、または `exit` でも終了できます。

```
> /exit
> /quit  # エイリアス
> /q     # 短縮形
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

### 画像入力（マルチモーダル）

```bash
# 画像ファイルを指定して起動
xelyon -i wireframe.png "この画面をReactで実装して"
xelyon --image error.png "このエラーを修正して"

# プロバイダー指定と組み合わせ
xelyon --image screenshot.png --provider gemini "このUIの問題点を教えて"
xelyon --image receipt.png --provider kimi "この画像の内容を要約して"
```

**対応フォーマット**: PNG, JPEG, GIF, WebP
**対応プロバイダー**: Kimi, Gemini, Claude, OpenAI, Azure OpenAI（DeepSeek, Ollama, Groqは非対応）
**制限**: `--image` は `--headless` / `--output-format json` と併用できません。`--resume` とも併用できません。

### その他のオプション

```bash
# バージョン表示
xelyon --version

# ヘルプ表示
xelyon --help

# 1ターンだけ実行して終了（位置引数だけでも one-shot になります）
xelyon --once "main.goを読んで説明して"
xelyon "main.goを読んで説明して"

# query 引数があっても対話 TUI を強制
xelyon --interactive "この続きから相談したい"

# one-shot のヘッダー/ステータス表示を抑制
xelyon --quiet "短く要約して"

# セッション再開（最後のセッションのみ）
xelyon --resume

# ツール確認を自動承認
xelyon --auto-approve "テストを直して"

# Headlessモード（JSON出力、対話なし）
xelyon --headless "main.goを読んで説明して"
xelyon --output-format json "バグを修正して"

# バージョンチェックを無効化
xelyon --no-update-check

# legacy classic REPL を使う（deprecated）
xelyon --no-tui
```

`--resume` は最後のセッションを再開します。session ID は受け取らず、query 引数や `--image` とは併用できません。`--once` とも併用できません。

`--quiet` は one-shot 実行専用です。`--interactive` や通常の対話セッションでは使用できません。

## 環境変数

詳細は [config.md](config.md) を参照してください。

```bash
# API キー
export DEEPSEEK_API_KEY=sk-...
export OPENAI_API_KEY=sk-...
export GEMINI_API_KEY=...
export ANTHROPIC_API_KEY=sk-ant-...
export GROQ_API_KEY=gsk_...

# 対話的確認モード（ツール実行前に確認）
export XELYON_INTERACTIVE_CONFIRM=1
```

## 使用例

### 基本的な対話

```bash
xelyon
> main.goを読んで、バグがあれば修正して
```

### プロバイダー切り替え

```bash
xelyon
> /use openai gpt-4
> この問題を詳しく分析して
> /use deepseek
> 同じ問題をもう一度分析して
```

## 利用可能なツール

AIが自動で以下のツールを使用します。ユーザーが直接呼び出す必要はありません。

デフォルトでは編集系ツールとして `apply_patch` のみを公開します。
開発デバッグ用に `XELYON_EDIT_TOOL=str_replace xelyon` で起動すると、legacy `str_replace` / `write_file` / `delete_file` を再度公開できます。

### ファイル操作

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `apply_patch` | Codex 互換の差分パッチでファイルを作成・更新・削除する既定編集ツール。1回で複数ファイルを扱える | `patch` |
| `read_file` | ファイル内容を読み込む。Go ファイルは `symbol` で関数/型/メソッド単位の読み出しも可能 | `path`, `symbol`, `start_line`, `end_line`, `paths` |
| `write_file` | legacy edit mode 用。ファイルを新規作成・上書き | `path`, `content` |
| `str_replace` | legacy edit mode 用。文字列置換でファイル編集（old_str優先。old_str空+start_line/end_line指定で行レンジ置換も可。Go ファイルは書き込み前に AST 構文チェックを実施） | `path`, `old_str`, `new_str`, `start_line`, `end_line` |
| `delete_file` | legacy edit mode 用。ファイルを削除 | `path` |
| `list_dir` | ディレクトリ一覧取得 | `path` |

**Note**: ファイル操作（mkdir, cp, mv, diff等）は `bash` ツールで実行可能です。

`read_file` の `symbol` 例:

```json
{"tool":"read_file","args":{"path":"internal/agent/agent.go","symbol":"maybeAutoCompress"}}
```

複数シンボルを読む場合はカンマ区切りで指定します。

### Git操作

すべてのGit操作は `bash` ツールで実行します。

```bash
# 例
bash: git status
bash: git diff
bash: git add -A && git commit -m "message"
bash: git checkout -b feature-branch
```

### コード調査・検索

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `search_code` | コード検索。`mode=auto` を既定に symbol-aware / literal / regex を language-aware に routing し、複数パターン・結果分類にも対応。対応言語では symbol-like query から定義・caller・参照・テストを自動解決。`intent=impact` は Go、TypeScript `.ts`、対象を絞った TSX `.tsx`、JavaScript `.js` で構造化 impact を優先し、それ以外は関連 multi-pattern 検索へフォールバック | `pattern`, `intent`, `mode`, `path`, `file_filter` 等 |
| `web_search` | ネイティブWeb検索（`web_search.provider` で Kimi / OpenAI / Gemini / Claude を選択可能） | `query` |
**注意**: メインプロバイダーがネイティブ検索非対応（DeepSeek / Groq / Ollama / OpenRouter / Bedrock など）の場合は、`config.yaml` で `web_search.provider` を設定してください。メインプロバイダーが Kimi の場合は provider 指定なしで Moonshot built-in `$web_search` を使います。Kimi で `$web_search` が起動すると call fee が発生し、XELYON は token usage と別枠で観測します。詳細は[config.md - Web検索](config.md#web検索)を参照してください。

### 開発支援

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `bash` | シェルコマンド実行 | `command` |

テスト、フォーマット、Lint はすべて bash で実行:
```bash
bash: go test ./...
bash: go fmt ./...
bash: golangci-lint run
```

### LSP（言語サーバー）

LSPは診断（エラー検知）、削除時参照チェック、Plan依存分析で内部的に利用されます。
コード検索には `search_code` ツールを使用してください。
詳細は [LSP連携ガイド](lsp.md) を参照してください。

### 使用例

AIは自然言語の指示に基づいてツールを自動選択します。

```bash
> main.goを読んで
# → read_file が実行される

> バグを修正して
# → read_file / search_code → apply_patch が実行される

> 複数ファイルをまとめて編集して
# → apply_patch が1回で複数ファイルに適用される

> git statusを見せて
# → bash で git status が実行される

> テストを実行して
# → bash で go test が実行される

> TODOを探して
# → search_code が実行される
```

**apply_patch の補足**
- `*** Begin Patch` / `*** End Patch` の境界が必須です
- `*** Add File:` / `*** Update File:` / `*** Delete File:` を1つ以上含めます
- `@@` と前後3行のコンテキストで変更位置を特定します
- 1つの patch で複数ファイルの追加・更新・削除をまとめて実行できます
- パスは相対パスのみを使います

**legacy str_replace の補足**
- `old_str` が非空のときは従来どおり文字列置換を行い、`start_line`/`end_line` は無視されます
- 行レンジ置換は `old_str: ""` かつ `start_line` と `end_line` を両方指定した場合のみ有効です（1-indexed, inclusive）
- Go ファイルでは置換後の内容を AST パースし、構文エラーが見つかった場合は警告を表示して tool result にも付与します
- AST 警告が出ても `str_replace` 自体は停止せず、ユーザー確認済みまたは auto-approve の場合はそのまま書き込みます

### MCP対応ツール

XELYONはMCP（Model Context Protocol）に対応しており、`~/.xelyon/mcp.json`で外部ツールを追加できます。

詳細は [MCP連携ガイド](mcp.md) を参照してください。

## 高度な機能

### Headlessモード

対話なしでJSON形式で結果を出力します。他のツールやスクリプトから呼び出す際に便利です。
Kimi `$web_search` を使った場合、既存の `cost` は token cost + web search call fee の合計を維持し、`web_search` object に `calls`、`fee_estimate`、`result_tokens` を分けて出します。検索結果 tokens は次 request の `prompt_tokens` に含まれる前提の表示用観測値で、headless JSON の token totals には再加算しません。

```bash
# JSON出力
xelyon --headless "main.goを読んで概要を説明して"

# 出力例
{
  "query": "main.goを読んで概要を説明して",
  "response": "このファイルは...",
  "success": true
}

# jqと組み合わせて
xelyon --headless "バグを修正して" | jq -r '.response'

# CI/CDパイプラインで使用
xelyon --output-format json "テストを実行して" | jq '.success'
```

### 対話的確認モード

ツール実行前に毎回確認プロンプトを表示します（デバッグ・学習用）。

```bash
export XELYON_INTERACTIVE_CONFIRM=1
xelyon

> main.goを編集して

# 各ツール実行前に確認
# [y] 実行 / [n] スキップ / [c] フィードバック付き / [s] 終了
```

## 関連ドキュメント

- [プロバイダー設定](providers.md)
- [設定リファレンス](config.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)
