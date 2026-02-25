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

// Dispatch sends a review request to the specified backend(s).
// If backend is "all", it dispatches to all configured backends in parallel.
func Dispatch(ctx context.Context, cfg *config.Config, content, ctxStr, filePath, backend string) (string, error) {
	promptTmpl := cfg.PromptTemplate
	reviewPrompt, err := prompt.Build(promptTmpl, content, ctxStr, filePath)
	if err != nil {
		return "", fmt.Errorf("build prompt: %w", err)
	}

	if backend == "all" {
		return dispatchAll(ctx, cfg, reviewPrompt)
	}
	return dispatchOne(ctx, cfg, reviewPrompt, backend)
}

func dispatchOne(ctx context.Context, cfg *config.Config, reviewPrompt, backend string) (string, error) {
	b, err := cfg.GetBackend(backend)
	if err != nil {
		return "", err
	}
	timeout := time.Duration(b.Timeout) * time.Second
	return adapter.OpenCode(ctx, reviewPrompt, b.Model, timeout)
}

func dispatchAll(ctx context.Context, cfg *config.Config, reviewPrompt string) (string, error) {
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
			timeout := time.Duration(b.Timeout) * time.Second
			output, err := adapter.OpenCode(ctx, reviewPrompt, b.Model, timeout)
			results[idx] = Result{Backend: backendName, Output: output, Err: err}
		}(i, name)
	}
	wg.Wait()

	return formatResults(results), nil
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
