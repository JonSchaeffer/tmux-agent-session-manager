package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // point at empty dir — no config file
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	if cfg.RepoRoot != home {
		t.Errorf("RepoRoot = %q, want %q", cfg.RepoRoot, home)
	}
	if cfg.DefaultAgent != "claude" {
		t.Errorf("DefaultAgent = %q, want %q", cfg.DefaultAgent, "claude")
	}
	if cfg.Keybinding != "A" {
		t.Errorf("Keybinding = %q, want %q", cfg.Keybinding, "A")
	}
}

func TestLoadMergesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tasm"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "repo_root: /tmp/repos\nkeybinding: B\n"
	if err := os.WriteFile(filepath.Join(dir, "tasm", "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.RepoRoot != "/tmp/repos" {
		t.Errorf("RepoRoot = %q, want /tmp/repos", cfg.RepoRoot)
	}
	if cfg.Keybinding != "B" {
		t.Errorf("Keybinding = %q, want B", cfg.Keybinding)
	}
	// Unset fields keep defaults.
	if cfg.DefaultAgent != "claude" {
		t.Errorf("DefaultAgent = %q, want claude", cfg.DefaultAgent)
	}
}

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := []struct{ in, want string }{
		{"~/code", home + "/code"},
		{"/abs/path", "/abs/path"},
		{"relative", "relative"},
	}
	for _, c := range cases {
		got := expandTilde(c.in)
		if got != c.want {
			t.Errorf("expandTilde(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidate(t *testing.T) {
	good := &Config{
		RepoRoot:     "/tmp",
		Agents:       map[string]string{"claude": "claude"},
		DefaultAgent: "claude",
		Keybinding:   "A",
		PopupWidth:   "80%",
		PopupHeight:  "70%",
	}
	cases := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{"valid", good, false},
		{"default_agent not in agents", &Config{
			RepoRoot: "/tmp", Agents: map[string]string{"claude": "claude"},
			DefaultAgent: "aider", Keybinding: "A", PopupWidth: "80%", PopupHeight: "70%",
		}, true},
		{"invalid agent name", &Config{
			RepoRoot: "/tmp", Agents: map[string]string{"bad name": "x"},
			DefaultAgent: "bad name", Keybinding: "A", PopupWidth: "80%", PopupHeight: "70%",
		}, true},
		{"empty agent command", &Config{
			RepoRoot: "/tmp", Agents: map[string]string{"claude": ""},
			DefaultAgent: "claude", Keybinding: "A", PopupWidth: "80%", PopupHeight: "70%",
		}, true},
		{"multi-char keybinding", &Config{
			RepoRoot: "/tmp", Agents: map[string]string{"claude": "claude"},
			DefaultAgent: "claude", Keybinding: "AB", PopupWidth: "80%", PopupHeight: "70%",
		}, true},
		{"bad popup width", &Config{
			RepoRoot: "/tmp", Agents: map[string]string{"claude": "claude"},
			DefaultAgent: "claude", Keybinding: "A", PopupWidth: "80px", PopupHeight: "70%",
		}, true},
		{"bad popup height", &Config{
			RepoRoot: "/tmp", Agents: map[string]string{"claude": "claude"},
			DefaultAgent: "claude", Keybinding: "A", PopupWidth: "80%", PopupHeight: "big",
		}, true},
		{"empty repo root", &Config{
			RepoRoot: "", Agents: map[string]string{"claude": "claude"},
			DefaultAgent: "claude", Keybinding: "A", PopupWidth: "80%", PopupHeight: "70%",
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
