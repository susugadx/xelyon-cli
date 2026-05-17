package ledger

import "sync"

// Store は RuntimeTaskState を mutex 付きで保持する。
type Store struct {
	mu                        sync.Mutex
	state                     RuntimeTaskState
	editReadinessObservations []EditReadinessObservation
	repoRoot                  string
	invocationCWD             string
}

// NewStore は空の runtime task ledger を返す。
func NewStore() *Store {
	return NewStoreForInvocationCWD(defaultInvocationCWD())
}

// NewStoreWithRoot は repo root を明示した runtime task ledger を返す。
func NewStoreWithRoot(root string) *Store {
	return NewStoreWithWorkspace(root, root)
}

// NewStoreForInvocationCWD は起動 cwd から repo root を推定した runtime task ledger を返す。
func NewStoreForInvocationCWD(cwd string) *Store {
	cwd = normalizeRepoRoot(cwd)
	if cwd == "" {
		cwd = normalizeRepoRoot(defaultInvocationCWD())
	}
	if cwd == "" {
		return NewStoreWithWorkspace("", "")
	}
	return NewStoreWithWorkspace(repoRootForInvocationCWD(cwd), cwd)
}

// NewStoreWithWorkspace は repo root と起動 cwd を明示した runtime task ledger を返す。
func NewStoreWithWorkspace(root, invocationCWD string) *Store {
	root = normalizeRepoRoot(root)
	invocationCWD = normalizeRepoRoot(invocationCWD)
	if invocationCWD == "" {
		invocationCWD = root
	}
	if root == "" {
		root = invocationCWD
	}
	return &Store{repoRoot: root, invocationCWD: invocationCWD}
}

// Snapshot は現在の RuntimeTaskState を防御コピーで返す。
func (s *Store) Snapshot() RuntimeTaskState {
	if s == nil {
		return RuntimeTaskState{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.clone()
}

// Reset は runtime task ledger を空に戻す。
func (s *Store) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = RuntimeTaskState{}
	s.editReadinessObservations = nil
}

// Recorder は Store への書き込み入口を返す。
func (s *Store) Recorder() *Recorder {
	if s == nil {
		return nil
	}
	return &Recorder{store: s}
}
