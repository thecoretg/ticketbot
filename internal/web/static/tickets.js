// ─────────────────────────────────────────────────────────
// Tickets
// ─────────────────────────────────────────────────────────
let tkFilters     = { board_id: '', status_id: '', closed: '', q: '', page: 1, page_size: 25 }
let tkBoards      = []
let tkSearchTimer = null
let tkRequestSeq  = 0      // drops stale responses when filters change quickly

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

    setContent(`<div class="tab-header">
        <h2>Tickets</h2>
        <div class="filter-bar">
            <select id="tk-board" class="filter-select" onchange="tkBoardChanged(this.value)">
                <option value="">All boards</option>${boardOpts}
            </select>
            <select id="tk-status" class="filter-select" onchange="tkSetFilter('status_id', this.value)" ${tkFilters.board_id ? '' : 'disabled'}>
                <option value="">All statuses</option>
            </select>
            <select id="tk-closed" class="filter-select" onchange="tkSetFilter('closed', this.value)">
                <option value=""${tkFilters.closed === '' ? ' selected' : ''}>Open + closed</option>
                <option value="false"${tkFilters.closed === 'false' ? ' selected' : ''}>Open only</option>
                <option value="true"${tkFilters.closed === 'true' ? ' selected' : ''}>Closed only</option>
            </select>
            <input id="tk-search" type="text" class="logs-search-input" placeholder="Search summary or #…" value="${esc(tkFilters.q)}" oninput="tkSetSearch(this.value)">
            <button class="btn btn-ghost btn-sm" onclick="refreshTicketTable()">Refresh</button>
        </div>
    </div>
    <div id="tk-table"><div class="loading-state">Loading…</div></div>
    <div id="tk-pager" class="pager"></div>`)

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
        table.innerHTML = `<div class="empty-state">${esc(e.message)}</div>`
        pager.innerHTML = ''
        return
    }
    if (seq !== tkRequestSeq || !document.getElementById('tk-table')) return

    const items = page?.items || []
    const thead = `<th>ID</th><th>Summary</th><th>Board</th><th>Status</th><th>Company</th><th>Owner</th><th>Updated</th>`
    const rows  = items.map(t => `<tr class="clickable" onclick="openTicket(${t.id})">
        <td><a class="tk-id" href="${esc(t.cw_url)}" target="_blank" rel="noopener" onclick="event.stopPropagation()" title="Open in ConnectWise">#${t.id}</a></td>
        <td class="cell-ellipsis" title="${esc(t.summary)}">${esc(t.summary)}${t.deleted ? ' ' + badgeTag('Deleted', 'off') : ''}</td>
        <td>${esc(t.board_name)}</td>
        <td>${esc(t.status_name)}${t.closed_flag ? ' ' + badgeTag('Closed', 'muted') : ''}</td>
        <td>${esc(t.company_name)}</td>
        <td>${esc(t.owner_name || '—')}</td>
        <td class="nowrap">${fmtDateTime(t.updated_on)}</td>
    </tr>`)

    table.innerHTML = tableWrap(thead, rows)

    const total    = page?.total || 0
    const size     = page?.page_size || tkFilters.page_size
    const current  = page?.page || 1
    const lastPage = Math.max(1, Math.ceil(total / size))
    const from     = total ? (current - 1) * size + 1 : 0
    const to       = Math.min(current * size, total)
    tkFilters.page = current

    pager.innerHTML = total ? `
        <span>Showing ${from}–${to} of ${total}</span>
        <div class="pager-btns">
            <button class="btn btn-ghost btn-sm" onclick="tkPage(-1)" ${current <= 1 ? 'disabled' : ''}>Prev</button>
            <span>Page ${current} of ${lastPage}</span>
            <button class="btn btn-ghost btn-sm" onclick="tkPage(1)" ${current >= lastPage ? 'disabled' : ''}>Next</button>
        </div>` : ''
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
        setContent(`<div class="tab-header"><div class="back-row">
                <button class="btn btn-ghost btn-sm" onclick="tkBack()">← Tickets</button><h2>Ticket #${id}</h2>
            </div></div>
            <div class="empty-state">${esc(e.message)}</div>`)
        return
    }
    renderTicketDetail(data)
}

