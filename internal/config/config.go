package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Backend defines an OpenCode model backend.
type Backend struct {
	Model   string `yaml:"model"`
	Timeout int    `yaml:"timeout"`
}

// Config is the quorum-cc configuration.
type Config struct {
	Version        string             `yaml:"version"`
	Backends       map[string]Backend `yaml:"backends"`
	Defaults       Defaults           `yaml:"defaults"`
	PromptTemplate string             `yaml:"prompt_template,omitempty"`
}

// Defaults holds default settings.
type Defaults struct {
	Backend string `yaml:"backend"`
}

// DefaultConfigDir returns ~/.config/quorum-cc.
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "quorum-cc"), nil
}

// DefaultConfigPath returns ~/.config/quorum-cc/config.yaml.
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads config from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// LoadDefault reads config from the default path.
func LoadDefault() (*Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

// Save writes config to the given path, creating directories as needed.
func Save(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// SaveDefault writes config to the default path.
func SaveDefault(cfg *Config) error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return Save(cfg, path)
}

// BackendNames returns the sorted list of configured backend names.
func (c *Config) BackendNames() []string {
	names := make([]string, 0, len(c.Backends))
	for name := range c.Backends {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetBackend returns a backend by name, or error if not found.
func (c *Config) GetBackend(name string) (Backend, error) {
	b, ok := c.Backends[name]
	if !ok {
		return Backend{}, fmt.Errorf("backend %q not found in config", name)
	}
	return b, nil
}

// Validate checks the config for common errors.
func (c *Config) Validate() error {
	for name, b := range c.Backends {
		if b.Timeout <= 0 {
			return fmt.Errorf("backend %q: timeout must be positive, got %d", name, b.Timeout)
		}
		if b.Model == "" {
			return fmt.Errorf("backend %q: model must not be empty", name)
		}
	}
	return nil
}
