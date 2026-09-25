// ─────────────────────────────────────────────────────────
// Workflows
//
// A workflow is a graph: trigger nodes accept ticket events, if nodes branch on a condition, action
// nodes do something, and edges wire an output port to a node's input. The editor is a pannable,
// zoomable canvas (cv*) with a step list docked left and an inspector docked right. Node positions
// are part of the document and are saved with it.
// ─────────────────────────────────────────────────────────
let wf           = null   // working copy of the workflow being edited
let wfOriginal   = ''     // JSON.stringify(wf) at load/save time, for dirty compare
let wfRecipients = []     // /webex/rooms cache (rooms + people)
let wfStatuses   = []     // statuses for this workflow's board
let wfMembers    = []     // /cw/members cache
let wfPriorities = []     // /cw/priorities cache (live from ConnectWise)
let wfPlaceholders = []   // /workflows/placeholders cache
let wfSimTicket  = ''     // last simulated ticket number, kept across re-renders
let wfSimAsNew   = false  // last simulated event

// Notify channels. Webex is the only transport today; a Slack or Teams room would be another
// channel here, not another step kind.
const WF_CHANNELS = [['webex_room', 'Webex room'], ['webex_person', 'Webex person'], ['resources_owner', 'Ticket resources & owner']]
// wfChannelRecipientType maps a channel to the recipient rows it picks from.
function wfChannelRecipientType(ch) { return ch === 'webex_person' ? 'person' : 'room' }
const WF_PATCH_EXAMPLE = '[\n  { "op": "replace", "path": "severity", "value": "High" }\n]'

// Node geometry, shared with ui.css: cards are 248 wide and a fixed height per kind, so the
// ports can be placed from constants. A card that sized itself from its content would put every
// wire endpoint off by the difference.
const CV_W = 248, CV_H = 88
const CV_ZOOM_MIN = 0.3, CV_ZOOM_MAX = 1.8

// CV_KINDS is what the canvas knows about each node kind: its label, tone, icon and where it sits
// in the step list. Every key is a NodeKind the server accepts.
const CV_KINDS = {
    trigger:      { label: 'Trigger',      tone: 't-start',  icon: 'bolt',   group: 'Triggers',      hint: 'Where a ticket event enters the flow' },
    if:           { label: 'If',           tone: 't-logic',  icon: 'branch', group: 'Logic',         hint: 'Branch on a condition' },
    skip_notify:  { label: 'Skip notify',  tone: 't-stop',   icon: 'ban',    group: 'Logic',         hint: 'Silence the notifies after it on this path' },
    notify:       { label: 'Notify',       tone: 't-notify', icon: 'bell',   group: 'Notify',        hint: 'Webex room, person or the ticket’s people' },
    add_note:     { label: 'Add note',     tone: 't-notify', icon: 'book',   group: 'Notify',        hint: 'Write a ticket note' },
    set_status:   { label: 'Set status',   tone: 't-write',  icon: 'edit',   group: 'Ticket writes', hint: 'Move the ticket to a status' },
    set_priority: { label: 'Set priority', tone: 't-write',  icon: 'edit',   group: 'Ticket writes', hint: 'Re-rank the ticket' },
    set_owner:    { label: 'Set owner',    tone: 't-write',  icon: 'edit',   group: 'Ticket writes', hint: 'Assign the ticket' },
    add_resource: { label: 'Add resource', tone: 't-write',  icon: 'edit',   group: 'Ticket writes', hint: 'Append a member to the resources' },
    patch:        { label: 'Patch',        tone: 't-write',  icon: 'edit',   group: 'Ticket writes', hint: 'Raw ConnectWise JSON patch' },
}
const CV_GROUPS = ['Triggers', 'Logic', 'Notify', 'Ticket writes']

// cvEditable is canEdit() minus replay: the results page shows a recorded run on a canvas that
// pans and zooms but never changes.
function cvEditable() { return canEdit() && !cv.replay }

function cvKind(kind) { return CV_KINDS[kind] || { label: kind, tone: 't-write', icon: 'edit', group: '', hint: '' } }
function cvIsAction(kind) { return kind !== 'trigger' && kind !== 'if' }

async function loadWorkflows(sub) {
    if (sub && /^\d+$/.test(sub)) {
        await loadWorkflowEditor(parseInt(sub))
        return
    }
    const m = sub && sub.match(/^results(?:\/([^?]+))?(\?.*)?$/)
    if (m) {
        if (m[1]) await loadWorkflowRun(m[1])
        else await loadWorkflowResults(m[2] ? m[2].slice(1) : '')
        return
    }
    await loadWorkflowList()
}

// ── List ─────────────────────────────────────────────────
async function loadWorkflowList() {
    try {
        const list = await api('GET', '/workflows')
        renderWorkflowList(list || [])
    } catch (e) {
        setContent(errorState(e.message))
    }
}

let wfList = []   // last-loaded /workflows, for delete confirmations

function renderWorkflowList(list) {
    wfList = list
    const banner = appConfig?.master_dry_run
        ? `<div class="banner warn">${icon('alert')}<div><b>Master dry run is on.</b> Every workflow runs as a dry run — nothing is written or sent. Turn it off under Config.</div></div>`
        : ''

    const thead = `<th>Board</th><th class="c">Enabled</th><th>Mode</th><th class="r">Steps</th><th class="r">Actions</th>`
    const rows  = list.map(w => `<tr class="clickable" onclick="openWorkflow(${w.id})">
        <td>
            <div class="cell-primary">${esc(w.board_name || w.name)}</div>
            ${w.name && w.name !== w.board_name ? `<div class="cell-sub">${esc(w.name)}</div>` : ''}
        </td>
        <td class="c">${w.enabled ? badgeTag('Enabled', 'ok') : badgeTag('Disabled', '')}</td>
        <td>${w.dry_run ? badgeTag('Dry run', 'warn') : (appConfig?.master_dry_run ? badgeTag('Dry run (master)', 'warn') : badgeTag('Live', 'ok'))}</td>
        <td class="r num">${(w.nodes || []).length}</td>
        <td class="r nowrap" onclick="event.stopPropagation()">
            <button class="btn btn-ghost btn-sm" onclick="openWorkflow(${w.id})">${icon('edit')}Open</button>
            ${deleteButton(`deleteWorkflow(${w.id})`)}
        </td>
    </tr>`)

    setContent(wfTabs('workflows') + pageActions(
        (list.length ? `<button class="btn btn-default" onclick="wfExportAll()">${icon('download')}Export</button>` : '') +
        editOnly(`<button class="btn btn-default" onclick="wfImportPick()">${icon('up')}Import</button>`) +
        editOnly(`<button class="btn btn-primary" onclick="showNewWorkflowModal()">${icon('plus')}New workflow</button>`)) +
    banner +
    tableCard(thead, rows, {
        empty: emptyState('No workflows yet',
            'Create a workflow for a board and ticketbot will start acting on its tickets.',
            editOnly(`<button class="btn btn-primary btn-sm" onclick="showNewWorkflowModal()">${icon('plus')}New workflow</button>`), 'bolt'),
        foot: `<span>${list.length} workflow${list.length === 1 ? '' : 's'}</span>`,
    }))
}

// ── Export / import ──────────────────────────────────────
// A bundle carries the workflows plus the recipients and lists they name, keyed by this
// instance's ids; import matches recipients by Webex id and lists by name, creating what is
// missing, and rewrites the references. It is how workflows move to a fresh instance.
async function wfExportAll() {
    try {
        const res = await fetch('/workflows/export', { credentials: 'same-origin' })
        if (!res.ok) throw new Error((await res.json().catch(() => ({})))?.error || res.statusText)
        const blob = await res.blob()
        const name = (res.headers.get('Content-Disposition') || '').match(/filename="([^"]+)"/)?.[1] || 'ticketbot-workflows.json'
        const a = Object.assign(document.createElement('a'), { href: URL.createObjectURL(blob), download: name })
        document.body.appendChild(a); a.click(); a.remove()
        setTimeout(() => URL.revokeObjectURL(a.href), 1000)
    } catch (e) { toast(e.message, 'error') }
}

function wfImportPick() {
    const input = Object.assign(document.createElement('input'), { type: 'file', accept: 'application/json,.json' })
    input.onchange = async () => {
        const file = input.files?.[0]
        if (!file) return
        let bundle
        try { bundle = JSON.parse(await file.text()) } catch { toast('That file is not JSON', 'error'); return }
        wfImportConfirm(bundle, file.name)
    }
    input.click()
}

function wfImportConfirm(bundle, fileName) {
    const n = bundle?.workflows?.length || 0
    openModal('Import workflows', `
        <div class="stack gap4">
            <p>${esc(fileName)} holds <b>${n}</b> workflow${n === 1 ? '' : 's'}, ${bundle?.recipients?.length || 0} recipient${(bundle?.recipients?.length || 0) === 1 ? '' : 's'} and ${bundle?.lists?.length || 0} list${(bundle?.lists?.length || 0) === 1 ? '' : 's'}. Recipients and lists that already exist here are reused; missing ones are created.</p>
            ${checkbox('Replace a board\u2019s existing workflow instead of skipping it', 'id="f-import-replace"')}
        </div>`, async () => {
        try {
            const rep = await api('POST', '/workflows/import', { bundle, replace: document.getElementById('f-import-replace').checked })
            closeModal()
            wfImportReport(rep)
            loadWorkflowList()
        } catch (e) { toast(e.message, 'error') }
    }, 'Import')
}

function wfImportReport(rep) {
    const tone = { created: 'ok', replaced: 'ok', skipped: 'warn', error: 'bad' }
    const rows = (rep.workflows || []).map(w => `<tr>
        <td class="cell-primary">${esc(w.board_name || w.name || `Board ${w.board_id}`)}${w.name && w.name !== w.board_name ? `<div class="cell-sub">${esc(w.name)}</div>` : ''}</td>
        <td>${badgeTag(w.result, tone[w.result] || '')}</td>
        <td class="muted">${esc(w.error || '')}</td>
    </tr>`).join('')
    const created = [
        rep.recipients_created?.length ? `Created recipients: ${rep.recipients_created.map(esc).join(', ')}.` : '',
        rep.lists_created?.length ? `Created lists: ${rep.lists_created.map(esc).join(', ')}.` : '',
    ].filter(Boolean).join(' ')
    openModal('Import finished', `
        <div class="stack gap4">
            ${created ? `<p>${created}</p>` : ''}
            <div class="table-wrap"><table class="tbl"><thead><tr><th>Workflow</th><th>Result</th><th>Detail</th></tr></thead><tbody>${rows}</tbody></table></div>
            ${rep.warnings?.length ? `<div class="callout warn">${icon('alert')}<div class="body">${rep.warnings.map(esc).join('<br>')}</div></div>` : ''}
        </div>`, async () => closeModal(), 'Done')
}

function openWorkflow(id) {
    switchTab('workflows', String(id))
}

// showNewWorkflowModal lets the admin pick a board without a workflow; preselect chooses one up front
// (used by the "Create workflow" button on a ticket). A new workflow starts with one trigger so the
// canvas has somewhere to begin.
async function showNewWorkflowModal(preselect = null) {
    let boards, existing
    try {
        ;[boards, existing] = await Promise.all([api('GET', '/cw/boards'), api('GET', '/workflows')])
    } catch (e) { toast(e.message, 'error'); return }

    const taken = new Set((existing || []).map(w => w.board_id))
    const free  = (boards || []).filter(b => !b.deleted && !taken.has(b.id))
    if (!free.length) { toast('Every board already has a workflow', 'error'); return }

    openModal('New workflow', `
        <div class="field">
            <label for="wf-new-board">Board</label>
            <select class="select" id="wf-new-board">${free.map(b => `<option value="${b.id}"${b.id === preselect ? ' selected' : ''}>${esc(b.name)}</option>`).join('')}</select>
            <span class="hint">A workflow starts with one trigger and nothing wired to it. Add steps on the canvas and save.</span>
        </div>`,
    async () => {
        const boardID = parseInt(document.getElementById('wf-new-board').value)
        try {
            const trigger = cvNewNode('trigger', 0, 0)
            const created = await api('POST', '/workflows', { board_id: boardID, nodes: [wfNodeForServer(trigger)], edges: [] })
            closeModal()
            openWorkflow(created.id)
        } catch (e) { toast(e.message, 'error') }
    })
}

function deleteWorkflow(id) {
    const w = wfList.find(x => x.id === id)
    const name = w ? (w.board_name || w.name) : `workflow ${id}`
    const steps = w?.nodes?.length || 0
    confirmModal({
        title: 'Delete this workflow?',
        body: `<b>${esc(name)}</b>Tickets on this board stop being processed: no notifications, notes or ticket updates.${
            steps ? ` Its ${steps} step${steps === 1 ? '' : 's'} ${steps === 1 ? 'is' : 'are'} deleted too.` : ''}`,
        confirmLabel: 'Delete workflow',
        onConfirm: async () => {
            try {
                await api('DELETE', `/workflows/${id}`)
                toast('Workflow deleted', 'success')
                loadWorkflowList()
            } catch (e) { toast(e.message, 'error') }
        },
    })
}

