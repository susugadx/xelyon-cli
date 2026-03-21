package api

import (
	"bytes"
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRuntimeFromContext(t *testing.T) {
	t.Run("context with runtime returns that runtime", func(t *testing.T) {
		buf := &bytes.Buffer{}
		runtime := ui.NewRuntime(nil, buf, buf)
		ctx := ui.WithRuntime(context.Background(), runtime)

		got := RuntimeFromContext(ctx)
		if got != runtime {
			t.Error("RuntimeFromContext() should return the injected runtime")
		}
	})

	t.Run("context without runtime returns default", func(t *testing.T) {
		ctx := context.Background()

		got := RuntimeFromContext(ctx)
		if got == nil {
			t.Error("RuntimeFromContext() should return non-nil default runtime")
		}
	})
}

func TestRuntimeOrDefault(t *testing.T) {
	t.Run("non-nil runtime is returned as-is", func(t *testing.T) {
		buf := &bytes.Buffer{}
		runtime := ui.NewRuntime(nil, buf, buf)

		got := RuntimeOrDefault(runtime)
		if got != runtime {
			t.Error("RuntimeOrDefault() should return the provided runtime")
		}
	})

	t.Run("nil runtime returns default", func(t *testing.T) {
		got := RuntimeOrDefault(nil)
		if got == nil {
			t.Error("RuntimeOrDefault(nil) should return non-nil default runtime")
		}
	})
}

func TestOutputWriterFromContext(t *testing.T) {
	t.Run("context with runtime returns runtime output", func(t *testing.T) {
		buf := &bytes.Buffer{}
		runtime := ui.NewRuntime(nil, buf, nil)
		ctx := ui.WithRuntime(context.Background(), runtime)

		got := OutputWriterFromContext(ctx)
		if got == nil {
			t.Error("OutputWriterFromContext() should return non-nil writer")
		}

		// writer に書き込みが可能であることを確認
		n, err := got.Write([]byte("test"))
		if err != nil {
			t.Errorf("OutputWriterFromContext() writer.Write() error = %v", err)
		}
		if n != 4 {
			t.Errorf("OutputWriterFromContext() writer.Write() = %d, want 4", n)
		}
		if buf.String() != "test" {
			t.Errorf("OutputWriterFromContext() wrote to wrong buffer, got %q", buf.String())
		}
	})

	t.Run("context without runtime returns non-nil writer", func(t *testing.T) {
		ctx := context.Background()

		got := OutputWriterFromContext(ctx)
		if got == nil {
			t.Error("OutputWriterFromContext() should return non-nil writer even without runtime")
		}
	})
}

func TestSpinnerFromContext(t *testing.T) {
	t.Run("context with runtime that has spinner", func(t *testing.T) {
		buf := &bytes.Buffer{}
		runtime := ui.NewRuntime(nil, buf, buf)
		spinner := runtime.NewSpinner()
		runtime.SetSpinner(spinner)
		ctx := ui.WithRuntime(context.Background(), runtime)

		got := SpinnerFromContext(ctx)
		if got != spinner {
			t.Error("SpinnerFromContext() should return the spinner set on runtime")
		}
	})

	t.Run("context with runtime without spinner", func(t *testing.T) {
		buf := &bytes.Buffer{}
		runtime := ui.NewRuntime(nil, buf, buf)
		ctx := ui.WithRuntime(context.Background(), runtime)

		got := SpinnerFromContext(ctx)
		if got != nil {
			t.Error("SpinnerFromContext() should return nil when no spinner is set")
		}
	})

	t.Run("context without runtime returns nil", func(t *testing.T) {
		ctx := context.Background()

		got := SpinnerFromContext(ctx)
		// デフォルト runtime の spinner は nil
		if got != nil {
			t.Error("SpinnerFromContext() should return nil for default runtime without spinner")
		}
	})
}
