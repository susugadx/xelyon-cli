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
- **コマンド処理**: `/save`, `/load`, `/sessions`, `/undo`, `/clear`, `/history`, `/help`
- **変更履歴管理**: 最大10件のファイル変更を追跡、Undo機能
- **セッション管理**: 会話履歴の自動保存・復元

#### 2. ツールシステム (internal/tools/)
- **15種類のツール**: bash, read_file, write_file, str_replace, list_dir, git_*, search_*
- **自動バックアップ**: write_file/str_replaceで.bakファイル作成
- **安全性**: 危険なコマンド（rm -rf, sed -i等）をブロック
- **FileChange追跡**: ファイル変更をメタデータと共に記録

#### 3. UI/UX (internal/ui/)
- **スピナー**: API呼び出し中のローディング表示（80ms更新）
- **ページング**: 100行を超える出力を自動的に分割表示
- **色付け**: cyan/green/yellow/redで情報を視覚的に区別

#### 4. 履歴管理 (internal/history/)
- **JSONL形式**: ストリーミング対応、1行1メッセージ
- **メタデータ分離**: session_id, model, timestamp, preview等
- **保存先**: `~/.xelyon/history/`

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

## 開発ガイド

### 新機能追加時のチェックリスト
- [ ] ツール追加 → `tools.go`のExecute()に分岐追加
- [ ] コマンド追加 → `agent.go`のhandleSpecialCommand()に追加
- [ ] ヘルプ更新 → `printHelp()`を更新
- [ ] README更新 → バージョン履歴に追記

### ビルド＆テスト
```bash
# ビルド
go build -o xelyon

# フォーマット
go fmt ./...

# テスト
go test ./...
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

## 今後の拡張案
- [ ] タイムスタンプ付き複数世代バックアップ
- [ ] `/undo all` - セッション中の全変更を一括取り消し
- [ ] `/changes` - 変更履歴一覧表示
- [ ] 永続的Undo - セッション終了後も復元可能
- [ ] .gitignoreへの.bak自動追加
