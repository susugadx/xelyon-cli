# MCP (Model Context Protocol) 連携

XELYON CLIは[Model Context Protocol (MCP)](https://modelcontextprotocol.io)に対応しており、外部ツールを動的に追加できます。

## MCPとは

Model Context Protocol（MCP）は、AIアシスタントが外部のツールやデータソースに安全にアクセスするための標準プロトコルです。

### 主な特徴

- **プラグイン方式**: 外部プロセス（MCPサーバー）としてツールを実行
- **stdioトランスポート**: 標準入出力でJSON-RPC 2.0通信
- **動的登録**: 起動時にMCPサーバーからツール一覧を自動取得
- **セキュリティ**: 実行コマンドを allowlist で制限し、system env は安全なキーだけ継承

## 設定

`config.yaml` で MCP の動作を制御できます:

```yaml
# ~/.xelyon/config.yaml
mcp:
  enabled: true    # MCP機能のON/OFF（デフォルト: true）
  headless: false  # ヘッドレスモードでMCPを使うか（デフォルト: false）
  surface_budget:
    max_tools: 80
    estimated_tokens: 32000
    max_schema_bytes_per_tool: 131072
```

| 設定 | 説明 | デフォルト |
|------|------|-----------|
| `enabled` | `false` にするとMCPサーバーへの接続をスキップ。トークン消費を削減。 | `true` |
| `headless` | `true` にすると `--headless` モードでもMCPツールが使える。 | `false` |
| `surface_budget.max_tools` | provider に公開する MCP tool 数の上限。 | `80` |
| `surface_budget.estimated_tokens` | provider tool definitions 相当の推定 token 上限。 | `32000` |
| `surface_budget.max_schema_bytes_per_tool` | 1 tool あたりの input schema byte 上限。 | `131072` |

`enabled: false` にしても `~/.xelyon/mcp.json` の設定はそのまま残るため、再度 `enabled: true` にすれば復活します。
`headless: true` はMCPツールの公開を許可するだけで、承認を自動化しません。headless でMCPツールを実行するには、対象サーバーまたはツールに `approval: "auto"` を明示してください。

## セットアップ

### 1. 設定ファイルの作成

`~/.xelyon/mcp.json` にMCPサーバーを定義します。
初回起動時にファイルがない場合は、無効化された filesystem のサンプル設定が作成されます。
使う場合は `disabled: true` を削除または `false` に変更し、対象ディレクトリを実パスに置き換えてください。

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": [
        "@modelcontextprotocol/server-filesystem",
        "/path/to/your/project"
      ],
      "env": {
        "NODE_OPTIONS": "--no-warnings"
      },
      "approval": "confirm"
    },
    "puppeteer": {
      "command": "npx",
      "args": [
        "@modelcontextprotocol/server-puppeteer"
      ],
      "approval": "confirm",
      "startupTimeoutSeconds": 120,
      "toolTimeoutSeconds": 600
    },
    "sequential-thinking": {
      "command": "npx",
      "args": [
        "@modelcontextprotocol/server-sequential-thinking"
      ]
    }
  }
}
```

### 2. MCPサーバーのインストール

```bash
# Filesystem MCP Server（ファイル操作）
npm install -g @modelcontextprotocol/server-filesystem

# Puppeteer MCP Server（Web自動化）
npm install -g @modelcontextprotocol/server-puppeteer

# Sequential Thinking MCP Server（推論補助）
npm install -g @modelcontextprotocol/server-sequential-thinking
```

### 3. 動作確認

```bash
xelyon
> どんなツールが使えるか教えて

# MCPツールが自動的に読み込まれて、AIに提示されます
```

設定だけを確認する場合は `doctor mcp` を使います。

```bash
# local-only。mcp.json を新規作成せず、MCP server process も起動しない
xelyon doctor mcp

# initialize / tools/list まで確認する。tools/call は実行しない
xelyon doctor mcp --connect

