// ─────────────────────────────────────────────────────────
// Tickets
// ─────────────────────────────────────────────────────────
let tkFilters     = { board_id: '', status_id: '', closed: '', q: '', page: 1, page_size: 25 }
let tkBoards      = []
let tkSearchTimer = null
let tkRequestSeq  = 0      // drops stale responses when filters change quickly
let tkShowNoops   = false  // history: also show runs where nothing happened (no workflow, no match, loop guard)
let tkDetail      = null   // last loaded ticket detail, for re-rendering the history toggle
let tkWfLinkHTML  = ''     // the "Open workflow" control for tkDetail, kept across re-renders

async function loadTickets(sub) {
    if (sub && /^\d+$/.test(sub)) {
        await loadTicketDetail(parseInt(sub))
        return
    }
    await renderTicketsPage()
}

// ── List ─────────────────────────────────────────────────
async function renderTicketsPage() {
    if (!tkBoards.length) {
        try { tkBoards = (await api('GET', '/cw/boards')) || [] } catch {}
    }
    const boardOpts = tkBoards.filter(b => !b.deleted).map(b =>
        `<option value="${b.id}"${String(b.id) === tkFilters.board_id ? ' selected' : ''}>${esc(b.name)}</option>`).join('')

    setContent(
    `<div class="card">
        <div class="filter-bar">
            <select id="tk-board" class="select" style="min-width:170px" onchange="tkBoardChanged(this.value)" aria-label="Filter by board">
                <option value="">All boards</option>${boardOpts}
            </select>
            <select id="tk-status" class="select" style="min-width:150px" onchange="tkSetFilter('status_id', this.value)" ${tkFilters.board_id ? '' : 'disabled'} aria-label="Filter by status">
                <option value="">All statuses</option>
            </select>
            <select id="tk-closed" class="select" style="min-width:150px" onchange="tkSetFilter('closed', this.value)" aria-label="Filter by open or closed">
                <option value=""${tkFilters.closed === '' ? ' selected' : ''}>Open + closed</option>
                <option value="false"${tkFilters.closed === 'false' ? ' selected' : ''}>Open only</option>
                <option value="true"${tkFilters.closed === 'true' ? ' selected' : ''}>Closed only</option>
            </select>
            <div class="input-group">
                ${icon('search')}
                <input id="tk-search" class="input" type="text" style="width:240px" placeholder="Search summary or #…" value="${esc(tkFilters.q)}" oninput="tkSetSearch(this.value)" aria-label="Search tickets">
            </div>
            <div class="grow"></div>
            <button class="btn btn-default btn-sm" onclick="refreshTicketTable()">Refresh</button>
        </div>
        <div id="tk-table"><div class="stack gap3" style="padding:var(--s5)" aria-busy="true">
            <div class="skeleton" style="height:13px;width:100%"></div>
            <div class="skeleton" style="height:13px;width:92%"></div>
            <div class="skeleton" style="height:13px;width:84%"></div>
        </div></div>
        <div id="tk-pager" class="card-foot"></div>
    </div>`)

    if (tkFilters.board_id) await tkLoadStatuses(tkFilters.board_id)
    await refreshTicketTable()
}

async function tkLoadStatuses(boardID) {
    const sel = document.getElementById('tk-status')
    if (!sel) return
    try {
        const statuses = (await api('GET', `/cw/boards/${boardID}/statuses`)) || []
        sel.innerHTML = '<option value="">All statuses</option>' + statuses.map(s =>
            `<option value="${s.id}"${String(s.id) === tkFilters.status_id ? ' selected' : ''}>${esc(s.name)}</option>`).join('')
        sel.disabled = false
    } catch (e) { toast(e.message, 'error') }
}

async function tkBoardChanged(value) {
    tkFilters.board_id  = value
    tkFilters.status_id = ''
    tkFilters.page      = 1
    const sel = document.getElementById('tk-status')
    if (value) {
        await tkLoadStatuses(value)
    } else if (sel) {
        sel.innerHTML = '<option value="">All statuses</option>'
        sel.disabled  = true
    }
    await refreshTicketTable()
}

