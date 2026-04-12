package viewfmt

import "testing"

func TestNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "small", n: 12, want: "12"},
		{name: "thousand", n: 1234, want: "1,234"},
		{name: "millions", n: 1234567, want: "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Number(tt.n); got != tt.want {
				t.Fatalf("Number(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestUSD(t *testing.T) {
	if got := USD(1.23456); got != "$1.2346" {
		t.Fatalf("USD() = %q, want %q", got, "$1.2346")
	}
}

func TestUSDWithSuffix(t *testing.T) {
	if got := USDWithSuffix(2.5); got != "$2.5000 USD" {
		t.Fatalf("USDWithSuffix() = %q, want %q", got, "$2.5000 USD")
	}
}

func TestFirstLine(t *testing.T) {
	if got := FirstLine("one\ntwo"); got != "one" {
		t.Fatalf("FirstLine() = %q, want %q", got, "one")
	}
	if got := FirstLine("single"); got != "single" {
		t.Fatalf("FirstLine() = %q, want %q", got, "single")
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Fatalf("Truncate() = %q, want %q", got, "hello")
	}
	if got := Truncate("hello world", 5); got != "hello..." {
		t.Fatalf("Truncate() = %q, want %q", got, "hello...")
	}
}