// ── Editor ───────────────────────────────────────────────
async function loadWorkflowEditor(id) {
    try {
        const [w, recips, members, placeholders, fields, boards, lists] = await Promise.all([
            api('GET', `/workflows/${id}`), api('GET', '/webex/rooms'), api('GET', '/cw/members'), api('GET', '/workflows/placeholders'),
            api('GET', '/workflows/fields'), api('GET', '/cw/boards'), api('GET', '/lists').catch(() => []),
        ])
        wfRecipients   = recips || []
        wfMembers      = (members || []).filter(m => !m.deleted)
        wfPlaceholders = placeholders || []
        wfFields       = fields || []
        wfBoards       = boards || []
        wfLists        = lists || []
        // statuses are board-scoped; priorities come live from ConnectWise and may be slow or fail
        const [statuses, priorities] = await Promise.all([
            api('GET', `/cw/boards/${w.board_id}/statuses`).catch(() => []),
            api('GET', '/cw/priorities').catch(e => { toast(`Priorities unavailable: ${e.message}`, 'error'); return [] }),
        ])
        wfStatuses   = (statuses || []).filter(s => !s.deleted && !s.inactive)
        wfPriorities = priorities || []
        wf = wfNormalize(w)
        const ifs = wf.nodes.filter(n => n.kind === 'if' || n.kind === 'trigger')
        await Promise.all(ifs.map(wfLoadNodeUI))
        await wfResolveNames(ifs)
        wfOriginal = JSON.stringify(wfStrip(wf))
        tabGuard = wfGuard
        cvReset()
        renderWorkflowEditor()
        cvFit()
        setCrumbHere(wf.board_name || wf.name)
    } catch (e) {
        setContent(backRow('workflows', 'Workflows') + errorState(e.message))
    }
}

function wfNormalize(w) {
    w.nodes = w.nodes || []
    w.edges = w.edges || []
    for (const n of w.nodes) {
        n.x = Math.round(n.x || 0)
        n.y = Math.round(n.y || 0)
        if (n.kind === 'if' || n.kind === 'trigger') n.condition = n.condition || ''
        if (n.kind === 'trigger') n.events = n.events || []
    }
    return w
}

// wfStrip returns a copy without editor-only state (the condition builder's rows).
function wfStrip(w) {
    const copy = JSON.parse(JSON.stringify(w))
    for (const n of copy.nodes) delete n._ui
    return copy
}

function wfIsDirty() {
    return wf !== null && JSON.stringify(wfStrip(wf)) !== wfOriginal
}

// wfGuard vetoes navigation while the editor has unsaved edits, then asks. The
// question is a modal, so it cannot answer in time: it re-runs `retry` itself
// once the user chooses to discard.
function wfGuard(retry) {
    if (!wfIsDirty()) return true
    confirmModal({
        title: 'Discard unsaved changes?',
        body: '<b>This workflow has edits that were never saved</b>Leaving now loses them.',
        confirmLabel: 'Discard changes',
        onConfirm: () => { tabGuard = null; retry?.() },
    })
    return false
}
wfGuard.isDirty = wfIsDirty

function wfNode(id) { return wf?.nodes.find(n => n.id === id) || null }
function wfEdge(id) { return wf?.edges.find(e => e.id === id) || null }

function renderWorkflowEditor() {
    const dirty = wfIsDirty()

    // No head and no trail: the topbar crumbs already say Workflows / this board,
    // and the canvas wants the height. What the blurb used to explain now lives in
    // the canvas hint bar, and saving rides at the end of the toolbar.
    setContent(`<div class="wf-page">${appConfig?.master_dry_run
        ? `<div class="banner warn">${icon('alert')}<div><b>Master dry run is on.</b> This workflow will not write to ConnectWise or send Webex messages, whatever its own dry-run setting says.</div></div>`
        : ''}

    <div class="toolbar wf-toolbar">
        <input class="input wf-name" type="text" value="${esc(wf.name)}" oninput="wfSet('name', this.value)" aria-label="Workflow name" data-tip="Workflow name, shown in the ticket history" data-tip-pos="bottom">
        ${toggle(`onchange="wfSet('enabled', this.checked)"`, wf.enabled, { label: 'Enabled' })}
        ${toggle(`onchange="wfSet('dry_run', this.checked)"`, wf.dry_run, { label: 'Dry run' })}
        <span class="grow"></span>
        <span class="cell-sub">Simulate</span>
        <input type="number" id="sim-ticket" class="input wf-ticket" placeholder="Ticket #" min="1" value="${esc(wfSimTicket)}" aria-label="Ticket number to simulate">
        <select id="sim-mode" class="select" style="width:auto" aria-label="Simulate as">
            <option value="update"${wfSimAsNew ? '' : ' selected'}>as an update</option>
            <option value="create"${wfSimAsNew ? ' selected' : ''}>as a new ticket</option>
        </select>
        <button class="btn btn-default btn-sm" onclick="wfSimulate()">${icon('play')}Run</button>
        <span id="wf-arrange-undo">${cv.undo ? `<button class="btn btn-ghost btn-sm" onclick="cvUndoArrange()">${icon('undo')}Undo arrange</button>` : ''}</span>
        <button class="btn btn-default btn-sm" onclick="cvArrange()">${icon('branch')}Auto-arrange</button>
        <button class="btn btn-default btn-sm" onclick="switchTab('workflows', 'results?board=${wf.board_id}')" data-tip="Runs of this workflow" data-tip-pos="bottom">${icon('chart')}Results</button>
        <button class="icon-btn" onclick="wfShowHelp()" aria-label="How workflows run" data-tip="How workflows run" data-tip-pos="bottom">${icon('info')}</button>
        <span id="wf-dirty" class="row gap2${dirty ? '' : ' hidden'}">
            <span class="dirty-dot"></span><span class="cell-sub">Unsaved changes</span>
        </span>
        ${editOnly(`<button id="wf-save" class="btn ${dirty ? 'btn-primary' : 'btn-default'}" onclick="saveWorkflow()" ${dirty ? '' : 'disabled'}>Save</button>`)}
    </div>

    <div class="canvas wf-canvas" id="cv">
        <div class="plane" id="cv-plane">
            <svg class="wires" id="cv-wires" viewBox="-4000 -4000 8000 8000" aria-hidden="true"></svg>
            <div id="cv-nodes"></div>
        </div>
        <div id="cv-rail"></div>
        <div id="cv-side"></div>
        <div class="dock-bar bl">
            <button class="icon-btn" onclick="cvZoomBy(0.8)" aria-label="Zoom out">${icon('minus')}</button>
            <span class="pct" id="cv-pct">100%</span>
            <button class="icon-btn" onclick="cvZoomBy(1.25)" aria-label="Zoom in">${icon('plus')}</button>
            <button class="icon-btn" onclick="cvFit()" aria-label="Fit the whole flow in view">${icon('fit')}</button>
        </div>
        <div class="dock-bar br" id="cv-hint"></div>
    </div>
    </div>`)

    cvMount()
    cvRenderGraph()
    cvRenderRail()
    cvRenderSide()
    cvApplyView()
}

// ── Canvas state ─────────────────────────────────────────
const cv = {
    el: null, zoom: 1, tx: 40, ty: 40,
    sel: null,       // selected node id
    multi: new Set(),// further selected node ids (shift-click, shift-drag box); sel is among them when set
    selEdge: null,   // selected edge id
    drag: null,      // { type: pan|node|link|pal, ... }
    rail: true,      // step list open
    run: null,       // simulation or recorded run { ticket, asNew, res, steps, step, record }
    replay: false,   // results page: the canvas shows a recorded run and cannot be edited
    timers: [],
    undo: null,      // node positions before the last auto-arrange
    errNode: null,   // node the last server validation error pointed at
    ghost: null,     // { kind, x, y } while a palette drag is over the canvas
    link: null,      // { x, y } pointer position in plane space during a link drag
}

function cvReset() {
    cvClearTimers()
    Object.assign(cv, { el: null, zoom: 1, tx: 40, ty: 40, sel: null, multi: new Set(), selEdge: null, drag: null, run: null, undo: null, errNode: null, ghost: null, link: null, replay: false })
}

function cvClearTimers() {
    cv.timers.forEach(clearTimeout)
    cv.timers = []
}

function cvPorts(n) { return n.kind === 'if' ? ['yes', 'no'] : ['out'] }
function cvOutPt(n, port) {
    if (n.kind !== 'if') return { x: n.x + CV_W / 2, y: n.y + CV_H }
    return port === 'yes' ? { x: n.x + 62, y: n.y + CV_H } : { x: n.x + 186, y: n.y + CV_H }
}
function cvInPt(n) { return { x: n.x + CV_W / 2, y: n.y } }
function cvCurve(p, q) {
    const dy = Math.max(30, Math.abs(q.y - p.y) * 0.45)
    return `M ${p.x} ${p.y} C ${p.x} ${p.y + dy}, ${q.x} ${q.y - dy}, ${q.x} ${q.y}`
}

// cvPt converts a pointer event to plane coordinates.
function cvPt(e) {
    const r = cv.el.getBoundingClientRect()
    return { x: (e.clientX - r.left - cv.tx) / cv.zoom, y: (e.clientY - r.top - cv.ty) / cv.zoom }
}

function cvHitNode(p) {
    for (let i = wf.nodes.length - 1; i >= 0; i--) {
        const n = wf.nodes[i]
        if (p.x >= n.x && p.x <= n.x + CV_W && p.y >= n.y && p.y <= n.y + CV_H) return n
    }
    return null
}

// the docks sit over the canvas, so pointer events inside them are not canvas gestures
function cvInChrome(t) { return !!(t && t.closest && t.closest('.dock, .dock-bar')) }

// ── Mount and events ─────────────────────────────────────
function cvMount() {
    cv.el = document.getElementById('cv')
    cv.el.addEventListener('pointerdown', cvOnDown)
    cv.el.addEventListener('wheel', cvOnWheel, { passive: false })
    cv.el.addEventListener('click', cvOnClick)
    cv.el.addEventListener('dblclick', cvOnDblClick)
    cv.el.addEventListener('contextmenu', cvOnContext)
    window.addEventListener('pointermove', cvOnMove)
    window.addEventListener('pointerup', cvOnUp)
    window.addEventListener('pointercancel', cvOnUp)
    document.addEventListener('keydown', cvOnKey)
}

// cvUnmounted reports whether the canvas has left the page; the window listeners then stand down.
function cvUnmounted() {
    if (cv.el && document.body.contains(cv.el)) return false
    window.removeEventListener('pointermove', cvOnMove)
    window.removeEventListener('pointerup', cvOnUp)
    window.removeEventListener('pointercancel', cvOnUp)
    document.removeEventListener('keydown', cvOnKey)
    cv.el = null
    return true
}

function cvOnDown(e) {
    if (e.button !== 0) return
    const t = e.target
    if (cvInChrome(t)) return

    const port = t.closest?.('.port')
    if (port && !port.classList.contains('inp')) {
        if (!cvEditable()) return
        e.preventDefault()
        cvGrabPort(port.dataset.portOf, port.dataset.port, e)
        return
    }
    if (t.closest?.('.wirex, .wirehit')) return
    const node = t.closest?.('.node')
    if (node) {
        e.preventDefault()
        cvGrabNode(node.dataset.node, e)
        return
    }
    e.preventDefault()
    if (e.shiftKey && cvEditable()) {
        const p = cvPt(e)
        cv.drag = { type: 'box', x0: p.x, y0: p.y, x1: p.x, y1: p.y }
        cvDragClass('drag-box')
        return
    }
    cv.drag = { type: 'pan', sx: e.clientX, sy: e.clientY, otx: cv.tx, oty: cv.ty, moved: false }
    cv.el.classList.add('panning')
    cvDragClass('drag-pan')
}

// cvOnContext opens the step or canvas menu at the pointer. Shift keeps the browser's own menu.
function cvOnContext(e) {
    if (e.shiftKey || cvInChrome(e.target) || cv.replay) return
    e.preventDefault()
    const card = e.target.closest?.('.node')
    if (card) {
        const id = card.dataset.node
        if (!cvSelection().includes(id)) cvSelect(id)
        openMenuAt(e.clientX, e.clientY, cvNodeMenu())
        return
    }
    cvDeselect()
    cv.paste = cvPt(e)
    openMenuAt(e.clientX, e.clientY, [
        { label: cvClipboard ? `Paste ${cvClipboard.nodes.length} step${cvClipboard.nodes.length === 1 ? '' : 's'}` : 'Paste', icon: 'copy', disabled: !cvClipboard || !cvEditable(), run: () => cvPaste(cv.paste) },
        { label: 'Add a step…', icon: 'plus', disabled: !cvEditable(), run: () => { if (!cv.rail) cvToggleRail() } },
        '-',
        { label: 'Fit to view', icon: 'fit', run: cvFit },
    ])
}

