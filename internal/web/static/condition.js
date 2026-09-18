// ─────────────────────────────────────────────────────────
// Condition builder
//
// A rule's condition is always stored as text (ConnectWise-style syntax evaluated by the server).
// The builder is a view over that text: rows compile to text on every change, and text is parsed
// back into rows (via /workflows/parse-condition) when a rule loads or the user switches modes.
// Conditions the builder cannot represent (nested groups, "not", unknown paths, complex wildcards)
// stay in advanced mode with a note saying why.
//
// Per-rule UI state lives on rule._ui = { mode: 'builder'|'advanced', join: 'and'|'or', rows, reason }
// and is stripped before the workflow is compared or sent to the server.
// ─────────────────────────────────────────────────────────
let wfFields = []                    // /workflows/fields
let wfBoards = []                    // /cw/boards
let wfLists  = []                    // /lists — admin lists for "is in list"
const wfNames = { companies: {}, contacts: {} }   // id → display name, for pickers that search
let wfTypeaheadTimer = null

const WF_OPS = {
    string:     [['eq', 'is'], ['ne', 'is not'], ['contains', 'contains'], ['starts', 'starts with'], ['in', 'is one of'], ['not_in', 'is not one of'], ['empty', 'is empty'], ['not_empty', 'is not empty']],
    number:     [['eq', 'is'], ['ne', 'is not'], ['gt', 'is greater than'], ['lt', 'is less than'], ['in', 'is one of'], ['not_in', 'is not one of'], ['empty', 'is empty'], ['not_empty', 'is not empty']],
    ref:        [['eq', 'is'], ['ne', 'is not'], ['in', 'is one of'], ['not_in', 'is not one of'], ['empty', 'is empty'], ['not_empty', 'is not empty']],
    identifier: [['eq', 'is'], ['ne', 'is not'], ['in', 'is one of'], ['not_in', 'is not one of'], ['empty', 'is empty'], ['not_empty', 'is not empty']],
    list:       [['contains', 'includes'], ['empty', 'is empty'], ['not_empty', 'is not empty']],
    bool:       [['true', 'is true'], ['false', 'is false']],
}
const WF_NO_VALUE_OPS = new Set(['empty', 'not_empty', 'true', 'false'])
const WF_MULTI_OPS    = new Set(['in', 'not_in'])
const WF_LIST_OPS     = new Set(['in_list', 'not_in_list'])

// wfOpsFor returns the operators for a field: its type's table, plus list membership when the
// field's Source has admin lists (list_type is set by the server).
function wfOpsFor(f) {
    const ops = WF_OPS[f.type] || []
    return f.list_type ? ops.concat([['in_list', 'is in list'], ['not_in_list', 'is not in list']]) : ops
}

function wfFieldByPath(path) {
    const p = (path || '').toLowerCase()
    return wfFields.find(f => f.path.toLowerCase() === p) || null
}

function wfDefaultUI() {
    return { mode: 'builder', join: 'and', rows: [], reason: '' }
}

function wfNewRow() {
    return { path: 'status/id', op: 'eq', value: null }
}

// ── Text ⇄ rows ──────────────────────────────────────────
function wfQuote(v) {
    return `'${String(v).replace(/'/g, "''")}'`
}

