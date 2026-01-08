package version

// Version はXELYON CLIのバージョン（Phase 4: 破壊的・複雑ツール対応版）
const Version = "0.19.0"

// GetVersion はバージョン文字列を返す
func GetVersion() string {
	return Version
}
