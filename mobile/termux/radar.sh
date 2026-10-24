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
ROOTFS="$PREFIX/var/lib/proot-distro/installed-rootfs/$DISTRO"  # only a hint; newer proot-distro stores containers elsewhere
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
# proot-distro login does not pass the inner exit status through, so every
# yes/no question to the workbench is answered by printing a marker.
wb_ok() { wb "{ $1 ; } >/dev/null 2>&1 && echo __RADAR_OK__" | grep -q __RADAR_OK__; }
wb_alive() { timeout 120 "$PREFIX/bin/proot-distro" login "$DISTRO" -- echo __RADAR_OK__ 2>/dev/null | grep -q __RADAR_OK__; }
wb_put() { # wb_put <local file> <path under the workbench home>: copy through stdin,
  # because the container's directory on disk differs between proot-distro versions
  timeout 120 "$PREFIX/bin/proot-distro" login "$DISTRO" -- bash -c \
    'mkdir -p "$(dirname ~/"$1")" && cat > ~/"$1"' _ "$2" < "$1"
}
WB_PATH='export PATH="$HOME/.local/bin:$HOME/.hermes/bin:$HOME/.cargo/bin:$HOME/bin:/usr/local/bin:$PATH" HERDR_PROCESS_DETECTION=child-groups HERDR_SOCKET_PATH=/tmp/herdr.sock; unset TERMUX_VERSION PREFIX; '
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
    echo "workbench: $(wb_alive && echo starts || echo DOES-NOT-START)  (legacy rootfs dir: $([ -d "$ROOTFS/usr" ] && echo yes || echo no))"
    echo "proot-distro: $("$PREFIX/bin/proot-distro" --version 2>&1 | head -1)"
  } >> "$REPORT" 2>&1
  endsection

  wb_alive || { bad "workbench does not start"; return; }
  section "workbench"
  wb 'set +e
    echo "os: $(. /etc/os-release; echo $PRETTY_NAME)"
    for t in herdr git gh node npm python3 claude codex gemini opencode copilot hermes; do
      if command -v $t >/dev/null; then printf "%-8s %s\n" $t "$(timeout 30 $t --version 2>&1 | head -1)"
      else printf "%-8s MISSING\n" $t; fi
    done
    echo "gh auth: $(gh auth status >/dev/null 2>&1 && echo signed-in || echo NOT signed-in)"
    echo "git user: $(git config --global user.name) <$(git config --global user.email | sed "s/^\(..\).*@/\1***@/")>"
    echo "herdr server running: $(pgrep -x herdr >/dev/null && echo yes || echo no)"
    echo "HERDR_SOCKET_PATH in login profile: $(grep -hs HERDR_SOCKET_PATH /etc/profile.d/*.sh | head -1)"
    # does a herdr server answer on a socket at this path? (isolated config dir)
    probe() {
      rm -rf "$1"; mkdir -p "$1"
      env -u HERDR_SOCKET_PATH XDG_CONFIG_HOME="$1" timeout 25 herdr server >"$1.log" 2>&1 &
      local pid=$! r=no i
      for i in $(seq 1 20); do
        sleep 1
        python3 -c "import socket,sys; s=socket.socket(socket.AF_UNIX); s.settimeout(2); s.connect(sys.argv[1])" "$1/herdr/herdr-client.sock" 2>/dev/null && { r="yes (${i}s)"; break; }
      done
      kill $pid 2>/dev/null; wait $pid 2>/dev/null
      echo "herdr socket probe $1: $r"; [ "$r" = no ] && tail -3 "$1.log"
      rm -rf "$1" "$1.log"
    }
    if command -v herdr >/dev/null; then probe /tmp/rprobe; probe /root/.rprobe-long; fi
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

# --version alone passed on the device while Hermes printed "dependency repair failed"
# `hermes --version` printed a version while every real command crashed with
# "No module named 'ruamel'", so judge Hermes by `hermes status`.
hermes_ok() { wb_ok 'command -v hermes && out="$(timeout 180 hermes status </dev/null 2>&1)"; ! printf "%s" "$out" | grep -qiE "Traceback|ModuleNotFoundError|ImportError|retry limit reached|refusing to update|repair failed"'; }

