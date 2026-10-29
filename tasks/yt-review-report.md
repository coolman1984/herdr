# Trip Orders: read-only review

Reviewed 29 October 2026. Project: `I:\Yousef-Transportation`.

**Decision:** fix the saving, recovery and access problems below before a customer pilot. Keep the Python server. For the desktop window, use **pywebview with WebView2**, subject to a successful Windows packaging test. Changing the window alone will not fix the slow reports.

## Scope and evidence

- Read the project `CLAUDE.md` and `LAUNCH_SCOPE.md`. The first sale is an assisted, single-office pilot; recovery and access protection remain required.
- No project files, live databases, real accounts or secrets were changed or inspected. No git commands, browser tools, installation, build or deployment were used. Public documentation was researched through server-side web search.
- Execution used copied source and synthetic data under `I:\herdr\tasks\tmp`. Python **3.14.7, Windows x64**, on this PC. The build workflow selects Python **3.11**, so these are source-level Windows results, not certification of the shipped executable.
- At the end, SHA-256 comparisons of 15 reviewed server modules against the copies found **zero differences**. Original `server/app.py` SHA-256: `327c809594db293cffcc1b89c5f6ebb8afa98b144dd715a843577eb5b9e77e38`.
- “Reproduced” below means the current copied code actually exhibited the result. “Source verified” means the branch is present in the code; the customer environment was **not verified**. Expected performance gains are separated from measurements.

## Part 1 — findings, ordered by severity

### 1. High — a category-limited user can restore every category

**Where:** `server/app.py:1355`, `server/store.py:519`, `server/store.py:550`.

**What happens:** the restore endpoint checks `backups.restore` but does not require access to all categories. `restore_from()` makes a forced, whole-database change without the normal category guard. A person restricted to category A can change or soft-delete category B's records through a backup restore.

**Trigger:** give a user category A and `backups.restore`; change a B trip after a backup; restore that backup as the A user. This is possible through a custom permission assignment.

**Proof — reproduced over HTTP:** the limited user's state contained **0 trips**. `POST /api/backups/restore` returned **200**, and the hidden B trip's destination changed from `New B` to `Old B`.

**Fix:** call `need_all_scopes()` before restore, as the trash restore and full export already do. Apply the same rule to any operation that replaces office-wide data. Test refusal and confirm that no safety backup or data change occurs on refusal.

### 2. High — deleting a private photo removes its access protection

**Where:** `server/store.py:661`, especially the query at `:665`; `server/app.py:988` and `:993`.

**What happens:** the photo-to-category lookup ignores deleted photos and deleted trips. An image with no remaining active reference is then treated as a public image, such as a logo. It also receives a long public cache lifetime. Soft deletion should preserve privacy as well as bytes.

**Trigger:** someone knows a private photo's URL, the last trip/photo reference is soft-deleted, and that person requests the same URL using a signed-in account with no permissions. The URL must already be known; this does not prove that hashes can be guessed.

**Proof — reproduced over HTTP:** the no-permissions account got **403** before deletion and **200** after deletion, for the same synthetic PNG URL.

**Fix:** retain access classification for deleted references and their parent categories. Make public brand images an explicit class; do not infer “public” from the absence of an active trip reference. Use private/no-store responses for protected files and test deleted parents, deleted children and shared references.

### 3. High — restore reports complete recovery while photos remain missing

**Where:** `server/backup.py:484`, particularly `:492` and `:497`.

**What happens:** verification checks the photos in the backup folder. Restore brings back database rows but does not copy those photos into the live uploads folder. Its success flag describes the backup, rather than the restored installation.

**Trigger:** a live attachment is lost or quarantined, its backup is intact, and the dispatcher uses Restore.

**Proof — reproduced:** created a manifest-backed attachment and backup, removed only the synthetic live file, then restored. Output: `backup_verified True`, `restore_incomplete False`, **`live_file_exists False`**.

**Fix:** stage and checksum the required backup attachments, publish them atomically to the live folder, preserve existing good files, and verify the restored installation before reporting complete success. Locked files or a full disk must produce a clear incomplete/retry result. Do not overwrite a good live file with an unverified copy.

### 4. High — a saved change can be reported as a failed save

**Where:** `server/store.py:345`, `:348`; the separate log writer is at `:244`.

**What happens:** the journal and database are committed before the monthly audit file is appended. Failure of that last file write escapes from `commit()`. The HTTP layer can return 500 even though the change is durable. A dispatcher may retry and create a second trip/import.

**Trigger:** disk fills, antivirus locks the monthly audit file, or its write permission changes just after the database save succeeds.

**Proof — reproduced with fault injection:** made `_audit_file()` raise `OSError('synthetic disk full')`. `commit()` raised that error, but the saved driver remained in `state()` — **one durable row**.

**Fix:** treat the journal as the durable source, return the successful transaction ID, and separately report/retry the failed audit-file copy. Add stable request IDs for create/import operations so an uncertain response cannot create a second record. Do not describe the whole save as rolled back after it has committed.

### 5. High — an old Excel preview can import the same trip twice

**Where:** `server/app.py:1234`, `:1239`, `:1240`; `server/excel_io.py:456`, `:468`, `:481` and `:539`.

