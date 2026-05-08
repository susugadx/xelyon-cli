package review

import (
	"reflect"
	"testing"
)

func TestParseProbeShortOptions(t *testing.T) {
	tests := []struct {
		name         string
		arg          string
		valueOptions map[byte]struct{}
		wantTokens   []probeShortOptionToken
		wantOK       bool
	}{
		{
			name:         "cluster with attached value option",
			arg:          "-nOcat",
			valueOptions: byteSet('O'),
			wantTokens: []probeShortOptionToken{
				{name: 'n'},
				{name: 'O', hasAttachedValue: true, attachedValue: "cat"},
			},
			wantOK: true,
		},
		{
			name:         "cluster with pattern file value",
			arg:          "-nf/usr/share/patterns",
			valueOptions: byteSet('f'),
			wantTokens: []probeShortOptionToken{
				{name: 'n'},
				{name: 'f', hasAttachedValue: true, attachedValue: "/usr/share/patterns"},
			},
			wantOK: true,
		},
		{
			name:         "attached value stops later chars from becoming flags",
			arg:          "-gL",
			valueOptions: byteSet('g'),
			wantTokens: []probeShortOptionToken{
				{name: 'g', hasAttachedValue: true, attachedValue: "L"},
			},
			wantOK: true,
		},
		{
			name:         "single value option consumes next",
			arg:          "-f",
			valueOptions: byteSet('f'),
			wantTokens: []probeShortOptionToken{
				{name: 'f', consumesNext: true},
			},
			wantOK: true,
		},
		{
			name:   "long option is not a short cluster",
			arg:    "--file=/tmp/patterns",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTokens, gotOK := parseProbeShortOptions(tt.arg, tt.valueOptions)
			if gotOK != tt.wantOK {
				t.Fatalf("parseProbeShortOptions(%q) ok = %v, want %v", tt.arg, gotOK, tt.wantOK)
			}
			if !reflect.DeepEqual(gotTokens, tt.wantTokens) {
				t.Fatalf("parseProbeShortOptions(%q) tokens = %#v, want %#v", tt.arg, gotTokens, tt.wantTokens)
			}
		})
	}
}
