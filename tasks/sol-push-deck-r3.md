# Task: commit, merge and push the Deck round 3 work to GitHub `main`

**Role:** you are a delegate (read AGENTS.md "If you were started by Claude"). Lead: Claude (pane wV:p1).
**Folder:** `D:\WORK\Software Development\GitHub\HERDR` (git repo, remote `origin` =
https://github.com/coolman1984/herdr.git). Local branch `main` tracks `origin/main`.
`origin/master` is the UPSTREAM Herdr app - never merge it, never push to it.

**Goal:** the uncommitted Deck round 3 work in the working tree ends up on GitHub `origin/main`, merged
via a feature branch, and local `main` is identical to `origin/main` afterwards.

**This spec explicitly allows:** `git add`, `git commit`, `git switch`/`checkout -b`, `git fetch`,
`git merge`, `git push origin <branch>` and `git push origin main`. Still forbidden: force push, reset,
rebase of `main`, stash, deleting branches, touching `origin/master`, `gh auth` (gh is not signed in -
do not try to sign in).

## Steps

0. **Fail fast.** Run `git fetch origin`. If it fails (network blocked by your sandbox), or if a later
   write into `.git` fails with access denied, STOP and answer `BLOCKED: <exact error>` - do not try
   workarounds.
1. `git status -sb`, `git log --oneline -1 origin/main`. If `origin/main` moved past local `main`, note it.
2. **Checks before committing** (from the repo root):
   - `pwsh -NoProfile -File check.ps1` - record pass/fail and the failure lines. Known baseline: main
     already had machine-setup failures (foreign key paths, stale installed skill copies, duplicate
     history/lesson numbers - HISTORY.md end of Phase 46). Those do not block. A failure caused by a
     file in this commit (e.g. a malformed HISTORY/LESSONS entry from Phase 47) DOES block: stop and report.
   - `cd app; go test ./...` (if `go` is not on PATH, use the Go path that `app\build.ps1` resolves).
     Any failing test blocks: stop and report the output.
3. **Branch:** `git switch -c deck-round3` (from current `main`, keeping the working tree).
4. **Ignore the junk** - append these lines to `app/.gitignore` (keep existing lines):
   ```
   /HerdrDeck.exe~
   /tests/tmp-edge/
   /tests/shots/
   ```
5. **Stage exactly:** all modified tracked files (`git add -u`), plus these untracked paths:
   `app/.gitignore`, `app/internal/deck/input.go`, `app/internal/deck/input_test.go`,
   `app/internal/deck/r3_test.go`, `app/tests/deck-ui.ps1`, `app/tests/fixtures/`,
   `app/tests/input-stub.ps1`, `app/tests/verify-ui.ps1`, every `tasks/deck-r3-*.md`,
   `tasks/deck-round3-*.md`, `tasks/opus-nerp-*.md`, `tasks/sol-free-c-drive.md`,
   `tasks/sol-free-c-drive-result.md`, `tasks/sol-push-deck-r3.md`.
   Do NOT stage: `app/HerdrDeck.exe~`, `app/tests/tmp-edge/`, `app/tests/shots/`,
   `tasks/sol-free-c-drive-audit-20261026/` (raw machine survey - stays local, untracked).
   Before committing, run `git diff --cached --stat` and check that no file over 1 MB and no
   secret-looking file (auth*, *.key, *.env, token) is staged.
6. **Commit** with message:
   `Deck round 3: input guard, R19-R24, UI speed; C: cleanup report (HISTORY 47)`
   (body: 2-4 lines summarising what HISTORY.md Phase 47 says; English only).
7. **Sync:** `git merge --no-edit origin/main` into `deck-round3` (no-op if up to date). On a conflict:
   `git merge --abort` and answer BLOCKED with the conflicting files.
8. **Push the branch:** `git push -u origin deck-round3`.
9. **Merge into main:** `git switch main`, `git merge --no-ff deck-round3 -m "Merge branch 'deck-round3': Deck round 3"`,
   then `git push origin main`.
10. **Verify:** `git fetch origin`, then `git rev-parse main origin/main` (must be equal),
    `git status -sb` (only the three ignored/untracked leftovers may remain: the audit folder),
    `git log --oneline -3`.

## Answer format (short)

```
RESULT: DONE | BLOCKED | FAILED
check.ps1: <pass | N failures, all baseline | blocking: ...>
go test: <pass | output>
commit: <sha> on deck-round3, files: <n>
pushed: deck-round3 -> <sha>, main -> <sha>
main == origin/main: <yes/no, both shas>
git status after: <lines>
doubts: <anything not verified>
```
