# LSP連携ガイド

XELYON CLIはLanguage Server Protocol (LSP) を活用してIDE並みのコード理解を実現します。

## LSPとは

Language Server Protocol (LSP) は、エディタとプログラミング言語の解析サーバー間の通信プロトコルです。これにより、AIが以下の機能を使用できます：

- **参照検索**: シンボルのすべての参照箇所を検索
- **定義ジャンプ**: 関数や型の定義位置を特定
- **ホバー情報**: 型情報やドキュメントを取得
- **診断情報**: ファイルのエラー・警告を取得
- **リネームプレビュー**: シンボルのリネーム変更箇所をプレビュー
- **削除時参照チェック**: ファイル削除前に外部参照を自動検出し警告

## 対応言語（23言語）

### メイン言語
| 言語 | LSPサーバー | インストールコマンド |
|------|------------|-------------------|
| Go | gopls | `go install golang.org/x/tools/gopls@latest` |
| TypeScript/JavaScript | vtsls | `npm i -g @vtsls/language-server typescript` |
| Python | pyright | `pip install pyright` または `npm i -g pyright` |
| Rust | rust-analyzer | `rustup component add rust-analyzer` |

### バックエンド言語
| 言語 | LSPサーバー | インストールコマンド |
|------|------------|-------------------|
| Java | jdtls | `brew install jdtls` |
| C/C++ | clangd | `brew install llvm` または `apt install clangd` |
| Ruby | solargraph | `gem install solargraph` |
| Kotlin | kotlin-language-server | `brew install kotlin-language-server` |
| Swift | sourcekit-lsp | Xcode/Swift toolchain に含まれる |
| C# | csharp-ls | `dotnet tool install --global csharp-ls` |
| Scala | metals | `brew install coursier/formulas/coursier && cs install metals` |
| PHP | intelephense | `npm i -g intelephense` |
| Elixir | elixir-ls | `brew install elixir-ls` |
| Lua | lua-language-server | `brew install lua-language-server` |

### フロントエンド言語
| 言語 | LSPサーバー | インストールコマンド |
|------|------------|-------------------|
| CSS/SCSS | vscode-css-language-server | `npm i -g vscode-langservers-extracted` |
| HTML | vscode-html-language-server | `npm i -g vscode-langservers-extracted` |
| Vue | @vue/language-server | `npm i -g @vue/language-server` |
| Svelte | svelte-language-server | `npm i -g svelte-language-server` |

### 設定/スクリプト言語
| 言語 | LSPサーバー | インストールコマンド |
|------|------------|-------------------|
| YAML | yaml-language-server | `npm i -g yaml-language-server` |
| TOML | taplo | `cargo install taplo-cli --locked` |
| SQL | sqls | `go install github.com/lighttiger2505/sqls@latest` |
| Bash | bash-language-server | `npm i -g bash-language-server` |
| Markdown | marksman | `brew install marksman` |

## セットアップ

`/lsp` は legacy classic (`--no-tui`) 用の診断コマンドです。TUI では候補と `/help` に表示されません。以下の `/lsp` コマンド例は `xelyon --no-tui` で起動した classic REPL 内で実行します。

### 1. 言語検出

プロジェクト内の言語を自動検出します。

```bash
xelyon --no-tui
> /lsp detect
```

**出力例:**
```
Detected languages / 検出された言語:
  - Go (3 files)
  - TypeScript (15 files)
  - Python (2 files)

LSP server status / LSPサーバー状態:
  ✅ Go: gopls (installed)
  ⚠️ TypeScript: vtsls (not installed)
  ⚠️ Python: pyright (not installed)

💡 Install with: /lsp install typescript
```

### 2. LSPサーバーのインストール

```bash
# 個別にインストール
> /lsp install go
> /lsp install typescript
> /lsp install python

# 未インストールの全サーバーをインストール
> /lsp install all
```

### 3. ステータス確認

```bash
> /lsp
# または
> /lsp status
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

## 設定

### 基本設定

`~/.xelyon/config.yaml`:

```yaml
lsp:
  enabled: true           # LSP連携の有効/無効（デフォルト: true）
```

### サーバーのカスタマイズ

```yaml
lsp:
  enabled: true
  servers:
    go:
      command: gopls
      # args: []          # オプション
    typescript:
      command: vtsls
      args: ["--stdio"]
    python:
      command: pyright-langserver
      args: ["--stdio"]
    rust:
      command: rust-analyzer
      disabled: true      # 個別サーバーの無効化
```

### 設定項目

| 項目 | 型 | デフォルト | 説明 |
|-----|---|---------|------|
| `lsp.enabled` | boolean | `true` | LSP連携の有効/無効 |
| `lsp.servers.<lang>.command` | string | - | LSPサーバーのコマンド |
| `lsp.servers.<lang>.args` | string[] | `[]` | コマンドに渡す引数 |
| `lsp.servers.<lang>.disabled` | boolean | `false` | 個別サーバーの無効化 |

## LSP機能

LSPは以下の用途で内部的に利用されます（専用ツールは不要）:

- **診断（エラー検知）**: `str_replace` 後の即座フィードバック、コミット前の自動チェック
- **削除時参照チェック**: `delete_file` 実行前に外部参照を自動検出し警告
- **Plan依存分析**: 変更ファイル間の依存関係を自動検出

コード検索には `search_code` ツールを使用してください。`[def]`/`[ref]`/`[call]` アノテーションで定義・参照を自動識別します。

## 削除時参照チェック

ファイル削除（`delete_file`）時にLSPが有効な場合、自動的に外部参照をチェックします。

**表示例:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🗑️  Delete File / ファイル削除
📂 Path / パス: internal/api/handler.go
📏 Size / サイズ: 1234 bytes (45 lines)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️  LSP Warning: This file contains 3 external references!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   HandleUser (2 references):
      - main.go:45
      - routes/api.go:123
   UserHandler (1 references):
      - main.go:67

Continue? (y/n/c):
```

## トラブルシューティング

### LSPサーバーが起動しない

```bash
# legacy classic (--no-tui) でサーバーがインストールされているか確認
> /lsp status

# 手動でサーバーを起動して確認
$ gopls version
$ vtsls --version
```

### 「not installed」と表示される

```bash
# インストールコマンドを実行
> /lsp install <言語>

# 例: TypeScript
> /lsp install typescript
```

### 参照が見つからない

1. ファイルを保存してから再試行
2. LSPサーバーを再起動（XELYONを再起動）
3. プロジェクトのルートディレクトリで起動しているか確認

### タイムアウトエラー

LSPサーバーの応答が遅い場合（デフォルト30秒でタイムアウト）:
- 大きなプロジェクトでは初回起動に時間がかかる場合があります
- 再度実行すると正常に動作することが多いです

### 特定の言語だけ無効化したい

```yaml
# ~/.xelyon/config.yaml
lsp:
  enabled: true
  servers:
    rust:
      disabled: true    # Rustのみ無効化
```

## 遅延起動

LSPサーバーは初回使用時に起動します（XELYON起動時には起動しません）。

- `GetServerForFile()` 呼び出し時に言語を検出
- 該当言語のサーバーがなければ起動
- 起動済みサーバーは再利用

これにより、起動時間を短縮し、必要なサーバーのみを起動します。

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [設定リファレンス](config.md)
- [プロバイダー設定](providers.md)
