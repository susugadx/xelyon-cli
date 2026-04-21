package file

import "testing"

func TestResolveGatherContextFallbackRouteMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		query                 string
		allowImplicitBareFile bool
		want                  gatherContextFallbackRouteMode
	}{
		{
			name:                  "only_path_candidate_routes_any",
			query:                 "pkg/errors.go",
			allowImplicitBareFile: false,
			want:                  gatherContextFallbackRouteModeAny,
		},
		{
			name:                  "mixed_candidate_with_symbol_routes_none",
			query:                 "pkg/errors.go,Builder",
			allowImplicitBareFile: true,
			want:                  gatherContextFallbackRouteModeNone,
		},
		{
			name:                  "bare_file_requires_allow_implicit_flag",
			query:                 "sample.go",
			allowImplicitBareFile: false,
			want:                  gatherContextFallbackRouteModeNone,
		},
		{
			name:                  "bare_file_with_implicit_enabled_routes_read",
			query:                 "sample.go",
			allowImplicitBareFile: true,
			want:                  gatherContextFallbackRouteModeRead,
		},
		{
			name:                  "path_candidate_batch_routes_any_when_allowed",
			query:                 "pkg/errors.go,main.ex",
			allowImplicitBareFile: true,
			want:                  gatherContextFallbackRouteModeAny,
		},
		{
			name:                  "path_candidate_batch_routes_none_when_bare_name_disallowed",
			query:                 "pkg/errors.go,Makefile",
			allowImplicitBareFile: false,
			want:                  gatherContextFallbackRouteModeNone,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input, ok := parseDirectQueryInput(tt.query)
			if !ok {
				t.Fatalf("parseDirectQueryInput(%q) failed", tt.query)
			}
			got := resolveGatherContextFallbackRouteMode(input, tt.allowImplicitBareFile)
			if got != tt.want {
				t.Fatalf("resolveGatherContextFallbackRouteMode(%q, allow=%v)=%q, want %q", tt.query, tt.allowImplicitBareFile, got, tt.want)
			}
		})
	}
}
