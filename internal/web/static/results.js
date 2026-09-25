// ─────────────────────────────────────────────────────────
// Workflow results
//
// Every intake that found an enabled workflow leaves a run summary (workflow_run) and its history
// events. The Results tab lists the summaries with filters carried in the hash
// (#results?board=..&outcome=..&from=..&to=..&ticket=..), and a run (#results/<run_id>) opens as the
// workflow's canvas in replay mode with the recorded path lit, the way Simulate draws it, plus the
// run's events underneath. Nothing here edits anything.
// ─────────────────────────────────────────────────────────
const RUN_OUTCOMES = [
    ['clean',           'Clean',           'ok'],
    ['errors',          'Errors',          'bad'],
    ['nobody_notified', 'Nobody notified', 'warn'],
    ['no_trigger',      'No trigger',      ''],
]
let rsFilter    = {}       // current filters, mirrored in the hash
let rsBoards    = []       // /cw/boards for the board filter
let rsRuns      = []       // runs shown so far (load more appends)
let rsPollTimer = null

function rsParseQuery(q) {
    const out = {}
    for (const [k, v] of new URLSearchParams(q || '')) if (v) out[k] = v
    return out
}

function rsQuery(f) {
    const p = new URLSearchParams()
    for (const k of ['board', 'outcome', 'from', 'to', 'ticket']) if (f[k]) p.set(k, f[k])
    const s = p.toString()
    return s ? `?${s}` : ''
}

// rsApiQuery turns the hash filters into the API's query string. Dates are local calendar days.
function rsApiQuery(f, extra = {}) {
    const p = new URLSearchParams({ limit: '50', ...extra })
    if (f.board)   p.set('board_id', f.board)
    if (f.outcome) p.set('outcome', f.outcome)
    if (f.ticket)  p.set('ticket_id', f.ticket)
    if (f.from)    p.set('from', new Date(`${f.from}T00:00:00`).toISOString())
    if (f.to)      { const d = new Date(`${f.to}T00:00:00`); d.setDate(d.getDate() + 1); p.set('to', d.toISOString()) }
    return `/workflows/runs?${p}`
}

// loadResults routes #results?… to the list and #results/<run_id> to a run.
async function loadResults(sub) {
    const q  = (sub || '').indexOf('?')
    const id = q < 0 ? sub : sub.slice(0, q)
    if (id) await loadWorkflowRun(id)
    else await loadWorkflowResults(q < 0 ? '' : sub.slice(q + 1))
}