# tool 名、exported name、visible / skipped reason、approval も表示
xelyon doctor mcp --connect --tools
```

`doctor mcp --connect` は live `tools/list` の結果から runtime 全体の tool surface summary、effective budget、server 別の total / registered / visible / omitted 数、schema bytes、estimated tokens、omitted reason、絞り込み提案も表示します。
estimated tokens は analysis に含めた tool の provider tool definition 相当から推定した合計です。schema body は表示しません。
`doctor mcp` は env value と raw args を表示しません。表示するのは command、arg 数、env key 名、timeout、approval、tool 名、集計済みの token / schema byte 数です。schema body や description 全文は表示しません。

対話中に現在セッションへ読み込まれている MCP runtime 状態を確認する場合は `/mcp status` を使います。
`/mcp status` は snapshot-only で、MCP server process の起動、再接続、`tools/list`、`tools/call` は行いません。
表示するのは runtime 有効状態、読み込み済み config の有無、server 状態、registered / visible / omitted tool 数、effective budget、tool surface のサンプル、server 別 token / schema byte 数、omitted reason、絞り込み提案です。
env value、raw args、server error detail、tool schema body、description 全文は表示しません。

```text
> /mcp status
```

## 設定項目

### `command` (必須)

実行するコマンド。安全のため、現在は `npx`, `node`, `python`, `python3`, `uvx`, `docker` のみ許可されます。

```json
{
  "mcpServers": {
    "my-server": {
      "command": "npx"
    }
  }
}
```

### `args` (オプション)

コマンドに渡す引数の配列。

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": [
        "@modelcontextprotocol/server-filesystem",
        "/home/user/projects"
      ]
    }
  }
}
```

### `env` (オプション)

MCPサーバーに明示的に渡す環境変数。
`env` に書いた値は `${VAR}` 形式の展開後に渡されます。
system env から自動継承されるのは `PATH`, `HOME`, `USER`, `LANG`, `LC_ALL`, `NODE_OPTIONS`, `PYTHONPATH` のみです。

```json
{
  "mcpServers": {
    "my-server": {
      "command": "node",
      "args": ["server.js"],
      "env": {
        "DEBUG": "true",
        "PORT": "8080"
      }
    }
  }
}
```

### `startupTimeoutSeconds` (オプション)

MCPサーバーの起動、接続、`tools/list` に使う timeout 秒数。
未指定、`0`、負値の場合はデフォルトの120秒を使います。
最大値は3600秒で、それより大きい値は3600秒に丸められます。

```json
{
  "mcpServers": {
    "slow-start-server": {
      "command": "npx",
      "args": ["-y", "@example/slow-start-mcp"],
      "startupTimeoutSeconds": 180
    }
  }
}
```

### `toolTimeoutSeconds` (オプション)

MCPツール実行に使う timeout 秒数。
未指定、`0`、負値の場合はデフォルトの600秒を使います。
最大値は3600秒で、それより大きい値は3600秒に丸められます。
ユーザー操作の cancel や request deadline が先に来た場合は、そちらが優先されます。

```json
{
  "mcpServers": {
    "browser": {
      "command": "npx",
      "args": ["-y", "chrome-devtools-mcp@latest"],
      "toolTimeoutSeconds": 900
    }
  }
}
```

**注意**: `env` に明示した値はMCPサーバーへ渡されます。
API token などの secret は必要なサーバーにだけ渡し、不要な値を `env` に書かないでください。

### `approval` (オプション)

MCPサーバーのツール実行承認ポリシー。
未指定時は `confirm` です。

| 値 | 説明 |
|---|---|
| `confirm` | 実行前に確認する。デフォルト |
| `auto` | 確認なしで実行する。信頼できるMCPサーバーにだけ使う |
| `deny` | ツールをモデルに公開せず、実行も拒否する |

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" },
      "approval": "confirm"
    }
  }
}
```

`--auto-approve` や `execution.mode: full_auto` を使っていても、MCPツールは `approval: "auto"` を明示しない限り自動実行されません。

### `toolApprovals` (オプション)

ツール単位で `approval` を上書きできます。
key はMCPサーバーが公開する raw tool name です。XELYON の実行名 `mcp_<server>_<tool>` ではありません。
ただし、サーバーの `approval` が `deny` の場合は server 全体の拒否が優先され、`toolApprovals` では解除できません。

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" },
      "approval": "confirm",
      "toolApprovals": {
        "get_issue": "auto",
        "list_issues": "auto",
        "create_issue": "confirm",
        "delete_repository": "deny"
      }
    }
  }
}
```

