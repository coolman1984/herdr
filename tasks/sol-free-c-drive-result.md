# C: safe cleanup result — 2026-10-26

Status: PARTIAL. Approved cleanup completed with skips; npm cache removal encountered access denied. Read-only survey completed, with access gaps listed below.

## Drive measurements

| Get-PSDrive C measurement | Timestamp | Free bytes | Free GB (GiB) |
|---|---|---:|---:|
| Before | 10/26/2026 08:15:16 | 1681948672 | 1.566437 |
| After | 10/26/2026 08:16:45 | 2716782592 | 2.530201 |
| Net gain | | 1034833920 | 0.963764 |

GB in this report means PowerShell 1GB = 1,073,741,824 bytes (GiB). The drive total is 123,077,652,480 bytes (114.62 GiB). The after measurement was recorded after the final dry run and before the read-only survey.

## Freed by target

| Approved target | Before logical bytes | Remaining logical bytes | Logical byte reduction | Remaining files |
|---|---:|---:|---:|---:|
| C:\Users\m.labib\AppData\Local\npm-cache | 217077018 | 78184333 | 138892685 | 2785 |
| C:\Users\m.labib\AppData\Local\CrashDumps | 50560534 | 0 | 50560534 | 0 |
| C:\Users\m.labib\AppData\Local\Temp | 957469097 | 211059 | 957258038 | 21 |
| C: Recycle Bin | Not separately measured | See system survey | Not separately attributable | See system survey |

Per-target reductions are file-length measurements, not physical allocation. They must not be summed as an exact disk-space gain; use the Get-PSDrive delta above. Recycle Bin was cleared by the skill; no separate reclaimed-byte estimate is claimed.

## Approved paths and invocation audit

Used the installed c-drive-safe-cleanup and windows-disk-cleanup-guard skills without editing their scripts. Executed in order: Get-PSDrive C; run-c-drive-cleanup.ps1 -Mode Inspect; -Mode Preview; -Mode Clean; -Mode VsCodePreview; -Mode Inspect; scan-cleanup-targets.ps1 -Json; clean-cleanup-targets.ps1 -DryRun; Get-PSDrive C; read-only survey.

The primary wrapper invokes the same guard cleanup script and already covers every standard target, including uv, CachedExtensionVSIXs and Temp. No additional guard -NoInteractive pass ran: there were no uncovered targets, and repeating it would retry failed/locked paths. npm cache clean --force was not run after the failed npm deletion.

Before live deletion, all three existing target roots resolved under C:\Users\m.labib\AppData\Local; recursive metadata enumeration found no reparse points in them. USERPROFILE, APPDATA and TEMP resolved to the expected C: profile paths. All actual file deletion attempts were inside the following allowlist:
- C:\Users\m.labib\AppData\Local\highlight-updater
- C:\Users\m.labib\AppData\Local\lm-studio-updater
- C:\Users\m.labib\AppData\Local\remnote-updater
- C:\Users\m.labib\AppData\Local\fellou-updater
- C:\Users\m.labib\AppData\Local\obsidian-updater
- C:\Users\m.labib\AppData\Local\perplexity-updater
- C:\Users\m.labib\AppData\Local\monica-updater
- C:\Users\m.labib\AppData\Local\notion-updater
- C:\Users\m.labib\AppData\Local\capacities-updater
- C:\Users\m.labib\AppData\Local\noteey-updater
- C:\Users\m.labib\AppData\Local\npm-cache
- C:\Users\m.labib\AppData\Local\pip\Cache
- C:\Users\m.labib\AppData\Local\uv\cache
- C:\Users\m.labib\.cache\codex-runtimes
- C:\Users\m.labib\AppData\Roaming\Code\CachedExtensionVSIXs
- C:\Users\m.labib\AppData\Local\CrashDumps
- C:\Users\m.labib\AppData\Local\Temp
- C: Recycle Bin

Recycle Bin safeguard: the existing script uses Clear-RecycleBin without a drive argument. In this one PowerShell invocation, $PSDefaultParameterValues['Clear-RecycleBin:DriveLetter'] = 'C' constrained that cmdlet to C:. No cleanup script was rewritten.

VS Code preview: installation not found at C:\Users\m.labib\AppData\Local\Programs\Microsoft VS Code. No old-version folders were deleted. The script found no code processes. No user processes were closed.

