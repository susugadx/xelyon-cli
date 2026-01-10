package version

import "fmt"

// Version はXELYON CLIのバージョン（GoReleaserでビルド時に上書きされる）
var Version = "0.30.0-dev"

// Commit はGitコミットハッシュ（GoReleaserでビルド時に設定される）
var Commit = "unknown"

// Date はビルド日時（GoReleaserでビルド時に設定される）
var Date = "unknown"

// GetVersion はバージョン文字列を返す
func GetVersion() string {
	return Version
}

// GetFullVersion は詳細バージョン情報を返す
func GetFullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
}
