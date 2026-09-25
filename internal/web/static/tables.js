// ─────────────────────────────────────────────────────────
// Tables
//
// dataTable renders a table card from a column model; tableWrap renders just the scrolling table,
// for a page that builds its own card around it (tickets). Every table gets sorting from its
// headers (a real button in each sortable th), column resizing and reordering with the mouse, and
// a header menu that resets them. Those choices are remembered per table id in localStorage.
//
// Nothing is set on the DOM after a render and left there. Sort, widths and order live in the
// prefs and are baked into the markup on every render, because refreshContent's morph strips any
// attribute the new markup lacks. A drag does change the DOM as it goes, but only while
// tblGesture is set, and refreshContent holds a poll's markup back until the gesture is over.
//
// A column is
//   { key, label, cls?: string | row => string, attrs?: row => string,
//     cell: row => html, sort?: row => value | true, firstDir?: 'asc' | 'desc' }
// sort is a function for a table sorted here; true marks a column the server sorts (the spec's
// onSort then fetches). A column without sort has a plain label.
// ─────────────────────────────────────────────────────────
const TBL_MIN_W      = 56    // narrowest a drag leaves a column
const TBL_FIT_MAX    = 640   // a double-click fit never grows a column past this
const TBL_DEFAULT_W  = 120   // width for a column a saved layout has not measured yet
const TBL_DRAG_START = 4     // pointer travel before a header press becomes a reorder

const tblSpecs    = new Map()   // id → the spec last rendered, for a re-render after a header action
const tblPrefsMem = new Map()   // id → prefs; still holds for this visit when storage is unavailable
let tblGesture        = null    // { kind, id } while a resize or reorder drag is running
let tblPendingRefresh = null    // markup a poll tried to paint during a gesture
let tblSuppressClick  = false   // swallows the click that ends a reorder drag on a sort button

// ── Prefs ────────────────────────────────────────────────
// { sort: { key, dir }, widths: { key: px }, order: [key] }, each part optional.
function tablePrefs(id) {
    if (!tblPrefsMem.has(id)) {
        let saved = {}
        try { saved = JSON.parse(localStorage.getItem(`tablePrefs:${id}`) || '{}') } catch { saved = {} }
        tblPrefsMem.set(id, tblCleanPrefs(saved))
    }
    return tblPrefsMem.get(id)
}

// saveTablePrefs merges patch into the table's prefs. persist=false keeps the change in memory
// only, for the moves of a drag; the drag writes storage once when it ends.
function saveTablePrefs(id, patch, persist = true) {
    const next = tblCleanPrefs({ ...tablePrefs(id), ...patch })
    tblPrefsMem.set(id, next)
    if (persist) tblStore(id, next)
}

function clearTablePrefs(id) {
    tblPrefsMem.set(id, {})
    tblStore(id, {})
}

function tblStore(id, prefs) {
    try {
        if (Object.keys(prefs).length) localStorage.setItem(`tablePrefs:${id}`, JSON.stringify(prefs))
        else localStorage.removeItem(`tablePrefs:${id}`)
    } catch { /* private window or blocked storage: the in-memory copy covers this visit */ }
}

// tblCleanPrefs drops anything that is not the shape a render expects, so a hand-edited or
// stale entry cannot break a table.
function tblCleanPrefs(p) {
    const out = {}
    if (!p || typeof p !== 'object') return out
    if (p.sort && typeof p.sort.key === 'string' && (p.sort.dir === 'asc' || p.sort.dir === 'desc')) {
        out.sort = { key: p.sort.key, dir: p.sort.dir }
    }
    if (p.widths && typeof p.widths === 'object') {
        const w = {}
        for (const [k, v] of Object.entries(p.widths)) if (Number.isFinite(v) && v > 0) w[k] = Math.round(v)
        if (Object.keys(w).length) out.widths = w
    }
    if (Array.isArray(p.order)) {
        const o = p.order.filter(k => typeof k === 'string')
        if (o.length) out.order = o
    }
    return out
}

// ── Model ────────────────────────────────────────────────
// tblColumns is the spec's columns in the saved order. A column the saved order does not name
// (new since it was saved) keeps its default slot.
function tblColumns(spec) {
    const order = tablePrefs(spec.id).order || []
    const byKey = new Map(spec.columns.map(c => [c.key, c]))
    const cols = [...new Set(order)].filter(k => byKey.has(k)).map(k => byKey.get(k))
    spec.columns.forEach((c, i) => { if (!cols.includes(c)) cols.splice(Math.min(i, cols.length), 0, c) })
    return cols
}

