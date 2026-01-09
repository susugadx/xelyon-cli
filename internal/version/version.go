package version

// Version はXELYON CLIのバージョン（セキュリティ・品質監査完了版）
const Version = "0.24.0"

// GetVersion はバージョン文字列を返す
func GetVersion() string {
	return Version
}
