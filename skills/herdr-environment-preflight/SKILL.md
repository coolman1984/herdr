---
name: herdr-environment-preflight
description: Mandatory before planning or dispatching any Herdr goal, and after environment or account changes. Discover actual machine constraints, enforce NASCA-authorized COM file access, choose PowerShell-first package-free fallbacks, and check observable provider quotas below 90 percent.
---

# Environment-first planning

Read docs/COMPANY-ENVIRONMENT-POLICY.md before drafting the execution plan.
Sequence: bounded environment discovery -> current-source research -> risk register and strategic plan -> capability/skill/account selection -> dispatch.

Required evidence:
- Machine/session/permissions/tool versions/resources/network and installation policy, with timestamps and unknowns.
- Confirmed authorized document path; corporate NASCA profile requires native authorized application automation, Office COM first. Never pass encrypted bytes to generic parsers or assume universal PDF COM.
- Windows tool ladder: available PowerShell + system COM; Python standard library; only then permitted installed/optional dependencies. A missing/denied package causes one bounded fallback transition, not an installation loop.
- Account/provider identity and observable five-hour/weekly/other limits. No fresh work at >=90% of any measured applicable limit; unknown/unsupported is explicit, never fabricated zero.
- A resource budget for all processes, task batch size, concurrency, quota headroom, fallback and safe resumption.
- Source-backed research and worst-case acceptance scenarios before implementation, with renewed investigation at unfamiliar failures.

Attach relevant capability facts, chosen fallback and quota policy to each delegate brief.
Project policy overrides generic upstream recommendations to install packages or assume powerful hardware.
This skill describes mandatory behavior; do not claim the engine implements it merely because the skill was loaded. Prove adapters, scheduling and fallbacks with tests and authorized real-device evidence.
