# CLAUDE.md - Claude Code開発ガイド

このファイルは、Claude Codeを使ってXELYON CLIを開発する際のガイドです。

## プロジェクト概要

XELYON CLIは、AI搭載のコーディングアシスタントCLIツールです。DeepSeek APIを使用して対話的にコード編集やGit操作を実行できます。

## アーキテクチャ

### 主要コンポーネント

1. **Agent（`internal/agent/agent.go`）**
   - 対話ループのメイン処理
   - ユーザー入力受付とAI応答の処理
   - ツール実行結果の統合
   - 会話履歴の自動保存

2. **Tools（`internal/tools/tools.go`）**
   - 15種類のツール実装（bash, read_file, write_file, str_replace, list_dir, git_*, search_*）
   - ツール呼び出しのパース
   - 安全性チェック（危険コマンドのブロック）
   - 自動ページング機能

3. **API（`internal/api/deepseek.go`）**
   - DeepSeek APIとの通信
   - ストリーミングレスポンスの処理
   - スピナー表示の統合

4. **History（`internal/history/`）**
   - セッション管理（session.go）
   - JSONL形式での永続化（storage.go）
   - メタデータ管理

5. **UI（`internal/ui/`）**
   - スピナーアニメーション（spinner.go）
   - 自動ページング（pager.go）

## 開発ガイドライン

### コーディングスタイル

- **エラーハンドリング**: 全てのエラーは適切にハンドリングする
- **コメント**: 日本語コメント推奨（関数・構造体の説明）
- **フォーマット**: `go fmt`でフォーマット
- **ネーミング**: GoのベストプラクティスにしたがってPascalCase/camelCaseを使用

### 新機能追加の流れ

1. **要件定義**: 何を実装するか明確にする
2. **設計**: どこに追加するか、既存構造との関係を考える
3. **実装**: 段階的に実装（Phase分け推奨）
4. **テスト**: ビルドして手動テスト
5. **ドキュメント更新**: README.md、CLAUDE.mdを更新

### ツール追加方法

新しいツールを追加する場合:

1. **`internal/tools/tools.go`のExecute関数にcase追加**
   ```go
   case "new_tool":
       result = executeNewTool(tc.Args["arg1"], tc.Args["arg2"])
   ```

2. **実装関数を追加**
   ```go
   func executeNewTool(arg1, arg2 string) string {
       // 実装
       return result
   }
   ```

3. **`internal/agent/agent.go`のSystemPromptに追記**
   ```
   - new_tool: Description. Args: {"arg1": "...", "arg2": "..."}
   ```

4. **printHelp()にも追加**

### 会話履歴の構造

**セッションファイル（JSONL）**: `~/.xelyon/history/{session_id}.jsonl`
```jsonl
{"timestamp":"2026-01-06T10:00:00Z","role":"user","content":"hello","model":"deepseek-chat"}
{"timestamp":"2026-01-06T10:00:05Z","role":"assistant","content":"Hi!","model":"deepseek-chat"}
```

**メタデータファイル（JSON）**: `~/.xelyon/history/metadata/{session_id}.json`
```json
{
  "session_id": "1704567890",
  "model": "deepseek-chat",
  "start_time": "2026-01-06T10:00:00Z",
  "last_modified": "2026-01-06T10:00:05Z",
  "message_count": 2,
  "preview": "hello"
}
```

## よくある開発タスク

### 1. 新しいコマンド追加

`internal/agent/agent.go`の`handleSpecialCommand`に追加:

```go
case "/newcommand":
    return handleNewCommand(agent, args)
```

対応するハンドラーを実装:

```go
func handleNewCommand(agent *Agent, args []string) bool {
    // 実装
    return true
}
```

### 2. UI改善

- スピナー: `internal/ui/spinner.go`
- ページング: `internal/ui/pager.go`
- 色付け: `github.com/fatih/color`パッケージ使用

### 3. API統合変更

`internal/api/deepseek.go`のChatWithTools関数を修正。ストリーミングレスポンスの処理に注意。

### 4. 永続化層の拡張

`internal/history/storage.go`に新しいメソッド追加。JSONL形式を維持すること。

## デバッグ方法

### ログ出力

開発中の一時的なデバッグには`fmt.Println`を使用:

```go
fmt.Printf("DEBUG: variable = %+v\n", variable)
```

### エラーハンドリング

エラーメッセージは詳細に:

```go
if err != nil {
    return fmt.Errorf("failed to process file %s: %w", path, err)
}
```

### 手動テスト

```bash
# ビルド
go build -o xelyon

# 対話モードでテスト
./xelyon

# ワンショットでテスト
./xelyon "test query"

# フラグのテスト
./xelyon --resume
./xelyon --coder
```

## トラブルシューティング

### ビルドエラー

```bash
# 依存関係の更新
go mod tidy

# クリーンビルド
go clean
go build -o xelyon
```

### 会話履歴が読めない

```bash
# 履歴ディレクトリの確認
ls -la ~/.xelyon/history/

# JSONLの妥当性チェック
cat ~/.xelyon/history/*.jsonl | jq .
```

### スピナーが表示されない

- ターミナルがANSI escape codesをサポートしているか確認
- `TERM`環境変数を確認

## 既知の制約

1. **ストリーミング**: DeepSeek APIはストリーミングレスポンスのみサポート
2. **履歴サイズ**: 大きなセッション（1000+メッセージ）は読み込みが遅くなる可能性
3. **ページング**: AIの応答にはページング適用されない（ツール出力のみ）
4. **並行実行**: 現在は1つのツールを順次実行（並列実行は未対応）

## 今後の実装予定

### 優先度: 高

- [ ] 設定ファイル（`~/.xelyon/config.yaml`）
- [ ] 動的モデル切り替え（`/model deepseek-coder`コマンド）
- [ ] `run_test`ツール（go test/npm test/pytest自動検出）

### 優先度: 中

- [ ] セッション検索機能
- [ ] セッションエクスポート（Markdown/HTML）
- [ ] ツール実行の並列化
- [ ] Claude API対応完全実装

### 優先度: 低

- [ ] セッション圧縮（gzip）
- [ ] セッションタグ付け
- [ ] AI応答のストリーミングページング
- [ ] プラグインシステム

## コントリビューション

新機能やバグ修正を実装する際:

1. まずClaude Codeに相談して設計を確認
2. 段階的に実装（Phase分け）
3. 各Phaseごとにビルド&テスト
4. README.md、CLAUDE.mdを更新
5. コミットメッセージは明確に

## 参考リンク

- [Go Best Practices](https://go.dev/doc/effective_go)
- [Cobra CLI](https://github.com/spf13/cobra)
- [DeepSeek API](https://platform.deepseek.com/api-docs/)
- [JSONL Format](https://jsonlines.org/)

## バージョン情報

- **現在のバージョン**: v0.3.0
- **Go バージョン**: 1.22+
- **主要依存**: cobra, fatih/color

---

**最終更新**: 2026-01-06