// tableSortState is the sort in force: the saved one while its column still sorts, else the
// spec's default, else none (rows stay in the order they came).
function tableSortState(spec) {
    const sortable = s => s && spec.columns.some(c => c.key === s.key && c.sort)
    const saved = tablePrefs(spec.id).sort
    if (sortable(saved)) return saved
    return sortable(spec.sort) ? spec.sort : null
}

const tblCollator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

function tblSortValue(v) {
    if (v === null || v === undefined || v === '' || Number.isNaN(v)) return null
    if (v instanceof Date) return v.getTime()
    if (typeof v === 'boolean') return v ? 1 : 0
    return v
}

// tblTime is a sort value for an ISO timestamp.
function tblTime(iso) { return iso ? Date.parse(iso) : null }

// tblSortRows orders a copy of the rows. Blanks go last in both directions; ties keep the order
// the rows came in, so a server's own ordering is the tiebreaker.
function tblSortRows(spec, sort) {
    const col = sort && spec.columns.find(c => c.key === sort.key)
    if (!col || typeof col.sort !== 'function') return spec.rows
    const dir = sort.dir === 'desc' ? -1 : 1
    return spec.rows
        .map((r, i) => ({ r, i, v: tblSortValue(col.sort(r)) }))
        .sort((a, b) => {
            if (a.v === null || b.v === null) return a.v === b.v ? a.i - b.i : a.v === null ? 1 : -1
            const c = typeof a.v === 'number' && typeof b.v === 'number'
                ? a.v - b.v
                : tblCollator.compare(String(a.v), String(b.v))
            return c * dir || a.i - b.i
        })
        .map(x => x.r)
}

// ── Markup ───────────────────────────────────────────────
// dataTable is the table card: toolbar, table, foot. spec is
//   { id, columns, rows, tr?: row => attrs, menu?: row => items, sort?: { key, dir },
//     onSort?: sort => void, toolbar?, empty?, foot? }
// menu gives each row a kebab in the trailing column instead of an Actions column; see tblRowItems.
// id must be stable: it keys the saved layout.
function dataTable(spec) {
    const { toolbar = '', foot = '' } = spec
    const bar = toolbar ? `<div class="toolbar">${toolbar}</div>` : ''
    if (!spec.rows.length) {
        tblSpecs.set(spec.id, spec)
        return `<div class="card">${bar}${tblEmpty(spec)}</div>`
    }
    return `<div class="card">${bar}${tableWrap(spec)}${foot ? `<div class="card-foot">${foot}</div>` : ''}</div>`
}

function tblEmpty(spec) {
    return spec.empty ?? emptyState('Nothing here yet', 'Items you create will show up in this table.')
}

// tableWrap is the .table-wrap and its table, or the empty state when there are no rows.
function tableWrap(spec) {
    tblSpecs.set(spec.id, spec)
    if (!spec.rows.length) return tblEmpty(spec)

    const prefs  = tablePrefs(spec.id)
    const cols   = tblColumns(spec)
    const sort   = tableSortState(spec)
    const rows   = spec.onSort ? spec.rows : tblSortRows(spec, sort)
    const widths = prefs.widths

    // With a saved layout the table is fixed-layout and every column has its width, except the
    // last, which takes whatever is left so the table still fills the card.
    const colgroup = cols.map((c, i) => {
        const w = widths && i < cols.length - 1 ? ` style="width:${widths[c.key] ?? TBL_DEFAULT_W}px"` : ''
        return `<col data-col="${esc(c.key)}"${w}>`
    }).join('') + '<col class="col-menu">'
    const table = widths
        ? `<table class="tbl is-sized" style="width:${tblTableWidth(cols.map(c => c.key), widths)}">`
        : '<table class="tbl">'

    spec.shown = rows   // a row kebab's data-row indexes this, the order the rows are drawn in
    const head = cols.map(c => tblTH(c, sort)).join('') +
        `<th class="th-menu"><button type="button" class="icon-btn" data-table-menu aria-haspopup="menu" aria-label="Column options">${icon('dots')}</button></th>`
    const body = rows.map((r, i) => `<tr${spec.tr ? ` ${spec.tr(r)}` : ''}>${
        cols.map(c => `<td${tblCellAttrs(c, r)}>${c.cell(r)}</td>`).join('')}<td class="td-menu">${
        tblRowItems(spec, r).length ? `<button type="button" class="icon-btn" data-row-menu="${i}" aria-haspopup="menu" aria-label="Row actions">${icon('dots')}</button>` : ''}</td></tr>`).join('')

    return `<div class="table-wrap" data-table="${esc(spec.id)}">${table}
        <colgroup>${colgroup}</colgroup>
        <thead><tr>${head}</tr></thead>
        <tbody>${body}</tbody>
    </table></div>`
}

