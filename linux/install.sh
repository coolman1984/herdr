#!/usr/bin/env bash
# Idempotent Linux setup for the owner's Herdr fork. Safe to re-run.
#   ./linux/install.sh
#   ./linux/install.sh --dry-run
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HERDR_HOME="${HERDR_HOME:-$(cd "$SCRIPT_DIR/.." && pwd)}"
BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/herdr"
CONFIG_FILE="$CONFIG_DIR/config.toml"
DRY_RUN=0

say() { printf '\033[1;34m[install]\033[0m %s\n' "$*"; }
run_or_dry() {
  if [[ "$DRY_RUN" -eq 1 ]]; then say "DRY-RUN: $*"; else "$@"; fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help)
      echo "Usage: HERDR_HOME=/path/to/herdr $SCRIPT_DIR/install.sh [--dry-run]"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

link_bin() {
  local bin_name="$1"
  local src="$SCRIPT_DIR/$bin_name"
  [[ -f "$src" ]] || { echo "missing $src" >&2; exit 1; }
  local dest="$BIN_DIR/$bin_name"
  if [[ -L "$dest" && "$(readlink -f "$dest")" == "$(readlink -f "$src")" ]]; then
    say "ok symlink $dest"
    return
  fi
  if [[ -e "$dest" && ! -L "$dest" ]]; then
    say "leave existing file (not our symlink): $dest"
    return
  fi
  run_or_dry mkdir -p "$BIN_DIR"
  run_or_dry ln -sf "$src" "$dest"
  say "linked $dest"
}

copy_if_missing() {
  local from="$1" to="$2"
  [[ -f "$from" ]] || return 0
  if [[ -e "$to" ]]; then
    say "exists (not overwritten): $to"
    return 0
  fi
  run_or_dry mkdir -p "$(dirname "$to")"
  run_or_dry cp "$from" "$to"
  say "copied $to"
}

copy_skill_tree_if_missing() {
  local skill_dir="$1" target_root="$2"
  [[ -d "$skill_dir" ]] || return 0
  local skill
  for skill in "$skill_dir"/*/SKILL.md; do
    [[ -f "$skill" ]] || continue
    local skill_name
    skill_name="$(basename "$(dirname "$skill")")"
    copy_if_missing "$skill" "$target_root/$skill_name/SKILL.md"
  done
}

seed_config() {
  if [[ -f "$CONFIG_FILE" ]]; then
    say "config exists (not modified): $CONFIG_FILE"
    return
  fi
  local seed="$CONFIG_DIR/config.herdr-home.toml"
  local body=""
  body+="# Herdr config seeded by herdr linux/install.sh"$'\n'
  body+="# HERDR_HOME=$HERDR_HOME"$'\n'$'\n'
  if [[ -f "$HERDR_HOME/config/settings.toml" ]]; then
    body+="$(cat "$HERDR_HOME/config/settings.toml")"$'\n'$'\n'
  fi
  if [[ -f "$SCRIPT_DIR/keys.toml" ]]; then
    body+="$(cat "$SCRIPT_DIR/keys.toml")"$'\n'
  fi
  if [[ "$DRY_RUN" -eq 1 ]]; then
    say "DRY-RUN: would write $CONFIG_FILE (${#body} bytes)"
    say "DRY-RUN: would write $seed (export HERDR_HOME)"
    return
  fi
  mkdir -p "$CONFIG_DIR"
  printf '%s' "$body" >"$CONFIG_FILE"
  printf 'export HERDR_HOME=%q\n' "$HERDR_HOME" >"$seed"
  say "wrote $CONFIG_FILE"
  say "wrote $seed (source from shell profile if you want HERDR_HOME everywhere)"
}

install_integrations() {
  local agent
  for agent in claude codex grok opencode; do
    command -v "$agent" >/dev/null 2>&1 || continue
    if [[ "$DRY_RUN" -eq 1 ]]; then
      say "DRY-RUN: herdr integration install $agent"
    else
      herdr integration install "$agent" >/dev/null 2>&1 && say "integration $agent" || say "integration $agent (skipped or failed)"
    fi
  done
  if command -v agy >/dev/null 2>&1 || command -v gemini >/dev/null 2>&1; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
      say "DRY-RUN: herdr integration install antigravity-cli"
    else
      herdr integration install antigravity-cli >/dev/null 2>&1 && say "integration antigravity-cli" || true
    fi
  fi
}

say "HERDR_HOME=$HERDR_HOME"

link_bin herdr-agents
link_bin herdr-layout

copy_if_missing "$HERDR_HOME/AGENTS.md" "$HOME/.codex/AGENTS.md"
copy_skill_tree_if_missing "$HERDR_HOME/skills" "$HOME/.claude/skills"

# Extra account folders: copy skills only when that folder already exists (never create accounts here).
if [[ -d "$HOME/.herdr-accounts/claude" ]]; then
  for acc in "$HOME/.herdr-accounts/claude"/*/; do
    [[ -d "$acc" ]] || continue
    copy_skill_tree_if_missing "$HERDR_HOME/skills" "${acc%/}/skills"
  done
fi

seed_config

if command -v herdr >/dev/null 2>&1; then
  install_integrations
else
  say "herdr not on PATH — skipped integrations"
fi

say "done. Start the team: herdr-layout   (after Herdr is running)"
