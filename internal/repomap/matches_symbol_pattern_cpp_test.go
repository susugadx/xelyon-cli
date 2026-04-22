package repomap

import "testing"

func TestPatterns_CppStructAndDefine(t *testing.T) {
	if !matchesSymbolPattern("lib.h", "struct MyConfig {") {
		t.Fatal("C struct pattern did not match")
	}
	if !matchesSymbolPattern("lib.h", "#define MAX_SIZE 1024") {
		t.Fatal("C #define pattern did not match")
	}
	if !matchesSymbolPattern("lib.cpp", "namespace utils {") {
		t.Fatal("C++ namespace pattern did not match")
	}
}

func TestPatterns_CppNoControlFlow(t *testing.T) {
	for _, line := range []string{
		"if (condition) {",
		"while (true) {",
		"for (int i = 0; i < n; i++) {",
		"switch (value) {",
		"return (result);",
	} {
		if matchesSymbolPattern("main.c", line) {
			t.Fatalf("control flow should not match: %s", line)
		}
	}
}
