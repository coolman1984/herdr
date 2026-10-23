# Task: merge the audit fixes into main and push the toolkit to GitHub

## 1. Role and goal
You are a careful release engineer working for the lead (Claude).
Goal: local `main` of this folder gets the reviewed branch `fix/bug-audit` (fast-forward), plus
two task-spec files, and the GitHub branch `windows-toolkit` of
`https://github.com/coolman1984/herdr` then holds exactly that `main`.
Why: the owner keeps this PowerShell toolkit on its own branch of a PUBLIC fork. The fork's
`main` and `master` hold the Herdr Rust application and must stay untouched - pushing there
would destroy it. The lead has already reviewed every change and scanned it for secrets.

## 2. Context
- Folder: `D:\WORK\Software Development\GitHub\HERDR` (your current folder), branch `main`.
- Expected state now: `main` = `d2fd00fa`; branch `fix/bug-audit` = `87056896` (3 commits ahead
  of `main`, `main` is its ancestor); untracked files `tasks/codex-bug-audit.md` and
  `tasks/push-fixes.md` (this file); nothing else changed.
- `origin/windows-toolkit` = `3d547035`, an ancestor of `main` - the push is a fast-forward.
- GitHub sign-in: Git Credential Manager. Never ask for, type or print a token or password.
  If git asks for credentials or a browser sign-in appears: STOP and report BLOCKED.

## 3. Exact scope - run ONLY these commands, in PowerShell, one at a time
`git status`, `git log`, `git rev-parse`, `git merge-base`, `git ls-remote`, `git fetch origin`,
`git merge --ff-only fix/bug-audit`, `git add tasks/codex-bug-audit.md tasks/push-fixes.md`,
`git commit -m "<message below>"`, `git push origin main:windows-toolkit`.
Do NOT: edit, create or delete any file; push to `main` or `master` on GitHub; use `--force`,
`-f`, `--force-with-lease`, `rebase`, `reset`, `checkout`, `stash`, `branch -d`,
`worktree remove`; change remotes or git config; run any other program or script.

## 4. Steps (stop at the first unexpected result and report it - do not improvise a fix)
1. `git status --short --branch` -> on `main`, only the two untracked task files listed above.
2. `git rev-parse main fix/bug-audit` -> `d2fd00fa...` and `87056896...`;
   `git merge-base --is-ancestor main fix/bug-audit` -> exit 0.
3. Record `git ls-remote --heads origin main master windows-toolkit` (the BEFORE values).
4. `git merge --ff-only fix/bug-audit` -> "Fast-forward"; `git rev-parse HEAD` = `87056896...`.
5. `git add tasks/codex-bug-audit.md tasks/push-fixes.md`, then
   `git commit -m "Task specs of the audit and of this push (HISTORY 25-27)"`.
6. `git fetch origin`; `git merge-base --is-ancestor origin/windows-toolkit HEAD` -> exit 0.
7. `git push origin main:windows-toolkit`.
8. `git ls-remote --heads origin main master windows-toolkit` (the AFTER values).

## 5. Definition of done
- [ ] AFTER `windows-toolkit` sha = `git rev-parse HEAD`.
- [ ] AFTER `main` and `master` shas = BEFORE (unchanged).
- [ ] `git status --short` is empty.

## 6. Answer format
Reply with exactly:
DONE | BLOCKED | PARTIAL
Merged: d2fd00fa -> <sha> (fast-forward), spec commit <sha>
Pushed: <old sha> -> <new sha> on windows-toolkit
BEFORE: main <sha> master <sha> windows-toolkit <sha>
AFTER:  main <sha> master <sha> windows-toolkit <sha>
Checks: <each check from section 5 and its result>
Not done / doubts: <list, or "none">
