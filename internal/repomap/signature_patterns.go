package repomap

// ExtractSignatureMetadata は行テキストからシンボル名と種別を抽出する（全言語）。
// Project Map 生成用。
func ExtractSignatureMetadata(sig string) (string, string, bool) {
	return extractSignatureMetadata(sig)
}

// ExtractSignatureMetadataForLang は指定言語に限定してシンボル名と種別を抽出する。
// search_code の多言語シンボル解決用。lang が空の場合は全言語パターンを適用する。
func ExtractSignatureMetadataForLang(sig, lang string) (string, string, bool) {
	return extractSignatureMetadataForLang(sig, lang)
}
