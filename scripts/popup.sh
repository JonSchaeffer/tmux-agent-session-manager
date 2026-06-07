#!/usr/bin/env bash

PLUGIN_DIR="${PLUGIN_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# Locate tasm binary.
if command -v tasm &>/dev/null; then
  TAISM="tasm"
elif [[ -x "$PLUGIN_DIR/bin/tasm" ]]; then
  TAISM="$PLUGIN_DIR/bin/tasm"
else
  echo "tasm binary not found. Run 'make install' in the plugin directory."
  exit 1
fi

session_list="$("$TAISM" list 2>/dev/null)"
combined="$(printf '[Create new session]\n%s' "$session_list")"

# Count actual sessions (non-empty lines), then scale the preview window so
# fewer sessions get a larger preview and more sessions get a smaller one.
# Past a threshold the list needs all the room, so the preview is hidden.
session_count="$(printf '%s' "$session_list" | grep -c '[^[:space:]]')"
if   (( session_count <= 2 ));  then preview_window="up,80%,wrap,nohidden"
elif (( session_count <= 5 ));  then preview_window="up,65%,wrap,nohidden"
elif (( session_count <= 10 )); then preview_window="up,45%,wrap,nohidden"
elif (( session_count <= 15 )); then preview_window="up,30%,wrap,nohidden"
else                                 preview_window="hidden"
fi

selected="$(echo "$combined" | fzf \
  --height=100% \
  --reverse \
  --no-sort \
  --no-multi \
  --header "enter: attach | tab/shift-tab: navigate | alt-bspace: delete | ctrl-n: new" \
  --preview 'session={}; session="${session%% *}"; if [ "$session" = "[Create" ]; then printf "tasm — tmux agent session manager\n\nCreate a new AI coding session:\n  1. Pick a git repo from your configured repo_root\n  2. Name a branch (becomes a git worktree via wt)\n  3. Pick an AI agent to launch\n\nEach session gets its own isolated worktree so\nmultiple agents can work on the same repo in parallel.\n\nPress Enter to get started."; else tmux capture-pane -e -t "$session" -p -S -50 2>/dev/null || echo "No preview available"; fi' \
  --preview-window "$preview_window" \
  --bind "tab:down,shift-tab:up" \
  --bind "alt-bspace:execute-silent($TAISM delete --force {1})+reload($TAISM list | { printf '[Create new session]\n'; cat; })" \
  --bind "ctrl-n:print([Create new session])+accept")"

[[ -z "$selected" ]] && exit 0

if [[ "$selected" == "[Create new session]" ]]; then
  exec "$PLUGIN_DIR/scripts/create.sh"
else
  session_name="$(echo "$selected" | awk '{print $1}')"
  exec "$TAISM" attach "$session_name"
fi
