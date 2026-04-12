package tui

import "testing"

func TestModel_StreamTextCarriageReturnRewriteKeepsTabAlignedSuffix(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "a\tb", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\rxy", Done: true})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[0]); got != "xy   b" {
		t.Fatalf("stripped raw line = %q, want %q", got, "xy   b")
	}
}

func TestModel_StreamTextCarriageReturnRewriteKeepsWideCharSuffix(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "あい", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\rxy", Done: true})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[0]); got != "xyい" {
		t.Fatalf("stripped raw line = %q, want %q", got, "xyい")
	}
}

func TestModel_StreamTextCarriageReturnRewriteTreatsCombiningClusterAsSingleCell(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "e\u0301x", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\rz", Done: true})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[0]); got != "zx" {
		t.Fatalf("stripped raw line = %q, want %q", got, "zx")
	}
}

func TestModel_StreamTextSplitCombiningMarkDoesNotPanicAndStaysCombined(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "e", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\u0301", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if got := m.rawLines[0]; got != "e\u0301" {
		t.Fatalf("rawLines[0] = %q, want %q", got, "e\u0301")
	}
	if got := stripANSI(m.rawLines[0]); got != "e\u0301" {
		t.Fatalf("stripped raw line = %q, want %q", got, "e\u0301")
	}
}
