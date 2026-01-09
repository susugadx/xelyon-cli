package version

// Version はXELYON CLIのバージョン（セキュリティ・品質監査完了版 + LOW優先度機能追加）
const Version = "0.25.0"

// GetVersion はバージョン文字列を返す
func GetVersion() string {
	return Version
}
