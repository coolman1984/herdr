import test from 'node:test';
import assert from 'node:assert/strict';
import { browser, NodeStub } from './dom.mjs';
const env = browser();
const U = await import('../../web/ui.js');
const { api, ROUTES } = await import('../../web/api.js');
const P = await import('../../web/panels.js');
const G = await import('../../web/guard.js');
const Q = await import('../../web/parity.js');
const { ScreenView } = await import('../../web/screen.js');
const deferred = () => { let resolve; const promise = new Promise(r => resolve = r); return { promise, resolve }; };
const pane = (key = 's|w:p1') => ({ key, pane_id: key.split('|')[1], cwd: 'D:/repo', name: 'codex-1', agent: 'codex', agent_status: 'unknown' });
const S = { safe: false, writeBlocked: false, panes: new Map([['s|w:p1', pane()]]), profiles: [], root: 'D:/repo', session: 's' };
const cards = new Map();
const ctx = { S, LS: { get: (k,d) => d, set() {} }, nameOf: p => p.name || p.key, statusOf: p => p.agent_status, schedule() {}, card: key => cards.get(key), currentKey: () => 's|w:p1', plainOf: () => '' };
P.init(ctx); G.init(ctx); Q.init(ctx);
test('C1 visible carries page identity, including empty hidden interest', async () => {
  const calls = []; globalThis.fetch = async (url, options) => { calls.push([url, JSON.parse(options.body)]); return new Response('{}'); };
  await api.visible(['s|w:p1'], 'c-3'); await api.visible([], 'c-3');
  assert.deepEqual(calls.map(c => c[1]), [{ client_id: 'c-3', keys: ['s|w:p1'] }, { client_id: 'c-3', keys: [] }]);
});
test('C3 raw transaction sends text and keys together', async () => {
  let sent; globalThis.fetch = async (url, options) => { sent = [url, JSON.parse(options.body)]; return new Response('{}'); };
  await api.raw('A', '你好', ['enter']); assert.match(sent[0], /\/api\/pane\/raw/); assert.deepEqual(sent[1], { key: 'A', text: '你好', keys: ['enter'] });
});
test('C3 per-pane queue preserves order and drain awaits acknowledgement', async () => {
  const wait = deferred(), calls = [];
  const queue = new U.OrderedQueue(async (key,text,keys) => { calls.push([key,text,keys]); if (text === 'first') await wait.promise; });
  const a = queue.push('A','first'); const b = queue.push('A','second',['enter']); await queue.push('B','other');
  assert.deepEqual(calls.map(c => c[1]), ['first','other']); let drained = false; queue.drain('A').then(() => drained = true); await Promise.resolve(); assert.equal(drained,false);
  wait.resolve(); await Promise.all([a,b]); assert.equal(calls[2][1],'second'); assert.equal(queue.tails.size,0);
});
test('C2 safe mode cancels queued raw chunks', async () => {
  const wait = deferred(), calls = []; const queue = new U.OrderedQueue(async (_,text) => { calls.push(text); await wait.promise; });
  const a = queue.push('A','first'), b = queue.push('A','discard'); await Promise.resolve(); queue.clear(); wait.resolve(); await Promise.all([a,b]); assert.deepEqual(calls,['first']);
});
test('P4 removing a pane cancels queued raw input even if its key is reused', async () => {
  const wait=deferred(), calls=[]; const queue=new U.OrderedQueue(async(_,text)=>{calls.push(text); if(text==='old') await wait.promise;});
  const first=queue.push('A','old'), queued=queue.push('A','discard'); await Promise.resolve(); queue.forget('A');
  await queue.push('A','new'); wait.resolve(); await Promise.all([first,queued]); assert.deepEqual(calls,['old','new']);
});
test('C3 IME ignores composing key events and pane switch drains old target', async () => {
  const a = 's|w:p1', b = 's|w:p2', wait = deferred(), calls = [];
  S.panes.set(b,pane(b)); for (const key of [a,b]) cards.set(key, { key, root: new NodeStub(), screenBox: new NodeStub() });
  api.raw = async (...args) => { calls.push(args); if (args[1] === 'a') await wait.promise; };
  await P.rawMode(cards.get(a));
  P.rawKey({ key: 'Enter', isComposing: true, preventDefault() { assert.fail('IME prevented'); } }, cards.get(a));
  P.rawKey({ key: 'a', preventDefault() {} }, cards.get(a));
  const switched = P.rawMode(cards.get(b)); await Promise.resolve(); await Promise.resolve();
  assert.equal(S.raw,null); assert.equal(calls[0][0],a); assert.equal(calls[0][1],'a'); wait.resolve(); await switched; assert.equal(S.raw,b); P.clearRaw();
});
test('C6 read timeout and not-sent are failures; started prompt network errors are uncertain', () => {
  assert.equal(U.deliveryUnknown({ status: 504 }),false); assert.equal(U.deliveryUnknown({ status: 502, dispatch_id:'x' }),false);
  assert.equal(U.deliveryUnknown({ message:'network timeout' }),false); assert.equal(U.deliveryUnknown({ status:504, dispatch_id:'x' }),true); assert.equal(U.deliveryUnknown({ promptSent:true }),true);
});
test('C6 API marks network errors only for a started prompt POST', async () => {
  globalThis.fetch = async () => { throw new TypeError('Failed to fetch'); };
  await assert.rejects(api.prompt('A','x'), e => e.promptSent === true); await assert.rejects(api.state(), e => !e.promptSent);
  globalThis.fetch = async () => new Response('invalid json');
  await assert.rejects(api.prompt('A','x'), e => !e.promptSent);
  globalThis.fetch = async () => ({ text: async () => { throw new TypeError('body disconnected'); } });
  await assert.rejects(api.prompt('A','x'), e => e.promptSent === true);
});
test('C7 input_unknown preserves certain failure, never records unknown delivery', async () => {
  const original = api.prompt; api.prompt = async () => { throw Object.assign(new Error('not sent'), { status:409, code:'input_unknown' }); };
  assert.equal(await G.sendGuarded('s|w:p1','draft'),null); assert.equal(G.isUncertain('s|w:p1'),false); assert.equal(document.querySelector('#toasts').children.at(-1).className,'toast warn'); api.prompt = original;
});
test('C8 preferences use whole-object load and write contracts', async () => {
  const calls = []; globalThis.fetch = async (url,options) => { calls.push([url, options.body]); return new Response('{"prefs":{"cols":3}}'); };
  assert.equal((await api.prefs()).prefs.cols,3); await api.savePrefs({ cols:2, theme:'dark' }); assert.match(calls[0][0],/\/api\/prefs/); assert.deepEqual(JSON.parse(calls[1][1]),{ prefs:{ cols:2,theme:'dark' } });
});
test('C9 null health capacity is unknown, real zero stays zero', () => { assert.equal(U.knownNumber(null),'unknown'); assert.equal(U.knownNumber(undefined),'unknown'); assert.equal(U.knownNumber(0),'0.0'); });
const task = () => ({ destination:'new', profile:'codex', acct:'', overlap:false, cur:pane(), chosen:'tasks/a.md', pathsCopy:['app/web/**'], owner:'codex', prompt:'task' });
test('C4/D13 reserve and fingerprint baseline precede dispatch; failure releases reservation', async () => {
  const calls = []; api.reserve = async () => { calls.push('reserve'); return { id:'r1' }; }; api.scopeBaseline = async () => { calls.push('baseline'); };
  api.start = async () => { calls.push('start'); throw new Error('not sent'); }; api.release = async id => calls.push('release:'+id);
  await G.dispatchTask(task()); assert.deepEqual(calls,['reserve','baseline','start','release:r1']);
});
test('C4 reservation conflicts dispatch nothing; explicit overlap reaches reserve', async () => {
  let started = false, overlap; api.reserve = async (_t,_o,_p,value) => { overlap = value; throw Object.assign(new Error('conflict'),{status:409}); }; api.start = async () => started = true;
  await G.dispatchTask({ ...task(), overlap:true }); assert.equal(overlap,true); assert.equal(started,false);
});
test('C4/C6 delivery unknown retains reservation; definite failure releases only new ownership', async () => {
  const original=api.prompt; let released=0; api.reserve=async()=>({id:'r-existing'}); api.scopeBaseline=async()=>({}); api.release=async()=>released++;
  api.prompt=async()=>{throw Object.assign(new Error('delivery unknown'),{status:504,dispatch_id:'d'});};
  await G.dispatchTask({...task(),destination:'s|w:p1'}); assert.equal(released,0); assert.equal(G.isUncertain('s|w:p1'),true);
  G.forgetPane('s|w:p1'); api.prompt=async()=>{throw Object.assign(new Error('read timeout'),{status:504,code:'read_timeout'});};
  await G.dispatchTask({...task(),destination:'s|w:p1',existingIds:['r-existing']}); assert.equal(released,0); assert.equal(G.isUncertain('s|w:p1'),false);
  await G.dispatchTask({...task(),destination:'s|w:p1'}); assert.equal(released,1); api.prompt=original;
});
test('D12 returned pane_id attaches two launches despite same-profile names and reversed identity', async () => {
  api.reserve = async () => ({id:'r'}); api.scopeBaseline = async () => ({}); let n=0; api.start = async () => ({ pane_id: ++n === 1 ? 'w:p8' : 'w:p7', name:'same-name' });
  await G.dispatchTask(task()); await G.dispatchTask({...task(),chosen:'tasks/b.md'});
  assert.equal(G.launches.get('s|w:p8').task,'tasks/a.md'); assert.equal(G.launches.get('s|w:p7').task,'tasks/b.md');
});
test('D12 missing pane_id stays visibly unattached; baseline failures stay unknown', async () => {
  api.start = async () => ({}); api.scopeBaseline = async () => { throw new Error('unavailable'); };
  await G.dispatchTask(task()); assert.equal(G.unattachedLaunches.at(-1).baseline,false); assert.equal(G.unattachedLaunches.at(-1).task,'tasks/a.md');
});
test('C5 held launch queue awaits each start response', async () => {
  const wait = deferred(), calls = []; G.setHealth({ ram_used_pct:20,disks:[{drive:'D:',free_gb:50}] });
  const first = G.enqueueStart('a',async () => { calls.push('a'); await wait.promise; }); G.enqueueStart('b',async () => calls.push('b'));
  await Promise.resolve(); assert.deepEqual(calls,['a']); wait.resolve(); await first; assert.deepEqual(calls,['a','b']);
});
test('D15 menu collision keeps edges in viewport', () => {
  const r = U.menuPosition({right:790,top:590,bottom:610},{width:250,height:300},{width:800,height:600}); assert.deepEqual(r,{left:540,top:290});
  assert.deepEqual(U.menuPosition({right:10,top:5,bottom:25},{width:200,height:100},{width:320,height:240}),{left:8,top:25});
});
test('D15 Escape and outside pointer close portal and restore trigger focus', () => {
  const trigger = new NodeStub('button'), menu = new NodeStub(); menu.append(new NodeStub('button')); document.body.append(trigger);
  U.openMenu(trigger,menu); assert.equal(menu.parentNode,document.body);
  document.dispatchEvent(new Event('pointerdown')); assert.equal(menu.parentNode,null); assert.equal(document.activeElement,trigger);
  U.openMenu(trigger,menu); const escape = new Event('keydown'); Object.assign(escape,{key:'Escape'}); document.dispatchEvent(escape); assert.equal(menu.parentNode,null); assert.equal(trigger.getAttribute('aria-expanded'),'false');
});
test('D16 late drawer generation cannot populate another drawer', async () => {
  const wait = deferred(); let signal;
  P.drawer('slow','Slow',async (b,s) => { signal=s; await wait.promise; b.append('wrong panel'); });
  P.drawer('fresh','Fresh',b => b.append('fresh panel')); await Promise.resolve(); wait.resolve(); await Promise.resolve(); await Promise.resolve();
  assert.equal(signal.aborted,true); assert.equal(document.querySelector('#drawer-body').textContent,'fresh panel'); P.closeDrawer();
});
test('D16 live drawer refresh preserves unsaved value, node, focus and working handler', async () => {
  let read;
  P.drawer('form','Form',b => { const input=U.el('input',{'aria-label':'Limit',value:'4'}); b.append(input,U.el('button',{onclick:()=>read=input.value},'Save')); },true);
  await Promise.resolve(); const body=document.querySelector('#drawer-body'), input=body.querySelector('input'); input.value='9'; input.focus();
  P.refreshOpen(true); await Promise.resolve(); assert.equal(body.querySelector('input'),input); assert.equal(input.value,'9'); assert.equal(document.activeElement,input);
  body.querySelector('button').dispatchEvent(new Event('click')); assert.equal(read,'9'); P.closeDrawer();
});
test('D16 keyed live fields survive an inserted status section', () => {
  const body=U.el('div'), section=U.el('section',{'data-drawer-key':'evidence'},U.el('input',{'aria-label':'Proof',value:'draft'})); body.append(section);
  const input=section.querySelector('input'); input.value='unsaved'; input.focus();
  U.patchContent(body,U.el('div',{},U.el('p',{},'Updated status'),U.el('section',{'data-drawer-key':'evidence'},U.el('input',{'aria-label':'Proof',value:'saved'}))));
  assert.equal(body.querySelector('input'),input); assert.equal(input.value,'unsaved'); assert.equal(document.activeElement,input);
});
test('D17 completion before response is upserted and never downgraded by running status', () => {
  const jobs=new Map(); U.upsertJob(jobs,{id:'j',status:'done',ok:true,output:'answer'}); U.upsertJob(jobs,{id:'j',label:'ask'}); U.upsertJob(jobs,{id:'j',status:'running',ok:false,output:''});
  assert.equal(jobs.get('j').done,true); assert.equal(jobs.get('j').output,'answer');
  U.upsertJob(jobs,{id:'k',status:'running',ok:false}); assert.equal(jobs.get('k').done,false);
});
test('D17 reconnect reconciliation discovers unknown finished jobs', async () => { api.jobs=async()=>({jobs:[{id:'finished-remote',status:'done',ok:true,output:'recovered'}]}); await Q.syncJobs(); assert.equal(Q.jobs.get('finished-remote').output,'recovered'); });
test('P1 repeated topology values compare equal, changed layout differs', () => { assert.equal(U.sameValues([{tab_id:'t',panes:[1]}],[{tab_id:'t',panes:[1]}]),true); assert.equal(U.sameValues([{tab_id:'t',panes:[1]}],[{tab_id:'t',panes:[2]}]),false); });
test('P4 queued screen patches cap count and bytes and drop poisoned backlog', () => {
  const p={}; for(let i=0;i<64;i++) assert.equal(U.queuePatch(p,{lines:['x']}),true); assert.equal(U.queuePatch(p,{lines:['x']}),false); assert.equal(p.pq,null); assert.equal(p.needsResync,true); assert.equal(U.queuePatch({}, {lines:['x'.repeat(140000)]}),false);
});
test('P4 job retention is bounded', () => { const jobs=new Map(); for(let i=0;i<150;i++) U.upsertJob(jobs,{id:String(i),status:'done',ok:true,output:'x'.repeat(40000)}); assert.equal(jobs.size,100); assert.equal(jobs.get('149').output.length,32768); });
test('P5 ResizeObserver only records entry sizes; fit reads and writes wait for frame passes', () => {
  const box=new NodeStub(), view=new ScreenView(box,()=>{},new AbortController().signal); let reads=0,writes=0;
  document.createElement = name => { const n=new NodeStub(name); n.getBoundingClientRect=()=>{reads++;return {width:1200};}; return n; };
  box.style={setProperty(){writes++;},removeProperty(){writes++;}}; view.cols=80;
  env.observers[0].callback([{target:box,contentRect:{width:400,height:300}}]); assert.equal(reads,0); assert.equal(writes,0);
  view.prepare(); assert.ok(reads>0); assert.equal(writes,0); view.paintSize(); assert.ok(writes>0);
});
test('mock contracts emit hello/settings/state, persist prefs, return pane IDs and retain stale panes', async () => {
  const { createMock } = await import('../../web/mock.js');
  const mock = createMock(ROUTES), events=[];
  mock.events([], { onOpen() {}, onEvent:(name,data)=>events.push([name,data]) });
  await env.tick(50); await env.tick(0);
  assert.deepEqual(events.slice(0,3).map(e=>e[0]),['hello','settings','state']);
  const call = async (method,route,data) => { const p=mock[method](route,data); await env.tick(120); return p; };
  await call('post',ROUTES.prefs,{prefs:{cols:3}}); assert.equal((await call('get',ROUTES.prefs)).prefs.cols,3);
  const launch=await call('post',ROUTES.start,{profile:'codex'}); assert.ok(launch.pane_id); assert.ok(launch.name);
  await call('post',ROUTES.readOnly,{enabled:true}); await env.tick(0); assert.equal(events.filter(e=>e[0]==='settings').at(-1)[1].read_only,true);
  await assert.rejects(call('post',ROUTES.raw,{key:'lab|w1:p1',text:'x'}),e=>e.status===403);
  window.__mock.setRunning(false); await env.tick(0); const stale=events.filter(e=>e[0]==='state').at(-1)[1]; assert.ok(stale.panes.length); assert.ok(stale.panes.every(p=>p.stale));
});
test('page applies settings, visibility and prefs; ordinary snapshots preserve rail and lazy cards; removal cleans records', async () => {
  const prefsWait=deferred(); api.prefs=()=>prefsWait.promise;
  let events; api.events=(_names,handlers)=>{events=handlers;return ()=>{};};
  const visible=[],saved=[]; api.visible=async(keys,id)=>visible.push({keys,id}); api.savePrefs=async p=>saved.push(structuredClone(p));
  api.state=async()=>({session:'s',herdr:{running:true},panes:[]});
  const A=await import('../../web/app.js'); await Promise.resolve();
  prefsWait.resolve({prefs:{cols:3,rail:false,theme:'light'}}); for(let i=0;i<12;i++) await Promise.resolve();
  assert.equal(A.S.cols,3); assert.equal(A.S.rail,false); assert.equal(document.documentElement.dataset.theme,'light'); assert.ok(events);
  events.onOpen(); events.onEvent('hello',{client_id:'c-17'}); events.onEvent('settings',{read_only:true}); assert.equal(A.S.safe,true);
  events.onEvent('settings',{read_only:false}); assert.equal(A.S.safe,false);
  // Preference writes also come through real drawer settings (look uses the shared LS).
  P.setLook('density','compact'); await env.tick(500); assert.equal(saved.at(-1).density,'compact'); assert.equal(saved.at(-1).cols,3);
  const ps=[pane(),pane('s|w:p2')]; A.S.hidden.add('s|w:p2');
  const state={session:'s',read_only:false,herdr:{running:true},panes:ps,workspaces:[{workspace_id:'w',label:'work'}],tabs:[{tab_id:'t',workspace_id:'w'}],layouts:[]};
  A.applyState(state); env.frame();
  assert.equal(A.cards.size,1); const rail=document.querySelector('#rail-nav').firstChild;
  A.applyState(structuredClone(state)); env.frame(); assert.equal(document.querySelector('#rail-nav').firstChild,rail);
  await env.tick(150); assert.deepEqual(visible.at(-1),{keys:['s|w:p1'],id:'c-17'});
  document.hidden=true; document.dispatchEvent(new Event('visibilitychange')); await env.tick(150); assert.deepEqual(visible.at(-1),{keys:[],id:'c-17'}); document.hidden=false;
  A.applyState({...state,herdr:{running:false},panes:ps.map(p=>({...p,stale:true}))}); env.frame(); assert.equal(A.S.writeBlocked,true); assert.ok(A.cards.get('s|w:p1').root.classList.contains('stale')); assert.equal(A.cards.get('s|w:p1').ta.disabled,true);
  A.applyState(state); env.frame(); assert.equal(A.S.panes.get('s|w:p1').stale,false); assert.equal(A.S.writeBlocked,false);
  A.mergePane({...ps[1],name:'',agent:''});
  api.reserve=async()=>({id:'new-launch'}); api.scopeBaseline=async()=>({}); api.start=async()=>({pane_id:'w:p2',name:'codex-fresh'});
  await G.dispatchTask({...task(),chosen:'shell-transition-task'}); A.mergePane({...ps[1],name:'codex-fresh',agent:'codex'});
  assert.equal(G.launches.get('s|w:p2').task,'shell-transition-task');
  A.S.hist.set('s|w:p2',[1]); A.S.alarmAt.set('s|w:p2',1); G.launches.set('s|w:p2',{task:'x'});
  const resync=deferred(); let reads=0; api.state=()=>{reads++;return resync.promise;};
  for(let i=0;i<130;i++) A.mergePane({key:'s|w:p2',screen_patch:{lines:['x'],start:0,version:i+1,from_version:i}});
  assert.equal(reads,1); assert.equal(A.S.panes.get('s|w:p2').pq,null);
  A.applyState({...state,panes:[ps[0]]}); env.frame(); assert.equal(A.S.hist.has('s|w:p2'),false); assert.equal(A.S.alarmAt.has('s|w:p2'),false); assert.equal(G.launches.has('s|w:p2'),false); assert.equal(A.S.hidden.has('s|w:p2'),false);
  // A long stream outage plus a failed state probe yields the persistent restart instruction.
  api.state=async()=>{throw new Error('offline');}; events.onDown(new Error('lost')); A.S.downSince=Date.now()-11000; await env.tick(2000); env.frame(); assert.equal(A.S.restarted,true); assert.match(document.querySelector('#banner').textContent,/Deck stopped or restarted - reopen it from the Herdr launcher/);
});
