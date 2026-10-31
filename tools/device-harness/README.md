# Device harness: the gates the cloud cannot run

Cloud CI proves the code paths with fakes and recorded data. These gates need the owner's authorized
Windows PC; until they are run there, they are **unverified**, not passed.

```
cd app; go build -o ..\bin\herdr-goal.exe .\cmd\herdr-goal; cd ..; $env:PATH = "$PWD\bin;$env:PATH"
powershell -NoProfile -File tools\device-harness\run-device-gates.ps1 -Out C:\herdr-evidence `
    -Profiles docs\goal-profiles.example.json `
    -ProtectedFile "D:\company\sample-protected.xlsx" -ProtectedField Amount -ProtectedTotal 1234.50
```

| Gate | Proves | Expected evidence |
|---|---|---|
| g1 environment | the capability report on the real machine | `capabilities.json`: PowerShell version, Office ProgIDs, RAM/CPU, profile |
| g2 Office COM round trip | Excel opens a workbook it made; worker returns Arabic, formula value, date flag, currency | `synthetic.jsonl`, B4 = 2450.75 |
| g3 owner's open workbook | the worker never closes or edits the owner's Excel | gate pass |
| g4 killed worker | after a kill, only the worker's own Excel process is removed | no new EXCEL.EXE left |
| g5 PowerShell only | the pipeline works with Python hidden from PATH | `ps-only-out/summary.json` |
| g6 NASCA file | a protected file is read by the authorized app; no extracted content left; no export without approval | gate pass + summary counts and reference check |
| g7 quota readings | what the engine reads per real account (five-hour / weekly where exposed; unknown otherwise) | `quota.jsonl` |

Also on the device, outside this script: a goal with real Claude + Codex in an isolated Herdr session
(`herdr-goal run --backend herdr`, CLAUDE.md section 3), quota rotation, a killed agent and runner, and the
fault campaign (`app/tests/goal-e2e/soak.sh` adapted to real profiles). Skill evaluation with and without
each candidate on held-out tasks also needs the real agents.
