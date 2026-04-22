package tools

// jsonObjectEndScanner は JSON オブジェクト終端検出の状態を保持する。
type jsonObjectEndScanner struct {
	depth    int
	inString bool
	escaped  bool
}

func findBalancedJSONObjectEnd(response string, start int) int {
	scanner := jsonObjectEndScanner{}
	for i := start; i < len(response); i++ {
		if scanner.consume(response[i]) {
			return i + 1
		}
	}
	return -1
}

func (s *jsonObjectEndScanner) consume(ch byte) bool {
	if s.escaped {
		s.escaped = false
		return false
	}

	if ch == '\\' && s.inString {
		s.escaped = true
		return false
	}

	if ch == '"' {
		s.inString = !s.inString
		return false
	}

	if s.inString {
		return false
	}

	switch ch {
	case '{':
		s.depth++
	case '}':
		s.depth--
		if s.depth == 0 {
			return true
		}
	}

	return false
}
