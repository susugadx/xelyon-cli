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
	snapshot := configTestScreen(t, *m).Snapshot()
	for i, candidate := range snapshot.EditStructKeys {
		if candidate == key {
			selectConfigStructMapKeyIndex(t, m, i)
			return
		}
	}
	t.Fatalf("key %q not found in editStructKeys", key)
}

func selectConfigStructMapKeyIndex(t *testing.T, m *Model, index int) string {
	t.Helper()
	snapshot := configTestScreen(t, *m).Snapshot()
	if index < 0 || index >= len(snapshot.EditStructKeys) {
		t.Fatalf("editStructIndex %d out of range for %d keys", index, len(snapshot.EditStructKeys))
	}
	for snapshot.EditStructIndex < index {
		*m = sendConfigKey(*m, "down")
		snapshot = configTestScreen(t, *m).Snapshot()
	}
	for snapshot.EditStructIndex > index {
		*m = sendConfigKey(*m, "up")
		snapshot = configTestScreen(t, *m).Snapshot()
	}
	return snapshot.EditStructKeys[index]
}

func enterStructMapEntryForKey(t *testing.T, path, key string) Model {
	t.Helper()
	m := enterStructMapEdit(t, path)
	selectConfigStructMapKey(t, &m, key)

	m = sendConfigKey(m, "enter")
	snapshot := configTestScreen(t, m).Snapshot()
	if !snapshot.EditEntryActive {
		t.Fatalf("editEntryActive should be true for key %q", key)
	}
	if snapshot.EditEntryKey != key {
		t.Fatalf("editEntryKey = %q, want %q", snapshot.EditEntryKey, key)
	}
	return m
}

func selectConfigEntryField(t *testing.T, m *Model, name string) int {
	t.Helper()
	snapshot := configTestScreen(t, *m).Snapshot()
	for i, ef := range snapshot.EditEntryFields {
		if ef.Name == name {
			for snapshot.EditEntryIndex < i {
				*m = sendConfigKey(*m, "down")
				snapshot = configTestScreen(t, *m).Snapshot()
			}
			for snapshot.EditEntryIndex > i {
				*m = sendConfigKey(*m, "up")
				snapshot = configTestScreen(t, *m).Snapshot()
			}
			return i
		}
	}
	t.Fatalf("entry field %q not found", name)
	return -1
}
