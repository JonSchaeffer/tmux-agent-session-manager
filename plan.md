# tmux-agent-session-manager — Detailed Spec

## Overview

A tmux plugin + Go CLI that manages AI coding sessions as tmux sessions, each scoped to a git repo and isolated via `wt` (worktrunk) worktrees. Think tmux-sessionx but purpose-built for running parallel AI agents.

## Architecture

```
tmux keybinding → fzf-tmux popup → `tasm` Go CLI binary → tmux / wt / git / AI CLI
```

- **`tasm`** — single Go binary, the brain. Subcommands handle all logic.
- **Tmux plugin layer** — thin shell script (`tmux-agent-session-manager.tmux`) that registers the keybinding and launches the popup. Installed via TPM.
- **fzf** — drives the interactive UI inside the tmux popup.
- **wt** - needed for git worktree support, drives creation of worktrees

## Core Concepts

| Concept | Implementation |
|---|---|
| **AI Session** | A tmux session named `agent/<repo>/<branch>` running an AI agent (e.g. `claude`). The branch name doubles as the worktree name. |
| **Isolation** | Each session gets its own `wt` worktree so agents don't conflict |
| **Repo root** | User-configured directory (e.g. `~/code`) scanned for git repos. Defaults to `$HOME` if not configured. |
| **Temp sessions** | *(v2)* Sessions not tied to a repo, run in a configurable scratch dir. No `wt` involved. |

## Dependencies

- **tmux** — session management
- **tpm** — tmux plugin manager (for installation)
- **fzf** / **fzf-tmux** — interactive picker UI
- **wt** (worktrunk) — git worktree management (`wt switch --create`, `wt remove`, `wt list`)
- **git** — repo discovery, branch listing
- **Go** — compiled CLI binary

## Configuration

Config file: `~/.config/tasm/config.yaml` (XDG-compliant)

```yaml
# Directory containing git repos (defaults to $HOME if not set)
repo_root: ~/code

# (v2) Directory for temp (repo-less) sessions
# temp_dir: ~/tmp/ai-sessions

# Available AI agents and their launch commands
agents:
  claude: "claude"
  aider: "aider"
  codex: "codex"
  pi: "pi"

# Default agent when creating a session
default_agent: claude

# Tmux keybinding to open the popup (single key after prefix; multi-char like "AI" not supported by tmux)
keybinding: "A"  # prefix + A

# Popup dimensions
popup_width: "80%"
popup_height: "70%"
```

Tmux options (set in `.tmux.conf`):
```
set -g @agent-session-bind 'A'
set -g @agent-session-repo-root '~/code'
set -g @agent-session-default-agent 'claude'
```

## User Flows

### 1. Open the popup (`<prefix>+A`)

fzf-tmux floating window appears showing active AI sessions:

```
  agent/myapp/feature-auth     claude   running   3m ago
  agent/myapp/fix-logging      claude   running  12m ago
  agent/infra-tools/refactor   claude   idle      1h ago
> [Create new session]
```

Keybindings inside the popup:
- **Enter** — attach to selected session
- **Alt+Backspace** — delete selected session (with confirmation)
- **Ctrl+N** — create new session

### 2. Create a new session

Sequential fzf prompts:

1. **Pick a repo** — fzf-searchable list of git repos found under `repo_root`. If multiple repos share the same name, the user is prompted to disambiguate (showing parent path).
2. **Name the branch / worktree** — type a new branch name. This becomes the `wt` worktree identifier.
3. **Pick an AI agent** — fzf list from configured agents (skip if only one configured).

Behind the scenes, the CLI:
1. Runs `git fetch` in the selected repo to ensure it's up to date
2. Runs `wt switch --create <branch>` in the selected repo to create an isolated worktree (branching from `main`/`master`)
3. Creates a tmux session named `agent/<repo>/<branch>` with its working directory set to the new worktree path
4. Launches the selected AI agent command inside that session
5. Attaches the user to the new session

> Note: the worktree must be created before the tmux session since the session needs the worktree path as its working directory.

### 3. Attach to an existing session

Select a session from the list, press Enter. The CLI runs `tmux switch-client -t <session-name>` to attach.

### 4. Delete a session

Select a session, press `Alt+Backspace`. The CLI:
1. Prompts for confirmation
2. Kills the AI agent process running in the session
3. Kills the tmux session (`tmux kill-session -t <name>`)
4. Removes the worktree (`wt remove <branch>` in the parent repo)
5. Optionally deletes the branch (`git branch -D <branch>`)