// cvNodeMenu is the right-click menu for the selected step or steps.
function cvNodeMenu() {
    const ids = cvSelection()
    const n = ids.length
    const many = n > 1
    const label = s => many ? `${s} ${n} steps` : s
    const nodes = ids.map(wfNode).filter(Boolean)
    const anyOn = nodes.some(x => x.enabled)
    const items = []
    if (!many) items.push({ label: 'Open inspector', icon: 'edit', run: () => cvSelect(ids[0]) })
    if (cvEditable()) {
        items.push(
            { label: label('Duplicate'), icon: 'copy', run: cvDuplicateSelection },
            { label: label('Copy'), icon: 'copy', run: cvCopySelection },
            { label: `${anyOn ? 'Disable' : 'Enable'}${many ? ` ${n} steps` : ''}`, icon: anyOn ? 'ban' : 'check', run: () => cvSetSelectionEnabled(!anyOn) },
            '-',
            { label: label('Delete'), icon: 'trash', danger: true, run: cvDeleteSelection },
        )
    }
    return items
}

function cvOnClick(e) {
    const t = e.target
    const hit = t.closest?.('.wirehit')
    if (hit) { cvSelectEdge(hit.dataset.edge); return }
    if (t.closest?.('.wirex')) { cvCutEdge(); return }
}

// double-clicking a step in the list appends it below the selection
function cvOnDblClick(e) {
    const pal = e.target.closest?.('.pal')
    if (pal && canEdit()) cvAppend(pal.dataset.kind)
}

function cvOnKey(e) {
    if (cvUnmounted()) return
    const tag = document.activeElement?.tagName
    const typing = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || document.activeElement?.isContentEditable
    if (e.key === 'Escape') {
        if (cv.run && !cv.replay) cvClearRun()
        else if (cv.sel || cv.selEdge) cvDeselect()
        return
    }
    if ((e.key === 'Delete' || e.key === 'Backspace') && !typing && cvEditable()) {
        if (cv.selEdge) { e.preventDefault(); cvCutEdge() }
        else if (cv.multi.size > 1) { e.preventDefault(); cvDeleteSelection() }
        else if (cv.sel) { e.preventDefault(); cvDeleteNode(cv.sel) }
        return
    }
    if ((e.metaKey || e.ctrlKey) && !typing && cvEditable() && !e.shiftKey && !e.altKey) {
        const k = e.key.toLowerCase()
        if (k === 'c' && cvSelection().length) { e.preventDefault(); cvCopySelection() }
        else if (k === 'v' && cvClipboard) { e.preventDefault(); cvPaste(null) }
        else if (k === 'd' && cvSelection().length) { e.preventDefault(); cvDuplicateSelection() }
    }
}

function cvOnWheel(e) {
    if (cvInChrome(e.target)) return
    e.preventDefault()
    const r = cv.el.getBoundingClientRect()
    const mx = e.clientX - r.left, my = e.clientY - r.top
    if (e.ctrlKey || e.metaKey) {
        cvZoomAt(cv.zoom * Math.exp(-e.deltaY * 0.012), mx, my)
    } else {
        cv.tx -= e.deltaX
        cv.ty -= e.deltaY
        cvApplyView()
    }
}

function cvGrabNode(id, e) {
    const n = wfNode(id)
    if (!n) return
    if (e.shiftKey && cvEditable()) {
        // shift-click adds to or removes from the selection; the press can still become a group drag
        const set = new Set(cvSelection())
        if (set.has(id)) set.delete(id); else set.add(id)
        cvSetSelection([...set], { render: false })
    } else if (!cvSelection().includes(id)) {
        cvSelect(id, { render: false })
    } else {
        cv.selEdge = null
        if (cv.run && !cv.replay) { cvClearTimers(); cv.run = null }
    }
    // dragging any selected card moves the whole selection
    const ids = cvSelection().includes(id) ? cvSelection() : [id]
    const orig = {}
    for (const i of ids) { const m = wfNode(i); if (m) orig[i] = { x: m.x, y: m.y } }
    cv.drag = cvEditable()
        ? { type: 'node', id, ids, orig, sx: e.clientX, sy: e.clientY, moved: false }
        : { type: 'pan', sx: e.clientX, sy: e.clientY, otx: cv.tx, oty: cv.ty, moved: false, fromNode: true }
    cvRenderGraph()
    cvRenderSide()
    cvDragClass(cv.drag.type === 'node' ? 'drag-node' : 'drag-pan')
}

// ── Selection ────────────────────────────────────────────
// cvSelection lists the selected step ids: the shift-selected set, or the single inspected one.
function cvSelection() {
    if (cv.multi.size) return [...cv.multi]
    return cv.sel ? [cv.sel] : []
}

// cvSetSelection selects several steps. One step is a plain selection with the inspector; more
// hide the inspector behind a small "N steps" panel.
function cvSetSelection(ids, opts = {}) {
    ids = ids.filter(id => wfNode(id))
    cv.selEdge = null
    if (cv.run && !cv.replay) { cvClearTimers(); cv.run = null }
    if (ids.length <= 1) { cv.multi = new Set(); cv.sel = ids[0] || null }
    else { cv.multi = new Set(ids); cv.sel = ids[0] }
    if (opts.render === false) return
    cvRenderGraph()
    cvRenderSide()
}

// ── Clipboard ────────────────────────────────────────────
// The clipboard lives for the browser session and crosses workflows and boards; save-time
// validation catches a status that belongs to another board.
let cvClipboard = null   // { nodes: [...copies without editor state], edges: [...between them] }

function cvSnapshotSelection() {
    const ids = new Set(cvSelection())
    const nodes = wf.nodes.filter(n => ids.has(n.id)).map(n => { const c = JSON.parse(JSON.stringify(n)); delete c._ui; return c })
    const edges = wf.edges.filter(e => ids.has(e.from) && ids.has(e.to)).map(e => ({ ...e }))
    return nodes.length ? { nodes, edges } : null
}

function cvCopySelection() {
    const snap = cvSnapshotSelection()
    if (!snap) return
    cvClipboard = snap
    toast(`Copied ${snap.nodes.length} step${snap.nodes.length === 1 ? '' : 's'}`, 'success')
}

// cvPlace adds copies of snap.nodes and the wires between them, with new ids, so the group's
// layout is kept. at is the top-left of the group in plane coordinates; null offsets the
// originals so a copy never lands exactly on its source.
async function cvPlace(snap, at) {
    if (!snap || !cvEditable()) return
    const ids = {}
    for (const n of snap.nodes) ids[n.id] = cvNewID()
    const x0 = Math.min(...snap.nodes.map(n => n.x)), y0 = Math.min(...snap.nodes.map(n => n.y))
    const dx = at ? Math.round(at.x) - x0 : 32, dy = at ? Math.round(at.y) - y0 : 32
    const added = snap.nodes.map(n => ({ ...JSON.parse(JSON.stringify(n)), id: ids[n.id], x: n.x + dx, y: n.y + dy }))
    for (const e of snap.edges) wf.edges.push({ id: cvNewID(), from: ids[e.from], to: ids[e.to], port: e.port })
    wf.nodes.push(...added)
    const ifs = added.filter(n => n.kind === 'if' || n.kind === 'trigger')
    await Promise.all(ifs.map(wfLoadNodeUI))
    await wfResolveNames(ifs)
    cvSetSelection(added.map(n => n.id))
    wfMarkDirty()
}

function cvPaste(at = null) { return cvPlace(cvClipboard, at) }

function cvDuplicateSelection() { return cvPlace(cvSnapshotSelection(), null) }

function cvSetSelectionEnabled(on) {
    for (const id of cvSelection()) { const n = wfNode(id); if (n) n.enabled = on }
    cvRenderGraph()
    cvRenderSide()
    wfMarkDirty()
}

// cvDeleteSelection removes the selected steps and their wires, keeping the last trigger.
function cvDeleteSelection() {
    const ids = new Set(cvSelection())
    if (!ids.size) return
    const keepTrigger = wf.nodes.filter(n => n.kind === 'trigger' && !ids.has(n.id)).length === 0
    if (keepTrigger) {
        const first = wf.nodes.find(n => n.kind === 'trigger' && ids.has(n.id))
        if (first) { ids.delete(first.id); toast('Kept one trigger: a workflow needs at least one', 'info') }
    }
    wf.nodes = wf.nodes.filter(n => !ids.has(n.id))
    wf.edges = wf.edges.filter(e => !ids.has(e.from) && !ids.has(e.to))
    cv.multi = new Set()
    cv.sel = null
    cvRenderGraph()
    cvRenderSide()
    wfMarkDirty()
}

// Dragging an output port picks up whatever was attached to it, so pulling a link off and
// dropping it on empty canvas is how you disconnect.
function cvGrabPort(id, port, e) {
    const n = wfNode(id)
    if (!n) return
    const had = wf.edges.find(x => x.from === id && x.port === port)
    wf.edges = wf.edges.filter(x => !(x.from === id && x.port === port))
    cv.selEdge = null
    cv.link = cvPt(e)
    cv.drag = { type: 'link', from: id, port, had: !!had }
    cvDragClass('drag-link')
    cvRenderGraph()
}

// No preventDefault here: it would also cancel the click and dblclick the same press produces,
// and double-clicking a step is how it gets appended. .is-dragging keeps text unselected instead.
function cvGrabPal(kind, e) {
    if (!canEdit()) return
    cv.drag = { type: 'pal', kind, started: false }
    cv.ghost = null
}

function cvOnMove(e) {
    const d = cv.drag
    if (!d || cvUnmounted()) return

    if (d.type === 'pan') {
        cv.tx = d.otx + (e.clientX - d.sx)
        cv.ty = d.oty + (e.clientY - d.sy)
        d.moved = d.moved || Math.abs(e.clientX - d.sx) + Math.abs(e.clientY - d.sy) > 3
        cvApplyView()
        return
    }
    if (d.type === 'node') {
        if (!d.moved && Math.abs(e.clientX - d.sx) + Math.abs(e.clientY - d.sy) <= 3) return
        d.moved = true
        const dx = (e.clientX - d.sx) / cv.zoom, dy = (e.clientY - d.sy) / cv.zoom
        for (const id of d.ids) {
            const n = wfNode(id), o = d.orig[id]
            if (!n || !o) continue
            n.x = Math.round(o.x + dx); n.y = Math.round(o.y + dy)
            cvMoveNodeEls(n)
        }
        cvRenderWires()
        return
    }
    const p = cvPt(e)
    if (d.type === 'box') {
        d.x1 = p.x; d.y1 = p.y
        cvRenderMarquee()
        return
    }
    if (d.type === 'link') {
        cv.link = p
        cvRenderWires()
        return
    }
    if (d.type === 'pal') {
        // the gesture only becomes a drag once the pointer moves; a plain press stays a click
        if (!d.started) { d.started = true; cvDragClass('drag-copy') }
        const r = cv.el.getBoundingClientRect()
        const inside = e.clientX >= r.left && e.clientX <= r.right && e.clientY >= r.top && e.clientY <= r.bottom
        cv.ghost = inside ? { kind: d.kind, x: Math.round(p.x) - CV_W / 2, y: Math.round(p.y) - CV_H / 2 } : null
        cvRenderGhost()
    }
}

function cvOnUp(e) {
    const d = cv.drag
    if (!d || cvUnmounted()) return
    cv.drag = null
    cv.el.classList.remove('panning')
    cvDragClass(null)

    if (d.type === 'link') {
        const target = cvHitNode(cvPt(e))
        const dupe = target && wf.edges.some(x => x.from === d.from && x.port === d.port && x.to === target.id)
        // several links may land on one node: that is how two paths rejoin. A trigger has no
        // input, so it can never be a target.
        if (target && target.id !== d.from && target.kind !== 'trigger' && !dupe) {
            wf.edges.push({ id: cvNewID(), from: d.from, to: target.id, port: d.port })
        }
        cv.link = null
        cvRenderGraph()
        wfMarkDirty()
        return
    }
    if (d.type === 'pal') {
        if (cv.ghost) cvDrop(d.kind, cv.ghost.x, cv.ghost.y)
        cv.ghost = null
        cvRenderGhost()
        return
    }
    if (d.type === 'node') {
        if (d.moved) { cvRenderGraph(); wfMarkDirty() }
        return
    }
    if (d.type === 'box') {
        const x0 = Math.min(d.x0, d.x1), x1 = Math.max(d.x0, d.x1), y0 = Math.min(d.y0, d.y1), y1 = Math.max(d.y0, d.y1)
        const hit = wf.nodes.filter(n => n.x < x1 && n.x + CV_W > x0 && n.y < y1 && n.y + CV_H > y0).map(n => n.id)
        document.querySelector('#cv-nodes .marquee')?.remove()
        cvSetSelection(hit)
        return
    }
    if (d.type === 'pan' && !d.moved && !d.fromNode) {
        cvDeselect()
    }
}