**What happens:** “new/duplicate” is decided during preview. `build_ops()` continues using those old decisions, even when passed a newer state. Each import creates new IDs. There is no transactional claim on the preview or uniqueness guarantee for an import key. A lost response before preview removal makes this worse.

**Trigger:** preview a file in two tabs before either saves; import both. A retry after finding 4 is another trigger.

**Proof — reproduced:** took one valid “new” row from `tests/fixtures/sample_sep26_synthetic.xlsx`; built and committed it twice, passing the latest state on the second call. Result: **2 imported trips, 1 distinct import key**. This sequential stale-preview case does not depend on a lucky race schedule.

**Fix:** recheck import keys inside the save transaction; give imports stable operation IDs and return the prior result on retry. Atomically reserve a preview, retain retry status on failure, and revalidate any remaining new lookups. Define how corrections and re-imports should behave rather than silently creating another trip.

### 6. High — incomplete automatic backups can replace the last complete one

**Where:** `server/backup.py:336`, `:342`, `:354`, `:452` and `:523`.

**What happens:** a set missing required photos is published, marked as the latest backup watermark and counted toward normal retention. Pruning counts automatic sets without checking completeness. The scheduler also sees an unchanged watermark after this incomplete run, so it need not retry merely because a temporarily inaccessible photo becomes available again.

**Trigger:** an old complete automatic backup exists; newer data refers to a photo that cannot be backed up; incomplete automatic sets eventually fill the retention count. The normal count is 200; the test used one to exercise the same rule quickly.

**Proof — reproduced:** with `keep_auto=1`, made a complete automatic set, then added a reference to a missing new photo and made another set. The first set initially verified **True**; afterward **the old set was removed and the new set was incomplete**.

**Fix:** protect a minimum number of verified complete sets independently of ordinary retention. Record “attempted” and “complete” watermarks separately, retry incomplete work, and prune only after a suitable replacement is verified. Still preserve incomplete sets when they are the only recoverable source of newer rows.

### 7. High — a password reset can race with login

**Where:** `server/auth.py:484`, `:490`, `:494`, `:528`; password reset at `:1274`.

**What happens:** login reads the account and verifies its password outside the transaction that creates the session. Inside that transaction it rereads only failure/lockout fields. If the password changes in between, the old password result can still create a new session after reset.

**Trigger:** an administrator resets a password while a login using the old password is being checked.

**Proof — reproduced by a controlled interleaving:** immediately after successful verification of the old hash, completed `reset_password()`, then let login continue. Output: **`old_password_accepted True`**, and the returned session was usable. This verifies the ordering defect; a timed stress race was not needed or claimed.

**Fix:** re-read the current hash/account version, status, permissions and password-change generation inside the session transaction. If they changed, reject or redo verification. Session creation must use the current account state.

### 8. High — reinstalling with “This PC only” may leave LAN access enabled

**Where:** `installer/trip-orders.iss:116`, especially `:120`; `server/app.py:64`.

**What happens:** the installer writes the loopback setting only when `config.json` does not exist. For an existing installation, ticking the option leaves an existing `host: 0.0.0.0` unchanged. Deleting this installer's firewall rule is not equivalent to changing the application's listener: other rules or local firewall settings can still allow access.

**Trigger:** install normally, then reinstall and tick “This PC only”, as the installer's own finished message suggests.

**Evidence:** **source verified; installer/network outcome not verified**. The existing-config branch performs no configuration update.

**Fix:** after a safety copy, update the existing host setting atomically when the user explicitly selects this option. Preserve unrelated settings and verify the resulting listener. Report failure instead of completing as though the restriction was applied. Test both fresh install and upgrade.

### 9. Medium — a tiny sparse workbook bypasses practical import limits

**Where:** `server/xlsx_read.py:69`, `:158`, `:166`; `server/excel_io.py:135`; generic dense row iteration at `server/xlsx_read.py:54`.

**What happens:** limits count XML cells and uncompressed bytes, but do not bound row coordinates. Parsing later loops through every row up to the largest coordinate. A tiny file can make a request spend a long time scanning empty space. Still larger coordinates are accepted, so the cost has no useful bound.

**Trigger:** choose a workbook with two real cells, one addressed as `A10000000`. No enormous upload is required.

**Proof — reproduced:** **789-byte ZIP, 2 cells, accepted maximum row 10,000,000; parsing took 9,083.9 ms**. Larger/hanging examples were deliberately not executed.

**Fix:** enforce valid Excel row/column bounds and a practical import-work budget. Iterate stored non-empty row numbers rather than the entire rectangle. Reject unsafe coordinates with a plain translated message; keep cell-count and ZIP limits too.

### 10. Medium — valid trips across the repeated Cairo hour are rejected or flagged wrongly

**Where:** `server/tripsvc.py:99`, `server/domain.py:140`, `:147`, `:279`.

**What happens:** duration uses real instants correctly, but validation and overlap checks compare local clock readings after removing their offsets. During the repeated hour, the clock reading can go backward while time moves forward.

**Trigger:** under the application's Cairo rules, start at `2026-10-29T23:50:00+03:00`, end at `2026-10-29T23:10:00+02:00`.

**Proof — reproduced:** `domain.duration()` returned **20 minutes**, but `check_trip()` rejected the trip as ending before it started. Two trips at 23:10–23:20 with the two different offsets, an hour apart in real time, were also incorrectly reported as overlapping.

