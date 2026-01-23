package tools

// ToolCacheInterface はツール結果のキャッシュインターフェース
// agent パッケージで実装され、tools パッケージから使用される
type ToolCacheInterface interface {
	// GetFile はファイル内容のキャッシュを取得
	// キャッシュヒット時は (content, true) を返す
	// ファイルが変更されていた場合やキャッシュミス時は ("", false) を返す
	GetFile(path string) (string, bool)

	// SetFile はファイル内容をキャッシュに保存
	// 1MB以上のファイルはキャッシュしない
	SetFile(path, content string)

	// GetDir はディレクトリ一覧のキャッシュを取得
	// キャッシュヒット時は (result, true) を返す
	// ディレクトリが変更されていた場合やキャッシュミス時は ("", false) を返す
	GetDir(path string) (string, bool)

	// SetDir はディレクトリ一覧をキャッシュに保存
	SetDir(path, result string)

	// InvalidateFile は指定ファイルのキャッシュを無効化
	// ファイル書き込み系ツール実行後に呼ばれる
	InvalidateFile(path string)

	// InvalidateDir は指定ディレクトリのキャッシュを無効化
	// ファイル作成/削除後に呼ばれる
	InvalidateDir(path string)

	// Clear は全キャッシュをクリア
	Clear()

	// GetSearch は検索結果のキャッシュを取得
	// キャッシュヒット時は (result, true) を返す
	// キャッシュミス時は ("", false) を返す
	GetSearch(pattern, path string) (string, bool)

	// SetSearch は検索結果をキャッシュに保存
	SetSearch(pattern, path, result string)

	// ClearSearchCache は検索キャッシュをクリア
	// ファイル変更系ツール実行時に呼ばれる
	ClearSearchCache()
}

// GlobalToolCache はグローバルなツールキャッシュ
// agent.NewAgent() で設定される
var GlobalToolCache ToolCacheInterface
