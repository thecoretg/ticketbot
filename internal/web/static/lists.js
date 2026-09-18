// ─────────────────────────────────────────────────────────
// Lists
//
// Admin-defined sets of ConnectWise contacts or companies. Conditions reference a list by id
// (`contact/id in list 3`); the builder offers "is in list" on any field whose type has lists.
// #lists is the index, #lists/<id> the detail page where members are added and removed.
// ─────────────────────────────────────────────────────────
let lsTypes       = []     // /lists/types
let lsCache       = []     // last index load, so Edit needs no refetch
let lsDetail      = null   // list currently shown on the detail page
let lsPickerTimer = null

async function loadLists(sub) {
    if (sub && /^\d+$/.test(sub)) {
        await loadListDetail(parseInt(sub))
        return
    }
    await loadListIndex()
}

async function lsLoadTypes() {
    if (lsTypes.length) return
    try { lsTypes = (await api('GET', '/lists/types')) || [] } catch {}
}

function lsTypeInfo(type) {
    return lsTypes.find(t => t.type === type) || { type, label: type, plural: type, source: '' }
}

function lsBack() {
    switchTab('lists')
}

function openList(id) {
    switchTab('lists', String(id))
}

// ── Index ────────────────────────────────────────────────
async function loadListIndex() {
    try {
        const [lists] = await Promise.all([api('GET', '/lists'), lsLoadTypes()])
        renderListIndex(lists || [])
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderListIndex(lists) {
    lsCache = lists

    const header = `<div class="tab-header">
        <h2>Lists</h2>
        <button class="btn btn-primary btn-sm" onclick="showListModal()">+ New List</button>
    </div>`

    const thead = '<th>Name</th><th>Type</th><th>Items</th><th>Description</th><th></th>'
    const rows  = lists.map(l => `<tr class="clickable" onclick="openList(${l.id})">
        <td><span class="tk-id">${esc(l.name)}</span></td>
        <td>${badgeTag(lsTypeInfo(l.item_type).plural, 'muted')}</td>
        <td>${l.item_count}</td>
        <td class="cell-ellipsis muted">${esc(l.description || '')}</td>
        <td class="actions" onclick="event.stopPropagation()">
            <button class="btn btn-ghost btn-sm" onclick="editList(${l.id})">Edit</button>
            <button class="btn btn-danger" onclick="deleteList(${l.id})">Delete</button>
        </td>
    </tr>`)

    setContent(header + tableWrap(thead, rows))
}

function editList(id) {
    const l = lsCache.find(x => x.id === id) || (lsDetail && lsDetail.id === id ? lsDetail : null)
    if (!l) { toast('List not found — reload the tab', 'error'); return }
    showListModal(l)
}

async function showListModal(existing = null) {
    await lsLoadTypes()
    if (!lsTypes.length) { toast('List types unavailable', 'error'); return }

    const typeOpts = lsTypes.map(t =>
        `<option value="${esc(t.type)}"${t.type === (existing?.item_type ?? lsTypes[0].type) ? ' selected' : ''}>${esc(t.plural)}</option>`).join('')

    openModal(existing ? 'Edit List' : 'New List', `
        <div class="form-group">
            <label>Name</label>
            <input type="text" id="ls-name" value="${esc(existing?.name ?? '')}" placeholder="Drop Notifications">
        </div>
        <div class="form-group">
            <label>Type${existing ? ' <span class="muted">(cannot be changed)</span>' : ''}</label>
            <select id="ls-type"${existing ? ' disabled' : ''}>${typeOpts}</select>
        </div>
        <div class="form-group">
            <label>Description <span class="muted">(optional)</span></label>
            <input type="text" id="ls-desc" value="${esc(existing?.description ?? '')}" placeholder="What this list is for">
        </div>`, async () => {
        const name = document.getElementById('ls-name').value.trim()
        const description = document.getElementById('ls-desc').value.trim()
        if (!name) { toast('Name is required', 'error'); return }

        try {
            if (existing) {
                await api('PUT', `/lists/${existing.id}`, { name, description })
            } else {
                const item_type = document.getElementById('ls-type').value
                await api('POST', '/lists', { name, item_type, description })
            }
            closeModal()
            toast(existing ? 'List updated' : 'List created', 'success')
            if (lsDetail && existing && lsDetail.id === existing.id) loadListDetail(existing.id)
            else loadListIndex()
        } catch (e) { toast(e.message, 'error') }
    }, existing ? 'Save' : 'Create')
}

async function deleteList(id, fromDetail = false) {
    if (!confirm('Delete this list? Rules that reference it will stop matching.')) return
    try {
        await api('DELETE', `/lists/${id}`)
        toast('List deleted', 'success')
        if (fromDetail) lsBack()
        else loadListIndex()
    } catch (e) {
        if (e.status === 409 && e.data?.references?.length) {
            const used = e.data.references.map(r => `${r.workflow_name} / ${r.rule_name}`).join(', ')
            toast(`This list is used by: ${used}. Remove those conditions first.`, 'error')
            return
        }
        toast(e.message, 'error')
    }
}

// ── Detail ───────────────────────────────────────────────
async function loadListDetail(id) {
    let d
    try {
        ;[d] = await Promise.all([api('GET', `/lists/${id}`), lsLoadTypes()])
    } catch (e) {
        setContent(`<div class="tab-header"><div class="back-row">
                <button class="btn btn-ghost btn-sm" onclick="lsBack()">← Lists</button><h2>List</h2>
            </div></div>
            <div class="empty-state">${esc(e.message)}</div>`)
        return
    }
    lsDetail = d
    renderListDetail(d)
}

function renderListDetail(d) {
    const info = lsTypeInfo(d.item_type)
    const usedBy = d.used_by?.length
        ? `<div class="info-banner">Used by ${d.used_by.map(r =>
            `<button class="btn btn-ghost btn-sm" onclick="openWorkflow(${r.workflow_id})" title="${esc(r.board_name || '')}">${esc(r.workflow_name)} / ${esc(r.rule_name)}</button>`).join(' ')}</div>`
        : ''

    const detail = !!info.detail_label  // e.g. a contact's company
    const thead = `<th>${esc(info.label)}</th>${detail ? `<th>${esc(info.detail_label)}</th>` : ''}<th>ID</th><th>Added</th><th></th>`
    const rows = (d.items || []).map(it => `<tr>
        <td>${esc(it.label)}${it.missing ? ' ' + badgeTag('Not synced', 'warn') : ''}</td>
        ${detail ? `<td class="muted">${esc(it.detail || '—')}</td>` : ''}
        <td class="muted">#${it.item_id}</td>
        <td class="muted nowrap">${fmtDateTime(it.added_on)}</td>
        <td class="actions"><button class="btn btn-danger" onclick="lsRemoveItem(${it.item_id})">Remove</button></td>
    </tr>`)

    setContent(`<div class="tab-header">
        <div class="back-row">
            <button class="btn btn-ghost btn-sm" onclick="lsBack()">← Lists</button>
            <h2 class="tk-title">${esc(d.name)} ${badgeTag(info.plural, 'muted')}</h2>
        </div>
        <div class="filter-bar">
            <button class="btn btn-ghost btn-sm" onclick="editList(${d.id})">Edit</button>
            <button class="btn btn-danger" onclick="deleteList(${d.id}, true)">Delete</button>
        </div>
    </div>
    ${d.description ? `<p class="muted" style="margin-bottom:16px">${esc(d.description)}</p>` : ''}
    ${usedBy}
    <div class="section-head">
        <h3>${d.items?.length || 0} ${esc(d.items?.length === 1 ? info.label.toLowerCase() : info.plural.toLowerCase())}</h3>
        <span class="typeahead" style="max-width:360px">
            <input type="text" id="ls-pick" list="ls-pick-list" autocomplete="off" placeholder="+ search ${esc(info.plural.toLowerCase())} to add…"
                oninput="lsPickerInput('${esc(info.source)}', this)">
            <datalist id="ls-pick-list"></datalist>
        </span>
    </div>
    ${tableWrap(thead, rows)}`)
}

// The picker mirrors the condition builder's typeahead: options render as "Name (#id)" and a
// pick is detected by that suffix, so it commits on `input` without waiting for blur.
function lsPickerInput(source, input) {
    clearTimeout(lsPickerTimer)
    const m = input.value.match(/\(#(\d+)\)\s*$/)
    if (m) {
        input.value = ''
        lsAddItem(parseInt(m[1]))
        return
    }
    const q = input.value.trim()
    if (q.length < 2) return
    lsPickerTimer = setTimeout(async () => {
        try {
            const found = (await api('GET', `/cw/${source}?${new URLSearchParams({ q, limit: 15 })}`)) || []
            const dl = document.getElementById('ls-pick-list')
            if (!dl) return
            const have = new Set((lsDetail?.items || []).map(it => it.item_id))
            dl.innerHTML = found.filter(it => !have.has(it.id))
                .map(it => `<option value="${esc(`${wfLookupLabel(source, it)} (#${it.id})`)}"></option>`).join('')
        } catch (e) { toast(e.message, 'error') }
    }, 250)
}

async function lsAddItem(itemId) {
    if (!lsDetail) return
    try {
        const it = await api('POST', `/lists/${lsDetail.id}/items`, { item_id: itemId })
        toast(`Added ${it.label}`, 'success')
        loadListDetail(lsDetail.id)
    } catch (e) { toast(e.message, 'error') }
}

async function lsRemoveItem(itemId) {
    if (!lsDetail) return
    try {
        await api('DELETE', `/lists/${lsDetail.id}/items/${itemId}`)
        loadListDetail(lsDetail.id)
    } catch (e) { toast(e.message, 'error') }
}

tabLoaders.lists = loadLists