**Fix:** validate ordering and overlaps using `domain.instant()` consistently. Retain offsets for ambiguous times and display a clear choice when manual entry cannot identify which hour is intended. Test both repeated and skipped hours. This is a comparison bug, not a claim that the bundled Egypt rules are legally current forever.

### 11. Medium — duplicate plates can be saved in one batch or concurrent requests

**Where:** `server/tripsvc.py:632`, `:635`; `server/app.py:1133`; `server/store.py:210`.

**What happens:** normalization checks uniqueness against existing rows before `commit()` takes its write lock. It neither checks earlier rows in the same proposed batch nor protects the whole check-and-save operation. The relevant database index is not unique.

**Trigger:** save two new vehicles with the same normalized plate in one batch; two users saving concurrently can reach the same gap. Driver/place keys have the same check pattern.

**Proof — reproduced:** one normalized batch successfully saved **v1 and v2, both `ABC 123`**.

**Fix:** check normalized keys against both the batch and current data while holding the transaction lock. Add database constraints where compatible with soft deletion. For distributed duplicates, define a merge/conflict rule so sync does not silently create inconsistent identities.

### 12. Medium — signing out leaves the data cache in an invalid state for the next user

**Where:** `js/app.js:28`, `js/data.js:22`, `:25`, `:28`, `:37`.

**What happens:** logout clears only `state`. Version, lookup index, insights, and pending loading state survive. The next login can request the same version and receive an “unchanged” response despite having no usable state. Old pending requests are also not invalidated by a new login generation.

**Trigger:** sign out, then sign in again without a business-data change, including on a shared dispatcher PC.

**Proof — reproduced in Node's VM using the real `data.js`, without a browser:** after state was cleared but version 7 remained, `D.load()` finished with **`state={version:7,unchanged:true}`, `trips=[]`**, while old insights remained. Native UI appearance and the pending-request race were **not verified**.

**Fix:** add `D.reset()` to clear state, version, indexes, insights and loading; increment the request generation so earlier responses are ignored. Call it on logout and account/scope change. Require an actual cached snapshot before sending a conditional-version request.

### 13. Medium — “set up as a new PC” does not archive the actual trips database

**Where:** `server/system.py:72`, especially `:78`; actual filename at `server/store.py:165`.

**What happens:** `_archive_copy()` moves names based on `triporders.db`, while this application stores business data in `trips.db`. Resetting a copied identity archives accounts/history/keys but leaves business rows behind; the next initialization bootstraps them into the new identity.

**Trigger:** use the existing “new PC” recovery choice after copied data is detected. Copying databases is prohibited by the project rules, but this recovery path explicitly exists to handle that mistake.

**Proof — reproduced in a synthetic installation:** after a `RESET_REQUESTED` restart, **the old trip was still visible and the new node was unconfigured**.

**Fix:** use the store's actual filename when archiving, including WAL/SHM companions. Include recovery metadata needed to detect and finish an interrupted archive. Test a partially completed move and never silently bootstrap the old office's trips as a new office.

### 14. Medium — the current installer cannot meet the non-admin requirement

**Where:** `installer/trip-orders.iss:39`, `:44`, `:83`, `:95`, `:98`; `server/to_main.py:22`.

**What happens:** installation requires administrator rights, uses Program Files and ProgramData, and runs ACL/firewall commands. Merely hiding the console or replacing the browser window does not change those requirements.

**Trigger:** a dispatcher without elevation tries to install the application.

**Evidence:** **source verified; clean standard-user install not verified**. `PrivilegesRequired=admin` is explicit.

**Fix:** provide a per-user, loopback-only desktop install: `PrivilegesRequired=lowest`, program under LocalAppData Programs, data under LocalAppData TripOrders, and per-user startup shortcuts. Keep an explicitly separate assisted LAN-host installation if needed. Migration of existing ProgramData records must be deliberate and must not silently start an empty office.

### 15. Medium — large offices repeatedly pay for the entire history during ordinary loading

**Where:** `server/store.py:298`; `server/tripsvc.py:437`; `server/reports.py:25`; `server/app.py:900`.

**What happens:** state reads all active rows; insights repeatedly construct destination-history lists and medians; each of the five report builders independently recomputes insights for the complete state, even when reporting one month. This makes a normal reports request take many seconds.

**Trigger:** 10,000 recorded trips to a frequently repeated destination. That is a realistic pattern for office shuttles; the test used 100 vehicles/drivers with non-overlapping trips to avoid artificially huge overlap results.

**Proof:** HTTP insights **4.43 s**, HTTP reports **22.18 s**. A separate same-process before/after experiment reused the identical enriched rows for the five builders: **17,991.48 → 3,658.94 ms**, **79.7% faster**, with identical report outputs. No source file was edited.

**Fix:** compute enrichment/insights once per consistent snapshot and reuse it. Then replace per-trip full destination scans with a grouped, sorted summary that correctly excludes the current trip. Add scoped/paged state retrieval and a recent-period first view. See performance details below.

## Other edge cases and safeguards checked

