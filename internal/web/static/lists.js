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
        setContent(errorState(e.message))
    }
}

function renderListIndex(lists) {
    lsCache = lists

    const columns = [
        { key: 'name', label: 'Name', cls: 'cell-primary', sort: l => l.name, cell: l => esc(l.name) },
        { key: 'type', label: 'Type', sort: l => lsTypeInfo(l.item_type).plural,
          cell: l => badgeTag(lsTypeInfo(l.item_type).plural, 'outline') },
        { key: 'items', label: 'Items', align: 'r', cls: 'num', sort: l => l.item_count, cell: l => l.item_count },
        { key: 'description', label: 'Description', cls: 'cell-ellipsis muted', sort: l => l.description,
          cell: l => esc(l.description || '') },
    ]
    const menu = l => [
        { label: 'Edit', icon: 'edit', edit: true, run: () => editList(l.id) },
        { label: 'Delete', icon: 'trash', danger: true, edit: true, run: () => deleteList(l.id) },
    ]

    setContent(pageActions(
        editOnly(`<button class="btn btn-primary" onclick="showListModal()">${icon('plus')}New list</button>`)) +
    dataTable({
        id: 'lists', columns, rows: lists, menu, tr: l => `class="clickable" onclick="openList(${l.id})"`,
        empty: emptyState('No lists yet',
            'Create a list of contacts or companies, then reference it from a rule condition.',
            editOnly(`<button class="btn btn-primary btn-sm" onclick="showListModal()">${icon('plus')}New list</button>`), 'blocks'),
        foot: `<span>${lists.length} list${lists.length === 1 ? '' : 's'}</span>`,
    }))
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

    openModal(existing ? 'Edit list' : 'New list', `
        <div class="stack gap4">
            <div class="field">
                <label for="ls-name">Name</label>
                <input class="input" type="text" id="ls-name" value="${esc(existing?.name ?? '')}" placeholder="Drop notifications">
            </div>
            <div class="field">
                <label for="ls-type">Type${existing ? ' <span class="muted">(cannot be changed)</span>' : ''}</label>
                <select class="select" id="ls-type"${existing ? ' disabled' : ''}>${typeOpts}</select>
            </div>
            <div class="field">
                <label for="ls-desc">Description <span class="muted">(optional)</span></label>
                <input class="input" type="text" id="ls-desc" value="${esc(existing?.description ?? '')}" placeholder="What this list is for">
            </div>
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

function deleteList(id, fromDetail = false) {
    const l = lsCache.find(x => x.id === id) || (lsDetail?.id === id ? lsDetail : null)
    const count = l?.item_count ?? l?.items?.length
    confirmModal({
        title: 'Delete this list?',
        body: `<b>${esc(l?.name || `List ${id}`)}</b>${
            count ? `Its ${count} member${count === 1 ? '' : 's'} go with it. ` : ''}Any rule condition that tests this list stops matching.`,
        confirmLabel: 'Delete list',
        onConfirm: async () => {
            try {
                await api('DELETE', `/lists/${id}`)
                toast('List deleted', 'success')
                if (fromDetail) lsBack()
                else loadListIndex()
            } catch (e) {
                if (e.status === 409 && e.data?.references?.length) {
                    const used = e.data.references.map(r => `${r.workflow_name} / ${r.node_title}`).join(', ')
                    toast(`This list is used by: ${used}. Remove those conditions first.`, 'error')
                    return
                }
                toast(e.message, 'error')
            }
        },
    })
}

// ── Detail ───────────────────────────────────────────────
async function loadListDetail(id) {
    let d
    try {
        ;[d] = await Promise.all([api('GET', `/lists/${id}`), lsLoadTypes()])
    } catch (e) {
        setContent(backRow('lists', 'Lists') + errorState(e.message))
        return
    }
    lsDetail = d
    renderListDetail(d)
    setCrumbHere(d.name)
}

function renderListDetail(d) {
    const info   = lsTypeInfo(d.item_type)
    const count  = d.items?.length || 0
    const noun   = count === 1 ? info.label.toLowerCase() : info.plural.toLowerCase()
    const usedBy = d.used_by?.length
        ? `<div class="banner">${icon('info')}<div>Used by ${d.used_by.map(r =>
            `<button class="btn btn-ghost btn-sm" onclick="openWorkflow(${r.workflow_id})" data-tip="${esc(r.board_name || '')}">${esc(r.workflow_name)} / ${esc(r.node_title)}</button>`).join(' ')}</div></div>`
        : ''

    const detail = !!info.detail_label  // e.g. a contact's company
    const columns = [
        { key: 'label', label: info.label, cls: 'cell-primary', sort: it => it.label,
          cell: it => `${esc(it.label)}${it.missing ? ' ' + badgeTag('Not synced', 'warn') : ''}` },
        ...(detail ? [{ key: 'detail', label: info.detail_label, cls: 'muted', sort: it => it.detail,
          cell: it => esc(it.detail || '—') }] : []),
        { key: 'id', label: 'ID', align: 'r', cls: 'num muted', sort: it => it.item_id, cell: it => `#${it.item_id}` },
        { key: 'added', label: 'Added', cls: 'muted nowrap', firstDir: 'desc', sort: it => tblTime(it.added_on),
          cell: it => fmtDateTime(it.added_on) },
    ]
    const menu = it => [{ label: 'Remove', icon: 'trash', danger: true, edit: true, run: () => lsRemoveItem(it.item_id) }]

    const picker = `<span class="typeahead" style="max-width:360px;flex:1">
        <input class="input" type="text" id="ls-pick" autocomplete="off"
            aria-label="Search ${esc(info.plural.toLowerCase())} to add"
            placeholder="+ search ${esc(info.plural.toLowerCase())} to add…"
            oninput="lsPickerInput('${esc(info.source)}', this)">
    </span>`

    setContent(`${backRow('lists', 'Lists', d.name)}
    <header class="page-head row spread wrap gap4">
        <div>
            <h1 class="page-title">${esc(d.name)}</h1>
            <p class="page-sub row gap2 wrap">
                ${badgeTag(info.plural, 'outline')}
                <span>${esc(d.description || `A set of ${info.plural.toLowerCase()} rule conditions can test against.`)}</span>
            </p>
        </div>
        <div class="row gap2 wrap">
            <button class="btn btn-default" onclick="editList(${d.id})">${icon('edit')}Edit</button>
            ${deleteButton(`deleteList(${d.id}, true)`)}
        </div>
    </header>
    ${usedBy}
    ${dataTable({
        // per list type: a contact list has a Company column a company list does not
        id: `list-items-${d.item_type}`, columns, menu, rows: d.items || [],
        toolbar: `<span class="cell-sub"><span class="num">${count}</span> ${esc(noun)}</span><div class="grow"></div>${picker}`,
        empty: emptyState(`No ${esc(info.plural.toLowerCase())} in this list`,
            'Search above to add the first one. An empty list never matches a condition.',
            '', 'blocks'),
        foot: `<span><span class="num">${count}</span> ${esc(noun)}</span>`,
    })}`)
}

// The picker uses the shared typeahead popup (app.js); a pick adds the member straight away.
function lsPickerInput(source, input) {
    clearTimeout(lsPickerTimer)
    typeaheadHide()
    const q = input.value.trim()
    if (q.length < 2) return
    lsPickerTimer = setTimeout(async () => {
        try {
            const found = (await api('GET', `/cw/${source}?${new URLSearchParams({ q, limit: 15 })}`)) || []
            const have  = new Set((lsDetail?.items || []).map(it => it.item_id))
            typeaheadShow(input,
                found.filter(it => !have.has(it.id)).map(it => ({ label: wfLookupLabel(source, it), sub: `#${it.id}`, id: it.id })),
                it => { input.value = ''; lsAddItem(it.id) })
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
