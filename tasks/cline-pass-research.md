# Task: research Cline Pass and its models for daily coding

## 1. Role and goal
You are a senior AI-tools analyst working for the lead (Claude). The owner just bought a
**Cline Pass** subscription and uses the Cline CLI (v3.0.73, provider id `cline-pass`) on
Windows. Goal: a complete, sourced, decision-ready report so the lead can recommend which
Cline Pass models to use daily for coding, and how to control Cline for the best results.
Why: the model saved in Cline (`stealth/ox-alpha`) was retired ("was ZAI's GLM-5.3 Flash"),
so the owner needs a correct model choice now.

## 2. Context
- Use web search and official pages first: cline.bot (pricing, Cline Pass, docs, blog,
  changelog), docs.cline.bot, github.com/cline/cline (README, releases, CHANGELOG),
  openrouter.ai model pages for each model. Secondary sources only to fill gaps, marked as such.
- Cline CLI flags on this PC (from `cline --help`): `-P/--provider`, `-m/--model`,
  `-p/--plan`, `--thinking none|low|medium|high|xhigh`, `--compaction agentic|basic|off`,
  `--worktree`, `--auto-approve`, `-y/--yolo`, `--retries`, `-t/--timeout`, `--json`,
  `--tui`; commands `auth`, `config`, `mcp`, `skill`, `hook`, `schedule`, `hub`, `doctor`.

## 3. Exact scope
- Write ONE new file: `D:\WORK\Software Development\GitHub\HERDR\docs\CLINE-PASS.md`.
- Do NOT change any other file, run git, install anything, run `cline auth`, or touch
  `%USERPROFILE%\.cline` (never read `secrets.json`).
- You may run read-only commands: `cline --help`, `cline <command> --help`, `cline version`.

## 4. What the report must contain (headings in this order)
1. **What Cline Pass is**: price(s), what is included, usage limits / quotas / fair use,
   rate limits, what happens at the limit, how it differs from paying per token.
2. **Every model available in Cline Pass today**, as a table: exact model id to pass with
   `-m` (verify the id format the CLI expects), maker, context window, thinking/reasoning
   support, speed, coding strengths, weaknesses, any limit or extra cost, retired/preview
   status. Mark anything you could not verify.
3. **Coding benchmarks** for those models (SWE-bench Verified/Pro, Terminal-Bench,
   LiveCodeBench, Aider polyglot ...) with source and date for each number.
4. **Recommendation for daily coding**: a default model, a fast/cheap model, a "hard
   problem" model, a plan-mode model - each with the exact CLI command, e.g.
   `cline -P cline-pass -m <id> --thinking medium "<task>"`, and why.
5. **How to control Cline for the best results**: plan vs act mode, thinking levels, context
   compaction, `.clinerules` / rules files, skills, MCP, worktrees, checkpoints, auto-approve
   and its risks, retries/timeouts, how to keep costs/limits under control, how to switch
   the saved default model (exact command), known problems and fixes on Windows.
6. **Sources**: every URL used, with the date read.

## 5. Definition of done
- [ ] `docs\CLINE-PASS.md` exists with all 6 sections; every model id and price has a source.
- [ ] Nothing else changed.

## 6. Answer format
DONE | PARTIAL
File: <path>
Default model recommended: <id> (one line why)
Unverified items: <list or "none">
