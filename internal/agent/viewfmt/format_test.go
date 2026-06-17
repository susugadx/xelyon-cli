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

func TestTokens(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: "0"},
		{name: "below thousand", n: 999, want: "999"},
		{name: "thousand", n: 1000, want: "1.0k"},
		{name: "rounded thousands", n: 12345, want: "12.3k"},
		{name: "below million uses k", n: 999999, want: "1000.0k"},
		{name: "million", n: 1000000, want: "1.0M"},
		{name: "rounded millions", n: 12345678, want: "12.3M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tokens(tt.n); got != tt.want {
				t.Fatalf("Tokens(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestFileSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "zero", bytes: 0, want: "0 B"},
		{name: "bytes", bytes: 512, want: "512 B"},
		{name: "below kilobyte", bytes: 1023, want: "1023 B"},
		{name: "kilobytes", bytes: 1536, want: "1.5 KB"},
		{name: "megabytes", bytes: 1048576, want: "1.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileSize(tt.bytes); got != tt.want {
				t.Fatalf("FileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
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
