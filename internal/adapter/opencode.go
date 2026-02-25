package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// execCommand is the function used to create exec.Cmd, replaceable in tests.
var execCommand = exec.CommandContext

// OpenCode calls `opencode run` with the given model and prompt.
func OpenCode(ctx context.Context, prompt, model string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := execCommand(ctx, "opencode", "run", "-m", model, prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("opencode run timed out after %s", timeout)
		}
		return "", fmt.Errorf("opencode run failed: %s", stderr.String())
	}
	return stdout.String(), nil
}