// wfCompileRow returns the condition fragment for one row, or { error } when the row is incomplete.
function wfCompileRow(row) {
    const f = wfFieldByPath(row.path)
    if (!f) return { error: `unknown field ${row.path}` }
    const numeric = f.type === 'ref' || f.type === 'number'
    const lit = v => numeric ? Number(v) : wfQuote(v)
    const need = () => (row.value === null || row.value === undefined || row.value === '') ? { error: `${f.label}: value required` } : null
    const badNum = v => numeric && (v === '' || Number.isNaN(Number(v)))

    switch (row.op) {
    case 'true':      return { text: `${f.path} = true` }
    case 'false':     return { text: `${f.path} = false` }
    case 'empty':     return { text: `${f.path} = null` }
    case 'not_empty': return { text: `${f.path} != null` }
    case 'eq': case 'ne': case 'gt': case 'lt': {
        const e = need(); if (e) return e
        if (badNum(row.value)) return { error: `${f.label}: must be a number` }
        const op = { eq: '=', ne: '!=', gt: '>', lt: '<' }[row.op]
        return { text: `${f.path} ${op} ${lit(row.value)}` }
    }
    case 'contains': {
        const e = need(); if (e) return e
        return { text: `${f.path} contains ${wfQuote(row.value)}` }
    }
    case 'starts': {
        const e = need(); if (e) return e
        return { text: `${f.path} like ${wfQuote(String(row.value) + '*')}` }
    }
    case 'in': case 'not_in': {
        const vals = Array.isArray(row.value) ? row.value.filter(v => v !== '' && v !== null && v !== undefined) : []
        if (!vals.length) return { error: `${f.label}: pick at least one value` }
        if (vals.some(badNum)) return { error: `${f.label}: values must be numbers` }
        return { text: `${f.path} ${row.op === 'in' ? 'in' : 'not in'} (${vals.map(lit).join(', ')})` }
    }
    case 'in_list': case 'not_in_list': {
        if (row.value === null || row.value === undefined || row.value === '' || Number.isNaN(Number(row.value))) return { error: `${f.label}: pick a list` }
        return { text: `${f.path} ${row.op === 'in_list' ? 'in list' : 'not in list'} ${Number(row.value)}` }
    }
    }
    return { error: `unknown operator ${row.op}` }
}

// wfCompile turns builder rows into condition text. Incomplete rows are skipped and reported.
function wfCompile(ui) {
    const parts = [], errors = []
    for (const [k, row] of ui.rows.entries()) {
        const r = wfCompileRow(row)
        if (r.error) errors.push(`Row ${k + 1}: ${r.error}`)
        else parts.push(r.text)
    }
    return { text: parts.join(` ${ui.join} `), errors }
}

// wfRowsFromNode maps a parsed condition tree onto builder rows. Throws when not representable.
function wfRowsFromNode(node) {
    if (!node || node.kind === 'always') return { join: 'and', rows: [] }

    const join = (node.kind === 'and' || node.kind === 'or') ? node.kind : 'and'
    const leaves = []
    const flatten = n => {
        if (n.kind === join) { flatten(n.left); flatten(n.right); return }
        if (n.kind === 'and' || n.kind === 'or') throw new Error('mixes "and" and "or" (use parentheses only in advanced mode)')
        if (n.kind === 'not') throw new Error('uses "not"')
        leaves.push(n)
    }
    flatten(node)

    return { join, rows: leaves.map(wfRowFromLeaf) }
}

function wfRowFromLeaf(n) {
    const f = wfFieldByPath(n.path)
    if (!f) throw new Error(`field "${n.path}" is not in the builder`)
    const ops = wfOpsFor(f).map(([v]) => v)
    const use = (op, value) => {
        if (!ops.includes(op)) throw new Error(`"${n.op}" is not available for ${f.label}`)
        return { path: f.path, op, value }
    }
    const scalar = lit => {
        if (f.type === 'ref' || f.type === 'number') {
            if (lit.type !== 'number') throw new Error(`${f.label} expects a number`)
            return lit.value
        }
        if (lit.type === 'null') throw new Error('null in a value list')
        return String(lit.value)
    }

    if (n.kind === 'in') return use(n.negate ? 'not_in' : 'in', n.values.map(scalar))
    if (n.kind === 'in_list') return use(n.negate ? 'not_in_list' : 'in_list', n.list_id)

    const v = n.value
    if (v.type === 'null') {
        if (n.op === '=')  return use('empty')
        if (n.op === '!=') return use('not_empty')
        throw new Error(`"${n.op} null" is not supported`)
    }
    if (f.type === 'bool') {
        if (v.type !== 'bool' || (n.op !== '=' && n.op !== '!=')) throw new Error(`${f.label} only supports "is true" / "is false"`)
        const truthy = (n.op === '=') === v.value
        return use(truthy ? 'true' : 'false')
    }
    switch (n.op) {
    case '=':        return use('eq', scalar(v))
    case '!=':       return use('ne', scalar(v))
    case '>':        return use('gt', scalar(v))
    case '<':        return use('lt', scalar(v))
    case 'contains': return use('contains', scalar(v))
    case 'like': {
        const s = String(v.value)
        const m = s.match(/^([^*%_]*)[*%]$/)
        if (!m) throw new Error(`wildcard pattern ${wfQuote(s)} is more than "starts with"`)
        return use('starts', m[1])
    }
    }
    throw new Error(`operator "${n.op}" is not supported by the builder`)
}

