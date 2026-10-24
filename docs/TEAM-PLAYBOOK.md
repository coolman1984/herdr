# Team playbook: solving hard problems with the agent team

How the lead (Claude) uses Codex, DeepSeek and the other agents on big or hard problems.
Built from web research (sources at the end) and from what this team actually did and measured
(HISTORY 25-28, docs/AGENT-ROSTER.md section 7). Evidence levels are marked: **[study]** = a
paper with measurements, **[practice]** = practitioner consensus, **[ours]** = measured here.

## 1. The team and who does what

| Role | Agent | Why |
|---|---|---|
| Lead: plan, specs, design, integration, final review, commit | Claude (Opus) | owns the result; keeps its tokens for thinking, not typing |
| Deep engineer: audits, tricky logic, cross-cutting fixes | Codex GPT-6.1-Sol high (`codex -Effort high`) | 30 real findings, tests that fail on old code **[ours]** |
| Workhorse + second reviewer: bulk code, tests, docs, git | DeepSeek V4.1 Flash (`ds-flash`, high) | 41/41, proves claims by running code, ~4x less fresh input **[ours]** |
| On trial | Codex GPT-6-Astra medium, Gemini 4 Argon High | promote only after `herdr-trial.ps1` / the trial kit |
| Not as reviewer | GLM-5.3 Flash | correct code, but confident wrong explanations **[ours]** |