// cvRenderMarquee draws the shift-drag selection box among the cards.
function cvRenderMarquee() {
    const d = cv.drag
    const host = document.getElementById('cv-nodes')
    if (!host || d?.type !== 'box') return
    let el = host.querySelector('.marquee')
    if (!el) { el = document.createElement('div'); el.className = 'marquee'; host.appendChild(el) }
    const x = Math.min(d.x0, d.x1), y = Math.min(d.y0, d.y1)
    el.style.left = `${x}px`; el.style.top = `${y}px`
    el.style.width = `${Math.abs(d.x1 - d.x0)}px`; el.style.height = `${Math.abs(d.y1 - d.y0)}px`
}

// cvDragClass marks the shell for the length of a gesture: no text selection, one cursor, and the
// docks stop taking pointer events so the drag passes over them.
function cvDragClass(kind) {
    const b = document.body
    b.classList.remove('is-dragging', 'drag-link', 'drag-copy', 'drag-box')
    if (kind) b.classList.add('is-dragging')
    if (kind === 'drag-link' || kind === 'drag-copy' || kind === 'drag-box') b.classList.add(kind)
    cvRenderHint()
}

// ── View ─────────────────────────────────────────────────
function cvApplyView() {
    const plane = document.getElementById('cv-plane')
    if (!plane) return
    plane.style.transform = `translate(${cv.tx}px, ${cv.ty}px) scale(${cv.zoom})`
    const pct = document.getElementById('cv-pct')
    if (pct) pct.textContent = `${Math.round(cv.zoom * 100)}%`
}

function cvZoomAt(z2, mx, my) {
    z2 = Math.min(CV_ZOOM_MAX, Math.max(CV_ZOOM_MIN, z2))
    const k = z2 / cv.zoom
    cv.tx = mx - (mx - cv.tx) * k
    cv.ty = my - (my - cv.ty) * k
    cv.zoom = z2
    cvApplyView()
}

function cvZoomBy(k) {
    if (!cv.el) return
    const r = cv.el.getBoundingClientRect()
    cvZoomAt(cv.zoom * k, r.width / 2, r.height / 2)
}

// cvFit frames every node in the space the open docks leave, so nothing lands underneath them.
function cvFit() {
    if (!cv.el || !wf?.nodes.length) return
    const r = cv.el.getBoundingClientRect()
    let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
    for (const n of wf.nodes) {
        x0 = Math.min(x0, n.x); y0 = Math.min(y0, n.y)
        x1 = Math.max(x1, n.x + CV_W); y1 = Math.max(y1, n.y + CV_H)
    }
    const left = (cv.rail ? 190 + 12 : 12) + 24
    const right = (cv.sel || cv.run ? 330 + 12 : 12) + 24
    const top = 24, bottom = 58
    const w = Math.max(200, r.width - left - right)
    const h = Math.max(160, r.height - top - bottom)
    const z = Math.min(1.2, Math.max(CV_ZOOM_MIN, Math.min(w / (x1 - x0), h / (y1 - y0))))
    cv.zoom = z
    cv.tx = left + (w - (x1 - x0) * z) / 2 - x0 * z
    cv.ty = top + (h - (y1 - y0) * z) / 2 - y0 * z
    cvApplyView()
}

// ── Selection ────────────────────────────────────────────
function cvSelect(id, opts = {}) {
    cv.sel = id
    cv.multi = new Set()
    cv.selEdge = null
    if (cv.run) { cvClearTimers(); cv.run = null }
    if (cv.errNode && cv.errNode !== id) cv.errNode = null
    if (opts.render === false) return
    cvRenderGraph()
    cvRenderSide()
}

function cvSelectEdge(id) {
    cv.selEdge = id
    cv.sel = null
    if (cv.run) { cvClearTimers(); cv.run = null }
    cvRenderGraph()
    cvRenderSide()
}

function cvDeselect() {
    cv.sel = null
    cv.multi = new Set()
    cv.selEdge = null
    cvRenderGraph()
    cvRenderSide()
}

function cvToggleRail() {
    cv.rail = !cv.rail
    cvRenderRail()
    cvRenderHint()
}

// ── Graph rendering ──────────────────────────────────────
// cvRenderGraph redraws every card, port, label and wire from wf. Drags move elements in place
// (cvMoveNodeEls, cvRenderWires) and call this once on release.
function cvRenderGraph() {
    const host = document.getElementById('cv-nodes')
    if (!host) return
    const run = cv.run
    const shown = run ? run.steps.slice(0, run.step) : []
    const onPath = {}
    for (const s of shown) if (!onPath[s.node_id]) onPath[s.node_id] = s.no
    const nowId = shown.length ? shown[shown.length - 1].node_id : null

    let html = ''
    for (const n of wf.nodes) {
        let cls = ''
        if (n.id === cv.sel || cv.multi.has(n.id)) cls += 'sel '
        if (!n.enabled) cls += 'off '
        if (n.id === cv.errNode) cls += 'err '
        if (run) cls += onPath[n.id] ? (n.id === nowId ? 'now ' : 'hit ') : 'dim '
        html += cvNodeHTML(n, cls, onPath[n.id] || 0)
    }
    for (const n of wf.nodes) {
        if (n.kind !== 'trigger') {
            const ip = cvInPt(n)
            html += `<button class="port inp" data-port-of="${n.id}" data-port="in" style="left:${ip.x}px;top:${ip.y}px" tabindex="-1" aria-label="Input of ${esc(n.title)}"></button>`
        }
        for (const p of cvPorts(n)) {
            const pt = cvOutPt(n, p)
            const wired = wf.edges.some(e => e.from === n.id && e.port === p)
            const tgt = cv.drag?.type === 'link' && cv.drag.from === n.id && cv.drag.port === p
            const what = n.kind === 'if' ? (p === 'yes' ? 'match' : 'else') : 'output'
            html += `<button class="port${wired ? ' wired' : ''}${tgt ? ' tgt' : ''}" data-port-of="${n.id}" data-port="${p}" style="left:${pt.x}px;top:${pt.y}px" aria-label="Drag to connect the ${what} port of ${esc(n.title)}"></button>`
            if (n.kind === 'if') html += `<span class="wlabel${p === 'yes' ? ' yes' : ''}" data-label-of="${n.id}" data-port="${p}" style="left:${pt.x}px;top:${pt.y + 24}px">${what}</span>`
        }
    }
    cvCondPopHide()
    cvCondPopBind(host)
    host.innerHTML = html
    cvRenderWires()
    cvRenderGhost()
    cvRenderHint()
}

function cvNodeHTML(n, cls, stepNo) {
    const k = cvKind(n.kind)
    return `<article class="node ${k.tone} ${cls}" data-node="${n.id}" style="transform:translate(${n.x}px, ${n.y}px)">
        ${stepNo ? `<span class="step-no">${stepNo}</span>` : ''}
        <div class="node-top">
            <span class="node-ico">${icon(k.icon)}</span>
            <span class="node-kind grow">${esc(k.label)}</span>
            ${cvCondBadgeHTML(n)}
            ${n.enabled ? '' : '<span class="badge outline">off</span>'}
        </div>
        <div class="node-body">
            <div class="node-title">${esc(n.title || k.label)}</div>
            <div class="node-sub">${esc(cvNodeSub(n))}</div>
        </div>
    </article>`
}

// cvCondInfo describes a trigger or if node's condition for its card: null when there is none,
// { advanced } when it is hand-written text, else the builder's join and rows.
function cvCondInfo(n) {
    if (n.kind !== 'trigger' && n.kind !== 'if') return null
    const text = (n.condition || '').trim()
    if (!text) return null
    const ui = n._ui
    const rows = ui?.mode === 'builder' ? (ui.rows || []).filter(r => !wfCompileRow(r).error) : []
    if (!rows.length) return { advanced: true, text }
    return { join: ui.join, rows }
}

// cvCondBadgeHTML is the condition count on a card; hovering or focusing it opens cvCondPop.
function cvCondBadgeHTML(n) {
    const c = cvCondInfo(n)
    if (!c) return ''
    const what = c.advanced ? 'Advanced condition' : `${c.rows.length} condition${c.rows.length === 1 ? '' : 's'}`
    return `<button class="node-cbadge hit-expand${c.advanced ? ' adv' : ''}" type="button" data-cond-of="${n.id}" aria-label="${what}, show">${icon('filter')}${c.advanced ? 'adv' : c.rows.length}</button>`
}

// cvCondValue names a row's value the way the builder's pickers do.
function cvCondValue(f, v) {
    if (!f?.source) return String(v)
    if (f.source === 'companies' || f.source === 'contacts') return wfNames[f.source][v] || `#${v}`
    return wfSourceList(f).find(o => String(o.value) === String(v))?.label ?? String(v)
}

// cvCondRowText is one builder row in words, for the card's sub line and the popover.
function cvCondRowText(row) {
    const f = wfFieldByPath(row.path)
    const label = f?.label || row.path
    if (row.op === 'true') return label
    if (row.op === 'false') return `not ${label}`
    const op = (f ? wfOpsFor(f) : []).find(([v]) => v === row.op)?.[1] || row.op
    if (WF_NO_VALUE_OPS.has(row.op)) return `${label} ${op}`
    if (WF_LIST_OPS.has(row.op)) return `${label} ${op} ${wfLists.find(l => String(l.id) === String(row.value))?.name || `#${row.value}`}`
    const vals = [].concat(row.value ?? [])
    return `${label} ${op} ${vals.map(v => cvCondValue(f, v)).join(', ')}`
}

const CV_POP_CHIPS = 8

function cvCondRowHTML(row) {
    const f = wfFieldByPath(row.path)
    const label = esc(f?.label || row.path)
    if (row.op === 'true') return `<div class="cond-pop-row flag">${icon('check')}<span class="cond-pop-field">${label}</span></div>`
    if (row.op === 'false') return `<div class="cond-pop-row flag no">${icon('x')}<span><span class="cond-pop-op">not</span> <span class="cond-pop-field">${label}</span></span></div>`
    if (!WF_MULTI_OPS.has(row.op)) {
        const text = cvCondRowText(row).slice((f?.label || row.path).length)
        return `<div class="cond-pop-row"><span class="cond-pop-field">${label}</span><span class="cond-pop-op">${esc(text)}</span></div>`
    }
    const op = wfOpsFor(f).find(([v]) => v === row.op)?.[1] || row.op
    const vals = [].concat(row.value ?? [])
    const shown = vals.length > CV_POP_CHIPS ? vals.slice(0, CV_POP_CHIPS - 1) : vals
    const more = vals.length - shown.length
    return `<div class="cond-pop-row">
        <span><span class="cond-pop-field">${label}</span> <span class="cond-pop-op">${esc(op)}</span></span>
        <span class="chips">${shown.map(v => `<span class="chip">${esc(cvCondValue(f, v))}</span>`).join('')}${more ? `<span class="chip more">+${more} more</span>` : ''}</span>
    </div>`
}

// cvCondPopShow opens the condition popover beside the card that owns badge b. It is fixed to
// the viewport rather than placed in the plane, so it stays readable at any zoom.
function cvCondPopShow(b) {
    const n = wfNode(b.dataset.condOf)
    const c = n && cvCondInfo(n)
    if (!c) return
    let pop = document.getElementById('cv-condpop')
    if (!pop) {
        pop = document.createElement('div')
        pop.id = 'cv-condpop'
        pop.className = 'cond-pop'
        pop.setAttribute('role', 'tooltip')
        document.body.appendChild(pop)
    }
    const head = c.advanced ? 'Advanced condition'
        : c.rows.length === 1 ? '1 condition'
        : c.join === 'or' ? `Match any of ${c.rows.length}` : `Match all ${c.rows.length}`
    pop.innerHTML = `<div class="cond-pop-head">${head}</div>${c.advanced
        ? `<div class="cond-pop-code">${esc(c.text)}</div>`
        : c.rows.map(cvCondRowHTML).join('')}`
    pop.hidden = false
    b.setAttribute('aria-describedby', 'cv-condpop')

    const card = b.closest('.node').getBoundingClientRect()
    const gap = 12
    const w = pop.offsetWidth, h = pop.offsetHeight
    let left = card.right + gap
    if (left + w > innerWidth - gap) left = Math.max(gap, card.left - gap - w)
    pop.style.left = `${left}px`
    pop.style.top = `${Math.max(gap, Math.min(card.top, innerHeight - gap - h))}px`
}