// wfLoadRuleUI parses a rule's condition into builder state. Never throws; falls back to advanced.
async function wfLoadRuleUI(rule) {
    const ui = wfDefaultUI()
    if (!(rule.condition || '').trim()) { rule._ui = ui; return }
    try {
        const res = await api('POST', '/workflows/parse-condition', { condition: rule.condition })
        if (!res.valid) throw new Error(res.error)
        Object.assign(ui, wfRowsFromNode(res.expr))
    } catch (e) {
        ui.mode = 'advanced'
        ui.reason = e.message
    }
    rule._ui = ui
}

// wfResolveNames fetches display names for company/contact ids referenced by builder rows.
async function wfResolveNames(rules) {
    const want = { companies: new Set(), contacts: new Set() }
    for (const r of rules) for (const row of r._ui?.rows || []) {
        const f = wfFieldByPath(row.path)
        if (!f || !want[f.source] || WF_LIST_OPS.has(row.op)) continue  // list ops hold a list id, not an entity id
        for (const v of [].concat(row.value ?? [])) if (v !== null && v !== '' && !wfNames[f.source][v]) want[f.source].add(v)
    }
    await Promise.all(Object.entries(want).map(async ([source, ids]) => {
        if (!ids.size) return
        try {
            const list = await api('GET', `/cw/${source}?ids=${[...ids].join(',')}&limit=200`)
            for (const it of list || []) wfNames[source][it.id] = wfLookupLabel(source, it)
        } catch {}
    }))
}

function wfLookupLabel(source, it) {
    if (source === 'contacts') return [it.first_name, it.last_name].filter(Boolean).join(' ') || `Contact ${it.id}`
    return it.name || `#${it.id}`
}

// ── Rendering ────────────────────────────────────────────
function wfConditionHTML(r, i) {
    const ui = r._ui || (r._ui = wfDefaultUI())
    const advanced = ui.mode === 'advanced'
    const compiled = advanced ? null : wfCompile(ui)
    const count    = advanced ? '' : `${ui.rows.length} condition${ui.rows.length === 1 ? '' : 's'}`

    // the join is one choice for the whole rule, so it lives in the header rather
    // than repeating as a cramped select on every row
    const join = !advanced && ui.rows.length > 1
        ? `<span class="cell-sub">Match</span>
           <div class="seg" role="group" aria-label="Match all or any condition">
               <button class="${ui.join === 'and' ? 'on' : ''}" onclick="wfSetJoin(${i}, 'and')">all</button>
               <button class="${ui.join === 'or' ? 'on' : ''}" onclick="wfSetJoin(${i}, 'or')">any</button>
           </div>`
        : ''

    const tabs = `<div class="cond-tabs">
        <button class="${advanced ? '' : 'on'}" onclick="wfSetCondMode(${i}, 'builder')">Builder</button>
        <button class="${advanced ? 'on' : ''}" onclick="wfSetCondMode(${i}, 'advanced')">Advanced</button>
        <span class="grow"></span>
        ${join}
        <span class="cell-sub">${count}</span>
    </div>`

    const body = advanced
        ? `<div class="cond-rows">
            ${ui.reason ? `<div class="callout warn">${icon('info')}<div class="body">
                <b>The builder cannot show this condition</b>${esc(ui.reason)}. Edit it as text here.
            </div></div>` : ''}
            <textarea class="textarea mono" id="cond-${i}" rows="2" spellcheck="false" aria-label="Condition"
                placeholder="status/name = 'New' and summary contains 'vpn'  — empty always matches"
                oninput="wfSetRule(${i}, 'condition', this.value)">${esc(r.condition)}</textarea>
        </div>`
        : `<div class="cond-rows" id="cond-builder-${i}">
            ${ui.rows.map((row, k) => wfRowHTML(i, k, row, ui)).join('')}
            <div><button class="btn btn-default btn-sm" onclick="wfAddRow(${i})">${icon('plus')}Add condition</button></div>
            ${ui.rows.length ? '' : '<p class="cell-sub">No conditions: this rule matches every ticket its trigger allows.</p>'}
        </div>`

    return `<div class="field">
        <label>Condition</label>
        <div class="cond">
            ${tabs}
            ${body}
            ${wfCondFootHTML(i, compiled)}
        </div>
    </div>`
}

