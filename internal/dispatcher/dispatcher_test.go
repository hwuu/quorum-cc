package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hwuu/quorum-cc/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Version: "1",
		Backends: map[string]config.Backend{
			"glm-5": {
				Model:   "siliconflow-cn/Pro/zai-org/GLM-5",
				Timeout: 300,
			},
			"minimax": {
				Model:   "siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5",
				Timeout: 300,
			},
		},
		Defaults: config.Defaults{Backend: "all"},
	}
}

func TestDispatchOneUnknownBackend(t *testing.T) {
	cfg := testConfig()
	_, err := Dispatch(context.Background(), cfg, "code", "", "", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestDispatchNoBackends(t *testing.T) {
	cfg := &config.Config{
		Version:  "1",
		Backends: map[string]config.Backend{},
	}
	_, err := Dispatch(context.Background(), cfg, "code", "", "", "all")
	if err == nil {
		t.Fatal("expected error for no backends")
	}
	if !strings.Contains(err.Error(), "no backends") {
		t.Errorf("error should mention 'no backends': %v", err)
	}
}

func TestDispatchInvalidTemplate(t *testing.T) {
	cfg := testConfig()
	cfg.PromptTemplate = "{{.Invalid"
	_, err := Dispatch(context.Background(), cfg, "code", "", "", "glm-5")
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestFormatResults(t *testing.T) {
	results := []Result{
		{Backend: "glm-5", Output: "looks good"},
		{Backend: "minimax", Err: fmt.Errorf("timed out")},
	}
	output := formatResults(results)
	if !strings.Contains(output, "## glm-5 Review") {
		t.Error("should contain glm-5 header")
	}
	if !strings.Contains(output, "looks good") {
		t.Error("should contain glm-5 output")
	}
	if !strings.Contains(output, "## minimax Review") {
		t.Error("should contain minimax header")
	}
	if !strings.Contains(output, "[ERROR] timed out") {
		t.Error("should contain minimax error")
	}
	if !strings.Contains(output, "---") {
		t.Error("should contain separator")
	}
}
