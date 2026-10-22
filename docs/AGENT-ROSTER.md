# Agent roster: who can work on this PC, and who gets which task

Scanned 2026-10-22 (re-run: section 6). Claude (Opus 5.5) leads and uses this table to
give each task to the right agent at the right thinking level and cost. Prices are from
third-party pages (approximate), so they guide choices; they are not invoices.

## 1. What is installed and whether it answers

| Agent (profile) | CLI / version | Models available here | Thinking levels | Works now? |
|---|---|---|---|---|
| **Claude Code** (`claude`) | 2.1.308 | Opus 5.5 (default, effort high), Sonnet 5.5, Haiku 5.5 | low ... max | **Yes** (the lead) |
| **Codex** (`codex`) | codex-cli 0.163.0 | GPT-6.1-Sol, 475k context; business plan | low, medium (default), high, xhigh, ultra, max (`-Effort`) | **Yes** |
| **Antigravity** (`agy`, `gemini`, `gemini-flash`) | agy 1.4.0 (Google, replaces Gemini CLI) | Gemini 4 Argon (low/med/high), Gemini 3.8 Flash (low/med/high), Claude Opus 5.5 / Sonnet 5.5, GPT-OSS 120B | in the model name, or `--effort` | **Yes** (`AGY-OK` in 19 s) |
| **Cline** (`cline`, `cline-flash`, `deepseek`, `kimi`) | 3.0.73 | **Cline Pass** ($9.99/month, owner signed in 2026-10-22): `cline-pass/glm-5.3`, `glm-5.3-flash`, `deepseek-v4-pro`, `kimi-k3` (+ 8 more, docs/CLINE-PASS.md) | `--thinking none..xhigh` | **Yes** - all four answered (~60 s start-up each); DeepSeek works through Cline Pass despite the network block on its own API |
| **OpenCode** (`opencode`) | 1.18.11 | Zen/Go catalogue: Kimi K2.6, GLM-5.1, DeepSeek V4 Pro/Flash, Gemini 3.x, GPT-5.x, MiMo ... | per model | **No**: OpenAI sign-in expired (401); Zen models answer with an error page |
| **Grok Build** (`grok`) | 0.2.106 | Grok 4.5 | - | **No**: not signed in |
| Not agents | Windsurf (an editor), LM Studio (no local models downloaded), GitHub CLI (not signed in) | | | - |

**To unlock more agents (the owner types these, in a pane):**
`cline auth` (re-sign in to Cline) · `opencode auth login` (OpenAI / Zen) · `grok login`
(SuperGrok) · DeepSeek through Cline also needs the network to allow its API (the HTML page
looks like the company proxy - check from home or ask IT).

## 2. Strengths, weaknesses, cost

