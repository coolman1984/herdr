// Global screen search off the main thread. Holds the latest plain text per pane.
const docs = new Map();
self.onmessage = (ev) => {
  const d = ev.data;
  if (d.drop) { docs.delete(d.drop); return; }
  if (d.doc) { docs.set(d.doc.key, { name: d.doc.name, lines: d.doc.text.split('\n') }); return; }
  if (d.q == null) return;
  const q = d.q.trim(), hits = [];
  let test;
  const rx = /^\/(.+)\/([a-z]*)$/.exec(q);
  if (!q) { self.postMessage({ seq: d.seq, hits }); return; }
  if (rx) {
    let re;
    try { re = new RegExp(rx[1], rx[2].includes('i') ? rx[2] : rx[2] + 'i'); } catch (e) { self.postMessage({ seq: d.seq, hits: [{ key: '', name: 'Bad regex', n: 0, line: e.message }] }); return; }
    test = (l) => { re.lastIndex = 0; return re.test(l); };   // /g and /y keep lastIndex between lines
  } else { const s = q.toLowerCase(); test = (l) => l.toLowerCase().includes(s); }
  outer: for (const [key, doc] of docs) {
    for (let n = doc.lines.length - 1; n >= 0; n--) {   // newest first
      const line = doc.lines[n];
      if (test(line)) { hits.push({ key, name: doc.name, n, line: line.trim().slice(0, 200) }); if (hits.length >= 200) break outer; }
    }
  }
  self.postMessage({ seq: d.seq, hits });
};
