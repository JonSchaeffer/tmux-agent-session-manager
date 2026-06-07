# tmux-ai-session-manager

A tmux plugin and CLI tool for managing AI coding agent sessions. It handles the full lifecycle — creating a git worktree, spawning a tmux session, launching your agent, and cleaning everything up when you're done — all from a single fuzzy-finder popup inside tmux.

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
set -g @plugin 'jonschaeffer/tmux-ai-session-manager'
```

Then press `prefix + I` to install.

### Manual

```bash
git clone https://github.com/jonschaeffer/tmux-ai-session-manager ~/.tmux/plugins/tmux-ai-session-manager
cd ~/.tmux/plugins/tmux-ai-session-manager
make install   # installs taism to /usr/local/bin
```

Add to `~/.tmux.conf`:

```tmux
run '~/.tmux/plugins/tmux-ai-session-manager/tmux-ai-session-manager.tmux'
```

Then reload tmux: `tmux source ~/.tmux.conf`.

## Configuration

Config file: `~/.config/taism/config.yaml` (or `$XDG_CONFIG_HOME/taism/config.yaml`)

```yaml
# Root directory scanned for git repositories (default: $HOME)
repo_root: ~/code

# Agent commands available in the picker
agents:
  claude: claude
  aider: aider

# Agent used when none is specified or only one is configured
default_agent: claude

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
set -g @ai-session-bind "A"
set -g @ai-session-popup-width "80%"
set -g @ai-session-popup-height "70%"
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
taism list
taism list --json

# Create a session (non-interactive)
taism create --repo ~/code/myapp --branch feature-auth
taism create --repo ~/code/myapp --branch feature-auth --agent aider

# Attach to a session
taism attach ai/myapp/feature-auth

# Delete a session and its worktree
taism delete ai/myapp/feature-auth
taism delete --force ai/myapp/feature-auth   # skip confirmation

# Discovery and config
taism repos        # list all git repos under repo_root
taism agents       # list configured agent names
taism config       # show current resolved config
```

### Session naming

Sessions are named `ai/<repo>/<branch>`, e.g. `ai/myapp/feature-auth`. This namespace keeps AI sessions grouped and separate from your regular tmux sessions.

## How it works

1. **Popup** — `prefix + A` opens an `fzf` popup listing active `ai/*` sessions via `taism list`.
2. **Create flow** — selecting "Create new session" runs `scripts/create.sh`, which chains three fzf prompts: pick a repo (`taism repos`), type a branch name, pick an agent (`taism agents`). It then calls `taism create`.
3. **Session creation** — `taism create` runs `git fetch`, creates a git worktree via `wt switch --create <branch>`, spawns a detached tmux session pointed at the worktree directory, and sends the agent command as keystrokes into the session.
4. **Deletion** — `taism delete` kills the tmux session first, then removes the worktree via `wt remove`. If worktree removal fails, the session is still considered deleted and a warning is printed.
5. **Agent detection** — `taism list` inspects the active pane command in each session to identify which agent (claude, aider, codex, pi) is running.
