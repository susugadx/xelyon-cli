package tools

type testDisplayTool struct {
	name   string
	result string
}

func restoreRegistryTool(name string, orig Tool) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	if orig != nil {
		DefaultRegistry.tools[name] = orig
		return
	}
	delete(DefaultRegistry.tools, name)
}

func (t *testDisplayTool) Name() string {
	return t.name
}

func (t *testDisplayTool) Description() string {
	return "test tool"
}

func (t *testDisplayTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testDisplayTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	execCtx.Output().Println("INTERNAL STDOUT")
	return t.result, nil, nil
}

type testQuietTool struct {
	name   string
	result string
}

func (t *testQuietTool) Name() string {
	return t.name
}

func (t *testQuietTool) Description() string {
	return "quiet test tool"
}

func (t *testQuietTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testQuietTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	out := execCtx.Output()
	out.Green.Printf("QUIET COLOR OUTPUT\n")
	out.Printf("QUIET STDOUT OUTPUT\n")
	return t.result, nil, nil
}

type testOverlapQuietTool struct {
	started chan string
	release map[string]chan struct{}
}

func (t *testOverlapQuietTool) Name() string {
	return "overlap_quiet_test"
}

func (t *testOverlapQuietTool) Description() string {
	return "overlap quiet test tool"
}

func (t *testOverlapQuietTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testOverlapQuietTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	id := args["id"]
	t.started <- id
	<-t.release[id]
	execCtx.Output().Printf("OVERLAP STDOUT %s\n", id)
	return "result " + id, nil, nil
}

type testContextIsolationTool struct {
	started chan string
	release chan struct{}
}

type testCancelledTool struct {
	ran bool
}

func (t *testContextIsolationTool) Name() string {
	return "context_isolation_test"
}

func (t *testContextIsolationTool) Description() string {
	return "context isolation test tool"
}

func (t *testContextIsolationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testContextIsolationTool) Run(execCtx ExecutionContext, args map[string]string) (string, *FileChange, error) {
	t.started <- execCtx.ProviderName
	<-t.release
	value := execCtx.ProviderName + "/" + execCtx.Model
	execCtx.Output().Printf("CTX %s\n", value)
	return value, nil, nil
}

func (t *testCancelledTool) Name() string {
	return "cancelled_test"
}

func (t *testCancelledTool) Description() string {
	return "cancelled test tool"
}

func (t *testCancelledTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *testCancelledTool) Run(_ ExecutionContext, _ map[string]string) (string, *FileChange, error) {
	t.ran = true
	return "should not run", nil, nil
}
