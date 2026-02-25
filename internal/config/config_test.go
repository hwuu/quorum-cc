package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Version: "1",
		Backends: map[string]Backend{
			"glm-5": {
				Model:   "siliconflow-cn/Pro/zai-org/GLM-5",
				Timeout: 300,
			},
			"minimax": {
				Model:   "siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5",
				Timeout: 300,
			},
		},
		Defaults: Defaults{Backend: "all"},
	}

	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version != "1" {
		t.Errorf("Version = %q, want %q", loaded.Version, "1")
	}
	if len(loaded.Backends) != 2 {
		t.Errorf("Backends count = %d, want 2", len(loaded.Backends))
	}
	if loaded.Backends["glm-5"].Model != "siliconflow-cn/Pro/zai-org/GLM-5" {
		t.Errorf("glm-5 model = %q", loaded.Backends["glm-5"].Model)
	}
	if loaded.Defaults.Backend != "all" {
		t.Errorf("Defaults.Backend = %q, want %q", loaded.Defaults.Backend, "all")
	}
}

func TestGetBackend(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"glm-5": {Model: "siliconflow-cn/Pro/zai-org/GLM-5", Timeout: 300},
		},
	}

	b, err := cfg.GetBackend("glm-5")
	if err != nil {
		t.Fatalf("GetBackend: %v", err)
	}
	if b.Timeout != 300 {
		t.Errorf("Timeout = %d, want 300", b.Timeout)
	}

	_, err = cfg.GetBackend("nonexistent")
	if err == nil {
		t.Error("GetBackend(nonexistent) should return error")
	}
}

func TestBackendNames(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"glm-5":   {Model: "a"},
			"minimax": {Model: "b"},
		},
	}

	names := cfg.BackendNames()
	if len(names) != 2 {
		t.Errorf("BackendNames count = %d, want 2", len(names))
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load nonexistent should return error")
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	cfg := &Config{Version: "1", Backends: map[string]Backend{}}
	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestPromptTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Version:        "1",
		Backends:       map[string]Backend{},
		PromptTemplate: "custom template: {{.Content}}",
	}

	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.PromptTemplate != "custom template: {{.Content}}" {
		t.Errorf("PromptTemplate = %q", loaded.PromptTemplate)
	}
}

func TestValidateOK(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"glm-5": {Model: "siliconflow-cn/Pro/zai-org/GLM-5", Timeout: 300},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate should pass: %v", err)
	}
}

func TestValidateZeroTimeout(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"glm-5": {Model: "some-model", Timeout: 0},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate should fail for zero timeout")
	}
}

func TestValidateNegativeTimeout(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"glm-5": {Model: "some-model", Timeout: -1},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate should fail for negative timeout")
	}
}

func TestValidateEmptyModel(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"glm-5": {Model: "", Timeout: 300},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate should fail for empty model")
	}
}

func TestBackendNamesSorted(t *testing.T) {
	cfg := &Config{
		Backends: map[string]Backend{
			"z-backend": {Model: "z"},
			"a-backend": {Model: "a"},
			"m-backend": {Model: "m"},
		},
	}
	names := cfg.BackendNames()
	if names[0] != "a-backend" || names[1] != "m-backend" || names[2] != "z-backend" {
		t.Errorf("BackendNames not sorted: %v", names)
	}
}
