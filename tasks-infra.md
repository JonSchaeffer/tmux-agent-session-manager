# Infrastructure & Hardening Tasks

> **These tasks are LINEAR — complete them in order (Task 1 → 4).** Each builds on the previous one, and the ordering avoids file conflicts (the disambiguation fix lands before tests so the tests cover it; CI/release lands before the bootstrap since the bootstrap downloads the release assets).
> Implement each task in its own commit.
> The project is a Go CLI (`tasm`) + shell scripts for a tmux plugin. Project root: `/Users/jon/code/tmux-agent-session-manager`. Build with `GOPATH=$HOME/go go build ./cmd/tasm` (the leading `GOPATH` is a local-env workaround; plain `go build` may also work on CI).

---

## Task 1: Fix Disambiguated Repo Names Breaking Session Names

### Problem

When two repos under `repo_root` share a directory name, `repo.Disambiguate()` renames one to include its parent, e.g. `work/api` (containing a `/`). But session names are built as `agent/<repo>/<branch>`, so a disambiguated repo produces `agent/work/api/feature`. The session-name parser splits on `/` into at most 3 parts, so it reads `repo = "work"` and `branch = "api/feature"` — wrong. This corrupts the repo/branch shown in `list` and could misroute delete logic.

### What to do

Ensure the repo segment of a session name never contains `/`. Sanitize the repo name when building session names (slashes → hyphens), so a disambiguated `work/api` becomes the segment `work-api` and parses back cleanly.

### Steps

1. Add a helper in `internal/session/session.go`:
   ```go
   func sanitizeRepoSegment(name string) string  // replaces "/" with "-"
   ```
2. Apply it wherever a session name is constructed from a repo name:
   - `internal/session/create.go` — when building `sessionName` in `Create()`. (`CreateOpts.RepoName` from the CLI path never has a slash, but the interactive flow passes the disambiguated name, so sanitize unconditionally here.)
   - `scripts/create.sh` — the `repo_name` extracted from `tasm repos` output (column 1) can contain `/`. Sanitize it (`repo_name="${repo_name//\//-}"`) before building the session name for both the `has-session` check and the final `attach`.
3. Confirm the existing parser stays correct: with a sanitized single-segment repo, `strings.SplitN(name, "/", 3)` yields exactly `["agent", "<repo>", "<branch>"]` even when the branch itself contains slashes.
4. Verify delete is unaffected: `internal/session/delete.go` derives the repo path from the session's working directory (via `tmux display-message`/git), not from the repo segment, so sanitizing the display segment is safe.

### Acceptance Criteria

- [ ] A disambiguated repo named `work/api` produces session name `agent/work-api/<branch>`, not `agent/work/api/<branch>`
- [ ] `tasm list` shows the correct repo and branch for such sessions
- [ ] Branch names containing `/` (e.g. `feature/auth`) still parse correctly alongside the sanitized repo segment
- [ ] The interactive create flow (`scripts/create.sh`) sanitizes the repo name consistently for the existence check, creation, and attach
- [ ] Delete still cleans up the correct worktree (unaffected, since it uses the session path)

### Files to modify

```
internal/session/session.go   (add sanitizeRepoSegment helper)
internal/session/create.go    (sanitize when building sessionName)
scripts/create.sh             (sanitize repo_name before building session name)
```

---

## Task 2: Unit Tests

### Problem

The codebase has zero tests. There's no automated guard against regressions (e.g., the config-parsing bug where keys nested under `agents:` were silently accepted, or the session-name parsing fixed in Task 1).

### What to do

Add Go unit tests for the pure logic. Some functions currently bury pure logic inside `exec.Command` calls — extract those into testable helpers first, then test them.

### Refactors needed for testability

1. **`internal/session/session.go`** — extract the per-line parsing inside `List()` into a pure function:
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
  - `Discover()` against a temp dir tree with fake repos (dirs with a `.git` subdir), a worktree (a `.git` *file*), a `node_modules` dir, and nesting beyond depth 3 → only real repos returned, worktrees/node_modules/deep dirs skipped, sorted alphabetically.
  - `Disambiguate()` table-driven: no collisions unchanged; two same-named repos get parent-prefixed names.
