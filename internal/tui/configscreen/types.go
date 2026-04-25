package configscreen

// Pane は config screen 内の3ペインのいずれかを表す。
type Pane int

const (
	// PaneCategory はカテゴリ一覧ペインを表す。
	PaneCategory Pane = iota
	// PaneField はフィールド一覧ペインを表す。
	PaneField
	// PaneDetail は詳細・編集ペインを表す。
	PaneDetail
)

// EditMode はフィールド編集中のモードを表す。
type EditMode int

const (
	// EditNone は編集していない状態を表す。
	EditNone EditMode = iota
	// EditInput は string/int/float の入力状態を表す。
	EditInput
	// EditSelect は select の選択状態を表す。
	EditSelect
	// EditSlice は []string のサブビュー状態を表す。
	EditSlice
	// EditStructMap は structmap のサブビュー状態を表す。
	EditStructMap
)

// SaveStatus は config screen の保存状態を表す。
type SaveStatus int

const (
	// StatusSaved は保存済み状態を表す。
	StatusSaved SaveStatus = iota
	// StatusModified は未保存変更がある状態を表す。
	StatusModified
	// StatusSaving は保存中状態を表す。
	StatusSaving
	// StatusFailed は保存失敗状態を表す。
	StatusFailed
)
