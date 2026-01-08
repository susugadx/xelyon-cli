package version

// Version はXELYON CLIのバージョン（マルチプロバイダー対応版）
const Version = "0.15.0"

// GetVersion はバージョン文字列を返す
func GetVersion() string {
	return Version
}
