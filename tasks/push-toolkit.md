# Task: push the HERDR toolkit's newest commit to its own branch on GitHub

## 1. Role and goal
You are a careful release engineer working for the lead (Claude).
Goal: the GitHub branch `windows-toolkit` of `https://github.com/coolman1984/herdr` must hold
exactly the local `main` of this folder (`git rev-parse HEAD`).
Why: the owner keeps this PowerShell toolkit on its own branch of a PUBLIC fork; the fork's
`main` / `master` hold the Herdr Rust application and must stay untouched.

## 2. Context
- Folder: `D:\WORK\Software Development\GitHub\HERDR` (git repo, branch `main`, clean).
- Remote `origin` = `https://github.com/coolman1984/herdr.git`, already set; sign-in to
  GitHub is handled by Git Credential Manager (never ask for or print a token).
- Remote branch `origin/windows-toolkit` is at `cd9779c7`; local `main` is ahead of it, so
  the push is a fast-forward.
- The lead already checked the new commit for secrets: none.

## 3. Exact scope
- Run only these git commands (PowerShell): `git status`, `git fetch origin`, `git log`,
  `git rev-parse`, `git ls-remote`, `git push origin main:windows-toolkit`.
- Do NOT: change, create or delete any file; commit; merge; rebase; push to `main` or
  `master`; use `--force` / `-f` / `--force-with-lease`; change remotes or git config.

## 4. Steps
1. `git status --short --branch` -> must be clean on `main`. If not clean: stop, report.
2. `git fetch origin`.
3. `git rev-parse origin/windows-toolkit` must be `cd9779c7...` and an ancestor of HEAD:
   `git merge-base --is-ancestor origin/windows-toolkit HEAD` (exit 0). If not: stop, report.
4. `git push origin main:windows-toolkit`.
5. `git ls-remote --heads origin windows-toolkit` and `git rev-parse HEAD` must match.

## 5. Definition of done
- [ ] `git ls-remote --heads origin windows-toolkit` = `git rev-parse HEAD`.
- [ ] `origin/main` and `origin/master` unchanged (`git ls-remote --heads origin main master`
      before and after are the same).
- [ ] Working tree still clean.

## 6. Answer format
Reply with exactly:
DONE | BLOCKED | PARTIAL
Pushed: <old sha> -> <new sha> on windows-toolkit
Checks: <each check from section 5 and its result>
Not done / doubts: <list, or "none">
