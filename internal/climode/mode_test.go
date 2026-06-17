package climode

import (
	"strings"
	"testing"
)

func TestResolveOutputFormat(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		headless  bool
		want      string
		wantErr   string
	}{
		{name: "default empty", want: OutputFormatText},
		{name: "text normalized", flagValue: " TEXT ", want: OutputFormatText},
		{name: "json normalized", flagValue: "Json", want: OutputFormatJSON},
		{name: "headless overrides text", flagValue: "text", headless: true, want: OutputFormatJSON},
		{name: "invalid", flagValue: "yaml", wantErr: "invalid --output-format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveOutputFormat(tt.flagValue, tt.headless)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ResolveOutputFormat() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveOutputFormat() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveOutputFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequestResolve(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		want    Mode
		wantErr string
	}{
		{name: "no query opens interactive", request: Request{OutputFormat: OutputFormatText}, want: ModeInteractive},
		{name: "positional query is implicit once", request: Request{OutputFormat: OutputFormatText, HasQuery: true}, want: ModeOnce},
		{name: "explicit once", request: Request{OutputFormat: OutputFormatText, HasQuery: true, Once: true}, want: ModeOnce},
		{name: "json is headless", request: Request{OutputFormat: OutputFormatJSON, HasQuery: true}, want: ModeHeadless},
		{name: "resume without query", request: Request{OutputFormat: OutputFormatText, Resume: true}, want: ModeResume},
		{name: "image implicit once", request: Request{OutputFormat: OutputFormatText, HasImage: true}, want: ModeOnceImage},
		{name: "interactive image", request: Request{OutputFormat: OutputFormatText, HasQuery: true, HasImage: true, Interactive: true}, want: ModeInteractiveImage},
		{name: "quiet requires once", request: Request{OutputFormat: OutputFormatText, Quiet: true}, wantErr: "--quiet can only be used with one-shot execution"},
		{name: "once requires query or image", request: Request{OutputFormat: OutputFormatText, Once: true}, wantErr: "query argument is required"},
		{name: "image disallowed in json", request: Request{OutputFormat: OutputFormatJSON, HasImage: true}, wantErr: "--image cannot be used"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.request.Resolve()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}
