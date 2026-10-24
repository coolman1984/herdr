#!/usr/bin/env bash
# Runs inside the Ubuntu workbench (proot) as root. Installs herdr, git, gh,
# Node.js, the agent CLIs and Hermes Agent, then the `dev-update` and `ship`
# helpers. Safe to re-run.
set -euo pipefail
WB="$HOME/.workbench"
# shellcheck source=tools.conf
. "$WB/tools.conf"
export DEBIAN_FRONTEND=noninteractive
# Hermes' installer refuses Termux; this is a glibc Ubuntu, so hide Termux markers.
unset TERMUX_VERSION PREFIX || true
export PATH="$HOME/.local/bin:$HOME/.hermes/bin:$PATH"

say() { printf '\n\033[1;35m[workbench] %s\033[0m\n' "$*"; }
try() { "$@" || printf '\033[1;33mwarning:\033[0m failed: %s (continuing)\n' "$*"; }
tty_ok() { [ -r /dev/tty ] && : </dev/tty 2>/dev/null; }

say "base packages"
apt-get update -y
apt-get upgrade -y
# shellcheck disable=SC2086
apt-get install -y --no-install-recommends ca-certificates curl wget gnupg git git-lfs \
  openssh-client build-essential python3 python3-pip python3-venv pipx ripgrep fd-find \
  jq unzip zip less nano vim tmux htop ffmpeg locales tzdata $APT_EXTRA
sed -i 's/^# *en_US.UTF-8/en_US.UTF-8/' /etc/locale.gen && try locale-gen

if ! command -v node >/dev/null || [ "$(node -p 'process.versions.node.split(".")[0]')" -lt 22 ]; then
  say "Node.js 22 LTS"
  curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
  apt-get install -y nodejs
fi

if ! command -v gh >/dev/null; then
  say "GitHub CLI"
  install -d -m 755 /etc/apt/keyrings
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg -o /etc/apt/keyrings/githubcli-archive-keyring.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    > /etc/apt/sources.list.d/github-cli.list
  apt-get update -y && apt-get install -y gh
fi

say "workbench helpers (dev-update, ship)"
cat > /usr/local/bin/dev-update <<'UPD'
#!/usr/bin/env bash
# dev-update        -> update if the last update is older than UPDATE_EVERY_HOURS
# dev-update --now  -> update everything now
WB="$HOME/.workbench"; . "$WB/tools.conf"
unset TERMUX_VERSION PREFIX || true
export PATH="$HOME/.local/bin:$HOME/.hermes/bin:$PATH" DEBIAN_FRONTEND=noninteractive
stamp="$WB/last-update"
if [ "${1:-}" != "--now" ]; then
  [ "${UPDATE_EVERY_HOURS:-0}" -gt 0 ] || exit 0
  [ -f "$stamp" ] && [ $(( $(date +%s) - $(cat "$stamp") )) -lt $(( UPDATE_EVERY_HOURS * 3600 )) ] && exit 0
fi
exec 9>"$WB/update.lock"; flock -n 9 || exit 0
step() { echo "== $*"; "$@" || echo "!! failed: $*"; }
npm_pkgs=()
[ "${CODEX:-0}" = 1 ]    && npm_pkgs+=(@openai/codex)
[ "${GEMINI:-0}" = 1 ]   && npm_pkgs+=(@google/gemini-cli)
[ "${OPENCODE:-0}" = 1 ] && npm_pkgs+=(opencode-ai)
[ "${COPILOT:-0}" = 1 ]  && npm_pkgs+=(@github/copilot)
# shellcheck disable=SC2206
npm_pkgs+=($NPM_EXTRA)
step apt-get update -y && step apt-get upgrade -y
[ ${#npm_pkgs[@]} -gt 0 ] && step npm install -g "${npm_pkgs[@]/%/@latest}"
if [ "${CLAUDE:-0}" = 1 ]; then
  if command -v claude >/dev/null; then step claude update
  else step bash -c 'curl -fsSL https://claude.ai/install.sh | bash || npm install -g @anthropic-ai/claude-code'; fi
fi
if [ "${HERDR:-0}" = 1 ]; then
  if command -v herdr >/dev/null; then step herdr update
  else step sh -c 'curl -fsSL https://herdr.dev/install.sh | sh'; fi
fi
if [ "${HERMES:-0}" = 1 ]; then
  if command -v hermes >/dev/null; then step hermes update
  else step bash -c 'curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash -s -- --skip-setup --skip-browser'; fi
fi
if command -v herdr >/dev/null; then
  for agent in claude codex gemini opencode copilot hermes; do
    command -v "$agent" >/dev/null && herdr integration install "$agent" >/dev/null 2>&1 || true
  done
fi
date +%s > "$stamp"
echo "== workbench up to date"
UPD
cat > /usr/local/bin/ship <<'SHIP'
#!/usr/bin/env bash
# ship [message]  -> commit everything in this repo and push it to GitHub
set -euo pipefail
git rev-parse --is-inside-work-tree >/dev/null
git add -A
git diff --cached --quiet || git commit -m "${*:-wip: $(date '+%Y-%m-%d %H:%M')}"
branch="$(git branch --show-current)"
for wait in 0 2 4 8 16; do
  sleep "$wait"
  git push -u origin "$branch" && { echo "pushed $branch"; exit 0; }
done
echo "push failed: check the network, then run ship again" >&2; exit 1
SHIP
chmod +x /usr/local/bin/dev-update /usr/local/bin/ship

say "installing and updating every tool (first run takes a while)"
dev-update --now

say "shell defaults"
mkdir -p "$HOME/code"
# login shells (`dev`, `dev shell`) read /etc/profile.d; Ubuntu's .bashrc
# returns early for non-interactive shells, so the environment lives here
cat > /etc/profile.d/herdr-mobile.sh <<'RC'
export PATH="$HOME/.local/bin:$HOME/.hermes/bin:$PATH"
export LANG=en_US.UTF-8 EDITOR=nano
# proot hides the terminal foreground process group; let herdr infer agents
export HERDR_PROCESS_DETECTION=child-groups
unset TERMUX_VERSION PREFIX
alias fd=fdfind 2>/dev/null
(dev-update >> "$HOME/.workbench/update.log" 2>&1 &)
RC

say "git + GitHub sign-in"
git config --global init.defaultBranch main
git config --global pull.rebase false
git config --global push.autoSetupRemote true
if tty_ok; then
  if [ -z "$(git config --global user.name || true)" ]; then
    read -r -p "your name for git commits: " n </dev/tty; git config --global user.name "$n"
  fi
  if [ -z "$(git config --global user.email || true)" ]; then
    read -r -p "your email for git commits: " e </dev/tty; git config --global user.email "$e"
  fi
  if ! gh auth status >/dev/null 2>&1; then
    echo "GitHub sign-in: copy the code it shows, open the link in your phone browser."
    try gh auth login --hostname github.com --git-protocol https --web </dev/tty
  fi
fi
gh auth status >/dev/null 2>&1 && try gh auth setup-git
[ -f "$HOME/.ssh/id_ed25519" ] || ssh-keygen -q -t ed25519 -N "" -f "$HOME/.ssh/id_ed25519" -C "phone-workbench"
gh auth status >/dev/null 2>&1 && try gh ssh-key add "$HOME/.ssh/id_ed25519.pub" --title "phone-workbench"

say "ready"
for t in herdr git gh node claude codex gemini opencode copilot hermes; do
  printf '  %-9s %s\n' "$t" "$(command -v "$t" >/dev/null && echo ok || echo missing)"
done