function tkSetFilter(key, value) {
    tkFilters[key] = value
    tkFilters.page = 1
    refreshTicketTable()
}

function tkSetSearch(value) {
    clearTimeout(tkSearchTimer)
    tkSearchTimer = setTimeout(() => tkSetFilter('q', value.trim()), 300)
}

function tkPage(delta) {
    tkFilters.page = Math.max(1, tkFilters.page + delta)
    refreshTicketTable()
}

async function refreshTicketTable() {
    const table = document.getElementById('tk-table')
    const pager = document.getElementById('tk-pager')
    if (!table) return

    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(tkFilters)) {
        if (v !== '' && v !== null && v !== undefined) params.set(k, v)
    }

    const seq = ++tkRequestSeq
    let page
    try {
        page = await api('GET', `/tickets?${params}`)
    } catch (e) {
        if (seq !== tkRequestSeq) return
        table.innerHTML = `<div class="empty">
            <div class="empty-art">${icon('alert')}</div>
            <div class="stack gap2"><h3>Could not load tickets</h3><p>${esc(e.message)}</p></div>
            <button class="btn btn-default btn-sm" onclick="refreshTicketTable()">Try again</button>
        </div>`
        pager.innerHTML = ''
        return
    }
    if (seq !== tkRequestSeq || !document.getElementById('tk-table')) return

    const items = page?.items || []
    const thead = `<th class="r">ID</th><th>Summary</th><th>Board</th><th>Status</th><th>Company</th><th>Owner</th><th>Updated</th>`
    const rows  = items.map(t => `<tr class="clickable" onclick="openTicket(${t.id})">
        <td class="r"><a class="link num" href="${esc(t.cw_url)}" target="_blank" rel="noopener" onclick="event.stopPropagation()" data-tip="Open in ConnectWise">#${t.id}</a></td>
        <td class="cell-ellipsis cell-primary" title="${esc(t.summary)}">${esc(t.summary)}${t.deleted ? ' ' + badgeTag('Deleted', 'bad') : ''}</td>
        <td class="nowrap">${esc(t.board_name)}</td>
        <td class="nowrap">${esc(t.status_name)}${t.closed_flag ? ' ' + badgeTag('Closed', '') : ''}</td>
        <td>${esc(t.company_name)}</td>
        <td>${esc(t.owner_name || '—')}</td>
        <td class="nowrap muted">${fmtDateTime(t.updated_on)}</td>
    </tr>`)

    table.innerHTML = rows.length
        ? `<div class="table-wrap"><table class="tbl">
            <thead><tr>${thead}</tr></thead><tbody>${rows.join('')}</tbody>
        </table></div>`
        : emptyState('No tickets match these filters',
            'Tickets appear here once ConnectWise sends a webhook, or after a ticket sync.',
            `<button class="btn btn-default btn-sm" onclick="tkClearFilters()">Clear filters</button>`, 'inbox')

    const total    = page?.total || 0
    const size     = page?.page_size || tkFilters.page_size
    const current  = page?.page || 1
    const lastPage = Math.max(1, Math.ceil(total / size))
    const from     = total ? (current - 1) * size + 1 : 0
    const to       = Math.min(current * size, total)
    tkFilters.page = current

    pager.innerHTML = total ? `
        <span>Showing <span class="num">${from}–${to}</span> of <span class="num">${total}</span></span>
        <div class="pagination">
            <button onclick="tkPage(-1)" ${current <= 1 ? 'disabled' : ''} aria-label="Previous page">${icon('arrowL')}</button>
            <span class="muted" style="padding:0 var(--s2)">Page <span class="num">${current}</span> of <span class="num">${lastPage}</span></span>
            <button onclick="tkPage(1)" ${current >= lastPage ? 'disabled' : ''} aria-label="Next page">${icon('arrowR')}</button>
        </div>` : ''
}

