# Infrastructure & Hardening Tasks

> These tasks have some ordering: **T2 (CI/releases)** defines the release-asset naming that **T3 (TPM auto-binary)** depends on. T1 (tests) and T4 (bug fix) are independent and can run in parallel with anything.
> Each task should be implemented in its own commit.
> The project is a Go CLI (`tasm`) + shell scripts for a tmux plugin. Project root: `/Users/jon/code/tmux-agent-session-manager`. Build with `GOPATH=$HOME/go go build ./cmd/tasm`.

---

## Task T1: Unit Tests

### Problem

The codebase has zero tests. Multiple agents are editing it in parallel, and there's no automated guard against regressions (e.g., the config-parsing bug where keys nested under `agents:` were silently accepted).

### What to do

Add Go unit tests for the pure logic. Some functions currently bury pure logic inside `exec.Command` calls — extract those into testable helpers first, then test them.

### Refactors needed for testability

1. **`internal/session/session.go`** — the per-line parsing inside `List()` should be extracted into a pure function:
   ```go
   func parseSessionLine(line string) (Session, bool)
   ```
   It takes a tab-separated `name\tcreated\tpath` line and returns the parsed `Session` and whether it's a valid `agent/` session. `List()` then calls it in the loop. (Agent/status detection still need tmux, so leave those out of the pure function — set them after parsing.)
2. **`internal/session/create.go`** — extract the porcelain parsing from `findWorktreePath` into:
   ```go
   func parseWorktreePath(porcelain, branch string) (string, error)
   ```
   so it can be tested against canned `git worktree list --porcelain` output.

### Tests to write

- `internal/config/config_test.go`:
  - `Load()` with `XDG_CONFIG_HOME` pointed at a temp dir containing a valid config → correct merged values.
  - `Load()` with no config file → defaults, no error.
  - `Validate()` table-driven: valid config passes; `default_agent` not in `agents` fails; agent key with invalid chars fails; empty agent command fails; multi-char keybinding fails; bad popup dimension fails.
  - `expandTilde("~/code")` → `$HOME/code`; non-tilde paths unchanged.
- `internal/repo/discovery_test.go`:
  - `Discover()` against a temp dir tree with fake repos (create dirs with a `.git` subdir), a worktree (a `.git` *file*), a `node_modules` dir, and nesting beyond depth 3 → only real repos returned, worktrees/node_modules/deep dirs skipped, sorted alphabetically.
  - `Disambiguate()` table-driven: no collisions unchanged; two same-named repos get parent-prefixed names.
- `internal/session/session_test.go`:
  - `parseSessionLine()` table-driven: valid `agent/repo/branch` line parses correctly; branch names containing `/` (e.g. `agent/repo/feature/auth`) keep the full branch; non-`agent/` lines rejected; malformed lines rejected.
  - `RelativeTime()`: seconds/minutes/hours/days boundaries.
- `internal/session/create_test.go`:
  - `parseWorktreePath()` against canned porcelain output → correct path for the branch; error when branch absent.
  - `validBranch` regex: accepts `feature/auth`, `fix-bug_1`; rejects names with spaces or `~`, `:`.

### Acceptance Criteria

- [ ] `go test ./...` passes
- [ ] `parseSessionLine` and `parseWorktreePath` extracted as pure functions and used by their callers (no behavior change)
- [ ] Config validation rules each have at least one passing and one failing test case
- [ ] Repo discovery test uses a temp dir tree and verifies worktree (`.git` file) exclusion
- [ ] Tests use only the stdlib `testing` package (no new test dependencies)
- [ ] No test depends on a running tmux server, real git repos outside temp dirs, or network

### Files to create/modify

```
internal/session/session.go        (extract parseSessionLine)
internal/session/create.go         (extract parseWorktreePath)
internal/config/config_test.go     (new)
internal/repo/discovery_test.go    (new)
internal/session/session_test.go   (new)
internal/session/create_test.go    (new)
```

---

## Task T2: CI + Release Pipeline

### Problem

There's no CI. Nothing checks that the code builds, vets, is formatted, or passes tests on push. There's also no mechanism to produce installable binaries (needed by T3).

### What to do