評価順は `disabled`、`tools.include` / `tools.exclude`、サーバーの `approval: "deny"`、`toolApprovals`、サーバーの `approval`、デフォルト `confirm` です。
不正な値は warning を出して `confirm` として扱います。

### サーバーの無効化

特定のMCPサーバーを一時的に無効化できます:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-github"],
      "disabled": true
    }
  }
}
```

### ツール単位のフィルタリング

MCPサーバーが提供するツールの中から、使用するツールを制限できます。
不要なツールを除外することでトークン消費を削減できます。

#### ホワイトリスト（include）

指定したツールのみ有効にします:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-github"],
      "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" },
      "tools": {
        "include": ["create_issue", "list_issues", "get_issue"]
      }
    }
  }
}
```

#### ブラックリスト（exclude）

指定したツールだけ除外し、残りを全て有効にします:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-github"],
      "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" },
      "tools": {
        "exclude": ["delete_repository", "transfer_repository"]
      }
    }
  }
}
```

| フィールド | 説明 |
|---|---|
| `disabled` | `true` でサーバー全体を無効化。設定は保持される |
| `tools.include` | ホワイトリスト。指定したツールのみ登録 |
| `tools.exclude` | ブラックリスト。指定したツールを除外（`include` 未設定時のみ有効） |
| `approval` | サーバー単位の承認ポリシー。`confirm` / `auto` / `deny` |
| `toolApprovals` | raw tool name ごとの承認ポリシー上書き |

> **Note**: `include` と `exclude` を両方設定した場合は `include` が優先されます。
> `include` / `exclude` のツール名はMCPサーバーが公開する raw name をそのまま使用します（`mcp_` プレフィックスなし）。
> XELYON がプロンプトと provider に公開する実行名は `mcp_<server>_<tool>` 形式です。同じ実行名に正規化される重複ツールは先に見つかったものを登録し、後続は skipped として扱います。

### モデルに公開する MCP ツール数の上限

XELYON は MCP サーバーへ接続して取得したツールのうち、モデルに公開する current surface を自動で絞ります。
既定では最大80ツール、推定32,000トークン、1ツールあたり128KiBの schema までを provider 定義、system prompt、request context に載せます。
上限を超えたツールは実行名も current surface から除外されるため、モデルから直接呼び出されません。

選定はサーバー名とツール名で deterministic に並べ、複数サーバーがある場合はサーバー間で round-robin します。
省略が発生した場合は stderr に warning を出します。
既定値は、大量の schema を常時 provider に渡して prompt 品質と latency を悪化させないために上げていません。
必要なツールが省略される場合は、まず `~/.xelyon/mcp.json` の `mcpServers.<server>.tools.include` / `tools.exclude` で MCP サーバー側の公開ツールを絞ってください。
その server が意図的に大きく、公開 tool を絞れない場合だけ `~/.xelyon/config.yaml` の `mcp.surface_budget` を上げます。

#### tool が多いときの確認手順

1. 対話中なら `/mcp status` を実行し、`Top omitted reasons`、`Top heavy servers`、`Largest schema tools`、`Highest estimated token tools`、`Recommendations` を確認します。
2. 起動前や別 HOME の設定を確認する場合は `xelyon doctor mcp --connect` を実行します。raw tool name まで確認したい場合は `--tools` を付けます。
3. `Recommendations` に出る `mcpServers` 断片を参考に、必要な raw tool name だけを `tools.include` に入れます。`tools.exclude` は除外したい少数の tool が明確な場合だけ使います。
4. それでも意図した tool が省略される場合だけ、`mcp.surface_budget` を上げます。
5. もう一度 `/mcp status` または `xelyon doctor mcp --connect` を実行し、visible / omitted 数と estimated tokens が意図通りになったことを確認します。

例:

```json
{
  "mcpServers": {
    "github": {
      "tools": {
        "include": ["list_issues", "get_issue", "create_issue"]
      }
    }
  }
}
```

意図的に大きい server の budget を上げる例:

```yaml
mcp:
  surface_budget:
    max_tools: 120
    estimated_tokens: 48000
    max_schema_bytes_per_tool: 131072
