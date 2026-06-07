# tmux-agent-session-manager

> [!NOTE]
> This project used AI to help create it. Opus 4.8 was used to help ideate and create work tasks. Sonnet 4.6 was used to help write code. All code was evaluated by a human.

[![CI](https://github.com/JonSchaeffer/tmux-agent-session-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/JonSchaeffer/tmux-agent-session-manager/actions/workflows/ci.yml)

A tmux plugin and CLI tool for managing AI coding agent sessions. It handles the full lifecycle for agent management. It creates a git worktree, spawns a tmux session, and launches your agent. When you are done, it cleans everything up. All from a single fuzzy-finder popup inside tmux.

## Demo

<!-- TODO: add GIF here -->

## Dependencies

| Dependency | Purpose | Install |
|------------|---------|---------|
| [tmux](https://github.com/tmux/tmux) ≥ 3.2 | Session management | `brew install tmux` |
| [fzf](https://github.com/junegunn/fzf) | Fuzzy-finder UI | `brew install fzf` |
| [wt (worktrunk)](https://github.com/max-sixty/worktrunk) | Git worktree management | `brew install worktrunk && wt config shell install` |
| [Go](https://go.dev) ≥ 1.21 | Build the CLI | `brew install go` |

## Installation

### Via TPM (recommended)

Add to `~/.tmux.conf`:

```tmux
set -g @plugin 'JonSchaeffer/tmux-agent-session-manager'
```

Press `prefix + I` to install. On first load, the plugin automatically fetches the `tasm` binary — either a prebuilt release binary for your platform (darwin/linux, arm64/amd64), or built from source if Go is installed. No manual build step needed.

### Manual

```bash
git clone https://github.com/jonschaeffer/tmux-agent-session-manager ~/.tmux/plugins/tmux-agent-session-manager
cd ~/.tmux/plugins/tmux-agent-session-manager
make install   # installs tasm to /usr/local/bin
```

Add to `~/.tmux.conf`:

```tmux
run '~/.tmux/plugins/tmux-agent-session-manager/tmux-agent-session-manager.tmux'
```

Then reload tmux: `tmux source ~/.tmux.conf`.

## Configuration

Config file: `~/.config/tasm/config.yaml` (or `$XDG_CONFIG_HOME/tasm/config.yaml`)

```yaml
# Root directory scanned for git repositories (default: $HOME)
repo_root: ~/code

# Agent commands available in the picker
agents:
  pi: pi
  claude: claude
  aider: aider

# Agent used when none is specified or only one is configured
default_agent: pi

# Tmux keybinding for the popup (prefix + <key>)
keybinding: A

# Popup dimensions
popup_width: 80%
popup_height: 70%
```

All fields are optional — the tool works with no config file present.

### Tmux option overrides

You can also configure the keybinding and popup size directly in `~/.tmux.conf`, which takes effect without editing the config file:

```tmux
set -g @agent-session-bind "A"
set -g @agent-session-popup-width "80%"
set -g @agent-session-popup-height "70%"
```

## Usage

### Popup (primary interface)

Press `prefix + A` inside any tmux session to open the session manager popup.

| Key | Action |
|-----|--------|
| `Enter` | Attach to the selected session |
| `Alt + Backspace` | Delete the selected session and its worktree |
| `Ctrl + N` | Start the interactive create flow |
| `Esc` / `Ctrl + C` | Close the popup |

### CLI subcommands

```bash
# Show all AI sessions
tasm list
tasm list --json

# Create a session (non-interactive)
tasm create --repo ~/code/myapp --branch feature-auth
tasm create --repo ~/code/myapp --branch feature-auth --agent aider

# Attach to a session
tasm attach agent/myapp/feature-auth

# Delete a session and its worktree
tasm delete agent/myapp/feature-auth
tasm delete --force agent/myapp/feature-auth   # skip confirmation

# Discovery and config
tasm repos        # list all git repos under repo_root
tasm agents       # list configured agent names
tasm config       # show current resolved config
```

### Session naming

Sessions are named `agent/<repo>/<branch>`, e.g. `agent/myapp/feature-auth`. This namespace keeps AI sessions grouped and separate from your regular tmux sessions.

## How it works

1. **Popup** — `prefix + A` opens an `fzf` popup listing active `agent/*` sessions via `tasm list`.
2. **Create flow** — selecting "Create new session" runs `scripts/create.sh`, which chains three fzf prompts: pick a repo (`tasm repos`), type a branch name, pick an agent (`tasm agents`). It then calls `tasm create`.
3. **Session creation** — `tasm create` runs `git fetch`, creates a git worktree via `wt switch --create <branch>`, spawns a detached tmux session pointed at the worktree directory, and sends the agent command as keystrokes into the session.
4. **Deletion** — `tasm delete` kills the tmux session first, then removes the worktree via `wt remove`. If worktree removal fails, the session is still considered deleted and a warning is printed.
5. **Agent detection** — `tasm list` inspects the active pane command in each session to identify which agent (claude, aider, codex, pi) is running.
