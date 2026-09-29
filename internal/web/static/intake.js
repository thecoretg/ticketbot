// ─────────────────────────────────────────────────────────
// Intake: the persisted webhook queue (admin)
//
// Every ConnectWise callback lands in webhook_intake and is drained by the
// workers. This page shows the queue's shape and lets an admin retry or
// discard rows every attempt failed on.
// ─────────────────────────────────────────────────────────
const INTAKE_STATUSES = [
    { value: 'failed',     label: 'Failed',     variant: 'bad' },
    { value: 'pending',    label: 'Pending',    variant: 'warn' },
    { value: 'processing', label: 'Processing', variant: 'info' },
    { value: 'done',       label: 'Done',       variant: 'ok' },
    { value: 'discarded',  label: 'Discarded',  variant: '' },
]
let intakeFilter    = 'failed'
let intakePollTimer = null

function fetchIntake() {
    return Promise.all([
        api('GET', '/intake/stats'),
        api('GET', `/intake?status=${encodeURIComponent(intakeFilter)}&limit=200`),
        api('GET', '/intake/hourly?days=7').catch(() => []),
    ])
}

async function loadIntake() {
    try {
        const [stats, rows, hourly] = await fetchIntake()
        renderIntake(stats, rows || [], hourly || [])
        startIntakePoll()
    } catch (e) {
        setContent(errorState(e.message))
    }
}

// refreshIntake redraws the page in place, for the poll and after a retry or discard.
async function refreshIntake() {
    const [stats, rows, hourly] = await fetchIntake()
    if (currentTab !== 'intake') return   // the user left while the fetch was out
    renderIntake(stats, rows || [], hourly || [], refreshContent)
}

// intakeChart draws webhooks per hour for the last seven days as an inline SVG bar chart. It is
// how the stale-webhook threshold gets tuned: quiet hours show as gaps.
function intakeChart(hourly) {
    const hours = 7 * 24
    const end = new Date(); end.setMinutes(0, 0, 0)
    const counts = new Array(hours).fill(0)
    const byHour = new Map(hourly.map(h => [new Date(h.hour).getTime(), h.count]))
    for (let i = 0; i < hours; i++) {
        const t = end.getTime() - (hours - 1 - i) * 3600e3
        counts[i] = byHour.get(t) || 0
    }
    const max = Math.max(1, ...counts)
    const W = 840, H = 120, pad = 4, bw = (W - pad * 2) / hours
    const bars = counts.map((c, i) => {
        const h = Math.round((c / max) * (H - 20))
        const t = new Date(end.getTime() - (hours - 1 - i) * 3600e3)
        return `<rect x="${(pad + i * bw).toFixed(1)}" y="${H - h}" width="${Math.max(1, bw - 1).toFixed(1)}" height="${h}" fill="var(--chart-1)"><title>${esc(fmtDateTime(t.toISOString()))}: ${c}</title></rect>`
    }).join('')
    const days = []
    for (let i = 0; i < hours; i += 24) {
        const t = new Date(end.getTime() - (hours - 1 - i) * 3600e3)
        days.push(`<text x="${(pad + i * bw).toFixed(1)}" y="${H - 4}" font-size="10" fill="var(--muted)">${t.toLocaleDateString(undefined, { weekday: 'short' })}</text>`)
    }
    return `<div class="card card-pad stack gap2">
        <div class="row spread"><span class="eyebrow">Webhooks per hour, last 7 days</span><span class="muted" style="font-size:var(--text-xs)">peak ${max}/h</span></div>
        <svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" role="img" aria-label="Webhooks received per hour over the last seven days">${bars}${days.join('')}</svg>
    </div>`
}

// intakeCatchupCard reports the missed-webhook check: how often ConnectWise changed a ticket
// without calling back, and whether the check itself is healthy.
function intakeCatchupCard(stats) {
    const c = stats?.catchup
    const minutes = appConfig?.catchup_interval_minutes
    const week = stats?.catchup_last_7_days || 0
    let summary
    if (minutes === 0) summary = 'Off. Turn it on under Config.'
    else if (!c?.last_run_at) summary = `Runs every ${minutes || 15} minutes; it has not run yet.`
    else summary = `Runs every ${minutes || 15} minutes. Last run ${fmtDateTime(c.last_run_at)}, queued ${c.last_queued}.`
    const failed = c?.last_error
        ? `<div class="callout warn">${icon('alert')}<div class="body"><b>The last run failed</b>${esc(c.last_error.replace(/\.?$/, '.'))} The next run checks the same window again.</div></div>`
        : ''
    return `<article class="card">
        <div class="card-head">
            <div><h3>Missed-webhook check</h3><p>Tickets ConnectWise changed without sending a webhook, queued as if it had.</p></div>
            <span class="badge outline">${week} in 7 days</span>
        </div>
        <div class="card-body stack gap3">
            <p class="muted">${esc(summary)}</p>
            ${failed}
        </div>
    </article>`
}

// Catch-up rows carry no ConnectWise action of their own; name them for what they are.
const INTAKE_ACTION_LABELS = { catchup: 'missed webhook' }

