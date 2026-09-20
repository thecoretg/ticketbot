/* navi — drag-to-reorder for grip-handled lists. No dependencies.
 *
 * The kit styles the parts (.rule-grip, .is-source, .drag-ghost, .drop-line);
 * this adds the gesture. It never reorders the DOM itself — it reports the move
 * and leaves the list to re-render from its own data, which is the only thing
 * that works when the list is redrawn from state on every change.
 *
 *   import { reorder } from './reorder.js'
 *
 *   const stop = reorder(document.querySelector('.rule-list'), {
 *     item:   '.rule-card',
 *     handle: '.rule-grip',
 *     onMove: (from, to) => { rules.splice(to, 0, rules.splice(from, 1)[0]); render() },
 *   })
 *
 * The list needs `position: relative` (.rule-list has it) — the ghost and the
 * drop line are positioned inside it.
 *
 * Pointer only. Keep the up/down buttons: they are the keyboard path, and this
 * helper deliberately does not try to be one.
 */

const GAP = 18;   // half the gap the kit leaves between cards, for the drop line

export function reorder(list, opts = {}) {
  const itemSel   = opts.item   || '[data-reorder-item]';
  const handleSel = opts.handle || '[data-reorder-handle]';
  const ghostOf   = opts.ghost  || (el => el.firstElementChild);
  const onMove    = opts.onMove || (() => {});

  let drag = null;   // { item, from, dx, dy, ghost, line, handle, pointerId }

  const items = () => Array.from(list.querySelectorAll(itemSel));

  // the slot the pointer is over: 0..n, an insertion point, not an item index
  function slotAt(clientY, all) {
    for (let i = 0; i < all.length; i++) {
      const r = all[i].getBoundingClientRect();
      if (clientY < r.top + r.height / 2) return i;
    }
    return all.length;
  }

  function slotY(slot, all) {
    if (slot < all.length) return Math.max(all[slot].offsetTop - GAP, -2);
    const last = all[all.length - 1];
    return last ? last.offsetTop + last.offsetHeight + GAP - 2 : 0;
  }

  function onDown(e) {
    if (e.button !== undefined && e.button !== 0) return;
    const handle = e.target.closest(handleSel);
    if (!handle || !list.contains(handle)) return;
    const item = handle.closest(itemSel);
    if (!item) return;

    const all = items();
    const from = all.indexOf(item);
    if (from < 0) return;

    e.preventDefault();
    /* capture keeps the gesture alive when the pointer leaves the handle, but it
       is an enhancement: if it is refused, the listeners on the list still see
       the move and the drop. Never let it abort the drag. */
    try { handle.setPointerCapture?.(e.pointerId); } catch (_) {}

    const ir = item.getBoundingClientRect();
    const lr = list.getBoundingClientRect();

    const ghost = document.createElement('div');
    ghost.className = 'drag-ghost';
    ghost.setAttribute('aria-hidden', 'true');
    ghost.style.width = ir.width + 'px';
    const face = ghostOf(item);
    if (face) {
      const clone = face.cloneNode(true);
      clone.removeAttribute('id');
      clone.querySelectorAll('[id]').forEach(n => n.removeAttribute('id'));
      ghost.appendChild(clone);
    }

    const line = document.createElement('div');
    line.className = 'drop-line';
    line.setAttribute('aria-hidden', 'true');

    list.append(ghost, line);
    item.classList.add('is-source');

    drag = {
      item, from, handle, ghost, line,
      pointerId: e.pointerId,
      dx: ir.left - e.clientX, dy: ir.top - e.clientY,
      lx: lr.left, ly: lr.top,
      slot: from,
    };
    paint(e.clientX, e.clientY);
    window.addEventListener('keydown', onKey, true);
  }

  function paint(x, y) {
    const all = items();   // the source stays in place, so it still counts as a slot
    drag.ghost.style.left = (x + drag.dx - drag.lx) + 'px';
    drag.ghost.style.top  = (y + drag.dy - drag.ly) + 'px';
    drag.slot = slotAt(y, all);
    drag.line.style.top = slotY(drag.slot, all) + 'px';
  }

  function onMoveEvt(e) {
    if (!drag || e.pointerId !== drag.pointerId) return;
    paint(e.clientX, e.clientY);
  }

  function onUp(e) {
    if (!drag || e.pointerId !== drag.pointerId) return;
    const { from, slot } = drag;
    const to = slot > from ? slot - 1 : slot;
    cleanup();
    if (to !== from) onMove(from, to);
  }

  function onKey(e) {
    if (e.key === 'Escape' && drag) { e.preventDefault(); cleanup(); }
  }

  function cleanup() {
    if (!drag) return;
    const { item, ghost, line, handle, pointerId } = drag;
    if (handle.releasePointerCapture) {
      try { handle.releasePointerCapture(pointerId); } catch (_) {}
    }
    item.classList.remove('is-source');
    ghost.remove();
    line.remove();
    drag = null;
    window.removeEventListener('keydown', onKey, true);
  }

  list.addEventListener('pointerdown', onDown);
  list.addEventListener('pointermove', onMoveEvt);
  list.addEventListener('pointerup', onUp);
  list.addEventListener('pointercancel', cleanup);

  return function stop() {
    cleanup();
    list.removeEventListener('pointerdown', onDown);
    list.removeEventListener('pointermove', onMoveEvt);
    list.removeEventListener('pointerup', onUp);
    list.removeEventListener('pointercancel', cleanup);
  };
}