Add two GitHub Actions workflows: a **CI** workflow on push/PR, and a **release** workflow on version tags that publishes per-platform binaries to GitHub Releases.

### Steps

1. Create `.github/workflows/ci.yml`:
   - Triggers: `push` and `pull_request`.
   - Single job on `ubuntu-latest` with `actions/setup-go` (Go 1.25.x — match `go.mod`).
   - Steps: `go build ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"` (fail if anything is unformatted), `go test ./...`.
2. Create `.github/workflows/release.yml`:
   - Trigger: `push` on tags matching `v*`.
   - Build a matrix of binaries: `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`.
   - Name each asset using a **stable, predictable scheme** that T3 will rely on:
     ```
     tasm-<os>-<arch>
     ```
     where `<os>` is `darwin` or `linux` and `<arch>` is `arm64` or `amd64`. Example: `tasm-darwin-arm64`.
   - Use `GOOS`/`GOARCH` env to cross-compile (`CGO_ENABLED=0`).
   - Attach all built binaries to a GitHub Release for the tag (use `softprops/action-gh-release` or `gh release create`).
   - **Document the asset-naming scheme in a comment at the top of the workflow** so T3 stays in sync.
3. Add a CI status badge to the top of `README.md`.

### Acceptance Criteria

- [ ] `ci.yml` runs build, vet, gofmt check, and tests on every push and PR
- [ ] CI fails if code is unformatted (`gofmt -l` non-empty)
- [ ] `release.yml` triggers on `v*` tags and produces 4 binaries (darwin/linux × arm64/amd64)
- [ ] Release assets follow the exact naming `tasm-<os>-<arch>` (no version in the filename, so T3 can construct URLs predictably)
- [ ] Binaries are statically linked (`CGO_ENABLED=0`)
- [ ] A Release is created with the binaries attached
- [ ] README has a CI status badge
- [ ] The Go version in both workflows matches `go.mod`

### Files to create/modify

```
.github/workflows/ci.yml        (new)
.github/workflows/release.yml   (new)
README.md                       (add CI badge)
```

---

## Task T3: TPM-Friendly Binary Bootstrap

### Problem

TPM installs the plugin by `git clone` only — it never builds the Go binary. So a fresh `set -g @plugin 'jonschaeffer/tmux-agent-session-manager'` install leaves users with no `tasm` binary, and the popup just errors with "binary not found." The plugin needs to obtain the binary itself on load.

> Depends on T2 for the release-asset naming scheme: `tasm-<os>-<arch>` (e.g. `tasm-darwin-arm64`).

### What to do

Make the `tmux-agent-session-manager.tmux` entrypoint ensure a working binary exists on load, downloading a prebuilt release binary if needed, with a build-from-source fallback.

### Steps

1. Create `scripts/ensure-binary.sh`:
   - Resolve `PLUGIN_DIR` and the target path `$PLUGIN_DIR/bin/tasm`.
   - If `$PLUGIN_DIR/bin/tasm` already exists and is executable → exit 0 (nothing to do).
   - If `tasm` is already on the user's `$PATH` → exit 0.
   - Otherwise, attempt download:
     - Detect OS: `uname -s` → `Darwin`→`darwin`, `Linux`→`linux`. Unsupported → skip to source fallback.
     - Detect arch: `uname -m` → `arm64`/`aarch64`→`arm64`, `x86_64`→`amd64`.
     - Construct the latest-release asset URL:
       ```
       https://github.com/jonschaeffer/tmux-agent-session-manager/releases/latest/download/tasm-<os>-<arch>
       ```
     - `curl -fsSL` the asset to `$PLUGIN_DIR/bin/tasm`, `chmod +x` it. Verify it runs (`tasm --help` exits 0); if the download is missing/invalid, fall through.
   - **Source fallback**: if download failed and `go` is on `$PATH`, run `make build` (or `go build -o bin/tasm ./cmd/tasm`) in `$PLUGIN_DIR`.
   - If both fail, print a clear message: `"tasm: could not download a prebuilt binary or build from source. Install Go, or download manually from the releases page."` and exit 1.
   - Keep it quiet on success (no output) so it doesn't spam tmux startup.