function cvCondPopHide() {
    const pop = document.getElementById('cv-condpop')
    if (pop) pop.hidden = true
    document.querySelectorAll('.node-cbadge[aria-describedby]').forEach(b => b.removeAttribute('aria-describedby'))
}

// cvCondPopBind wires the badges once per canvas host; the handlers are delegated, so
// re-rendered cards need nothing.
function cvCondPopBind(host) {
    if (host.dataset.condPop) return
    host.dataset.condPop = '1'
    const badge = e => e.target.closest?.('.node-cbadge')
    host.addEventListener('mouseover', e => { const b = badge(e); if (b && !cv.drag) cvCondPopShow(b) })
    host.addEventListener('mouseout', e => { const b = badge(e); if (b && !b.contains(e.relatedTarget)) cvCondPopHide() })
    host.addEventListener('focusin', e => { const b = badge(e); if (b) cvCondPopShow(b) })
    host.addEventListener('focusout', e => { if (badge(e)) cvCondPopHide() })
    host.addEventListener('pointerdown', cvCondPopHide)
    host.addEventListener('keydown', e => { if (e.key === 'Escape') cvCondPopHide() })
    document.addEventListener('wheel', cvCondPopHide, { passive: true })
}

// cvNodeSub is the one line a card says about its settings.
function cvNodeSub(n) {
    switch (n.kind) {
    case 'trigger': {
        const ev = n.events || []
        if (!ev.length) return 'no events — nothing enters'
        return `on ticket ${ev.join(' or ')}`
    }
    case 'if': {
        const c = cvCondInfo(n)
        if (!c) return 'always matches'
        if (c.advanced) return 'advanced condition'
        if (c.rows.length === 1) return cvCondRowText(c.rows[0])
        return c.join === 'or' ? 'Match any' : 'Match all'
    }
    case 'notify': {
        const s = n.notify || {}
        if (s.channel === 'resources_owner') return 'ticket resources & owner'
        const kind = wfChannelRecipientType(s.channel)
        const r = wfRecipients.find(r => r.id === s.recipient_id)
        return r ? `${kind} · ${r.name}` : `choose a ${kind}`
    }
    case 'add_note': {
        const s = n.add_note || {}
        const flags = ['discussion', 'internal', 'resolution'].filter(f => s[f]).join(' · ')
        return (s.text || '').trim() ? `${flags || 'note'} · ${s.text.trim()}` : (flags || 'no text yet')
    }
    case 'set_status': {
        const s = n.set_status || {}
        return s.status_id ? (wfStatuses.find(x => x.id === s.status_id)?.name || s.status_name || `status ${s.status_id}`) : 'choose a status'
    }
    case 'set_priority': {
        const s = n.set_priority || {}
        return s.priority_id ? (wfPriorities.find(x => x.id === s.priority_id)?.name || s.priority_name || `priority ${s.priority_id}`) : 'choose a priority'
    }
    case 'set_owner':
    case 'add_resource': {
        const s = n[n.kind] || {}
        const m = wfMembers.find(x => x.id === s.member_id)
        return s.member_id ? (m ? memberLabel(m) : s.identifier || `member ${s.member_id}`) : 'choose a member'
    }
    case 'patch': {
        const ops = n.patch?.ops
        const arr = typeof ops === 'string' ? (() => { try { return JSON.parse(ops) } catch { return null } })() : ops
        return Array.isArray(arr) && arr.length ? `${arr.length} operation${arr.length === 1 ? '' : 's'}` : 'JSON operations'
    }
    case 'skip_notify':
        return 'silences later notifies on this path'
    }
    return ''
}

// cvRefreshNode redraws one card's text after an inspector edit, without touching the rest.
function cvRefreshNode(id) {
    const n = wfNode(id)
    const el = document.querySelector(`#cv-nodes .node[data-node="${id}"]`)
    if (!n || !el) return
    const k = cvKind(n.kind)
    el.classList.toggle('off', !n.enabled)
    el.querySelector('.node-title').textContent = n.title || k.label
    el.querySelector('.node-sub').textContent = cvNodeSub(n)
    el.querySelector('.node-cbadge')?.remove()
    el.querySelector('.node-kind').insertAdjacentHTML('afterend', cvCondBadgeHTML(n))
    const badge = el.querySelector('.node-top .badge')
    if (!n.enabled && !badge) el.querySelector('.node-top').insertAdjacentHTML('beforeend', '<span class="badge outline">off</span>')
    if (n.enabled && badge) badge.remove()
}

// cvMoveNodeEls repositions a card and its ports and labels during a drag.
function cvMoveNodeEls(n) {
    const card = document.querySelector(`#cv-nodes .node[data-node="${n.id}"]`)
    if (card) card.style.transform = `translate(${n.x}px, ${n.y}px)`
    document.querySelectorAll(`#cv-nodes .port[data-port-of="${n.id}"]`).forEach(p => {
        const pt = p.dataset.port === 'in' ? cvInPt(n) : cvOutPt(n, p.dataset.port)
        p.style.left = `${pt.x}px`; p.style.top = `${pt.y}px`
    })
    document.querySelectorAll(`#cv-nodes .wlabel[data-label-of="${n.id}"]`).forEach(l => {
        const pt = cvOutPt(n, l.dataset.port)
        l.style.left = `${pt.x}px`; l.style.top = `${pt.y + 24}px`
    })
}

function cvRenderWires() {
    const svg = document.getElementById('cv-wires')
    if (!svg) return
    const run = cv.run
    const shown = run ? run.steps.slice(0, run.step) : []
    const ran = new Set(shown.map(s => s.via).filter(Boolean))

    let html = ''
    let cut = null
    for (const e of wf.edges) {
        const a = wfNode(e.from), b = wfNode(e.to)
        if (!a || !b) continue
        const p = cvOutPt(a, e.port), q = cvInPt(b)
        let cls = ''
        if (run) cls = ran.has(e.id) ? 'ran' : 'dim'
        else if (e.id === cv.selEdge) cls = 'pick'
        else if (e.from === cv.sel || e.to === cv.sel) cls = 'hot'
        const d = cvCurve(p, q)
        html += `<g><path class="wire ${cls}" d="${d}"></path><path class="wirehit" data-edge="${e.id}" d="${d}"></path></g>`
        if (e.id === cv.selEdge && !run) cut = { x: (p.x + q.x) / 2, y: (p.y + q.y) / 2 }
    }
    if (cv.drag?.type === 'link' && cv.link) {
        const a = wfNode(cv.drag.from)
        if (a) html += `<path class="wire temp" d="${cvCurve(cvOutPt(a, cv.drag.port), cv.link)}"></path>`
    }
    svg.innerHTML = html

    // the disconnect button lives among the nodes so it takes clicks; one per selected wire
    document.querySelector('#cv-nodes .wirex')?.remove()
    if (cut && canEdit()) {
        document.getElementById('cv-nodes').insertAdjacentHTML('beforeend',
            `<button class="wirex" style="left:${cut.x}px;top:${cut.y}px" aria-label="Disconnect this wire">${icon('x')}</button>`)
    }
}

function cvRenderGhost() {
    const host = document.getElementById('cv-nodes')
    if (!host) return
    host.querySelector('.ghost')?.remove()
    if (cv.ghost) {
        host.insertAdjacentHTML('beforeend', `<div class="ghost" style="transform:translate(${cv.ghost.x}px, ${cv.ghost.y}px)">drop ${esc(cvKind(cv.ghost.kind).label)}</div>`)
    }
}

function cvRenderHint() {
    const el = document.getElementById('cv-hint')
    if (!el) return
    let hint = 'drag a step in from the list · drag to pan · shift-drag to select several · right-click for actions'
    if (cv.drag?.type === 'link') hint = 'drop on a step to connect, or on empty canvas to disconnect'
    else if (cv.drag?.type === 'pal') hint = 'release over the canvas to place it'
    else if (cv.run?.record) hint = 'recorded path highlighted · drag to pan · pinch or ⌘-scroll to zoom'
    else if (cv.run) hint = 'simulated path highlighted — nothing was sent'
    else if (cv.drag?.type === 'box') hint = 'release to select the steps inside the box'
    else if (cv.multi.size > 1) hint = `${cv.multi.size} steps selected · drag to move them · ⌘C copies · ⌘D duplicates · Delete removes them`
    else if (cv.selEdge) hint = 'wire selected · × or Delete disconnects it'
    else if (cv.sel) hint = 'drag a port to wire it · Delete removes the step · ⌘C copies it'
    el.textContent = hint
}

// ── Step list (left dock) ────────────────────────────────
function cvRenderRail() {
    const host = document.getElementById('cv-rail')
    if (!host) return
    if (!cvEditable()) { host.innerHTML = ''; return }
    if (!cv.rail) {
        host.innerHTML = `<button class="dock-bar tl" style="cursor:pointer" onclick="cvToggleRail()">${icon('plus')}<span>Add a step</span></button>`
        return
    }
    const groups = CV_GROUPS.map(g => {
        const kinds = Object.entries(CV_KINDS).filter(([, k]) => k.group === g)
        return `<div class="eyebrow" style="margin:var(--s3) 0 var(--s2)">${esc(g)}</div>
        <div class="pal-list">${kinds.map(([kind, k]) =>
            `<button class="pal ${k.tone}" data-kind="${kind}" onpointerdown="cvGrabPal('${kind}', event)" title="Drag onto the canvas, or double-click to add it below the selected step">
                <span class="swat"></span><span class="grow">${esc(k.label)}</span>
            </button>`).join('')}</div>`
    }).join('')
    host.innerHTML = `<aside class="dock dock-l">
        <article class="card">
            <div class="card-head">
                <div><h3 style="font-size:var(--text-sm)">Steps</h3></div>
                <button class="icon-btn" aria-label="Hide the step list" onclick="cvToggleRail()">${icon('panelL')}</button>
            </div>
            <div class="card-body">${groups}</div>
        </article>
    </aside>`
}

// ── Inspector and simulation trace (right dock) ──────────
function cvRenderSide() {
    const host = document.getElementById('cv-side')
    if (!host) return
    if (cv.run) { host.innerHTML = cvRunHTML(); return }
    if (cv.multi.size > 1) {
        const n = cv.multi.size
        const anyOn = [...cv.multi].some(id => wfNode(id)?.enabled)
        host.innerHTML = `<aside class="dock dock-r"><article class="card">
            <div class="card-head">
                <div><h3>${n} steps selected</h3><p>Drag any of them to move the group. Right-click for the same actions.</p></div>
                <button class="icon-btn" aria-label="Clear the selection" onclick="cvDeselect()">${icon('x')}</button>
            </div>
            <div class="card-body row gap2 wrap">
                <button class="btn btn-default btn-sm" onclick="cvDuplicateSelection()">${icon('copy')}Duplicate</button>
                <button class="btn btn-default btn-sm" onclick="cvCopySelection()">${icon('copy')}Copy</button>
                <button class="btn btn-default btn-sm" onclick="cvSetSelectionEnabled(${anyOn ? 'false' : 'true'})">${icon(anyOn ? 'ban' : 'check')}${anyOn ? 'Disable' : 'Enable'}</button>
                <button class="btn btn-ghost btn-sm" onclick="cvDeleteSelection()">${icon('trash')}Delete</button>
            </div>
        </article></aside>`
        return
    }
    const n = cv.sel ? wfNode(cv.sel) : null
    if (!n) { host.innerHTML = ''; return }
    host.innerHTML = `<aside class="dock dock-r"><article class="card">${cvInspectorHTML(n)}</article></aside>`
}

// cvRerenderSide redraws the inspector for the selected node; used after a structural change to
// a node's settings (a kind-specific block appearing or disappearing).
function cvRerenderSide() {
    cvRenderSide()
    if (cv.sel) cvRefreshNode(cv.sel)
}

