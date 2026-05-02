package configgen

// StructInfo stores parsed config struct information for docs generation.
type StructInfo struct {
	Name    string
	Comment string
	Fields  []FieldInfo
}

// FieldInfo stores parsed field information for docs generation.
type FieldInfo struct {
	Name       string
	Type       string
	YAMLTag    string
	Comment    string
	IsOptional bool
}