// wfCondFootHTML is the compiled preview plus the validate / test controls.
function wfCondFootHTML(i, compiled) {
    return `<div class="cond-foot">
        <div class="grow" style="min-width:220px" id="cond-preview-${i}">${wfPreviewHTML(compiled)}</div>
        <button class="btn btn-default btn-sm" onclick="wfValidate(${i})">Validate</button>
        <span id="cond-result-${i}" class="cond-result" role="status"></span>
        <input type="number" id="test-ticket-${i}" class="input" style="width:110px" placeholder="Ticket #" min="1" aria-label="Ticket number to test this condition against">
        <button class="btn btn-default btn-sm" onclick="wfTest(${i})">Test</button>
        <span id="test-result-${i}" class="cond-result" role="status"></span>
    </div>`
}

// wfPreviewHTML shows the condition text the builder rows compile to. compiled is
// null in advanced mode, where the textarea already is the source of truth.
function wfPreviewHTML(compiled) {
    if (!compiled) return '<span class="cell-sub">Condition is edited as text.</span>'
    const code = compiled.text
        ? `<pre class="code" style="white-space:pre-wrap">${esc(compiled.text)}</pre>`
        : '<span class="cell-sub">Matches every ticket its trigger allows.</span>'
    return compiled.errors.length
        ? `${code}<div class="cond-result err" style="margin-top:var(--s2)">${esc(compiled.errors[0])}</div>`
        : code
}

function wfRowHTML(i, k, row, ui) {
    const f = wfFieldByPath(row.path) || wfFields[0]
    const groups = {}
    for (const fd of wfFields) (groups[fd.group] = groups[fd.group] || []).push(fd)

    // rows after the first are prefixed with the rule's join, set in the header
    const join = k === 0
        ? '<span class="cond-join-spacer"></span>'
        : `<span class="cond-join">${esc(ui.join)}</span>`

    const fieldSel = `<select class="select" style="min-width:180px" aria-label="Field" onchange="wfSetRowField(${i}, ${k}, this.value)">${
        Object.entries(groups).map(([g, fs]) => `<optgroup label="${esc(g)}">${fs.map(fd => `<option value="${esc(fd.path)}"${fd.path === f.path ? ' selected' : ''}>${esc(fd.label)}</option>`).join('')}</optgroup>`).join('')
    }</select>`
    const opSel = `<select class="select" style="min-width:130px" aria-label="Operator" onchange="wfSetRowOp(${i}, ${k}, this.value)">${
        wfOpsFor(f).map(([v, l]) => `<option value="${v}"${v === row.op ? ' selected' : ''}>${l}</option>`).join('')
    }</select>`

    return `<div class="cond-row" id="cond-row-${i}-${k}">
        ${join}${fieldSel}${opSel}
        <div class="row gap2 grow" style="min-width:200px">${wfValueHTML(i, k, row, f)}</div>
        <button class="icon-btn hit-expand" style="width:26px;height:26px" aria-label="Remove condition ${k + 1}" onclick="wfRemoveRow(${i}, ${k})">${icon('trash')}</button>
    </div>`
}

