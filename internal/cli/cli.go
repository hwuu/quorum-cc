package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hwuu/quorum-cc/internal/config"
)

// Init detects environment, generates config, and registers MCP server.
func Init() error {
	fmt.Println("quorum-cc — Quorum for Claude Code")
	fmt.Println("===================================")
	fmt.Println()

	// [1/4] Check environment
	fmt.Println("[1/4] Check environment...")
	opencodePath, err := checkOpenCode()
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ opencode found at %s\n", opencodePath)

	ccPath, err := checkClaudeCode()
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Claude Code found at %s\n", ccPath)
	fmt.Println()

	// [2/4] Detect available models
	fmt.Println("[2/4] Detect available models...")
	available, err := detectModels()
	if err != nil {
		fmt.Printf("  ⚠ Could not auto-detect models: %v\n", err)
		fmt.Println("  Using default backends (edit config to customize)")
	}
	selected := selectDefaultModels(available)
	for _, m := range selected {
		fmt.Printf("  ✓ %s\n", m)
	}
	fmt.Println()

	// [3/4] Generate config
	fmt.Println("[3/4] Generate config...")
	cfg := buildConfig(selected)
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return err
	}
	if err := config.Save(cfg, configPath); err != nil {
		return err
	}
	fmt.Printf("  ✓ Created %s\n", configPath)
	fmt.Println()

	// [4/4] Register MCP server
	fmt.Println("[4/4] Register MCP server...")
	if err := registerMCP(); err != nil {
		return fmt.Errorf("register MCP: %w", err)
	}
	fmt.Println()

	fmt.Println("Done! Restart Claude Code, then try:")
	fmt.Println("  \"review this file with quorum\"")
	return nil
}

// Status checks opencode availability, configured backends, and MCP registration.
func Status() error {
	fmt.Println("quorum-cc status")
	fmt.Println("================")
	fmt.Println()

	// OpenCode
	opencodePath, err := checkOpenCode()
	if err != nil {
		fmt.Println("OpenCode:  ✗ not found")
	} else {
		fmt.Printf("OpenCode:  ✓ %s\n", opencodePath)
	}

	// Config
	configPath, _ := config.DefaultConfigPath()
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Printf("Config:    ✗ %s not found\n", configPath)
	} else {
		fmt.Printf("Config:    ✓ %s\n", configPath)
		fmt.Printf("Backends:  %s\n", strings.Join(cfg.BackendNames(), ", "))
		fmt.Printf("Default:   %s\n", cfg.Defaults.Backend)
	}

	// MCP registration
	registered, err := isMCPRegistered()
	if err != nil || !registered {
		fmt.Println("MCP:       ✗ not registered in Claude Code")
	} else {
		fmt.Println("MCP:       ✓ registered in Claude Code")
	}

	return nil
}

// Test sends a test review request to verify connectivity.
func Test() error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w (run 'quorum-cc init' first)", err)
	}

	fmt.Println("Testing backends...")
	for name, b := range cfg.Backends {
		fmt.Printf("  %s (%s)... ", name, b.Model)
		cmd := exec.Command("opencode", "run", "-m", b.Model, "Say 'hello' in one word.")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("✗ %v\n", err)
			continue
		}
		firstLine := strings.SplitN(strings.TrimSpace(string(output)), "\n", 2)[0]
		if len(firstLine) > 80 {
			firstLine = firstLine[:80] + "..."
		}
		fmt.Printf("✓ %s\n", firstLine)
	}
	return nil
}

func checkOpenCode() (string, error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return "", fmt.Errorf("opencode not found in PATH. Install it first: https://opencode.ai")
	}
	return path, nil
}

func checkClaudeCode() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH. Install Claude Code first")
	}
	return path, nil
}

func detectModels() ([]string, error) {
	cmd := exec.Command("opencode", "models")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var models []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		models = append(models, line)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no models detected")
	}
	return models, nil
}

func defaultModels() []string {
	return []string{
		"siliconflow-cn/Pro/zai-org/GLM-5",
		"siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5",
	}
}

// selectDefaultModels picks preferred models from the available list.
// Falls back to hardcoded defaults if detection failed.
func selectDefaultModels(available []string) []string {
	preferred := defaultModels()
	if len(available) == 0 {
		return preferred
	}

	avSet := make(map[string]bool, len(available))
	for _, m := range available {
		avSet[m] = true
	}

	var selected []string
	for _, p := range preferred {
		if avSet[p] {
			selected = append(selected, p)
		}
	}
	if len(selected) == 0 {
		return preferred
	}
	return selected
}

func buildConfig(models []string) *config.Config {
	backends := make(map[string]config.Backend)
	for _, model := range models {
		parts := strings.Split(model, "/")
		name := parts[len(parts)-1]
		backends[name] = config.Backend{
			Model:   model,
			Timeout: 600,
		}
	}
	return &config.Config{
		Version:  "1",
		Backends: backends,
		Defaults: config.Defaults{Backend: "all"},
	}
}

func claudeConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

func registerMCP() error {
	path, err := claudeConfigPath()
	if err != nil {
		return err
	}

	qccPath, err := exec.LookPath("quorum-cc")
	if err != nil {
		// Fallback: use the binary in current directory or go bin
		qccPath = "quorum-cc"
	}

	// Read existing config or create new
	var claudeCfg map[string]any
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &claudeCfg); err != nil {
			claudeCfg = make(map[string]any)
		}
	} else {
		claudeCfg = make(map[string]any)
	}

	// Ensure mcpServers key exists
	mcpServers, ok := claudeCfg["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	// Add quorum-cc entry
	mcpServers["quorum-cc"] = map[string]any{
		"command": qccPath,
		"args":    []string{"serve"},
		"description": "Multi-model code review via OpenCode backends",
	}
	claudeCfg["mcpServers"] = mcpServers

	// Write back
	out, err := json.MarshalIndent(claudeCfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return err
	}
	fmt.Printf("  ✓ Registered in %s\n", path)
	return nil
}

func isMCPRegistered() (bool, error) {
	path, err := claudeConfigPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var claudeCfg map[string]any
	if err := json.Unmarshal(data, &claudeCfg); err != nil {
		return false, err
	}
	mcpServers, ok := claudeCfg["mcpServers"].(map[string]any)
	if !ok {
		return false, nil
	}
	_, ok = mcpServers["quorum-cc"]
	return ok, nil
}
