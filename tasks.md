# Implementation Tasks

> These tasks are **linear** — each depends on the ones before it. Complete them in order.
> Each task should be implemented in its own commit.
> Refer to `plan.md` for the full spec when context is needed.

---

## Task 1: Go Project Scaffold + CLI Skeleton

### What to do

Initialize the Go project and set up the CLI framework with empty subcommands. This is pure scaffolding — no real logic yet.

### Steps

1. Run `go mod init github.com/jonschaeffer/tmux-agent-session-manager` in the project root.
2. Install cobra as the CLI framework: `go get github.com/spf13/cobra@latest`.
3. Create `cmd/tasm/main.go` — the binary entrypoint. It should call the root command's `Execute()`.
4. Create `internal/cli/root.go` — define the root cobra command. When invoked with no args, it should print `"tasm - tmux agent session manager"` and exit (placeholder — will launch the popup later in Task 8).
5. Register the following subcommands as empty stubs (each prints `"not implemented yet"` and exits with code 1):
   - `list` in `internal/cli/list.go`
   - `create` in `internal/cli/create.go`
   - `attach` in `internal/cli/attach.go`
   - `delete` in `internal/cli/delete.go`
   - `repos` in `internal/cli/repos.go`
   - `config` in `internal/cli/config.go`
6. Create a `Makefile` with:
   - `build`: compiles the binary to `bin/tasm`
   - `install`: copies `bin/tasm` to `$(GOPATH)/bin/` (or `/usr/local/bin/`)
   - `clean`: removes the `bin/` directory
7. Add `bin/` to `.gitignore`.
8. Verify: `make build && ./bin/tasm` prints the placeholder message. `./bin/tasm list` prints "not implemented yet" and exits 1.

### Acceptance Criteria

- [ ] `go build ./cmd/tasm` succeeds with zero errors
- [ ] Running `./bin/tasm` prints the tool name and exits 0
- [ ] Running `./bin/tasm list`, `create`, `attach`, `delete`, `repos`, `config` each print "not implemented yet" and exit 1
- [ ] Running `./bin/tasm --help` shows all six subcommands
- [ ] `Makefile` has `build`, `install`, and `clean` targets that work
- [ ] `.gitignore` excludes `bin/`

### Files to create

```
cmd/tasm/main.go
internal/cli/root.go
internal/cli/list.go
internal/cli/create.go
internal/cli/attach.go
internal/cli/delete.go
internal/cli/repos.go
internal/cli/config.go
Makefile
.gitignore
```

---

## Task 2: Configuration Loading

### What to do

Implement config loading from a YAML file at `~/.config/tasm/config.yaml`. The config struct should have defaults for every field so the tool works without a config file present.

### Steps

1. Install the YAML library: `go get gopkg.in/yaml.v3`.
2. Create `internal/config/config.go` with:
   - A `Config` struct with these fields and defaults:
     ```
     RepoRoot      string            — default: $HOME
     Agents        map[string]string — default: {"claude": "claude"}
     DefaultAgent  string            — default: "claude"
     Keybinding    string            — default: "A"
     PopupWidth    string            — default: "80%"
     PopupHeight   string            — default: "70%"
     ```
   - A `Load() (*Config, error)` function that:
     1. Constructs the config path using `$XDG_CONFIG_HOME/tasm/config.yaml` (falling back to `~/.config/tasm/config.yaml` if `$XDG_CONFIG_HOME` is unset).
     2. If the file exists, reads and unmarshals it into the struct, merging with defaults (i.e., any field not specified in the file keeps its default value).
     3. If the file doesn't exist, returns the defaults (no error).
     4. Expands `~` in `RepoRoot` to the actual home directory.
   - A `ConfigPath() string` function that returns the resolved config file path.
3. Update the `config` subcommand in `internal/cli/config.go` to:
   - Call `config.Load()`
   - Print the resolved config as YAML to stdout
   - Exit 0

### Acceptance Criteria

