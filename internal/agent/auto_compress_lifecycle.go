package agent

type autoCompressionTurnState struct {
	attempted  bool
	compressed bool
}

func newAutoCompressionTurnState() *autoCompressionTurnState {
	return &autoCompressionTurnState{}
}

func (s *autoCompressionTurnState) recordAttempt(compressed bool) {
	if s == nil {
		return
	}
	s.attempted = true
	if compressed {
		s.compressed = true
	}
}

func (s *autoCompressionTurnState) compressedThisTurn() bool {
	return s != nil && s.compressed
}

func (s *autoCompressionTurnState) attemptedThisTurn() bool {
	return s != nil && s.attempted
}

func (a *Agent) maybeAutoCompressAfterTurn(state *autoCompressionTurnState) bool {
	if state != nil && state.attemptedThisTurn() {
		return false
	}
	result := a.maybeAutoCompressAttempt()
	if result.attempted && state != nil {
		state.recordAttempt(result.compressed)
	}
	return result.compressed
}
