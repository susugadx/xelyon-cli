# MCP (Model Context Protocol) 連携

XELYON CLIは[Model Context Protocol (MCP)](https://modelcontextprotocol.io)に対応しており、外部ツールを動的に追加できます。

## MCPとは

Model Context Protocol（MCP）は、AIアシスタントが外部のツールやデータソースに安全にアクセスするための標準プロトコルです。

### 主な特徴

- **プラグイン方式**: 外部プロセス（MCPサーバー）としてツールを実行
- **stdioトランスポート**: 標準入出力でJSON-RPC 2.0通信
- **動的登録**: 起動時にMCPサーバーからツール一覧を自動取得
- **セキュリティ**: API キーなどの機密情報は自動的に除外

## 設定

`config.yaml` で MCP の動作を制御できます:

```yaml
# ~/.xelyon/config.yaml
mcp:
  enabled: true    # MCP機能のON/OFF（デフォルト: true）
  headless: false  # ヘッドレスモードでMCPを使うか（デフォルト: false）
```

| 設定 | 説明 | デフォルト |
|------|------|-----------|
| `enabled` | `false` にするとMCPサーバーへの接続をスキップ。トークン消費を削減。 | `true` |
| `headless` | `true` にすると `--headless` モードでもMCPツールが使える。 | `false` |

`enabled: false` にしても `~/.xelyon/mcp.json` の設定はそのまま残るため、再度 `enabled: true` にすれば復活します。

## セットアップ

### 1. 設定ファイルの作成

`~/.xelyon/mcp.json` にMCPサーバーを定義します。

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
      }
    },
    "puppeteer": {
      "command": "npx",
      "args": [
        "@modelcontextprotocol/server-puppeteer"
      ]
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

## 設定項目

### `command` (必須)

実行するコマンド。npx, node, python, sh など。

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

MCPサーバーに渡す環境変数。

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

**セキュリティ**: 以下のパターンに一致する環境変数は**自動的に除外**されます。
- `*_KEY`
- `*_TOKEN`
- `*_SECRET`
- `*_PASSWORD`
- `*_API_KEY`

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

> **Note**: `include` と `exclude` を両方設定した場合は `include` が優先されます。
> `include` / `exclude` のツール名はMCPサーバーが公開する raw name をそのまま使用します（`mcp_` プレフィックスなし）。
> XELYON がプロンプトと provider に公開する実行名は `mcp_<server>_<tool>` 形式です。同じ実行名に正規化される重複ツールは先に見つかったものを登録し、後続は skipped として扱います。

#### 利用可能なツール名の確認

MCPサーバーが提供するツール一覧はXELYON起動時のログで確認できます:

```
🔌 MCP server 'github' connected (5 tools, 25 skipped)
```

全ツールを確認するには、フィルタなしで一度起動してください。

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
      ]
    }
  }
}
```

```bash
xelyon
> /home/user/documents 配下のファイル一覧を取得して

# AIがMCPツールを自動で使用
```

### 2. Puppeteerサーバー（Web自動化）

```json
{
  "mcpServers": {
    "puppeteer": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-puppeteer"]
    }
  }
}
```

```bash
xelyon
> https://example.com のスクリーンショットを撮って

# AIがPuppeteerを使ってスクリーンショット取得
```

### 3. PostgreSQLサーバー（データベース操作）

```json
{
  "mcpServers": {
    "postgres": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-postgres"],
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

# AIがPostgreSQLに接続してクエリ実行
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

### XELYONでの使用

```json
{
  "mcpServers": {
    "my-custom-server": {
      "command": "node",
      "args": ["/path/to/my-custom-server.js"]
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

MCPのタイムアウトは現在どちらもデフォルト30秒です。

- ツール実行 timeout: `internal/mcptool` が `tools.ExecutionContext.EffectiveContext()` を親にして管理します。ユーザー操作の cancel や request deadline が先に来た場合は、30秒の tool timeout より優先されます。
- 起動時の connect / `ListTools` timeout: `internal/mcp` がサーバーごとの接続と tool listing を管理します。caller context の deadline が30秒より短い場合はそちらが優先されます。

設定項目はまだありません。変更する場合はコード修正が必要です:

```go
// internal/mcptool/wrapper.go
const defaultMCPToolCallTimeout = 30 * time.Second

// internal/mcp/client.go
const defaultMCPServerOperationTimeout = 30 * time.Second
```

## セキュリティ

### 環境変数のサニタイズ

XELYONは以下の機密情報を自動的にMCPサーバーに渡しません:

- `DEEPSEEK_API_KEY`
- `OPENAI_API_KEY`
- `GEMINI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GROQ_API_KEY`
- その他 `*_KEY`, `*_TOKEN`, `*_SECRET`, `*_PASSWORD`, `*_API_KEY` パターン

### 推奨事項

1. **最小権限の原則**: MCPサーバーには必要最小限のアクセス権限のみを与える
2. **信頼できるサーバーのみ**: 公式サーバーまたは信頼できるソースからのみインストール
3. **定期的な更新**: MCPサーバーを最新版に保つ
4. **ログ確認**: 異常な動作がないか定期的に確認

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [設定リファレンス](config.md)
- [プロバイダー設定](providers.md)
- [Model Context Protocol 公式サイト](https://modelcontextprotocol.io)
- [MCP Servers リポジトリ](https://github.com/modelcontextprotocol/servers)