- [ ] `tasm config` with no config file present prints the default config as YAML and exits 0
- [ ] `tasm config` with a config file present prints the merged config (user values override defaults, unset fields keep defaults)
- [ ] `RepoRoot` defaults to `$HOME` when not specified
- [ ] `Agents` map defaults to `{"claude": "claude"}` when not specified
- [ ] `~` in `RepoRoot` is expanded to the actual home directory path in the output
- [ ] If the config file has invalid YAML, the command exits with a clear error message

### Files to create/modify

```
internal/config/config.go  (new)
internal/cli/config.go     (modify — replace stub)
```

---

## Task 3: Repo Discovery

### What to do

Implement scanning of `repo_root` to find all git repositories. This powers the repo picker when creating sessions.

### Steps

1. Create `internal/repo/discovery.go` with:
   - A `Repo` struct: `Name string`, `Path string` (absolute path to the repo root).
   - A `Discover(root string) ([]Repo, error)` function that:
     1. Walks `root` looking for directories that contain a `.git` subdirectory (or `.git` file for worktrees).
     2. Does NOT recurse into `.git` directories, `node_modules`, or directories that are themselves git repos (i.e., once you find a `.git`, don't look for nested repos inside it).
     3. Limits scan depth to 3 levels below `root` to avoid scanning the entire filesystem when `root` is `$HOME`.
     4. Returns the list sorted alphabetically by `Name`.
   - A `Disambiguate(repos []Repo) []Repo` function that:
     1. Finds repos that share the same `Name` (directory name).
     2. For colliding repos, updates their `Name` to include the parent directory: `parentdir/reponame` (just enough to disambiguate).
2. Update the `repos` subcommand in `internal/cli/repos.go` to:
   - Load config
   - Call `Discover(config.RepoRoot)`
   - Call `Disambiguate()` on the results
   - Print each repo as `<name>\t<path>`, one per line
   - Exit 0

### Acceptance Criteria

- [ ] `tasm repos` lists git repos found under the configured `repo_root`
- [ ] Output format is `<name>\t<path>` (tab-separated), one repo per line
- [ ] Repos are sorted alphabetically by name
- [ ] `.git`, `node_modules` directories are not recursed into
- [ ] Nested repos inside a git repo are not discovered (e.g., git submodules are skipped)
- [ ] Scan depth is capped at 3 levels below root
- [ ] If two repos share the same directory name (e.g., `~/code/work/api` and `~/code/personal/api`), they are disambiguated as `work/api` and `personal/api`
- [ ] If `repo_root` doesn't exist, the command exits with a clear error message

### Files to create/modify

```
internal/repo/discovery.go  (new)
internal/cli/repos.go       (modify — replace stub)
```

---

## Task 4: Session Listing

### What to do

Implement the `list` subcommand that queries tmux for active AI sessions and outputs structured data. This is what powers the fzf popup display.

### Steps

1. Create `internal/session/session.go` with:
   - A `Session` struct:
     ```
     Name      string    // e.g. "agent/myapp/feature-auth"
     RepoName  string    // e.g. "myapp"
     Branch    string    // e.g. "feature-auth"
     Agent     string    // e.g. "claude"
     CreatedAt time.Time
     WorkDir   string    // working directory of the session
     ```
   - A `List() ([]Session, error)` function that:
     1. Runs `tmux list-sessions -F "#{session_name}\t#{session_created}\t#{session_path}"`.
     2. Filters to sessions whose name starts with `agent/`.
     3. Parses each matching session into a `Session` struct. The repo name and branch are extracted by splitting the session name on `/` (format: `agent/<repo>/<branch>`).
     4. Detects the agent type: check the session's active pane command via `tmux list-panes -t <session> -F "#{pane_current_command}"`. If the command matches a known agent name (claude, aider, codex, pi), set it. Otherwise set to `"unknown"`.
     5. Returns sessions sorted by creation time (newest first).
2. Update the `list` subcommand in `internal/cli/list.go` to:
   - Accept a `--json` flag. When set, output the sessions as a JSON array.
   - When `--json` is NOT set, output in a human-readable format suitable for fzf display:
     ```
     agent/myapp/feature-auth     claude   3m ago
     agent/myapp/fix-logging      claude   12m ago
     ```
     The format is: `<session-name>` (left-padded to 40 chars), then `<agent>` (left-padded to 10 chars), then relative time.
   - If no sessions exist, output nothing and exit 0 (not an error).

### Acceptance Criteria

- [ ] `tasm list` outputs only tmux sessions that start with `agent/`
- [ ] Each line contains session name, agent type, and relative age
- [ ] `tasm list --json` outputs a valid JSON array of session objects
- [ ] Agent detection correctly identifies the running process (claude, aider, codex, pi)
- [ ] Sessions are sorted newest-first
- [ ] If tmux is not running, the command exits with a clear error: "tmux is not running"
- [ ] If there are no AI sessions, the command outputs nothing and exits 0

### Files to create/modify

```
internal/session/session.go  (new)
internal/cli/list.go         (modify — replace stub)
```

---

## Task 5: Session Creation (non-interactive)

### What to do

Implement the core session creation logic as a programmatic function and wire it to the `create` subcommand with flags. The interactive fzf-driven flow comes later in Task 9 — this task builds the engine underneath.

### Steps

1. Create `internal/session/create.go` with:
   - A `CreateOpts` struct:
     ```
     RepoPath  string  // absolute path to the git repo
     RepoName  string  // display name of the repo
     Branch    string  // branch/worktree name to create
     Agent     string  // agent command to launch (e.g. "claude")
     ```
   - A `Create(opts CreateOpts) error` function that does the following **in this exact order**:
     1. **Validate**: check that `RepoPath` exists and is a git repo. Check that `Branch` is non-empty and is a valid git branch name (no spaces, no special chars except `-`, `_`, `/`).
     2. **Fetch**: run `git -C <RepoPath> fetch` to update remote refs.
     3. **Create worktree**: run `wt switch --create <Branch>` with the working directory set to `RepoPath`. Capture stdout/stderr. If `wt` fails, return the error with the stderr output.
     4. **Determine worktree path**: after `wt switch --create`, the worktree will be at a path determined by `wt`. Run `wt list` in the repo directory, parse the output to find the row matching `<Branch>`, and extract the worktree path. Alternatively, use `git worktree list --porcelain` and find the worktree for the branch.
     5. **Create tmux session**: run `tmux new-session -d -s "agent/<RepoName>/<Branch>" -c <worktree-path>`. The `-d` flag creates it detached. If a session with this name already exists, return an error: `"session agent/<RepoName>/<Branch> already exists"`.
     6. **Launch agent**: run `tmux send-keys -t "agent/<RepoName>/<Branch>" "<Agent>" Enter` to start the agent in the session.
     7. Return nil on success.
2. Update the `create` subcommand in `internal/cli/create.go` to:
   - Accept flags: `--repo` (path), `--branch` (name), `--agent` (command, defaults to config's `default_agent`).
   - All three of `--repo`, `--branch` are required. If missing, print usage and exit 1.
   - Call `session.Create()` with the provided options.
   - On success, print `"Created session agent/<repo>/<branch>"` and exit 0.
   - On failure, print the error and exit 1.

### Acceptance Criteria

- [ ] `tasm create --repo /path/to/repo --branch my-feature` creates a worktree via `wt`, a tmux session named `agent/<repo-dir-name>/my-feature`, and launches the default agent
- [ ] The tmux session's working directory is the worktree path (not the main repo)
- [ ] `git fetch` runs before worktree creation
- [ ] If the session name already exists, it exits with a clear error
- [ ] If `wt switch --create` fails (e.g., branch already exists), the error message from `wt` is shown to the user and no tmux session is created
- [ ] If `--repo` or `--branch` are missing, usage is printed and exit code is 1
- [ ] `--agent` defaults to the configured `default_agent`
- [ ] The agent command is launched inside the tmux session (visible if you attach to it)
- [ ] Branch name validation rejects names with spaces or invalid git characters

### Files to create/modify

```
internal/session/create.go  (new)
internal/cli/create.go      (modify — replace stub)
```

---

## Task 6: Session Attach

### What to do

Implement attaching to an existing AI session by name.

### Steps

1. Create `internal/session/attach.go` with:
   - An `Attach(sessionName string) error` function that:
     1. Verifies the session exists by running `tmux has-session -t <sessionName>`. If it doesn't exist, return error: `"session <sessionName> does not exist"`.
     2. Runs `tmux switch-client -t <sessionName>` to switch to it.
     3. If `switch-client` fails (e.g., not inside tmux), fall back to `tmux attach-session -t <sessionName>`.
2. Update the `attach` subcommand in `internal/cli/attach.go` to:
   - Require exactly one positional argument: the session name.
   - Call `session.Attach()`.
   - On failure, print the error and exit 1.

### Acceptance Criteria

- [ ] `tasm attach agent/myapp/feature-auth` switches to that tmux session
- [ ] If already inside tmux, uses `switch-client`. If not inside tmux, uses `attach-session`.
- [ ] If the session doesn't exist, exits with `"session <name> does not exist"` and exit code 1
- [ ] If no argument is provided, prints usage and exits 1

### Files to create/modify

```
internal/session/attach.go  (new)
internal/cli/attach.go      (modify — replace stub)
```

---

## Task 7: Session Deletion

### What to do

Implement deleting a session, which includes killing the tmux session and cleaning up the worktree.

### Steps

1. Create `internal/session/delete.go` with:
   - A `Delete(sessionName string) error` function that:
     1. **Parse** the session name to extract `repoName` and `branch` (split `agent/<repo>/<branch>` on `/`).
     2. **Find the repo path**: look up the session's working directory via `tmux display-message -t <sessionName> -p "#{session_path}"`. From this, derive the parent repo path (the worktree's parent repo).
     3. **Kill the tmux session**: run `tmux kill-session -t <sessionName>`.
     4. **Remove the worktree**: run `wt remove <branch>` with the working directory set to the parent repo. If `wt remove` fails, log a warning but don't fail the overall delete (the tmux session is already gone — warn the user to clean up manually).
   - A `DeleteWithConfirmation(sessionName string, confirm func(string) bool) error` function that:
     1. Calls `confirm("Delete session <sessionName>? This will remove the worktree. [y/N]")`.
     2. If confirmed, calls `Delete()`.
     3. If not confirmed, returns nil (no-op).
2. Update the `delete` subcommand in `internal/cli/delete.go` to:
   - Require exactly one positional argument: the session name.
   - Accept a `--force` flag that skips confirmation.
   - Without `--force`, prompt for confirmation on stdin.
   - Call `Delete()` (or `DeleteWithConfirmation()`).
   - On success, print `"Deleted session <name>"`.
   - On failure, print the error and exit 1.

### Acceptance Criteria

- [ ] `tasm delete agent/myapp/feature-auth` prompts for confirmation, then kills the tmux session and removes the worktree
- [ ] `tasm delete --force agent/myapp/feature-auth` skips confirmation
- [ ] The tmux session is killed before worktree removal
- [ ] If the worktree removal fails, the command still succeeds but prints a warning: `"warning: failed to remove worktree <branch>: <error>. Clean up manually with: wt remove <branch>"`
- [ ] If the session doesn't exist, exits with a clear error
- [ ] If no argument is provided, prints usage and exits 1

### Files to create/modify

```
internal/session/delete.go  (new)
internal/cli/delete.go      (modify — replace stub)
```

---

## Task 8: TPM Plugin Entrypoint + Popup Shell Script

### What to do

Create the tmux plugin entrypoint that registers the keybinding, and the popup shell script that launches fzf-tmux with the session list.

### Steps

1. Create `tmux-agent-session-manager.tmux` (the TPM entrypoint):
   - Must be executable (`chmod +x`).
   - Read the keybinding from tmux option: `tmux show-option -gqv @agent-session-bind`, default to `A`.
   - Resolve the plugin directory: `PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"`.
   - Bind the key: `tmux bind-key <keybinding> display-popup -E -w <width> -h <height> "$PLUGIN_DIR/scripts/popup.sh"`.
   - Read popup width/height from tmux options `@agent-session-popup-width` and `@agent-session-popup-height`, defaulting to `80%` and `70%`.
2. Create `scripts/popup.sh`:
   - Must be executable.
   - Locate the `tasm` binary: check `$PATH` first, then fall back to `$PLUGIN_DIR/bin/tasm`.
   - Check that `tasm` exists. If not, print `"tasm binary not found. Run 'make install' in the plugin directory."` and exit 1.
   - Run `tasm list` to get the session list.
   - Prepend `[Create new session]` as the first line.
   - Pipe the combined list into `fzf-tmux` (or `fzf` since we're already in a popup) with these settings:
     - `--reverse` (list from top)
     - `--no-sort` (preserve our sort order: newest first)
     - `--bind "alt-bspace:execute-silent(tasm delete --force {})+reload(tasm list)"` — delete and refresh
     - `--bind "ctrl-n:print([Create new session])+accept"` — jump to create flow
     - `--header "enter: attach | alt-⌫: delete | ctrl-n: new"` — help text
   - Handle the selected result:
     - If the user selected `[Create new session]` → exec `$PLUGIN_DIR/scripts/create.sh`
     - Otherwise → extract the session name (first whitespace-delimited field) and run `tasm attach <session-name>`

### Acceptance Criteria

- [ ] After installing via TPM (or sourcing `tmux-agent-session-manager.tmux`), pressing `prefix + A` opens a floating popup
- [ ] The popup shows existing AI sessions plus `[Create new session]` at the top
- [ ] Selecting a session and pressing Enter attaches to it
- [ ] Pressing `Alt+Backspace` on a session deletes it and refreshes the list
- [ ] Pressing `Ctrl+N` or selecting `[Create new session]` launches the create flow
- [ ] The header line shows available keybindings
- [ ] If `tasm` binary is not found, shows an error message
- [ ] Both `.tmux` and `.sh` files are executable

### Files to create

```
tmux-agent-session-manager.tmux  (new, executable)
scripts/popup.sh               (new, executable)
```

---

## Task 9: Interactive Create Flow (Shell Script)

### What to do

Create the interactive session creation script that chains fzf prompts for repo selection, branch naming, and agent selection, then calls `tasm create`.

### Steps

1. Create `scripts/create.sh`:
   - Must be executable.
   - Locate `tasm` binary (same logic as `popup.sh` — extract to a shared helper if cleaner, or just duplicate the 3 lines).
   - **Step 1 — Pick a repo**:
     - Run `tasm repos` to get the repo list.
     - Pipe into `fzf --prompt "repo> " --reverse`.
     - Extract the selected repo path (second tab-separated field).
     - If the user cancels (Ctrl+C / Esc), exit 0 (return to popup or close).
   - **Step 2 — Name the branch**:
     - Prompt the user with `fzf --prompt "branch name> " --print-query --no-info` with an empty input list. This lets the user type a branch name freehand.
     - The branch name is whatever the user typed (from `--print-query`).
     - If empty or cancelled, exit 0.
   - **Step 3 — Pick an agent**:
     - Load the agent list from `tasm config` output (parse the YAML `agents:` section — or add a `tasm agents` helper that just prints agent names, one per line). Simplest approach: `tasm config --json | jq -r '.agents | keys[]'` — but that adds a `jq` dependency. Better: add a `tasm agents` subcommand that prints agent names one per line.
     - If there's only one agent, skip this step and use it automatically.
     - Otherwise, pipe into `fzf --prompt "agent> " --reverse`.
     - If cancelled, exit 0.
   - **Step 4 — Create the session**:
     - Run `tasm create --repo <repo-path> --branch <branch-name> --agent <agent>`.
     - If it succeeds, run `tasm attach agent/<repo-name>/<branch>`.
     - If it fails, show the error and wait for a keypress before exiting (so the user can read it).

2. Add a `tasm agents` subcommand (in `internal/cli/agents.go`):
   - Loads config.
   - Prints each agent name (the map key), one per line, sorted alphabetically.
   - This avoids requiring `jq` as a dependency.

### Acceptance Criteria

- [ ] Running `scripts/create.sh` walks the user through three fzf steps: repo → branch → agent
- [ ] The repo picker shows all repos discovered by `tasm repos`
- [ ] The branch name step accepts freehand text input
- [ ] The agent picker is skipped if only one agent is configured
- [ ] Cancelling at any step (Ctrl+C) exits cleanly without creating anything
- [ ] After all steps, `tasm create` is called with the correct flags
- [ ] On success, the user is attached to the new session
- [ ] On failure, the error is displayed and the script waits before closing
- [ ] `tasm agents` prints agent names, one per line

### Files to create/modify

```
scripts/create.sh        (new, executable)
internal/cli/agents.go   (new)
```

---

## Task 10: Error Handling + Dependency Checks

### What to do

Add startup validation so the tool fails fast with helpful messages when dependencies are missing.

### Steps

1. Create `internal/deps/check.go` with:
   - A `Check() error` function that verifies these binaries are on `$PATH`:
     - `tmux`
     - `fzf`
     - `wt`
     - `git`
   - For each missing binary, collect the name. If any are missing, return an error listing all of them: `"missing required dependencies: tmux, wt"`.
   - A `CheckTmuxRunning() error` function that runs `tmux list-sessions` and checks the exit code. If tmux isn't running, return `"tmux server is not running. Start tmux first."`.
2. Wire the checks into the CLI:
   - In the root command's `PersistentPreRunE`, call `deps.Check()`. This runs before every subcommand.
   - In subcommands that need tmux (`list`, `create`, `attach`, `delete`), additionally call `deps.CheckTmuxRunning()` in their `PreRunE`.
   - The `config`, `repos`, and `agents` subcommands do NOT require tmux to be running (they're read-only).
3. Add error handling to `internal/session/create.go`:
   - If `git fetch` fails, print a warning but continue (the user may be offline).
   - If `wt` is not installed, print: `"wt (worktrunk) is required. Install: brew install max-sixty/tap/worktrunk"`.

### Acceptance Criteria

- [ ] Running any subcommand without `tmux` on PATH exits with: `"missing required dependencies: tmux"`
- [ ] Running any subcommand without `wt` on PATH exits with: `"missing required dependencies: wt"` (with install hint)
- [ ] Running `tasm list` when tmux isn't running exits with: `"tmux server is not running. Start tmux first."`
- [ ] Running `tasm config` when tmux isn't running still works (no tmux check)
- [ ] Running `tasm repos` when tmux isn't running still works
- [ ] If multiple dependencies are missing, all are listed in a single error message
- [ ] `git fetch` failure during create is a warning, not a fatal error

### Files to create/modify

```
internal/deps/check.go   (new)
internal/cli/root.go     (modify — add PersistentPreRunE)
internal/cli/list.go     (modify — add PreRunE)
internal/cli/create.go   (modify — add PreRunE)
internal/cli/attach.go   (modify — add PreRunE)
internal/cli/delete.go   (modify — add PreRunE)
internal/session/create.go (modify — improve error handling)
```

---

## Task 11: End-to-End Integration + README

### What to do

Verify the full workflow works end-to-end and write a README with installation and usage instructions.

### Steps

1. Manually test the full workflow in order:
   - `make build && make install`
   - `tasm config` — verify defaults
   - `tasm repos` — verify repo discovery
   - `tasm create --repo <path> --branch test-session --agent claude` — verify worktree + tmux session + agent launch
   - `tasm list` — verify the new session shows up
   - `tasm attach agent/<repo>/test-session` — verify attach works
   - `tasm delete --force agent/<repo>/test-session` — verify cleanup
   - Source the tmux plugin and test `prefix + A` popup
   - Test the full interactive create flow via the popup
2. Fix any bugs found during testing.
3. Create `README.md` with these sections:
   - **What it is** — one paragraph
   - **Demo** — placeholder for a GIF/screenshot
   - **Dependencies** — list with install commands (tmux, fzf, wt, Go)
   - **Installation** — TPM install (`set -g @plugin '...'`), manual install (`make install`), and how to add the plugin to `.tmux.conf`
   - **Configuration** — show the config file path, full example config with comments, and tmux option overrides
   - **Usage** — keybindings, subcommands, examples
   - **How it works** — brief architecture description

### Acceptance Criteria

- [ ] Full workflow works: create → list → attach → delete
- [ ] Interactive popup works via tmux keybinding
- [ ] Interactive create flow works (repo → branch → agent → session created)
- [ ] Deleting a session cleans up both the tmux session and the worktree
- [ ] README covers installation, configuration, and usage
- [ ] `make build` produces a working binary
- [ ] Plugin loads correctly via TPM
