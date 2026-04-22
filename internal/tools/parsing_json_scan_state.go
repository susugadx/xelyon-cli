package tools

type jsonToolCallScanState struct {
	searchFrom int
	done       bool
}

func newJSONToolCallScanState() *jsonToolCallScanState {
	return &jsonToolCallScanState{}
}

func (s *jsonToolCallScanState) IsDone() bool {
	return s.done
}

func (s *jsonToolCallScanState) MarkDone() {
	s.done = true
}

func (s *jsonToolCallScanState) SearchFrom() int {
	return s.searchFrom
}

func (s *jsonToolCallScanState) AdvanceTo(next int) {
	s.searchFrom = next
}

func (s *jsonToolCallScanState) AdvancePast(pos int) {
	s.searchFrom = pos + 1
}
