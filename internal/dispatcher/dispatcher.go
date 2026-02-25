package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hwuu/quorum-cc/internal/adapter"
	"github.com/hwuu/quorum-cc/internal/config"
	"github.com/hwuu/quorum-cc/internal/prompt"
)

// Result holds the review result from a single backend.
type Result struct {
	Backend string
	Output  string
	Err     error
}

// maxContentLen is the maximum content length passed to OpenCode (~100KB).
const maxContentLen = 100000

// Dispatch sends a review request to the specified backend(s).
// If backend is "all", it dispatches to all configured backends in parallel.
// timeoutOverride (seconds) overrides the per-backend config timeout if > 0.
func Dispatch(ctx context.Context, cfg *config.Config, content, ctxStr, filePath, backend string, timeoutOverride int) (string, error) {
	if len(content) > maxContentLen {
		content = content[:maxContentLen] + "\n\n[内容已截断，原始长度: " + fmt.Sprintf("%d", len(content)) + " 字节]"
	}

	promptTmpl := cfg.PromptTemplate
	reviewPrompt, err := prompt.Build(promptTmpl, content, ctxStr, filePath)
	if err != nil {
		return "", fmt.Errorf("build prompt: %w", err)
	}

	if backend == "all" {
		return dispatchAll(ctx, cfg, reviewPrompt, timeoutOverride)
	}
	return dispatchOne(ctx, cfg, reviewPrompt, backend, timeoutOverride)
}

func dispatchOne(ctx context.Context, cfg *config.Config, reviewPrompt, backend string, timeoutOverride int) (string, error) {
	b, err := cfg.GetBackend(backend)
	if err != nil {
		return "", err
	}
	timeout := resolveTimeout(b.Timeout, timeoutOverride)
	return adapter.OpenCode(ctx, reviewPrompt, b.Model, timeout)
}

func dispatchAll(ctx context.Context, cfg *config.Config, reviewPrompt string, timeoutOverride int) (string, error) {
	names := cfg.BackendNames()
	if len(names) == 0 {
		return "", fmt.Errorf("no backends configured")
	}

	results := make([]Result, len(names))
	var wg sync.WaitGroup

	for i, name := range names {
		wg.Add(1)
		go func(idx int, backendName string) {
			defer wg.Done()
			b, err := cfg.GetBackend(backendName)
			if err != nil {
				results[idx] = Result{Backend: backendName, Err: err}
				return
			}
			timeout := resolveTimeout(b.Timeout, timeoutOverride)
			output, err := adapter.OpenCode(ctx, reviewPrompt, b.Model, timeout)
			results[idx] = Result{Backend: backendName, Output: output, Err: err}
		}(i, name)
	}
	wg.Wait()

	return formatResults(results), nil
}

func resolveTimeout(configTimeout, overrideSec int) time.Duration {
	if overrideSec > 0 {
		return time.Duration(overrideSec) * time.Second
	}
	return time.Duration(configTimeout) * time.Second
}

func formatResults(results []Result) string {
	var parts []string
	for _, r := range results {
		var section string
		if r.Err != nil {
			section = fmt.Sprintf("## %s Review\n\n[ERROR] %s", r.Backend, r.Err.Error())
		} else {
			section = fmt.Sprintf("## %s Review\n\n%s", r.Backend, r.Output)
		}
		parts = append(parts, section)
	}
	return strings.Join(parts, "\n\n---\n\n")
}
