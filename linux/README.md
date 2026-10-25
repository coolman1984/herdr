# Herdr on Linux

One-time setup from the repo root:

```bash
./linux/install.sh
```

Start Herdr (GUI or `herdr` in a terminal), then open the six-agent column layout:

```bash
herdr-layout
```

`HERDR_HOME` defaults to the repo directory (override if the fork lives elsewhere). Scripts on `PATH`: `herdr-agents`, `herdr-layout`. Agent profiles and approval flags come from [`../agents.json`](../agents.json); separate sign-ins use `~/.herdr-accounts/<tool>/<name>`.

Dry-run install: `./linux/install.sh --dry-run`. Layout plan only: `herdr-layout --dry-run`.