# --- 3. repair ------------------------------------------------------------
repair() {
  step "repairing"
  section "repairs"
  termux_health

  if ! wb_alive; then
    log "workbench missing: running full setup"
    curl -fsSL "$RAW/setup.sh" | bash >> "$REPORT" 2>&1
  fi

  for f in "$PREFIX/bin/dev" "$HOME/.termux/boot/10-herdr-server"; do
    if [ -f "$f" ] && ! grep -q PROOT_NO_SECCOMP "$f"; then
      sed -i '2i export PROOT_NO_SECCOMP=1  # radar: proot new_child crash workaround' "$f"
      log "added proot crash workaround to $f"
    fi
  done

  printf '%s\n' '# radar: herdr sockets must stay under the 107-byte unix socket limit' \
    'export HERDR_SOCKET_PATH=/tmp/herdr.sock' > "$STATE/herdr-sockets.sh"
  wb_put "$STATE/herdr-sockets.sh" ../etc/profile.d/herdr-sockets.sh && log "herdr socket profile installed"

  log "--- fixing apt/dpkg inside the workbench"
  wb 'dpkg --configure -a; apt-get -f install -y' | tail -5 | redact >> "$REPORT"

  # refresh the workbench helpers from main without the full first-time setup
  curl -fsSL "$RAW/tools.conf" -o "$STATE/tools.conf" && wb_put "$STATE/tools.conf" .workbench/tools.conf.default

  wb '[ -f ~/.workbench/tools.conf ] || cp ~/.workbench/tools.conf.default ~/.workbench/tools.conf' >/dev/null
  if ! wb_ok 'command -v dev-update'; then
    # an older hand-made workbench: install our helpers and missing tools once
    log "--- workbench helpers missing: running workbench-setup.sh"
    curl -fsSL "$RAW/workbench-setup.sh" -o "$STATE/workbench-setup.sh" && wb_put "$STATE/workbench-setup.sh" .workbench/setup.sh
    WB_TIMEOUT=3600 wb 'bash ~/.workbench/setup.sh </dev/tty' | tee "$STATE/workbench-setup.log"
    tail -60 "$STATE/workbench-setup.log" | redact >> "$REPORT"
  fi
  if wb_ok 'for t in herdr claude codex gemini; do command -v $t || exit 1; done'; then
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
  # A fresh install replaces trees Hermes refuses to update (on the device:
  # "venv/bin/ty is owned by uid 10362", then "No module named 'ruamel'").
  if ! hermes_ok; then
    log "--- hermes still broken after reinstall: pm repair"
    wb 'timeout 900 hermes pm repair </dev/null 2>&1 | tail -15' | redact >> "$REPORT"
  fi
  log "--- hermes doctor --fix"
  wb 'timeout 300 hermes doctor --fix' | tail -40 | redact >> "$REPORT"
  hermes_ok && log "
hermes: WORKING" || log "
hermes: STILL BROKEN"

  # the herdr client starts its own server; radar only makes sure the
  # Termux-side launchers (dev, boot script) exist and carry the proot fix
  if ! grep -qs PROOT_NO_SECCOMP "$PREFIX/bin/dev" || [ ! -x "$HOME/.termux/boot/10-herdr-server" ]; then
    curl -fsSL "$RAW/setup.sh" -o "$STATE/setup.sh" && bash "$STATE/setup.sh" --launchers >> "$REPORT" 2>&1 && log "installed dev launcher and boot script"
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
# summarize the final state only (after repair when there was one)
{ grep -q '^# after repair' "$REPORT" && sed -n '/^# after repair/,$p' "$REPORT" || cat "$REPORT"; } | grep -oE "herdr socket probe /tmp[^ ]*: [a-z]+|^[a-z]+ +MISSING|NOT signed-in|hermes: (STILL BROKEN|WORKING)|assertion failed|Error code: 402[^,]*" | sort -u | sed 's/^/  /'
hermes_ok && ok "Hermes works" || bad "Hermes still broken (the report tells Claude why)"
if [ "$MODE" != "scan" ] && [ -t 0 ]; then
  echo; echo "opening herdr..."; sleep 2
  if command -v dev >/dev/null; then exec dev; fi
  exec proot-distro login "$DISTRO" --shared-tmp -- bash -lc 'cd ~/code 2>/dev/null; exec herdr'
fi
