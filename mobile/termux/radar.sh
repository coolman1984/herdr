#!/data/data/com.termux/files/usr/bin/bash
# radar: scan the phone workbench, repair what it can, start herdr, and upload
# a redacted report to the private GitHub repo <you>/phone-radar so the
# workbench can be diagnosed remotely.
#
#   radar           scan + repair + report + open herdr (quick scan)
#   radar scan      quick scan + report (repairs only broken Termux packages)
#   radar deep      full scan (Hermes doctor/dump, every binary on disk) + repair
#   radar hermes    force a clean Hermes reinstall (keeps ~/.hermes settings)
#   radar fast      measure proot speed again and pick the faster safe mode
#   radar safe      always use proot's slower crash-safe mode (if you ever see "assertion failed")
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
# proot mode, chosen by `speed` below and shared with the dev launcher and boot script:
# PROOT_NO_SECCOMP=1 avoided a "new_child: assertion failed" crash seen once on the device,
# but makes every system call stop in proot; seccomp mode is faster when it is stable.
PROOT_ENV="$STATE/proot.env"
if [ -f "$PROOT_ENV" ]; then . "$PROOT_ENV"; else export PROOT_NO_SECCOMP=1; fi

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
# Termux's bin dir leaks into the workbench PATH; Hermes' installer then picked
# Termux's python3.14 (bionic) over Ubuntu's 3.13 and built a broken
# "linux-arm64-bionic" runtime ("cannot import name YAML from ruamel.yaml").
WB_PATH='PATH="$(printf %s "$PATH" | tr : "\n" | grep -v "^/data/data/com.termux" | paste -sd: -)"; export PATH="$HOME/.local/bin:$HOME/.hermes/bin:$HOME/.cargo/bin:$HOME/bin:/usr/local/bin:$PATH" HERDR_PROCESS_DETECTION=child-groups HERDR_SOCKET_PATH=/tmp/herdr.sock; unset TERMUX_VERSION PREFIX; '
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
  wb "RADAR_DEEP=$DEEP"'
    set +e
    echo "os: $(. /etc/os-release; echo $PRETTY_NAME)"
    for t in herdr git gh node npm python3 claude codex gemini opencode copilot hermes; do
      if command -v $t >/dev/null; then printf "%-8s %s\n" $t "$(timeout 30 $t --version 2>&1 | head -1)"
      else printf "%-8s MISSING\n" $t; fi
    done
    echo "gh auth: $(gh auth status >/dev/null 2>&1 && echo signed-in || echo NOT signed-in)"
    echo "git user: $(git config --global user.name) <$(git config --global user.email | sed "s/^\(..\).*@/\1***@/")>"
    echo "herdr server running: $(pgrep -x herdr >/dev/null && echo yes || echo no)"
    echo "HERDR_SOCKET_PATH in login profile: $(grep -hs HERDR_SOCKET_PATH /etc/profile.d/*.sh | head -1)"
    echo "herdr mobile tuning: $(grep -hs "radar mobile" ~/.config/herdr/config.toml | head -1)"
    echo "memory: $(free -m 2>/dev/null | awk "/Mem:/{print \$3\" MB used of \"\$2}")"
    [ "$RADAR_DEEP" = 1 ] || exit 0
    echo "binaries named herdr/hermes:"; find / -xdev \( -path /proc -o -path /sys -o -path /data -o -path /storage -o -path /sdcard \) -prune -o -type f \( -name herdr -o -name hermes \) -print 2>/dev/null | head -10
    echo "PATH lines in shell files:"; grep -hs "PATH" ~/.bashrc ~/.profile ~/.bash_profile /etc/profile.d/*.sh | head -10
    echo "dpkg audit: $(dpkg --audit 2>&1 | head -3)"
    echo "last update: $(cat ~/.workbench/last-update 2>/dev/null | xargs -I{} date -d @{} 2>/dev/null || echo never)"
    echo; echo "--- update.log (tail)"; tail -n 30 ~/.workbench/update.log 2>/dev/null
    echo; echo "--- server.log (tail)"; tail -n 20 ~/.workbench/server.log 2>/dev/null
    H="${HERMES_HOME:-$HOME/.hermes}"
    echo "files under ~/.hermes (depth 4) not owned by this user: $(find "$H" -maxdepth 4 ! -user root 2>/dev/null | wc -l)"
    echo "hermes uv cache: $(du -sh "$H/cache/uv" 2>/dev/null | cut -f1)"
    echo "ruamel/yaml/__init__.py copies:"; find "$H" -maxdepth 9 -path "*ruamel/yaml/__init__.py" 2>/dev/null | grep -v broken | head -3 | xargs -r ls -la
    echo; echo "--- hermes dump"; timeout 120 hermes dump 2>&1 | head -80
    echo; echo "--- hermes doctor"; timeout 180 hermes doctor 2>&1 | tail -60
    echo; echo "--- hermes logs (tail)"; ls -t ~/.hermes/logs/*.log 2>/dev/null | head -2 | xargs -r tail -n 25
  ' | redact >> "$REPORT"
  endsection
  grep -q "assertion" "$REPORT" && log "NOTE: proot assertion seen during scan"
  log "proot mode: $(sed -n 's/.*mode=\([a-z]*\).*/\1/p' "$PROOT_ENV" 2>/dev/null | head -1 | grep . || echo safe) ($(head -1 "$PROOT_ENV" 2>/dev/null | sed 's/^# //' || echo 'not measured yet'))"
}