// tkClearFilters resets the toolbar to "everything", from the empty state.
function tkClearFilters() {
    tkFilters = { board_id: '', status_id: '', closed: '', q: '', page: 1, page_size: tkFilters.page_size }
    renderTicketsPage()
}

function openTicket(id) {
    switchTab('tickets', String(id))
}

function tkBack() {
    switchTab('tickets')
}

// ── Detail ───────────────────────────────────────────────
async function loadTicketDetail(id) {
    let data
    try {
        data = await api('GET', `/tickets/${id}`)
    } catch (e) {
        setContent(backRow('tickets', 'Tickets', `#${id}`) + errorState(e.message))
        return
    }
    tkDetail     = data
    tkWfLinkHTML = ''
    renderTicketDetail(data)
    setCrumbHere(`#${data.ticket.id}`)
    tkLoadWorkflowLink(data.ticket)
}

// tkLoadWorkflowLink adds an "Open workflow" button once we know whether the board has one.
async function tkLoadWorkflowLink(t) {
    const slot = document.getElementById('tk-workflow-link')
    if (!slot || !t?.board_id) return
    let html = ''
    try {
        const w = await api('GET', `/workflows/board/${t.board_id}`)
        if (w?.id) html = `<button class="btn btn-default" onclick="openWorkflow(${w.id})" data-tip="Edit the ${esc(t.board_name)} workflow">${icon('bolt')}Open workflow${w.dry_run ? ' ' + badgeTag('Dry run', 'warn') : ''}${w.enabled ? '' : ' ' + badgeTag('Disabled', '')}</button>`
    } catch (e) {
        if (e.status === 404) html = `<button class="btn btn-default" onclick="tkCreateWorkflow(${t.board_id})" data-tip="This board has no workflow yet">${icon('plus')}Create workflow</button>`
    }
    tkWfLinkHTML = html
    const el = document.getElementById('tk-workflow-link')
    if (el) el.innerHTML = html
}

function tkCreateWorkflow(boardID) {
    switchTab('workflows')
    showNewWorkflowModal(boardID)
}

function tkToggleNoops(on) {
    tkShowNoops = on
    if (tkDetail) renderTicketDetail(tkDetail)
}

// tkIsNoop reports events that only say "nothing happened": a board with no workflow, a disabled
// workflow, a run that reached no action step, or a self-authored update the loop guard skipped.
// Anything that ran, would run (dry run) or failed stays visible. Runs recorded before the graph
// editor carry `rules` instead of `steps`.
function tkIsNoop(ev) {
    const p = ev.payload || {}
    switch (ev.kind) {
    case 'loop_guard':
        return true
    case 'workflow':
        if (!p.found || !p.enabled) return true
        if (p.steps) return !p.steps.some(s => s.error || (s.kind !== 'trigger' && s.kind !== 'if' && !s.skipped))
        return !(p.rules || []).some(r => r.matched || r.error)
    }
    return false
}

