#!/usr/bin/env bash

PLUGIN_DIR="${PLUGIN_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# Locate taism binary.
if command -v taism &>/dev/null; then
  TAISM="taism"
elif [[ -x "$PLUGIN_DIR/bin/taism" ]]; then
  TAISM="$PLUGIN_DIR/bin/taism"
else
  echo "taism binary not found. Run 'make install' in the plugin directory."
  exit 1
fi

session_list="$("$TAISM" list 2>/dev/null)"
combined="$(printf '[Create new session]\n%s' "$session_list")"

selected="$(echo "$combined" | fzf \
  --reverse \
  --no-sort \
  --no-multi \
  --header "enter: attach | tab/shift-tab: navigate | alt-bspace: delete | ctrl-n: new" \
  --preview 'session={}; session="${session%% *}"; if [ "$session" = "[Create" ]; then echo "Create a new AI coding session"; else tmux capture-pane -e -t "$session" -p -S -50 2>/dev/null || echo "No preview available"; fi' \
  --preview-window "up,70%,wrap,nohidden" \
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
