package repomap

import "testing"

func TestPatterns_CommentExclusion(t *testing.T) {
	if matchesSymbolPattern("main.go", "// func Build() error") {
		t.Fatal("commented Go definition should not match")
	}
	if matchesSymbolPattern("tasks.py", "# def build_map():") {
		t.Fatal("commented Python definition should not match")
	}
	if matchesSymbolPattern("build.sh", "# function build_map") {
		t.Fatal("commented Shell definition should not match")
	}
}