function wfValueHTML(i, k, row, f) {
    if (WF_NO_VALUE_OPS.has(row.op)) return ''
    if (WF_LIST_OPS.has(row.op)) return wfListSelectHTML(i, k, row, f)
    const multi = WF_MULTI_OPS.has(row.op)

    if (f.source) {
        const list = wfSourceList(f)
        const searchable = f.source === 'companies' || f.source === 'contacts'
        const labelFor = v => searchable ? (wfNames[f.source][v] || `#${v}`) : (list.find(o => String(o.value) === String(v))?.label ?? String(v))

        if (!multi) {
            if (searchable) return wfTypeaheadHTML(i, k, f, row.value === null || row.value === '' ? '' : `${labelFor(row.value)}`)
            return `<select class="select grow" aria-label="Value" onchange="wfSetRowValue(${i}, ${k}, this.value)">
                <option value="">— choose —</option>${list.map(o => `<option value="${esc(String(o.value))}"${String(o.value) === String(row.value ?? '') ? ' selected' : ''}>${esc(o.label)}</option>`).join('')}
            </select>`
        }

        const vals = Array.isArray(row.value) ? row.value : []
        const chips = vals.map(v => `<span class="chip">${esc(labelFor(v))}<button class="chip-x hit-expand" type="button" aria-label="Remove ${esc(labelFor(v))}" onclick="wfRemoveRowValue(${i}, ${k}, '${esc(String(v))}')">${icon('x')}</button></span>`).join('')
        const adder = searchable
            ? wfTypeaheadHTML(i, k, f, '', true)
            : `<select class="select" style="width:auto" aria-label="Add a value" onchange="if (this.value) { wfAddRowValue(${i}, ${k}, this.value); this.value = '' }">
                <option value="">+ add…</option>${list.filter(o => !vals.map(String).includes(String(o.value))).map(o => `<option value="${esc(String(o.value))}">${esc(o.label)}</option>`).join('')}
            </select>`
        return `<div class="chips grow">${chips}${adder}</div>`
    }

    const type = f.type === 'number' ? 'number' : 'text'
    if (multi) {
        const text = Array.isArray(row.value) ? row.value.join(', ') : ''
        return `<input class="input grow" type="text" aria-label="Values, comma separated" placeholder="value, value, …" value="${esc(text)}" oninput="wfSetRowValue(${i}, ${k}, this.value.split(',').map(s => s.trim()).filter(Boolean))">`
    }
    return `<input class="input grow" type="${type}" aria-label="Value" placeholder="value" value="${esc(row.value ?? '')}" oninput="wfSetRowValue(${i}, ${k}, this.value)">`
}

// wfListSelectHTML offers the admin lists whose item type matches the field.
function wfListSelectHTML(i, k, row, f) {
    const lists = wfLists.filter(l => l.item_type === f.list_type)
    const current = row.value === null || row.value === undefined ? '' : String(row.value)
    const known = lists.some(l => String(l.id) === current)
    const opts = lists.map(l => `<option value="${l.id}"${String(l.id) === current ? ' selected' : ''}>${esc(l.name)} (${l.item_count})</option>`).join('')
    const missing = current && !known ? `<option value="${esc(current)}" selected disabled>(missing list #${esc(current)})</option>` : ''
    const empty = lists.length ? '' : `<option value="" disabled>No ${esc(f.list_type)} lists yet — create one under Lists</option>`
    return `<select class="select grow" aria-label="List" onchange="wfSetRowValue(${i}, ${k}, this.value === '' ? null : Number(this.value))">
        <option value="">— choose list —</option>${missing}${opts}${empty}
    </select>`
}

function wfTypeaheadHTML(i, k, f, current, add = false) {
    const id = `ta-${i}-${k}-${add ? 'add' : 'one'}`
    return `<span class="typeahead grow">
        <input class="input" type="text" id="${id}" list="${id}-list" autocomplete="off"
            aria-label="${add ? `Add a ${esc(f.source)}` : `Search ${esc(f.source)}`}"
            placeholder="${add ? '+ search…' : `search ${esc(f.source)}…`}" value="${esc(current)}"
            oninput="wfTypeahead(${i}, ${k}, '${f.source}', this)">
        <datalist id="${id}-list"></datalist>
    </span>`
}

function wfSourceList(f) {
    switch (f.source) {
    case 'statuses':   return wfStatuses.map(s => ({ value: s.id, label: s.name }))
    case 'priorities': return wfPriorities.map(p => ({ value: p.id, label: p.name }))
    case 'boards':     return wfBoards.filter(b => !b.deleted).map(b => ({ value: b.id, label: b.name }))
    case 'members':
        return wfMembers.slice().sort((a, b) => memberLabel(a).localeCompare(memberLabel(b)))
            .map(m => ({ value: f.type === 'ref' ? m.id : m.identifier, label: `${memberLabel(m)} (${m.identifier})` }))
    }
    return []
}

