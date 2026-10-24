#!/data/data/com.termux/files/usr/bin/bash
# radar: scan the phone workbench, repair what it can, start herdr, and upload
# a redacted report to the private GitHub repo <you>/phone-radar so the
# workbench can be diagnosed remotely.
#
#   radar           scan + repair + report + open herdr
#   radar scan      scan + report (repairs only broken Termux packages)
#   radar hermes    force a clean Hermes reinstall (keeps ~/.hermes settings)
#
# radar updates itself from main on every run, so fixes ship by editing this file.
set -uo pipefail

# radar must run in Termux itself, not inside the Ubuntu workbench
if [ ! -x "${PREFIX:-/nonexistent}/bin/pkg" ]; then
  printf '\n\033[1;33m⚠ You are inside the Ubuntu workbench.\033[0m\n'
  printf '  Type:  exit   then paste the radar command again.\n'
  printf '  اكتب exit وبعدين الصق أمر الرادار تاني.\n\n'
  exit 1
fi

RAW="${HERDR_MOBILE_RAW:-https://raw.githubusercontent.com/coolman1984/herdr/${HERDR_MOBILE_REF:-main}/mobile/termux}"
DISTRO="${HERDR_MOBILE_DISTRO:-ubuntu}"
ROOTFS="$PREFIX/var/lib/proot-distro/installed-rootfs/$DISTRO"
MODE="${1:-fix}"
STATE="$HOME/.radar"; mkdir -p "$STATE"
STAMP="$(date -u +%Y%m%d-%H%M%S)"
REPORT="$STATE/report-$STAMP.md"
export PROOT_NO_SECCOMP=1   # works around proot's "new_child: assertion failed" crash

ok()   { printf '  \033[32m✔\033[0m %s\n' "$*"; }
bad()  { printf '  \033[31m✘\033[0m %s\n' "$*"; }
step() { printf '\n\033[1;36m▶ %s\033[0m\n' "$*"; }
log()  { printf '%s\n' "$*" >> "$REPORT"; }
section() { printf '\n## %s\n\n```\n' "$1" >> "$REPORT"; }
endsection() { printf '```\n' >> "$REPORT"; }
redact() {
  sed -E \
    -e 's/(gh[pousr]_|github_pat_)[A-Za-z0-9_]+/\1[REDACTED]/g' \
    -e 's/(sk-|sk-ant-|xox[abp]-|AIza)[A-Za-z0-9_-]+/\1[REDACTED]/g' \
    -e 's/((api[_-]?key|token|secret|password|authorization|bearer)[^:=]{0,20}[:=][[:space:]]*)[^[:space:]]+/\1[REDACTED]/Ig' \
    -e 's/[A-Za-z0-9_+\/=-]{40,}/[REDACTED-LONG]/g'
}
WB_PATH='export PATH="$HOME/.local/bin:$HOME/.hermes/bin:$HOME/.cargo/bin:$HOME/bin:/usr/local/bin:$PATH" HERDR_PROCESS_DETECTION=child-groups; unset TERMUX_VERSION PREFIX; '
wb() { # run a command inside the workbench, bounded. Plain (non-login) bash:
  # the login profile starts a background update that would keep proot open.
  timeout --foreground "${WB_TIMEOUT:-900}" "$PREFIX/bin/proot-distro" login "$DISTRO" --shared-tmp -- \
    bash -c "$WB_PATH$1" 2>&1
}

# --- 1. self-update -------------------------------------------------------
if [ -z "${RADAR_FRESH:-}" ]; then
  if curl -fsSL "$RAW/radar.sh" -o "$STATE/radar.new" 2>/dev/null && bash -n "$STATE/radar.new"; then
    if ! cmp -s "$STATE/radar.new" "$PREFIX/bin/radar"; then
      install -m 755 "$STATE/radar.new" "$PREFIX/bin/radar"
      echo "radar updated to the latest version"
      RADAR_FRESH=1 exec "$PREFIX/bin/radar" "$@"
    fi
  fi