function renderTicketDetail(d) {
    const t   = d.ticket
    const all = d.events || []
    const evs = tkShowNoops ? all : all.filter(ev => !tkIsNoop(ev))
    const hidden = all.length - evs.length
    const contact = d.contact ? [d.contact.first_name, d.contact.last_name].filter(Boolean).join(' ') : '—'
    const resources = (d.resources || []).map(m => memberLabel(m)).join(', ') || (t.resources || '—')

    const meta = (label, value) => `<div><div class="eyebrow">${label}</div><div class="val">${value}</div></div>`

    setContent(`${backRow('tickets', 'Tickets', `#${t.id}`)}
    <header class="page-head row spread wrap gap4">
        <div>
            <h1 class="page-title">${esc(t.summary)}</h1>
            <p class="page-sub row gap2 wrap">
                <a class="link num" href="${esc(t.cw_url)}" target="_blank" rel="noopener">#${t.id} in ConnectWise</a>
                ${t.deleted ? badgeTag('Deleted', 'bad') : ''}
            </p>
        </div>
        <div class="row gap2 wrap">
            <span id="tk-workflow-link">${tkWfLinkHTML}</span>
            <button class="btn btn-default" onclick="loadTicketDetail(${t.id})">Refresh</button>
        </div>
    </header>
    <div class="meta-grid">
        ${meta('Board', esc(t.board_name))}
        ${meta('Status', esc(t.status_name) + (t.closed_flag ? ' ' + badgeTag('Closed', '') : ''))}
        ${meta('Company', esc(t.company_name))}
        ${meta('Priority', esc(t.priority_name || '—'))}
        ${meta('Owner', esc(t.owner_name || '—'))}
        ${meta('Contact', esc(contact))}
        ${meta('Resources', esc(resources))}
        ${meta('Type', esc([t.type_name, t.subtype_name, t.item_name].filter(Boolean).join(' / ') || '—'))}
        ${meta('Updated', fmtDateTime(t.updated_on) + (t.updated_by ? ` <span class="muted">by ${esc(t.updated_by)}</span>` : ''))}
        ${meta('Added', fmtDateTime(t.added_on))}
    </div>
    <div class="section-head row spread wrap gap4">
        <div class="row gap3 wrap" style="align-items:baseline">
            <h3>History</h3>
            <p>${evs.length} event${evs.length === 1 ? '' : 's'}, oldest first</p>
        </div>
        ${checkbox(`Show runs where nothing happened${!tkShowNoops && hidden ? ` <span class="muted">(${hidden})</span>` : ''}`,
            'onchange="tkToggleNoops(this.checked)"', tkShowNoops)}
    </div>
    ${evs.length
        ? `<div class="events">${evs.map(tkEventHTML).join('')}</div>`
        : `<div class="card">${emptyState('No events recorded',
            'Events appear here when ConnectWise sends a webhook for this ticket and a workflow runs.',
            '', 'clock')}</div>`}`)
}

function memberLabel(m) {
    const name = [m.first_name, m.last_name].filter(Boolean).join(' ')
    return name || m.identifier || ''
}

// ── Event history ────────────────────────────────────────
// Each entry is an .event card: a dot on the rail whose tone says what
// happened, plus a card with the detail.
function tkEventHTML(ev) {
    const p   = ev.payload || {}
    let title = ev.kind, body = '', tone = ''

    switch (ev.kind) {
    case 'created':
        title = 'Ticket created'
        tone  = 'ok'
        body  = tkChangeBody(p)
        break
    case 'updated':
        title = 'Ticket updated'
        tone  = 'accent'
        body  = tkChangeBody(p)
        break
    case 'deleted':
        title = 'Ticket deleted in ConnectWise'
        tone  = 'bad'
        break
    case 'loop_guard':
        title = 'Rules skipped: self-authored update'
        body  = `<div class="event-detail">${esc(tkLoopReason(p.reason))}${p.identifier ? ` (<code class="code inline">${esc(p.identifier)}</code>)` : ''}</div>`
        break
    case 'workflow':
        title = p.found ? `Workflow: ${esc(p.workflow_name || '')}` : 'No workflow for this board'
        tone  = p.found && p.enabled ? 'accent' : ''
        if (p.found && !p.enabled) body = '<div class="event-detail">Workflow is disabled</div>'
        else if (p.steps?.length) body = `<div class="row wrap gap3">${p.steps.map(tkStepChip).join('')}</div>`
        else if (p.steps) body = `<div class="event-detail">No trigger listens for a ${esc(p.event || '')} ticket</div>`
        else if (p.rules?.length) body = `<div class="row wrap gap3">${p.rules.map(tkRuleChip).join('')}</div>`
        break
    case 'action':
        title = `${esc(p.title || p.rule_name || 'Step')} · ${tkActionLabel(p.kind)}`
        tone  = p.result === 'error' ? 'bad' : 'warn'
        body  = tkActionBody(p, ev.dry_run)
        break
    case 'notification':
        title = `${esc(p.title || p.rule_name || 'Notify')} → ${esc(p.recipient_name || '?')}`
        tone  = p.result === 'error' ? 'bad' : 'warn'
        body  = tkNotificationBody(p, ev.dry_run)
        break
    case 'error':
        title = `Error${p.stage ? ` during ${esc(p.stage)}` : ''}`
        tone  = 'bad'
        body  = `<div class="event-error">${esc(p.error || '')}</div>`
        break
    }

    return `<div class="event ${tone}">
        <div class="event-card">
            <div class="event-head">
                <span class="title">${title}</span>
                ${ev.dry_run ? badgeTag('Dry run', 'warn') : ''}
                <span class="src">${esc(ev.source)}</span>
                <span class="time">${fmtDateTime(ev.occurred_at)}</span>
            </div>
            ${body ? `<div class="event-body stack gap2">${body}</div>` : ''}
        </div>
    </div>`
}

function tkChangeBody(p) {
    let html = ''
    if (p.updated_by) html += `<div class="event-detail">by ${esc(p.updated_by)}</div>`
    if (p.changes?.length) html += tkDiffTableHTML(p.changes)
    if (p.new_note) {
        const n = p.new_note
        html += `<div class="note">
            <div class="note-author">${esc(n.author_name || n.author_identifier || 'Unknown')}${n.internal ? ' ' + badgeTag('Internal', '') : ''}</div>
            <div class="note-text">${esc(n.preview || '')}</div>
        </div>`
    }
    return html
}

function tkDiffTableHTML(changes) {
    const val = v => {
        if (v === null || v === undefined || v === '') return '—'
        if (Array.isArray(v)) return v.length ? esc(v.join(', ')) : '—'
        if (typeof v === 'object') return v.id ? esc(v.name ? `${v.name} (${v.id})` : String(v.id)) : '—'
        return esc(String(v))
    }
    return `<table class="diff-table"><thead><tr><th>Field</th><th>Was</th><th>Now</th></tr></thead><tbody>${
        changes.map(c => `<tr>
            <td class="field">${esc(c.field)}</td>
            <td><span class="diff-old">${val(c.old)}</span></td>
            <td><span class="diff-new">${val(c.new)}</span></td>
        </tr>`).join('')
    }</tbody></table>`
}

// tkStepChip summarizes one node the run visited: what it decided, or why it was skipped.
function tkStepChip(s) {
    let variant = '', label = ''
    if (s.error)                     { variant = 'bad'; label = 'error' }
    else if (s.skipped === 'joined') { label = 'already ran' }
    else if (s.skipped)              { label = s.skipped }
    else if (s.kind === 'trigger')   { variant = 'accent'; label = 'fired' }
    else if (s.kind === 'if')        { variant = s.matched ? 'ok' : ''; label = s.matched ? 'match' : 'else' }
    else                             { variant = 'ok'; label = 'ran' }
    const tip = s.error ? ` data-tip="${esc(s.error)}"` : ''
    return `<span class="row gap2"${tip}><span class="num muted">${s.no}</span><span class="cell-sub">${esc(s.title)}</span>${badgeTag(label, variant)}</span>`
}

// tkRuleChip renders a step from a run recorded before the graph editor (payload.rules).
function tkRuleChip(r) {
    let variant = '', label = 'no match'
    if (r.error)        { variant = 'bad'; label = 'error' }
    else if (r.skipped) { variant = '';    label = r.skipped === 'trigger' ? 'trigger mismatch' : r.skipped }
    else if (r.matched) { variant = 'ok';  label = r.stopped ? 'matched · stop' : 'matched' }
    const tip = r.error ? ` data-tip="${esc(r.error)}"` : ''
    return `<span class="row gap2"${tip}><span class="cell-sub">${esc(r.rule_name)}</span>${badgeTag(label, variant)}</span>`
}

function tkActionLabel(kind) {
    return {
        notify: 'Notify', add_note: 'Add note', skip_notify: 'Skip notify',
        set_status: 'Set status', set_priority: 'Set priority', set_owner: 'Set owner',
        add_resource: 'Add resource', patch: 'Patch ticket',
    }[kind] || esc(kind || 'Action')
}

// tkActionSummary is the short " → target" suffix shown after an action label.
function tkActionSummary(kind, o) {
    o = o || {}
    switch (kind) {
    case 'notify':       return o.target ? ` → ${esc(tkTargetLabel(o.target))}${o.custom_message ? ' <span class="muted">(custom message)</span>' : ''}` : ''
    case 'add_note':     return o.text ? `: <span class="muted">${esc(o.text)}</span>` : ''
    case 'set_status':   return ` → ${esc(o.status_name || `status ${o.status_id ?? '?'}`)}`
    case 'set_priority': return ` → ${esc(o.priority_name || `priority ${o.priority_id ?? '?'}`)}`
    case 'set_owner':
    case 'add_resource': return ` → ${esc(o.identifier || `member ${o.member_id ?? '?'}`)}`
    case 'patch':        return o.ops ? ` <span class="muted">(${o.ops.length} op${o.ops.length === 1 ? '' : 's'})</span>` : ''
    }
    return ''
}

function tkResultBadge(result, dryRun) {
    switch (result) {
    case 'ok':         return badgeTag('Done', 'ok')
    case 'sent':       return badgeTag('Sent', 'ok')
    case 'queued':     return badgeTag(dryRun ? 'Would notify' : 'Queued', dryRun ? 'warn' : 'ok')
    case 'would_run':  return badgeTag('Would run', 'warn')
    case 'would_send': return badgeTag('Would send', 'warn')
    case 'skipped':    return badgeTag('Skipped', '')
    case 'error':      return badgeTag('Error', 'bad')
    default:           return badgeTag(result || '?', '')
    }
}

function tkActionBody(p, dryRun) {
    let html = `<div class="row gap2 wrap">${tkResultBadge(p.result, dryRun)}${p.reason ? `<span class="event-detail">${esc(p.reason)}</span>` : ''}</div>`
    const o = p.output || {}
    if (p.kind === 'notify' && o.target) {
        html += `<div class="event-detail">Target: ${esc(tkTargetLabel(o.target))}${o.recipient_id ? ` (recipient ${o.recipient_id})` : ''}</div>`
    }
    if (p.kind === 'add_note' && o.text) {
        const flags = ['internal', 'discussion', 'resolution'].filter(f => o[f]).join(', ')
        html += `<div class="note">${flags ? `<div class="note-author">${esc(flags)}</div>` : ''}<div class="note-text">${esc(o.text)}</div>${o.note_id ? `<div class="event-detail">note #${o.note_id}</div>` : ''}</div>`
    }
    if (['set_status', 'set_priority', 'set_owner', 'add_resource'].includes(p.kind)) {
        html += `<div class="event-detail">${tkActionSummary(p.kind, o).replace(/^ → /, '')}${p.kind === 'add_resource' && o.resources ? ` <span class="muted">(resources: ${esc(o.resources)})</span>` : ''}</div>`
    }
    if (p.kind === 'patch' && o.ops) {
        html += `<pre class="code">${esc(JSON.stringify(o.ops, null, 2))}</pre>`
    }
    if (p.error) html += `<div class="event-error">${esc(p.error)}</div>`
    return html
}

function tkNotificationBody(p, dryRun) {
    let html = `<div class="row gap2 wrap">${tkResultBadge(p.result, dryRun)}<span class="event-detail">${esc(p.recipient_type || '')}</span></div>`
    if (p.forwarded_from?.length) html += `<div class="event-detail">Forwarded from ${esc(p.forwarded_from.join(' → '))}</div>`
    if (p.error) html += `<div class="event-error">${esc(p.error)}</div>`
    return html
}

function tkTargetLabel(t) {
    return { room: 'Webex room', person: 'Webex person', resources_owner: 'Ticket resources & owner' }[t] || t
}

function tkLoopReason(r) {
    return {
        note_author:    'The latest note was written by ticketbot',
        updated_by:     'The ticket was last updated by ticketbot',
        webhook_member: 'The webhook was triggered by ticketbot',
    }[r] || (r || 'Self-authored update')
}

tabLoaders.tickets = loadTickets
