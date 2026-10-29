# Task: learn this project, then wait for orders

You are Claude Opus 5.5 (medium), a helper started by the lead (Claude in the pane to your left).
Working folder: `D:\WORK\Software Development\GitHub\Computer Use\opening-nerp-tcode`.

## Goal
Build a working understanding of this project so you can take real automation tasks next.
Focus on the **department one-click CLI tools** in `OneClick_Tools/` (Departments_Daily, PE_Mechanical,
Press, SMD_VD, `_engine`, `_source`, `catalog.json`, `CATALOG.md`, `README.txt`) and on how they
call the GMES / NERP automation engine (`GMES/app/engine/`, `gmes_*.py`, `cdp_common.py`).

## Read, in this order
1. `CLAUDE.md` and `AGENTS.md` of this project (its rules bind you), `README.md`, `docs/FILES.md`.
2. `OneClick_Tools/README.txt`, `CATALOG.md`, `catalog.json`, then each department folder.
3. `ARCHITECTURE.md`, and only the parts of `AGENT_PLAYBOOK.md` / `HISTORY.md` (it is ~940 KB -
   search it, do not read it whole) that explain the department tools and their newest phases.
4. The engine code the tools actually call - follow the calls, do not read every file.

## Rules
- **Read only.** Change, create, delete or format nothing. No git commands that write.
  The working tree has many uncommitted changes that are the owner's - leave them as they are.
- Run nothing that logs in to GMES/NERP, opens a browser, or touches real company systems.
  Never open or print credentials (`gmes_credentials.py` data, keys, passwords, tokens).
- No subagents, no other agents, no browser tools.

## Answer (brief, in English)
1. What the project does, in 5 lines.
2. A table of the department tools: department | tool | what it automates | entry file | how it is run.
3. How a tool flows: launcher -> engine -> GMES/NERP screen -> output (files and functions).
4. How a new department tool is added (the steps and files, from the docs/code).
5. Open questions and risks you noticed (unclear docs, failing pieces) - mark anything unverified.

Then write `READY - waiting for the next task.` and **stop and wait**. The next order will come
from the owner or the lead in this pane.
