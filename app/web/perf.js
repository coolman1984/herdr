// ?perf=1 overlay: fps, p95 frame time, long tasks, slowest input event, render work per frame,
// JS heap, DOM nodes, rows in the DOM, SSE messages/s. window.__deckPerf.report() returns the numbers
// (used for app/PERF.md). Runs only when asked for and pauses while the window is hidden.
import { setPerfHook } from './parity.js';

const frames = [], work = [];
let longTasks = 0, longMax = 0, evMax = 0, msgs = 0, msgsRate = 0, last = 0, raf = 0, started = performance.now();
const box = document.createElement('div');
box.className = 'perf'; box.setAttribute('aria-hidden', 'true');
document.body.append(box);
const P = window.__deckPerf = {
  msg() { msgs++; },
  report() {
    const f = frames.slice().sort((a, b) => a - b), w = work.slice().sort((a, b) => a - b);
    const q = (arr, p) => (arr.length ? arr[Math.min(arr.length - 1, Math.floor(arr.length * p))] : 0);
    return {
      seconds: Math.round((performance.now() - started) / 1000), frames: f.length,
      fps: f.length ? Math.round(1000 / (f.reduce((a, b) => a + b, 0) / f.length)) : 0,
      p95_frame_ms: +q(f, 0.95).toFixed(1), max_frame_ms: +(f[f.length - 1] || 0).toFixed(1),
      frames_over_16_7: f.filter((x) => x > 16.7 + 0.5).length,
      p95_render_ms: +q(w, 0.95).toFixed(2), max_render_ms: +(w[w.length - 1] || 0).toFixed(2),
      long_tasks: longTasks, long_task_max_ms: Math.round(longMax), slowest_input_ms: Math.round(evMax),
      heap_mb: performance.memory ? +(performance.memory.usedJSHeapSize / 1048576).toFixed(1) : null,
      dom_nodes: document.getElementsByTagName('*').length, screen_rows_in_dom: document.querySelectorAll('.trow:not([hidden])').length,
      sse_msgs_per_s: msgsRate,
    };
  },
  reset() { frames.length = 0; work.length = 0; longTasks = 0; longMax = 0; evMax = 0; started = performance.now(); },
};
try {
  new PerformanceObserver((l) => { for (const e of l.getEntries()) { longTasks++; longMax = Math.max(longMax, e.duration); } }).observe({ type: 'longtask', buffered: true });
  new PerformanceObserver((l) => { for (const e of l.getEntries()) evMax = Math.max(evMax, e.duration); }).observe({ type: 'event', durationThreshold: 16, buffered: true });
} catch (e) { console.warn('PerformanceObserver unavailable', e); }
setPerfHook((ms) => { work.push(ms); if (work.length > 3000) work.shift(); });
function loop(t) {
  if (last) { frames.push(t - last); if (frames.length > 3000) frames.shift(); }
  last = t;
  raf = requestAnimationFrame(loop);
}
let tick = 0;
function show() {
  if (document.hidden) return;
  msgsRate = msgs; msgs = 0;
  const r = P.report();
  box.textContent = `fps ${r.fps}  p95 ${r.p95_frame_ms} ms  max ${r.max_frame_ms}\nrender p95 ${r.p95_render_ms} ms  long tasks ${r.long_tasks}\n` +
    `input max ${r.slowest_input_ms} ms  heap ${r.heap_mb ?? '?'} MB\nDOM ${r.dom_nodes}  rows ${r.screen_rows_in_dom}  sse ${r.sse_msgs_per_s}/s`;
}
document.addEventListener('visibilitychange', () => {
  if (document.hidden) { cancelAnimationFrame(raf); raf = 0; clearInterval(tick); tick = 0; last = 0; }
  else { raf = requestAnimationFrame(loop); tick = setInterval(show, 1000); }
});
raf = requestAnimationFrame(loop); tick = setInterval(show, 1000);
