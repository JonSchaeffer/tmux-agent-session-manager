#!/usr/bin/env bash

PLUGIN_DIR="${PLUGIN_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

if command -v taism &>/dev/null; then
  TAISM="taism"
elif [[ -x "$PLUGIN_DIR/bin/taism" ]]; then
  TAISM="$PLUGIN_DIR/bin/taism"
else
  echo "taism binary not found. Run 'make install' in the plugin directory."
  exit 1
fi

# Step 1: Pick a repo.
repo_line="$("$TAISM" repos | fzf --height=100% --prompt "repo> " --reverse --no-multi --bind "tab:down,shift-tab:up")"
[[ -z "$repo_line" ]] && exit 0
repo_path="$(echo "$repo_line" | awk -F'\t' '{print $2}')"
repo_name="$(echo "$repo_line" | awk -F'\t' '{print $1}')"

# Step 2: Name the branch (freehand input). Loop until unique name or user cancels.
while true; do
  branch="$(echo "" | fzf --height=100% --prompt "branch name> " --print-query --no-info 2>/dev/null | head -1)"
  [[ -z "$branch" ]] && exit 0

  session_name="ai/${repo_name}/${branch}"
  if tmux has-session -t "$session_name" 2>/dev/null; then
    printf 'Session %s already exists. Attach to it? [Y/n] ' "$session_name"
    read -r answer
    if [[ -z "$answer" || "$answer" =~ ^[Yy] ]]; then
      exec "$TAISM" attach "$session_name"
    fi
    # User said no — loop back to re-prompt for a branch name.
    continue
  fi
  break
done

# Step 3: Pick an agent (skip if only one configured).
agent_list="$("$TAISM" agents 2>/dev/null)"
agent_count="$(echo "$agent_list" | grep -c '[^ ]')"

if [[ "$agent_count" -le 1 ]]; then
  agent="$(echo "$agent_list" | head -1)"
else
  agent="$(echo "$agent_list" | fzf --height=100% --prompt "agent> " --reverse)"
  [[ -z "$agent" ]] && exit 0
fi

# Step 4: Create the session.
if ! "$TAISM" create --repo "$repo_path" --branch "$branch" --agent "$agent"; then
  echo ""
  echo "Press any key to close..."
  read -r -n1
  exit 1
fi

"$TAISM" attach "ai/${repo_name}/${branch}"