# --- proot speed: measure both modes once a week, keep the faster one -----
# Seccomp mode must win clearly (>= 30 %) because the safe mode avoided a crash once.
speed() {
  [ -f "$STATE/proot-safe" ] && { log "proot mode: safe (owner chose radar safe)"; return; }
  if [ "$MODE" != fast ] && [ -f "$PROOT_ENV" ] && [ -z "$(find "$PROOT_ENV" -mtime +7 2>/dev/null)" ]; then return; fi
  step "measuring workbench speed (two short runs)"
  bench() {
    local t0 t1; t0=$(date +%s%N)
    env "$@" timeout 180 "$PREFIX/bin/proot-distro" login "$DISTRO" -- \
      bash -c 'for i in $(seq 200); do /bin/true; done; echo __RADAR_OK__' 2>/dev/null | grep -q __RADAR_OK__ || { echo fail; return; }
    t1=$(date +%s%N); echo $(( (t1 - t0) / 1000000 ))
  }
  local safe fast; safe=$(bench PROOT_NO_SECCOMP=1); fast=$(bench -u PROOT_NO_SECCOMP)
  if [ "$fast" != fail ] && [ "$safe" != fail ] && [ $(( fast * 10 )) -le $(( safe * 7 )) ]; then
    printf '# radar %s: mode=fast (seccomp) %s ms vs safe %s ms for 200 commands\nunset PROOT_NO_SECCOMP\n' "$STAMP" "$fast" "$safe" > "$PROOT_ENV"
    unset PROOT_NO_SECCOMP; ok "fast mode: ${fast} ms vs ${safe} ms"
  else
    printf '# radar %s: mode=safe %s ms vs seccomp %s ms for 200 commands\nexport PROOT_NO_SECCOMP=1\n' "$STAMP" "$safe" "$fast" > "$PROOT_ENV"
    export PROOT_NO_SECCOMP=1; ok "safe mode kept: ${safe} ms vs ${fast} ms"
  fi
  log "proot speed: safe=${safe} ms seccomp=${fast} ms (200 commands)"
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
hermes_ok() { wb_ok 'command -v hermes >/dev/null || exit 1; out="$(timeout 180 hermes status </dev/null 2>&1)"; ! printf "%s" "$out" | grep -qiE "Traceback|ModuleNotFoundError|ImportError|retry limit reached|refusing to update|repair failed"'; }

# --- 3. repair ------------------------------------------------------------
repair() {
  step "repairing"
  section "repairs"
  termux_health

  if ! wb_alive; then
    log "workbench missing: running full setup"
    curl -fsSL "$RAW/setup.sh" | bash >> "$REPORT" 2>&1
  fi

  speed

  # herdr on a phone: less scrollback per pane (RAM), no sounds (no audio in proot)
  wb 'f=~/.config/herdr/config.toml; mkdir -p ~/.config/herdr; touch "$f"
      grep -q "radar mobile" "$f" && exit 0
      python3 - "$f" <<"PY2"
import sys, tomllib
p = sys.argv[1]; cfg = tomllib.load(open(p, "rb"))
add = ["", "# radar mobile tuning (2026-10-25): delete a section to undo it"]
if "advanced" not in cfg: add += ["[advanced]", "scrollback_limit_bytes = 2000000"]
if "sound" not in cfg.get("ui", {}): add += ["[ui.sound]", "enabled = false"]
open(p, "a").write("\n".join(add) + "\n")
print("herdr mobile tuning applied")
PY2' | tail -2 >> "$REPORT"

  printf '%s\n' '# radar: herdr sockets must stay under the 107-byte unix socket limit' \
    'export HERDR_SOCKET_PATH=/tmp/herdr.sock' > "$STATE/herdr-sockets.sh"
  wb_put "$STATE/herdr-sockets.sh" ../etc/profile.d/herdr-sockets.sh && log "herdr socket profile installed"

  # keep the workbench's dev-update current (it runs `hermes update` every 12 h)
  if curl -fsSL "$RAW/workbench-setup.sh" -o "$STATE/workbench-setup.sh" \
     && sed -n "/<<'UPD'\$/,/^UPD\$/p" "$STATE/workbench-setup.sh" | sed '1d;$d' > "$STATE/dev-update" \
     && [ -s "$STATE/dev-update" ] && wb_put "$STATE/dev-update" ../usr/local/bin/dev-update; then
    wb 'chmod +x /usr/local/bin/dev-update' >/dev/null && log "dev-update refreshed"
  fi

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
      ts=$(date +%s)
      [ -d "$H/hermes-agent" ] && mv "$H/hermes-agent" "$H/hermes-agent.broken-$ts"
      # the reinstall reused the uv cache from the old install and failed the
      # same way ("cannot import name YAML from ruamel.yaml (unknown location)")
      [ -d "$H/cache/uv" ] && mv "$H/cache/uv" "$H/cache/uv.broken-$ts"
      ls -dt "$H"/cache/uv.broken-* 2>/dev/null | tail -n +2 | xargs -r rm -rf
      export UV_LINK_MODE=copy
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
  if ! grep -qs 'radar/proot.env' "$PREFIX/bin/dev" || [ ! -x "$HOME/.termux/boot/10-herdr-server" ]; then
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
DEEP=0; [ "$MODE" = deep ] && DEEP=1
case "$MODE" in
  safe) touch "$STATE/proot-safe"; printf '# radar: mode=safe chosen by the owner\nexport PROOT_NO_SECCOMP=1\n' > "$PROOT_ENV"
        ok "proot safe mode from now on (radar fast to measure again)"; exit 0 ;;
  fast) rm -f "$STATE/proot-safe" ;;
esac
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
{ grep -q '^# after repair' "$REPORT" && sed -n '/^# after repair/,$p' "$REPORT" || cat "$REPORT"; } | grep -oE "proot mode: [a-z]+|^[a-z]+ +MISSING|NOT signed-in|hermes: (STILL BROKEN|WORKING)|assertion failed|Error code: 402[^,]*" | sort -u | sed 's/^/  /'
hermes_ok && ok "Hermes works" || bad "Hermes still broken (the report tells Claude why)"
if [ "$MODE" != "scan" ] && [ -t 0 ]; then
  echo; echo "opening herdr..."; sleep 2
  if command -v dev >/dev/null; then exec dev; fi
  exec proot-distro login "$DISTRO" --shared-tmp -- bash -lc 'cd ~/code 2>/dev/null; exec herdr'
fi