| Case | Result and remaining limit |
| --- | --- |
| Empty office | Server starts and returns an empty state. Normal first start creates the built-in administrator; it does not seed training trips. |
| Crash between journal save and table commit | **Passed a forced-process-exit test.** Exited with code 23 immediately after `append_local()`. Reopening recovered the one saved synthetic driver. This verifies that boundary, not every possible power failure. |
| Second instance, same data folder | **Passed cross-process lock test:** the second `lock_data()` returned None. Current second launch opens a browser URL; it does not focus an owned app window. |
| Web port occupied by an unrelated listener | **Reproduced:** startup printed the occupied-port instructions and exited with code **0**. With no visible console this is a poor failure signal. Use a native startup error and nonzero exit; never assume the listener is Trip Orders. |
| Sync port occupied | `sync.py:378` catches the bind error and continues with outgoing sync. Source verified; full multi-PC behavior was not exercised. Surface this to the dispatcher instead of relying on a log. |
| Disk full / locked files | Audit-file fault injection found finding 4. Existing backup failure/space tests passed. A real disk was not filled and antivirus was not manipulated. |
| Backups, failed copying and checksums | Selected existing tests for consistency, failed snapshots, attachments, corruption, space checks and restoration passed. Findings 3 and 6 are gaps beyond those tests. |
| Arabic/RTL | Source uses bundled fonts, logical CSS properties and UTF-8; synthetic performance data included Arabic destinations. `content_disposition()` supports UTF-8 filenames. Real Arabic display, selection, printing and Office rendering were not visually verified. |
| Excel formulas | The writer uses a dedicated `Formula` type; ordinary text is written as shared strings. No claim of text-to-formula injection was found in that path. Standard XLSX tests passed. |
| Training | `_config()` gives training its own data and backup directories, disables the gateway and sign-in, and assigns separate ports. Measured first launch seeded **41 synthetic trips**. A full training child-close/reset UI journey was not run. Port collision and real locked-folder recovery need acceptance checks. |
| Offline | Local HTTP requests, native XLSX processing, and server startup worked without requiring a gateway in the isolated config. This does not prove the complete desktop installer works on a disconnected clean PC. |
| Default access policy | `sign_in=False`, LAN listening and code-free automatic PC enrollment are explicit owner decisions in the source, not accidental bypasses invented by this review. They mean a reachable trusted-network client has broad authority by design. The first-pilot deployment must explicitly choose the intended trust boundary; the desktop wrapper does not secure the LAN. |
| Sync resilience | Signed history, pinned TLS, retrying folds and atomic attachment publication exist. Multi-PC fault/revocation stress was not rerun. `TLSServer` also lacks the main web server's connection semaphore; overload resistance remains unverified. |

## Part 2 — measured performance

### Method

Timing used `time.perf_counter()` in Python, real loopback HTTP requests using urllib, synthetic SQLite data saved through the current journal/store code, and static file byte counts. No browser, GUI timing, Lighthouse or render tracing was used. OS caches were not flushed: **these are process-start measurements with warm filesystem caches, not cold boot claims**.

HTTP runs used a logged-in synthetic administrator. The capacity dataset had 100 vehicles/drivers represented by IDs, increasing odometers, distinct non-overlapping dates per vehicle, and a common Arabic destination. Some master records were intentionally absent; timings isolate history size rather than reproduce a named customer office. Small samples show scale, not production p95 guarantees. Do not add parallel HTTP times as if they were sequential screen time.

### Startup

| Scenario | Process launch → first successful auth-status response | App import stage | Demo/setup stage | Startup backup |
| --- | ---: | ---: | ---: | ---: |
| Fresh normal office, no sign-in | **1,203.23 ms** | 851.68 ms | 244.28 ms | Not needed: no trips |
| Same normal office restarted | **803.21 ms** | 724.48 ms | 0.21 ms | Not needed: no trips |
| First training launch, 41 trips | **2,809.47 ms** | 857.02 ms | 1,836.28 ms | 37.92 ms |
| Training restart | **840.15 ms** | 713.21 ms | 0.36 ms | 39.80 ms |
| Office with 10,000 trips restarted | **1,186.87 ms** | 814.68 ms | 0.23 ms | 284.28 ms |

App import includes runtime/database initialization; it is not pure Python parsing. Separate `python -B -X importtime -c "import app;app.SYSTEM.close()"`, with an isolated `TO_HOME`, reported app cumulative time **846.55 ms**; significant imported subtrees included auth **156.50 ms**, excel_io **92.85 ms**, domain **44.26 ms**, and http.server **41.10 ms**. These figures overlap and must not be summed.

**Changes that help:**

- Remove the fixed browser-opening timer at `app.py:1695`: it adds **800 ms** before the current browser launch, beyond server work. An owned window should load on a readiness signal. The saving is removal of that fixed wait; actual first-paint saving is unverified.
- Initialize Excel/Word import modules only when needed after checking import relationships. The excel_io subtree offers an upper-bound opportunity of about **93 ms** in this run; shared imports mean the full amount is not guaranteed.
- Training seed cost is **1.84 s** once. Show a clear starting screen immediately; keep the transactional, one-time seed. Do not seed real offices to hide an empty screen.
- Startup backup cost was **284 ms** at 10,000 trips, without a photo collection or second drive. If needed, present the window while the backup runs and show its state; preserve the required pre-upgrade/pre-restore safety backups. Moving this work cannot be advertised as reducing actual backup time.

