package configmeta

// CanonicalFieldPath は section と field から生成コードで使う正規化パスを返す。
func CanonicalFieldPath(sectionName, fieldName string) string {
	if fieldName == sectionName {
		return sectionName
	}
	return sectionName + "." + fieldName
}