No protected personal, browser, agent-state, application-install or Windows locations were deleted. No git, browser, install, settings change, administrator launch, DISM, cleanmgr or powercfg command ran. D: received only this requested report and its audit artifacts.

## Skipped / locked and exact errors

Captured 30 removal error records for 30 distinct paths: 21 locked files, 8 nonempty parent directories, and 1 access-denied npm file. No retries. npm had 2,785 files remaining; these are not all claimed to be locked, because its recursive deletion stopped at the first denied file. Temp had 21 files remaining, totaling 211,059 bytes.

The canonical cleanup summary reports Deleted: 2 / Failed: 1 / Skipped: 0 at target level. Its Temp branch suppresses individual errors and credits the full pre-clean size; captured errors and post-clean measurements above provide the actual residual evidence.

| Path | Exact error |
|---|---|
| C:\Users\m.labib\AppData\Local\Temp\f2a74122-2608-4744-98bb-ba0854091f2c.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\f2a74122-2608-4744-98bb-ba0854091f2c.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\e57cc7ac-598d-4d1a-bc38-e5ff4f00eccc.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\e57cc7ac-598d-4d1a-bc38-e5ff4f00eccc.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\c2f507ee-b3c3-4972-9c60-28e96c3424af.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\c2f507ee-b3c3-4972-9c60-28e96c3424af.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\c2499692-f858-4516-a052-f07f463e2d41.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\c2499692-f858-4516-a052-f07f463e2d41.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\b5a48874-aa84-4db0-b255-27193a1bb3c6.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\b5a48874-aa84-4db0-b255-27193a1bb3c6.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\ad02288a-d265-4911-9ee9-5853b408627a.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\ad02288a-d265-4911-9ee9-5853b408627a.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\a89724ea-f4ec-4afb-ad16-ed096da9132d.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\a89724ea-f4ec-4afb-ad16-ed096da9132d.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\a0057260-056a-45eb-ae01-dc995e123d46.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\a0057260-056a-45eb-ae01-dc995e123d46.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\995938c5-3bc6-489a-84b6-a0ed41a1d419.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\995938c5-3bc6-489a-84b6-a0ed41a1d419.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\98dc3197-7556-4b3a-b953-f35d682c6f33.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\98dc3197-7556-4b3a-b953-f35d682c6f33.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\943e6a71-d72c-409c-bdda-a58fb0097c45.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\943e6a71-d72c-409c-bdda-a58fb0097c45.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\777e5c20-8ea1-4efb-b415-befcf4499b33.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\777e5c20-8ea1-4efb-b415-befcf4499b33.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\6d79d3a1-d3ec-4edd-966e-1cf6055648a3.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\6d79d3a1-d3ec-4edd-966e-1cf6055648a3.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\4c366e49-e510-40b3-811b-2c2d56914ace.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\4c366e49-e510-40b3-811b-2c2d56914ace.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\43b5fde7-2473-4853-8013-7cac62ef02fc.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\43b5fde7-2473-4853-8013-7cac62ef02fc.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\1a3b4b41-2a90-4d89-82d5-1a89481d652d.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\1a3b4b41-2a90-4d89-82d5-1a89481d652d.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\138660cc-9a7c-4a1c-a3da-8ac1d5558955.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\138660cc-9a7c-4a1c-a3da-8ac1d5558955.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\02f59c0e-ac8f-4ae3-bf62-736b92223095.tmp | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\02f59c0e-ac8f-4ae3-bf62-736b92223095.tmp' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\SVPNManager | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\SVPNManager'. |
| C:\Users\m.labib\AppData\Local\Temp\SVPNManager\log_2026-10-26.log | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\SVPNManager\log_2026-10-26.log' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\Samsung | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\Samsung'. |
| C:\Users\m.labib\AppData\Local\Temp\Samsung\SamsungUpdate3 | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\Samsung\SamsungUpdate3'. |
| C:\Users\m.labib\AppData\Local\Temp\Samsung\SamsungUpdate3\log | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\Samsung\SamsungUpdate3\log'. |
| C:\Users\m.labib\AppData\Local\Temp\Samsung\SamsungUpdate3\log\SU3UserModeWorker.log | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\Samsung\SamsungUpdate3\log\SU3UserModeWorker.log' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\Temp\claude | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\claude'. |
| C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR'. |
| C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR\0ad4bc81-6eef-4dab-9b55-dda1eed09244 | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR\0ad4bc81-6eef-4dab-9b55-dda1eed09244'. |
| C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR\0ad4bc81-6eef-4dab-9b55-dda1eed09244\tasks | The directory is not empty. : 'C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR\0ad4bc81-6eef-4dab-9b55-dda1eed09244\tasks'. |
| C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR\0ad4bc81-6eef-4dab-9b55-dda1eed09244\tasks\bzd3dweb6.output | The process cannot access the file 'C:\Users\m.labib\AppData\Local\Temp\claude\D--WORK-Software-Development-GitHub-HERDR\0ad4bc81-6eef-4dab-9b55-dda1eed09244\tasks\bzd3dweb6.output' because it is being used by another process. |
| C:\Users\m.labib\AppData\Local\npm-cache\_npx\6583fba12287d067\node_modules\@napi-rs\canvas-win32-x64-msvc\skia.win32-x64-msvc.node | Access to the path 'C:\Users\m.labib\AppData\Local\npm-cache\_npx\6583fba12287d067\node_modules\@napi-rs\canvas-win32-x64-msvc\skia.win32-x64-msvc.node' is denied. |

