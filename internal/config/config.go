package config

import (
	"os"
	"path/filepath"
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
	return filepath.Join(base, "taism", "config.yaml")
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

	return cfg, nil
}

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, _ := os.UserHomeDir()
	return home + path[1:]
}