// ── Typeahead (companies / contacts) ─────────────────────
// Options are rendered as "Name (#id)". Picking one from the datalist fires `input` with that
// exact text, so we commit immediately instead of waiting for `change` (blur / Enter).
function wfTypeahead(i, k, source, input) {
    clearTimeout(wfTypeaheadTimer)
    if (wfTypeaheadPick(i, k, source, input, input.id.endsWith('-add'))) return
    const q = input.value.trim()
    if (q.length < 2) return
    wfTypeaheadTimer = setTimeout(async () => {
        const params = new URLSearchParams({ q, limit: 15 })
        if (source === 'contacts') {
            const company = wfSiblingCompany(i, k)
            if (company) params.set('company_id', company)
        }
        try {
            const list = (await api('GET', `/cw/${source}?${params}`)) || []
            const dl = document.getElementById(`${input.id}-list`)
            if (!dl) return
            dl.innerHTML = list.map(it => {
                const label = wfLookupLabel(source, it)
                wfNames[source][it.id] = label
                return `<option value="${esc(`${label} (#${it.id})`)}"></option>`
            }).join('')
        } catch (e) { toast(e.message, 'error') }
    }, 250)
}

function wfTypeaheadPick(i, k, source, input, add) {
    const m = input.value.match(/\(#(\d+)\)\s*$/)
    if (!m) return false
    const id = parseInt(m[1])
    if (add) { input.value = ''; wfAddRowValue(i, k, id) }
    else wfSetRowValue(i, k, id, true)
    return true
}

// wfSiblingCompany finds a "Company is X" row in the same rule, to scope contact searches.
function wfSiblingCompany(i, k) {
    for (const [j, row] of wf.rules[i]._ui.rows.entries()) {
        if (j !== k && row.path === 'company/id' && row.op === 'eq' && row.value) return row.value
    }
    return null
}

// ── Mutations ────────────────────────────────────────────
function wfSyncCondition(i) {
    const r = wf.rules[i]
    const compiled = wfCompile(r._ui)
    r.condition = compiled.text
    const el = document.getElementById(`cond-preview-${i}`)
    if (el) el.innerHTML = wfPreviewHTML(compiled)
    wfMarkDirty()
}

function wfRerenderCondition(i) {
    wfSyncCondition(i)
    wfRerenderRule(i)
}

async function wfSetCondMode(i, mode) {
    const r = wf.rules[i]
    if (mode === 'advanced') {
        r._ui.mode = 'advanced'
        r._ui.reason = ''
        wfRerenderRule(i)
        return
    }
    await wfLoadRuleUI(r)
    if (r._ui.mode !== 'builder') {
        toast(`Builder can't show this condition: ${r._ui.reason}`, 'error')
    } else {
        await wfResolveNames([r])
    }
    wfRerenderRule(i)
}

function wfSetJoin(i, join) {
    wf.rules[i]._ui.join = join
    wfRerenderCondition(i)
}

function wfAddRow(i) {
    wf.rules[i]._ui.rows.push(wfNewRow())
    wfRerenderCondition(i)
}

function wfRemoveRow(i, k) {
    wf.rules[i]._ui.rows.splice(k, 1)
    wfRerenderCondition(i)
}

function wfSetRowField(i, k, path) {
    const row = wf.rules[i]._ui.rows[k]
    const f = wfFieldByPath(path)
    row.path = f.path
    row.op = wfOpsFor(f)[0][0]
    row.value = null
    wfRerenderCondition(i)
}

function wfSetRowOp(i, k, op) {
    const row = wf.rules[i]._ui.rows[k]
    const wasMulti = WF_MULTI_OPS.has(row.op), isMulti = WF_MULTI_OPS.has(op)
    const listChanged = WF_LIST_OPS.has(row.op) !== WF_LIST_OPS.has(op)  // a list id is not an entity id
    row.op = op
    if (WF_NO_VALUE_OPS.has(op) || listChanged) row.value = null
    else if (isMulti && !wasMulti) row.value = row.value === null || row.value === '' ? [] : [row.value]
    else if (!isMulti && wasMulti) row.value = Array.isArray(row.value) ? (row.value[0] ?? null) : row.value
    wfRerenderCondition(i)
}

function wfSetRowValue(i, k, value, rerender = false) {
    wf.rules[i]._ui.rows[k].value = value
    if (rerender) wfRerenderCondition(i)
    else wfSyncCondition(i)
}

function wfAddRowValue(i, k, value) {
    const row = wf.rules[i]._ui.rows[k]
    const f = wfFieldByPath(row.path)
    const v = (f.type === 'ref' || f.type === 'number') ? Number(value) : value
    row.value = Array.isArray(row.value) ? row.value : []
    if (row.value.map(String).includes(String(v))) return
    row.value.push(v)
    wfRerenderCondition(i)
}

function wfRemoveRowValue(i, k, value) {
    const row = wf.rules[i]._ui.rows[k]
    row.value = (Array.isArray(row.value) ? row.value : []).filter(v => String(v) !== String(value))
    wfRerenderCondition(i)
}