async function loadWorkflowResults(query) {
    rsFilter = rsParseQuery(query)
    try {
        const [runs, boards] = await Promise.all([
            api('GET', rsApiQuery(rsFilter)),
            rsBoards.length ? rsBoards : api('GET', '/cw/boards').catch(() => []),
        ])
        rsBoards = boards || []
        rsRuns = runs || []
        renderWorkflowResults()
        rsStartPoll()
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function rsOutcomeBadge(o) {
    const m = RUN_OUTCOMES.find(x => x[0] === o) || [o, o, '']
    return badgeTag(m[1], m[2])
}

function renderWorkflowResults(paint = setContent) {
    const f = rsFilter
    const boardOpts = rsBoards.filter(b => !b.deleted).sort((a, b) => a.name.localeCompare(b.name))
        .map(b => `<option value="${b.id}"${String(b.id) === f.board ? ' selected' : ''}>${esc(b.name)}</option>`).join('')
    const toolbar = `
        <select class="select" aria-label="Board" onchange="rsSetFilter('board', this.value)"><option value="">All boards</option>${boardOpts}</select>
        <select class="select" aria-label="Outcome" onchange="rsSetFilter('outcome', this.value)"><option value="">Any outcome</option>${RUN_OUTCOMES.map(([v, l]) => `<option value="${v}"${v === f.outcome ? ' selected' : ''}>${l}</option>`).join('')}</select>
        <input class="input" type="date" aria-label="From date" value="${esc(f.from || '')}" onchange="rsSetFilter('from', this.value)" style="width:150px">
        <input class="input" type="date" aria-label="To date" value="${esc(f.to || '')}" onchange="rsSetFilter('to', this.value)" style="width:150px">
        <input class="input" type="number" min="1" placeholder="Ticket #" aria-label="Ticket number" value="${esc(f.ticket || '')}" onchange="rsSetFilter('ticket', this.value)" style="width:120px">
        ${Object.keys(f).length ? `<button class="btn btn-ghost btn-sm" onclick="rsClearFilters()">${icon('x')}Clear</button>` : ''}`

    // No column sorts: runs page by started_at ("Load older runs"), so any other order would only
    // hold for the pages loaded so far. Resizing, reordering and reset still apply.
    const columns = [
        { key: 'started',  label: 'Started',  cls: 'muted nowrap', cell: r => fmtDateTime(r.started_at) },
        { key: 'ticket',   label: 'Ticket',   cell: r => `<a class="link num" href="#tickets/${r.ticket_id}" onclick="event.stopPropagation()">#${r.ticket_id}</a>` },
        { key: 'board',    label: 'Board',    cls: 'cell-primary', cell: r => esc(r.board_name || r.workflow_name || `Board ${r.board_id}`) },
        { key: 'event',    label: 'Event',    cell: r => esc(r.event) },
        { key: 'steps',    label: 'Steps',    cls: 'num', cell: r => r.steps },
        { key: 'notified', label: 'Notified', cls: 'muted', cell: r => rsNotifSummary(r) },
        { key: 'writes',   label: 'Writes',   cls: 'num', cell: r => r.writes },
        { key: 'outcome',  label: 'Outcome',  cell: r => `<div class="row gap2 wrap">${rsOutcomeBadge(r.outcome)}${r.dry_run ? badgeTag('Dry run', 'warn') : ''}</div>` },
    ]

    const last = rsRuns[rsRuns.length - 1]
    const more = rsRuns.length >= 50 && last
        ? `<button class="btn btn-default btn-sm" onclick="rsLoadMore()">Load older runs</button>` : ''

    paint(dataTable({
        id: 'results', columns, rows: rsRuns,
        tr: r => `class="clickable" onclick="switchTab('results', '${r.run_id}')"`,
        toolbar,
        empty: emptyState('No runs match', Object.keys(f).length ? 'Loosen a filter or clear them.' : 'Runs appear here once a ticket event reaches an enabled workflow.', '', 'bolt'),
        foot: `<span>${rsRuns.length} run${rsRuns.length === 1 ? '' : 's'} shown · refreshes every few seconds</span>${more}`,
    }))
}

function rsNotifSummary(r) {
    // counts in the mono figure face with a hard space, so "1 sent" never reads as "1sent"
    const count = (n, word) => `<span class="num">${n}</span>&nbsp;${word}`
    const parts = []
    if (r.notif_sent)       parts.push(count(r.notif_sent, 'sent'))
    if (r.notif_would_send) parts.push(count(r.notif_would_send, 'would send'))
    if (r.notif_none)       parts.push(r.notif_none === 1 ? 'nobody to notify' : count(r.notif_none, 'steps with nobody to notify'))
    return parts.length ? parts.join(' · ') : '—'
}

function rsSetFilter(key, value) {
    const f = { ...rsFilter }
    if (value) f[key] = value; else delete f[key]
    switchTab('results', rsQuery(f) || null)
}

function rsClearFilters() { switchTab('results') }

async function rsLoadMore() {
    const last = rsRuns[rsRuns.length - 1]
    if (!last) return
    try {
        const older = await api('GET', rsApiQuery(rsFilter, { before: last.started_at }))
        rsRuns = rsRuns.concat(older || [])
        renderWorkflowResults(refreshContent)
    } catch (e) { toast(e.message, 'error') }
}

function rsStartPoll() {
    rsStopPoll()
    const onList = () => currentTab === 'results' && !currentHash.startsWith('results/')
    rsPollTimer = setInterval(async () => {
        if (!onList()) { rsStopPoll(); return }
        if (document.getElementById('modal')?.classList.contains('on')) return
        try {
            // refresh only the first page; older pages the user loaded stay as they are
            const f = rsFilter
            const fresh = await api('GET', rsApiQuery(f)) || []
            if (!onList() || f !== rsFilter) return   // the user left or refiltered while the fetch was out
            const seen = new Set(fresh.map(r => r.run_id))
            rsRuns = fresh.concat(rsRuns.filter(r => !seen.has(r.run_id)))
            renderWorkflowResults(refreshContent)
        } catch { rsStopPoll() }
    }, 5000)
}

function rsStopPoll() {
    clearInterval(rsPollTimer)
    rsPollTimer = null
}

// ── Run detail: the canvas in replay ─────────────────────
async function loadWorkflowRun(runID) {
    try {
        const d = await api('GET', `/workflows/runs/${encodeURIComponent(runID)}`)
        const run = d.run
        const wfEvent = (d.events || []).find(e => e.kind === 'workflow')
        const steps = wfEvent?.payload?.steps || []
        const actions = (d.events || []).filter(e => e.kind === 'action').map(e => e.payload)

        let canvas = ''
        if (d.workflow) {
            const [recips, members] = await Promise.all([api('GET', '/webex/rooms').catch(() => []), api('GET', '/cw/members').catch(() => [])])
            wfRecipients = recips || []
            wfMembers = (members || []).filter(m => !m.deleted)
            wf = wfNormalize(d.workflow)
            tabGuard = null
            cvReset()
            cv.replay = true
            cv.rail = false
            cv.run = { record: true, ticket: run.ticket_id, asNew: run.event === 'created', res: { actions, workflow: wfEvent?.payload }, steps, step: steps.length, outcome: run.outcome, dryRun: run.dry_run }
            canvas = `<div class="canvas wf-canvas run-canvas" id="cv">
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
            </div>`
        } else {
            canvas = `<div class="callout warn">${icon('alert')}<div class="body"><b>This workflow no longer exists</b>The steps and events below are the record of the run.</div></div>`
        }

        // steps whose node has since been removed or renamed cannot be lit on the canvas
        const gone = d.workflow ? steps.filter(s => !wf.nodes.some(n => n.id === s.node_id)) : steps
        const goneHTML = gone.length ? `<div class="card card-pad stack gap2">
            <div class="eyebrow">Steps no longer on the canvas</div>
            <div class="row wrap gap3">${gone.map(tkStepChip).join('')}</div>
        </div>` : ''

        const shown = (d.events || []).filter(e => e.kind !== 'workflow')
        const eventsHTML = `<div class="card card-pad stack gap3">
            <div class="eyebrow">What happened</div>
            <div class="events">${shown.map(tkEventHTML).join('') || '<span class="cell-sub">No events recorded</span>'}</div>
        </div>`

        setContent(`<div class="stack gap5">
            ${backRow(`results${rsQuery(rsFilter)}`, 'Results', run.run_id.slice(0, 8))}
            <div class="row spread wrap gap3">
                <div class="row gap3 wrap">
                    <h3 style="margin:0">${esc(run.board_name || run.workflow_name || `Board ${run.board_id}`)}</h3>
                    <a class="link num" href="#tickets/${run.ticket_id}">#${run.ticket_id}</a>
                    ${badgeTag(run.event, 'accent')}
                    ${rsOutcomeBadge(run.outcome)}
                    ${run.dry_run ? badgeTag('Dry run', 'warn') : ''}
                    <span class="muted">${fmtDateTime(run.started_at)} · ${run.duration_ms} ms · ${esc(run.source)}</span>
                </div>
                ${d.workflow && run.workflow_id ? `<button class="btn btn-default btn-sm" onclick="openWorkflow(${run.workflow_id})">${icon('edit')}Open workflow</button>` : ''}
            </div>
            ${canvas}
            ${goneHTML}
            ${eventsHTML}
        </div>`)
        setCrumbHere(`Run ${run.run_id.slice(0, 8)}`)

        if (d.workflow) {
            cvMount()
            cvRenderGraph()
            cvRenderRail()
            cvRenderSide()
            cvApplyView()
            cvFit()
        }
    } catch (e) {
        setContent(backRow('results', 'Results') + errorState(e.message))
    }
}

tabLoaders.results = loadResults