- `internal/session/session_test.go`:
  - `parseSessionLine()` table-driven: valid `agent/repo/branch` line parses correctly; branch names containing `/` (e.g. `agent/repo/feature/auth`) keep the full branch; non-`agent/` lines rejected; malformed lines rejected.
  - `sanitizeRepoSegment()` (from Task 1): `work/api` → `work-api`; names without slashes unchanged. Include a round-trip: sanitize a repo, build `agent/<repo>/<branch>`, parse it back, confirm repo+branch are correct.
  - `RelativeTime()`: seconds/minutes/hours/days boundaries.
- `internal/session/create_test.go`:
  - `parseWorktreePath()` against canned porcelain output → correct path for the branch; error when branch absent.
  - `validBranch` regex: accepts `feature/auth`, `fix-bug_1`; rejects names with spaces or `~`, `:`.

### Acceptance Criteria

- [ ] `go test ./...` passes
- [ ] `parseSessionLine` and `parseWorktreePath` extracted as pure functions and used by their callers (no behavior change)
- [ ] Config validation rules each have at least one passing and one failing test case
- [ ] Repo discovery test uses a temp dir tree and verifies worktree (`.git` file) exclusion
- [ ] `sanitizeRepoSegment` has a round-trip test (sanitize → build name → parse back)
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

## Task 3: Version Stamping + CI + Release Pipeline

### Problem

There's no CI (nothing checks build/vet/fmt/tests on push), no way to produce installable binaries (needed by Task 4), and no way to tell what version a binary is — which matters once Task 4 starts downloading binaries onto users' machines.

### What to do

Add build-time version stamping with a `tasm version` command, a **CI** workflow on push/PR, and a **release** workflow on version tags that publishes versioned, per-platform binaries to GitHub Releases.

### Steps

