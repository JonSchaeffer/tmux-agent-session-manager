package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RepoRoot     string            `yaml:"repo_root"`
	Agents       map[string]string `yaml:"agents"`
	DefaultAgent string            `yaml:"default_agent"`
	Keybinding   string            `yaml:"keybinding"`
	PopupWidth   string            `yaml:"popup_width"`
	PopupHeight  string            `yaml:"popup_height"`
}

func defaults() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		RepoRoot:     home,
		Agents:       map[string]string{"claude": "claude"},
		DefaultAgent: "claude",
		Keybinding:   "A",
		PopupWidth:   "80%",
		PopupHeight:  "70%",
	}
}

func ConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "tasm", "config.yaml")
}

func Load() (*Config, error) {
	cfg := defaults()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.RepoRoot = expandTilde(cfg.RepoRoot)

	// Re-apply defaults for zero-value fields that weren't in the file.
	d := defaults()
	if cfg.Agents == nil {
		cfg.Agents = d.Agents
	}
	if cfg.DefaultAgent == "" {
		cfg.DefaultAgent = d.DefaultAgent
	}
	if cfg.Keybinding == "" {
		cfg.Keybinding = d.Keybinding
	}
	if cfg.PopupWidth == "" {
		cfg.PopupWidth = d.PopupWidth
	}
	if cfg.PopupHeight == "" {
		cfg.PopupHeight = d.PopupHeight
	}
	if cfg.RepoRoot == "" {
		cfg.RepoRoot = d.RepoRoot
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return cfg, nil
}

var validAgentName = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
var validDimension = regexp.MustCompile(`^\d+%?$`)

func (c *Config) Validate() error {
	if c.RepoRoot == "" {
		return fmt.Errorf("repo_root must not be empty")
	}
	for key, val := range c.Agents {
		if !validAgentName.MatchString(key) {
			return fmt.Errorf("invalid agent name %q: agent names must be alphanumeric (hyphens allowed)", key)
		}
		if val == "" {
			return fmt.Errorf("agent %q has an empty command", key)
		}
	}
	if c.DefaultAgent == "" {
		return fmt.Errorf("default_agent must not be empty")
	}
	if _, ok := c.Agents[c.DefaultAgent]; !ok {
		return fmt.Errorf("default_agent %q is not defined in agents", c.DefaultAgent)
	}
	if len([]rune(c.Keybinding)) != 1 {
		return fmt.Errorf("keybinding must be a single character, got %q", c.Keybinding)
	}
	if !validDimension.MatchString(c.PopupWidth) {
		return fmt.Errorf("popup_width %q must be a number or percentage (e.g. 80%%)", c.PopupWidth)
	}
	if !validDimension.MatchString(c.PopupHeight) {
		return fmt.Errorf("popup_height %q must be a number or percentage (e.g. 70%%)", c.PopupHeight)
	}
	return nil
}

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, _ := os.UserHomeDir()
	return home + path[1:]
}
