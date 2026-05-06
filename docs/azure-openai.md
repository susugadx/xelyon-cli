# Azure OpenAI 利用メモ

会社環境で Azure OpenAI を使う人向けの最短手順です。詳しい provider 仕様は [providers.md](providers.md#3-azure-openai) を参照してください。

## まず決めること

Azure OpenAI では、XELYON の `model` には OpenAI のモデル名ではなく **Azure 側の deployment 名**を指定します。

- `default_model`: Azure OpenAI resource に作成した deployment 名
- `catalog_model`: deployment の実モデル名。例: `gpt-5.4`, `gpt-5.5`, `gpt-5.5-pro`, `gpt-5.3-codex`

deployment 名と実モデル名が同じとは限りません。会社側で `corp-gpt55-prod` のような deployment 名を付けている場合、`default_model` は `corp-gpt55-prod`、`catalog_model` は `gpt-5.5` のように分けます。
Codex 系 deployment でも同じで、たとえば `default_model` は `corp-codex-prod`、`catalog_model` は `gpt-5.3-codex` のように分けます。

## 推奨設定

会社環境では、固定 bearer token より `AZURE_OPENAI_AUTH_TOKEN_COMMAND` を推奨します。長時間の作業中に token が切れても、401 応答後に 1 回だけ token を再取得して retry できます。

```bash
export AZURE_OPENAI_BASE_URL=https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1
unset AZURE_OPENAI_API_KEY
unset AZURE_OPENAI_AUTH_TOKEN
export AZURE_OPENAI_AUTH_TOKEN_COMMAND='az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv'
export AZURE_OPENAI_AUTH_TOKEN_COMMAND_TIMEOUT=10s
```

必要なら先に Azure CLI でログインします。

```bash
az login
az account set --subscription YOUR-SUBSCRIPTION
```

tenant を明示する必要がある環境では、社内の案内に従って `az login --tenant ...` や token command 側の引数を調整してください。

## XELYON 設定

`xelyon doctor azure --print-config` で YAML 断片を作れます。

```bash
xelyon doctor azure --deployment corp-codex-prod --catalog-model gpt-5.3-codex --print-config
```

例:

```yaml
default_provider: azure

provider_models:
  azure:
    default_model: corp-codex-prod
    catalog_model: gpt-5.3-codex
```

`default_model` に `gpt-5.3-codex` や `gpt-5.5` のような実モデル名を入れるのは、その名前で Azure deployment を作っている場合だけです。

## 動作確認

設定だけ確認する場合:

```bash
xelyon doctor azure
```

実 deployment へ最小リクエストを送る場合:

```bash
xelyon doctor azure --deployment corp-gpt55-prod --catalog-model gpt-5.5 --smoke
```

tool payload まで確認する場合:

```bash
xelyon doctor azure --deployment corp-gpt55-prod --catalog-model gpt-5.5 --tool-smoke
```

`--smoke` / `--tool-smoke` は live API request を送ります。単なる設定確認では付けないでください。

## 認証方式の優先順位

XELYON は次の順で Azure 認証を使います。

1. `AZURE_OPENAI_API_KEY`
2. `AZURE_OPENAI_AUTH_TOKEN`
3. `AZURE_OPENAI_AUTH_TOKEN_COMMAND`

`AZURE_OPENAI_API_KEY` が残っていると API key 認証が優先されます。Entra ID token command を確認したい場合は `unset AZURE_OPENAI_API_KEY` してください。

`AZURE_OPENAI_AUTH_TOKEN_COMMAND` はローカル shell で実行され、stdout の最初の空でない行を bearer token として扱います。信頼できる command だけを設定してください。取得した token は process memory にだけ保持し、config や session には保存しません。

## よくある詰まり方

- 401: API key / bearer token / token command の誤り、token 期限切れ、別 resource の credential
- 403: Azure OpenAI への権限、Entra ID role assignment、subscription / resource access の不足
- 404: deployment 名の誤り、または `AZURE_OPENAI_BASE_URL` が別 resource / 間違った path
- 429: quota、rate limit、Azure capacity
- tool payload rejected: deployment が tool payload 非対応の可能性。必要なら `AZURE_OPENAI_FUNCTION_CALLING=0`

まず `xelyon doctor azure --json` を取得すると、問い合わせ時に状態を共有しやすくなります。

## 通常いじらない設定

`responses.store` / `responses.persist_response_id` は通常変更しないでください。既定値は Responses API の `previous_response_id` 継続と session reload を安定させるための推奨設定です。`responses.server_compaction` はこの chain request 向けの自動圧縮トリガー設定で、Compact API（`/responses/compact` / `/compress --compact`）とは別機能です。

会社の data retention 方針で provider 側に response state を残せない場合だけ、[Responses API retention 設定](config.md#responses-api-retention-設定-responses高度な設定) を確認してから変更してください。

## 問い合わせ前チェック

- `AZURE_OPENAI_BASE_URL` は `https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1` 形式か
- `AZURE_OPENAI_BASE_URL` に `/deployments/<name>` や `api-version` を含めていないか
- `provider_models.azure.default_model` は Azure deployment 名か
- `provider_models.azure.catalog_model` は実モデル名か
- Entra ID を使う場合、`AZURE_OPENAI_API_KEY` が残っていないか
- `AZURE_OPENAI_AUTH_TOKEN_COMMAND` を terminal で単体実行すると token が 1 行で出るか
- `xelyon doctor azure` の fail / warn を確認したか
