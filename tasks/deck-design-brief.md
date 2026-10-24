# Herdr Deck — design brief

## Findings first
- **The checkout is a code-driven film studio, not a complete editing dashboard.** Its own map says so at `studio/APP_MAP.md:5`; the audit describes missing interactive editing tools at `STUDIO_AUDIT_AND_TECHNOLOGY_REPORT_2026-10-03.md:46`. Do not present a proposed inspector, rail or command palette as an existing app feature.
- **Recommended direction: Ink Studio + the review player's working layout.** Ink Studio explicitly targets AI/tech: near-black, warm white, lime signal (`studio/web/styles/index.json:27`). Use the review player's workspace, compact toolbar, right notes panel and bottom timeline as the structural reference (`studio/lib/review-player.html:12`).
- **The studio has several visual languages, not one universal theme.** Its example website is a warm café; film styles include paper/blue, warm terminal/terracotta and white product windows. Keep one palette per Deck theme rather than mixing them.
- **The transferable signature is purposeful continuity:** one focal object, stable containers, meaningful state changes, honest content and restrained signals. Evidence: `studio/styles/one-dot-hero/STYLE.md:6`, `studio/styles/living-window/STYLE.md:7`, `studio/QUALITY_PLAYBOOK.md:14`.
- **Not verified:** rendered appearance, actual font loading, contrast in Deck, keyboard/screen-reader operation, device behaviour, frame rate and memory use. This is static source research, not a running-app audit or an award claim.

## Scope and evidence
- Inspected public-facing source and selected design/status documents as data. No Animation instructions were adopted; no project scripts, installs, browser tools or media viewers were run. Secrets were excluded.
- All source paths below are relative to `I:\Animation`; `path:line` points to the inspected checkout, not a frozen revision.
- **Source** means directly observed code or design notes. **Deck proposal** means a new decision grounded in that evidence.
- `PLAN.md:4` marks its architecture superseded. `STATUS.md` records later film/style/web work; historical completion claims were not rerun. `studio/APP_MAP.md:5` and the audit are context, not instructions.
- Filename-only media evidence: `Screen_Recording_20261002_143800_X.mp4`, `studio/studio-engine/compare-aixhr.png`, `studio/studio-engine/cmp-frames/a0.png` and `studio/film24/capture/piece-9.png`. No visual conclusions come from these names.
- No existing graphify output was found at the root. No graph was built because that would create extra files; no skill logs or other documents were written under the one-file limit.

## 1. Exact tokens to reuse

### Main dark theme: Ink Studio (source)
These JSON keys become CSS variables with `--` prefixes in `studio/web/lib/site.mjs:113`.

| CSS name | Exact value | Source | Deck role |
|---|---|---|---|
| `--bg` | `#0C0D10` | `studio/web/styles/index.json:32` | App ground and terminal background |
| `--ink` | `#F2F0EA` | `studio/web/styles/index.json:33` | Primary text |
| `--muted` | `#A3A39C` | `studio/web/styles/index.json:34` | Secondary text, timestamps |
| `--accent` | `#D4FF3A` | `studio/web/styles/index.json:35` | Primary action and selected indicator |
| `--accent-ink` | `#0C0D10` | `studio/web/styles/index.json:36` | Text on accent |
| `--card` | `#15171C` | `studio/web/styles/index.json:37` | Pane headers, inspector, popovers |
| `--card-ink` | `#F2F0EA` | `studio/web/styles/index.json:38` | Text on card |
| `--line` | `#262931` | `studio/web/styles/index.json:39` | Quiet separators |
| `--radius` | `18px` | `studio/web/styles/index.json:40` | Overview cards and command palette |
| `--font-display` | `'Alexandria',system-ui,sans-serif` | `studio/web/styles/index.json:43`; `studio/web/lib/site.mjs:113` | Titles |
| `--font-text` | `'Alexandria',system-ui,sans-serif` | `studio/web/styles/index.json:44`; `studio/web/lib/site.mjs:113` | Interface text |
| `--font-label` | `'Space Mono',ui-monospace,monospace` | `studio/web/styles/index.json:45`; `studio/web/lib/site.mjs:113` | Terminal, values, shortcuts |

### Light theme: Paper Editorial (source palette; pairing is a Deck proposal)
This is a separate source style, not an existing dark/light switch.