1. **Version stamping**:
   - Add `var version = "dev"` to the `internal/cli` package (e.g. a new `internal/cli/version.go`).
   - Add a `version` cobra subcommand that prints it; register it in `root.go`. Also wire `--version` on the root command (cobra's `Version` field) to the same value.
   - The build injects it via ldflags targeting that package var:
     ```
     -ldflags "-s -w -X github.com/jonschaeffer/tmux-agent-session-manager/internal/cli.version=$VERSION"
     ```
     (`-s -w` also strips debug info → ~3 MB binaries instead of ~4.5 MB.)
   - Update the `Makefile` `build` target to pass these ldflags, deriving `VERSION` from `git describe --tags --always --dirty` (fall back to `dev` if not a git repo).
2. Create `.github/workflows/ci.yml`:
   - Triggers: `push` and `pull_request`.
   - Single job on `ubuntu-latest` with `actions/setup-go` (Go version matching `go.mod`, currently `1.25.x`).
   - Steps: `go build ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"` (fail if anything is unformatted), `go test ./...`.
3. Create `.github/workflows/release.yml`:
   - Trigger: `push` on tags matching `v*`.
   - Cross-compile a matrix: `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`.
   - **`CGO_ENABLED=0` is required** — without it, cross-compiling to linux fails trying to invoke a C toolchain. Also pass the version ldflags from step 1 with `VERSION=${GITHUB_REF_NAME}` (the tag).
   - Name each asset with a **stable scheme** Task 4 relies on (no version in the filename, so the `releases/latest/download/` URL is predictable):
     ```
     tasm-<os>-<arch>      e.g. tasm-darwin-arm64
     ```
   - Attach all four binaries to a GitHub Release for the tag (`softprops/action-gh-release` or `gh release create`).
   - **Comment the asset-naming scheme at the top of the workflow** so Task 4 stays in sync.
4. Add a CI status badge to the top of `README.md`.

### Acceptance Criteria

- [ ] `tasm version` and `tasm --version` print the injected version; a plain `go build` (no ldflags) prints `dev`
- [ ] `make build` stamps the version from `git describe` and strips with `-s -w`
- [ ] `ci.yml` runs build, vet, gofmt check, and tests on every push and PR; fails if `gofmt -l` is non-empty
- [ ] `release.yml` triggers on `v*` tags and produces 4 statically-linked binaries (`CGO_ENABLED=0`)
- [ ] Release binaries report the tag version via `tasm version`
- [ ] Assets are named exactly `tasm-<os>-<arch>`
- [ ] A Release is created with the binaries attached
- [ ] README has a CI status badge; Go version in workflows matches `go.mod`

### Files to create/modify

```
internal/cli/version.go         (new — version var + version subcommand)
internal/cli/root.go            (register versionCmd, set rootCmd.Version)
Makefile                        (version ldflags + -s -w in build target)
.github/workflows/ci.yml        (new)
.github/workflows/release.yml   (new)
README.md                       (add CI badge)
```

---

## Task 4: TPM-Friendly Binary Bootstrap

### Problem

TPM installs the plugin by `git clone` only — it never builds the Go binary. So a fresh `set -g @plugin 'JonSchaeffer/tmux-agent-session-manager'` install leaves users with no `tasm` binary, and the popup just errors with "binary not found." The plugin needs to obtain the binary itself on load.

> Depends on Task 3 for the release-asset naming scheme: `tasm-<os>-<arch>` (e.g. `tasm-darwin-arm64`).

### What to do

Make the `tmux-agent-session-manager.tmux` entrypoint ensure a working binary exists on load, downloading the matching prebuilt release binary if needed, with a build-from-source fallback.

### Steps

1. Create `scripts/ensure-binary.sh` (executable):
   - Resolve `PLUGIN_DIR` and the target path `$PLUGIN_DIR/bin/tasm`.
   - If `$PLUGIN_DIR/bin/tasm` exists and is executable → exit 0. If `tasm` is already on `$PATH` → exit 0.
   - Otherwise, attempt download:
     - OS: `uname -s` → `Darwin`→`darwin`, `Linux`→`linux`. Unsupported → source fallback.
     - Arch: `uname -m` → `arm64`/`aarch64`→`arm64`, `x86_64`→`amd64`.
     - URL: `https://github.com/JonSchaeffer/tmux-agent-session-manager/releases/latest/download/tasm-<os>-<arch>`
     - `curl -fsSL` to `$PLUGIN_DIR/bin/tasm`, `chmod +x`, verify it runs (`bin/tasm version` exits 0). If the download is missing/invalid, fall through.
   - **Source fallback**: if download failed and `go` is on `$PATH`, run `make build` in `$PLUGIN_DIR`.
   - If both fail: print `"tasm: could not download a prebuilt binary or build from source. Install Go, or download manually from the releases page."` and exit 1.
   - Silent on success (don't spam tmux startup).
2. Update `tmux-agent-session-manager.tmux`:
   - Before binding the key, run `"$PLUGIN_DIR/scripts/ensure-binary.sh" &` in the background so a slow download doesn't block tmux startup.
3. Update `README.md`:
   - Fix the "Via TPM" section: the binary is fetched automatically on first load (prebuilt download, or built from source if Go is present). Keep manual `make install` as an alternative.

### Acceptance Criteria

- [ ] On a fresh clone with no binary, sourcing the plugin downloads the correct `tasm-<os>-<arch>` binary into `bin/`
- [ ] If the binary already exists or `tasm` is on `$PATH`, no download happens
- [ ] OS/arch detection covers darwin+linux and arm64+amd64
- [ ] If download fails but Go is installed, it builds from source
- [ ] If both fail, a clear actionable error is printed
- [ ] The bootstrap does not block tmux startup (runs in background) and is silent on success
- [ ] README's TPM section accurately describes the auto-bootstrap
- [ ] Asset URL matches Task 3's naming exactly

### Files to create/modify

```
scripts/ensure-binary.sh             (new, executable)
tmux-agent-session-manager.tmux      (call ensure-binary.sh on load)
README.md                            (fix TPM install section)
```
