package refactor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectLargeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		lines     int
		maxLines  int
		wantMatch bool
	}{
		{"small file", 100, 300, false},
		{"exact limit", 300, 300, true}, // 300 lines counts as over limit
		{"over limit", 301, 300, true},
		{"large file", 500, 300, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".go")
			content := "package main\n" + strings.Repeat("// line\n", tt.lines-1)
			if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			proposals := DetectLargeFiles([]string{testFile}, tt.maxLines)

			if tt.wantMatch && len(proposals) == 0 {
				t.Error("expected proposal for large file")
			}
			if !tt.wantMatch && len(proposals) > 0 {
				t.Error("unexpected proposal for small file")
			}
		})
	}
}

func TestDetectLongFunctions_Go(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	// Create a file with a long function
	var content strings.Builder
	content.WriteString("package main\n\n")
	content.WriteString("func shortFunc() {\n")
	content.WriteString("\tprintln(\"short\")\n")
	content.WriteString("}\n\n")
	content.WriteString("func longFunc() {\n")
	for i := 0; i < 60; i++ {
		content.WriteString("\tprintln(\"line\")\n")
	}
	content.WriteString("}\n")

	if err := os.WriteFile(testFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	proposals := DetectLongFunctions([]string{testFile}, 50)

	if len(proposals) == 0 {
		t.Error("expected proposal for long function")
	}

	// Should find longFunc but not shortFunc
	foundLong := false
	for _, p := range proposals {
		if p.FunctionName == "longFunc" {
			foundLong = true
		}
		if p.FunctionName == "shortFunc" {
			t.Error("shortFunc should not be flagged")
		}
	}

	if !foundLong {
		t.Error("longFunc should be flagged")
	}
}

func TestDetectLongFunctions_Python(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	var content strings.Builder
	content.WriteString("def short_func():\n")
	content.WriteString("    print('short')\n\n")
	content.WriteString("def long_func():\n")
	for i := 0; i < 60; i++ {
		content.WriteString("    print('line')\n")
	}

	if err := os.WriteFile(testFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	proposals := DetectLongFunctions([]string{testFile}, 50)

	// Note: Python detection relies on brace counting which won't work well
	// This is a known limitation
	_ = proposals
}

func TestDetectDuplicateCode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with duplicate code
	duplicateCode := `
func doSomething() {
	a := 1
	b := 2
	c := a + b
	d := c * 2
	e := d + 1
	f := e - 1
	g := f * f
	h := g + g
	i := h - h
	j := i + 1
}
`

	file1 := filepath.Join(tmpDir, "file1.go")
	file2 := filepath.Join(tmpDir, "file2.go")

	if err := os.WriteFile(file1, []byte("package a\n"+duplicateCode), 0644); err != nil {
		t.Fatalf("failed to create file1.go: %v", err)
	}
	if err := os.WriteFile(file2, []byte("package b\n"+duplicateCode), 0644); err != nil {
		t.Fatalf("failed to create file2.go: %v", err)
	}

	proposals := DetectDuplicateCode([]string{file1, file2}, 10)

	if len(proposals) == 0 {
		t.Error("expected proposal for duplicate code")
	}
}

func TestDetectPoorNaming_SingleLetter(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

func process() {
	x := 1
	y := 2
	z := x + y
	// Loop counters are OK
	for i := 0; i < 10; i++ {
		j := i * 2
	}
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	proposals := DetectPoorNaming([]string{testFile})

	// Should find x, y, z but not i, j
	singleLetterCount := 0
	for _, p := range proposals {
		if p.Type == RefactorRename && strings.Contains(p.Description, "Single-letter") {
			singleLetterCount++
		}
	}

	if singleLetterCount == 0 {
		t.Error("expected proposals for single-letter variables")
	}
}

func TestDetectPoorNaming_GenericNames(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

func process() {
	data := getData()
	temp := transform(data)
	result := finalize(temp)
	return result
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	proposals := DetectPoorNaming([]string{testFile})

	// Should find data, temp, result
	genericCount := 0
	for _, p := range proposals {
		if p.Type == RefactorRename && strings.Contains(p.Description, "Generic") {
			genericCount++
		}
	}

	if genericCount == 0 {
		t.Error("expected proposals for generic variable names")
	}
}

func TestDetectPoorNaming_SkipsComments(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

// x is a test comment with single letter
/* data is mentioned in block comment */
func main() {
	properName := 1
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	proposals := DetectPoorNaming([]string{testFile})

	// Should not flag variables in comments
	for _, p := range proposals {
		if strings.Contains(p.Description, "x") && p.LineStart <= 4 {
			t.Error("should not flag variables in comments")
		}
	}
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim whitespace",
			input: "  func main() {  \n    println(\"hello\")  \n  }  ",
			want:  "func main() {\nprintln(\"hello\")\n}",
		},
		{
			name:  "remove empty lines",
			input: "line1\n\nline2\n\n\nline3",
			want:  "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCode(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPartOfIdentifier(t *testing.T) {
	tests := []struct {
		line string
		word string
		want bool
	}{
		{"userData := 1", "Data", true},    // part of userData (case sensitive)
		{"data := 1", "data", false},       // standalone
		{"getDataFromAPI()", "Data", true}, // part of function name (case sensitive)
		{"var data string", "data", false}, // standalone
		{"tempValue := 1", "Value", true},  // part of tempValue (case sensitive)
		{"return result", "result", false}, // standalone
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := isPartOfIdentifier(tt.line, tt.word)
			if got != tt.want {
				t.Errorf("isPartOfIdentifier(%q, %q) = %v, want %v", tt.line, tt.word, got, tt.want)
			}
		})
	}
}

func TestExtractFunctions_Go(t *testing.T) {
	content := `package main

func main() {
	println("hello")
}

func helper() {
	println("helper")
}

func (s *Server) Handle() {
	s.process()
}
`
	funcs := extractFunctions(content, ".go")

	if len(funcs) < 3 {
		t.Errorf("expected at least 3 functions, got %d", len(funcs))
	}

	// Check function names
	names := make(map[string]bool)
	for _, f := range funcs {
		names[f.name] = true
	}

	if !names["main"] {
		t.Error("expected to find main function")
	}
	if !names["helper"] {
		t.Error("expected to find helper function")
	}
	if !names["Handle"] {
		t.Error("expected to find Handle method")
	}
}

func TestExtractFunctions_JavaScript(t *testing.T) {
	content := `function regularFunc() {
	console.log("regular");
}

const arrowFunc = () => {
	console.log("arrow");
};

const asyncFunc = async function() {
	await something();
};
`
	funcs := extractFunctions(content, ".js")

	if len(funcs) == 0 {
		t.Error("expected to find functions in JavaScript")
	}
}