| CSS name | Exact value | Source |
|---|---|---|
| `--bg` | `#F4F1EA` | `studio/web/styles/index.json:10` |
| `--ink` | `#16140F` | `studio/web/styles/index.json:11` |
| `--muted` | `#5E5A52` | `studio/web/styles/index.json:12` |
| `--accent` | `#B8430F` | `studio/web/styles/index.json:13` |
| `--accent-ink` | `#FFFFFF` | `studio/web/styles/index.json:14` |
| `--card` | `#FFFFFF` | `studio/web/styles/index.json:15` |
| `--card-ink` | `#16140F` | `studio/web/styles/index.json:16` |
| `--line` | `#E2DDD2` | `studio/web/styles/index.json:17` |
| `--radius` | `22px` | `studio/web/styles/index.json:18` |

Deck proposal: retain Alexandria/Space Mono and 18px large-card radius across themes to prevent layout changes. The unmodified Paper style instead uses Readex Pro for display/text (`studio/web/styles/index.json:21`). Offer System/Dark/Light; persist the choice. Theme-switch behaviour is new.

### Review-player tokens (source; reference and semantic signals)
All nine are declared together at `studio/lib/review-player.html:7`.

| Name | Exact value | Use |
|---|---|---|
| `--bg` | `#111317` | Alternative review-tool ground |
| `--panel` | `#1A1D23` | Alternative panel surface |
| `--line` | `#2A2F38` | Alternative separator |
| `--text` | `#E6E8EB` | Alternative main text |
| `--dim` | `#9AA3AE` | Alternative secondary text |
| `--blue` | `#4C9AFF` | Informational signal |
| `--red` | `#FF5C5C` | Error/blocker signal |
| `--amber` | `#FFB547` | Waiting/warning signal |
| `--green` | `#3DDC97` | Successful/approved signal |

Deck proposal: import only the four signal names into Ink Studio; keep its own surface tokens. On light surfaces, put the signal beside dark text instead of using pale signal colours for small labels. Contrast remains not verified.

### Geometry, spacing and shadows: exact source literals
There is **no complete named spacing/radius/shadow scale** in the inspected UI. The following selector/value pairs are real; proposed scale names must not be attributed to the source.

| Source selector or property | Exact value | Source |
|---|---|---|
| Main columns / gap / padding | `1fr 340px` / `12px` / `12px` | `studio/lib/review-player.html:12` |
| Stage and video gaps | `8px` | `studio/lib/review-player.html:13` and `:14` |
| Toolbar gap | `6px` | `studio/lib/review-player.html:18` |
| Controls radius / padding | `5px` / `5px 9px` | `studio/lib/review-player.html:19` |
| Stage / timeline / aside radius | `6px` | `studio/lib/review-player.html:14`, `:22`, `:29` |
| Aside padding / gap | `10px` / `10px` | `studio/lib/review-player.html:29` |
| Note border / padding / radius | `3px` / `4px 8px` / `4px` | `studio/lib/review-player.html:32` |
| Keyboard badge radius / padding | `3px` / `0 4px` | `studio/lib/review-player.html:35` |
| Web focus outline / offset / radius | `3px solid var(--accent)` / `3px` / `6px` | `studio/web/motion/site.css:10` |
| Web button height / radius | `52px` minimum / `999px` | `studio/web/motion/site.css:35` |
| Web card padding | `clamp(22px, 2.4vw, 34px)` | `studio/web/motion/site.css:49` |
| Web card hover shadow | `0 24px 50px -30px rgba(0,0,0,.45)` | `studio/web/motion/site.css:50` |
| Web button resting shadow | `0 10px 30px -12px color-mix(in srgb, var(--accent) 70%, transparent)` | `studio/web/motion/site.css:35` |
| Web button hover shadow | `0 16px 40px -14px color-mix(in srgb, var(--accent) 80%, transparent)` | `studio/web/motion/site.css:36` |

Deck proposal: spacing steps 4/8/12/16/24/32px; nested gaps use 4/8, panel gaps 12, page inset 16. Keep 6px terminal shells, 5px controls, 18px overview cards, pill chips. Working panes stay flat; reserve one static shadow for floating overlays. These scales are new, not extracted tokens.