function cvInspectorHTML(n) {
    const k = cvKind(n.kind)
    const ro = !canEdit()
    const dis = ro ? ' disabled' : ''
    const id = n.id
    let body = ''

    switch (n.kind) {
    case 'trigger': {
        const ev = n.events || []
        const has = e => ev.includes(e)
        body = `<div class="field">
            <label>Enters this flow when a ticket is</label>
            <div class="stack gap3" style="padding-top:var(--s1)">
                ${checkbox('created', `onchange="cvToggleEvent('${id}', 'created', this.checked)"${dis}`, has('created'))}
                ${checkbox('updated', `onchange="cvToggleEvent('${id}', 'updated', this.checked)"${dis}`, has('updated'))}
            </div>
            <span class="hint">A flow can have more than one trigger. Every trigger that accepts an event runs; a step two paths both reach runs once.</span>
        </div>
        <div id="trigger-warn-${id}">${ev.length ? '' : `<div class="callout warn">${icon('alert')}<div>No events selected, so nothing ever enters here.</div></div>`}</div>
        ${wfConditionHTML(n)}`
        break
    }
    case 'if':
        body = `${wfConditionHTML(n)}
        <div class="field">
            <label>Branches</label>
            <span class="hint">The <b>match</b> port leaves when the condition holds, <b>else</b> when it does not. An unwired port simply ends that path. A disabled If always takes else.</span>
        </div>`
        break
    case 'notify': {
        const s = n.notify || (n.notify = { channel: 'webex_room', recipient_id: null })
        const hasMsg = !!s.message
        const rtype = wfChannelRecipientType(s.channel)
        body = `<div class="field">
            <label for="insp-channel">Send to</label>
            <select id="insp-channel" class="select" onchange="cvSetNotifyChannel('${id}', this.value)"${dis}>${WF_CHANNELS.map(([v, l]) => `<option value="${v}"${v === s.channel ? ' selected' : ''}>${l}</option>`).join('')}</select>
        </div>
        ${s.channel === 'resources_owner'
            ? `<span class="hint">Everyone assigned to the ticket, plus the owner. Skips whoever wrote the triggering note; forwards apply.</span>`
            : `<div class="field">
                <label for="insp-recipient">${rtype === 'person' ? 'Person' : 'Room'}</label>
                <select id="insp-recipient" class="select" onchange="cvSetSetting('${id}', 'notify.recipient_id', this.value ? parseInt(this.value) : null)"${dis}>
                    <option value="">— choose a ${rtype} —</option>${wfRecipientOptions(rtype, s.recipient_id)}</select>
            </div>`}
        <div class="field">
            <label for="insp-msg">Custom message${hasMsg ? '' : ' <span class="muted">(optional)</span>'}</label>
            <textarea id="insp-msg" class="textarea mono" rows="4" aria-label="Custom message" placeholder="{{event}}: {{ticket.link}} {{ticket.summary}}&#10;**Company:** {{company}}&#10;{{note.quote}}" oninput="cvSetSetting('${id}', 'notify.message', this.value)"${dis}>${esc(s.message || '')}</textarea>
            <span class="hint">Empty uses the default layout. Click a token to insert it.</span>
            <div class="placeholder-list">${wfPlaceholders.map(p => `<code class="code inline" data-tip="${esc(p.description)}" onclick="wfInsertPlaceholder(this, '${p.name}')">{{${p.name}}}</code>`).join('')}</div>
        </div>
        <div class="field">
            <div class="row spread gap2">
                <label for="insp-preview-ticket">Preview with a ticket</label>
                <button class="icon-btn hit-expand" style="width:22px;height:22px" onclick="wfShowHelp('messages')" aria-label="How messages work">${icon('info')}</button>
            </div>
            <div class="row gap2 wrap">
                <input type="number" id="insp-preview-ticket" class="input" style="width:120px" min="1" placeholder="Ticket #" value="${esc(wfSimTicket)}" aria-label="Ticket number to render the message with">
                <button class="btn btn-default btn-sm" onclick="wfPreviewMessage('${id}')">${icon('play')}Preview</button>
            </div>
            <div id="insp-preview-out" class="stack gap2"></div>
        </div>`
        break
    }
    case 'add_note': {
        const s = n.add_note || (n.add_note = { text: '', internal: true, discussion: false, resolution: false })
        const flag = (key, label) => checkbox(label, `onchange="cvSetSetting('${id}', 'add_note.${key}', this.checked)"${dis}`, !!s[key])
        body = `<div class="field">
            <label for="insp-note">Note text</label>
            <textarea id="insp-note" class="textarea" rows="4" placeholder="Note text" oninput="cvSetSetting('${id}', 'add_note.text', this.value)"${dis}>${esc(s.text || '')}</textarea>
        </div>
        <div class="field">
            <label>Post as</label>
            <div class="stack gap3" style="padding-top:var(--s1)">${flag('discussion', 'Discussion')}${flag('internal', 'Internal')}${flag('resolution', 'Resolution')}</div>
        </div>`
        break
    }
    case 'set_status': {
        const s = n.set_status || (n.set_status = { status_id: 0 })
        const known = wfStatuses.some(x => x.id === s.status_id)
        body = `<div class="field">
            <label for="insp-status">Status</label>
            <select id="insp-status" class="select" onchange="cvSetStatus('${id}', this.value)"${dis}>
                <option value="">— choose a status —</option>
                ${wfStatuses.map(x => `<option value="${x.id}"${x.id === s.status_id ? ' selected' : ''}>${esc(x.name)}${x.closed ? ' (closed)' : ''}</option>`).join('')}
                ${s.status_id && !known ? `<option value="${s.status_id}" selected>${esc(s.status_name || `Status ${s.status_id}`)} (not on this board)</option>` : ''}
            </select>
            <span class="hint">Skipped when the ticket is already in this status.</span>
        </div>`
        break
    }
    case 'set_priority': {
        const s = n.set_priority || (n.set_priority = { priority_id: 0 })
        const known = wfPriorities.some(x => x.id === s.priority_id)
        body = `<div class="field">
            <label for="insp-priority">Priority</label>
            <select id="insp-priority" class="select" onchange="cvSetPriority('${id}', this.value)"${dis}>
                <option value="">— choose a priority —</option>
                ${wfPriorities.map(x => `<option value="${x.id}"${x.id === s.priority_id ? ' selected' : ''}>${esc(x.name)}</option>`).join('')}
                ${s.priority_id && !known ? `<option value="${s.priority_id}" selected>${esc(s.priority_name || `Priority ${s.priority_id}`)}</option>` : ''}
            </select>
            <span class="hint">Skipped when the ticket already has this priority.</span>
        </div>`
        break
    }
    case 'set_owner':
    case 'add_resource': {
        const s = n[n.kind] || (n[n.kind] = { member_id: 0 })
        body = `<div class="field">
            <label for="insp-member">Member</label>
            <select id="insp-member" class="select" onchange="cvSetMember('${id}', this.value)"${dis}>
                <option value="">— choose a member —</option>${wfMemberOptions(s.member_id)}</select>
            <span class="hint">${n.kind === 'set_owner' ? 'Skipped if already the owner.' : 'Skipped if already assigned.'}</span>
        </div>`
        break
    }
    case 'patch': {
        const s = n.patch || (n.patch = { ops: '' })
        const ops = typeof s.ops === 'string' ? s.ops : (s.ops ? JSON.stringify(s.ops, null, 2) : '')
        body = `<div class="field">
            <label for="insp-ops">Operations</label>
            <textarea id="insp-ops" rows="6" class="textarea mono" spellcheck="false" placeholder='${WF_PATCH_EXAMPLE}' oninput="cvSetPatchOps('${id}', this.value)"${dis}>${esc(ops)}</textarea>
            <span class="hint">JSON array of ConnectWise patch operations: <code class="code inline">op</code> (add / replace / remove), <code class="code inline">path</code>, <code class="code inline">value</code>. Sent as-is to PATCH /service/tickets/{id}.</span>
        </div>`
        break
    }
    case 'skip_notify':
        body = `<span class="hint">Every Notify step after this one on the same path is skipped. Other paths are not affected.</span>`
        break
    }

    const canDelete = canEdit() && !(n.kind === 'trigger' && wf.nodes.filter(x => x.kind === 'trigger').length < 2)
    return `<div class="card-head">
        <div><h3>${esc(k.label)}</h3><p>${esc(k.hint)}</p></div>
        <button class="icon-btn" aria-label="Close the inspector" onclick="cvDeselect()">${icon('x')}</button>
    </div>
    <div class="card-body stack gap4">
        <div class="field">
            <label for="insp-title">Step name</label>
            <input id="insp-title" class="input" type="text" value="${esc(n.title)}" placeholder="${esc(k.label)}" oninput="wfSetNode('${id}', 'title', this.value)"${dis}>
        </div>
        <div class="row spread gap3">
            ${toggle(`onchange="wfSetNode('${id}', 'enabled', this.checked)"${dis}`, n.enabled, { label: 'Enabled' })}
            ${canEdit() ? `<button class="btn btn-ghost btn-sm" onclick="cvDeleteNode('${id}')"${canDelete ? '' : ' disabled data-tip="A workflow needs at least one trigger" data-tip-align="right"'}>${icon('trash')}Delete step</button>` : ''}
        </div>
        ${body}
    </div>`
}

function wfMemberOptions(selected) {
    return wfMembers
        .slice().sort((a, b) => memberLabel(a).localeCompare(memberLabel(b)))
        .map(m => `<option value="${m.id}"${m.id === selected ? ' selected' : ''}>${esc(memberLabel(m))}${m.identifier ? ` (${esc(m.identifier)})` : ''}</option>`)
        .join('')
}

function wfRecipientOptions(type, selected) {
    return wfRecipients
        .filter(r => r.type === type)
        .sort((a, b) => a.name.localeCompare(b.name))
        .map(r => `<option value="${r.id}"${r.id === selected ? ' selected' : ''}>${esc(r.name)}${r.email ? ` (${esc(r.email)})` : ''}</option>`)
        .join('')
}

// ── State mutations (no re-render, keeps focus) ──────────
function wfSet(field, value) {
    wf[field] = value
    wfMarkDirty()
}

function wfSetNode(id, field, value) {
    const n = wfNode(id)
    if (!n) return
    n[field] = value
    cvRefreshNode(id)
    wfMarkDirty()
}

// path is "notify.recipient_id", "add_note.text", ...
function cvSetSetting(id, path, value) {
    const n = wfNode(id)
    if (!n) return
    const [group, key] = path.split('.')
    n[group] = n[group] || {}
    n[group][key] = value
    cvRefreshNode(id)
    wfMarkDirty()
}

function cvToggleEvent(id, ev, on) {
    const n = wfNode(id)
    if (!n) return
    const set = new Set(n.events || [])
    if (on) set.add(ev); else set.delete(ev)
    n.events = ['created', 'updated'].filter(e => set.has(e))
    const warn = document.getElementById(`trigger-warn-${id}`)
    if (warn) warn.innerHTML = n.events.length ? '' : `<div class="callout warn">${icon('alert')}<div>No events selected, so nothing ever enters here.</div></div>`
    cvRefreshNode(id)
    wfMarkDirty()
}

function cvSetNotifyChannel(id, channel) {
    const n = wfNode(id)
    if (!n) return
    n.notify = { channel, message: n.notify?.message || '' }
    if (channel !== 'resources_owner') n.notify.recipient_id = null
    cvRerenderSide()
    wfMarkDirty()
}

function cvSetStatus(id, value) {
    const sid = value ? parseInt(value) : 0
    const st = wfStatuses.find(s => s.id === sid)
    wfNode(id).set_status = { status_id: sid, status_name: st ? st.name : '' }
    cvRefreshNode(id)
    wfMarkDirty()
}

function cvSetPriority(id, value) {
    const pid = value ? parseInt(value) : 0
    const p = wfPriorities.find(p => p.id === pid)
    wfNode(id).set_priority = { priority_id: pid, priority_name: p ? p.name : '' }
    cvRefreshNode(id)
    wfMarkDirty()
}

function cvSetMember(id, value) {
    const n = wfNode(id)
    const mid = value ? parseInt(value) : 0
    const m = wfMembers.find(m => m.id === mid)
    n[n.kind] = { member_id: mid, identifier: m ? m.identifier : '' }
    cvRefreshNode(id)
    wfMarkDirty()
}

// patch ops are kept as the raw string while editing; wfPrepareForServer parses on save/simulate
function cvSetPatchOps(id, value) {
    wfNode(id).patch = { ops: value }
    cvRefreshNode(id)
    wfMarkDirty()
}

// wfPreviewMessage renders the notify step's message (or the default layout when it is empty)
// against a stored ticket, so the editor sees what the placeholders produce.
async function wfPreviewMessage(id) {
    const n = wfNode(id)
    const input = document.getElementById('insp-preview-ticket')
    const out = document.getElementById('insp-preview-out')
    const tid = parseInt(input?.value || wfSimTicket)
    if (!tid) { toast('Enter a ticket number to preview with', 'error'); return }
    wfSimTicket = String(tid)
    const message = (n?.notify?.message || '').trim() || '{{event}}: {{ticket.link}} {{ticket.summary}}\n**Company:** {{company}}\n{{note.quote}}'
    out.innerHTML = '<span class="cell-sub">Rendering…</span>'
    try {
        const res = await api('POST', '/workflows/preview-message', { message, ticket_id: tid, as_new: wfSimAsNew })
        out.innerHTML = `<pre class="code prewrap">${esc(res.rendered)}</pre>${(n?.notify?.message || '').trim() ? '' : '<span class="cell-sub">No custom message set: this is the default layout.</span>'}`
    } catch (e) {
        out.innerHTML = `<span class="cond-result err">${esc(e.message)}</span>`
    }
}

