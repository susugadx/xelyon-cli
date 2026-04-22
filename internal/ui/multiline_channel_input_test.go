package ui

import (
	"strings"
	"testing"
)

// newTestReaderWithChannel creates a MultilineReader with channels for testing.
func newTestReaderWithChannel() (*MultilineReader, chan byte, chan error) {
	r := NewMultilineReader(strings.NewReader(""))
	r.byteChan = make(chan byte, 4096)
	r.errChan = make(chan error, 1)
	r.rawModeInit = true
	return r, r.byteChan, r.errChan
}

// feedBytes sends bytes to the channel followed by Enter (\r).
func feedBytes(ch chan byte, data []byte) {
	for _, b := range data {
		ch <- b
	}
	ch <- '\r' // Enter to finish
}

func TestReadLineFromChannel_BackspaceCases(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "ASCII 1文字削除",
			input: []byte{'a', 'b', 'c', 0x7f},
			want:  "ab",
		},
		{
			name:  "ASCII 複数削除",
			input: []byte{'h', 'e', 'l', 'l', 'o', 0x7f, 0x7f, 0x7f},
			want:  "he",
		},
		{
			name:  "空バッファで削除しても継続",
			input: []byte{0x7f, 0x7f, 'a'},
			want:  "a",
		},
		{
			name:  "全削除",
			input: []byte{'a', 'b', 'c', 0x7f, 0x7f, 0x7f},
			want:  "",
		},
		{
			name:  "UTF-8 + ASCII を削除",
			input: []byte{0xE3, 0x81, 0x82, 'a', 0x7f}, // あa -> あ
			want:  "あ",
		},
		{
			name:  "UTF-8 マルチバイト削除",
			input: []byte{'a', 0xE3, 0x81, 0x82, 0x7f}, // aあ -> a
			want:  "a",
		},
		{
			name:  "BS(0x08) も削除として扱う",
			input: []byte{'a', 'b', 0x08},
			want:  "a",
		},
		{
			name: "ASCIIとUTF-8混在",
			input: []byte{
				'h',
				0xE3, 0x81, 0x82, // あ
				'i',
				0xE3, 0x81, 0x86, // う
				0x7f, // delete う
				0x7f, // delete i
			},
			want: "hあ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ch, _ := newTestReaderWithChannel()
			go feedBytes(ch, tt.input)

			got, err := r.readLineFromChannel()
			if err != nil {
				t.Fatalf("readLineFromChannel() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("readLineFromChannel() = %q, want %q", got, tt.want)
			}
		})
	}
}