```

#### 利用可能なツール名の確認

MCPサーバーが提供するツール一覧は `xelyon doctor mcp --connect --tools` または XELYON 起動時のログで確認できます:

```
🔌 MCP server 'github' connected (5 tools, 25 skipped)
```

全ツールを確認するには、フィルタなしで `xelyon doctor mcp --connect --tools` を実行してください。

## 利用可能な公式MCPサーバー

Anthropicが公開しているMCPサーバー一覧: https://github.com/modelcontextprotocol/servers

### 推奨サーバー

| サーバー名 | 説明 | インストール |
|-----------|------|------------|
| `@modelcontextprotocol/server-filesystem` | ファイル操作 | `npm i -g @modelcontextprotocol/server-filesystem` |
| `@modelcontextprotocol/server-puppeteer` | Web自動化 | `npm i -g @modelcontextprotocol/server-puppeteer` |
| `@modelcontextprotocol/server-sequential-thinking` | 推論補助 | `npm i -g @modelcontextprotocol/server-sequential-thinking` |
| `@modelcontextprotocol/server-postgres` | PostgreSQL操作 | `npm i -g @modelcontextprotocol/server-postgres` |
| `@modelcontextprotocol/server-slack` | Slack連携 | `npm i -g @modelcontextprotocol/server-slack` |

## 使用例

### 1. Filesystemサーバー（ファイル操作）

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": [
        "@modelcontextprotocol/server-filesystem",
        "/home/user/documents"
      ],
      "approval": "confirm"
    }
  }
}
```

```bash
xelyon
> /home/user/documents 配下のファイル一覧を取得して

# AIがMCPツールを提案し、確認後に使用
```

### 2. Puppeteerサーバー（Web自動化）

```json
{
  "mcpServers": {
    "puppeteer": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-puppeteer"],
      "approval": "confirm"
    }
  }
}
```

```bash
xelyon
> https://example.com のスクリーンショットを撮って

# AIがPuppeteerを使うMCPツールを提案し、確認後に実行
```

### 3. PostgreSQLサーバー（データベース操作）

```json
{
  "mcpServers": {
    "postgres": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-postgres"],
      "approval": "confirm",
      "env": {
        "POSTGRES_CONNECTION_STRING": "postgresql://user:password@localhost/mydb"
      }
    }
  }
}
```

```bash
xelyon
> usersテーブルの件数を教えて

# AIがPostgreSQL用MCPツールを提案し、確認後に実行
```

### 4. GitHubサーバー（GitHub連携）

GitHub MCPサーバーを使用すると、AIがGitHub操作を直接実行できます。

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxxxxxxxxxxx"
      },
      "approval": "confirm",
      "toolApprovals": {
        "get_issue": "auto",
        "list_issues": "auto",
        "create_issue": "confirm",
        "delete_repository": "deny"
      }
    }
  }
}
```

**対応操作**:
- Issue作成・取得・一覧
- Pull Request作成・一覧
- GitHub Actions ワークフロー確認
- ファイル内容取得
- コード検索

```bash
xelyon
> このバグのIssueを作成して

# AIがmcp_github_create_issueツールを使用

xelyon
> CIの状態を確認して

# AIがmcp_github_get_workflow_runsツールを使用

xelyon
> Issue #123の内容を見せて

# AIがmcp_github_get_issueツールを使用
```

**特徴**: GitHub MCPサーバーのツールは通常のMCPツールと同じく `mcp_<server>_<tool>` 形式で登録され、AIは公開されたツール定義と入力スキーマに従って呼び出します。

## カスタムMCPサーバーの作成

### Node.jsでの実装例

```javascript
#!/usr/bin/env node
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';

const server = new Server(
  { name: 'my-custom-server', version: '1.0.0' },
  { capabilities: { tools: {} } }
);

server.setRequestHandler('tools/list', async () => {
  return {
    tools: [
      {
        name: 'my_tool',
        description: 'カスタムツールの説明',
        inputSchema: {
          type: 'object',
          properties: {
            arg1: { type: 'string', description: '引数1の説明' }
          },
          required: ['arg1']
        }
      }
    ]
  };
});