// wfShowHelp opens the concepts guide. section scrolls to one of: flow, conditions, messages, testing.
function wfShowHelp(section = '') {
    const h = (id, title) => `<h4 id="wf-help-${id}" style="margin-top:var(--s4)">${title}</h4>`
    openModal('How workflows run', `<div class="stack gap3" style="max-height:60vh;overflow-y:auto">
        ${h('flow', 'The flow')}
        <p>A workflow belongs to one board. Every ticket event on that board (created, or updated) enters at each <b>Trigger</b> that listens for it, and walks the wires from there. A trigger can carry an <b>Only when</b> condition; when it does not hold, that lane simply does not run.</p>
        <p>An <b>If</b> step sends the walk out of its <b>match</b> port when its condition holds and <b>else</b> when it does not. A port with nothing wired to it simply ends that path. A step that two paths both reach runs once.</p>
        <p><b>Skip notify</b> silences the Notify steps after it on its own path only. Other paths still notify.</p>
        <p>Right-click a step to duplicate, copy, disable or delete it. Shift-drag on empty canvas selects several steps; dragging any of them moves the group, and a copied group pastes into another workflow with its wires.</p>
        <p>Ticket writes (status, priority, owner, resources, patch) are collected and sent to ConnectWise as one change after the whole flow has run. If two steps set the same field, the later one wins and the ticket history says so. Notes are added after that change, then notifications go out.</p>
        ${h('conditions', 'Conditions')}
        <p>Each row is a field, a comparison and a value. <b>Match all</b> means every row must hold, <b>match any</b> means one is enough.</p>
        <p><b>changed to</b> holds only on the update that moved the field to that value, so "Status changed to Waiting" fires once, when it happens. <b>changed from</b> matches the value it left. "is" holds on every update while the field has that value.</p>
        <p>The builder covers everyday conditions. <b>Advanced</b> shows the same condition as text and lets you write anything the engine understands, including parentheses and "not". A condition the builder cannot show opens in Advanced with a note saying why.</p>
        ${h('messages', 'Messages')}
        <p>A Notify step sends the default layout unless you give it a custom message. Click a <code class="code inline">{{token}}</code> to insert it at the cursor; <b>Preview</b> renders the message with a real ticket so you can see what each token produces.</p>
        ${h('testing', 'Trying it safely')}
        <p>Turn on the workflow's <b>Dry run</b> first: the flow runs and records what it would have done in the ticket history, but writes nothing to ConnectWise. <b>Simulate</b> in the toolbar runs the flow against a ticket right now and highlights the path it took. Turn dry run off once the history looks right.</p>
    </div>`, async () => closeModal(), 'Got it')
    if (section) setTimeout(() => document.getElementById(`wf-help-${section}`)?.scrollIntoView({ block: 'start' }), 50)
}

function wfInsertPlaceholder(el, name) {
    const ta = el.closest('.field')?.querySelector('textarea')
    if (!ta || ta.disabled) return
    const start = ta.selectionStart ?? ta.value.length
    const tok = `{{${name}}}`
    ta.value = ta.value.slice(0, start) + tok + ta.value.slice(ta.selectionEnd ?? start)
    ta.focus()
    ta.setSelectionRange(start + tok.length, start + tok.length)
    ta.dispatchEvent(new Event('input'))
}

function wfMarkDirty() {
    const dirty = wfIsDirty()
    document.getElementById('wf-dirty')?.classList.toggle('hidden', !dirty)
    const save = document.getElementById('wf-save')
    if (!save) return
    // the accent is the invitation to save: it appears only when there is something to save
    save.disabled = !dirty
    save.classList.toggle('btn-primary', dirty)
    save.classList.toggle('btn-default', !dirty)
}

// wfRerenderCondBlock replaces the inspector's condition block after the builder changes shape.
function wfRerenderCondBlock(id) {
    const n = wfNode(id)
    const el = document.getElementById(`cond-wrap-${id}`)
    if (n && el) el.outerHTML = wfConditionHTML(n)
    cvRefreshNode(id)
}

// ── Structural changes ───────────────────────────────────
function cvNewID() {
    if (crypto.randomUUID) return crypto.randomUUID()
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = Math.random() * 16 | 0
        return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16)
    })
}

function cvNewNode(kind, x, y) {
    const n = { id: cvNewID(), kind, title: cvKind(kind).label, enabled: true, x: Math.round(x), y: Math.round(y) }
    switch (kind) {
    case 'trigger':      n.events = ['created', 'updated']; n.condition = ''; n._ui = wfDefaultUI(); break
    case 'if':           n.condition = ''; n._ui = wfDefaultUI(); break
    case 'notify':       n.notify       = { channel: 'webex_room', recipient_id: null }; break
    case 'add_note':     n.add_note     = { text: '', internal: true, discussion: false, resolution: false }; break
    case 'set_status':   n.set_status   = { status_id: 0 }; break
    case 'set_priority': n.set_priority = { priority_id: 0 }; break
    case 'set_owner':    n.set_owner    = { member_id: 0 }; break
    case 'add_resource': n.add_resource = { member_id: 0 }; break
    case 'patch':        n.patch        = { ops: '' }; break
    }
    return n
}

// cvDrop places a new step and wires it from the nearest free port above it, so a step dropped
// under a card joins the flow without a second gesture.
function cvDrop(kind, x, y) {
    const n = cvNewNode(kind, x, y)
    wf.nodes.push(n)
    if (kind !== 'trigger') {
        const parent = cvNearestFree(y, x)
        if (parent) wf.edges.push({ id: cvNewID(), from: parent.id, to: n.id, port: parent.port })
    }
    cvSelect(n.id)
    wfMarkDirty()
    document.getElementById('insp-title')?.select()
}

// cvAppend adds a step below the selected node (or the lowest node) and wires it there.
function cvAppend(kind) {
    const anchor = wfNode(cv.sel) || wf.nodes.reduce((a, n) => (!a || n.y > a.y ? n : a), null)
    if (!anchor) { cvDrop(kind, 0, 0); return }
    cvDrop(kind, anchor.x, anchor.y + CV_H + 56)
}

function cvNearestFree(y, x) {
    let best = null
    for (const n of wf.nodes) {
        const bottom = n.y + CV_H
        if (bottom > y) continue
        for (const p of cvPorts(n)) {
            if (wf.edges.some(e => e.from === n.id && e.port === p)) continue
            const dist = (y - bottom) + Math.abs(n.x - x) * 0.5
            if (!best || dist < best.dist) best = { id: n.id, port: p, dist }
        }
    }
    return best && best.dist < 460 ? best : null
}

function cvDeleteNode(id) {
    const n = wfNode(id)
    if (!n) return
    if (n.kind === 'trigger' && wf.nodes.filter(x => x.kind === 'trigger').length < 2) {
        toast('A workflow needs at least one trigger', 'error')
        return
    }
    wf.nodes = wf.nodes.filter(x => x.id !== id)
    wf.edges = wf.edges.filter(e => e.from !== id && e.to !== id)
    if (cv.sel === id) cv.sel = null
    cvRenderGraph()
    cvRenderSide()
    wfMarkDirty()
}

function cvCutEdge() {
    if (!cv.selEdge) return
    wf.edges = wf.edges.filter(e => e.id !== cv.selEdge)
    cv.selEdge = null
    cvRenderGraph()
    wfMarkDirty()
}

// ── Auto-arrange: a layered forest, top to bottom ────────
// Depth is the LONGEST path from any trigger, so a node several paths reach lands below all of
// them and the merge reads as a merge. Positions persist, so the previous layout is kept for undo.
function cvArrange() {
    const GAP = 48, VGAP = 66
    const nodes = wf.nodes, edges = wf.edges
    if (!nodes.length) return
    cv.undo = Object.fromEntries(nodes.map(n => [n.id, { x: n.x, y: n.y }]))

    // a child remembers the port it hangs off: match branches lean left, else branches right,
    // so an if node's two paths fan out instead of stacking in one column
    const parents = {}, depth = {}
    const lean = { yes: -(CV_W + GAP) / 2, no: (CV_W + GAP) / 2, out: 0 }
    for (const n of nodes) { parents[n.id] = []; depth[n.id] = 0 }
    for (const e of edges) if (parents[e.to]) parents[e.to].push({ id: e.from, lean: lean[e.port] || 0 })
    for (let i = 0; i < nodes.length; i++) {
        let moved = false
        for (const e of edges) {
            if (depth[e.to] === undefined || depth[e.from] === undefined) continue
            if (depth[e.from] + 1 > depth[e.to]) { depth[e.to] = depth[e.from] + 1; moved = true }
        }
        if (!moved) break
    }

    const levels = {}
    for (const n of nodes) (levels[depth[n.id]] = levels[depth[n.id]] || []).push(n.id)
    const maxD = Math.max(0, ...Object.keys(levels).map(Number))

    const y = {}
    let cur = 24
    for (let d = 0; d <= maxD; d++) {
        y[d] = cur
        cur += CV_H + VGAP
    }

    // x: each node starts under the average of its parents, then siblings on the level are
    // pushed apart until nothing overlaps
    const x = {}
    let slot = 0
    for (const id of (levels[0] || []).sort((a, b) => wfNode(a).x - wfNode(b).x)) { x[id] = slot; slot += CV_W + GAP }
    for (let d = 1; d <= maxD; d++) {
        const ids = (levels[d] || []).slice()
        for (const id of ids) {
            const ps = parents[id].filter(p => x[p.id] !== undefined)
            x[id] = ps.length ? ps.reduce((a, p) => a + x[p.id] + p.lean, 0) / ps.length : 0
        }
        ids.sort((a, b) => x[a] - x[b])
        for (let i = 1; i < ids.length; i++) {
            const min = x[ids[i - 1]] + CV_W + GAP
            if (x[ids[i]] < min) x[ids[i]] = min
        }
    }

    for (const n of nodes) { n.x = Math.round(x[n.id] || 0); n.y = y[depth[n.id]] }
    cvRenderGraph()
    cvFit()
    wfMarkDirty()
    cvRenderUndo()
}

function cvUndoArrange() {
    if (!cv.undo) return
    for (const n of wf.nodes) {
        const p = cv.undo[n.id]
        if (p) { n.x = p.x; n.y = p.y }
    }
    cv.undo = null
    cvRenderGraph()
    cvFit()
    wfMarkDirty()
    cvRenderUndo()
}

function cvRenderUndo() {
    const el = document.getElementById('wf-arrange-undo')
    if (el) el.innerHTML = cv.undo ? `<button class="btn btn-ghost btn-sm" onclick="cvUndoArrange()">${icon('undo')}Undo arrange</button>` : ''
}

// ── Condition tools (inspector footer) ───────────────────
async function wfValidate(id) {
    const out = document.getElementById(`cond-result-${id}`)
    const ta  = document.getElementById(`cond-${id}`)
    const n   = wfNode(id)
    if (!out || !n) return
    out.className = 'cond-result'
    out.textContent = 'Checking…'
    try {
        const res = await api('POST', '/workflows/validate-condition', { condition: n.condition })
        if (res.valid) {
            out.className = 'cond-result ok'
            out.textContent = n.condition.trim() ? 'Valid' : 'Valid (matches everything)'
        } else {
            out.className = 'cond-result err'
            out.textContent = `${res.error} (position ${res.pos})`
            if (ta && typeof res.pos === 'number') { ta.focus(); ta.setSelectionRange(res.pos, res.pos) }
        }
    } catch (e) {
        out.className = 'cond-result err'
        out.textContent = e.message
    }
}

async function wfTest(id) {
    const out = document.getElementById(`test-result-${id}`)
    const tid = parseInt(document.getElementById(`test-ticket-${id}`)?.value)
    const n   = wfNode(id)
    if (!out || !n) return
    out.className = 'cond-result'
    if (!tid) { out.className = 'cond-result err'; out.textContent = 'Enter a ticket number'; return }
    out.textContent = 'Testing…'
    try {
        const res = await api('POST', '/workflows/evaluate-condition', { condition: n.condition, ticket_id: tid })
        out.className = `cond-result ${res.matches ? 'ok' : 'err'}`
        out.textContent = `${res.matches ? 'Matches' : 'No match'} against #${tid} (${res.source})`
    } catch (e) {
        out.className = 'cond-result err'
        out.textContent = e.message
    }
}

