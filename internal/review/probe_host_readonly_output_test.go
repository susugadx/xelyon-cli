package review

import "testing"

func TestCappedOutput_Write_Unlimited(t *testing.T) {
	out := newCappedOutput(0)

	n, err := out.Write([]byte("abc"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 3 {
		t.Fatalf("Write() n = %d, want 3", n)
	}
	if out.String() != "abc" {
		t.Fatalf("String() = %q, want %q", out.String(), "abc")
	}
	if out.Truncated() {
		t.Fatal("Truncated() = true, want false")
	}
}

func TestCappedOutput_Write_TruncatesAtBoundary(t *testing.T) {
	out := newCappedOutput(4)

	_, _ = out.Write([]byte("ab"))
	n, err := out.Write([]byte("cdef"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 4 {
		t.Fatalf("Write() n = %d, want 4", n)
	}
	if out.String() != "abcd" {
		t.Fatalf("String() = %q, want %q", out.String(), "abcd")
	}
	if !out.Truncated() {
		t.Fatal("Truncated() = false, want true")
	}
}

func TestCappedOutput_Write_WhenAlreadyFull(t *testing.T) {
	out := newCappedOutput(1)
	_, _ = out.Write([]byte("x"))

	n, err := out.Write([]byte("yz"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 2 {
		t.Fatalf("Write() n = %d, want 2", n)
	}
	if out.String() != "x" {
		t.Fatalf("String() = %q, want %q", out.String(), "x")
	}
	if !out.Truncated() {
		t.Fatal("Truncated() = false, want true")
	}
}
