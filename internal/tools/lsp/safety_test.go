package lsp

import (
	"strings"
	"testing"
)

func TestExtractSymbolsFromContent_Go(t *testing.T) {
	content := `package main

func main() {
	fmt.Println("hello")
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// handle request
}

type Server struct {
	port int
}

func (s *Server) Start() error {
	return nil
}

var globalConfig Config

const MaxRetries = 3
`

	symbols := ExtractSymbolsFromContent(content, "go")

	// Expected symbols: main, handleRequest, Server, Start, globalConfig, MaxRetries
	expected := map[string]string{
		"main":          "function",
		"handleRequest": "function",
		"Server":        "type",
		"Start":         "function",
		"globalConfig":  "variable",
		"MaxRetries":    "variable",
	}

	if len(symbols) < len(expected) {
		t.Errorf("Expected at least %d symbols, got %d", len(expected), len(symbols))
	}

	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
		if expectedKind, ok := expected[sym.Name]; ok {
			if sym.Kind != expectedKind {
				t.Errorf("Symbol %s: expected kind %s, got %s", sym.Name, expectedKind, sym.Kind)
			}
		}
	}

	for name := range expected {
		if !found[name] {
			t.Errorf("Expected symbol %s not found", name)
		}
	}
}

func TestExtractSymbolsFromContent_TypeScript(t *testing.T) {
	content := `export function fetchData() {
	return fetch('/api/data');
}

export class UserService {
	private users: User[];

	getUser(id: string): User {
		return this.users.find(u => u.id === id);
	}
}

export interface User {
	id: string;
	name: string;
}

export const API_URL = 'https://api.example.com';

export type UserId = string;
`

	symbols := ExtractSymbolsFromContent(content, "typescript")

	expected := []string{"fetchData", "UserService", "User", "API_URL", "UserId"}

	if len(symbols) < len(expected) {
		t.Errorf("Expected at least %d symbols, got %d", len(expected), len(symbols))
	}

	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Errorf("Expected symbol %s not found", name)
		}
	}
}

func TestExtractSymbolsFromContent_Python(t *testing.T) {
	content := `def process_data(data):
    return data.upper()

class DataProcessor:
    def __init__(self):
        self.data = []

    def process(self):
        pass
`

	symbols := ExtractSymbolsFromContent(content, "python")

	expected := []string{"process_data", "DataProcessor", "__init__", "process"}

	if len(symbols) < len(expected) {
		t.Errorf("Expected at least %d symbols, got %d", len(expected), len(symbols))
	}

	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Errorf("Expected symbol %s not found", name)
		}
	}
}

func TestExtractSymbolsFromContent_Rust(t *testing.T) {
	content := `fn main() {
    println!("Hello, world!");
}

pub struct Server {
    port: u16,
}

impl Server {
    pub fn new(port: u16) -> Self {
        Server { port }
    }

    fn start(&self) -> Result<(), Error> {
        Ok(())
    }
}

pub enum Status {
    Active,
    Inactive,
}

pub trait Handler {
    fn handle(&self);
}

pub type Port = u16;
`

	symbols := ExtractSymbolsFromContent(content, "rust")

	expected := []string{"main", "Server", "new", "start", "Status", "Handler", "Port"}

	if len(symbols) < len(expected) {
		t.Errorf("Expected at least %d symbols, got %d", len(expected), len(symbols))
	}

	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Errorf("Expected symbol %s not found", name)
		}
	}
}

func TestExtractSymbolsFromContent_UnknownLanguage(t *testing.T) {
	content := "some content"
	symbols := ExtractSymbolsFromContent(content, "unknown")

	if symbols != nil {
		t.Errorf("Expected nil for unknown language, got %v", symbols)
	}
}

func TestGetExternalReferences(t *testing.T) {
	refs := []ReferenceInfo{
		{Symbol: "handleUser", FilePath: "/project/main.go", Line: 45, IsLocal: false},
		{Symbol: "handleUser", FilePath: "/project/handler.go", Line: 10, IsLocal: true},
		{Symbol: "handleUser", FilePath: "/project/routes.go", Line: 23, IsLocal: false},
	}

	external := GetExternalReferences(refs)

	if len(external) != 2 {
		t.Errorf("Expected 2 external references, got %d", len(external))
	}

	for _, ref := range external {
		if ref.IsLocal {
			t.Errorf("Expected external reference, got local: %v", ref)
		}
	}
}

func TestFormatReferenceWarning_NoExternal(t *testing.T) {
	refs := []ReferenceInfo{
		{Symbol: "handleUser", FilePath: "/project/handler.go", Line: 10, IsLocal: true},
	}

	warning := FormatReferenceWarning(refs)

	if warning != "" {
		t.Errorf("Expected empty warning for no external refs, got: %s", warning)
	}
}

func TestFormatReferenceWarning_WithExternal(t *testing.T) {
	refs := []ReferenceInfo{
		{Symbol: "handleUser", FilePath: "/project/main.go", Line: 45, IsLocal: false},
		{Symbol: "handleUser", FilePath: "/project/routes.go", Line: 23, IsLocal: false},
		{Symbol: "UserHandler", FilePath: "/project/main.go", Line: 67, IsLocal: false},
	}

	warning := FormatReferenceWarning(refs)

	if warning == "" {
		t.Error("Expected warning message, got empty string")
	}

	if !strings.Contains(warning, "handleUser") {
		t.Error("Warning should contain 'handleUser'")
	}

	if !strings.Contains(warning, "UserHandler") {
		t.Error("Warning should contain 'UserHandler'")
	}

	if !strings.Contains(warning, ":45") {
		t.Error("Warning should contain line number :45")
	}
}

func TestDetectKind(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{`func\s+(\w+)\s*\(`, "function"},
		{`function\s+(\w+)\s*\(`, "function"},
		{`def\s+(\w+)\s*\(`, "function"},
		{`fn\s+(\w+)\s*\(`, "function"},
		{`class\s+(\w+)`, "type"},
		{`struct\s+(\w+)`, "type"},
		{`interface\s+(\w+)`, "type"},
		{`trait\s+(\w+)`, "type"},
		{`type\s+(\w+)\s*=`, "type"},
		{`const\s+(\w+)\s*=`, "variable"},
		{`var\s+(\w+)\s+`, "variable"},
		{`let\s+(\w+)\s*=`, "variable"},
		{`enum\s+(\w+)`, "enum"},
		{`something_else`, "symbol"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := detectKind(tt.pattern)
			if result != tt.expected {
				t.Errorf("detectKind(%s) = %s, want %s", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestExtractSymbolsFromOldStr(t *testing.T) {
	oldStr := `func handleUser(w http.ResponseWriter, r *http.Request) {
	// handle user
}

type UserHandler struct {
	db *Database
}`

	symbols := ExtractSymbolsFromOldStr(oldStr, "go")

	if len(symbols) != 2 {
		t.Errorf("Expected 2 symbols, got %d", len(symbols))
	}

	names := make(map[string]bool)
	for _, sym := range symbols {
		names[sym.Name] = true
	}

	if !names["handleUser"] {
		t.Error("Expected 'handleUser' symbol")
	}
	if !names["UserHandler"] {
		t.Error("Expected 'UserHandler' symbol")
	}
}
