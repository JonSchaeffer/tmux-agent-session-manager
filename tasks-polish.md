# Polish Tasks

> These tasks are **independent** — they can be worked on in parallel by separate agents.
> Each task should be implemented in its own commit.
> Refer to `plan.md` for the full spec. The codebase is a Go CLI (`tasm`) + shell scripts for a tmux plugin.

---

## Task P1: Config Validation

### Problem

The config file at `~/.config/tasm/config.yaml` is parsed with `gopkg.in/yaml.v3`. If a user accidentally indents keys under the wrong parent (e.g., puts `default_agent` and `keybinding` under `agents:`), those keys silently become agent entries. There is no validation that catches this. The tool should fail loudly on bad config rather than behaving unexpectedly.

### What to do

Add validation to the config loading in `internal/config/config.go` that runs after unmarshaling.

### Steps

1. Add a `Validate() error` method on the `Config` struct in `internal/config/config.go`.
2. The method should check:
   - `RepoRoot` is non-empty.
   - `DefaultAgent` is non-empty and exists as a key in the `Agents` map. If the user sets `default_agent: aider` but doesn't define `aider` in `agents:`, that's an error: `"default_agent 'aider' is not defined in agents"`.
   - Each key in `Agents` is a simple name (alphanumeric + hyphens only, no spaces). This catches keys like `default_agent` or `keybinding` that were accidentally indented under `agents:`: `"invalid agent name 'default_agent': agent names must be alphanumeric (hyphens allowed)"`.
   - Each value in `Agents` is non-empty (the agent command can't be blank).
   - `Keybinding` is a single character.
   - `PopupWidth` and `PopupHeight` match the pattern `\d+%` or are pure integers.
3. Call `Validate()` at the end of `Load()`, after all merging and tilde expansion. Return the validation error if any.
4. When validation fails, the error message should include the config file path so the user knows where to fix: `"/Users/jon/.config/tasm/config.yaml: default_agent 'aider' is not defined in agents"`.

### Acceptance Criteria

- [ ] A config with `default_agent: aider` but no `aider` in `agents` produces a clear error naming the config file
- [ ] A config with `default_agent` accidentally nested under `agents:` produces: `"invalid agent name 'default_agent'..."` 
- [ ] A config with an empty agent command (e.g., `claude: ""`) produces an error
- [ ] A config with `keybinding: AI` (multi-char) produces an error
- [ ] A valid config (or no config file at all) passes validation with no errors
- [ ] All error messages include the config file path
- [ ] `tasm config` shows the validation error and exits 1 on bad config
- [ ] All other subcommands that load config also fail on bad config (they all call `config.Load()`)

### Files to modify

```
internal/config/config.go   (add Validate method, call it in Load)
```

---

## Task P2: Session Status Detection (Running vs Idle)

### Problem

The session list currently shows the agent name and creation time, but doesn't indicate whether the agent is actively running or has exited (idle shell). Users can't tell at a glance which sessions still have an active agent.

### What to do

Add a `Status` field to the `Session` struct and detect whether the agent process is still the foreground process in the tmux pane.

### Steps

1. Add a `Status string` field to the `Session` struct in `internal/session/session.go` (with json tag `"status"`). Values: `"running"` or `"idle"`.
2. Create a `detectStatus(sessionName string) string` function:
   - Run `tmux list-panes -t <sessionName> -F "#{pane_current_command}"`.
   - If the pane's current command is a shell (`bash`, `zsh`, `sh`, `fish`), the agent has exited — return `"idle"`.
   - If the pane's current command is anything else (the agent or a subprocess of the agent), return `"running"`.
   - On error, return `"unknown"`.
3. Call `detectStatus()` in the `List()` function and populate each session's `Status` field.
4. Update the display format in `internal/cli/list.go` to include the status. Change the format string from:
   ```
   %-40s %-10s %s
   ```
   to:
   ```
   %-40s %-10s %-10s %s
   ```
   So the output looks like:
   ```
   agent/myapp/feature-auth                    claude     running    3m ago
   agent/myapp/fix-logging                     claude     idle       12m ago
   ```
5. Include `Status` in the `--json` output (it's automatic since the struct field has a json tag).

### Acceptance Criteria

- [ ] `tasm list` shows `running` or `idle` for each session
- [ ] A session with an active agent (e.g., claude is the foreground process) shows `running`
- [ ] A session where the agent has exited (pane is back to a shell like zsh/bash) shows `idle`
- [ ] `tasm list --json` includes a `"status"` field on each session object
- [ ] The popup display (which pipes `tasm list`) shows the status column between agent and time
- [ ] If pane inspection fails, status defaults to `"unknown"`

### Files to modify

```
internal/session/session.go  (add Status field, detectStatus function, call in List)
internal/cli/list.go         (update format string to include status column)
```

---

## Task P3: Graceful Create Failure Rollback

### Problem

If session creation fails partway through (e.g., `wt` succeeds but `tmux new-session` fails), the worktree is left orphaned with no tmux session pointing to it. The user has to manually clean up with `wt remove`.

### What to do

Add rollback logic to `session.Create()` that cleans up partial state on failure.

### Steps

1. In `internal/session/create.go`, restructure `Create()` to track what was created and roll back on failure:
   - After `wt switch --create` succeeds, if any subsequent step fails, run `wt remove <branch>` (with working directory set to `RepoPath`) before returning the error.
   - After `tmux new-session` succeeds, if agent launch fails, run `tmux kill-session -t <sessionName>` and then `wt remove <branch>` before returning the error.
2. Implement this with a cleanup pattern. One clean approach:
   ```go
   var worktreeCreated bool
   var sessionCreated bool
   
   cleanup := func() {
       if sessionCreated {
           exec.Command("tmux", "kill-session", "-t", sessionName).Run()
       }
       if worktreeCreated {
           cmd := exec.Command("wt", "remove", opts.Branch)
           cmd.Dir = opts.RepoPath
           cmd.Run()
       }
   }
   ```
   Call `cleanup()` before each error return after the relevant step has succeeded. Do NOT use `defer cleanup()` — cleanup should only run on failure, not on success.
3. If rollback itself fails (e.g., `wt remove` errors during cleanup), append a warning to the error: `"additionally, rollback failed: <reason>. Manual cleanup may be needed."`.

### Acceptance Criteria

- [ ] If `tmux new-session` fails after `wt` succeeds, the worktree is automatically removed
- [ ] If `tmux send-keys` (agent launch) fails after the session is created, both the session and worktree are removed
- [ ] If rollback fails, the error message includes both the original error and the rollback failure
- [ ] Successful creation (the happy path) is not affected — no cleanup runs
- [ ] `git fetch` failure does NOT trigger any rollback (it's a warning, not an error)

### Files to modify

```
internal/session/create.go  (restructure Create with rollback logic)
```

---

## Task P4: Stale Session Cleanup Command

### Problem

Over time, users will accumulate sessions where the agent has exited and the work is done. There's no easy way to clean up all idle sessions at once. Users have to delete them one by one.

### What to do

Add a `tasm cleanup` subcommand that finds and deletes stale sessions.

### Steps

1. Create `internal/cli/cleanup.go` with a new `cleanup` cobra command:
   - It should call `session.List()` to get all sessions.
   - Filter to sessions where the pane's foreground process is a shell (same logic as status detection from Task P2 — if P2 is not yet done, implement the shell detection inline here).
   - Print the list of stale sessions to the user:
     ```
     Found 3 idle sessions:
       agent/myapp/feature-auth        idle  2h ago
       agent/infra/old-refactor        idle  1d ago
       agent/myapp/fix-typo            idle  3d ago
     ```
   - Prompt for confirmation: `"Delete all 3 idle sessions? [y/N]"`.
   - Accept a `--force` flag to skip confirmation.
   - Accept a `--older-than` flag (duration string like `1h`, `30m`, `7d`) to only clean up sessions older than the given threshold. Default: no threshold (all idle sessions).
   - For each confirmed session, call `session.Delete()`. Print each deletion. If one fails, print a warning and continue with the rest (don't abort).
   - Print a summary at the end: `"Deleted 3 sessions, 0 errors"` or `"Deleted 2 sessions, 1 error"`.
2. Register the command in `internal/cli/root.go`'s `init()`.
3. Add `PreRunE` that calls both `deps.Check()` and `deps.CheckTmuxRunning()`.

### Acceptance Criteria

- [ ] `tasm cleanup` lists all idle sessions and asks for confirmation before deleting
- [ ] `tasm cleanup --force` skips confirmation
- [ ] `tasm cleanup --older-than 1h` only targets idle sessions created more than 1 hour ago
- [ ] Each deletion removes both the tmux session and the worktree
- [ ] If one deletion fails, the others still proceed
- [ ] A summary line is printed at the end with counts
- [ ] If no idle sessions exist, prints `"No idle sessions found"` and exits 0
- [ ] The command has proper `PreRunE` checks for dependencies and tmux

### Files to create/modify

```
internal/cli/cleanup.go  (new)
internal/cli/root.go     (register cleanupCmd in init)
```

---

## Task P5: Duplicate Session Prevention in the UI

### Problem

If a user tries to create a session for a repo + branch combination that already exists, the `tasm create` command returns an error. But this happens late in the flow — after the user has already gone through three fzf prompts. It's a bad experience.

### What to do

Detect the conflict early in the create flow and warn the user or offer to attach instead.

### Steps

1. Modify `scripts/create.sh` to check for an existing session after the user picks a repo and branch name (after Step 2, before Step 3).
2. After the user types a branch name, construct the session name: `agent/<repo_name>/<branch>`.
3. Run `tmux has-session -t "agent/<repo_name>/<branch>" 2>/dev/null`. If it exits 0, the session already exists.
4. When a conflict is detected, show a prompt:
   ```
   Session agent/myapp/feature-auth already exists. Attach to it? [Y/n]
   ```
   - If yes (or Enter), run `tasm attach agent/<repo_name>/<branch>` and exit.
   - If no, go back to the branch name step (re-prompt for a different branch name). Implement this as a simple loop around Step 2.

### Acceptance Criteria

- [ ] If a session already exists for the chosen repo + branch, the user is warned before agent selection
- [ ] The user can choose to attach to the existing session directly
- [ ] The user can choose to go back and pick a different branch name
- [ ] If the session doesn't exist, the create flow proceeds as normal
- [ ] The check uses the same naming convention (`agent/<repo>/<branch>`) as `tasm create`

### Files to modify

```
scripts/create.sh  (add session existence check after branch name step)
```

---

## Task P6: Improve `tasm list` Empty State in Popup

### Problem

When there are no AI sessions, the popup shows only `[Create new session]` with an empty preview pane. This is functional but gives no context to a first-time user about what the tool does or how to use it.

### What to do

Show a helpful welcome message in the preview pane when `[Create new session]` is selected and there are no existing sessions.

### Steps

1. Modify `scripts/popup.sh`'s preview command. When the selected item is `[Create new session]`, instead of the single line `"Create a new AI coding session"`, show a more helpful message:
   ```
   tasm — tmux agent session manager

   Create a new AI coding session:
     1. Pick a git repo from your configured repo_root
     2. Name a branch (becomes a git worktree via wt)
     3. Pick an AI agent to launch

   Each session gets its own isolated worktree so
   multiple agents can work on the same repo in parallel.

   Press Enter to get started.
   ```
2. This is a shell string change only — no Go code needed.

### Acceptance Criteria

- [ ] When `[Create new session]` is highlighted, the preview shows the welcome/help message
- [ ] When an existing session is highlighted, the preview still shows the tmux pane capture
- [ ] The message is readable and fits within the preview pane

### Files to modify

```
scripts/popup.sh  (update the preview command's Create branch)
```