A separate captured Get-Process error says: Cannot find a process with the name "code". Verify the process name and call the cmdlet again. This is a process-detection result, excluded from removal-error counts. Post-clean sizing recorded Cannot find path 'C:\Users\m.labib\AppData\Local\CrashDumps' because it does not exist; this is the successfully removed target, not an unreadable folder.

## Audit logs

- C:\Users\m.labib\.cache\disk-cleanup-guard\cleanup-log.txt
- C:\Users\m.labib\.cache\disk-cleanup-guard\scan-log.txt
- C:\Users\m.labib\.cache\disk-cleanup-guard\vscode-cleanup-log.txt

This run also saved canonical-log excerpts, command transcripts, exact Get-PSDrive JSON readings, before/after target measurements, the full residual file list, and exact deletion errors in:

D:\WORK\Software Development\GitHub\HERDR\tasks\sol-free-c-drive-audit-20261026
- 01-inspect.txt
- 02-preview.txt
- 03-clean.txt
- 04-vscode-preview.txt
- 05-inspect-after.txt
- 06-guard-scan.json
- 07-guard-dryrun.txt
- after.json
- before.json
- cleanup-errors.json
- cleanup-log.txt
- scan-log.txt
- skipped-paths.json
- targets-after.json
- targets-before.json
- vscode-cleanup-log.txt

## Read-only survey

Completed 10/26/2026 08:30:37: 68327 directories enumerated. Ranked folders at depths 1–3 under C:\Users\m.labib, including AppData\Local and AppData\Roaming, by recursively summed file lengths. Only metadata was read. Parent and child rows overlap: do not add them together. Sizes exclude unreadable locations and 3,331 reparse entries (files/directories, including junctions and cloud placeholders); those links were not followed to other drives. This is a logical-size ranking, not physical allocated disk usage or an exhaustive C: drive inventory.

| Rank | Path | Size GB (GiB) | Suggestion |
|---:|---|---:|---|
| 1 | C:\Users\m.labib\AppData | 21.1544 | app-managed |
| 2 | C:\Users\m.labib\AppData\Local | 14.4026 | app-managed |
| 3 | C:\Users\m.labib\AppData\Roaming | 6.5938 | app-managed |
| 4 | C:\Users\m.labib\Downloads | 6.4082 | owner decides |
| 5 | C:\Users\m.labib\AppData\Local\Microsoft | 4.3457 | app-managed |
| 6 | C:\Users\m.labib\.codex | 3.2649 | keep |
| 7 | C:\Users\m.labib\AppData\Local\Google | 2.6886 | keep |
| 8 | C:\Users\m.labib\.rustup | 2.6363 | owner decides |
| 9 | C:\Users\m.labib\.rustup\toolchains | 2.5098 | owner decides |
| 10 | C:\Users\m.labib\AppData\Roaming\Claude | 2.3094 | app-managed |
| 11 | C:\Users\m.labib\Downloads\03_AI_Agent_Builder_&_Automation | 1.9982 | owner decides |
| 12 | C:\Users\m.labib\Documents | 1.7531 | owner decides |
| 13 | C:\Users\m.labib\.local | 1.4523 | owner decides |
| 14 | C:\Users\m.labib\Downloads\03_AI_Agent_Builder_&_Automation\03_AI_Tools_&_Code_Repositories | 1.4113 | owner decides |
| 15 | C:\Users\m.labib\.rustup\toolchains\stable-x86_64-pc-windows-msvc | 1.2989 | owner decides |
| 16 | C:\Users\m.labib\AppData\Roaming\npm | 1.2894 | app-managed |
| 17 | C:\Users\m.labib\.rustup\toolchains\1.96.1-x86_64-pc-windows-msvc | 1.2109 | owner decides |
| 18 | C:\Users\m.labib\.codex\sessions\2026 | 1.1146 | keep |
| 19 | C:\Users\m.labib\.codex\sessions | 1.1146 | keep |
| 20 | C:\Users\m.labib\.local\share | 1.1132 | owner decides |
| 21 | C:\Users\m.labib\.nuget\packages | 0.9879 | owner decides |
| 22 | C:\Users\m.labib\.nuget | 0.9879 | owner decides |
| 23 | C:\Users\m.labib\.claude | 0.9228 | keep |
| 24 | C:\Users\m.labib\AppData\Local\Packages | 0.9002 | app-managed |
| 25 | C:\Users\m.labib\Downloads\02_AX_&_Factory_Operations | 0.8656 | owner decides |