### Requests and growing history

| Operation | 1,000 trips | 10,000 trips | Notes |
| --- | ---: | ---: | --- |
| `Store.state()` | **7.72 ms** | **89.01 ms** | Median of 3 |
| HTTP `/api/state` | **20.34 ms** | **110.98 ms** | Median of 3; response 281,692 / 2,853,892 bytes |
| HTTP `/api/insights` | **79.45 ms** | **4,431.23 ms** | Median of 2 |
| HTTP `/api/reports?ym=2026-01` | **381.84 ms** | **22,184.85 ms** | Median of 2; all five report sections for administrator |
| HTTP monthly Excel export | **34.89 ms** | **165.77 ms** | January subset; 45,360 / 124,584-byte workbooks |
| Startup-style backup creation | **54.56 ms** | **274.47 ms** | Median of 2; no attachment collection/USB/network disk |

The empty-state HTTP median was **11.73 ms**. A separately repeated, thread-free insights call for 10,000 trips took **3,637.36 ms**, showing run-to-run variation but the same bottleneck.

**Measured improvement experiment:** in memory only, cached `reports.enrich()` within one request/snapshot and let the same five report builders reuse it. Same data, same outputs: **17,991.48 ms before, 3,658.94 ms after, 79.7% reduction**. This is a verified prototype result; the app has not been changed. A production request-local reuse should remove four of five enrichment passes. A global cache would additionally need data-version, scope, parameter and time-sensitive invalidation.

**Further expected gains, not implemented or verified:**

- Replace `insights()`'s per-trip destination-history list and median work with one grouped/sorted calculation, preserving leave-one-out behavior. This removes quadratic work for common destinations. A subsecond 10,000-trip target is reasonable to test, not a result claimed here.
- Compute requested-page/scoped state in SQL and return incremental changes. Filtering currently happens after all entities are loaded. The Trips view already initially draws only **40 rows**, then adds 200; its first DOM table is therefore not 10,000 rows. It still loads and filters the full history.
- API JSON currently gets no gzip from `send()`, even when accepted. Compressing the measured synthetic state offline at level 6 reduced **2,853,833 → 164,902 bytes**, a **94.2% byte reduction**. That saves transfer/parse pressure, not the 3–4 seconds spent computing insights. CPU and latency after implementing compression are unverified.
- Reject or sparsely iterate the pathological workbook in finding 9; the measured avoidable empty-row scan was **9.08 s**.

### Front-end load

| Eager resource group from index.html | Count | Raw bytes | Separately gzip-compressed bytes |
| --- | ---: | ---: | ---: |
| JavaScript | **33** | **784,847** | **220,831** |
| Stylesheets | **6** | **80,370** | **17,151** |
| All bundled WOFF2 fonts | **34** | **748,616** | Already compressed; not an initial-download total |

The HTML plus listed JS/CSS transfers about **238,748 bytes** with current gzip support, before headers and fonts. A loopback request for the compressed index took **1.44 ms** median. Largest scripts: Arabic translations **185,730 raw / 45,903 gzip**, English translations **141,567 / 40,714**, list views **59,517 / 14,810**, trip views **51,632 / 13,348**.

`index.html:9` begins six stylesheets. `:15` loads boot.js synchronously in the head. The remaining scripts are classic ordered scripts in the body; no defer/module loading is used. The page waits for DOMContentLoaded, auth status and the state/insights pair before starting the shell. These are source facts; a browser waterfall and paint cost were **not measured**.

Fonts are local and use `font-display:swap`, so there is no external Google Fonts dependency. The 748,616-byte figure is the installed catalogue, **not proof all fonts download at startup**. Plex is the default; alternative families contribute Cairo **115,848**, Kufi **224,496**, Tajawal **57,048** bytes on disk. Fonts load according to use and Unicode ranges.

**Specific front-end changes:** load only the selected translation initially (roughly **40.7–45.9 KB gzip** deferred), load report/import/help views on demand, preserve dependency order when deferring scripts, and add ETag/304 or versioned caching for assets. Current static serving says no-cache but does not implement validator responses. Expected byte savings are calculable; first-paint, layout, memory, scrolling and Arabic shaping improvements are **not verified** without a browser. Measure them later on the exact desktop candidate.

## Part 3 — desktop delivery

### First, correct one premise