## 2. Typography and colour rules
- **Source:** Alexandria + Space Mono is a repeated two-family pairing (`studio/styles/one-dot-hero/STYLE.md:19`). Ink Studio has Arabic/Latin Alexandria faces with weight range 100–900; Space Mono has 400/700 (`studio/web/styles/index.json:84`). File declarations do not prove fonts load.
- **Source:** review body is 14px system UI; title 15px; timecode 600 15px monospace; inspector heading 13px uppercase with .06em tracking; keyboard badges 12px (`studio/lib/review-player.html:9`, `:11`, `:21`, `:30`, `:35`).
- **Source:** web body is 17–20px/1.7; headings use 700, −.01em tracking and 1.15 line height; labels 14px/.08em (`studio/web/motion/site.css:7`, `:14`, `:20`). These are website sizes, not a terminal scale.
- **Deck proposal:** five sizes: 12 metadata, 14 controls/terminal, 16 reading text, 20 section title, 28 page title. Terminal line height 1.5; UI 1.4; prose 1.6. Weights 400/600/700. Offer a terminal size control and comfortable density.
- Keep monospaced numerals for changing values so headers do not jump. Truncate long paths in headers, expose the full path on focus/copy and in the inspector; terminal lines scroll or wrap by user choice.
- Use neutral text for normal work, lime only for the primary action/selection, amber for waiting, red for failure, green for confirmed success. Running is a text label with a neutral activity mark; no green implication of completion.
- Dark and light are curated palettes, not inversion filters. Status always has a word and icon; semantic colour never carries meaning alone.
- Arabic: keep words intact, use logical spacing and isolate Latin paths/numbers. Avoid uppercase/tracking on Arabic labels. The studio preserves Arabic words (`studio/web/motion/motion.js:17`) and warns against a middle dot beside Arabic-Indic digits (`studio/styles/living-window/STYLE.md:19`).
- **Icons:** no shared named icon set was established. Web features use Unicode symbols (`studio/web/sites/example/site.json:16`); the review player uses text/glyph controls (`studio/lib/review-player.html:44`). Deck proposal: one locally stored SVG family, 20px box, 1.5px stroke, round caps, 16px compact variants, explicit accessible names. No runtime icon service.

## 3. Layout and arrangement
**Source:** the review tool puts a flexible stage beside a 340px notes panel, with 12px gutters; stage → toolbar → 74px timeline. Its aside scrolls separately (`studio/lib/review-player.html:12`, `:22`, `:29`, `:40`). It has no rail, terminal grid or command palette.

**Deck proposal — desktop at 1280px and above:**
- A 56px top bar: Deck name, workspace switcher, search/commands, New agent. Only New agent is filled with accent.
- A 64px left rail: Workspaces, Agents, Activity, Settings; selected item gets a short lime marker and clear label in its tooltip. Expand to 216px by choice to show workspace names and counts.
- Main centre: 12px gutters; default two equal pane columns when each can remain at least 360px wide; one column below that limit. Pane minimum height 280px. More panes scroll, not squeeze terminal text.
- Optional 340px right inspector: selected agent's task, model, folder, context, latest event, permissions requiring attention and actions. Keep section order fixed. Hide/show preserves focus and selected pane.
- Each pane has a 40px header: agent name left, state word and context ring right, then overflow. Model/path are secondary. Output fills remaining space; bottom input area is visually separate.
- Bottom activity drawer: 120px initial height, user-resizable, with one row per agent and timestamped intervals; collapsed summary 32px. It filters activity, not terminal history itself.
- A 28px status bar: connection state + last update left; running/waiting/error counts right. Unknown/stale data is written explicitly. Do not fabricate cost, usage or progress.
- Command palette: centred near the upper third, 640px maximum width, 16px viewport margins, one search field, grouped actions and a fixed shortcut hint row. Results scroll within 60vh.

**Timeline mapping:** source shots occupy a 20px band; copy occupies 6px at top 22px; cues are 2px × 10px; notes span the track; playhead is 2px white (`studio/lib/review-player.html:24`). Deck maps intervals to working/waiting spans, cues to tool/task events, notes to errors/reviews, playhead to selected historical time. Do not use a moving live playhead when paused. Add a list view with identical information and keyboard-selectable events.

**Responsive source:** web nav hides at 720px; hero/steps collapse at 860px; cards auto-fit from 280px (`studio/web/motion/site.css:32`, `:47`, `:48`, `:61`). The review player has no responsive media rules in its style block; small-screen usability is not verified.

