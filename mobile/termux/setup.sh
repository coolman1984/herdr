#!/data/data/com.termux/files/usr/bin/bash
# One-command Android workbench: Termux + an Ubuntu userland (proot) holding
# herdr, git, gh, node, the coding-agent CLIs and Hermes Agent.
#
#   curl -fsSL https://raw.githubusercontent.com/coolman1984/herdr/main/mobile/termux/setup.sh | bash
#
# Safe to re-run: every step skips work that is already done.
set -euo pipefail

REPO_RAW="${HERDR_MOBILE_RAW:-https://raw.githubusercontent.com/coolman1984/herdr/${HERDR_MOBILE_REF:-main}/mobile/termux}"
DISTRO="${HERDR_MOBILE_DISTRO:-ubuntu}"
ROOTFS="$PREFIX/var/lib/proot-distro/installed-rootfs/$DISTRO"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || true)"

say() { printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

[ -n "${PREFIX:-}" ] && [ -d "$PREFIX" ] && [[ "$PREFIX" == *com.termux* ]] \
  || die "run this inside the Termux app (install it from F-Droid or GitHub, not Google Play)"
[ "$(uname -m)" = "aarch64" ] || printf 'warning: %s is untested; arm64 phones are the target\n' "$(uname -m)"

fetch() { # fetch <name> <dest>: prefer the local checkout, else download
  if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/$1" ]; then cp "$SCRIPT_DIR/$1" "$2"
  else curl -fsSL "$REPO_RAW/$1" -o "$2"; fi
}

say "updating Termux packages"
export DEBIAN_FRONTEND=noninteractive
pkg update -y -o Dpkg::Options::=--force-confnew
pkg upgrade -y -o Dpkg::Options::=--force-confnew
pkg install -y proot-distro git openssh curl termux-api termux-tools

if [ ! -d "$HOME/storage" ]; then
  say "asking for storage access (tap Allow) so ~/storage/shared reaches your phone files"
  termux-setup-storage || true
fi

if [ ! -d "$ROOTFS/usr" ]; then
  say "installing the $DISTRO workbench (one time, a few minutes)"
  proot-distro install "$DISTRO"
else
  say "$DISTRO workbench already installed"
fi

say "setting up tools inside the workbench"
mkdir -p "$ROOTFS/root/.workbench"
fetch workbench-setup.sh "$ROOTFS/root/.workbench/setup.sh"
fetch tools.conf "$ROOTFS/root/.workbench/tools.conf.default"
[ -f "$ROOTFS/root/.workbench/tools.conf" ] || cp "$ROOTFS/root/.workbench/tools.conf.default" "$ROOTFS/root/.workbench/tools.conf"
if : </dev/tty 2>/dev/null; then
  proot-distro login "$DISTRO" --shared-tmp -- bash /root/.workbench/setup.sh </dev/tty
else
  proot-distro login "$DISTRO" --shared-tmp -- bash /root/.workbench/setup.sh
fi

say "installing the 'dev' launcher"
cat > "$PREFIX/bin/dev" <<LAUNCH
#!/data/data/com.termux/files/usr/bin/bash
# dev          -> open herdr in ~/code inside the workbench
# dev shell    -> plain shell inside the workbench
# dev update   -> update everything now
# dev <cmd...> -> run one command inside the workbench
termux-wake-lock 2>/dev/null || true
export PROOT_NO_SECCOMP=1  # radar: proot new_child crash workaround
binds=()
[ -d /storage/emulated/0 ] && binds+=(--bind /storage/emulated/0:/sdcard)
run() { exec proot-distro login $DISTRO --shared-tmp "\${binds[@]}" -- bash -lc "\$1"; }
case "\${1:-}" in
  "")     run 'cd ~/code && exec herdr' ;;
  shell)  run 'cd ~/code && exec bash -l' ;;
  update) run 'dev-update --now' ;;
  *)      run "cd ~/code && \$(printf '%q ' "\$@")" ;;
esac
LAUNCH
chmod +x "$PREFIX/bin/dev"

say "installing radar (scan, repair and report: type radar)"
fetch radar.sh "$PREFIX/bin/radar" && chmod +x "$PREFIX/bin/radar"

say "keeping the herdr server alive after phone reboot (needs the Termux:Boot app)"
mkdir -p "$HOME/.termux/boot"
cat > "$HOME/.termux/boot/10-herdr-server" <<BOOT
#!/data/data/com.termux/files/usr/bin/bash
export PROOT_NO_SECCOMP=1  # radar: proot new_child crash workaround
termux-wake-lock
proot-distro login $DISTRO --shared-tmp -- bash -lc 'cd ~/code && exec herdr server >> ~/.workbench/server.log 2>&1' &
BOOT
chmod +x "$HOME/.termux/boot/10-herdr-server"

grep -q 'herdr-mobile' "$HOME/.bashrc" 2>/dev/null || cat >> "$HOME/.bashrc" <<'RC'
# herdr-mobile: type `dev` to open herdr, `dev shell` for a shell
echo "type: dev  (herdr)  |  dev shell  |  dev update  |  radar (fix everything)"
RC

say "done. type:  dev"
cat <<'MSG'
Recommended once:
  - Android settings > Apps > Termux > Battery > Unrestricted
  - install Termux:Boot and Termux:API from the same store you got Termux from, open Termux:Boot once
MSG