Rules: 3-5 agents at once **[practice]**; one file = one owner; each agent on a worktree
branch when it edits code **[practice]**; Codex gets one task per fresh conversation
(LESSONS #45); escalate effort only after a failure (high -> xhigh), never by default **[ours]**.

### 1b. Control cards - how to steer each agent

Durable rules every agent reads at startup are in `AGENTS.md` (Codex, Cline and Antigravity
all load it); the spec only adds the task. Keep AGENTS.md short - long rule files cost tokens
without raising success **[practice]**.

| Agent | Its power | Steer it with | Warning signs | Correct with |
|---|---|---|---|---|
| Codex Sol / Astra | long autonomous implement-test loops; self-checking | medium by default, high/xhigh only for the hardest **[OpenAI guide]**; "deliver working code, not a plan"; exact files; findings-first review format | conversation > ~150k tokens (gateway 403, #45); re-editing the same file; credits (monthly limit - check `/status`) | `ask -Fresh` per task; `esc` + one precise correction; ask it for a retrospective and fold the rule into AGENTS.md **[OpenAI guide]** |
| DeepSeek V4.1 Flash (`ds-flash`) | cheap volume; runs code to prove claims; honest reports | high (= xhigh quality here **[ours]**); plan -> tools -> verify -> answer; answer file in `tasks/` | Herdr says `idle` while it works (#41); runs commands in parallel and misreads a race | wait for its answer FILE, not its state; re-run the check serially |
| Gemini Argon (agy) | different family for plans and design critique; big reads | separate explore / plan / execute **[Antigravity docs]**; `--mode plan` for read-only; `/fork` for risky ideas | confident unproven explanations (same wrong claim as GLM **[ours]**); edits without asking in accept-edits | require proof for every claim; worktree when it edits |
| GLM-5.3 Flash (`cline-flash`) | correct code for its size | high, never xhigh (it loops **[ours]**) | wrong causes stated as facts | not as reviewer; proof required |
| Claude Opus 5.5 (other account) | complex, open-ended, cross-module work; sustained judgment **[Anthropic]** | `medium` default, raise on measured failure; "track requirements, continue through blockers, stop only for decisions" **[Anthropic Opus guide]** | may end a turn with an update while work remains | answer file + explicit completion check; same family as the lead - not the only reviewer |
| Claude Sonnet 5.5 (other account) | scoped everyday coding, bugs, design; ~30% faster output than Sonnet 5 **[Anthropic]** | `medium` for specified tasks, `high` for harder/longer; `max` caused timeouts/scope violations in Anthropic's runs **[Anthropic]** | adds unrequested files or starts extra reviews at higher effort | "avoid unrelated additions; no extra review loops" in the spec |
| Claude Haiku 5.5 (other account) | narrow, repetitive work; fastest; cheapest ($0.10/$0.50 per M) **[Anthropic]** | `medium` default, `high` for strict instruction following; `xhigh` only if it beats Sonnet on evals **[Anthropic Haiku guide]** | at `low`/`medium` reports done without running a check; at `xhigh` may end with NO visible answer (answer only in its thinking) **[Anthropic]**; weaker on complex terminal coding (39% vs Sonnet 71%) | spec demands a real check + an answer FILE; read the file, not the screen |
| Claude helpers in general | - | they read `AGENTS.md` only via the `@AGENTS.md` import in CLAUDE.md (a CLAUDE.md suppresses AGENTS.md) **[Claude Code docs]**; run on accounts 1984/2040, never the lead's | startup dialogs on a new account (LESSONS #53) | owner answers dialogs; read the screen before the first prompt |

### 1c. Is a team worth it here? (decide before spawning)

A multi-agent run costs ~15x the tokens of a chat, and pays off only for work that splits into
independent parts; "most coding work" has few parallel steps **[Anthropic research system]**.

| Task | Team size **[Anthropic's scaling rules, adapted]** |
|---|---|
| Small fix, one file, clear cause | lead alone, or ONE `ds-flash` + lead review |
| Feature over 2-4 files, or a comparison | 2-3 agents, one file set each |
| Audit, big refactor, unknown root cause | 3-5 agents, divided responsibilities, verification ladder |
| Tight dependencies, shared context needed | NOT a team: one agent sequentially, lead checks each step |

Budget: Codex has a monthly credit limit (check `/status`); route volume to `ds-flash`
(Cline Pass) and keep Codex for the parts that need it.

## 2. Pick the strategy from the shape of the problem

| Problem shape | Strategy (section) |
|---|---|
| Clear goal, many files, parts known | 3.1 Orchestrator-workers |
| Bug with unknown cause | 3.2 Parallel hypotheses |
| One hard function / algorithm / parser, correctness critical | 3.3 N-version + test-based selection |
| Architecture or API decision | 3.4 Plan + adversarial review |
| Large mechanical change (rename, migration, many files) | 3.5 Characterize, shard, verify |
| Vague request | 3.6 Spec first (requirements -> design -> tasks) |
| "Is this code correct?" (any of the above) | 4 Verification ladder |

## 3. Strategies

### 3.1 Orchestrator-workers **[practice]** (Anthropic's pattern; what rounds 1-2 used)
1. Lead splits the work by FILE, writes one spec per agent from `tasks/TEMPLATE.md`
   (role, why, exact scope, design with signatures, traps, definition of done, answer format).
2. Agents run in parallel beside the lead (`team` or `start ... -Worktree <branch>`).
3. Lead monitors (`read <name> -Visible`), corrects drift at once, never lets two agents share a file.
4. Integration and HISTORY/LESSONS stay with the lead.

### 3.2 Parallel hypotheses (unknown root cause)
1. Lead writes the symptom, a reproduction command and the evidence rules.
2. `fanout` a READ-ONLY investigation to 2-3 agents of different families, each told to
   pursue a DIFFERENT hypothesis (assigned by the lead) and to return: hypothesis, proof
   (command + output), confidence, the smallest fix.
3. Accept only a hypothesis with a reproduction that fails before and passes after the fix.
4. One agent implements; the others review. Cap: 2 rounds, then the lead investigates directly.

### 3.3 N-version + test-based selection **[study: CodeT, AlphaCodium]**
For one critical function where a wrong answer is costly.
1. Lead fixes the exact signature and the spec (inputs, outputs, errors, edge cases).
2. Test author (agent A) writes tests FROM THE SPEC ONLY - it never sees an implementation.
   First a list of edge cases, then the tests (AlphaCodium: reason about the problem and
   edge cases before code; iterate against public + generated tests - 19% -> 44% pass@5).
3. Two or three implementers (different models, e.g. Codex + ds-flash) build it independently.
4. Run every candidate against every test, then DIFFERENTIAL testing: random/generated inputs to
   all candidates, compare outputs **[study]**. Disagreement = a bug in one candidate or a gap
   in the spec -> the lead decides, the spec is updated.
5. Pick the candidate that passes the tests and agrees with the others (CodeT's "dual execution
   agreement": +18.8 points pass@1 on HumanEval **[study]**).

### 3.4 Plan + adversarial review (architecture, API, data formats)
1. One agent (or the lead) writes the plan; a reviewer of a DIFFERENT model family reviews it
   "as a staff engineer": risks, missing cases, simpler alternative, what breaks later.
   Different families make different mistakes **[practice; studies indirect]** - same-family
   reviewers can agree on the same error ("diversity collapse").
2. The lead decides and writes the decision + reason into the spec. No implementation before
   the plan is approved **[practice]**: an architecture error is cheapest before code exists.

### 3.5 Characterize, shard, verify (large mechanical changes)
1. First characterization tests (golden master): record today's behaviour on many inputs.
2. Shard by file into independent chunks; cheap workers (`ds-flash`) take one shard each.
3. Every shard must keep the characterization tests green; the lead samples diffs.

### 3.6 Spec first **[practice: GitHub Spec Kit, Kiro]**
Requirements -> design -> tasks, each a checkpoint: requirements catch scope errors, design
catches architecture errors, tasks catch sequencing errors. The spec is the source of truth;
code is checked against it. In this repo: the spec file in `tasks/`, from `TEMPLATE.md`.

## 4. The verification ladder (every change climbs it)

Generating code is cheap; verification is the bottleneck **[practice]**. Gates, in order:
1. Parses / lints; `check.ps1` passes.
2. The author's own tests pass.
3. **Tests fail on the old code** (run the suite against `git show HEAD:<file>` copies) - a test
   that passes on old code proves nothing **[ours: every round]**.
4. **Tests check behaviour, not text** (HISTORY 26: a test failed only because a line was missing).
5. **Mutation check**: break the new code on purpose (flip a comparison, drop a branch) - a good
   suite fails. Surviving mutants go back to the test author as feedback **[study: Meta ACH,
   MuTAP]**. The lead did this by hand for the trial kit.
6. **Hidden acceptance tests** from the lead that the implementer never sees; validated first
   against a reference solution (must pass) and a mutant (must fail) **[ours]**.
7. **Property tests** for invariants (sorting, round-trips, totals) when inputs are wide **[study]**.
8. **Cross-model review**: the reviewer must PROVE each finding by running code; unproven claims
   are not accepted (GLM's confident wrong explanation, 2026-10-23) **[ours]**.
9. **Lead reads the whole diff.** Then a live check in an isolated Herdr session (CLAUDE.md 3),
   with the environment a real pane has (HISTORY 28: an empty environment hid a bug).
10. Owner decides merges, pushes, approvals and anything with secrets.

## 5. Loops - with caps and exits

| Loop | Steps | Cap / exit |
|---|---|---|
| Implement-verify | edge cases -> code -> run tests -> fix | 3 iterations, then escalate effort or model |
| Review | author -> reviewer (other family, proof required) -> lead triage -> fix | 2 rounds per finding, then the lead decides |
| Escalation ladder | ds-flash high -> Codex medium -> Codex high -> Codex xhigh -> lead | one step per failure, with the failure written into the next spec |
| Mutation | mutate -> tests must fail -> surviving mutant -> new test | until no meaningful mutant survives |
| Correction | lead sees drift on screen -> `esc` -> one precise correction | at once; after 2 drifts, stop the agent and rewrite the spec |

Every loop has a maximum (unbounded loops burn tokens **[practice]**) and writes its result
into a file the lead reads, not only the chat.

## 6. Failure modes seen here and the counter-measure

| Seen | Counter-measure |
|---|---|
| `ask` said DONE while the agent had just started (LESSONS #44) | confirm with `agent get`; read the result back |
| Herdr shows Cline `idle` while it works (#41) | wait for a result FILE (REPORT.md, answer file) |
| Gateway 403 on a long Codex conversation (#45) | one task per fresh conversation; context in a spec file |
| Model states a wrong cause with confidence | "prove every claim by running it"; verify before accepting |
| Spec author's blind spot (HISTORY 28) | test with the real pane environment; review the lead's own design too |
| Agent edits outside its files | file ownership in the spec; `git status` scope check; `diff` |
| Context fills, details lost | small tasks; start fresh with a summary at ~80% |
| Two agents, one file | never; split by file or worktree |

## 7. Sources

- Agent-specific guidance: [OpenAI Codex prompting guide](https://developers.openai.com/cookbook/examples/gpt-5/codex_prompting_guide),
  [Anthropic: multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system),
  [Antigravity CLI best practices](https://antigravity.google/docs/cli/best-practices/),
  [Cline memory bank / plan-act](https://docs.cline.bot/prompting/cline-memory-bank),
  [AGENTS.md size and content](https://www.betterclaw.io/blog/agents-md-best-practices),
  [DeepSeek V4 for agentic coding (third-party)](https://techjacksolutions.com/ai-tools/deepseek/deepseek-v4-coding-and-agentic-workflows/)

- Anthropic patterns (orchestrator-workers, evaluator-optimizer), summarised:
  [Simon Willison](https://simonwillison.net/2024/Dec/20/building-effective-agents),
  [WorkOS](https://workos.com/blog/enterprise-ai-agent-playbook-what-anthropic-and-openai-reveal-about-building-production-ready-systems)
- Team size, isolation, plan gates, "verification is the bottleneck":
  [Addy Osmani's Code Agent Orchestra (summary)](https://codex.danielvaughan.com/2026/05/14/addy-osmani-code-agent-orchestra/),
  [Morph: agent orchestration](https://www.morphllm.com/ai-agent-orchestration),
  [Claude Code power-user tips](https://support.claude.com/en/articles/14554000-claude-code-power-user-tips)
- Flow engineering: [AlphaCodium (arXiv 2401.08500)](https://ar5iv.labs.arxiv.org/html/2401.08500)
- Test-based selection: [CodeT (arXiv 2207.10397)](https://arxiv.org/abs/2207.10397v2)
- Mutation / property / differential testing: [Meta ACH (arXiv 2501.12862)](https://arxiv.org/pdf/2501.12862),
  [LLMs and property-based tests (arXiv 2307.04346)](https://arxiv.org/pdf/2307.04346),
  [LLMs for unit testing, review (arXiv 2506.15227)](https://arxiv.org/pdf/2506.15227)
- Cross-model review and debate (evidence mostly indirect):
  [LLM consortium for design refinement (arXiv 2606.01490)](https://arxiv.org/pdf/2606.01490),
  [Debate for coding tasks (arXiv 2503.12029)](https://arxiv.org/pdf/2503.12029),
  [Model heterogeneity in debate (arXiv 2502.08788)](https://arxiv.org/html/2502.08788v3),
  [Debate diversity collapse](https://tianpan.co/blog/2026/04/26/debate-diversity-collapse-multi-agent-ensemble)
- Spec-driven development: [Morph: SDD with Kiro](https://www.morphllm.com/spec-driven-development),
  [SDD tools comparison](https://codemyspec.com/blog/spec-driven-development)
