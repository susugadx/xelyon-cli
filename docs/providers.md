# プロバイダー設定

XELYON CLIは複数のLLMプロバイダーに対応しています。

## 対応プロバイダー

| プロバイダー | 画像入力 | 環境変数 | 公式サイト |
|------------|---------|---------|-----------|
| DeepSeek | ❌ | `DEEPSEEK_API_KEY` | https://platform.deepseek.com |
| OpenAI | ✅ | `OPENAI_API_KEY` | https://platform.openai.com |
| Gemini | ✅ | `GEMINI_API_KEY` | https://ai.google.dev |
| Claude | ✅ | `ANTHROPIC_API_KEY` | https://console.anthropic.com |
| Groq | ❌ | `GROQ_API_KEY` | https://console.groq.com |
| Ollama | ❌ | - | https://ollama.com |

## セットアップ

### 1. DeepSeek

```bash
# API キー取得: https://platform.deepseek.com
export DEEPSEEK_API_KEY=sk-...

# 使用例
xelyon --provider deepseek --model deepseek-coder
xelyon --provider deepseek --model deepseek-chat
xelyon --provider deepseek --model deepseek-reasoner
```

**特徴:**
- 高速・低コスト
- コード生成に特化したモデルあり
- 画像入力非対応

### 2. OpenAI

```bash
# API キー取得: https://platform.openai.com/api-keys
export OPENAI_API_KEY=sk-...

# 使用例
xelyon --provider openai --model gpt-4
xelyon --provider openai --model gpt-4-turbo
xelyon --provider openai --model gpt-5.2
```

**特徴:**
- 高品質な回答
- GPT-4Vで画像入力対応
- 豊富なモデルラインナップ

### 3. Gemini

```bash
# API キー取得: https://aistudio.google.com/app/apikey
export GEMINI_API_KEY=...

# 使用例
xelyon --provider gemini --model gemini-2.0-flash-exp
xelyon --provider gemini --model gemini-2.5-flash
xelyon --provider gemini --model gemini-pro
```

**特徴:**
- 長いコンテキスト対応
- 画像入力対応
- 無料枠あり

### 4. Claude

```bash
# API キー取得: https://console.anthropic.com
export ANTHROPIC_API_KEY=sk-ant-...

# 使用例
xelyon --provider claude --model claude-sonnet-4-5-20250514
xelyon --provider claude --model claude-opus-4
```

**特徴:**
- 長文理解に優れる
- 倫理的な回答
- 画像入力対応

### 5. Groq

```bash
# API キー取得: https://console.groq.com/keys
export GROQ_API_KEY=gsk_...

# 使用例
xelyon --provider groq --model meta-llama/llama-4-scout-17b-16e-instruct
```

**特徴:**
- 超高速推論
- Llama系モデル
- 画像入力非対応

### 6. Ollama

```bash
# インストール: https://ollama.com/download
# サーバー起動
ollama serve

# モデルダウンロード
ollama pull qwen2.5-coder:7b

# 使用例
xelyon --provider ollama --model qwen2.5-coder:7b
xelyon --provider ollama --model llama3.1:8b
```

**特徴:**
- ローカル実行（APIキー不要）
- プライバシー保護
- 無料
- 画像入力非対応

## モデル指定方法

### 1. コマンドラインフラグ（最優先）

```bash
xelyon --provider openai --model gpt-4
```

### 2. 環境変数

```bash
export XELYON_PROVIDER=deepseek
export XELYON_MODEL=deepseek-coder
xelyon
```

### 3. 設定ファイル（`~/.xelyon/config.yaml`）

```yaml
default_provider: deepseek
default_model: deepseek-coder

provider_models:
  deepseek:
    default_model: deepseek-coder
  openai:
    default_model: gpt-4
  gemini:
    default_model: gemini-2.0-flash-exp
  claude:
    default_model: claude-sonnet-4-5-20250514
  ollama:
    default_model: qwen2.5-coder:7b
  groq:
    default_model: meta-llama/llama-4-scout-17b-16e-instruct
```

### 4. セッション中の切り替え（`/use`コマンド）

```bash
xelyon
> /use openai gpt-4
> 質問1
> /use deepseek
> 質問2
```

## プロバイダー選択のヒント

### コード生成・編集
- **DeepSeek Coder**: 高速・低コスト・高品質
- **Qwen2.5-Coder (Ollama)**: ローカル実行

### 複雑な問題解決
- **Claude Opus 4**: 長文理解・推論
- **GPT-4**: バランスの良い性能

### 高速レスポンス
- **Groq**: 超高速推論
- **Gemini Flash**: バランス良く高速

### 画像解析
- **GPT-4V**: 高品質な画像理解
- **Gemini**: マルチモーダル対応
- **Claude**: 画像+長文の組み合わせ

### プライバシー重視
- **Ollama**: 完全ローカル実行

## トラブルシューティング

### API キーエラー

```bash
# 環境変数が設定されているか確認
echo $DEEPSEEK_API_KEY

# .zshrc / .bashrc に追加
export DEEPSEEK_API_KEY=sk-...
source ~/.zshrc
```

### Ollama接続エラー

```bash
# サーバーが起動しているか確認
ollama list

# サーバー起動
ollama serve
```

### モデルが見つからない

```bash
# 利用可能なモデル一覧を確認
xelyon
> /providers

# 正しいモデル名を指定
xelyon --provider openai --model gpt-4
```

### レート制限エラー

APIプロバイダーのダッシュボードで使用状況とレート制限を確認してください。

- DeepSeek: https://platform.deepseek.com/usage
- OpenAI: https://platform.openai.com/usage
- Gemini: https://aistudio.google.com
- Claude: https://console.anthropic.com
- Groq: https://console.groq.com

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [設定リファレンス](config.md)