function renderIntake(stats, rows, hourly = [], paint = setContent) {
    const counts = stats?.counts || {}
    const tiles = INTAKE_STATUSES.filter(s => s.value !== 'discarded').map(s => `
        <article class="card stat">
            <div class="stat-label eyebrow">${s.label}</div>
            <div class="stat-value">${counts[s.value] || 0}</div>
        </article>`).join('')
    const last = stats?.last_received_at
        ? `Last webhook received ${fmtDateTime(stats.last_received_at)}.`
        : 'No webhook has been received yet.'

    const filter = `<label class="row gap2"><span class="muted">Show</span>
        <select class="select" aria-label="Queue status" onchange="intakeSetFilter(this.value)">
            ${INTAKE_STATUSES.map(s => `<option value="${s.value}" ${s.value === intakeFilter ? 'selected' : ''}>${s.label}</option>`).join('')}
        </select></label>`

    const status = r => INTAKE_STATUSES.find(s => s.value === r.status) || { label: r.status, variant: '' }
    const columns = [
        { key: 'id', label: 'ID', cls: 'num muted', sort: r => r.id, cell: r => r.id },
        { key: 'ticket', label: 'Ticket', cls: 'cell-primary', sort: r => r.ticket_id,
          cell: r => `<a href="#tickets/${r.ticket_id}" class="num">#${r.ticket_id}</a>` },
        { key: 'action', label: 'Action', sort: r => r.action, cell: r => esc(INTAKE_ACTION_LABELS[r.action] || r.action) },
        { key: 'status', label: 'Status', sort: r => status(r).label, cell: r => badgeTag(status(r).label, status(r).variant) },
        { key: 'attempts', label: 'Attempts', cls: 'num', sort: r => r.attempts, cell: r => r.attempts },
        { key: 'received', label: 'Received', cls: 'muted nowrap', firstDir: 'desc', sort: r => tblTime(r.received_at),
          cell: r => fmtDateTime(r.received_at) },
        { key: 'last_error', label: 'Last error', cls: 'muted', attrs: () => 'style="max-width:360px;overflow-wrap:anywhere"',
          sort: r => r.last_error, cell: r => esc(r.last_error || '—') },
    ]
    // only a failed row can be retried or discarded; every other row has no kebab
    const menu = r => r.status === 'failed' ? [
        { label: 'Retry', icon: 'undo', run: () => intakeRetry(r.id) },
        { label: 'Discard', icon: 'trash', danger: true, run: () => intakeDiscard(r.id) },
    ] : []

    const emptyCopy = {
        failed:     ['Nothing has failed', 'A webhook lands here only after every retry failed. Retries run for about two hours before giving up.'],
        pending:    ['Nothing waiting', 'Webhooks wait here between attempts; most are processed within a second of arriving.'],
        processing: ['Nothing in flight', 'Rows appear here while a worker is fetching the ticket and running its workflow.'],
        done:       ['Nothing processed yet', 'Processed webhooks stay here for the intake retention period, then are purged.'],
        discarded:  ['Nothing discarded', 'Failed webhooks an admin dropped are kept here until purged.'],
    }[intakeFilter] || ['Nothing here', '']

    paint(`<div class="stack gap6">
        <div class="grid g4">${tiles}</div>
        ${intakeChart(hourly)}
        ${intakeCatchupCard(stats)}
        <p class="muted">${esc(last)} The table refreshes every few seconds.</p>
        ${dataTable({
            id: 'intake', columns, rows, menu,
            toolbar: filter,
            empty: emptyState(emptyCopy[0], emptyCopy[1], '', 'inbox'),
            foot: `<span>${rows.length} row${rows.length === 1 ? '' : 's'}</span>`,
        })}
    </div>`)
}

function intakeSetFilter(value) {
    intakeFilter = value
    loadIntake()
}

async function intakeRetry(id) {
    try {
        await api('POST', `/intake/${id}/retry`)
        toast('Webhook re-queued', 'success')
        refreshIntake().catch(e => toast(e.message, 'error'))
    } catch (e) { toast(e.message, 'error') }
}

function intakeDiscard(id) {
    confirmModal({
        title: 'Discard this webhook?',
        body: '<b>The change it carried is not applied</b>The ticket is left as ticketbot last stored it until its next webhook or a sync.',
        confirmLabel: 'Discard',
        onConfirm: async () => {
            try {
                await api('POST', `/intake/${id}/discard`)
                toast('Webhook discarded', 'success')
                refreshIntake().catch(e => toast(e.message, 'error'))
            } catch (e) { toast(e.message, 'error') }
        },
    })
}

function startIntakePoll() {
    stopIntakePoll()
    intakePollTimer = setInterval(async () => {
        if (currentTab !== 'intake') { stopIntakePoll(); return }
        if (document.getElementById('modal')?.classList.contains('on')) return
        try { await refreshIntake() } catch { stopIntakePoll() }
    }, 5000)
}

function stopIntakePoll() {
    clearInterval(intakePollTimer)
    intakePollTimer = null
}

tabLoaders.intake = loadIntake
