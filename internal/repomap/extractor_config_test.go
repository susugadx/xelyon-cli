//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// ================== Bash Tests ==================

func TestExtractBashFunction(t *testing.T) {
	content := `#!/bin/bash

function hello() {
    echo "Hello, $1"
}

greet() {
    echo "Hi there!"
}

main() {
    hello "World"
    greet
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "script.sh", content)
	testFile := filepath.Join(tmpDir, "script.sh")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数が抽出されているか確認
	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 Bash function, got %d", len(fileSymbols.Symbols))
	}

	// hello関数の検証
	found := false
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "hello" && sym.Kind == "function" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'hello' function")
	}
}

// ================== Makefile Tests ==================

func TestExtractMakefileTargets(t *testing.T) {
	content := `BINARY=myapp

.PHONY: all clean test

all: build

build:
	go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)

install: build
	cp $(BINARY) /usr/local/bin/
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Makefile", content)
	testFile := filepath.Join(tmpDir, "Makefile")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// ターゲットが抽出されているか確認（.PHONYは除外）
	if len(fileSymbols.Symbols) < 4 {
		t.Fatalf("Expected at least 4 Makefile targets, got %d", len(fileSymbols.Symbols))
	}

	// build, test, clean, install が含まれているか確認
	targets := make(map[string]bool)
	for _, sym := range fileSymbols.Symbols {
		targets[sym.Name] = true
	}

	expected := []string{"all", "build", "test", "clean", "install"}
	for _, name := range expected {
		if !targets[name] {
			t.Errorf("Expected to find target '%s'", name)
		}
	}
}

// ================== Markdown Tests ==================

func TestExtractMarkdownHeadings(t *testing.T) {
	content := `# Main Title

Some introductory text.

## Getting Started

First steps here.

### Installation

Installation instructions.

## Usage

How to use the tool.

### Basic Commands

Command examples.
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "README.md", content)
	testFile := filepath.Join(tmpDir, "README.md")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 見出しが抽出されているか確認
	if len(fileSymbols.Symbols) < 3 {
		t.Fatalf("Expected at least 3 Markdown headings, got %d", len(fileSymbols.Symbols))
	}

	// h1, h2, h3 が含まれているか確認
	kindCounts := make(map[string]int)
	for _, sym := range fileSymbols.Symbols {
		kindCounts[sym.Kind]++
	}

	if kindCounts["h1"] < 1 {
		t.Errorf("Expected at least 1 h1 heading, got %d", kindCounts["h1"])
	}
	if kindCounts["h2"] < 1 {
		t.Errorf("Expected at least 1 h2 heading, got %d", kindCounts["h2"])
	}
}

// ================== Dockerfile Tests ==================

func TestExtractDockerfileInstructions(t *testing.T) {
	content := `FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest

COPY --from=builder /app/main /main

CMD ["/main"]
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Dockerfile", content)
	testFile := filepath.Join(tmpDir, "Dockerfile")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// FROM, RUN, CMD が抽出されているか確認
	kindCounts := make(map[string]int)
	for _, sym := range fileSymbols.Symbols {
		kindCounts[sym.Kind]++
	}

	if kindCounts["from"] < 2 {
		t.Errorf("Expected at least 2 FROM instructions, got %d", kindCounts["from"])
	}
	if kindCounts["run"] < 2 {
		t.Errorf("Expected at least 2 RUN instructions, got %d", kindCounts["run"])
	}
	if kindCounts["cmd"] < 1 {
		t.Errorf("Expected at least 1 CMD instruction, got %d", kindCounts["cmd"])
	}
}