**Deck proposal — responsive:** 860–1279px: rail stays compact; inspector becomes a drawer so panes retain width. Below 860px: one pane, agent switcher, inspector as full-height sheet, activity as list. Below 720px: explicit menu button replaces rail; top bar retains workspace and New agent. Test 390/820/1440px, 200% zoom and keyboard-only use; avoid whole-page horizontal scrolling. Terminal content may scroll horizontally inside its own pane.

**Keyboard proposal:** Ctrl/Cmd+K palette, Escape closes overlay and restores focus; palette arrows/Enter navigate/choose. Provide visible commands for next/previous pane and activity event. No single-letter global shortcuts while terminal/input/select/contenteditable has focus. The source review tool skips only INPUT/TEXTAREA (`studio/lib/review-player.html:108`); Deck must cover terminal capture and all editable controls too. Do not steal shell shortcuts.

## 4. Motion specification

| Source motion | Exact timing/behaviour | Evidence |
|---|---|---|
| Web button | transform .35s `cubic-bezier(.2,.8,.2,1)`; shadow .35s | `studio/web/motion/site.css:35` |
| Web card | transform .5s same curve; hover −4px; shadow .5s | `studio/web/motion/site.css:49` |
| Step swap | transform .6s same curve; opacity .4s | `studio/web/motion/site.css:57` |
| Reveal | opacity/transform .9s same curve, filter .9s | `studio/web/motion/site.css:98` |
| Word reveal | .7s with 70ms stagger; transform uses same curve | `studio/web/motion/site.css:100` |
| Product-window content | .24s in, .2s out; spring damping .72–.8 | `studio/styles/living-window/STYLE.md:22` |
| Reduced motion | animations/transitions off; content visible, transform/filter removed | `studio/web/motion/site.css:106` |

**Deck proposal:** retain `cubic-bezier(.2,.8,.2,1)` as the sole entrance curve. Use 120ms for hover/press opacity, 160ms for menu/toast entrance, 200ms out and 240ms in for inspector/palette content. Move overlays at most 8px; a new pane may fade once in 240ms. Apply the source's 350ms duration only to a deliberate layout-preset change, using a temporary transform preview; terminal text settles immediately afterwards.

- Animate opacity and transforms only; resize live panes directly under the pointer. Do not tween grid widths, blur, shadows, context-ring strokes or thousands of output nodes. Use static borders/shadows and bounded overlays.
- Keep source film springs, camera moves, grain, circular theme wipes and typing sound as references, not dashboard defaults. They serve films; they would interfere with reading terminal output.
- Update the context arc discretely when data changes; never spin it. Completion can show a single 160ms check fade; no perpetual success pulse.
- Honour reduced motion from first paint and when the preference changes: immediate state changes, no stagger or movement. Stop optional animation when the page is hidden. No animated skeleton shimmer.
- Performance target: no sustained decorative animation, event-driven updates and bounded output/history. A 60fps interaction target is **not verified**; measure on the owner's PC with multiple active panes before claiming it.

## 5. Component build specification and states
Everything in this section is a **Deck proposal** unless a source reference is given.

