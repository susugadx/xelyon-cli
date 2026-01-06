# XELYON CLI プロジェクト設定

## 概要
Go製のAI搭載コーディングアシスタントCLI。DeepSeek APIを使った対話型エージェントで、ツールを使って実際にコード編集・Git操作を実行します。

## 技術スタック
- **Go 1.22+**
- **Cobra** - CLIフレームワーク
- **DeepSeek API** - AI推論（V3, Coder, R1モデル）
- **fatih/color** - ターミナル色付け

## アーキテクチャ

### ディレクトリ構造
```
xelyon-cli/
├── main.go                 # エントリーポイント
├── cmd/
│   └── root.go            # Cobraコマンド定義
├── internal/
│   ├── agent/             # エージェントロジック
│   │   └── agent.go       # 対話ループ、コマンド処理、Undo管理
│   ├── api/               # API クライアント
│   │   ├── deepseek.go    # DeepSeek API（ストリーミング、スピナー統合）
│   │   └── xelyon.go      # RAG検索API
│   ├── config/            # 設定管理
│   │   └── config.go      # 設定ファイル読み書き
│   ├── tools/             # ツール実行エンジン
│   │   └── tools.go       # ツール実装、バックアップ管理
│   ├── ui/                # UI コンポーネント
│   │   ├── spinner.go     # ローディングスピナー
│   │   └── pager.go       # 自動ページング（100行超え）
│   ├── history/           # セッション管理
│   │   ├── session.go     # セッション構造
│   │   └── storage.go     # JSONL永続化
│   └── file/              # ファイルI/O
│       ├── reader.go
│       └── writer.go
└── XELYON.md              # このファイル（自動読み込み）
```

### コア機能

#### 1. エージェントシステム (internal/agent/)
- **対話ループ**: ユーザー入力 → AI推論 → ツール実行 → 結果表示
- **コマンド処理**: `/save`, `/load`, `/sessions`, `/undo`, `/config`, `/model`, `/version`, `/clear`, `/history`, `/help`
- **変更履歴管理**: 最大10件のファイル変更を追跡、Undo機能
- **セッション管理**: 会話履歴の自動保存・復元
- **ループ検知**: 同じツール呼び出しが3回繰り返されると自動中断
- **APIリトライ**: エラー時に最大2回自動リトライ（指数バックオフ）

#### 2. ツールシステム (internal/tools/)
- **15種類のツール**: bash, read_file, write_file, str_replace, list_dir, git_*, search_*
- **自動バックアップ**: write_file/str_replaceで.bakファイル作成
- **安全性**: 危険なコマンド（rm -rf, sed -i等）をブロック
- **FileChange追跡**: ファイル変更をメタデータと共に記録
- **差分表示の省略**: 10行超えた場合、最初の10行のみ表示
- **エラーヒント**: 複数マッチ時にファイルプレビュー（先頭50行）を表示

#### 3. UI/UX (internal/ui/)
- **スピナー**: API呼び出し中のローディング表示（80ms更新）
- **ページング**: 100行を超える出力を自動的に分割表示
- **色付け**: cyan/green/yellow/redで情報を視覚的に区別

#### 4. 履歴管理 (internal/history/)
- **JSONL形式**: ストリーミング対応、1行1メッセージ
- **メタデータ分離**: session_id, model, timestamp, preview等
- **保存先**: `~/.xelyon/history/`

#### 5. 設定管理 (internal/config/)
- **YAML形式**: `~/.xelyon/config.yaml`
- **デフォルトモデル**: default_model設定で起動時のモデル指定
- **自動作成**: 初回起動時にデフォルト設定を自動生成
- **バリデーション**: 無効なモデル名を拒否

#### 6. バージョン管理 (internal/version/)
- **一元管理**: `version.go`でバージョンを定数管理
- **複数表示**: 起動バナー、`/version`コマンド、`--version`フラグ
- **自動反映**: Version定数を変更するだけで全体に反映

## コーディングルール

### エラーハンドリング
- すべてのI/O操作でエラーチェック必須
- ユーザー体験を妨げない範囲でサイレント失敗を許容
- 重要なエラーは`red.Printf()`で明示

### ツール設計原則
1. **冪等性**: 同じ入力で同じ結果
2. **確認プロンプト**: 破壊的操作（write_file, str_replace）は必ず確認
3. **diff表示**: 変更内容を視覚化
4. **バックアップ**: 既存ファイル編集時は.bakを作成

### Undo機能の仕組み
```go
// FileChange構造体
type FileChange struct {
    FilePath    string    // 変更したファイル
    BackupPath  string    // .bakファイルパス
    Timestamp   time.Time // 変更時刻
    Tool        string    // 使用ツール
    Description string    // 変更内容
}

// Agent.changeStack: 最大10件保持（FIFO）
// /undo: スタックからpop → .bakから復元
```

