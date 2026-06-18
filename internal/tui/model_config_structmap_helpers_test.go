package tui

import "testing"

func enterConfigStructMapEdit(t *testing.T, m *Model, path string) {
	t.Helper()
	selectConfigFieldByPath(t, m, path)
	*m = sendConfigKey(*m, "enter")
	if got := configSnapshot(t, *m).editMode; got != editStructMap {
		t.Fatalf("editMode = %d, want %d", got, editStructMap)
	}
}

func enterStructMapEdit(t *testing.T, path string) Model {
	t.Helper()
	m := newConfigTestModel()
	enterConfigStructMapEdit(t, &m, path)
	return m
}

func selectConfigStructMapKey(t *testing.T, m *Model, key string) {
	t.Helper()
	cs := configTestScreen(t, *m)
	for i, candidate := range cs.editStructKeys {
		if candidate == key {
			cs.editStructIndex = i
			return
		}
	}
	t.Fatalf("key %q not found in editStructKeys", key)
}

func selectConfigStructMapKeyIndex(t *testing.T, m *Model, index int) string {
	t.Helper()
	cs := configTestScreen(t, *m)
	if index < 0 || index >= len(cs.editStructKeys) {
		t.Fatalf("editStructIndex %d out of range for %d keys", index, len(cs.editStructKeys))
	}
	cs.editStructIndex = index
	return cs.editStructKeys[index]
}

func enterStructMapEntryForKey(t *testing.T, path, key string) Model {
	t.Helper()
	m := enterStructMapEdit(t, path)
	selectConfigStructMapKey(t, &m, key)

	m = sendConfigKey(m, "enter")
	cs := configTestScreen(t, m)
	if !cs.editEntryActive {
		t.Fatalf("editEntryActive should be true for key %q", key)
	}
	if cs.editEntryKey != key {
		t.Fatalf("editEntryKey = %q, want %q", cs.editEntryKey, key)
	}
	return m
}

func selectConfigEntryField(t *testing.T, m *Model, name string) int {
	t.Helper()
	cs := configTestScreen(t, *m)
	for i, ef := range cs.editEntryFields {
		if ef.Name == name {
			cs.editEntryIndex = i
			return i
		}
	}
	t.Fatalf("entry field %q not found", name)
	return -1
}