// tblTableWidth fills the card, or grows past it (and scrolls) when the saved widths add up to more.
function tblTableWidth(keys, widths) {
    const sum = keys.reduce((n, k) => n + (widths[k] ?? TBL_DEFAULT_W), 0)
    return `max(100%, calc(${sum}px + var(--tbl-menu-w)))`
}

// tblTH is a header cell. Headers and cells are all left-aligned, numbers included: a centred or
// right-aligned label would travel with the divider while its column is resized, and every cell
// starts where its label does.
function tblTH(c, sort) {
    const on    = sort && sort.key === c.key
    const aria  = on ? ` aria-sort="${sort.dir === 'desc' ? 'descending' : 'ascending'}"` : ''
    const glyph = on ? (sort.dir === 'desc' ? '↓' : '↑') : '↕'
    const label = c.sort
        ? `<button type="button" class="th-sort" data-sort="${esc(c.key)}">${esc(c.label)}<span class="sort" aria-hidden="true">${glyph}</span></button>`
        : esc(c.label)
    return `<th data-col="${esc(c.key)}"${aria}>${label}<span class="col-resize" aria-hidden="true"></span></th>`
}

// tblRowItems is a row's kebab menu: spec.menu(row) in buildMenu's item shape, where an item
// marked edit: true is dropped for a viewer. No items, no kebab.
function tblRowItems(spec, r) {
    return (spec.menu?.(r) || []).filter(it => it === '-' || !it.edit || canEdit())
}

function tblCellAttrs(c, r) {
    const cls = typeof c.cls === 'function' ? c.cls(r) : c.cls
    return `${cls ? ` class="${cls}"` : ''}${c.attrs ? ` ${c.attrs(r)}` : ''}`
}

// ── Re-render ────────────────────────────────────────────
function tblEl(id) {
    return document.querySelector(`#content .table-wrap[data-table="${CSS.escape(id)}"]`)
}

// tableRerender redraws one table from its last spec, patched in place like a poll.
function tableRerender(id) {
    const spec = tblSpecs.get(id), el = tblEl(id)
    if (!spec || !el) return
    const t = document.createElement('template')
    t.innerHTML = tableWrap(spec)
    if (t.content.firstElementChild) morphNode(el, t.content.firstElementChild)
    tableStickHeads()
}

function tableSortBy(id, key) {
    const spec = tblSpecs.get(id)
    const col  = spec?.columns.find(c => c.key === key)
    if (!col?.sort) return
    const cur = tableSortState(spec)
    const dir = cur && cur.key === key ? (cur.dir === 'asc' ? 'desc' : 'asc') : (col.firstDir || 'asc')
    saveTablePrefs(id, { sort: { key, dir } })
    tableRerender(id)   // the header shows the new sort at once; a server sort then fetches
    spec.onSort?.({ key, dir })
}

function tableReset(id) {
    clearTablePrefs(id)
    tableRerender(id)
    const spec = tblSpecs.get(id)
    spec?.onSort?.(tableSortState(spec))
}

function tblMenuItems(id) {
    return [{
        label: 'Reset columns', icon: 'undo',
        disabled: !Object.keys(tablePrefs(id)).length,
        run: () => tableReset(id),
    }]
}

// tableGestureActive tells refreshContent to hold a poll back; tableGestureDefer is where it goes.
function tableGestureActive() { return !!tblGesture }
function tableGestureDefer(html) { tblPendingRefresh = html }

// tblEndGesture paints a poll that arrived mid-drag, then the table with its final layout.
function tblEndGesture(id) {
    tblGesture = null
    document.getElementById('app').classList.remove('is-dragging', 'drag-resize')
    const html = tblPendingRefresh
    tblPendingRefresh = null
    if (html !== null) refreshContent(html)
    tableRerender(id)
}

// ── Sticky header ────────────────────────────────────────
// .table-wrap scrolls sideways for a wide table, which makes it the scroll container a
// position:sticky header would stick inside, and it never scrolls vertically. So the header
// cells are moved by a transform instead: --thead-y is how far the page has scrolled past the
// wrap's top, clamped so the header never leaves its own table.
function tableStickHeads() {
    const bar = document.querySelector('.topbar')
    const top = bar && bar.offsetParent !== null ? bar.getBoundingClientRect().bottom : 0
    for (const wrap of document.querySelectorAll('#content .table-wrap')) {
        const head = wrap.querySelector(':scope > .tbl > thead')
        if (!head) continue
        const r = wrap.getBoundingClientRect()
        const y = Math.round(Math.max(0, Math.min(top - r.top, r.height - head.offsetHeight)))
        if (y > 0) wrap.style.setProperty('--thead-y', `${y}px`)
        else wrap.style.removeProperty('--thead-y')
    }
}
window.addEventListener('scroll', tableStickHeads, { passive: true })
window.addEventListener('resize', tableStickHeads)