function renderTicketDetail(d) {
    const t   = d.ticket
    const evs = d.events || []
    const contact = d.contact ? [d.contact.first_name, d.contact.last_name].filter(Boolean).join(' ') : '—'
    const resources = (d.resources || []).map(m => memberLabel(m)).join(', ') || (t.resources || '—')

    const meta = (label, value) => `<div><span class="meta-label">${label}</span><span>${value}</span></div>`

    setContent(`<div class="tab-header">
        <div class="back-row">
            <button class="btn btn-ghost btn-sm" onclick="tkBack()">← Tickets</button>
            <h2 class="tk-title">
                <a class="ext-link" href="${esc(t.cw_url)}" target="_blank" rel="noopener" title="Open in ConnectWise">#${t.id} ↗</a>
                <span class="tk-summary">${esc(t.summary)}</span>
                ${t.deleted ? badgeTag('Deleted', 'off') : ''}
            </h2>
        </div>
        <button class="btn btn-ghost btn-sm" onclick="loadTicketDetail(${t.id})">Refresh</button>
    </div>
    <div class="ticket-meta">
        ${meta('Board', esc(t.board_name))}
        ${meta('Status', esc(t.status_name) + (t.closed_flag ? ' ' + badgeTag('Closed', 'muted') : ''))}
        ${meta('Company', esc(t.company_name))}
        ${meta('Priority', esc(t.priority_name || '—'))}
        ${meta('Owner', esc(t.owner_name || '—'))}
        ${meta('Contact', esc(contact))}
        ${meta('Resources', esc(resources))}
        ${meta('Type', esc([t.type_name, t.subtype_name, t.item_name].filter(Boolean).join(' / ') || '—'))}
        ${meta('Updated', fmtDateTime(t.updated_on) + (t.updated_by ? ` <span class="muted">by ${esc(t.updated_by)}</span>` : ''))}
        ${meta('Added', fmtDateTime(t.added_on))}
    </div>
    <div class="section-head"><h3>History</h3><span class="config-desc">${evs.length} event${evs.length === 1 ? '' : 's'}, oldest first</span></div>
    <div class="timeline">${evs.map(tkEventHTML).join('') || '<div class="empty-state">No events recorded</div>'}</div>`)
}

function memberLabel(m) {
    const name = [m.first_name, m.last_name].filter(Boolean).join(' ')
    return name || m.identifier || ''
}

// ── Timeline ─────────────────────────────────────────────
function tkEventHTML(ev) {
    const p   = ev.payload || {}
    let title = ev.kind, body = '', cls = ev.kind

    switch (ev.kind) {
    case 'created':
        title = 'Ticket created'
        body  = tkChangeBody(p)
        break
    case 'updated':
        title = 'Ticket updated'
        body  = tkChangeBody(p)
        break
    case 'deleted':
        title = 'Ticket deleted in ConnectWise'
        break
    case 'loop_guard':
        title = 'Rules skipped: self-authored update'
        body  = `<div class="tl-detail">${esc(tkLoopReason(p.reason))}${p.identifier ? ` (<code>${esc(p.identifier)}</code>)` : ''}</div>`
        break
    case 'workflow':
        title = p.found ? `Workflow: ${esc(p.workflow_name || '')}` : 'No workflow for this board'
        if (p.found && !p.enabled) body = '<div class="tl-detail">Workflow is disabled</div>'
        else if (p.rules?.length) body = `<div class="rule-chips">${p.rules.map(tkRuleChip).join('')}</div>`
        break
    case 'action':
        title = `${esc(p.rule_name || 'Rule')} · ${tkActionLabel(p.kind)}`
        body  = tkActionBody(p, ev.dry_run)
        if (p.result === 'error') cls += ' tl-error'
        break
    case 'notification':
        title = `${esc(p.rule_name || 'Notify')} → ${esc(p.recipient_name || '?')}`
        body  = tkNotificationBody(p, ev.dry_run)
        if (p.result === 'error') cls += ' tl-error'
        break
    case 'error':
        title = `Error${p.stage ? ` during ${esc(p.stage)}` : ''}`
        body  = `<div class="tl-error-text">${esc(p.error || '')}</div>`
        cls  += ' tl-error'
        break
    }

    return `<div class="tl-item tl-${cls}">
        <div class="tl-dot"></div>
        <div class="tl-card">
            <div class="tl-head">
                <span class="tl-title">${title}</span>
                ${ev.dry_run ? badgeTag('DRY RUN', 'warn') : ''}
                <span class="tl-source">${esc(ev.source)}</span>
                <span class="tl-time">${fmtDateTime(ev.occurred_at)}</span>
            </div>
            ${body ? `<div class="tl-body">${body}</div>` : ''}
        </div>
    </div>`
}

