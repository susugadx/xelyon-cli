package uiconfig

import (
	"io"

	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func NewRuntime(in io.Reader, out, err io.Writer) *uiruntime.Runtime {
	return uiruntime.NewRuntime(in, out, err)
}