// ── Resize ───────────────────────────────────────────────
// tblSnapshotWidths gives every column an explicit width, measured as it is laid out now, the
// first time a column is resized, so the other columns do not move when one changes.
function tblSnapshotWidths(id, wrap) {
    const saved  = tablePrefs(id).widths || {}
    const widths = { ...saved }
    for (const th of wrap.querySelectorAll('thead th[data-col]')) {
        if (!widths[th.dataset.col]) widths[th.dataset.col] = Math.round(th.getBoundingClientRect().width)
    }
    if (Object.keys(widths).length === Object.keys(saved).length) return
    saveTablePrefs(id, { widths }, false)
    tableRerender(id)
}

// tblPaintWidths is the live side of a resize: the colgroup and table width, from the prefs,
// the same way tableWrap writes them.
function tblPaintWidths(id) {
    const table = tblEl(id)?.querySelector('.tbl')
    const widths = tablePrefs(id).widths
    if (!table || !widths) return
    const cols = [...table.querySelectorAll('col[data-col]')]
    cols.forEach((col, i) => {
        col.style.width = i < cols.length - 1 ? `${widths[col.dataset.col] ?? TBL_DEFAULT_W}px` : ''
    })
    table.style.width = tblTableWidth(cols.map(c => c.dataset.col), widths)
}

