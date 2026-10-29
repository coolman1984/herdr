# Task: commit, merge and push the Deck round 4 work to GitHub `main` and `windows-toolkit`

**Role:** you are a delegate (read AGENTS.md "If you were started by Claude"). Lead: Claude (pane wV:p1).
**Folder:** `D:\WORK\Software Development\GitHub\HERDR` (git repo, remote `origin` =
https://github.com/coolman1984/herdr.git, a PUBLIC repo). Local `main` tracks `origin/main`.
`origin/master` (upstream Herdr app) and `origin/herdr-app` (backup) - never merge, never push to them.

**Goal:** the uncommitted Deck round 4 work ends up on `origin/main` via a feature branch, `origin/windows-toolkit`
is fast-forwarded to the same commit, and local `main` equals `origin/main`.

**This spec explicitly allows:** `git add`, `git commit`, `git switch -c`, `git switch main`, `git fetch`,
`git merge`, `git push origin deck-round4`, `git push origin main`, `git push origin main:windows-toolkit`,
`git ls-remote`. Still forbidden: force push (`-f`, `--force*`, `+refspec`), reset, rebase, stash, deleting
branches, touching `master`/`herdr-app`, `gh auth`.

## Steps
0. **Fail fast.** `git fetch origin`. If it fails, or a write into `.git` is denied, STOP: `BLOCKED: <exact error>`.
1. Record `git ls-remote --heads origin main windows-toolkit` (before). `git status -sb`.
   If `origin/main` is ahead of local `main`, note it (step 7 merges it).
2. **Checks before committing** (repo root):
   - `pwsh -NoProfile -File check.ps1`: the known baseline is **28 FAIL** lines (20 key-binding lines for
     `I:\herdr\...`, 6 "skill DIFFERS", "LESSONS.md numbering breaks at 66", "HISTORY.md phase numbering
     breaks at Phase 43"). Any OTHER failure blocks: stop and report it.
   - `pwsh -NoProfile -File app\tests\go-locked.ps1 test ./... -count=1` must pass (never run go.exe directly).
   - `node --test app/tests/web/` must pass (30 tests).
3. **Branch:** `git switch -c deck-round4` (from current `main`, keeping the working tree).
4. **Stage exactly:** `git add -u` (all modified tracked files) plus these untracked paths:
   `app/internal/deck/admission.go`, `app/internal/deck/r4.go`, `app/internal/deck/r4_test.go`,
   `app/tests/go-lock.ps1`, `app/tests/go-locked.ps1`, `app/tests/web/` (only `contracts.test.mjs`,
   `dom.mjs`, `package.json` - stage nothing else from there), every `tasks/deck-r4-*.md`,
   `tasks/sol-push-deck-r4.md`.
   Do NOT stage: `tasks/sol-free-c-drive-audit-20261026/` (stays local, untracked), any `.exe`,
   `app/tests/go.lock`, `app/tests/runtime/`, `app/tests/shots/`, `app/tests/evidence/` changes are fine
   only if already tracked.
   Then `git diff --cached --stat`: no file over 1 MB, no binary, no secret-looking file (auth*, *.key,
   *.env, credentials, token files). Scan the staged diff for secrets:
   `git diff --cached -U0 | Select-String -Pattern '(sk-[A-Za-z0-9]{20,}|ghp_[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY|password\s*=\s*\S+)'`
   Any hit: STOP and report the file and line (do not print the secret itself).
5. **Commit** with message
   `Deck round 4: Astra review + fixes - reflow, agent_exited, contracts C1-C10 (HISTORY 50-51)`
   body: 3-5 lines summarising HISTORY.md Phases 50 and 51 (English only).
6. Verify the commit: `git show --stat HEAD | Select-Object -First 60`.
7. **Sync:** `git merge --no-edit origin/main` into `deck-round4` (no-op if up to date). Conflict:
   `git merge --abort`, answer BLOCKED with the files.
8. **Push the branch:** `git push -u origin deck-round4`.
9. **Merge into main:** `git switch main`; `git merge --no-ff deck-round4 -m "Merge branch 'deck-round4': Deck round 4"`;
   `git push origin main`; then `git push origin main:windows-toolkit` (must be a fast-forward; if refused,
   STOP and report - never force).
10. **Verify:** `git fetch origin`; `git rev-parse main origin/main origin/windows-toolkit` (all three equal);
    `git ls-remote --heads origin main windows-toolkit` (after); `git status -sb` (only the audit folder may
    remain untracked); `git log --oneline -3`.

## Answer format (short)
```
RESULT: DONE | BLOCKED | FAILED
check.ps1: <28 baseline | other failures: ...>
go test: <pass | output>   node test: <30 pass | output>
secret scan: <clean | hits (file:line only)>
commit: <sha> on deck-round4, files: <n>
pushed: deck-round4 -> <sha>, main -> <sha>, windows-toolkit -> <sha>
ls-remote before: <lines>   after: <lines>
main == origin/main == origin/windows-toolkit: <yes/no, shas>
git status after: <lines>
doubts: <anything not verified>
```