Current `tools/build_windows.py:37` uses **`--windows-console-mode=attach`**. According to Nuitka, attach reuses a console when launched from one and does not create a new console when launched normally. Thus this source does **not** establish that the current Explorer-launched build opens a console. An older customer executable may differ. The browser launch and misleading “Keep this window open” text are present. [Nuitka console modes](https://nuitka.net/user-documentation/user-manual.html#windows-console-window-control).

No executable/installer candidate was available at the inspected build/dist paths, and none was built or launched. There are **no measured wrapper startup or executable-size comparisons** in this report.

### Comparison for this application

| Choice | Opening and size | Build/runtime implications | Window ownership, close and second launch | Existing UI features |
| --- | --- | --- | --- | --- |
| Edge/Chrome `--app=http://127.0.0.1:PORT/` | No bundled browser payload if a compatible browser is already installed. Can reuse a warm browser, but profile/proxy/policy costs are unmeasured. | Smallest code/build change; keeps current Nuitka package. Requires finding a browser and a writable dedicated profile. A managed PC can block app mode or lack the assumed browser. | Looks like an app window, but browser process lifetime is not the window lifetime. A returned launcher PID is not a reliable close signal. Native focus/IPC/window identification must be added; data lock alone is insufficient. | Browser handles downloads, uploads and printing. Dedicated profile avoids mixing normal browsing cookies. Arabic uses the existing Chromium UI. No actual UI tests run. |
| **pywebview + WebView2 — recommended** | Keeps Python and shares the WebView2 engine. Adds Python/.NET bridge and loader files; increment in installed/installer size and cold opening time must be measured. | Official docs allow Nuitka. Windows uses pythonnet/.NET plus WebView2; dynamic modules and DLL packaging make this a real compatibility gate. Explicitly select edgechromium, with no old-IE fallback. | Owned native window and close events fit the existing server. Add local IPC to restore/focus it on second launch. Minimize keeps server running; Close stops it through one shutdown path. | Enable downloads explicitly. Keep requests/cookies on the existing HTTP origin. Test blob downloads, ordinary download links, target=_blank, file picking and printing. Arabic/RTL is rendered by Chromium; actual output still needs testing. |
| Tauri + Nuitka Python sidecar | Small webview host, **plus the existing Python distribution**. It does not remove Python's cost. Same WebView2 prerequisite. Exact package bytes unmeasured. | Adds Rust, sidecar lifecycle, permissions and another packaging system to a Python/Inno project. More build/upgrade work now. | Strong native lifecycle and a single-instance plugin, but focus must be implemented in its callback. Need orderly sidecar shutdown and handling of orphan processes. | Download, popup and print behavior need wrapper-specific integration; same HTML/RTL can be reused. |
| Electron + Python sidecar | Bundles Chromium and Node alongside Python, so it adds an entire engine rather than sharing one. Largest structural payload here; no exact MB or timing claimed. | No WebView2 prerequisite. Adds Node/Electron packaging, engine updates and sidecar coordination. No need to translate the existing backend. | Good window, tray, single-instance and print/download APIs. More total runtime to own and update for a single-office tool. | Existing UI reusable, but keep Node access disabled in the page, use isolation, and allowlist navigation. |
| Small native WinForms/C++ WebView2 host + Python sidecar | Potentially lean host with shared engine; Python still present. No speed advantage proven. | Avoids Python GUI bridging but adds a second implementation/toolchain. A valid later fallback if the pywebview packaging proof fails, not a simpler first delivery. | Full native control; IPC and backend lifecycle still need implementation. | Direct native download/print APIs available, but integration and testing remain work. |

Basis: [pywebview freezing](https://pywebview.flowrl.com/guide/freezing), [Windows engine requirements](https://pywebview.flowrl.com/guide/web_engine.html), [pywebview installation source](https://github.com/r0x0r/pywebview/blob/master/docs/guide/installation.md?plain=1), [Tauri sidecars](https://v2.tauri.app/develop/sidecar/), [Tauri single instance](https://v2.tauri.app/plugin/single-instance/), [Electron's embedded engines](https://www.electronjs.org/docs/latest/).

**WebView2 distribution:** it is included with Windows 11 and present on most active Windows 10 machines, but detection is still required. For an offline customer, include the Evergreen **standalone** runtime installer, not just the small downloader. Unelevated runtime installation supports per-user use. A Fixed Version runtime adds **over 250 MB** of runtime binaries and transfers patch responsibility to the app, so it is not the preferred default. The offline installer's exact release size must be measured when chosen. [Microsoft distribution guide](https://learn.microsoft.com/microsoft-edge/webview2/concepts/distribution), [Windows rollout](https://blogs.windows.com/msedgedev/2022/12/14/delivering-microsoft-edge-webview2-runtime-to-managed-windows-10-devices/).

**Nuitka risk is concrete:** pywebview's guide supports Nuitka, but an upstream report describes a pywebview 6.2.1 packaging failure involving `webview.platforms.win32`. This is evidence for pinning and testing the exact dependency set, not proof every version fails. The existing workflow's broad `nuitka>=2.4,<3` range is not a tested pywebview compatibility specification. [Upstream packaging report](https://github.com/r0x0r/pywebview/issues/1817).

**Antivirus:** none of these choices guarantees freedom from false positives. Retain an installed standalone directory rather than introduce one-file extraction solely for appearance; sign the executable and installer, keep stable product metadata, avoid packers, and test the exact release with Defender/SmartScreen. No antivirus verdict was obtained. Signing improves publisher identification; it does not guarantee a clean verdict.

### Recommended change contract — specification only, not implemented

1. **`server/to_main.py:main()` and `home_dir()`:** route ordinary launches to a new `server/desktop.py:run_desktop()`. Keep the `tool` route independent of the GUI. For a new per-user desktop install, use `%LOCALAPPDATA%\TripOrders` for writable data/config and `%LOCALAPPDATA%\Programs\TripOrders` for program files. Respect `TO_HOME` for portable/testing use. Detect a prior ProgramData installation and provide an explicit, backed-up migration; never quietly open an empty replacement office.

2. **`server/app.py`:** extract module-level initialization into `initialize_runtime()`, HTTP/service start into `start_server()`, and the existing finally cleanup into idempotent `shutdown_server()`. Keep a compatibility `main()` for source/maintenance use. Desktop mode must not call `webbrowser.open()` or the 0.8-second timer. Start HTTP work on a worker thread; signal readiness only after the correct server can answer. Route initialization/bind errors to a native error dialog and the UTF-8 log, with a nonzero failure exit.

3. **New `server/desktop.py`:** create the native window on the main thread, show a simple local “Starting Trip Orders…” screen, and then load the verified `http://127.0.0.1:PORT/` origin when ready. Use `webview.start(gui='edgechromium')`. Keep the current authenticated HTTP API; do not move business permissions into a JavaScript bridge. Keep WebView user data in a writable per-user folder. Use persistent storage deliberately so the app's language/theme preferences survive; clear account data correctly on logout.

4. **Single instance before expensive initialization:** implement `desktop.py:notify_existing_instance()` and `listen_for_activation()` using a Windows named mutex/pipe restricted to the current Windows user, keyed by canonical data-directory identity. Second launch sends “activate”, restores/minimizes appropriately and focuses the existing window, then exits. Retain `system.lock_data()` as the final data safety lock, including for maintenance tools. Do not add an unauthenticated HTTP shutdown/focus endpoint. If the owner instance is still starting, wait with a timeout; if another app owns the web port, show an error instead of loading it.

5. **Clear lifecycle:** default Close means **stop this installation's server and close the app**. Explain once, in Arabic/English, that office PCs lose access. Minimize leaves a visible taskbar item and keeps serving. Route close through `request_close()` and `shutdown_server()`: stop accepting new changes, drain active saves, stop the training child and background workers, flush activity, close HTTP sockets/database connections and release locks. Never force-kill a normal save. Replace the installer’s invisible `--background` autostart with a visible minimized window (`--minimized`), so no tray package is needed for the first version. If a later “keep in tray” option is accepted, it must have obvious Open and Stop commands; it must never silently leave a server behind.

6. **Navigation, downloads and print:** enable `webview.settings['ALLOW_DOWNLOADS']=True`; keep file URLs disabled. Send external WhatsApp/help links to the normal browser without exposing a privileged bridge there. Handle same-origin attachment popups inside the app or as authenticated downloads: the current `target=_blank` photo links cannot assume the external browser has the WebView session cookie. Preserve `/api/excel/export`, `/api/xlsx`, `/api/word/form`, `/api/fleet/roster.xlsx` and blob-based downloads. Keep `js/views/print.js:TO.printHTML()` and test `window.print()`/afterprint with the chosen runtime; if required, supply a narrowly scoped native print action. [pywebview API and download settings](https://pywebview.flowrl.com/api/).

7. **Config keys:** add `desktop_enabled=true`, `desktop_renderer='edgechromium'`, `desktop_close_action='exit'`, `desktop_start_minimized=false`, `desktop_width=1280`, `desktop_height=850`. Derive the WebView profile directory from the home directory, or define `desktop_profile_dir` if configuration is necessary. Set `open_browser=false` for desktop mode. A new non-admin, local-only install uses `host='127.0.0.1'`; retain an explicitly chosen LAN-host configuration during migration. Keep the HTTP port stable because preferences and sessions depend on the origin. A port fallback must update the origin/profile design deliberately, not silently.

8. **`tools/build_windows.py:compile_program()` and build workflow:** keep x64 standalone Nuitka and the current Python 3.11 build baseline initially. Change GUI console mode to `disable`; keep maintenance output usable through a console helper or documented file logging. Pin a pywebview/pythonnet/clr-loader/Nuitka set after a successful packaging proof. Explicitly include the dynamically selected `webview.platforms.winforms`, `webview.platforms.edgechromium` and their actual dependencies. Produce a Nuitka compilation report; include the selected WebView2 loader/managed DLLs and Python.Runtime files using supported package/DLL configuration. If required, add `tools/webview.nuitka-package.config.yml` and pass `--user-package-configuration-file`. Do not assume a blanket data-file flag collects DLL dependencies. Keep unnecessary Qt/CEF packages out of the clean build environment. [Nuitka package configuration](https://nuitka.github.io/Nuitka-website/doc/nuitka-package-config.html).

9. **`installer/trip-orders.iss`:** add the per-user mode described in finding 14, repair the existing “This PC only” update path, detect the WebView2 runtime, and include/run the offline x64 standalone redistributable only when needed. Check its exit status and show a plain error. Do not run admin-only firewall/ProgramData ACL commands in per-user mode. Use per-user startup entries. Preserve data on update/uninstall. For upgrades, request graceful application shutdown and verify it; the existing global `taskkill /F /IM TripOrders.exe` can terminate unrelated instances and should not be the normal upgrade strategy.

10. **Performance gate:** apply the request-local report reuse and address initial insights loading before calling the app fast. Showing a native window immediately improves feedback, but it does not make a 22-second report usable. Measure launch-to-window and launch-to-usable-screen separately on the exact compiled artifact, together with installer and installed-folder sizes.

### Acceptance tests for the chosen desktop option

| Test | Required result |
| --- | --- |
| Clean Windows 10 and 11 standard user, disconnected network | Installs and opens without Python, build tools or elevation; missing WebView2 is handled from the offline package. Verify required .NET runtime too. |
| Explorer launch, taskbar launch, login autostart | One owned branded window, no console flash, correct minimized behavior, clear error if initialization fails. |
| Rapid second/third launch, including while first starts | Exactly one server/data writer and one main window; existing window comes forward. Unrelated port owner is never shown as Trip Orders. |
| Close with a save in progress; minimize; forced crash; reopen | Normal Close drains or safely rejects work and stops serving; minimize keeps serving; crash recovery retains acknowledged changes; no hidden orphan. |
| 0 / 1,000 / 10,000 trips, attachment-heavy backup | Record launch-to-first-window, usable-screen, reports, memory, installer size and installed size. Compare the same data and cache conditions. |
| Arabic and English at 100/150/200% display scaling | RTL forms, mixed Arabic/Latin plates/numbers, dialogs, tables and focus order remain readable. |
| XLSX/CSV import, all workbook/Word exports, blob downloads and private photo popup | File picker and Save As work; Arabic paths work; exports open in the office's Excel; private photos do not lose authentication. |
| A4 trip order, report printing, Print to PDF | Correct Arabic layout, pagination and afterprint cleanup; cancel printing also works. |
| Logout/login as a differently scoped person | Cache is cleared; previous records/photos are not shown; denied operations stay denied. |
| Full disk, locked uploads, missing runtime, unavailable USB, update with app running | Clear translated failure; no false “saved”/“restored” result; old usable data and backups remain available. |
| Release signing and antivirus check | Test the exact executable and installer; record actual verdicts, without promising zero false positives. |

## Verification commands and result ledger

All custom probes were supplied through PowerShell here-strings to **`python -B -`**, with imports pointed only at the temporary source copy and with isolated homes/data. JavaScript cache verification used **`node`** with the copied `js/data.js` in a VM and stubbed HTTP responses; it did not automate or open a browser. The finding sections give the exact operations, injected boundary and observed outputs needed to repeat each probe. Temporary test material is disposable; the durable deliverable is this report.

Temporary source/data remain in `I:\herdr\tasks\tmp\yt-review-oyhvolat` and the initial harness folder `I:\herdr\tasks\tmp\to-excel-19400-d47_lr2s`. Automatic approval review rejected their recursive removal with “blocked by policy”; no alternate deletion method was attempted. The known orphan test server was stopped before cleanup was requested.

Existing test selection:

`python -B -m unittest test_xlsx test_time test_backup.CreateTest test_backup.CoherenceTest test_backup.AttachmentTest test_backup.WatermarkTest test_backup.RestoreTest test_excel_io`

The supplied Windows test harness initially failed cleanup because it uses Unix-only `signal.SIGKILL`; two time-zone tests also require Unix-only `time.tzset()`. The first invocation additionally named a nonexistent `test_training` module; that was a review command error, not an application defect. The final run used the same selected suite with only an **in-memory** harness adapter to kill its own Popen children and skipped the two tzset-dependent methods. No test file was edited.

**Final result: 55 tests run, 0 failures, 0 errors, 4 skips, 22.470 seconds.** The other two skips were retained from the suite; skipped paths are not treated as verified. The failed initial cleanup's isolated server was identified by its temporary data directory and local port and stopped. Other processes were not stopped.

Additional observed outputs:

- Audit-file fault: exception returned, durable rows **1**.
- Scoped restore: hidden trips **0**, restore HTTP **200**, hidden destination reverted.
- Private photo: **403 before deletion → 200 afterward**.
- Stale import: imported rows **2**, unique import keys **1**.
- Backup photo restore: backup verified **True**, incomplete **False**, live file exists **False**.
- Retention: old complete set removed; new set incomplete.
- Password-reset interleaving: old password accepted **True**.
- Duplicate plates: **two** active rows with the same plate in one batch.
- Sparse XLSX: **789 bytes, 2 cells, 10 million addressed rows, 9,083.9 ms**.
- Repeated hour: true duration **20 minutes**, validation rejected, non-overlapping real trips flagged as overlapping.
- Crash boundary: child exit **23**, recovered rows **1**.
- Second lock: refused **True**.
- Occupied web port: explanatory output, process exit **0**.
- New-device reset: old trips still visible **1**.
- Request-local report experiment: **17,991.48 → 3,658.94 ms**, outputs identical.

## Doubts / not verified

- No built executable, Nuitka/pywebview integration, installer upgrade, signature, actual antivirus scan, WebView2 absence, or clean non-admin Windows installation was tested. pywebview is not installed in the review interpreter.
- No browser rendering, first paint, scrolling, native focus, tray, printing, GUI downloads, camera or visual RTL test was performed, as requested. Wrapper timing/size claims require the acceptance build.
- No real disk was filled, antivirus rule changed, power removed, or customer data used. Fault injection proves particular error boundaries, not every storage-device failure.
- No full multi-PC/offline mailbox acceptance journey, Python 3.11 Windows release test suite, independent second review, or complete repository-wide security audit is claimed.
- The selected tests pass around several reproduced bugs; passing them does not establish pilot readiness. The first-sale recovery and customer-device gates in `LAUNCH_SCOPE.md` remain open.
