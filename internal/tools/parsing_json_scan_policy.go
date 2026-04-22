package tools

type jsonToolCallScanDecision int

const (
	jsonToolCallScanDecisionContinue jsonToolCallScanDecision = iota
	jsonToolCallScanDecisionStop
)

type jsonToolCallScanErrorKind int

const (
	jsonToolCallScanErrorIncompleteJSONObject jsonToolCallScanErrorKind = iota + 1
)

type jsonToolCallScanError struct {
	kind  jsonToolCallScanErrorKind
	start int
}

type jsonToolCallScanErrorPolicy struct {
	onIncompleteJSONObject jsonToolCallScanDecision
}

func defaultJSONToolCallScanErrorPolicy() jsonToolCallScanErrorPolicy {
	// 既存挙動互換: 途中で不完全 JSON を見つけたら JSON 走査を終了する。
	return jsonToolCallScanErrorPolicy{
		onIncompleteJSONObject: jsonToolCallScanDecisionStop,
	}
}

func (p jsonToolCallScanErrorPolicy) Decide(err jsonToolCallScanError) jsonToolCallScanDecision {
	switch err.kind {
	case jsonToolCallScanErrorIncompleteJSONObject:
		return p.onIncompleteJSONObject
	default:
		return jsonToolCallScanDecisionStop
	}
}
