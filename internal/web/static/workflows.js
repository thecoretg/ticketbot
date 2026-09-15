// ─────────────────────────────────────────────────────────
// Workflows
// ─────────────────────────────────────────────────────────
let wf           = null   // working copy of the workflow being edited
let wfOriginal   = ''     // JSON.stringify(wf) at load/save time, for dirty compare
let wfRecipients = []     // /webex/rooms cache (rooms + people)

const WF_TRIGGERS = [['create', 'New tickets'], ['update', 'Updated tickets'], ['both', 'New and updated']]
const WF_KINDS    = [['notify', 'Notify'], ['add_note', 'Add note'], ['skip_notify', 'Skip notify']]
const WF_TARGETS  = [['room', 'Webex room'], ['person', 'Webex person'], ['resources_owner', 'Ticket resources & owner']]

async function loadWorkflows(sub) {
    if (sub && /^\d+$/.test(sub)) {
        await loadWorkflowEditor(parseInt(sub))
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
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderWorkflowList(list) {
    const banner = appConfig?.master_dry_run
        ? '<div class="info-banner">Master dry run is on: every workflow runs as a dry run (Config → Master Dry Run).</div>' : ''

    const thead = `<th>Board</th><th>Enabled</th><th>Mode</th><th>Rules</th><th></th>`
    const rows  = list.map(w => `<tr class="clickable" onclick="openWorkflow(${w.id})">
        <td><strong>${esc(w.board_name || w.name)}</strong>${w.name && w.name !== w.board_name ? `<div class="muted">${esc(w.name)}</div>` : ''}</td>
        <td>${badge(w.enabled)}</td>
        <td>${w.dry_run ? badgeTag('Dry run', 'warn') : badgeTag('Live', 'on')}</td>
        <td>${(w.rules || []).length}</td>
        <td class="actions" onclick="event.stopPropagation()">
            <button class="btn btn-ghost btn-sm" onclick="openWorkflow(${w.id})">Open</button>
            <button class="btn btn-danger" onclick="deleteWorkflow(${w.id}, '${esc(w.board_name || w.name)}')">Delete</button>
        </td>
    </tr>`)

    setContent(`<div class="tab-header">
        <h2>Workflows</h2>
        <button class="btn btn-primary btn-sm" onclick="showNewWorkflowModal()">+ New Workflow</button>
    </div>
    ${banner}
    ${rows.length ? tableWrap(thead, rows) : '<div class="empty-state">No workflows yet. Create one for a board to start notifying.</div>'}`)
}

function openWorkflow(id) {
    switchTab('workflows', String(id))
}

async function showNewWorkflowModal() {
    let boards, existing
    try {
        ;[boards, existing] = await Promise.all([api('GET', '/cw/boards'), api('GET', '/workflows')])
    } catch (e) { toast(e.message, 'error'); return }

    const taken = new Set((existing || []).map(w => w.board_id))
    const free  = (boards || []).filter(b => !b.deleted && !taken.has(b.id))
    if (!free.length) { toast('Every board already has a workflow', 'error'); return }

    openModal('New Workflow', `
        <div class="form-group">
            <label>Board</label>
            <select id="wf-new-board">${free.map(b => `<option value="${b.id}">${esc(b.name)}</option>`).join('')}</select>
        </div>
        <p class="config-desc">A workflow starts with no rules. Add rules in the editor and save.</p>`,
    async () => {
        const boardID = parseInt(document.getElementById('wf-new-board').value)
        try {
            const created = await api('POST', '/workflows', { board_id: boardID })
            closeModal()
            openWorkflow(created.id)
        } catch (e) { toast(e.message, 'error') }
    })
}

async function deleteWorkflow(id, name) {
    if (!confirm(`Delete the workflow for ${name}? Tickets on this board will no longer be processed.`)) return
    try {
        await api('DELETE', `/workflows/${id}`)
        toast('Workflow deleted', 'success')
        loadWorkflowList()
    } catch (e) { toast(e.message, 'error') }
}

// ── Editor ───────────────────────────────────────────────
async function loadWorkflowEditor(id) {
    try {
        const [w, recips] = await Promise.all([api('GET', `/workflows/${id}`), api('GET', '/webex/rooms')])
        wfRecipients = recips || []
        wf = wfNormalize(w)
        wfOriginal = JSON.stringify(wf)
        tabGuard = wfGuard
        renderWorkflowEditor()
    } catch (e) {
        setContent(`<div class="tab-header"><div class="back-row">
                <button class="btn btn-ghost btn-sm" onclick="wfBack()">← Workflows</button><h2>Workflow</h2>
            </div></div>
            <div class="empty-state">${esc(e.message)}</div>`)
    }
}

function wfNormalize(w) {
    w.rules = w.rules || []
    for (const r of w.rules) {
        r.actions = r.actions || []
        r.condition = r.condition || ''
    }
    return w
}

function wfIsDirty() {
    return wf !== null && JSON.stringify(wf) !== wfOriginal
}

function wfGuard() {
    return !wfIsDirty() || confirm('You have unsaved workflow changes. Discard them?')
}
wfGuard.isDirty = wfIsDirty

function wfBack() {
    switchTab('workflows')
}

function renderWorkflowEditor() {
    const rules = wf.rules.map((r, i) => wfRuleCardHTML(r, i)).join('')

    setContent(`<div class="tab-header">
        <div class="back-row">
            <button class="btn btn-ghost btn-sm" onclick="wfBack()">← Workflows</button>
            <h2>${esc(wf.board_name || wf.name)}</h2>
            <span id="wf-dirty" class="dirty-dot${wfIsDirty() ? '' : ' hidden'}" title="Unsaved changes"></span>
        </div>
        <button id="wf-save" class="btn btn-primary btn-sm" onclick="saveWorkflow()" ${wfIsDirty() ? '' : 'disabled'}>Save</button>
    </div>
    ${appConfig?.master_dry_run ? '<div class="info-banner">Master dry run is on: this workflow will not write to ConnectWise or send Webex messages regardless of its own dry-run setting.</div>' : ''}
    <div class="config-form wf-head">
        <div class="config-row">
            <div>
                <div class="config-label">Workflow name</div>
                <div class="config-desc">Shown in the ticket history</div>
            </div>
            <input class="config-input config-input--wide" type="text" value="${esc(wf.name)}" oninput="wfSet('name', this.value)">
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Enabled</div>
                <div class="config-desc">Process tickets on this board</div>
            </div>
            <label class="toggle"><input type="checkbox" ${wf.enabled ? 'checked' : ''} onchange="wfSet('enabled', this.checked)"><span class="toggle-track"></span></label>
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Dry run</div>
                <div class="config-desc">Record what would happen; no ConnectWise writes, no Webex messages</div>
            </div>
            <label class="toggle"><input type="checkbox" ${wf.dry_run ? 'checked' : ''} onchange="wfSet('dry_run', this.checked)"><span class="toggle-track"></span></label>
        </div>
    </div>
    <div class="section-head"><h3>Rules</h3><span class="config-desc">Evaluated top to bottom for every new or updated ticket</span></div>
    <div class="rule-list" id="rule-list">${rules || '<div class="empty-state">No rules yet</div>'}</div>
    <button class="btn btn-ghost btn-sm" onclick="wfAddRule()">+ Add rule</button>`)
}

function wfRuleCardHTML(r, i) {
    const last = wf.rules.length - 1
    const opts = (list, sel) => list.map(([v, l]) => `<option value="${v}"${v === sel ? ' selected' : ''}>${l}</option>`).join('')

    return `<div class="rule-card${r.enabled ? '' : ' is-disabled'}" id="rule-${i}">
        <div class="rule-card-head">
            <div class="order-btns">
                <button class="btn-icon" title="Move up" onclick="wfMoveRule(${i}, -1)" ${i === 0 ? 'disabled' : ''}>▲</button>
                <button class="btn-icon" title="Move down" onclick="wfMoveRule(${i}, 1)" ${i === last ? 'disabled' : ''}>▼</button>
            </div>
            <span class="rule-index">${i + 1}</span>
            <input type="text" class="rule-name" value="${esc(r.name)}" placeholder="Rule name" oninput="wfSetRule(${i}, 'name', this.value)">
            <label class="toggle" title="Rule enabled"><input type="checkbox" ${r.enabled ? 'checked' : ''} onchange="wfSetRule(${i}, 'enabled', this.checked); document.getElementById('rule-${i}').classList.toggle('is-disabled', !this.checked)"><span class="toggle-track"></span></label>
            <button class="btn btn-danger" onclick="wfDeleteRule(${i})">Delete</button>
        </div>
        <div class="rule-grid">
            <div class="form-group">
                <label>Run for</label>
                <select onchange="wfSetRule(${i}, 'trigger', this.value)">${opts(WF_TRIGGERS, r.trigger)}</select>
            </div>
            <div class="form-group">
                <label>After this rule matches</label>
                <label class="check-inline"><input type="checkbox" ${r.stop_processing ? 'checked' : ''} onchange="wfSetRule(${i}, 'stop_processing', this.checked)"> Stop processing further rules</label>
            </div>
        </div>
        <div class="form-group">
            <label>Condition <span class="muted">— ConnectWise syntax, e.g. <code>status/name = 'New' and summary contains 'vpn'</code>. Empty always matches.</span></label>
            <textarea class="cond-textarea" id="cond-${i}" rows="2" spellcheck="false" oninput="wfSetRule(${i}, 'condition', this.value)">${esc(r.condition)}</textarea>
            <div class="cond-tools">
                <button class="btn btn-ghost btn-sm" onclick="wfValidate(${i})">Validate</button>
                <span id="cond-result-${i}" class="cond-result"></span>
                <span class="cond-spacer"></span>
                <input type="number" id="test-ticket-${i}" class="config-input cond-ticket" placeholder="Ticket #" min="1">
                <button class="btn btn-ghost btn-sm" onclick="wfTest(${i})">Test</button>
                <span id="test-result-${i}" class="cond-result"></span>
            </div>
        </div>
        <div class="form-group">
            <label>Actions</label>
            <div class="action-list" id="actions-${i}">${r.actions.map((a, j) => wfActionRowHTML(a, i, j)).join('') || '<div class="muted action-empty">No actions — this rule only affects the chain if "stop processing" is set.</div>'}</div>
            <div><button class="btn btn-ghost btn-sm" onclick="wfAddAction(${i})">+ Add action</button></div>
        </div>
    </div>`
}

function wfActionRowHTML(a, i, j) {
    const last = wf.rules[i].actions.length - 1
    const opts = (list, sel) => list.map(([v, l]) => `<option value="${v}"${v === sel ? ' selected' : ''}>${l}</option>`).join('')
    let fields = ''

    switch (a.kind) {
    case 'notify': {
        const n = a.notify || {}
        fields = `<select onchange="wfChangeTargetKind(${i}, ${j}, this.value)">${opts(WF_TARGETS, n.target)}</select>`
        if (n.target === 'room' || n.target === 'person') {
            fields += `<select class="fill" onchange="wfSetAction(${i}, ${j}, 'notify.recipient_id', this.value ? parseInt(this.value) : null)">
                <option value="">— choose a ${n.target} —</option>${wfRecipientOptions(n.target, n.recipient_id)}</select>`
        } else {
            fields += `<span class="muted fill">Everyone assigned to the ticket, plus the owner. Skips whoever wrote the triggering note; forwards apply.</span>`
        }
        break
    }
    case 'add_note': {
        const n = a.add_note || {}
        const flag = (key, label) => `<label class="check-inline"><input type="checkbox" ${n[key] ? 'checked' : ''} onchange="wfSetAction(${i}, ${j}, 'add_note.${key}', this.checked)"> ${label}</label>`
        fields = `<div class="fill action-note">
            <textarea rows="2" placeholder="Note text" oninput="wfSetAction(${i}, ${j}, 'add_note.text', this.value)">${esc(n.text || '')}</textarea>
            <div class="action-flags">${flag('discussion', 'Discussion')}${flag('internal', 'Internal')}${flag('resolution', 'Resolution')}</div>
        </div>`
        break
    }
    case 'skip_notify':
        fields = `<span class="muted fill">Suppresses every notify action later in this run.</span>`
        break
    }

    return `<div class="action-row${a.enabled ? '' : ' is-disabled'}" id="act-${i}-${j}">
        <div class="order-btns">
            <button class="btn-icon" title="Move up" onclick="wfMoveAction(${i}, ${j}, -1)" ${j === 0 ? 'disabled' : ''}>▲</button>
            <button class="btn-icon" title="Move down" onclick="wfMoveAction(${i}, ${j}, 1)" ${j === last ? 'disabled' : ''}>▼</button>
        </div>
        <label class="toggle toggle-sm" title="Action enabled"><input type="checkbox" ${a.enabled ? 'checked' : ''} onchange="wfSetAction(${i}, ${j}, 'enabled', this.checked); document.getElementById('act-${i}-${j}').classList.toggle('is-disabled', !this.checked)"><span class="toggle-track"></span></label>
        <select class="action-kind" onchange="wfChangeActionKind(${i}, ${j}, this.value)">${opts(WF_KINDS, a.kind)}</select>
        ${fields}
        <button class="btn-icon" title="Remove action" onclick="wfDeleteAction(${i}, ${j})">✕</button>
    </div>`
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

function wfSetRule(i, field, value) {
    wf.rules[i][field] = value
    wfMarkDirty()
}

// path is "enabled", "notify.recipient_id", "add_note.text", ...
function wfSetAction(i, j, path, value) {
    const a = wf.rules[i].actions[j]
    const [group, key] = path.split('.')
    if (key) {
        a[group] = a[group] || {}
        a[group][key] = value
    } else {
        a[group] = value
    }
    wfMarkDirty()
}

function wfMarkDirty() {
    const dirty = wfIsDirty()
    document.getElementById('wf-dirty')?.classList.toggle('hidden', !dirty)
    const save = document.getElementById('wf-save')
    if (save) save.disabled = !dirty
}

// ── Structural changes (re-render) ───────────────────────
function wfNewRule() {
    return { id: '', name: `Rule ${wf.rules.length + 1}`, enabled: true, trigger: 'both', condition: '', stop_processing: false, actions: [] }
}

function wfNewAction(kind) {
    const a = { kind, enabled: true }
    if (kind === 'notify')   a.notify   = { target: 'room', recipient_id: null }
    if (kind === 'add_note') a.add_note = { text: '', internal: true, discussion: false, resolution: false }
    return a
}

function wfAddRule() {
    wf.rules.push(wfNewRule())
    renderWorkflowEditor()
    document.getElementById(`rule-${wf.rules.length - 1}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    wfMarkDirty()
}

function wfDeleteRule(i) {
    const r = wf.rules[i]
    if ((r.actions.length || r.condition) && !confirm(`Delete rule "${r.name}"?`)) return
    wf.rules.splice(i, 1)
    renderWorkflowEditor()
    wfMarkDirty()
}

function wfMoveRule(i, dir) {
    const j = i + dir
    if (j < 0 || j >= wf.rules.length) return
    ;[wf.rules[i], wf.rules[j]] = [wf.rules[j], wf.rules[i]]
    renderWorkflowEditor()
    wfMarkDirty()
}

function wfAddAction(i) {
    wf.rules[i].actions.push(wfNewAction('notify'))
    wfRerenderRule(i)
    wfMarkDirty()
}

function wfDeleteAction(i, j) {
    wf.rules[i].actions.splice(j, 1)
    wfRerenderRule(i)
    wfMarkDirty()
}

function wfMoveAction(i, j, dir) {
    const acts = wf.rules[i].actions
    const k = j + dir
    if (k < 0 || k >= acts.length) return
    ;[acts[j], acts[k]] = [acts[k], acts[j]]
    wfRerenderRule(i)
    wfMarkDirty()
}

function wfChangeActionKind(i, j, kind) {
    const prev = wf.rules[i].actions[j]
    const next = wfNewAction(kind)
    next.enabled = prev.enabled
    wf.rules[i].actions[j] = next
    wfRerenderAction(i, j)
    wfMarkDirty()
}

function wfChangeTargetKind(i, j, target) {
    const a = wf.rules[i].actions[j]
    a.notify = { target }
    if (target !== 'resources_owner') a.notify.recipient_id = null
    wfRerenderAction(i, j)
    wfMarkDirty()
}

function wfRerenderRule(i) {
    const el = document.getElementById(`rule-${i}`)
    if (!el) { renderWorkflowEditor(); return }
    el.outerHTML = wfRuleCardHTML(wf.rules[i], i)
}

function wfRerenderAction(i, j) {
    const el = document.getElementById(`act-${i}-${j}`)
    if (!el) { wfRerenderRule(i); return }
    el.outerHTML = wfActionRowHTML(wf.rules[i].actions[j], i, j)
}

// ── Condition tools ──────────────────────────────────────
async function wfValidate(i) {
    const out = document.getElementById(`cond-result-${i}`)
    const ta  = document.getElementById(`cond-${i}`)
    out.className = 'cond-result'
    out.textContent = 'Checking…'
    try {
        const res = await api('POST', '/workflows/validate-condition', { condition: wf.rules[i].condition })
        if (res.valid) {
            out.className = 'cond-result ok'
            out.textContent = wf.rules[i].condition.trim() ? 'Valid' : 'Valid (matches everything)'
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

async function wfTest(i) {
    const out = document.getElementById(`test-result-${i}`)
    const id  = parseInt(document.getElementById(`test-ticket-${i}`).value)
    out.className = 'cond-result'
    if (!id) { out.className = 'cond-result err'; out.textContent = 'Enter a ticket number'; return }
    out.textContent = 'Testing…'
    try {
        const res = await api('POST', '/workflows/evaluate-condition', { condition: wf.rules[i].condition, ticket_id: id })
        out.className = `cond-result ${res.matches ? 'ok' : 'err'}`
        out.textContent = `${res.matches ? 'Matches' : 'No match'} against #${id} (${res.source})`
    } catch (e) {
        out.className = 'cond-result err'
        out.textContent = e.message
    }
}

// ── Save ─────────────────────────────────────────────────
function wfClientValidate() {
    for (const [i, r] of wf.rules.entries()) {
        if (!r.name.trim()) return { i, msg: 'Rule name is required' }
        for (const [j, a] of r.actions.entries()) {
            if (a.kind === 'notify' && a.notify.target !== 'resources_owner' && !a.notify.recipient_id)
                return { i, j, msg: `Action ${j + 1}: choose a ${a.notify.target}` }
            if (a.kind === 'add_note') {
                if (!a.add_note.text.trim()) return { i, j, msg: `Action ${j + 1}: note text is required` }
                if (!a.add_note.internal && !a.add_note.discussion && !a.add_note.resolution)
                    return { i, j, msg: `Action ${j + 1}: pick at least one note type` }
            }
        }
    }
    return null
}

async function saveWorkflow() {
    const bad = wfClientValidate()
    if (bad) {
        toast(`Rule ${bad.i + 1}: ${bad.msg}`, 'error')
        document.getElementById(`rule-${bad.i}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
        return
    }

    // strip nulls the server rejects for resources_owner
    for (const r of wf.rules) for (const a of r.actions) {
        if (a.kind === 'notify' && a.notify.recipient_id === null) delete a.notify.recipient_id
    }

    const btn = document.getElementById('wf-save')
    if (btn) { btn.disabled = true; btn.textContent = 'Saving…' }
    try {
        const saved = await api('PUT', `/workflows/${wf.id}`, wf)
        wf = wfNormalize(saved)
        wfOriginal = JSON.stringify(wf)
        renderWorkflowEditor()
        toast('Workflow saved', 'success')
    } catch (e) {
        const d = e.data?.details?.[0]
        if (d) {
            const where = `Rule ${d.rule_index + 1}${d.action_index != null ? `, action ${d.action_index + 1}` : ''}`
            toast(`${where}: ${d.field}: ${d.message}`, 'error')
            document.getElementById(`rule-${d.rule_index}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
            if (d.field === 'condition' && typeof d.pos === 'number') {
                const ta = document.getElementById(`cond-${d.rule_index}`)
                if (ta) { ta.focus(); ta.setSelectionRange(d.pos, d.pos) }
            }
        } else {
            toast(e.message, 'error')
        }
        if (btn) { btn.disabled = false; btn.textContent = 'Save' }
    }
}

tabLoaders.workflows = loadWorkflows
