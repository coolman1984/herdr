# Herdr Deck - how two agents build it together (2026-10-24)

Main spec: `tasks/herdr-deck-app.md` (read it first, it still rules). Two agents, no overlap:

| Owner | Agent | Files it may write |
|---|---|---|
| BACKEND | Codex GPT-6.1-Sol medium (already building) | `app/main.go`, `app/internal/**`, `app/build.ps1`, `app/tests/**`, `app/API.md`, `app/README.md`, `docs/FILES.md` |
| FRONTEND + FEATURES | Claude Opus 5.5 medium (account 2040) | `app/web/**` (index.html, app.js, style.css, modules), `app/FEATURES.md`, `app/API-REQUESTS.md` |

Rules for both:
- Never edit the other's files. Need a change? Backend writes it in `app/API.md`; frontend asks in
  `app/API-REQUESTS.md` (one request per block: endpoint, JSON in/out, why). Each re-reads the other's
  file before every work step. HISTORY/LESSONS: BACKEND writes the final entries, FRONTEND sends its
  lines to BACKEND by appending them to `app/FEATURES.md` under "For HISTORY/LESSONS".
- CONTRACT FIRST: BACKEND writes `app/API.md` in its first 10 minutes (endpoints, SSE events, JSON
  shapes with examples, auth token rule). FRONTEND builds against it and against a built-in mock
  (`/?mock=1` loads `web/mock.js` with fake state and fake SSE) so it never waits for the backend.
- No frameworks, no CDN, no npm: vanilla JS/CSS, ES modules, < 120 KB total, no timers when hidden.
- Safety unchanged (CLAUDE.md/AGENTS.md): blocked panes are NEVER answered by the app; never read or
  show secrets; test writes only in the isolated lab session.

## The feature list (everything useful - build in this priority order)
P0 (must): live pane grid + focus/zoom, status dots, prompt box per pane, key buttons, close/rename,
  blocked highlight + toast, workspaces/tabs rail, "+ Agent" (profiles from `agents.json`, model,
  effort, account), "+ Shell", Herdr-not-running empty state, Ctrl+K palette.
P1 (high value):
  1. **Real copy/paste**: native Ctrl+V/right-click paste into the prompt box AND a "Paste to pane"
     (sends text to the pane, e.g. sign-in codes - the owner could not paste in the terminal);
     per-card "Copy screen"/select-to-copy.
  2. **Attention inbox**: all blocked/idle-finished agents in one list, sorted blocked first, with
     Windows notifications (Notification API) + optional sound; never auto-answers.
  3. **Task launcher**: list `tasks/*.md`, preview, send "Read CLAUDE.md and AGENTS.md, then do the
     task in <file>" to a chosen agent (or start a new agent for it); show its answer when done.
  4. **Broadcast / fan-out**: one prompt to several selected agents; compare answers side by side.
  5. **Usage and limits**: Claude accounts (2030/2040/1984) 5-hour/weekly bars, Codex limits, tokens
     per pane (from `herdr-agents usage`, `claude-limits.ps1`).
  6. **Health widget**: RAM, free disk (C:/G:/I:), number of panes, list of idle panes with a
     one-click "close finished panes" (owner rule: PC is short on RAM/disk).
  7. **Team presets**: one click = Claude + Codex, Claude + Codex + Gemini, research (headless), etc.
  8. **Sign-in helper**: per tool a card (claude accounts, codex, cline, grok, agy, cursor) showing
     signed-in state (`claude auth status`, `cursor-agent status`...), a button that opens a login
     pane; shows device codes; the owner finishes the sign-in - the app never handles secrets.
P2 (nice):
  9. **Changes view** (read-only): `git status`/`git diff --stat` of a pane's folder and worktrees,
     per-agent "what did it change".
  10. **Run history**: `%APPDATA%\herdr\agent-runs\*.md` task/answer viewer with search.
  11. **Docs hub**: HISTORY/LESSONS/AGENT-ROSTER/TEAM-PLAYBOOK viewer with search; one button
      "add lesson" (append-only form that writes through the backend to LESSONS.md - owner rule 0).
  12. **Check runner**: button runs `check.ps1` and shows OK/WARN/FAIL lines.
  13. **Pane layouts**: save/restore card layouts (grid columns, which panes are pinned/hidden).
  14. **Transcript export**: save a pane's screen as Markdown; "copy as prompt context".
  15. **Prompt library**: saved prompts/snippets with variables, per-agent history, Up-arrow recall.
  16. **Raw terminal mode**: focus a card and type straight into the pane (keys mapped to
      `pane send-keys`/`send-text`), with a visible "you are typing into <agent>" bar.
  17. **Themes** (dark default, light, high contrast), font size, density; all keyboard shortcuts
      listed in a `?` overlay; fully usable by keyboard.
  18. **Safe mode toggle**: "read-only" switch that disables every write button.
  19. **Activity timeline**: when each agent started/finished/blocked today.
  20. **Scheduled/queued prompts**: queue a prompt for an agent to send when it becomes idle.
Backend support for P1/P2 (new `/api/...` endpoints, file readers with path allow-lists inside the
toolkit and `%APPDATA%\herdr`, process runners for allowed scripts only) is BACKEND's job; it
implements them in the same order as the frontend requests them.

## Done means
Everything in `tasks/herdr-deck-app.md` section 6, plus: each P0/P1 feature works in the lab session
(evidence listed per feature in `app/FEATURES.md`), P2 items either work or are marked "not done".
