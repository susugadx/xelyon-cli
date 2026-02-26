package embedding

import (
	"strings"
	"testing"
)

func TestChunkFile(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		content       string
		expectedCount int
		checkChunk    func(*testing.T, []Chunk)
	}{
		{
			name:          "Empty file",
			filePath:      "empty.go",
			content:       "",
			expectedCount: 0,
		},
		{
			name:          "Very small file",
			filePath:      "small.go",
			content:       "package main\n\nfunc main() {}\n",
			expectedCount: 1,
			checkChunk: func(t *testing.T, chunks []Chunk) {
				if chunks[0].StartLine != 1 || chunks[0].EndLine != 4 {
					t.Errorf("Unexpected bounds: %v", chunks[0])
				}
			},
		},
		{
			name:     "Go file with multiple funcs and top-level",
			filePath: "main.go",
			content: `package main

import "fmt"

func func1() {
	fmt.Println("1")
}

func func2() {
	fmt.Println("2")
}
`,
			expectedCount: 3, // top-level (package+import), func1, func2
			checkChunk: func(t *testing.T, chunks []Chunk) {
				hasTopLevel := false
				hasFunc1 := false
				hasFunc2 := false
				for _, c := range chunks {
					if c.BlockName == "" && strings.Contains(c.Content, "import") {
						hasTopLevel = true
					}
					if c.BlockName == "func func1" {
						hasFunc1 = true
					}
					if c.BlockName == "func func2" {
						hasFunc2 = true
					}
				}
				if !hasTopLevel || !hasFunc1 || !hasFunc2 {
					t.Errorf("Missing chunks. top:%v func1:%v func2:%v", hasTopLevel, hasFunc1, hasFunc2)
				}
			},
		},
		{
			name:     "Python file with indent blocks",
			filePath: "app.py",
			content: `import sys

def main():
    print("main")

class App:
    def run(self):
        pass
`,
			expectedCount: 3, // import, def main, class App
			checkChunk: func(t *testing.T, chunks []Chunk) {
				hasMain := false
				hasApp := false
				for _, c := range chunks {
					if c.BlockName == "def main" {
						hasMain = true
					}
					if c.BlockName == "class App" {
						hasApp = true
					}
				}
				if !hasMain || !hasApp {
					t.Errorf("Missing python chunks. main:%v app:%v", hasMain, hasApp)
				}
			},
		},
		{
			name:     "Large block splitting",
			filePath: "large.go",
			content: func() string {
				lines := []string{"package main", "", "func huge() {"}
				for i := 0; i < 150; i++ {
					lines = append(lines, "\t// line")
				}
				lines = append(lines, "}")
				return strings.Join(lines, "\n")
			}(),
			// Expected chunks: top-level + 4 splits of 'huge' (150 lines + header/footer = 152 lines -> 50, 50, 50, 2 with 10 line overlap -> 1-50, 41-90, 81-130, 121-152 -> 4 chunks for block)
			expectedCount: 5,
			checkChunk: func(t *testing.T, chunks []Chunk) {
				var hugeChunks []Chunk
				for _, c := range chunks {
					if c.BlockName == "func huge" {
						hugeChunks = append(hugeChunks, c)
					}
				}
				if len(hugeChunks) != 4 {
					t.Errorf("Expected 4 chunks for huge block, got %d", len(hugeChunks))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkFile(tt.filePath, tt.content)
			if len(chunks) != tt.expectedCount {
				t.Errorf("Expected %d chunks, got %d", tt.expectedCount, len(chunks))
			}
			if tt.checkChunk != nil {
				tt.checkChunk(t, chunks)
			}
		})
	}
}
