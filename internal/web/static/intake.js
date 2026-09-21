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

async function loadIntake() {
    try {
        const [stats, rows, hourly] = await Promise.all([
            api('GET', '/intake/stats'),
            api('GET', `/intake?status=${encodeURIComponent(intakeFilter)}&limit=200`),
            api('GET', '/intake/hourly?days=7').catch(() => []),
        ])
        renderIntake(stats, rows || [], hourly || [])
        startIntakePoll()
    } catch (e) {
        setContent(errorState(e.message))
    }
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

function renderIntake(stats, rows, hourly = []) {
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

    const thead = '<th class="r">ID</th><th>Ticket</th><th>Action</th><th>Status</th><th class="r">Attempts</th><th>Received</th><th>Last error</th><th class="r">Actions</th>'
    const trs = rows.map(r => {
        const st = INTAKE_STATUSES.find(s => s.value === r.status) || { label: r.status, variant: '' }
        const failed = r.status === 'failed'
        return `<tr>
            <td class="r num muted">${r.id}</td>
            <td class="cell-primary"><a href="#tickets/${r.ticket_id}" class="num">#${r.ticket_id}</a></td>
            <td>${esc(r.action)}</td>
            <td>${badgeTag(st.label, st.variant)}</td>
            <td class="r num">${r.attempts}</td>
            <td class="muted nowrap">${fmtDateTime(r.received_at)}</td>
            <td class="muted" style="max-width:360px;overflow-wrap:anywhere">${esc(r.last_error || '—')}</td>
            <td class="r nowrap">${failed ? `
                <button class="btn btn-default btn-sm" onclick="intakeRetry(${r.id})">${icon('undo')}Retry</button>
                <button class="btn btn-ghost btn-sm" onclick="intakeDiscard(${r.id})">${icon('trash')}Discard</button>` : ''}</td>
        </tr>`
    })

    const emptyCopy = {
        failed:     ['Nothing has failed', 'A webhook lands here only after every retry failed. Retries run for about two hours before giving up.'],
        pending:    ['Nothing waiting', 'Webhooks wait here between attempts; most are processed within a second of arriving.'],
        processing: ['Nothing in flight', 'Rows appear here while a worker is fetching the ticket and running its workflow.'],
        done:       ['Nothing processed yet', 'Processed webhooks stay here for the log retention period, then are purged.'],
        discarded:  ['Nothing discarded', 'Failed webhooks an admin dropped are kept here until purged.'],
    }[intakeFilter] || ['Nothing here', '']

    setContent(`<div class="stack gap6">
        <div class="grid g4">${tiles}</div>
        ${intakeChart(hourly)}
        <p class="muted">${esc(last)} The table refreshes every few seconds.</p>
        ${tableCard(thead, trs, {
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
        loadIntake()
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
                loadIntake()
            } catch (e) { toast(e.message, 'error') }
        },
    })
}

function startIntakePoll() {
    stopIntakePoll()
    intakePollTimer = setInterval(async () => {
        if (currentTab !== 'intake') { stopIntakePoll(); return }
        if (document.getElementById('modal')?.classList.contains('on')) return
        try {
            const [stats, rows, hourly] = await Promise.all([
                api('GET', '/intake/stats'),
                api('GET', `/intake?status=${encodeURIComponent(intakeFilter)}&limit=200`),
                api('GET', '/intake/hourly?days=7').catch(() => []),
            ])
            renderIntake(stats, rows || [], hourly || [])
        } catch { stopIntakePoll() }
    }, 5000)
}

function stopIntakePoll() {
    clearInterval(intakePollTimer)
    intakePollTimer = null
}

tabLoaders.intake = loadIntake
