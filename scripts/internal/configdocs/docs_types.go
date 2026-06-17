package configdocs

// structInfo は docs 生成用に解析した config struct 情報を保持する。
type structInfo struct {
	Name    string
	Comment string
	Fields  []fieldInfo
}

// fieldInfo は docs 生成用に解析した field 情報を保持する。
type fieldInfo struct {
	Name       string
	Type       string
	YAMLTag    string
	Comment    string
	IsOptional bool
}