function tkChangeBody(p) {
    let html = ''
    if (p.updated_by) html += `<div class="tl-detail">by ${esc(p.updated_by)}</div>`
    if (p.changes?.length) html += tkDiffTableHTML(p.changes)
    if (p.new_note) {
        const n = p.new_note
        html += `<div class="note-preview"><div class="note-author">${esc(n.author_name || n.author_identifier || 'Unknown')}${n.internal ? ' ' + badgeTag('Internal', 'muted') : ''}</div><div class="note-text">${esc(n.preview || '')}</div></div>`
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
    return `<table class="diff-table"><thead><tr><th>Field</th><th>Old</th><th>New</th></tr></thead><tbody>${
        changes.map(c => `<tr>
            <td class="diff-field">${esc(c.field)}</td>
            <td class="diff-old">${val(c.old)}</td>
            <td class="diff-new">${val(c.new)}</td>
        </tr>`).join('')
    }</tbody></table>`
}

function tkRuleChip(r) {
    let variant = 'muted', label = 'no match'
    if (r.error)        { variant = 'off';  label = 'error' }
    else if (r.skipped) { variant = 'muted'; label = r.skipped === 'trigger' ? 'trigger mismatch' : r.skipped }
    else if (r.matched) { variant = 'on';   label = r.stopped ? 'matched · stop' : 'matched' }
    return `<span class="rule-chip" title="${esc(r.error || '')}">${esc(r.rule_name)} ${badgeTag(label, variant)}</span>`
}

function tkActionLabel(kind) {
    return { notify: 'Notify', add_note: 'Add note', skip_notify: 'Skip notify' }[kind] || esc(kind || 'Action')
}

function tkResultBadge(result, dryRun) {
    switch (result) {
    case 'ok':         return badgeTag('Done', 'on')
    case 'sent':       return badgeTag('Sent', 'on')
    case 'queued':     return badgeTag(dryRun ? 'Would notify' : 'Queued', dryRun ? 'warn' : 'on')
    case 'would_run':  return badgeTag('Would run', 'warn')
    case 'would_send': return badgeTag('Would send', 'warn')
    case 'skipped':    return badgeTag('Skipped', 'muted')
    case 'error':      return badgeTag('Error', 'off')
    default:           return badgeTag(result || '?', 'muted')
    }
}

function tkActionBody(p, dryRun) {
    let html = `<div class="tl-row">${tkResultBadge(p.result, dryRun)}${p.reason ? `<span class="tl-detail">${esc(p.reason)}</span>` : ''}</div>`
    const o = p.output || {}
    if (p.kind === 'notify' && o.target) {
        html += `<div class="tl-detail">Target: ${esc(tkTargetLabel(o.target))}${o.recipient_id ? ` (recipient ${o.recipient_id})` : ''}</div>`
    }
    if (p.kind === 'add_note' && o.text) {
        const flags = ['internal', 'discussion', 'resolution'].filter(f => o[f]).join(', ')
        html += `<div class="note-preview">${flags ? `<div class="note-author">${esc(flags)}</div>` : ''}<div class="note-text">${esc(o.text)}</div>${o.note_id ? `<div class="tl-detail">note #${o.note_id}</div>` : ''}</div>`
    }
    if (p.error) html += `<div class="tl-error-text">${esc(p.error)}</div>`
    return html
}

function tkNotificationBody(p, dryRun) {
    let html = `<div class="tl-row">${tkResultBadge(p.result, dryRun)}<span class="tl-detail">${esc(p.recipient_type || '')}</span></div>`
    if (p.forwarded_from?.length) html += `<div class="tl-detail">Forwarded from ${esc(p.forwarded_from.join(' → '))}</div>`
    if (p.error) html += `<div class="tl-error-text">${esc(p.error)}</div>`
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
