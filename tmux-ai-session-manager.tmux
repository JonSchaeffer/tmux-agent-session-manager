#!/usr/bin/env bash

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

keybinding="$(tmux show-option -gqv @ai-session-bind)"
keybinding="${keybinding:-A}"

popup_width="$(tmux show-option -gqv @ai-session-popup-width)"
popup_width="${popup_width:-90%}"

popup_height="$(tmux show-option -gqv @ai-session-popup-height)"
popup_height="${popup_height:-85%}"

tmux bind-key "$keybinding" display-popup -E \
  -w "$popup_width" \
  -h "$popup_height" \
  "PLUGIN_DIR='$PLUGIN_DIR' '$PLUGIN_DIR/scripts/popup.sh'"