fi

printf '# phone radar report %s (mode: %s)\n' "$STAMP" "$MODE" > "$REPORT"

# --- 2. scan --------------------------------------------------------------
scan() {
  step "scanning the phone"
  section "phone"
  {
    echo "android: $(getprop ro.build.version.release 2>/dev/null) sdk $(getprop ro.build.version.sdk 2>/dev/null)"
    echo "model: $(getprop ro.product.manufacturer 2>/dev/null) $(getprop ro.product.model 2>/dev/null)"
    echo "arch: $(uname -m)  kernel: $(uname -r)"
    echo "termux: ${TERMUX_VERSION:-unknown}"
    grep -E 'MemTotal|MemAvailable' /proc/meminfo
    df -h "$HOME" | tail -1
    for p in proot-distro proot git gh curl termux-api; do
      printf '%-13s %s\n' "$p" "$(dpkg-query -W -f='${Version}' "$p" 2>/dev/null || echo missing)"
    done
    echo "boot script: $([ -x "$HOME/.termux/boot/10-herdr-server" ] && echo yes || echo no)"
    echo "launcher seccomp fix: $(grep -q PROOT_NO_SECCOMP "$PREFIX/bin/dev" 2>/dev/null && echo yes || echo no)"
    echo "workbench rootfs: $([ -d "$ROOTFS/usr" ] && echo present || echo missing)"
  } >> "$REPORT" 2>&1
  endsection

  [ -d "$ROOTFS/usr" ] || { bad "workbench missing"; return; }
  section "workbench"
  wb 'set +e
    echo "os: $(. /etc/os-release; echo $PRETTY_NAME)"
    for t in herdr git gh node npm python3 claude codex gemini opencode copilot hermes; do
      if command -v $t >/dev/null; then printf "%-8s %s\n" $t "$(timeout 30 $t --version 2>&1 | head -1)"
      else printf "%-8s MISSING\n" $t; fi
    done
    echo "gh auth: $(gh auth status >/dev/null 2>&1 && echo signed-in || echo NOT signed-in)"
    echo "git user: $(git config --global user.name) <$(git config --global user.email | sed "s/^\(..\).*@/\1***@/")>"
    echo "herdr server running: $(pgrep -f "herdr server" >/dev/null && echo yes || echo no)"
    echo "binaries named herdr/hermes:"; find / -xdev \( -path /proc -o -path /sys -o -path /data -o -path /storage -o -path /sdcard \) -prune -o -type f \( -name herdr -o -name hermes \) -print 2>/dev/null | head -10
    echo "PATH lines in shell files:"; grep -hs "PATH" ~/.bashrc ~/.profile ~/.bash_profile /etc/profile.d/*.sh | head -10
    echo "dpkg audit: $(dpkg --audit 2>&1 | head -3)"
    echo "last update: $(cat ~/.workbench/last-update 2>/dev/null | xargs -I{} date -d @{} 2>/dev/null || echo never)"
    echo; echo "--- update.log (tail)"; tail -n 30 ~/.workbench/update.log 2>/dev/null
    echo; echo "--- server.log (tail)"; tail -n 20 ~/.workbench/server.log 2>/dev/null
    echo; echo "--- hermes dump"; timeout 120 hermes dump 2>&1 | head -80
    echo; echo "--- hermes doctor"; timeout 180 hermes doctor 2>&1 | tail -60
    echo; echo "--- hermes logs (tail)"; ls -t ~/.hermes/logs/*.log 2>/dev/null | head -2 | xargs -r tail -n 25
  ' | redact >> "$REPORT"
  endsection
  grep -q "assertion" "$REPORT" && log "NOTE: proot assertion seen during scan"
}

# Termux packages must move together: installing one package on a stale
# system (the first device run) left proot-distro pointing at a missing
# python3.14. A full upgrade brings every dependency to the same release.
pd_ok() { "$PREFIX/bin/proot-distro" --help >/dev/null 2>&1; }
termux_health() {
  local missing=() p
  for p in proot-distro python git gh curl termux-api; do dpkg -s "$p" >/dev/null 2>&1 || missing+=("$p"); done
  if [ ${#missing[@]} -eq 0 ] && pd_ok; then return 0; fi
  step "repairing Termux packages (full upgrade, a few minutes)"
  {
    echo "termux repair: missing=[${missing[*]}] proot-distro-ok=$(pd_ok && echo yes || echo no)"
    # apt directly: the pkg wrapper needs curl, which a half-upgraded system breaks
    apt-get update
    apt-get full-upgrade -y -o Dpkg::Options::=--force-confnew
    apt-get install -y -o Dpkg::Options::=--force-confnew proot-distro python git gh curl termux-api
  } 2>&1 | tee -a "$STATE/termux-repair.log" | tail -3
  hash -r
  if pd_ok; then ok "proot-distro works"; log "termux repair: proot-distro works"
  else bad "proot-distro still broken"; log "termux repair FAILED: $(head -1 "$PREFIX/bin/proot-distro")"; tail -20 "$STATE/termux-repair.log" >> "$REPORT"; fi
}

hermes_ok() { wb 'command -v hermes >/dev/null && timeout 60 hermes --version >/dev/null 2>&1' >/dev/null; }

# --- 3. repair ------------------------------------------------------------
repair() {
  step "repairing"
  section "repairs"
  termux_health

  if [ ! -d "$ROOTFS/usr" ]; then
    log "workbench missing: running full setup"
    curl -fsSL "$RAW/setup.sh" | bash >> "$REPORT" 2>&1
  fi

  for f in "$PREFIX/bin/dev" "$HOME/.termux/boot/10-herdr-server"; do
    if [ -f "$f" ] && ! grep -q PROOT_NO_SECCOMP "$f"; then
      sed -i '2i export PROOT_NO_SECCOMP=1  # radar: proot new_child crash workaround' "$f"
      log "added proot crash workaround to $f"
    fi
  done

  log "--- fixing apt/dpkg inside the workbench"
  wb 'dpkg --configure -a; apt-get -f install -y' | tail -5 | redact >> "$REPORT"

  # refresh the workbench helpers from main without the full first-time setup
  mkdir -p "$ROOTFS/root/.workbench"
  curl -fsSL "$RAW/tools.conf" -o "$ROOTFS/root/.workbench/tools.conf.default" 2>/dev/null

  [ -f "$ROOTFS/root/.workbench/tools.conf" ] || cp "$ROOTFS/root/.workbench/tools.conf.default" "$ROOTFS/root/.workbench/tools.conf"
  if ! wb 'command -v dev-update >/dev/null' >/dev/null; then
    # an older hand-made workbench: install our helpers and missing tools once
    log "--- workbench helpers missing: running workbench-setup.sh"
    curl -fsSL "$RAW/workbench-setup.sh" -o "$ROOTFS/root/.workbench/setup.sh"
    WB_TIMEOUT=3600 wb 'bash ~/.workbench/setup.sh </dev/tty' | tee "$STATE/workbench-setup.log"
    tail -60 "$STATE/workbench-setup.log" | redact >> "$REPORT"
  fi
  if wb 'for t in herdr claude codex gemini; do command -v $t >/dev/null || exit 1; done' >/dev/null; then
    log "core tools present"
  else
    log "--- tools missing: dev-update --now"
    wb 'dev-update --now' | tail -40 | redact >> "$REPORT"
  fi

  if [ "$MODE" = "hermes" ] || ! hermes_ok; then
    log "--- hermes broken or reinstall requested: clean reinstall (settings kept)"
    wb 'set +e
      unset TERMUX_VERSION PREFIX
      H="${HERMES_HOME:-$HOME/.hermes}"
      [ -d "$H/hermes-agent" ] && mv "$H/hermes-agent" "$H/hermes-agent.broken-$(date +%s)"
      rm -f "$HOME/.local/bin/hermes"
      curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash -s -- --skip-setup --skip-browser
      ls -dt "$H"/hermes-agent.broken-* 2>/dev/null | tail -n +3 | xargs -r rm -rf' | tail -40 | redact >> "$REPORT"
  fi
  log "--- hermes doctor --fix"
  wb 'timeout 300 hermes doctor --fix' | tail -40 | redact >> "$REPORT"
  hermes_ok && log "hermes: WORKING" || log "hermes: STILL BROKEN"

  if ! wb 'pgrep -f "herdr server" >/dev/null' >/dev/null; then
    (proot-distro login "$DISTRO" --shared-tmp -- bash -c "$WB_PATH"'mkdir -p ~/code ~/.workbench; cd ~/code && exec herdr server >> ~/.workbench/server.log 2>&1' >/dev/null 2>&1 &)
    log "started herdr server"
  fi
  endsection
}

# --- 4. upload ------------------------------------------------------------
upload() {
  step "sending the report to GitHub (private repo phone-radar)"
  local token user repo path
  token="$(gh auth token 2>/dev/null || true)"
  [ -n "$token" ] || token="$(wb 'gh auth token' 2>/dev/null | tail -1 | tr -d '\r')"
  case "$token" in gh*_*|github_pat_*) ;; *)
    if : </dev/tty 2>/dev/null; then
      printf '\n  \033[1;33mGitHub sign-in (one time):\033[0m copy the 8-letter code, press Enter, then paste it in the page that opens.\n'
      printf '  تسجيل دخول GitHub مرة واحدة: انسخ الكود، اضغط Enter، والصقه في الصفحة اللي هتفتح.\n\n'
      gh auth login --hostname github.com --git-protocol https --web </dev/tty
      token="$(gh auth token 2>/dev/null || true)"
    fi ;;
  esac
  case "$token" in gh*_*|github_pat_*) ;; *) bad "not signed in to GitHub; report kept on the phone"; return 1 ;; esac
  export GH_TOKEN="$token"
  user="$(gh api user -q .login 2>/dev/null)" || { bad "GitHub not reachable"; return 1; }
  repo="$user/phone-radar"
  gh repo view "$repo" >/dev/null 2>&1 || gh repo create "$repo" --private --add-readme \
    -d "Private phone workbench reports for Claude" >/dev/null 2>&1
  for path in "reports/$STAMP.md" "latest.md"; do
    local sha; sha="$(gh api "repos/$repo/contents/$path" -q .sha 2>/dev/null || true)"
    gh api -X PUT "repos/$repo/contents/$path" -f message="radar: $STAMP ($MODE)" \
      -f content="$(base64 -w0 < "$REPORT")" ${sha:+-f sha="$sha"} >/dev/null 2>&1 \
      || { bad "upload failed"; return 1; }
  done
  ok "report sent: github.com/$repo/blob/main/latest.md"
}

# --- run ------------------------------------------------------------------
termux_health
scan
if [ "$MODE" != "scan" ]; then
  repair
  printf '\n# after repair\n' >> "$REPORT"
  scan
fi
cp "$REPORT" "$HOME/storage/downloads/radar-report.md" 2>/dev/null
upload || echo "report saved: ~/storage/downloads/radar-report.md"

step "summary"
grep -E "MISSING|NOT signed-in|STILL BROKEN|WORKING|assertion" "$REPORT" | sort -u | sed 's/^/  /'
hermes_ok && ok "Hermes works" || bad "Hermes still broken (the report tells Claude why)"
if [ "$MODE" != "scan" ] && [ -t 0 ]; then
  echo; echo "opening herdr..."; sleep 2
  if command -v dev >/dev/null; then exec dev; fi
  exec proot-distro login "$DISTRO" --shared-tmp -- bash -lc 'cd ~/code 2>/dev/null; exec herdr'
fi
