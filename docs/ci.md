# Headless CI guide

XELYON headless mode は、CI / script から stdout の JSON と process exit code を安定して扱うための automation surface です。CLI flags、JSON schema、exit code の詳細 contract は [commands.md の Headless モード](commands.md#headlessモード) を参照してください。

## 推奨 command

PR smoke や review-only CI では、prompt を file に置き、workspace mutation を禁止して実行します。

```bash
xelyon --headless --prompt-file prompt.md --exit-code-policy ci --fail-on-tool-error --read-only
```

provider / model はリポジトリや workflow の方針に合わせて、config または CLI flag で指定してください。

```bash
xelyon --provider openai --model gpt-5.3-codex --headless --prompt-file prompt.md --exit-code-policy ci --fail-on-tool-error --read-only
```

## GitHub Actions PR smoke

以下は same-repo pull request で headless read-only review を 1 回だけ実行する最小例です。repository secrets には利用する provider の credential を設定してください。provider setup の詳細は [providers.md](providers.md) を参照してください。

fork PR では `pull_request` workflow に repository secrets が渡らないため、この例は job-level guard で skip します。fork PR の AI review は、secrets と untrusted code の扱いを別途設計してください。

```yaml
name: XELYON Headless PR Smoke

on:
  pull_request:

permissions:
  contents: read

jobs:
  xelyon-headless:
    if: ${{ github.event.pull_request.head.repo.full_name == github.repository }}
    runs-on: ubuntu-latest
    timeout-minutes: 20
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install XELYON
        run: |
          curl -fsSL https://raw.githubusercontent.com/susugadx/xelyon-cli/main/install.sh | bash -s -- --yes
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"

      - name: Create prompt
        run: |
          BASE_SHA="${{ github.event.pull_request.base.sha }}"
          HEAD_SHA="${{ github.event.pull_request.head.sha }}"
          MERGE_BASE="$(git merge-base "${BASE_SHA}" "${HEAD_SHA}")"
          cat > prompt.md <<'PROMPT'
          Review this pull request in read-only mode.
          Focus on correctness regressions, failing test risk, and missing verification.
          Do not modify files.
          PROMPT
          {
            echo
            echo "Base SHA: ${BASE_SHA}"
            echo "Head SHA: ${HEAD_SHA}"
            echo "Merge base SHA: ${MERGE_BASE}"
            echo
            echo "Changed files:"
            git diff --name-status "${MERGE_BASE}" "${HEAD_SHA}"
            echo
            echo "Diff:"
            echo '```diff'
            git diff --find-renames "${MERGE_BASE}" "${HEAD_SHA}"
            echo '```'
          } >> prompt.md

      - name: Run XELYON headless
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          xelyon --provider openai --model gpt-5.3-codex --headless --prompt-file prompt.md --exit-code-policy ci --fail-on-tool-error --read-only > xelyon-result.json
          jq -e '.status == "success"' xelyon-result.json
```

この例では `prompt.md` に PR の base/head SHA、merge base SHA、merge-base から head までの changed-file list と diff を埋め込んでから XELYON を起動します。`--read-only` 実行では、この prompt 内の delta が review source of truth です。runtime 側の bash / MCP / sub-agent 経由の差分探索には依存できないため、PR review として使う workflow では実行前に delta を prompt へ渡してください。

`--exit-code-policy ci` では、`usage_error`、provider setup 不足、tool error、final check failure、API error、cancel、read-only violation、unsupported capability などが `failure_reason` と `recommended_exit_code` に分類されます。workflow step の失敗判定は process exit code に任せ、必要に応じて `xelyon-result.json` を artifact として保存してください。

## Provider setup

GitHub Actions では `OPENAI_API_KEY`、`GEMINI_API_KEY`、`ANTHROPIC_API_KEY`、`AZURE_OPENAI_BASE_URL` / `AZURE_OPENAI_API_KEY` など、利用 provider に対応した secret を設定します。provider ごとの環境変数、OAuth、local provider の制約は [providers.md](providers.md) が source of truth です。

`openai_subscription` provider は個人 dogfood / local CLI 向けで、CI、shared server、production automation には推奨しません。

## Nightly eval

Nightly eval runner や Batch backend はまだこの guide では提供しません。将来追加する場合も、この headless command と JSON contract の上に載せる想定です。