2. Update `tmux-agent-session-manager.tmux`:
   - Before binding the key, run `"$PLUGIN_DIR/scripts/ensure-binary.sh"` (run it in the background with `&` so a slow download doesn't block tmux startup, OR run synchronously but only the first time — pick background to be safe).
3. Update `README.md`:
   - Fix the "Via TPM" section to explain that the binary is fetched automatically on first load (prebuilt download, or built from source if Go is present).
   - Keep the manual `make install` instructions as an alternative.

### Acceptance Criteria

- [ ] On a fresh clone with no binary, sourcing the plugin downloads the correct `tasm-<os>-<arch>` binary into `bin/`
- [ ] If the binary already exists or `tasm` is on `$PATH`, no download happens
- [ ] OS/arch detection covers darwin+linux and arm64+amd64
- [ ] If download fails but Go is installed, it builds from source
- [ ] If both fail, a clear actionable error is printed
- [ ] The bootstrap does not block tmux startup (runs in background) and is silent on success
- [ ] README's TPM section accurately describes the auto-bootstrap
- [ ] Asset URL matches T2's naming exactly

### Files to create/modify

```
scripts/ensure-binary.sh          (new, executable)
tmux-agent-session-manager.tmux      (call ensure-binary.sh on load)
README.md                         (fix TPM install section)
```

---

## Task T4: Fix Disambiguated Repo Names Breaking Session Names

### Problem

When two repos under `repo_root` share a directory name, `repo.Disambiguate()` renames one to include its parent, e.g. `work/api` (containing a `/`). But session names are built as `agent/<repo>/<branch>`, so a disambiguated repo produces `agent/work/api/feature`. The session-name parser splits on `/` into at most 3 parts, so it reads `repo = "work"` and `branch = "api/feature"` — wrong. This corrupts the repo/branch shown in `list` and could misroute delete logic.

### What to do

Ensure the repo segment of a session name never contains `/`. Sanitize the repo name when building session names (slashes → hyphens), so a disambiguated `work/api` becomes the segment `work-api` and parses back cleanly.

### Steps

1. Add a helper (e.g. in `internal/session/session.go`):
   ```go
   func sanitizeRepoSegment(name string) string  // replaces "/" with "-"
   ```
2. Apply it wherever a session name is constructed from a repo name:
   - `internal/session/create.go` — when building `sessionName` in `Create()`. Note: `CreateOpts.RepoName` is currently set in `internal/cli/create.go` via `filepath.Base(createRepo)`, which never has a slash for the CLI path. But the interactive flow (`scripts/create.sh`) passes the **disambiguated** repo name (column 1 of `tasm repos`, which can be `work/api`) into `agent/${repo_name}/${branch}`. So the fix must cover both the Go side and the shell side.
   - `scripts/create.sh` — the `repo_name` extracted from `tasm repos` output can contain `/`. Sanitize it (e.g. `repo_name="${repo_name//\//-}"`) before building the session name for both the `has-session` check and the final `attach`.
3. Make sure `parseSessionLine` (from T1, if present) and the existing parser remain correct: with a sanitized single-segment repo, `SplitN(name, "/", 3)` yields exactly `["ai", "<repo>", "<branch>"]` even when the branch itself contains slashes.
4. Verify delete still works: `internal/session/delete.go` derives the repo path from the session's working directory (via `tmux display-message`/git), not from the repo segment, so sanitizing the display segment is safe.

### Acceptance Criteria

- [ ] A disambiguated repo named `work/api` produces session name `agent/work-api/<branch>`, not `agent/work/api/<branch>`
- [ ] `tasm list` shows the correct repo and branch for such sessions
- [ ] Branch names containing `/` (e.g. `feature/auth`) still parse correctly alongside the sanitized repo segment
- [ ] The interactive create flow (`scripts/create.sh`) sanitizes the repo name consistently for the existence check, creation, and attach
- [ ] Delete still cleans up the correct worktree (unaffected, since it uses the session path)
- [ ] If T1 is merged, a test covers the sanitize helper and a round-trip parse

### Files to modify

```
internal/session/session.go   (add sanitizeRepoSegment helper)
internal/session/create.go    (sanitize when building sessionName)
scripts/create.sh             (sanitize repo_name before building session name)
```
