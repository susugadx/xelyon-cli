package tools

type xmlParamsParseStrategy interface {
	Parse(content string) xmlParamsParseOutcome
}

type xmlTagParamsParseStrategy struct{}

type xmlJSONParamsParseStrategy struct{}

type xmlParamsParser struct {
	strategies []xmlParamsParseStrategy
}

type xmlParamsParseOutcome struct {
	args map[string]string
	// handled は「この戦略が入力形式の責務 owner だったか」を表す。
	// true の場合、args が空でも他戦略へフォールバックしない。
	handled bool
}

func newDefaultXMLParamsParser() xmlParamsParser {
	return xmlParamsParser{
		strategies: []xmlParamsParseStrategy{
			xmlTagParamsParseStrategy{},
			xmlJSONParamsParseStrategy{},
		},
	}
}

// parseXMLParams は XML 内部コンテンツからパラメータを抽出する。
// パターン1: <args><param>value</param>...</args> （args ラッパーあり）
// パターン2: <param>value</param>... （args ラッパーなし）
// パターン3: {"key": "value"} （JSON 形式）
func parseXMLParams(content string) map[string]string {
	content = unwrapXMLArgsContent(content)
	parser := newDefaultXMLParamsParser()
	for _, strategy := range parser.strategies {
		outcome := strategy.Parse(content)
		if !outcome.handled {
			continue
		}
		if outcome.args == nil {
			return map[string]string{}
		}
		return outcome.args
	}
	return map[string]string{}
}

func handledXMLParamsOutcome(args map[string]string) xmlParamsParseOutcome {
	return xmlParamsParseOutcome{
		args:    args,
		handled: true,
	}
}

func unhandledXMLParamsOutcome() xmlParamsParseOutcome {
	return xmlParamsParseOutcome{
		args:    nil,
		handled: false,
	}
}