### SystemPromptルール（重要）
AIエージェントは以下のルールに従う:
1. ファイル編集は**必ず**`str_replace`ツールを使用（bashでsedは禁止）
2. `write_file`は**新規ファイル作成のみ**（既存ファイル編集は禁止）
3. `str_replace`の方が安全（diff表示、確認プロンプト、バックアップ）
4. **v0.8.0追加**: old_strが複数マッチした場合、前後のコンテキストを含めて一意にする
5. **v0.8.0追加**: 長いファイル編集は複数回のstr_replaceに分割する（10行程度ずつ）
6. **v0.8.0追加**: 同じファイルを連続編集する場合、read_fileで最新状態を確認してから次の変更を行う

## 開発ガイド

### 新機能追加時のチェックリスト
- [ ] ツール追加 → `tools.go`のExecute()に分岐追加
- [ ] コマンド追加 → `agent.go`のhandleSpecialCommand()に追加
- [ ] ヘルプ更新 → `printHelp()`を更新
- [ ] README更新 → バージョン履歴に追記

### ツール追加方法

新しいツールを追加する場合の手順:

1. **`internal/tools/tools.go`のExecute関数にcase追加**
```go
case "new_tool":
    result = executeNewTool(tc.Args["arg1"], tc.Args["arg2"])
```

2. **実装関数を追加**
```go
func executeNewTool(arg1, arg2 string) string {
    // バリデーション
    if arg1 == "" {
        return "Error: arg1 is required"
    }

    // 処理実装
    result := // ...

    return result
}
```

3. **`internal/agent/agent.go`のSystemPromptに追記**
```
- new_tool: Description of the tool. Args: {"arg1": "...", "arg2": "..."}
```

4. **`printHelp()`にも追加**

### コマンド追加方法

新しい対話コマンドを追加する場合:

1. **`internal/agent/agent.go`の`handleSpecialCommand`に追加**
```go
case "/newcommand":
    return handleNewCommand(agent, args)
```

2. **対応するハンドラーを実装**
```go
func handleNewCommand(agent *Agent, args []string) bool {
    // コマンド処理
    green.Println("Command executed!")
    return true
}
```

3. **`printHelp()`を更新**

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

### デバッグ方法

**ログ出力**（開発中の一時的なデバッグ）:
```go
fmt.Printf("DEBUG: variable = %+v\n", variable)
```

**エラーメッセージは詳細に**:
```go
if err != nil {
    return fmt.Errorf("failed to process file %s: %w", path, err)
}
```

**手動テスト**:
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
./xelyon --think
```

### ビルド＆テスト
```bash
# ビルド
go build -o xelyon

# フォーマット
go fmt ./...

# テスト
go test ./...

# 依存関係の更新
go mod tidy

# クリーンビルド
go clean
go build -o xelyon
```

## トラブルシューティング

### 履歴が保存されない
- `~/.xelyon/history/`ディレクトリの権限確認
- `storage.Save()`のエラーログ確認（サイレント失敗）

### スピナーが止まらない
- API応答の`firstChunk`フラグ確認
- `spinner.Stop()`呼び出し漏れチェック

### Undoできない
- `changeStack`が空 → 編集操作をまだ実行していない
- `.bak`ファイルなし → 新規ファイル作成（バックアップ不要）

### ビルドエラー
- 依存関係の問題 → `go mod tidy`実行
- クリーンビルド → `go clean && go build -o xelyon`

### 会話履歴が読めない
- `~/.xelyon/history/`ディレクトリの権限確認
- JSONLの妥当性チェック: `cat ~/.xelyon/history/*.jsonl | jq .`

### 設定ファイルが読めない
- `~/.xelyon/config.yaml`の権限確認
- YAMLの妥当性チェック: `cat ~/.xelyon/config.yaml`
- 削除して再起動でデフォルト設定が再作成される

## 既知の制約

1. **ストリーミング**: DeepSeek APIはストリーミングレスポンスのみサポート
2. **履歴サイズ**: 大きなセッション（1000+メッセージ）は読み込みが遅くなる可能性
3. **ページング**: AIの応答にはページング適用されない（ツール出力のみ）
4. **並行実行**: 現在は1つのツールを順次実行（並列実行は未対応）
5. **バックアップ**: .bakファイルは1世代のみ保持（複数世代バックアップ未対応）
6. **ループ検知**: 3回同じツール呼び出しで中断（v0.8.0で追加）
7. **APIリトライ**: 最大2回までリトライ（v0.8.0で追加）

## 今後の拡張案
- [ ] タイムスタンプ付き複数世代バックアップ
- [ ] `/undo all` - セッション中の全変更を一括取り消し
- [ ] `/changes` - 変更履歴一覧表示
- [ ] 永続的Undo - セッション終了後も復元可能
- [ ] .gitignoreへの.bak自動追加
- [ ] ループ検知の回数をカスタマイズ可能に
- [ ] APIリトライの回数・待機時間をカスタマイズ可能に
- [ ] 差分表示の省略行数をカスタマイズ可能に

## v0.8.0 で実装済み
- [x] ループ検知機能（3回で中断）
- [x] APIリトライ機能（最大2回）
- [x] 長い差分表示の省略（10行超え）
- [x] キャンセル時のメッセージ改善
- [x] 複数マッチ時のファイルプレビュー
- [x] バージョン管理の一元化
- [x] `/model`コマンドで再起動なしに切り替え
- [x] `/version`コマンド
- [x] `--version`フラグ