## CLI Subcommands

```
tasm                     # No args: launch the popup (main entry point)
tasm list                # JSON output of all active AI sessions
tasm create              # Interactive session creation flow
tasm attach <session>    # Attach to a session by name
tasm delete <session>    # Delete a session and clean up worktree
tasm repos               # List discovered repos under repo_root
tasm config              # Print resolved config
```

The `list` subcommand powers the fzf display. It gathers data from:
- `tmux list-sessions` — active tmux sessions matching the `agent/` prefix
- `wt list` — worktree status per repo
- Process inspection — whether the AI agent is still running or idle

## Session Naming Convention

```
agent/<repo-name>/<branch>
```

Examples:
- `agent/myapp/feature-auth`
- `agent/infra-tools/refactor-ci`
- `agent/temp/quick-experiment`

The `<repo-name>` is the directory name of the repo (not the full path). Collisions (two repos named `myapp`) are handled by appending a short suffix.

## Tmux Plugin Structure

```
tmux-agent-session-manager/
├── tmux-agent-session-manager.tmux   # TPM entrypoint: registers keybinding
├── scripts/
│   ├── popup.sh                   # Launched by keybinding, runs fzf-tmux
│   ├── create.sh                  # Interactive create flow (fzf prompts)
│   └── delete.sh                  # Delete with confirmation
├── cmd/
│   └── tasm/
│       └── main.go                # CLI entrypoint
├── internal/
│   ├── config/                    # Config loading (YAML + tmux options)
│   ├── session/                   # Session CRUD (tmux + wt orchestration)
│   ├── repo/                      # Repo discovery under root dir
│   └── display/                   # Format session list for fzf
├── go.mod
├── go.sum
├── plan.md
└── idea.md
```

The shell scripts are thin wrappers — they invoke `tasm` subcommands and pipe output to fzf. All real logic lives in Go.

## Implementation Phases

### Phase 1: Foundation
- [ ] Go project scaffold (`go mod init`, CLI framework with cobra or similar)
- [ ] Config loading (YAML file + defaults)
- [ ] Repo discovery (walk `repo_root`, find directories with `.git`)
- [ ] Session listing (query tmux for `agent/*` sessions, enrich with metadata)

### Phase 2: Core CRUD
- [ ] Session creation (repo picker → branch name → `wt switch --create` → `tmux new-session` → launch agent)
- [ ] Session attach (`tmux switch-client`)
- [ ] Session deletion (kill session → `wt remove` → optional branch cleanup)

### Phase 3: Tmux Plugin + UI
- [ ] TPM plugin entrypoint script
- [ ] fzf-tmux popup integration (floating window, keybindings)
- [ ] Interactive create flow via chained fzf prompts
- [ ] Delete confirmation prompt

### Phase 4: Polish
- [ ] Session status display (running/idle detection, age, agent type)
- [ ] Error handling (repo not found, wt failures, tmux not running)
- [ ] Install instructions (TPM, Homebrew formula for the Go binary)
- [ ] Config validation and helpful error messages

### Stretch Goals
- [ ] Multiple agent support per session (e.g. switch agent mid-session)
- [ ] Session logging / history
- [ ] Integration with `wt list --full` for CI status in the popup
- [ ] Preview pane in fzf showing recent session output
- [ ] Auto-cleanup of stale sessions (agent exited, worktree orphaned)
- [ ] **Real activity status (working/waiting/exited).** Today `status` only distinguishes agent-alive (`running`) from agent-exited (`idle`) via the pane's foreground process — it can't tell an actively-processing agent from one waiting at the prompt. To show a true "working" state, inspect pane *content* for the agent's processing indicator (Claude Code prints an `… (esc to interrupt)` line + spinner while working, vs. just the `❯` prompt box when idle). Three states: `working` (indicator present), `waiting` (agent alive, no indicator), `exited` (bare shell). Caveat: the indicator is agent-specific and can change across Claude Code versions; would need per-agent patterns for aider/codex/etc.

## Resolved Decisions

1. **Agent lifecycle** — v1: leave the tmux session open when the agent exits. v2: monitor agent process and mark sessions as "done."
2. **Branch strategy** — always `git fetch` then branch from `main`/`master`. No base branch picker in v1.
3. **Temp sessions** — deferred to v2.
4. **Multi-repo collisions** — prompt the user to disambiguate (show parent directory path alongside repo name).