Suggestions are for owner review only. Personal folders and all agent/browser state remain protected. No survey result was deleted or moved.

### Required system paths — read-only

| Path | Readable bytes | Size GB (GiB) | Suggestion | Result |
|---|---:|---:|---|---|
| C:\Windows\SoftwareDistribution | 986915369 | 0.9191 | keep | Measured |
| C:\hiberfil.sys |  |  | keep | Not found; size unavailable, not zero |
| C:\pagefile.sys | 2147483648 | 2 | keep | Measured |
| C:\$Recycle.Bin | 131 | 0 | keep | Lower bound; 6 access-denied folders |

Unreadable folders: 7 total — 1 under the user profile, plus 6 protected SID folders under C:\$Recycle.Bin. Missing hiberfil.sys is separate and is not counted as an unreadable folder. Excluded reparse entries are also counted separately.

- Unreadable profile folder: C:\Users\m.labib\Downloads\03_AI_Agent_Builder_&_Automation\02_Excel_Automation_Blueprints\Sample_Sales_Performance_App\Sample_Sales_Performance_App\.pytest_cache
- Cannot find path 'C:\hiberfil.sys' because it does not exist.
- Access to the path 'C:\$Recycle.Bin\S-1-5-18' is denied.
- Access to the path 'C:\$Recycle.Bin\S-1-5-21-1446346333-3922582964-4124780767-500' is denied.
- Access to the path 'C:\$Recycle.Bin\S-1-5-21-2621231314-1007001964-1357254385-500' is denied.
- Access to the path 'C:\$Recycle.Bin\S-1-5-21-2755395243-1138931703-1776990300-1000' is denied.
- Access to the path 'C:\$Recycle.Bin\S-1-5-21-2755395243-1138931703-1776990300-1001' is denied.
- Access to the path 'C:\$Recycle.Bin\S-1-5-21-2755395243-1138931703-1776990300-500' is denied.

The 131 readable bytes in C:\$Recycle.Bin do not prove every account's bin is empty; six protected SID directories could not be inspected. No elevation or permission changes were attempted.

Additional survey audit files:
- D:\WORK\Software Development\GitHub\HERDR\tasks\sol-free-c-drive-audit-20261026\08-profile-survey.json
- D:\WORK\Software Development\GitHub\HERDR\tasks\sol-free-c-drive-audit-20261026\09-system-survey.json

## Not done / doubts

- npm cache deletion was partial due to the exact access-denied error above. Locked Temp files and their nonempty parent folders were left in place; no processes were stopped or deletion retries attempted.
- The guard's extra live pass was unnecessary: its full allowlist had already been attempted by the primary wrapper. No fresh target was found by the second scan/dry run.
- Seven folders could not be read; 3,331 reparse entries were excluded. Reported folder sizes are readable logical totals, not allocated size, and nested rows overlap.
- hiberfil.sys was not found; its size is unavailable. Recycle Bin reclaimed bytes were not independently attributable, and other accounts' bins were unreadable.
- C: still has only about 2.53 GiB free at the recorded after measurement. Remaining large locations were surveyed only and require owner decisions outside this cleanup scope.

