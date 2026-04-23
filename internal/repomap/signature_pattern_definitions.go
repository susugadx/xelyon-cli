package repomap

import "regexp"

type signaturePattern struct {
	re   *regexp.Regexp
	kind string
	lang string // 対象言語（"go","js","py","rs","java","rb","php","c","swift","scala","sh"、"" で全言語）
}

var signaturePatterns = collectSignaturePatterns()

func collectSignaturePatterns() []signaturePattern {
	var patterns []signaturePattern
	patterns = append(patterns, signaturePatternsForGo()...)
	patterns = append(patterns, signaturePatternsForJavaScript()...)
	patterns = append(patterns, signaturePatternsForPython()...)
	patterns = append(patterns, signaturePatternsForRust()...)
	patterns = append(patterns, signaturePatternsForJVM()...)
	patterns = append(patterns, signaturePatternsForRuby()...)
	patterns = append(patterns, signaturePatternsForPHP()...)
	patterns = append(patterns, signaturePatternsForCSharp()...)
	patterns = append(patterns, signaturePatternsForC()...)
	patterns = append(patterns, signaturePatternsForSwift()...)
	patterns = append(patterns, signaturePatternsForScala()...)
	patterns = append(patterns, signaturePatternsForElixir()...)
	patterns = append(patterns, signaturePatternsForLua()...)
	patterns = append(patterns, signaturePatternsForShell()...)
	return patterns
}