server.setRequestHandler('tools/call', async (request) => {
  if (request.params.name === 'my_tool') {
    // ツールの処理
    return {
      content: [
        { type: 'text', text: '実行結果' }
      ]
    };
  }
});

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main();
```

`inputSchema` の必須引数は JSON Schema 標準の top-level `required` 配列で指定してください。
`properties.<name>.required=true` のような property 内の marker は必須判定には使われません。

### XELYONでの使用

```json
{
  "mcpServers": {
    "my-custom-server": {
      "command": "node",
      "args": ["/path/to/my-custom-server.js"],
      "approval": "confirm"
    }
  }
}
```

## トラブルシューティング

### MCPサーバーが起動しない

```bash
# エラーログ確認
xelyon

# 出力例:
# [WARNING] Failed to connect to MCP server 'filesystem': ...
```

**解決策**:
1. MCPサーバーがインストールされているか確認
   ```bash
   npx @modelcontextprotocol/server-filesystem --version
   ```

2. `mcp.json` の `command` と `args` が正しいか確認

3. 手動でMCPサーバーを起動して動作確認
   ```bash
   npx @modelcontextprotocol/server-filesystem /path/to/directory
   ```

### ツールが表示されない

MCPサーバーは接続できているが、ツールが使えない場合:

```bash
xelyon
> 使えるツール一覧を見せて
```

AIが応答でMCPツールを含めない場合は、SystemPromptに正しくツールが追加されているか確認。

### タイムアウトエラー

MCPのデフォルト timeout は、起動・接続・`tools/list` が120秒、ツール実行が600秒です。

- ツール実行 timeout: `internal/mcptool` が `tools.ExecutionContext.EffectiveContext()` を親にして管理します。ユーザー操作の cancel や request deadline が先に来た場合は、`toolTimeoutSeconds` より優先されます。
- 起動時の connect / `ListTools` timeout: `internal/mcp` がサーバーごとの接続と tool listing を管理します。caller context の deadline が `startupTimeoutSeconds` より短い場合はそちらが優先されます。
- `startupTimeoutSeconds` / `toolTimeoutSeconds` はサーバーごとに設定できます。未指定時は上記デフォルト、最大値は3600秒です。

### MCP ツール結果が大きすぎる

MCP ツール結果が64KiBを超える場合、XELYON は結果を履歴へ保存する前に compact します。
履歴、session conversation、provider へ返す `function_call_output`、画面表示には metadata と、内容が secret/private と判定されない場合だけ redaction 済みの head/tail excerpt が残ります。
全文は `provider_history_reduction.raw_output_artifacts.mode: apply` のときだけ、既存の raw output artifact store に `surface=mcp_tool_result` として保存を試みます。

secret-like / private-looking な内容、store disabled、quota 超過、artifact 作成失敗などでは全文保存を行わず、placeholder に `full_output_omitted_reason` を記録します。
この runtime guard は provider history reduction より前に動作するため、大きな MCP 結果がそのまま履歴や provider payload に入り続けることを防ぎます。

## セキュリティ

### 環境変数のサニタイズ

XELYONはsystem envから `PATH`, `HOME`, `USER`, `LANG`, `LC_ALL`, `NODE_OPTIONS`, `PYTHONPATH` だけを継承します。
`OPENAI_API_KEY` や `GITHUB_TOKEN` のような値はsystem envからは自動継承されません。
ただし、`mcp.json` の `env` に明示した値は `${VAR}` 展開後にMCPサーバーへ渡されます。

### 推奨事項

1. **最小権限の原則**: MCPサーバーには必要最小限のアクセス権限のみを与える
2. **承認ポリシー**: デフォルトの `confirm` を基本にし、`auto` は信頼できる read-only 系ツールなどに限定する
3. **危険なツールの拒否**: 使わない破壊的ツールは `toolApprovals` で `deny` にする
4. **信頼できるサーバーのみ**: 公式サーバーまたは信頼できるソースからのみインストール
5. **定期的な更新**: MCPサーバーを最新版に保つ
6. **ログ確認**: 異常な動作がないか定期的に確認

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [設定リファレンス](config.md)
- [プロバイダー設定](providers.md)
- [Model Context Protocol 公式サイト](https://modelcontextprotocol.io)
- [MCP Servers リポジトリ](https://github.com/modelcontextprotocol/servers)