| Component | Shape and content | Required states and behaviour |
|---|---|---|
| Agent pane/card | 6px pane radius; 1px line; header 40px; 12px inner spacing; independent output scroll. Overview summary uses 18px radius. | Idle, connecting, working, waiting for owner, completed, failed, disconnected, stale. Selection adds an accent edge; focus adds separate 3px outline. A failure keeps output and offers a labelled recovery action. |
| State chip | 24px high pill, 8px side padding; 6px mark + state word; normal surface with coloured mark. | Working neutral, waiting amber, completed green, failed red, unknown muted. No colour-only states; filters expose pressed state and have larger hit areas. |
| Context ring | 28px SVG with 3px track/arc, adjacent 12px numeric percentage; accessible name includes used/total and data age. | Known percentage, warning at 75%, critical at 90% (new thresholds), unavailable as dash, stale as explicit label. Clamp arc to 0–100%; never turn missing data into zero. Numbers/labels remain readable without the ring. |
| Primary/secondary buttons | 36px visual height, 12px side padding, 5px radius; 44px touch hit area. Accent fill for primary, card + border for secondary, plain text/icon for quiet. | Hover, focus, pressed, disabled, busy, error. Busy retains width/label and adds Working…; prevent duplicate activation. Disabled includes a visible explanation where needed. Destructive action is a separate labelled menu item. |
| Search/input/composer | 40px field, 8px radius, 12px inset; visible label; helper/error below. Composer grows to 5 lines before scrolling. | Empty, focus, populated, invalid, disabled, submitting. Preserve typed text after errors. Enter/newline behaviour is explained beside Send. No placeholder-only labels. |
| Inspector section | Fixed heading + value rows; 16px section gap; long task text wraps; path has copy control. | No agent selected, loading details, current data, stale data, retrieval error. Keep headings stable during refresh; never replace last known values with invented defaults. |
| Toolbar/segmented layout | Small labelled groups separated by 1px lines; 8px group gaps; icons plus text where space permits. | Default, hovered, focused, selected, unavailable. Use pressed/selected semantics; title alone is insufficient for the accessible label. Source compact wrapping bar: `studio/lib/review-player.html:18`. |
| Command palette | 18px radius card; 48px search row; 44px result rows with icon, verb, target and shortcut. | Closed/open, searching, results, no match, unavailable action, action failure. Keyboard highlight differs from hover; trap focus within dialog, restore it on close; unavailable items explain why. |
| Menus | 240px default width, 8px radius, 6px padding; 36px visual rows with generous touch hit area. | Open, focused item, selected/checkmarked, disabled, submenu. Arrow-key navigation, Escape, outside click; reposition inside viewport. Keep destructive items last, separated. |
| Toasts | Max 360px width; card surface; 3px semantic edge; icon, short message, optional Undo and Close. Stack at most three. | Success/info dismiss after 5s, paused on hover/focus; errors persist until dismissed. Never make toast the sole record of failure. Polite announcement for ordinary notices; urgent only for a real blocker. |
| Empty states | Small functional symbol, 20px title, one short explanatory sentence, one clear action; no giant decorative image. | First use: Add your first agent. Filter empty: No agents match / Clear filters. No selection: Select an agent. Empty activity: Activity appears when agents work. |
| Loading/error/connection | Keep panel geometry; static muted placeholders and Loading… label; connection strip in status bar. | Initial loading, background refresh, retryable failure, offline, reconnecting, stale. Use busy semantics; preserve last known output. Announce transition once, not every log line. |
| Activity event | 28px visual row; timestamp, agent, event name, result; selected event links to relevant output. | Unread, selected, waiting, failed, acknowledged. Provide ≥44px touch selection or comfortable list mode. Paused history shows Resume live; new arrivals accumulate as a count without stealing position. |

Source gaps: no implemented command palette, context ring, toast system, common loading/empty-state library or full disabled-state system was established. The design template lists these states as requirements (`studio/web/templates/DESIGN.md:20`), which is not implementation proof.

## 6. Usability principles and ten upgrades

**Principles found in the source:**
- Stable working area with tools near the thing being reviewed; details alongside, not over the stage (`studio/lib/review-player.html:40`).
- Severity is visible in both notes and timeline; clicking a note seeks its evidence (`studio/lib/review-player.html:145`).
- Exact-result truth matters: approval becomes void when the export changes (`studio/lib/review-player.html:154`). Translate this into task/run identity, timestamp and data freshness.
- One brand signal and mostly neutral surfaces keep attention clear (`studio/styles/one-dot-hero/STYLE.md:15`).
- Visible focus, skip link, native disclosure controls, reduced motion and logical direction-aware layout exist in the web base (`studio/web/motion/site.css:3`, `:10`, `:82`, `:106`).
- Source honesty is uneven: review startup explains offline storage (`studio/lib/review-player.html:131`), but note-save failures are swallowed at `:136`. Carry forward the visible state, not the silent-failure pattern.

