[CmdletBinding()]
param([int]$StressSeconds = 20, [switch]$SkipLayout)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'deck-ui.ps1')
$script:results = [Collections.Generic.List[object]]::new()
$script:layoutRows = [Collections.Generic.List[object]]::new()
$script:perfRows = [Collections.Generic.List[object]]::new()
$script:repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$script:bugFile = Join-Path $script:repo 'tasks/deck-r3-codex-bugs.md'
$script:started = Get-Date
function J($Value) { ConvertTo-Json -InputObject $Value -Depth 20 -Compress }
function Assert($Condition, [string]$Reason) { if (-not $Condition) { throw $Reason } }
function JS([string]$Code) {
    if ($Code.StartsWith('(await ')) { $Code = '(async()=>('+$Code+'))()' }
    $r=Send-Cdp 'Runtime.evaluate' @{expression=$Code;returnByValue=$true;awaitPromise=$true} -TimeoutMs 150000
    if($r.exceptionDetails){throw "JS error: $($r.exceptionDetails.exception.description ?? $r.exceptionDetails.text)"}
    $r.result.value
}
function Wait-Js([string]$Code, [int]$Seconds = 20) {
    $until = [DateTime]::UtcNow.AddSeconds($Seconds)
    do { if (JS ('(async()=>('+$Code+'))()')) { return }; Start-Sleep -Milliseconds 200 } while ([DateTime]::UtcNow -lt $until)
    throw "Timed out: $Code; UI: $(JS 'document.querySelector("#toasts-loud").innerText')"
}
function Check([string]$Name, [scriptblock]$Body) {
    try { $detail = (& $Body | Out-String).Trim(); $script:results.Add(@{name=$Name;status='PASS';detail=$detail}); Write-Output "PASS $Name $detail" }
    catch {
        $reason = $_.Exception.Message
        $script:results.Add(@{name=$Name;status='FAIL';detail=$reason}); Write-Output "FAIL $Name : $reason"
        $existing = if (Test-Path -LiteralPath $script:bugFile) { Get-Content -LiteralPath $script:bugFile -Raw } else { '' }
        if (-not $existing.Contains("Observed: $reason`n")) {
            $number = [regex]::Matches($existing, '(?m)^## B\d+').Count + 1
            Add-Content -LiteralPath $script:bugFile -Value "`n## B$number`nFeature: $Name`nSeverity: pending triage (verification failure)`nSteps: pwsh -NoProfile -File app/tests/verify-ui.ps1; check named above.`nExpected: behavioral assertion passes against isolated lab.`nObserved: $reason`nTime: $(Get-Date -Format o)`n"
        }
        try { JS 'document.querySelectorAll("dialog[open]").forEach(d=>d.close()); document.querySelector("#drawer-close").click()' | Out-Null } catch { Write-Warning "Dialog cleanup: $_" }
    }
}
function Api([string]$Route, $Data = $null) {
    $path = J $Route
    $expression = if ($null -eq $Data) { "api.get($path)" } else { "api.post($path,$(J $Data))" }
    JS "(async()=>{const {api}=await import('./api.js'); const r=await fetch($path+($path.includes('?')?'&':'?')+location.search.slice(1),{method:$(if($null -eq $Data){'"GET"'}else{'"POST"'}),headers:{'Content-Type':'application/json'},body:$(if($null -eq $Data){'undefined'}else{J (J $Data)})}); const b=await r.json(); if(!r.ok)throw new Error(r.status+' '+JSON.stringify(b));return b})()"
}
function Click([string]$Selector) { JS "document.querySelector($(J $Selector)).click()" | Out-Null }
function Button([string]$Text, [string]$Within = 'dialog[open]') {
    JS "(()=>{const b=[...document.querySelectorAll($(J ($Within+' button')))].find(b=>b.textContent.trim()===$(J $Text));if(!b)throw Error('Button missing: '+$(J $Text));b.click()})()" | Out-Null
}
function Fill([string]$Selector,[string]$Value) { JS "(()=>{const e=document.querySelector($(J $Selector));e.value=$(J $Value);e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}))})()" | Out-Null }
function Chord([string]$Key) {
    JS "document.activeElement?.blur();document.dispatchEvent(new KeyboardEvent('keydown',{key:'b',ctrlKey:true,bubbles:true}));document.dispatchEvent(new KeyboardEvent('keydown',{key:$(J $Key),bubbles:true}))" | Out-Null
}
function Command([string]$Label) {
    Click '#btn-palette'
    Fill 'dialog [aria-label="Command"]' $Label
    JS "(()=>{const n=[...document.querySelectorAll('#pal-list li')].find(n=>n.firstElementChild.textContent===$(J $Label));if(!n)throw Error('Missing command '+$(J $Label));n.dispatchEvent(new MouseEvent('mousedown',{bubbles:true}))})()" | Out-Null
}
function Card([string]$Tail='') { '[data-key="'+$script:key.Replace('\','\\')+'"]'+$Tail }
function Screen([string]$Pane=$script:pane) { [string](Invoke-Lab @('pane','read',$Pane,'--source','visible','--format','text')) }
function State { Api '/api/state' }
function Select-Pane([string]$Pane) {
    $script:pane=$Pane; $script:key=(State).session+'|'+$Pane
    JS "(async()=>{const {S}=await import('./app.js');S.current=$(J $script:key)})()" | Out-Null
}
function Wait-Jobs {
    $until=[DateTime]::UtcNow.AddSeconds(125)
    do {
        $jobs=(Api '/api/jobs').jobs
        $active=@($jobs | Where-Object { $_.state -in @('running','queued') -or $_.status -in @('running','queued') })
        if (-not $active.Count) { return (J $jobs) }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $until)
    throw 'Async job timeout'
}
try {
    Start-DeckUi -Lab -EdgeArgs @('--no-proxy-server')
    $script:ownedPids=@($script:ui.Herdr.Id,$script:ui.Deck.Id,$script:ui.Edge.Id) | Where-Object { $_ }
    Set-DeckSize 1440 900
    Open-DeckPage '?theme=dark'
    JS 'localStorage.clear()' | Out-Null
    Open-DeckPage '?theme=dark'
    $script:initialWs=@((Invoke-Lab @('workspace','list')).result.workspaces.workspace_id)
    $created=Invoke-Lab @('workspace','create','--cwd',$script:repo,'--label','Deck R3 verification','--focus')
    $script:ws=$created.result.workspace.workspace_id
    Select-Pane $created.result.root_pane.pane_id
    Check 'Live grid and SSE' {
        Wait-Js "!!document.querySelector($(J (Card)))" 30
        Assert ((State).panes.pane_id -contains $script:pane) 'Created pane missing from state'
        "pane=$script:pane"
    }
    Check 'Prompt box and dispatch ledger' {
        Fill (Card ' textarea.pin') "Write-Output ('UI-PROMPT-' + 'OK')"
        Click (Card ' button.send')
        Invoke-Lab @('pane','wait-output',$script:pane,'--match','UI-PROMPT-OK','--timeout','10000') | Out-Null
        $ledger=Api ('/api/dispatches?key='+[Uri]::EscapeDataString($script:key))
        Assert ($ledger.dispatches.Count -gt 0) 'No dispatch record'
        Assert ($ledger.dispatches[0].state -eq 'sent') (J $ledger)
        J $ledger.dispatches[0]
    }
    Check 'Paste without Enter then key button' {
        Click (Card ' [aria-label="Paste to pane (Ctrl+Shift+V)"]')
        Fill 'dialog textarea' "Write-Output ('UI-PASTE-' + 'OK')"
        Button 'Send text'
        Start-Sleep -Milliseconds 1200
        Assert (-not ((Screen) -match '(?m)^UI-PASTE-OK\s*$')) 'Paste executed before Enter'
        Click (Card ' [aria-label="Send key Enter"]')
        Invoke-Lab @('pane','wait-output',$script:pane,'--match','UI-PASTE-OK','--timeout','10000') | Out-Null
        'Computed output marker reached after Enter'
    }
    Check 'Copy screen clipboard read-back' {
        Send-Cdp 'Emulation.setFocusEmulationEnabled' @{enabled=$true} | Out-Null
        Send-Cdp 'Browser.grantPermissions' @{origin=$script:ui.Origin;permissions=@('clipboardReadWrite','clipboardSanitizedWrite')} | Out-Null
        Wait-Js "document.querySelector($(J (Card ' .screen'))).innerText.includes('UI-PASTE-OK')"
        Click (Card ' [aria-label="Copy screen (Ctrl+Shift+C)"]')
        $clip=JS 'navigator.clipboard.readText()'
        Assert ($clip.Contains('UI-PASTE-OK')) 'Clipboard lacks computed output marker'
        "Clipboard chars=$($clip.Length)"
    }
    Check 'Rename pane dialog and read-back' {
        Click (Card ' [aria-label="Rename (Ctrl+B Shift+P)"]')
        Fill 'dialog input' 'UI renamed'
        Button 'Save'
        Wait-Js "document.querySelector($(J (Card ' .ttl'))).textContent==='UI renamed'"
        Assert ((Invoke-Lab @('pane','get',$script:pane)).result.pane.label -eq 'UI renamed') 'CLI label mismatch'
    }
    Check 'Zoom and grid columns' {
        Click (Card ' .ttl'); Wait-Js "(await import('./app.js')).S.zoom===$(J $script:key)"
        Click (Card ' .ttl'); Click '[data-cols="3"]'
        Assert (JS '(await import("./app.js")).S.cols===3') 'Three-column setting missing'
        Click '[data-cols="2"]'
    }
    Check 'Split right and close confirmation' {
        $before=@((State).panes.pane_id)
        Click (Card ' [aria-label="Split right (Ctrl+B V)"]')
        Wait-Js ("(await import('./api.js')).api.state().then(s=>s.panes.length>"+$before.Count+')')
        $script:second=((State).panes | Where-Object { $_.pane_id -notin $before } | Select-Object -First 1).pane_id
        Assert ($script:second) 'No new pane ID'
        "second=$script:second"
    }
    Check 'Drag swap via card drop' {
        $before=J (Invoke-Lab @('pane','layout','--pane',$script:pane))
        JS "(()=>{const a=document.querySelector($(J (Card ' .card-h'))), b=[...document.querySelectorAll('.card')].find(c=>c.dataset.key.endsWith('|$($script:second)'));const dt=new DataTransfer();a.dispatchEvent(new DragEvent('dragstart',{bubbles:true,dataTransfer:dt}));b.dispatchEvent(new DragEvent('dragover',{bubbles:true,dataTransfer:dt}));b.dispatchEvent(new DragEvent('drop',{bubbles:true,dataTransfer:dt}))})()" | Out-Null
        Start-Sleep -Milliseconds 1200
        $after=J (Invoke-Lab @('pane','layout','--pane',$script:pane))
        Assert ($before -ne $after) 'Geometry did not change after drop'
        $after
    }
    Check 'Divider mouse resize' {
        JS "[...document.querySelectorAll('.rail-item.ws')].find(b=>b.querySelector('.rl1').textContent==='Deck R3 verification').click()" | Out-Null
        Click '[data-view="layout"]'
        Wait-Js '!!document.querySelector("#grid .divider.v")'
        $before=J (Invoke-Lab @('pane','layout','--pane',$script:pane))
        $pos=JS '(()=>{const r=document.querySelector("#grid .divider.v").getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+Math.min(100,r.height/2)}})()'
        Send-Cdp 'Input.dispatchMouseEvent' @{type='mousePressed';x=$pos.x;y=$pos.y;button='left';clickCount=1} | Out-Null
        Send-Cdp 'Input.dispatchMouseEvent' @{type='mouseMoved';x=($pos.x+70);y=$pos.y;button='left';buttons=1} | Out-Null
        Send-Cdp 'Input.dispatchMouseEvent' @{type='mouseReleased';x=($pos.x+70);y=$pos.y;button='left';clickCount=1} | Out-Null
        Start-Sleep -Milliseconds 1200
        $after=J (Invoke-Lab @('pane','layout','--pane',$script:pane))
        Assert ($before -ne $after) 'Pane geometry did not resize'
        Click '[data-cols="2"]'; $after
        Click '.rail-item.all'
    }
    Check 'Pane focus and Herdr zoom API' {
        Api '/api/pane/focus' @{key=$script:key} | Out-Null
        Api '/api/pane/zoom' @{key=$script:key} | Out-Null
        $z=Invoke-Lab @('pane','layout','--pane',$script:pane)
        Api '/api/pane/zoom' @{key=$script:key} | Out-Null
        J $z
    }
    Check 'Tabs create rename focus close' {
        $old=@((State).tabs.tab_id)
        Api '/api/tab/create' @{workspace_id=$script:ws;cwd=$script:repo} | Out-Null
        Start-Sleep -Milliseconds 1500
        $t=((State).tabs | Where-Object tab_id -notin $old | Select-Object -First 1).tab_id
        Assert ($t) 'Tab not created'
        Api '/api/tab/rename' @{tab_id=$t;label='UI test tab'} | Out-Null
        Api '/api/tab/focus' @{tab_id=$t} | Out-Null
        Start-Sleep -Milliseconds 700
        Assert (((State).tabs | Where-Object tab_id -eq $t).label -eq 'UI test tab') 'Tab label mismatch'
        Api '/api/tab/close' @{tab_id=$t} | Out-Null
        Start-Sleep -Milliseconds 700
        Assert ($t -notin (State).tabs.tab_id) 'Tab still exists'
    }
    Check 'Folder picker and workspace rename focus close' {
        Click '#btn-folder'; Wait-Js '!!document.querySelector("dialog input[aria-label=\"Folder path\"]")'
        Fill 'dialog input[aria-label="Folder path"]' $script:repo
        $old=@((State).workspaces.workspace_id)
        Button 'Open'
        Wait-Js "(await import('./api.js')).api.state().then(s=>s.workspaces.length>$($old.Count))" 65
        $w=((State).workspaces | Where-Object workspace_id -notin $old | Select-Object -First 1).workspace_id
        Api '/api/workspace/rename' @{workspace_id=$w;label='UI folder renamed'} | Out-Null
        Api '/api/workspace/focus' @{workspace_id=$w} | Out-Null
        Start-Sleep -Milliseconds 700
        Assert (((State).workspaces | Where-Object workspace_id -eq $w).label -eq 'UI folder renamed') 'Workspace label mismatch'
        Api '/api/workspace/close' @{workspace_id=$w} | Out-Null
        Start-Sleep -Milliseconds 700
        Assert ($w -notin (State).workspaces.workspace_id) 'Workspace still exists'
    }
    Check 'Equal columns dialog' {
        Select-Pane $script:pane
        $before=@((State).panes).Count
        Chord 'V'; Fill 'dialog [aria-label="Count"]' '3'; Button 'Split'
        Wait-Js "(await import('./api.js')).api.state().then(s=>s.panes.length===$($before+2))" 40
        J (Invoke-Lab @('pane','layout','--pane',$script:pane))
    }
    Check 'Layout save and shell-only build' {
        $plan=@{Name='r3-test';Where=1;WorkspaceId=$script:ws;Columns=@(@{Folder=$script:repo;Type='shell';Value=''},@{Folder=$script:repo;Type='shell';Value=''})}
        Api '/api/layouts/save' @{name='r3-test';plan=$plan} | Out-Null
        Assert ((Api '/api/layouts').presets.name -contains 'r3-test') 'Saved preset missing'
        $before=@((State).panes).Count
        $job=Api '/api/layouts/build' @{plan=$plan}
        Wait-Jobs
        Wait-Js "(await import('./api.js')).api.state().then(s=>s.panes.length>=$($before+2))" 30
        J $job
    }
    Check 'Spaces join and separate' {
        $a=Invoke-Lab @('workspace','create','--cwd',$script:repo,'--label','R3 join A','--no-focus')
        $b=Invoke-Lab @('workspace','create','--cwd',$script:repo,'--label','R3 join B','--no-focus')
        Start-Sleep -Milliseconds 2200
        $count=@((State).workspaces).Count
        Api '/api/spaces/join' @{workspace_ids=@($a.result.workspace.workspace_id,$b.result.workspace.workspace_id)} | Out-Null
        Wait-Jobs
        Assert (@((State).workspaces).Count -eq ($count-1)) 'Join did not reduce workspace count'
        Api '/api/spaces/separate' @{} | Out-Null; Wait-Jobs
        Assert (@((State).workspaces).Count -eq $count) 'Separate did not restore workspace count'
    }
    Check 'Reservations conflict and release' {
        $r=Api '/api/reservations' @{task='tasks/r3-test';owner='offline';paths=@('app/tests/r3-owned/**')}
        Assert ((Api '/api/reservations').reservations.id -contains $r.id) 'Reservation missing'
        $refused=$false
        try { Api '/api/reservations' @{task='tasks/r3-other';owner='other';paths=@('app/tests/r3-owned/a.ps1')} | Out-Null } catch { $refused=$_.Exception.Message.Contains('409') }
        Api '/api/reservations/release' @{id=$r.id} | Out-Null
        Assert $refused 'Overlapping reservation was not refused 409'
        Assert ($r.id -notin (Api '/api/reservations').reservations.id) 'Release did not remove reservation'
    }
    Check 'Git root and paths' {
        $g=Api ('/api/git?key='+[Uri]::EscapeDataString($script:key))
        Assert ($g.root -eq $script:repo.Replace('\','/')) ('Wrong root: '+$g.root)
        Assert ($g.paths.path -contains 'app/tests/verify-ui.ps1') 'Untracked verifier missing from paths'
        "root=$($g.root); paths=$($g.paths.Count)"
    }
    Check 'Health usage and read-only panels' {
        $h=Api '/api/health'; Assert ($h.ram_free_mb -gt 0 -and $h.disks.Count -gt 0) 'Health values absent'
        Click '#btn-usage'; Wait-Js '!document.querySelector("#drawer-body").innerText.includes("Loading")' 40
        $u=JS 'document.querySelector("#drawer-body").innerText'
        Assert ($u.Length -gt 0) 'Usage panel empty'; Click '#drawer-close'
        "RAM free=$($h.ram_free_mb); usage=$u"
    }
    Check 'Tasks docs help board and tools' {
        foreach($route in @('/api/tasks','/api/docs?name=KEYS','/api/runs','/api/profiles','/api/accounts','/api/agents/tools','/api/agents/status','/api/headless/routes')) {
            $r=Api $route; "$route => $(J $r)"
        }
        Click '#btn-help'; Wait-Js 'document.querySelector("#drawer-body").innerText.includes("Ctrl")'; Click '#drawer-close'
        Click '#btn-board'; Wait-Js 'document.querySelector("#drawer-title").innerText.toLowerCase().includes("board")'; Click '#drawer-close'
        Click '#btn-tools'; Wait-Js 'document.querySelector("#drawer-title").innerText.includes("Tools")'; Click '#drawer-close'
    }
    Check 'Safe mode rejects writes' {
        Api '/api/read-only' @{enabled=$true} | Out-Null
        $refused=$false
        try { Api '/api/pane/text' @{key=$script:key;text='MUST-NOT-WRITE'} | Out-Null } catch { $refused=$_.Exception.Message.Contains('403') }
        Api '/api/read-only' @{enabled=$false} | Out-Null
        Assert $refused 'Write was not refused 403'
        Assert (-not (Screen).Contains('MUST-NOT-WRITE')) 'Safe-mode text reached pane'
    }
    Check 'Prompt history snippets and local layouts' {
        Select-Pane $script:pane
        Fill (Card ' textarea.pin') ''
        JS "document.querySelector($(J (Card ' textarea.pin'))).dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowUp',bubbles:true}))" | Out-Null
        Assert (JS "document.querySelector($(J (Card ' textarea.pin'))).value.includes('UI-PROMPT-')") 'Prompt history did not recall sent text'
        Command 'Save prompt box as snippet'
        Fill 'dialog [aria-label="Name"]' 'r3-snippet'; Fill 'dialog [aria-label="Snippet text"]' 'Hello {name}'
        Button 'Save'; Start-Sleep -Milliseconds 200
        Command 'Snippet: r3-snippet'; Fill 'dialog input' 'fixture'; Button 'Save'
        Wait-Js "document.querySelector($(J (Card ' textarea.pin'))).value==='Hello fixture'"
        Command 'Save layout'; Fill 'dialog input' 'r3-local'; Button 'Save'
        Start-Sleep -Milliseconds 200; Click '[data-cols="3"]'; Command 'Load layout r3-local'
        Assert (JS '(await import("./app.js")).S.cols===2') 'Saved layout did not restore two columns'
    }
    Check 'Raw terminal typing and activity timeline' {
        Select-Pane $script:pane
        Click (Card ' details.more summary'); Button 'Type into pane (raw mode)' (Card ' .menu')
        Assert (JS '!document.querySelector("#rawbar").hidden') 'Raw typing banner absent'
        JS "(()=>{const e=document.querySelector($(J (Card ' .screen')));for(const key of 'echo RAW-R3')e.dispatchEvent(new KeyboardEvent('keydown',{key,bubbles:true}))})()" | Out-Null
        Start-Sleep -Milliseconds 900
        Assert ((Screen).Contains('RAW-R3')) 'Raw characters did not reach shell'
        Api '/api/pane/keys' @{key=$script:key;keys=@('ctrl+c')} | Out-Null
        Click '#rawbar-stop'; Command 'Activity timeline'
        Assert (JS 'document.querySelector("#drawer-title").innerText.includes("Activity")') 'Timeline panel missing'
        Click '#drawer-close'
    }
    Check 'Broadcast and compare shells' {
        $ids=@($script:pane,$script:second)
        JS "(()=>{document.querySelectorAll('.card .sel:checked').forEach(e=>e.click());for(const id of $(J $ids)){const c=[...document.querySelectorAll('.card')].find(c=>c.dataset.key.endsWith('|'+id));c.querySelector('.sel').click()}})()" | Out-Null
        Click '#bcast-compare'; Fill '#bcast-text' "Write-Output ('BCAST-R3-' + 'OK')"
        Click '#bcast button[type="submit"]'; Button 'Send to all'
        foreach($id in $ids){Invoke-Lab @('pane','wait-output',$id,'--match','BCAST-R3-OK','--timeout','10000') | Out-Null}
        Click '#bcast-clear'; Click '[data-cols="2"]'
    }
    Check 'Readiness research refusal and proof audit' {
        $ready=JS '(async()=>{const g=await import("./guard.js");return {empty:g.readiness("no contract"),full:g.readiness("# Scope\n# Answer format\n# Checks\nBLOCKED")}})()'
        Assert (@($ready.empty|Where-Object ok).Count -eq 0 -and @($ready.full|Where-Object ok).Count -eq 4) 'Readiness did not distinguish incomplete contract'
        $refused=$false
        try { Api '/api/headless' @{purpose='research'} | Out-Null } catch { $refused=$_.Exception.Message.Contains('409') }
        Assert $refused 'Research route not refused'
        $audit=Api '/api/proof/audit' @{text='Claim without evidence. https://example.invalid/claim'}
        J $audit
    }
    Check 'Typed-input guard toast and zero writes' {
        $stub=Join-Path $PSScriptRoot 'input-stub.ps1'
        Invoke-Lab @('pane','run',$script:pane,('pwsh -NoProfile -File "'+$stub+'"')) | Out-Null
        Invoke-Lab @('pane','wait-output',$script:pane,'--match','RECEIVED=0','--timeout','10000') | Out-Null
        Invoke-Lab @('pane','report-agent',$script:pane,'--source','deck-r3-test','--agent','claude','--state','idle') | Out-Null
        Start-Sleep -Milliseconds 2400
        $before=Screen
        Fill (Card ' textarea.pin') 'GUARD-MUST-NOT-SEND'; Click (Card ' button.send')
        Wait-Js 'document.querySelector("#toasts-loud").innerText.includes("has text in its input box - not sent")' 15
        Start-Sleep -Milliseconds 600
        $after=Screen
        Assert ($before -eq $after) 'Offline input fixture changed after refused send'
        Assert (-not $after.Contains('RECEIVED=1')) 'Fixture received input'
        JS 'document.querySelector("#toasts-loud").innerText'
    }
    Check 'HUD status board and blocked inbox' {
        Invoke-Lab @('pane','report-metadata',$script:pane,'--source','deck-r3-test','--title','R3 HUD','--token','model=offline-fixture','--token','ctx=12%','--token','out=42','--ttl-ms','120000') | Out-Null
        Invoke-Lab @('pane','report-agent',$script:pane,'--source','deck-r3-test','--agent','claude','--state','blocked') | Out-Null
        Wait-Js "document.querySelector($(J (Card))).dataset.s==='blocked'" 20
        Assert (JS "document.querySelector($(J (Card))).innerText.includes('offline-fixture')") 'HUD model missing'
        Click '#btn-inbox'; Assert (JS 'document.querySelector("#drawer-body").innerText.length>0') 'Inbox empty'; Click '#drawer-close'
    }
    if (-not $SkipLayout) {
        foreach($theme in @('dark','light')) {
            foreach($width in @(900,1100,1440,2560)) {
                $row=@{theme=$theme;width=$width}
                foreach($mode in @('lab','mock')) {
                    Check "Layout $mode $width $theme" {
                        Set-DeckSize $width 900
                        Open-DeckPage ("?theme=$theme"+$(if($mode -eq 'mock'){'&mock=1'}else{''}))
                        JS '(await import("./ui.js")).toast("Layout blocked fixture", "blocked", 60000)' | Out-Null
                        $measure=JS '(()=>{const h=document.querySelector("#btn-help").getBoundingClientRect(),ts=[...document.querySelectorAll(".toast")].map(e=>e.getBoundingClientRect()),hs=[...document.querySelectorAll(".card:not([hidden]) .card-h")].map(e=>e.getBoundingClientRect());return {width:innerWidth,scroll:document.documentElement.scrollWidth,help:h.left>=0&&h.right<=innerWidth,overlap:ts.some(t=>hs.some(h=>t.left<h.right&&t.right>h.left&&t.top<h.bottom&&t.bottom>h.top)),theme:document.documentElement.dataset.theme,cards:hs.length}})()'
                        $row[$mode]=$measure
                        if($width -eq 1440 -and $mode -eq 'lab') { Save-DeckShot "r3-lab-$theme" | Out-Null }
                        Assert ($measure.scroll -le $width -and $measure.help -and -not $measure.overlap -and $measure.theme -eq $theme -and $measure.cards -gt 0) (J $measure)
                        J $measure
                    }
                }
                $script:layoutRows.Add($row)
            }
        }
    }
    Check 'Mock stress cold and warm performance' {
        Set-DeckSize 1440 900
        foreach($kind in @('cold','warm')) {
            if($kind -eq 'cold') { Send-Cdp 'Network.clearBrowserCache' | Out-Null }
            Open-DeckPage '?mock=1&stress=8&perf=1&theme=dark'
            $paint=JS 'performance.getEntriesByType("paint").map(e=>({name:e.name,ms:e.startTime}))'
            JS 'window.__deckPerf.reset()' | Out-Null
            Start-Sleep -Seconds $StressSeconds
            $report=JS 'window.__deckPerf.report()'
            $script:perfRows.Add(@{kind=$kind;paint=$paint;report=$report})
            J $script:perfRows[$script:perfRows.Count-1]
        }
    }
    Open-DeckPage '?theme=dark'
    Check 'Close finished panes' {
        Invoke-Lab @('pane','report-agent',$script:pane,'--source','deck-r3-test','--agent','claude','--state','idle') | Out-Null
        Wait-Js '[...document.querySelectorAll("#health button")].some(b=>b.textContent==="Close finished panes")' 25
        Button 'Close finished panes' '#health'
        Assert (JS 'document.activeElement.textContent==="Cancel"') 'Cancel is not default focus'
        $label=JS '[...document.querySelectorAll("dialog button")].find(b=>b.classList.contains("danger")).textContent'
        Button $label
        Wait-Js "(await import('./api.js')).api.state().then(s=>!s.panes.some(p=>p.pane_id===$(J $script:pane)))" 30
    }
    foreach ($w in @((State).workspaces.workspace_id | Where-Object { $_ -notin $script:initialWs })) {
        Api '/api/workspace/close' @{workspace_id=$w} | Out-Null
    }
    Check 'Stop-all typed confirmation and offline state' {
        Chord 'Q'; Fill 'dialog input' 'wrong'; Button 'Confirm'
        Assert (JS '!!document.querySelector("dialog[open]")') 'Wrong word accepted'
        Assert ((State).herdr.running) 'Lab stopped with wrong confirmation'
        Fill 'dialog input' 'stop'; Button 'Confirm'
        Wait-Js 'document.querySelector("#empty").innerText.includes("Herdr is not running")' 45
        Assert (-not (State).herdr.running) 'Lab still running'
    }
} finally {
    Stop-DeckUi
    $remaining=@($script:ownedPids | Where-Object { Get-Process -Id $_ -ErrorAction SilentlyContinue })
    $script:results.Add(@{name='Owned-process cleanup';status=$(if($remaining.Count){'FAIL'}else{'PASS'});detail="remaining=$($remaining -join ',')"})
    $pass=@($script:results|Where-Object status -eq 'PASS').Count
    $fail=@($script:results|Where-Object status -eq 'FAIL').Count
    Write-Output "SUMMARY PASS=$pass FAIL=$fail"
    Write-Output ('LAYOUT '+(J @($script:layoutRows)))
    Write-Output ('PERF '+(J @($script:perfRows)))
    # Durable evidence is kept in the task answer (the task's permitted evidence file).
    $report="PARTIAL`nChanged: app/tests/verify-ui.ps1; app/tests/input-stub.ps1; tasks/deck-r3-codex-bugs.md; tasks/deck-r3-codex-terra-answer.md`nItems: 1..5 verification in progress`nChecks: PASS=$pass FAIL=$fail`n`n"+(@($script:results)|ForEach-Object { $_.status+' '+$_.name+' '+$_.detail }) -join "`n"
    $report+="`nLAYOUT $(J @($script:layoutRows))`nPERF $(J @($script:perfRows))`n"
    Set-Content -LiteralPath (Join-Path $script:repo 'tasks/deck-r3-codex-terra-answer.md') -Value $report
}
if (@($script:results|Where-Object status -eq 'FAIL').Count) { exit 1 }


