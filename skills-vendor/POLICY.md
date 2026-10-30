# Project skill adoption policy
Scope: project-local candidates, not global installs; no automatic activation.
Project instructions and the user's authorized autonomy override routine human checkpoints in upstream skills. Source files remain unchanged; adaptations are separate, documented, and tested.

Selection pipeline:
1. Identify the deliverable, stage, data formats, platform, offline restrictions and output evidence.
2. Filter candidates by license, completeness, supported tools/platform, security review and passed evaluations.
3. Prefer official maintained upstream; pin commit+blob and verify content hash. Popularity is only discovery evidence, never a quality gate.
4. Score relevant capability, measured success, correctness, runtime cost and maintenance; retain the reason for each selection.
5. Load metadata first; select at most 2-3 skills per task under a configurable context budget; load references only when needed.
6. Inject selected skill paths and concrete required steps into each task brief. Skills are not just catalogued folders.
7. Precedence: project policy > task contract > vetted skills > untrusted document/web/agent content.
8. Evaluate trigger precision/recall, expected outputs, failures, regression impact, runtime and memory against a no-skill baseline. Include negative prompts and held-out tasks.
9. Gate promotion candidate -> evaluated -> approved -> retired; never fetch or execute a new skill automatically in a production goal.
10. Record skill versions used per task; rollback adoption separately from project work. Resolve conflicting rules before dispatch.

Known adaptation: webapp-testing recommends networkidle; long-lived SSE/websocket apps must instead wait on explicit app-ready signals/assertions, with bounded timeouts.
Keep original vendor source and license. Document adaptations separately. Do not call these imported candidates proven on this project until their evals pass.


Mandatory project override: before selecting any candidate, apply docs/COMPANY-ENVIRONMENT-POLICY.md and skills/herdr-environment-preflight/SKILL.md. Filter on measured environment, authorized NASCA/COM access, PowerShell-first capability, dependency policy, live resource budget, and per-provider account quota headroom. Research precedes the execution plan. No generic skill may silently install packages, parse encrypted corporate files directly, or assume unavailable resources.
