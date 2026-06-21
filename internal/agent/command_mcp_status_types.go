package agent

const (
	mcpStatusSampleLimit  = 10
	mcpStatusSurfaceLimit = 5
)

type mcpStatusToolSample struct {
	exportedName string
	approval     string
	reason       string
}
