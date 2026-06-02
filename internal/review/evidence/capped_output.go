package evidence

import (
	"bytes"
	"sync"
)

type cappedOutput struct {
	maxBytes int64

	mu        sync.Mutex
	buf       bytes.Buffer
	truncated bool
}

func newCappedOutput(maxBytes int64) *cappedOutput {
	return &cappedOutput{
		maxBytes: maxBytes,
	}
}

func (c *cappedOutput) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxBytes <= 0 {
		_, _ = c.buf.Write(p)
		return len(p), nil
	}

	remaining := c.maxBytes - int64(c.buf.Len())
	if remaining <= 0 {
		c.truncated = true
		return len(p), nil
	}

	if int64(len(p)) <= remaining {
		_, _ = c.buf.Write(p)
		return len(p), nil
	}

	_, _ = c.buf.Write(p[:remaining])
	c.truncated = true
	return len(p), nil
}

func (c *cappedOutput) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func (c *cappedOutput) Truncated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncated
}