// ── Simulate ─────────────────────────────────────────────
// The server walks the draft against a stored ticket and returns the path; the canvas replays it
// one step at a time and the right dock becomes the trace.
async function wfSimulate() {
    const id = parseInt(document.getElementById('sim-ticket')?.value)
    wfSimTicket = id ? String(id) : ''
    wfSimAsNew  = document.getElementById('sim-mode')?.value === 'create'
    if (!id) { toast('Enter a ticket number to simulate against', 'error'); return }

    const bad = wfClientValidate()
    if (bad) { cvShowProblem(bad); return }

    try {
        const draft = wfPrepareForServer(JSON.parse(JSON.stringify(wf)))
        const res = await api('POST', `/workflows/${wf.id}/simulate`, { ticket_id: id, as_new: wfSimAsNew, workflow: draft })
        cvClearTimers()
        cv.sel = null
        cv.selEdge = null
        cv.run = { ticket: id, asNew: wfSimAsNew, res, steps: res.workflow?.steps || [], step: 0 }
        cvRenderGraph()
        cvRenderSide()
        cvRenderHint()
        const total = cv.run.steps.length
        for (let i = 1; i <= total; i++) {
            cv.timers.push(setTimeout(() => {
                if (!cv.run) return
                cv.run.step = i
                cvRenderGraph()
                cvRenderSide()
            }, i * 420))
        }
    } catch (e) {
        const d = e.data?.details?.[0]
        if (d) cvShowProblem(wfProblemFromDetail(d))
        else toast(e.message, 'error')
    }
}

function cvClearRun() {
    cvClearTimers()
    cv.run = null
    cvRenderGraph()
    cvRenderSide()
    cvRenderHint()
}

function cvRunHTML() {
    const run = cv.run
    const res = run.res
    const shown = run.steps.slice(0, run.step)
    const done = run.step >= run.steps.length
    const steps = shown.map(s => {
        const n = wfNode(s.node_id)
        const [tone, body] = cvStepText(s, n)
        return `<div class="event ${tone}">
            <div class="event-card">
                <div class="event-head">
                    <span class="title">${esc(s.title || n?.title || '?')}</span>
                    <span class="src">${esc(cvKind(s.kind).label.toLowerCase())}</span>
                    <span class="time num">step ${s.no}</span>
                </div>
                <div class="event-body">${body}</div>
            </div>
        </div>`
    }).join('')

    const none = !run.steps.length
    let after = ''
    if (done && !none) {
        const recips = (res.recipients || []).map(r => `<div class="stack gap1">
            <div class="row gap2 wrap">
                ${r.error ? badgeTag('Error', 'bad') : badgeTag(r.recipient_type, '')}
                <span>${r.error
                    ? `<b>${esc(r.title)}</b>: ${esc(r.error)}`
                    : `<b>${esc(r.recipient_name)}</b> <span class="muted">via ${esc(r.title)}${r.forwarded_from?.length ? `, forwarded from ${esc(r.forwarded_from.join(' → '))}` : ''}</span>`}</span>
            </div>
            ${r.message ? `<pre class="code prewrap">${esc(r.message)}</pre>` : ''}
        </div>`).join('') || '<span class="cell-sub">Nobody would be notified</span>'
        after = `<div class="stack gap2" style="margin-top:var(--s4)"><div class="eyebrow">Would notify</div><div class="stack gap3">${recips}</div></div>`
    }

    const summary = none
        ? 'no trigger accepts this event'
        : (done ? `${run.steps.length} step${run.steps.length === 1 ? '' : 's'} · ${esc(res.source)} snapshot` : 'running…')

    if (run.record) {
        return `<aside class="dock dock-r"><article class="card">
            <div class="card-head">
                <div>
                    <h3>Recorded run</h3>
                    <p>Ticket <a class="link num" href="#tickets/${run.ticket}">#${run.ticket}</a> · ${run.asNew ? 'created' : 'updated'}</p>
                </div>
            </div>
            <div class="card-body">
                ${none ? `<div class="callout warn">${icon('alert')}<div>No trigger listened for a ${run.asNew ? 'created' : 'updated'} ticket, so nothing ran.</div></div>` : `<div class="events">${steps}</div>`}
            </div>
            <div class="card-foot">
                <span class="cell-sub">${none ? 'no trigger fired' : `${run.steps.length} step${run.steps.length === 1 ? '' : 's'}`}</span>
                ${run.dryRun ? '<span class="badge warn">dry run</span>' : ''}
            </div>
        </article></aside>`
    }

    return `<aside class="dock dock-r"><article class="card">
        <div class="card-head">
            <div>
                <h3>Simulation</h3>
                <p>Ticket <a class="link num" href="#tickets/${run.ticket}">#${run.ticket}</a> · ${run.asNew ? 'created' : 'updated'}</p>
            </div>
            <button class="icon-btn" aria-label="Close the simulation trace" onclick="cvClearRun()">${icon('x')}</button>
        </div>
        <div class="card-body">
            ${none ? `<div class="callout warn">${icon('alert')}<div>No trigger listens for a ${run.asNew ? 'created' : 'updated'} ticket, so nothing runs.</div></div>` : `<div class="events">${steps}</div>`}
            ${after}
        </div>
        <div class="card-foot">
            <span class="cell-sub">${summary}</span>
            <span class="badge warn">nothing sent</span>
        </div>
    </article></aside>`
}

// cvStepText explains one step of the trace: a tone for the rail dot and a sentence.
function cvStepText(s, n) {
    if (s.skipped === 'joined') return ['', 'Already ran on another trigger’s path; this walk stops here.']
    if (s.error) return ['bad', `Error: ${esc(s.error)}${s.kind === 'if' ? ' — took the else branch.' : ''}`]
    switch (s.kind) {
    case 'trigger':
        if (s.matched === false) return ['warn', 'Its condition did not hold, so this lane did not run.']
        return ['accent', `Accepted the ${esc(cv.run.asNew ? 'created' : 'updated')} event${s.matched ? ' and its condition held' : ''}.`]
    case 'if':
        if (s.skipped === 'disabled') return ['warn', 'Disabled — took the else branch.']
        return s.matched ? ['ok', 'Matched — took the match branch.'] : ['warn', 'Did not match — took the else branch.']
    }
    const a = (cv.run.res.actions || []).find(x => x.node_id === s.node_id && x.no === s.no) || {}
    const what = `${tkActionLabel(s.kind)}${tkActionSummary(s.kind, a.output)}`
    const rec = cv.run.record
    switch (a.result) {
    case 'queued':    return ['ok', `${rec ? 'Asked to notify' : 'Would notify'}: ${what}.`]
    case 'would_run': return ['ok', `${rec ? 'Dry run, would have written' : 'Would write'} to ConnectWise: ${what}.`]
    case 'superseded': return ['warn', `Superseded${a.reason ? ` — ${esc(a.reason)}` : ''}.`]
    case 'ok':        return ['ok', s.kind === 'skip_notify' ? 'Later notify steps on this path are silenced.' : `Done: ${what}.`]
    case 'skipped':   return ['', `Skipped${a.reason ? ` — ${esc(a.reason)}` : ''}.`]
    case 'error':     return ['bad', `Error: ${esc(a.error || 'unknown')}`]
    }
    return ['', what]
}

// ── Save ─────────────────────────────────────────────────
// wfClientValidate catches what the inspector can show before a round trip: an incomplete step.
// Structural problems (loops, unreachable steps) come back from the server.
function wfClientValidate() {
    if (!wf.nodes.some(n => n.kind === 'trigger')) return { msg: 'A workflow needs at least one trigger' }
    for (const n of wf.nodes) {
        const name = n.title || cvKind(n.kind).label
        const bad = msg => ({ id: n.id, msg: `${name}: ${msg}` })
        switch (n.kind) {
        case 'trigger':
            if (!(n.events || []).length) return bad('pick at least one event')
            if (n._ui?.mode === 'builder') {
                const c = wfCompile(n._ui)
                if (c.errors.length) return bad(c.errors[0])
            }
            break
        case 'if':
            if (n._ui?.mode === 'builder') {
                const c = wfCompile(n._ui)
                if (c.errors.length) return bad(c.errors[0])
            }
            break
        case 'notify':
            if (n.notify?.channel !== 'resources_owner' && !n.notify?.recipient_id) return bad(`choose a ${wfChannelRecipientType(n.notify?.channel)}`)
            break
        case 'add_note':
            if (!n.add_note?.text?.trim()) return bad('note text is required')
            if (!n.add_note.internal && !n.add_note.discussion && !n.add_note.resolution) return bad('pick at least one note type')
            break
        case 'set_status':
            if (!n.set_status?.status_id) return bad('choose a status')
            break
        case 'set_priority':
            if (!n.set_priority?.priority_id) return bad('choose a priority')
            break
        case 'set_owner':
        case 'add_resource':
            if (!n[n.kind]?.member_id) return bad('choose a member')
            break
        case 'patch': {
            const err = wfPatchOpsError(n.patch?.ops)
            if (err) return bad(err)
            break
        }
        }
    }
    return null
}

// wfPatchOpsError returns a message when ops is not a JSON array of {op, path[, value]}.
function wfPatchOpsError(ops) {
    let parsed = ops
    if (typeof ops === 'string') {
        if (!ops.trim()) return 'patch operations are required'
        try { parsed = JSON.parse(ops) } catch (e) { return `patch JSON is invalid: ${e.message}` }
    }
    if (!Array.isArray(parsed) || !parsed.length) return 'patch must be a non-empty JSON array'
    for (const [k, op] of parsed.entries()) {
        if (!op || typeof op !== 'object') return `op ${k + 1} must be an object`
        if (!['add', 'replace', 'remove'].includes(op.op)) return `op ${k + 1}: op must be add, replace or remove`
        if (!op.path || typeof op.path !== 'string') return `op ${k + 1}: path is required`
        if (op.op !== 'remove' && op.value === undefined) return `op ${k + 1}: value is required`
    }
    return null
}

// wfNodeForServer normalizes one node into what the API accepts: editor-only state dropped, empty
// optional settings removed, patch ops parsed.
function wfNodeForServer(n) {
    const out = { id: n.id, kind: n.kind, title: n.title, enabled: !!n.enabled, x: Math.round(n.x || 0), y: Math.round(n.y || 0) }
    if (n.kind === 'trigger') { out.events = n.events || []; if (n.condition) out.condition = n.condition }
    else if (n.kind === 'if') { if (n.condition) out.condition = n.condition }
    else if (n[n.kind]) {
        const s = JSON.parse(JSON.stringify(n[n.kind]))
        if (n.kind === 'notify') {
            if (s.recipient_id === null || s.recipient_id === undefined) delete s.recipient_id   // server rejects null for resources_owner
            if (!s.message) delete s.message
        }
        if (n.kind === 'patch' && typeof s.ops === 'string') {
            try { s.ops = JSON.parse(s.ops) } catch { /* server reports the syntax error */ }
        }
        out[n.kind] = s
    }
    return out
}

// wfPrepareForServer normalizes the whole document. It mutates and returns w; pass a copy when
// the editor state must be preserved.
function wfPrepareForServer(w) {
    w.nodes = (w.nodes || []).map(wfNodeForServer)
    w.edges = (w.edges || []).map(e => ({ id: e.id, from: e.from, to: e.to, port: e.port }))
    return w
}

// wfProblemFromDetail turns a server validation error into something the canvas can point at.
function wfProblemFromDetail(d) {
    if (d.node_id) {
        const n = wfNode(d.node_id)
        return { id: d.node_id, pos: d.pos, field: d.field, msg: `${n?.title || 'Step'}: ${d.field}: ${d.message}` }
    }
    if (d.edge_id) return { edge: d.edge_id, msg: `Wire: ${d.field}: ${d.message}` }
    return { msg: `${d.field}: ${d.message}` }
}

// cvShowProblem selects the offending step or wire, marks it, and says what is wrong.
function cvShowProblem(p) {
    toast(p.msg, 'error')
    if (p.edge) { cvSelectEdge(p.edge); return }
    if (!p.id) return
    cv.errNode = p.id
    cvSelect(p.id)
    if (p.field === 'condition' && typeof p.pos === 'number') {
        const ta = document.getElementById(`cond-${p.id}`)
        if (ta) { ta.focus(); ta.setSelectionRange(p.pos, p.pos) }
    }
}

async function saveWorkflow() {
    const bad = wfClientValidate()
    if (bad) { cvShowProblem(bad); return }

    const btn = document.getElementById('wf-save')
    if (btn) { btn.disabled = true; btn.textContent = 'Saving…' }
    try {
        const saved = await api('PUT', `/workflows/${wf.id}`, wfPrepareForServer(JSON.parse(JSON.stringify(wf))))
        const uis = Object.fromEntries(wf.nodes.map(n => [n.id, n._ui]))
        wf = wfNormalize(saved)
        for (const n of wf.nodes) if (n.kind === 'if') n._ui = uis[n.id] || wfDefaultUI()
        wfOriginal = JSON.stringify(wfStrip(wf))
        cv.errNode = null
        if (cv.sel && !wfNode(cv.sel)) cv.sel = null
        renderWorkflowEditor()
        toast('Workflow saved', 'success')
    } catch (e) {
        const d = e.data?.details?.[0]
        if (d) cvShowProblem(wfProblemFromDetail(d))
        else toast(e.message, 'error')
        if (btn) { btn.disabled = false; btn.textContent = 'Save' }
    }
}

tabLoaders.workflows = loadWorkflows
