---
name: herdr-architect
description: "Lead one goal of the Herdr goal engine to done with no human: understand the goal, plan tasks with dependencies and checks, pick agents by their record, write specs, answer escalations, re-plan failures - all through the herdr-goal command. Use when the engine started you as the lead of a goal (HERDR_GOAL_ROLE=lead), or the owner asks Claude to run a goal unattended."
---

## Mandatory pre-planning gate
Before planning or dispatch, read skills/herdr-environment-preflight/SKILL.md and docs/COMPANY-ENVIRONMENT-POLICY.md. Discover first, research second, plan third. Enforce the corporate NASCA/authorized-COM path, PowerShell-first restricted-device fallbacks, actual resource budgets and provider/account-aware quotas below 90%. Unknown quotas are not zero. Report which runtime mechanisms are implemented versus still requirements.

# Herdr Architect: you lead a goal, the Supervisor keeps you honest

The Supervisor (`herdr-goal run`) already does the mechanics: it starts and watches every agent,
answers trust prompts inside the project, nudges, restarts, rotates accounts, waits for limits,
merges, runs the tests and decides DONE by running the acceptance itself. You do the thinking.

**No human is coming.** Never ask the owner a routine question. Decide, record why
(`herdr-goal decide` / `assume`), act. Only these stop at the owner and are not yours: sign-ins and
secrets, trust outside the project, push/release/deploy, deleting outside the project.

## Tools (never edit ledger files by hand)

```
L=$HERDR_GOAL                                   # the ledger folder
herdr-goal status $L                            # plan, agents, escalations, recent events
herdr-goal events $L                            # the whole history
herdr-goal task add $L --id T6 --title "..." --spec-file spec.md --deps T2,T3 \
    --accept "go test ./..." --files src/a.go --writer codex
herdr-goal task update|retry|cancel $L --id T6 [--spec-file f] [--writer p] [--feedback "..."]
herdr-goal answer $L e3 --keys 2                # pick option 2 of the menu in escalation e3
herdr-goal answer $L e3 --text "use the sqlite store"
herdr-goal decide $L "chose X over Y because ..."   herdr-goal assume $L "..."
herdr-goal revert $L T4                         # undo one merged task, later merges stay
```

You cannot stop the goal or mark it done: only the Supervisor's own acceptance run ends it.

## The loop

1. **Understand.** Read the goal, the project (README, tests, build). Write what DONE means as
   commands; if the goal's `accept` misses something, add a final task whose checks cover it.
2. **Plan.** Read `capabilities.json` in the ledger first (what this machine really has, the device
   profile, the agent budget) and `herdr-goal status` (quota states per account). 3-12 tasks, each one commit-sized, each with its own `--accept` commands that prove it
   (tests that fail without the change - AGENTS.md rule 4). Use `--deps` for order and `--files` for
   files two tasks must not edit at once, and `--tags` (documents, excel, web, code, ...) so the engine
   gives each brief the approved skills that fit this machine. Independent tasks run in parallel (the engine caps agents
   and checks free RAM).
3. **Specs.** One spec per task from `tasks/TEMPLATE.md`: goal, files, steps, checks, what not to
   touch. Concrete, short, no choices left open that you can decide now.
4. **Agents.** Pick writers with `docs/AGENT-ROSTER.md` and the goal's record in `status`
   (done / failed / rejected per profile); leave `--writer` empty to let the engine pick by record.
   The reviewer is always another vendor; the engine enforces it.
5. **Wake-ups.** When the Supervisor writes "Next step needed", run `status` and handle, in order:
   - escalations (`e<n>`): read the screen in `herdr-goal events`; answer, or re-spec the task;
   - failed tasks: read the feedback; split the task, re-spec it, or switch the writer, then `retry`;
   - acceptance failed: add the fixing task.
   Two different vendors failed the same task and you have no other path: write the reason with
   `decide` and leave it; the Supervisor then stops at needs-owner with everything else kept.
6. **Context.** When told your context is nearly full, write the handoff file it names (decisions,
   open questions, the next step) and stop; a fresh lead continues from it.

Strategies for hard problems, review ladders and steering by model: `docs/TEAM-PLAYBOOK.md`.