**Ten concrete Deck upgrades — new design proposals:**
1. **A three-second attention map:** top summary shows Needs you, Working, Finished. Needs you jumps to the blocking pane without rearranging all panes. Test whether the user can find the blocker immediately.
2. **One shared active mark:** use the same small lime bar in rail selection, focused pane header and selected timeline event. It connects navigation to evidence; it stays still while output streams.
3. **Density that respects reading:** Compact/Comfortable presets change padding and row heights, while terminal font size stays independently adjustable. Save preferences per device.
4. **Evidence beside status:** selecting Failed reveals the exact event, relevant output and next action in the inspector. Keep run identity visible so old success cannot masquerade as the current result.
5. **A useful context ring:** percentage plus used/limit and freshness; thresholds explain consequences in plain words. Missing telemetry says Unavailable. This turns decoration into an early decision aid.
6. **Quiet live activity:** when reading older output, pause auto-follow and show 12 new lines / Jump to latest. Do not pull the viewport or keyboard focus away; resume only on request.
7. **Reversible spatial control:** named layouts such as Focus, Pair and Review; pane moves retain scroll/input; offer Undo after layout changes. Keep pane identities stable to protect the user's mental map.
8. **Accessible precision:** every icon has a name, controls have distinct focus, keyboard alternatives cover resizing/reordering, status has text, and forced-colours mode remains usable. Show a discoverable shortcut guide.
9. **Small, meaningful delight:** a successful task replaces its working mark with one check fade; Copy becomes Copied without shifting width. No celebration layer or sound by default; reduced motion is instant.
10. **Polish with a measured budget:** local fonts/icons, minimal initial JavaScript, bounded terminal DOM, one batched paint update and no background decorative loop. Profile many active panes on this PC; reduce effects before reducing legibility.

Acceptance before calling the result award-quality: consistent optical alignment, zero jumping headers, all states designed, real task/output data, light/dark contrast review, keyboard and screen-reader trials, responsive/zoom checks, and measured interaction/memory behaviour. All remain **not verified** in this research task.

## 7. Source files worth reading

| Path | Why |
|---|---|
| `studio/web/styles/index.json:27` | Best-fit dark palette, exact colour/radius/font values; Paper alternative at line 5. |
| `studio/web/motion/site.css:1` | Actual web geometry, typography, controls, shadows, breakpoints and motion/reduced-motion rules. |
| `studio/web/lib/site.mjs:110` | How style keys become CSS variables/fonts; semantic markup, skip link and icons. |
| `studio/lib/review-player.html:7` | Closest functional tool UI: compact controls, split stage/aside, timeline, notes, severity and shortcuts. |
| `studio/styles/terminal-cursor/STYLE.md:1` | Warm terminal alternative; Space Mono code, focused cursor, result-only signal colours. |
| `studio/styles/living-window/STYLE.md:1` | One persistent window, purposeful content swaps and mode changes; film guidance, not dashboard code. |
| `studio/styles/one-dot-hero/STYLE.md:1` | Single visual motif, neutral ground and consistent brand signal. |
| `studio/styles/index.json:74` | Exact terracotta terminal palette if a separate warm theme is later requested. |
| `studio/web/motion/motion.js:1` | Dependency-free interaction, observer-based reveals, Arabic word handling; inspect perpetual ring loop before reusing anything. |
| `studio/web/sites/example/DESIGN.md:15` | Records muted-contrast, tap-target and overflow fixes; clearly identifies café-only choices and unverified devices. |
| `studio/web/templates/DESIGN.md:14` | Intended design checklist: state coverage, two font families, contrast, breakpoints and critique; template, not implemented system. |
| `studio/QUALITY_PLAYBOOK.md:23` | Visual concept and continuity principles; separate film theatrics from functional dashboard use. |
| `studio/APP_MAP.md:5` | Establishes actual product scope and explains why no full editing dashboard was found. |
| `PLAN.md:4`; `STATUS.md:1`; `STUDIO_AUDIT_AND_TECHNOLOGY_REPORT_2026-10-03.md:46` | Historical plan, current handoff context and capability limits; skim only, do not treat as a build queue. |

## Doubts / not verified
- No screenshots or recordings were viewed; notes and code cannot establish final rendered quality.
- No live studio theme toggle, shared icon library, rail or command palette was established in the inspected first-party UI. Film depictions are not proof of app features. This was a targeted inspection, not a complete vendor-code audit.
- Source palettes are copied exactly; Deck contrast, font fit, terminal ANSI colour mapping and semantic signal contrast need measurement after implementation.
- CSS transform/opacity choices lower likely rendering cost; they do not guarantee 60fps or acceptable memory on this PC. No benchmark was run.
- This deliverable changes no behaviour. No project test suite was needed or executed; verification is file read-back, reference checks and the under-400-line limit.
