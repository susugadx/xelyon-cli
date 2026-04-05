package file

import "testing"

func TestParsePath_NoRange(t *testing.T) {
	path, start, end := parsePath("internal/tools/file/read.go")
	if path != "internal/tools/file/read.go" || start != 0 || end != 0 {
		t.Errorf("Expected (internal/tools/file/read.go, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_StartOnly(t *testing.T) {
	path, start, end := parsePath("file.go:10")
	if path != "file.go" || start != 10 || end != 0 {
		t.Errorf("Expected (file.go, 10, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_StartEnd(t *testing.T) {
	path, start, end := parsePath("file.go:10-20")
	if path != "file.go" || start != 10 || end != 20 {
		t.Errorf("Expected (file.go, 10, 20), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_InvalidSuffix(t *testing.T) {
	path, start, end := parsePath("file.go:abc")
	if path != "file.go:abc" || start != 0 || end != 0 {
		t.Errorf("Expected (file.go:abc, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_InvalidRange(t *testing.T) {
	path, start, end := parsePath("file.go:abc-def")
	if path != "file.go:abc-def" || start != 0 || end != 0 {
		t.Errorf("Expected (file.go:abc-def, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}
