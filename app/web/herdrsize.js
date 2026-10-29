// Herdr pane size from the Deck cards. With no Herdr window attached, Herdr keeps every tab at 120x40,
// so two panes get 60 columns however wide the cards are (LESSONS #95). The backend joins Herdr as a
// size-only client (app/internal/deck/sizer.go); this module tells it the card area in terminal cells.
// Measured at the BASE font, so the screen's fit-to-width font (which only grows) never changes the
// answer: no resize loop between Deck and Herdr.
import { cellSize, SCREEN_PAD } from './screen.js';

// Herdr keeps a frame around each pane's terminal. Lab, two panes side by side: a 60x40 pane rect
// runs its program at 57x38, a 105x37 rect at 102x35 -> 3 columns and 2 rows per pane.
export const PANE_FRAME = { cols: 3, rows: 2 };

// rects: the visible cards' screen boxes. The first row of cards (same top) gives the columns and
// the first column (same left) gives the rows, like the Herdr split that the cards stand for.
export function areaCells(rects, cell, pad = SCREEN_PAD, frame = PANE_FRAME) {
  if (!rects.length || !(cell.w > 0) || !(cell.h > 0)) return null;
  const [first] = rects;
  const row = rects.filter((r) => Math.abs(r.top - first.top) < 4);
  const col = rects.filter((r) => Math.abs(r.left - first.left) < 4);
  const cols = row.reduce((n, r) => n + Math.max(1, Math.floor((r.width - pad) / cell.w)) + frame.cols, 0);
  const rows = col.reduce((n, r) => n + Math.max(1, Math.floor(r.height / cell.h)) + frame.rows, 0);
  return { cols: Math.min(1000, Math.max(20, cols)), rows: Math.min(500, Math.max(5, rows)) };
}

// Debounced reporter: send(cols, rows) runs at most once per `wait` ms and only for a new size.
export function sizeReporter(measure, send, wait = 300) {
  let timer = 0, last = '';
  return {
    soon() { clearTimeout(timer); timer = setTimeout(() => this.now(), wait); },
    now() {
      const a = measure();
      if (!a) return;
      const key = a.cols + 'x' + a.rows;
      if (key === last) return;
      last = key;
      Promise.resolve(send(a.cols, a.rows)).catch(() => { last = ''; });
    },
    reset() { last = ''; },
  };
}