// The press is not prevented (that could cost the double-click that fits the column); the shell's
// .is-dragging, set before the mousedown that follows, is what stops it starting a text selection.
function tblStartResize(e, wrap, th) {
    const id = wrap.dataset.table, key = th.dataset.col
    tblSnapshotWidths(id, wrap)
    const startW = tablePrefs(id).widths[key], startX = e.clientX
    tblGesture = { kind: 'resize', id }
    document.getElementById('app').classList.add('is-dragging', 'drag-resize')
    th.querySelector('.col-resize')?.classList.add('is-active')   // the closing re-render drops it

    const move = ev => {
        const w = Math.max(TBL_MIN_W, Math.round(startW + ev.clientX - startX))
        saveTablePrefs(id, { widths: { ...tablePrefs(id).widths, [key]: w } }, false)
        tblPaintWidths(id)
    }
    const up = () => {
        window.removeEventListener('pointermove', move)
        window.removeEventListener('pointerup', up)
        window.removeEventListener('pointercancel', up)
        saveTablePrefs(id, {})
        tblEndGesture(id)
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
    window.addEventListener('pointercancel', up)
}

// tblFit sizes a column to its widest cell: the table is laid out at its natural width for a
// moment, with this column's own width taken off, and the header cell measured.
function tblFit(wrap, key) {
    const id = wrap.dataset.table
    tblSnapshotWidths(id, wrap)
    const table = wrap.querySelector('.tbl')
    const col   = table.querySelector(`col[data-col="${CSS.escape(key)}"]`)
    const th    = table.querySelector(`thead th[data-col="${CSS.escape(key)}"]`)
    if (!col || !th) return
    const was = col.style.width
    col.style.width = ''
    table.classList.add('is-measuring')
    const w = th.getBoundingClientRect().width
    table.classList.remove('is-measuring')
    col.style.width = was
    saveTablePrefs(id, { widths: { ...tablePrefs(id).widths, [key]: Math.min(TBL_FIT_MAX, Math.max(TBL_MIN_W, Math.ceil(w))) } })
    tableRerender(id)
}

// ── Reorder ──────────────────────────────────────────────
// A press on a header becomes a drag once it travels; until then it is a click (and sorts).
function tblArmReorder(e, wrap, th) {
    const startX = e.clientX, startY = e.clientY
    let drag = null
    const move = ev => {
        if (!drag) {
            if (Math.abs(ev.clientX - startX) < TBL_DRAG_START && Math.abs(ev.clientY - startY) < TBL_DRAG_START) return
            drag = tblBeginReorder(wrap, th)
        }
        tblMoveReorder(drag, ev)
    }
    const up = ev => {
        window.removeEventListener('pointermove', move)
        window.removeEventListener('pointerup', up)
        window.removeEventListener('pointercancel', up)
        if (!drag) return
        tblSuppressClick = true
        setTimeout(() => { tblSuppressClick = false }, 0)
        tblFinishReorder(drag, ev.type === 'pointerup')
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
    window.addEventListener('pointercancel', up)
}

function tblBeginReorder(wrap, th) {
    const id = wrap.dataset.table
    tblGesture = { kind: 'reorder', id }
    document.getElementById('app').classList.add('is-dragging')
    window.getSelection()?.removeAllRanges()
    th.classList.add('is-col-source')
    const ghost = Object.assign(document.createElement('div'), { className: 'tbl-ghost', textContent: th.textContent.replace(/[↑↓↕]/g, '').trim() })
    const line  = Object.assign(document.createElement('div'), { className: 'tbl-drop' })
    document.body.append(ghost, line)
    const keys = [...wrap.querySelectorAll('thead th[data-col]')].map(t => t.dataset.col)
    return { id, wrap, th, key: th.dataset.col, keys, ghost, line, slot: keys.indexOf(th.dataset.col) }
}

// tblMoveReorder follows the pointer with the ghost and puts the drop line on the column
// boundary nearest to it.
function tblMoveReorder(d, ev) {
    d.ghost.style.left = `${ev.clientX + 12}px`
    d.ghost.style.top  = `${ev.clientY + 12}px`
    const ths = [...d.wrap.querySelectorAll('thead th[data-col]')]
    const rects = ths.map(t => t.getBoundingClientRect())
    let slot = rects.findIndex(r => ev.clientX < r.left + r.width / 2)
    if (slot < 0) slot = rects.length
    d.slot = slot
    const wr   = d.wrap.getBoundingClientRect()
    const edge = slot < rects.length ? rects[slot].left : rects[rects.length - 1].right
    const top  = Math.max(wr.top, rects[0].top)
    d.line.style.left   = `${Math.min(Math.max(edge, wr.left), wr.right)}px`
    d.line.style.top    = `${top}px`
    d.line.style.height = `${Math.max(0, Math.min(wr.bottom, window.innerHeight) - top)}px`
}

function tblFinishReorder(d, drop) {
    d.ghost.remove()
    d.line.remove()
    d.th.classList.remove('is-col-source')
    const from = d.keys.indexOf(d.key)
    const to   = d.slot > from ? d.slot - 1 : d.slot
    if (drop && to !== from) {
        const keys = d.keys.filter(k => k !== d.key)
        keys.splice(to, 0, d.key)
        saveTablePrefs(d.id, { order: keys })
    }
    tblEndGesture(d.id)
}

// ── Wiring ───────────────────────────────────────────────
// Delegated once, so a table needs no handlers in its markup and a morph cannot drop them.
document.addEventListener('click', e => {
    if (!tblSuppressClick) return
    tblSuppressClick = false
    e.preventDefault()
    e.stopPropagation()
}, true)

document.addEventListener('click', e => {
    const wrap = e.target.closest('#content .table-wrap[data-table]')
    if (!wrap) return
    const sortBtn = e.target.closest('thead .th-sort')
    if (sortBtn) { tableSortBy(wrap.dataset.table, sortBtn.dataset.sort); return }
    const menuBtn = e.target.closest('thead [data-table-menu]')
    if (menuBtn) toggleMenu(menuBtn, tblMenuItems(wrap.dataset.table))
})

// A row kebab is caught in the capture phase, before the row's own onclick would open the record.
document.addEventListener('click', e => {
    const btn  = e.target.closest('#content .table-wrap[data-table] tbody [data-row-menu]')
    if (!btn) return
    e.stopPropagation()
    const spec = tblSpecs.get(btn.closest('.table-wrap').dataset.table)
    const row  = spec?.shown?.[Number(btn.dataset.rowMenu)]
    if (row) toggleMenu(btn, tblRowItems(spec, row))
}, true)

document.addEventListener('contextmenu', e => {
    const wrap = e.target.closest('#content .table-wrap[data-table]')
    if (!wrap || !e.target.closest('thead')) return
    e.preventDefault()
    openMenuAt(e.clientX, e.clientY, tblMenuItems(wrap.dataset.table))
})

document.addEventListener('pointerdown', e => {
    if (e.pointerType !== 'mouse' || e.button !== 0 || tblGesture) return
    const wrap = e.target.closest('#content .table-wrap[data-table]')
    const th   = e.target.closest('thead th[data-col]')
    if (!wrap || !th) return
    if (e.target.closest('.col-resize')) tblStartResize(e, wrap, th)
    else tblArmReorder(e, wrap, th)
})

document.addEventListener('dblclick', e => {
    const handle = e.target.closest('#content .table-wrap[data-table] thead .col-resize')
    if (handle) tblFit(handle.closest('.table-wrap'), handle.closest('th').dataset.col)
})