| Agent | Best at | Weak at / watch out | Cost (approx.) |
|---|---|---|---|
| **Claude Opus 5.5** (lead) | architecture, plans, specs, risky/critical changes, integration, review, security, docs, Windows/PowerShell details | most expensive per token; context fills (watch `ctx %`) | highest (this session ~$10); keep it for thinking, not bulk typing |
| **Codex GPT-6.1-Sol** | implementing a precise spec, refactors, tests, self-checking its work, long focused coding | brings its own environment rules (LESSONS #31); first prompt can stall (#21); asks to trust each new repo (#23); sandbox writes only inside its folder | $2 in / $10 out per 1M; reasoning tokens bill as output - effort drives cost |
| **Gemini 4 Argon** (agy) | a second opinion from another model family, reading large codebases/docs, research summaries, UI/web | little public benchmark data; Antigravity's default mode `accept-edits` changes files WITHOUT asking; our profiles start with `--dangerously-skip-permissions` (owner decision 2026-10-22; `--mode auto` still asked before commands) - give it a worktree (`-Worktree`) when it edits code; `--mode plan` = read-only | Google account quota |
| **Gemini 3.8 Flash** (agy) | fast, cheap: summaries, search across files, simple scripts, drafts | shallow on hard logic | cheapest of the working agents |
| **DeepSeek V4 Pro** (`deepseek`, through Cline Pass) | cheap bulk coding and tests; strong LiveCodeBench | weaker on long agent runs than Kimi | ~$0.44 in / $0.87 out per 1M (cheapest) |
| **Kimi K2.6** (OpenCode, when signed in) | agentic multi-step coding (SWE-Bench Pro ~58.6%) | needs OpenCode Zen | ~$0.95 / $4 |
| **GLM-5.1** (OpenCode, when signed in) | coding close to Kimi | prices vary by host | ~$1.4 / $4.4 |
| **Grok Build 4.5** (when signed in) | `--best-of-n` parallel tries, `--check` self-verification, up to 8 sub-agents | subscription; benchmarks unclear | SuperGrok $30/month |

## 3. Routing rules (who gets what)

1. **Critical, risky, cross-cutting, security, final integration and review** -> Claude itself.
2. **A well-specified implementation** (spec from `tasks/TEMPLATE.md`) -> Codex, effort
   `medium`; `high` for tricky logic or concurrency; `low` for mechanical edits; `max`
   only for a hard bug after medium failed.
3. **Second opinion / independent review** of a design or diff -> a different model family
   than the author (Codex wrote it -> Gemini Argon reviews; Claude decides).
4. **Read-and-summarise, search, "where is X used", docs drafts** -> Gemini 3.8 Flash (cheap).
5. **Large volume of simple code or tests** -> Cline Pass: `cline-flash` (GLM-5.3 Flash) or
   `deepseek` (DeepSeek V4 Pro); daily coding `cline` (GLM-5.3); hard multi-file problems
   `kimi` (Kimi K3, uses the Cline Pass quota fastest). Details: docs/CLINE-PASS.md.
6. **Exploration with several approaches** -> `fanout` the same question to Codex and
   Gemini Argon (and Grok `--best-of-n` once signed in), compare.
7. Anything touching secrets, sign-ins, approvals, production systems -> the owner.

## 4. Thinking level vs cost

| Level | Use for | Cost effect |
|---|---|---|
| low | renames, formatting, moving code, simple scripts | cheapest, fastest |
| medium (default) | normal features and fixes from a clear spec | balanced |
| high / xhigh | tricky logic, debugging, unfamiliar code | 2-4x the reasoning tokens |
| max / ultra | one hard problem that failed at high | most expensive; use rarely |

Reasoning tokens bill as output tokens. The HUD shows each agent's context and output
tokens live, and Claude's dollar cost, so you can spot an expensive agent early.

## 5. Edge cases the lead must handle

- **Not signed in / expired** (NOT READY or an auth error): stop, tell the owner the exact
  command (section 1); never type keys.
- **Network-blocked provider** (HTML page instead of an answer): mark it unavailable, reroute.
- **BLOCKED on approval**: show the owner; approve only what the owner allowed in this request.
- **Trust prompt** (Codex per repository, Antigravity per folder): the owner answers once.
- **Stall / UNCONFIRMED** (LESSONS #21): read the screen before resending.
- **Two agents, one file**: never; split by file or use `-Worktree` per agent.
- **Rate limits**: about 4-5 heavy agents per subscription at once; more go in pages
  (`team` overflow) but start them in waves.
- **Context full** (`ctx` > 80%): ask the agent to summarise, then start a fresh one with the summary.
- **Agent edits outside its scope**: `diff` after every task; revert and re-instruct.
- **Agent brings its own tools/rules** (LESSONS #31): name the exact API in the spec.
- **Review is the bottleneck**: 3-5 parallel agents is the practical maximum for one lead.

## 6. Re-run the scan

```
herdr-agents tools                                   # profiles + installed
claude --version; codex --version; cline --version; opencode --version; grok --version; agy --version
agy models                                           # Google models available
opencode models                                      # OpenCode catalogue
agy -p "Reply with exactly: OK" --model gemini-3.8-flash-low   # cheap health check
```
Update this file and `agents.json` when something changes.

Sources: [GPT-6.1 Sol pricing](https://emergent.sh/learn/gpt-6-1-sol-pricing),
[GPT-6.1 Sol for coding](https://www.layer3labs.io/guides/gpt-6-1-sol-for-coding),
[DeepSeek V4 Pro vs Kimi K2.6](https://www.gradually.ai/en/llm-comparison/deepseek-v4-pro-vs-kimi-k2.6/),
[Kimi vs DeepSeek vs GLM](https://codersera.com/blog/kimi-k2-6-vs-deepseek-v4-vs-glm-5-1-2026/),
[Antigravity CLI vs Grok Build CLI](https://terminaltrove.com/compare/ai-coding-agents/antigravity-cli-vs-grok-build-cli/),
[Gemini CLI -> Antigravity CLI](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/),
[Grok Build](https://en.wikipedia.org/wiki/Grok_Build).
