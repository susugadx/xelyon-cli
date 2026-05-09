# Bedrock Provider 運用

## Runtime Support Policy

Bedrock provider は runtime support と pricing support を分けて扱います。

- Claude on Bedrock は Claude Messages route で実行します。
- Claude 以外の runtime supported モデルは `ConverseStream` 経路で実行します。
- Agent 実行では structured tool calling が必須です。
- `ConverseStream` 経路では、streaming tool use 対応を実 API smoke で確認したモデルだけを runtime supported にします。
- 料金表にあるモデルでも、streaming tool use が未確認または非対応なら runtime unsupported として扱います。
- Text response や non-streaming tool use が可能なだけでは runtime supported にしません。

## Live Smoke

Bedrock live smoke は AWS 認証チェーンを使って実 API を呼びます。

```bash
# Doctor 経路の設定診断
xelyon doctor bedrock
xelyon doctor bedrock --model global.anthropic.claude-sonnet-4-6 --smoke
xelyon doctor bedrock --model amazon.nova-pro-v1:0 --smoke --tool-smoke
xelyon doctor bedrock --json

# 単発 smoke
make bedrock-smoke

# Doctor 経路だけを実環境で確認
make bedrock-doctor-smoke

# Runtime supported モデルの matrix smoke
make bedrock-smoke-matrix

# Streaming tool-use unsupported/unverified モデルの probe
make bedrock-smoke-probe
```

既定の matrix は Claude route、Amazon Nova Pro、Moonshot Kimi K2.5 を確認します。

```bash
BEDROCK_SMOKE_CLAUDE_MODEL="global.anthropic.claude-sonnet-4-6" \
BEDROCK_SMOKE_CONVERSE_MODELS="amazon.nova-pro-v1:0 moonshotai.kimi-k2.5" \
make bedrock-smoke-matrix
```

`doctor bedrock` の live smoke は request 単位で AWS SDK `ResultMetadata` の `request_id`、usage、概算 cost を表示します。複数 smoke を指定した場合は `smoke.requests[]` に分け、summary usage / cost は request 単位の観測値を合算します。`request_id`、usage、pricing が返らない場合は warn ですが、API request が成功していれば smoke 成功自体は fail にしません。複数 request のうち 1 件でも usage が返らない場合、summary cost は部分値を確定値として表示せず usage unavailable とします。Bedrock では Azure の `response_id` alias は出しません。

`--smoke` は text smoke、`--tool-smoke` は dummy tool call、`--image-smoke` は tiny PNG 画像入力、`--thinking-smoke` は Extended Thinking request です。Claude route では指定された request を実 API に送ります。`BEDROCK_FUNCTION_CALLING=0` の場合、`--tool-smoke` は function calling 無効として warn skip します。ConverseStream route は text / tool smoke だけを実行し、image / thinking smoke は未対応 request shape として warn skip します。

Probe は候補モデルが text streaming できることと、xelyon runtime では streaming tool-use unsupported として早期 reject されることを確認します。probe 成功だけでは supported に昇格しません。

```bash
BEDROCK_PROBE_CONVERSE_MODELS="deepseek.v3.2" \
make bedrock-smoke-probe
```

候補モデルを追加する場合は、AWS の `list-foundation-models` で Bedrock model ID を確認してから `BEDROCK_PROBE_CONVERSE_MODELS` に追加します。

## Supported モデル追加手順

1. AWS 公式価格に基づいて [internal/cost/pricing.yaml](../internal/cost/pricing.yaml) の Bedrock pricing / `known_models.exact` を更新します。
2. Converse モデルの場合は [internal/llmcatalog/bedrock.go](../internal/llmcatalog/bedrock.go) の最大出力トークン上限を追加します。未知上限のまま runtime supported にしないでください。
3. 実 API で `ConverseStream` の text + tool use が通った Converse モデルだけ、`BedrockConverseToolUseSupported` の allowlist に追加します。
4. `BEDROCK_SMOKE_CONVERSE_MODELS="amazon.nova-pro-v1:0 moonshotai.kimi-k2.5 <model>" make bedrock-smoke-matrix` を実行し、Claude route と supported Converse matrix の live smoke を通します。
5. [docs/providers.md](providers.md) の利用可能モデル表を更新します。
6. streaming tool use が未確認または非対応の候補を継続確認したい場合は、`BEDROCK_PROBE_CONVERSE_MODELS` の例を更新します。
