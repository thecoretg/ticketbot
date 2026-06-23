// ─────────────────────────────────────────────────────────
// Theme
// ─────────────────────────────────────────────────────────
function initTheme() {
    const stored    = localStorage.getItem('theme')
    const preferred = window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
    applyTheme(stored || preferred)
}

function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
    const btn = document.getElementById('theme-toggle')
    if (btn) btn.textContent = theme === 'light' ? '☽' : '☀'
}

function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme')
    applyTheme(current === 'light' ? 'dark' : 'light')
}

initTheme()

// ─────────────────────────────────────────────────────────
// State
// ─────────────────────────────────────────────────────────
let currentTab        = 'rules'
let currentUser       = null
let pendingToken      = null   // pending TOTP token after password login
let totpEnabled       = false  // cached TOTP status for account menu
let totpSetupRequired = false  // true when server enforces TOTP and user hasn't set it up
let requireTOTP       = false  // cached value of config.require_totp
let syncPollTimer     = null
let modalSubmitFn     = null

// ─────────────────────────────────────────────────────────
// Password requirements
// ─────────────────────────────────────────────────────────
const PWD_REQS = [
    { label: '8+ characters',    test: p => p.length >= 8 },
    { label: 'Uppercase letter', test: p => /[A-Z]/.test(p) },
    { label: 'Lowercase letter', test: p => /[a-z]/.test(p) },
    { label: 'Number',           test: p => /[0-9]/.test(p) },
]

function passwordValid(pwd) {
    return PWD_REQS.every(r => r.test(pwd))
}

function attachPwdReqs(inputId, containerId) {
    const input     = document.getElementById(inputId)
    const container = document.getElementById(containerId)
    if (!input || !container) return
    const update = () => {
        const p = input.value
        container.innerHTML = PWD_REQS.map(r => {
            const ok = r.test(p)
            return `<span class="pwd-req${ok ? ' ok' : ''}">${ok ? '✓' : '○'} ${r.label}</span>`
        }).join('')
    }
    input.addEventListener('input', update)
    update()
}

// ─────────────────────────────────────────────────────────
// API helper
// ─────────────────────────────────────────────────────────
async function api(method, path, body = null) {
    const opts = {
        method,
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
    }
    if (body !== null) opts.body = JSON.stringify(body)

    const res = await fetch(path, opts)

    const ct = res.headers.get('content-type') || ''
    if (!ct.includes('application/json')) {
        if (!res.ok) throw new Error(`Request failed: ${res.status}`)
        return null
    }

    const data = await res.json()
    if (!res.ok) throw new Error(data.error || `Request failed: ${res.status}`)
    return data
}

// ─────────────────────────────────────────────────────────
// Auth
// ─────────────────────────────────────────────────────────
async function login() {
    const email    = document.getElementById('login-email').value.trim()
    const password = document.getElementById('login-password').value
    const errEl    = document.getElementById('login-err')
    const btn      = document.getElementById('login-btn')

    if (!email || !password) {
        showLoginErr('Email and password are required')
        return
    }

    btn.disabled    = true
    btn.textContent = 'Signing in…'
    errEl.classList.add('hidden')

    try {
        const res = await api('POST', '/auth/login', { email, password })
        if (res?.totp_required) {
            pendingToken = res.pending_token
            showTOTPVerify()
        } else if (res?.reset_required) {
            showPasswordReset()
        } else {
            await showApp()
            if (res?.totp_setup_required) showTOTPSetupModal(true)
        }
    } catch {
        showLoginErr('Invalid email or password')
    } finally {
        btn.disabled    = false
        btn.textContent = 'Login'
    }
}

function showLoginErr(msg) {
    const el = document.getElementById('login-err')
    el.textContent = msg
    el.classList.remove('hidden')
}

async function logout() {
    try { await api('POST', '/auth/logout') } catch {}
    currentUser  = null
    pendingToken = null
    totpEnabled  = false
    stopSyncPoll()
    document.getElementById('login-email').value    = ''
    document.getElementById('login-password').value = ''
    document.getElementById('login-err').classList.add('hidden')
    document.getElementById('account-dropdown').classList.add('hidden')
    document.getElementById('login').classList.remove('hidden')
    document.getElementById('app').classList.add('hidden')
    document.getElementById('password-reset').classList.add('hidden')
    document.getElementById('totp-verify').classList.add('hidden')
}

function showTOTPVerify() {
    document.getElementById('login').classList.add('hidden')
    document.getElementById('totp-verify').classList.remove('hidden')
    document.getElementById('totp-code').value = ''
    document.getElementById('totp-err').classList.add('hidden')
    document.getElementById('totp-code').focus()
}

async function submitTOTPVerify() {
    const code  = document.getElementById('totp-code').value.trim()
    const errEl = document.getElementById('totp-err')
    const btn   = document.getElementById('totp-btn')

    errEl.classList.add('hidden')
    if (!code) {
        errEl.textContent = 'Code is required'
        errEl.classList.remove('hidden')
        return
    }

    btn.disabled    = true
    btn.textContent = 'Verifying…'

    try {
        const res = await api('POST', '/auth/totp/verify', { pending_token: pendingToken, code })
        pendingToken = null
        if (res.reset_required) {
            document.getElementById('totp-verify').classList.add('hidden')
            showPasswordReset()
        } else {
            showApp()
            if (res.recovery_code_used) {
                toast('You logged in with a recovery code. Check your 2FA setup in the account menu.', 'error')
            }
        }
    } catch {
        errEl.textContent = 'Invalid or expired code'
        errEl.classList.remove('hidden')
    } finally {
        btn.disabled    = false
        btn.textContent = 'Verify'
    }
}

function showPasswordReset() {
    document.getElementById('login').classList.add('hidden')
    document.getElementById('totp-verify').classList.add('hidden')
    document.getElementById('password-reset').classList.remove('hidden')
    attachPwdReqs('reset-new', 'reset-reqs')
    document.getElementById('reset-current').focus()
}

async function submitPasswordReset() {
    const currentPwd = document.getElementById('reset-current').value
    const newPwd     = document.getElementById('reset-new').value
    const confirm    = document.getElementById('reset-confirm').value
    const errEl      = document.getElementById('reset-err')
    const btn        = document.getElementById('reset-btn')

    errEl.classList.add('hidden')

    if (!currentPwd || !newPwd || !confirm) {
        errEl.textContent = 'All fields are required'
        errEl.classList.remove('hidden')
        return
    }
    if (!passwordValid(newPwd)) {
        errEl.textContent = 'Password does not meet requirements'
        errEl.classList.remove('hidden')
        return
    }
    if (newPwd !== confirm) {
        errEl.textContent = 'New passwords do not match'
        errEl.classList.remove('hidden')
        return
    }

    btn.disabled    = true
    btn.textContent = 'Saving…'

    try {
        await api('PUT', '/auth/password', { current_password: currentPwd, new_password: newPwd })
        document.getElementById('password-reset').classList.add('hidden')
        document.getElementById('reset-current').value = ''
        document.getElementById('reset-new').value     = ''
        document.getElementById('reset-confirm').value = ''
        showApp()
    } catch (e) {
        errEl.textContent = e.message || 'Failed to change password'
        errEl.classList.remove('hidden')
    } finally {
        btn.disabled    = false
        btn.textContent = 'Change Password'
    }
}

async function showApp() {
    document.getElementById('login').classList.add('hidden')
    document.getElementById('totp-verify').classList.add('hidden')
    document.getElementById('password-reset').classList.add('hidden')
    document.getElementById('app').classList.remove('hidden')
    try {
        const [me, totp, cfg] = await Promise.all([
            api('GET', '/users/me'),
            api('GET', '/auth/totp'),
            api('GET', '/config'),
        ])
        currentUser = me
        totpEnabled = totp.enabled
        requireTOTP = cfg.require_totp
        document.getElementById('header-email').textContent   = currentUser.email_address
        document.getElementById('dropdown-email').textContent = currentUser.email_address
        updateTOTPMenuItem()
        if (requireTOTP && !totpEnabled) {
            showTOTPSetupModal(true)
            return
        }
    } catch {}
    const hash = window.location.hash.replace('#', '')
    switchTab(tabLoaders[hash] ? hash : 'tickets')
}

// ─────────────────────────────────────────────────────────
// Account menu
// ─────────────────────────────────────────────────────────
function toggleAccountMenu(e) {
    e.stopPropagation()
    document.getElementById('account-dropdown').classList.toggle('hidden')
}

function showChangePasswordModal() {
    document.getElementById('account-dropdown').classList.add('hidden')
    openModal('Change Password', `
        <div class="form-group">
            <label>Current Password</label>
            <input type="password" id="f-cur-pwd" autocomplete="current-password">
        </div>
        <div class="form-group">
            <label>New Password</label>
            <input type="password" id="f-new-pwd" autocomplete="new-password">
            <div id="f-pwd-reqs" class="pwd-reqs"></div>
        </div>
        <div class="form-group">
            <label>Confirm New Password</label>
            <input type="password" id="f-confirm-pwd" autocomplete="new-password">
        </div>`, async () => {
        const cur     = document.getElementById('f-cur-pwd').value
        const newPwd  = document.getElementById('f-new-pwd').value
        const confirm = document.getElementById('f-confirm-pwd').value
        if (!passwordValid(newPwd))  { toast('Password does not meet requirements', 'error'); return }
        if (newPwd !== confirm)      { toast('New passwords do not match', 'error'); return }
        try {
            await api('PUT', '/auth/password', { current_password: cur, new_password: newPwd })
            closeModal()
            toast('Password changed', 'success')
        } catch (e) { toast(e.message, 'error') }
    }, 'Change Password')
    setTimeout(() => attachPwdReqs('f-new-pwd', 'f-pwd-reqs'), 50)
}

function updateTOTPMenuItem() {
    const btn = document.getElementById('totp-menu-btn')
    if (!btn) return
    if (!totpEnabled) {
        btn.textContent = 'Set Up 2FA'
    } else if (requireTOTP) {
        btn.textContent = 'Reset 2FA'
    } else {
        btn.textContent = 'Disable 2FA'
    }
}

function handleTOTPMenuClick() {
    document.getElementById('account-dropdown').classList.add('hidden')
    if (!totpEnabled) {
        showTOTPSetupModal()
    } else if (requireTOTP) {
        showTOTPSetupModal()  // reset: dismissible, old secret stays until new one is confirmed
    } else {
        showTOTPDisableModal()
    }
}

async function showTOTPSetupModal(required = false) {
    // Phase 1: fetch QR code and secret
    let setupData
    try {
        setupData = await api('POST', '/auth/totp/setup')
    } catch (e) { toast(e.message, 'error'); return }

    if (required) totpSetupRequired = true

    const desc = required
        ? '<p style="color:var(--warning);font-size:13px">Two-factor authentication is required for this account. Set it up to continue.</p>'
        : '<p style="color:var(--muted);font-size:13px">Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.).</p>'

    openModal('Set Up Two-Factor Auth', `
        ${desc}
        <img class="qr-code" src="data:image/png;base64,${setupData.qr_png}" alt="TOTP QR Code">
        <div class="form-group">
            <label>Or enter this secret manually</label>
            <div class="secret-display">${esc(setupData.secret)}</div>
        </div>
        <div class="form-group">
            <label>Current Password</label>
            <input type="password" id="f-totp-pwd" autocomplete="current-password">
        </div>
        <div class="form-group">
            <label>Confirmation Code <span style="color:var(--muted)">(from your authenticator app)</span></label>
            <input type="text" id="f-totp-code" inputmode="numeric" autocomplete="one-time-code" placeholder="000000" maxlength="6">
        </div>`, async () => {
        const pwd  = document.getElementById('f-totp-pwd').value
        const code = document.getElementById('f-totp-code').value.trim()
        if (!pwd)  { toast('Password is required', 'error'); return }
        if (!code) { toast('Confirmation code is required', 'error'); return }
        try {
            const res = await api('PUT', '/auth/totp/setup', {
                password: pwd,
                code,
                secret: setupData.secret,
            })
            totpEnabled = true
            totpSetupRequired = false
            updateTOTPMenuItem()
            // Replace modal body with recovery codes (shown once)
            document.getElementById('modal-title').textContent = '2FA Enabled'
            document.getElementById('modal-body').innerHTML = `
                <p style="color:var(--warning);font-size:13px">
                    ⚠ Save these recovery codes somewhere safe. Each can only be used once and they will not be shown again.
                </p>
                <div class="recovery-codes">
                    ${res.recovery_codes.map(c => `<div class="recovery-code">${esc(c)}</div>`).join('')}
                </div>`
            document.getElementById('modal-footer').innerHTML = `
                <button class="btn btn-ghost" onclick="copyRecoveryCodes()">Copy All</button>
                <button class="btn btn-primary" onclick="finishTOTPSetup()">Done</button>`
            modalSubmitFn = null
            window._recoveryCodes = res.recovery_codes
        } catch (e) { toast(e.message || 'Failed to enable 2FA', 'error') }
    }, 'Enable 2FA')

    if (required) {
        // Remove the cancel button — setup cannot be skipped when required
        const cancel = document.querySelector('#modal-footer .btn-ghost')
        if (cancel) cancel.remove()
    }
}

function finishTOTPSetup() {
    document.getElementById('modal-overlay').classList.add('hidden')
    modalSubmitFn = null
    const hash = window.location.hash.replace('#', '')
    switchTab(tabLoaders[hash] ? hash : 'tickets')
}

function copyRecoveryCodes() {
    const codes = (window._recoveryCodes || []).join('\n')
    navigator.clipboard.writeText(codes).then(() => toast('Recovery codes copied', 'success'))
}

function showTOTPDisableModal() {
    openModal('Disable Two-Factor Auth', `
        <p style="color:var(--muted);font-size:13px">Enter your current password to disable 2FA. Your recovery codes will also be removed.</p>
        <div class="form-group">
            <label>Current Password</label>
            <input type="password" id="f-disable-pwd" autocomplete="current-password">
        </div>`, async () => {
        const pwd = document.getElementById('f-disable-pwd').value
        if (!pwd) { toast('Password is required', 'error'); return }
        try {
            await api('DELETE', '/auth/totp', { password: pwd })
            totpEnabled = false
            updateTOTPMenuItem()
            closeModal()
            toast('Two-factor authentication disabled', 'success')
        } catch (e) { toast(e.message || 'Failed to disable 2FA', 'error') }
    }, 'Disable 2FA')
}

function confirmRestart() {
    document.getElementById('account-dropdown').classList.add('hidden')
    openModal('Restart Server', `
        <p style="color:var(--muted);font-size:13px">
            The server will restart and reconnect automatically. This usually takes a few seconds.
        </p>`, async () => {
        try {
            await api('POST', '/admin/restart')
        } catch { /* server closes the connection during shutdown, that's fine */ }
        closeModal()
        showRestartBanner()
    }, 'Restart')
}

function showRestartBanner() {
    const content = document.getElementById('content')
    const banner = document.createElement('div')
    banner.id = 'restart-banner'
    banner.innerHTML = `<div class="restart-banner">Restarting… reconnecting</div>`
    document.getElementById('app').prepend(banner)

    const poll = setInterval(async () => {
        try {
            await fetch('/healthcheck')
            clearInterval(poll)
            banner.remove()
            toast('Server restarted successfully', 'success')
        } catch { /* still down, keep polling */ }
    }, 1500)
}

function checkSavedKey() {
    api('GET', '/authtest')
        .then(showApp)
        .catch(() => {
            document.getElementById('login').classList.remove('hidden')
        })
}

// ─────────────────────────────────────────────────────────
// Tabs
// ─────────────────────────────────────────────────────────
const tabLoaders = {
    tickets:   loadTickets,
    workflows: loadWorkflows,
    notifier:  loadNotifier,
    users:    loadUsers,
    keys:     loadKeys,
    sync:     loadSync,
    config:   loadConfig,
    logs:     loadLogs,
}

function switchTab(tab) {
    stopSyncPoll()
    stopLogsPoll()
    stopTicketsPoll()
    currentTab = tab
    window.location.hash = tab
    document.querySelectorAll('.nav-item').forEach(el => {
        el.classList.toggle('active', el.dataset.tab === tab)
    })
    setContent('<div class="loading-state">Loading…</div>')
    tabLoaders[tab]()
}

function setContent(html) {
    document.getElementById('content').innerHTML = html
}

// ─────────────────────────────────────────────────────────
// Toast
// ─────────────────────────────────────────────────────────
let toastTimer = null
function toast(msg, type = 'info') {
    const el = document.getElementById('toast')
    el.textContent = msg
    el.className   = `toast ${type}`
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => el.classList.add('hidden'), 3500)
}

// ─────────────────────────────────────────────────────────
// Modal
// ─────────────────────────────────────────────────────────
function openModal(title, bodyHTML, submitFn, submitLabel = 'Create') {
    document.getElementById('modal-title').textContent  = title
    document.getElementById('modal-body').innerHTML     = bodyHTML
    document.getElementById('modal-footer').innerHTML   = `
        <button class="btn btn-ghost" onclick="closeModal()">Cancel</button>
        <button id="modal-submit" class="btn btn-primary">${submitLabel}</button>`
    document.getElementById('modal-submit').addEventListener('click', handleModalSubmit)
    modalSubmitFn = submitFn
    document.getElementById('modal-overlay').classList.remove('hidden')
    setTimeout(() => {
        const first = document.querySelector('#modal-body input, #modal-body select')
        if (first) first.focus()
    }, 50)
}

async function handleModalSubmit() {
    if (!modalSubmitFn) return
    const btn    = document.getElementById('modal-submit')
    btn.disabled = true
    try {
        await modalSubmitFn()
    } finally {
        if (btn) btn.disabled = false
    }
}

function closeModal() {
    if (totpSetupRequired) return
    document.getElementById('modal-overlay').classList.add('hidden')
    modalSubmitFn = null
}

function handleOverlayClick(e) {
    if (e.target === document.getElementById('modal-overlay')) closeModal()
}

// ─────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────
function esc(str) {
    return String(str ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
}

function fmtDateTime(d) {
    if (!d) return '—'
    return new Date(d).toLocaleString()
}

function fmtDateRange(start, end) {
    if (!start && !end) return '—'
    const fmt = d => new Date(d).toLocaleString('en-US', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
    const s   = start ? fmt(start) : '∞'
    const e   = end   ? fmt(end)   : '∞'
    return `${s} – ${e}`
}

function badge(val) {
    return val
        ? '<span class="badge badge-on">Yes</span>'
        : '<span class="badge badge-off">No</span>'
}

function tableWrap(thead, rows) {
    if (!rows.length) return '<div class="empty-state">No items found</div>'
    return `<div class="table-wrap"><table>
        <thead><tr>${thead}</tr></thead>
        <tbody>${rows.join('')}</tbody>
    </table></div>`
}

// ─────────────────────────────────────────────────────────
// Tickets (per-ticket lifecycle journal)
// ─────────────────────────────────────────────────────────
const TICKET_COLUMNS = [
    { key: 'ticket_id',    label: '#',             sort: 'num' },
    { key: 'summary',      label: 'Summary' },
    { key: 'company_name', label: 'Company' },
    { key: 'contact_name', label: 'Contact' },
    { key: 'status_name',  label: 'Status' },
    { key: 'board_name',   label: 'Board' },
    { key: 'owner_name',   label: 'Owner' },
    { key: 'last_run',     label: 'Last Activity', sort: 'date' },
    { key: 'last_outcome', label: 'Outcome' },
]

function ticketsPrefs() {
    try { return JSON.parse(localStorage.getItem('ticketsPrefs') || '{}') } catch { return {} }
}
function saveTicketsPrefs(patch) {
    localStorage.setItem('ticketsPrefs', JSON.stringify({ ...ticketsPrefs(), ...patch }))
}

let ticketsList      = []
let ticketsPollTimer = null
let ticketsView      = 'list'
const _tp            = ticketsPrefs()
let ticketsSearch    = ''
let ticketsOutcome   = _tp.outcome    ?? 'ALL'
let ticketsHideNoOp  = _tp.hideNoOp   ?? false
let ticketsSortKey   = _tp.sortKey    ?? 'last_run'
let ticketsSortDir   = _tp.sortDir    ?? 'desc'

async function loadTickets() {
    ticketsView = 'list'
    try {
        const items = await api('GET', '/tickets')
        ticketsList = items || []
        renderTickets()
        startTicketsPoll()
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function ticketOutcomeMatch(t) {
    switch (ticketsOutcome) {
        case 'COMPLETED': return !t.had_error && t.last_outcome === 'Completed'
        case 'ERRORS':    return !!t.had_error
        case 'NOOP':      return t.last_outcome === 'Nothing to do'
        default:          return true
    }
}

function renderTickets() {
    let rows = ticketsList.slice()

    if (ticketsHideNoOp) rows = rows.filter(t => t.last_outcome !== 'Nothing to do')
    rows = rows.filter(ticketOutcomeMatch)

    if (ticketsSearch) {
        const term = ticketsSearch.toLowerCase()
        rows = rows.filter(t => TICKET_COLUMNS.some(c => String(t[c.key] ?? '').toLowerCase().includes(term)))
    }

    const col = TICKET_COLUMNS.find(c => c.key === ticketsSortKey) || TICKET_COLUMNS[0]
    rows.sort((a, b) => {
        let av = a[col.key], bv = b[col.key]
        if (col.sort === 'num')  { av = +av || 0; bv = +bv || 0 }
        else if (col.sort === 'date') { av = av ? new Date(av).getTime() : 0; bv = bv ? new Date(bv).getTime() : 0 }
        else { av = String(av ?? '').toLowerCase(); bv = String(bv ?? '').toLowerCase() }
        const cmp = av < bv ? -1 : av > bv ? 1 : 0
        return ticketsSortDir === 'asc' ? cmp : -cmp
    })

    const arrow = (k) => ticketsSortKey === k ? (ticketsSortDir === 'asc' ? ' ▲' : ' ▼') : ''
    const thead = TICKET_COLUMNS.map(c =>
        `<th class="th-sort" onclick="ticketsSort('${c.key}')">${esc(c.label)}${arrow(c.key)}</th>`).join('')

    const body = rows.map(t => `<tr class="row-click" onclick="showTicketJournal(${t.ticket_id})">
        <td style="color:var(--muted)">${t.ticket_id}</td>
        <td>${esc(t.summary) || '<span style="color:var(--muted)">—</span>'}</td>
        <td>${esc(t.company_name) || '—'}</td>
        <td>${esc(t.contact_name) || '—'}</td>
        <td>${esc(t.status_name) || '—'}</td>
        <td>${esc(t.board_name) || '—'}</td>
        <td>${esc(t.owner_name) || '—'}</td>
        <td style="white-space:nowrap;color:var(--muted)">${t.last_run ? new Date(t.last_run).toLocaleString() : '—'}</td>
        <td>${outcomeBadge(t)}</td>
    </tr>`)

    const table = rows.length
        ? `<div class="table-wrap"><table><thead><tr>${thead}</tr></thead><tbody>${body.join('')}</tbody></table></div>`
        : '<div class="empty-state">No tickets found</div>'

    setContent(`<div class="tab-header">
        <h2>Tickets</h2>
        <div class="logs-controls">
            <input class="logs-search-input" type="text" placeholder="Search tickets…" value="${esc(ticketsSearch)}"
                oninput="ticketsSearch=this.value; renderTickets()">
            <select class="logs-search-input" style="width:auto" onchange="ticketsOutcome=this.value; saveTicketsPrefs({outcome:this.value}); renderTickets()">
                ${[['ALL','All outcomes'],['COMPLETED','Completed'],['ERRORS','Errors'],['NOOP','Nothing to do']].map(([v,l]) =>
                    `<option value="${v}"${ticketsOutcome===v?' selected':''}>${l}</option>`).join('')}
            </select>
            <label class="logs-opt"><input type="checkbox" ${ticketsHideNoOp?'checked':''}
                onchange="ticketsHideNoOp=this.checked; saveTicketsPrefs({hideNoOp:this.checked}); renderTickets()"> Hide no-op tickets</label>
            <button class="btn btn-ghost btn-sm" onclick="loadTickets()">Refresh</button>
        </div>
    </div>${table}`)
}

function ticketsSort(key) {
    if (ticketsSortKey === key) ticketsSortDir = ticketsSortDir === 'asc' ? 'desc' : 'asc'
    else { ticketsSortKey = key; ticketsSortDir = 'asc' }
    saveTicketsPrefs({ sortKey: ticketsSortKey, sortDir: ticketsSortDir })
    renderTickets()
}

function outcomeBadge(t) {
    if (t.had_error) return `<span class="badge badge-off">${esc(t.last_outcome || 'Error')}</span>`
    if (t.last_outcome === 'Completed') return `<span class="badge badge-on">Completed</span>`
    return `<span class="badge badge-neutral">${esc(t.last_outcome || '—')}</span>`
}

let journalHideNoOp = ticketsPrefs().journalHideNoOp ?? false

async function showTicketJournal(id) {
    ticketsView = 'detail'
    stopTicketsPoll()
    let j
    try {
        j = await api('GET', `/tickets/${id}`)
    } catch (e) { toast(e.message, 'error'); return }
    renderTicketJournal(j)
}

function renderTicketJournal(j) {
    const runs = (j.runs || []).slice().reverse() // newest first
    const shown = journalHideNoOp ? runs.filter(r => r.outcome !== 'Nothing to do') : runs
    const hidden = runs.length - shown.length

    const prop = (label, val) => `<div class="jr-prop"><span class="jr-prop-label">${label}</span><span>${esc(val) || '—'}</span></div>`

    const timeline = shown.length ? shown.map(r => `
        <div class="run-card ${r.had_error ? 'run-error' : ''}">
            <div class="run-head">
                <span class="run-trigger">${esc(r.trigger)}</span>
                <span class="run-time">${r.time ? new Date(r.time).toLocaleString() : ''}</span>
                ${outcomeBadge({ had_error: r.had_error, last_outcome: r.outcome })}
            </div>
            <div class="run-events">
                ${(r.events || []).map(e => `<div class="journal-event ev-${esc(e.status)}">${esc(e.text)}</div>`).join('') ||
                    '<div class="journal-event ev-info">No actions taken</div>'}
            </div>
        </div>`).join('')
        : '<div class="empty-state">No runs to show</div>'

    setContent(`<div class="tab-header">
        <h2><button class="btn btn-ghost btn-sm" onclick="loadTickets()">← Tickets</button> &nbsp; Ticket #${j.ticket_id}</h2>
        <label class="logs-opt"><input type="checkbox" ${journalHideNoOp ? 'checked' : ''}
            onchange="journalHideNoOp=this.checked; saveTicketsPrefs({journalHideNoOp:this.checked}); rerenderTicketJournal()"> Hide no-op runs${hidden > 0 ? ` (${hidden} hidden)` : ''}</label>
    </div>
    <div class="jr-summary">${esc(j.summary) || '<span style="color:var(--muted)">(no summary)</span>'}</div>
    <div class="jr-props">
        ${prop('Company', j.company_name)}
        ${prop('Contact', j.contact_name)}
        ${prop('Status', j.status_name)}
        ${prop('Board', j.board_name)}
        ${prop('Owner', j.owner_name)}
    </div>
    <div class="timeline">${timeline}</div>`)

    window._journalCache = j
}

function rerenderTicketJournal() {
    if (window._journalCache) renderTicketJournal(window._journalCache)
}

function startTicketsPoll() {
    stopTicketsPoll()
    ticketsPollTimer = setInterval(async () => {
        if (currentTab !== 'tickets' || ticketsView !== 'list') { stopTicketsPoll(); return }
        try {
            const items = await api('GET', '/tickets')
            ticketsList = items || []
            renderTickets()
        } catch { stopTicketsPoll() }
    }, 5000)
}

function stopTicketsPoll() {
    if (ticketsPollTimer) { clearInterval(ticketsPollTimer); ticketsPollTimer = null }
}

// ─────────────────────────────────────────────────────────
// Notifier (Board Settings + Forwards under one menu)
// ─────────────────────────────────────────────────────────
let notifierSubtab = 'board-settings'

function loadNotifier() {
    renderNotifierShell()
    loadNotifierSubtab()
}

function renderNotifierShell() {
    setContent(`
        <div class="tab-header"><h2>Notifier</h2></div>
        <div class="subtabs">
            <button class="subtab" data-sub="board-settings" onclick="switchNotifierSubtab('board-settings')">Board Settings</button>
            <button class="subtab" data-sub="forwards" onclick="switchNotifierSubtab('forwards')">Forwards</button>
        </div>
        <div id="notifier-panel"></div>`)
    updateNotifierSubtabActive()
}

function updateNotifierSubtabActive() {
    document.querySelectorAll('.subtab').forEach(el =>
        el.classList.toggle('active', el.dataset.sub === notifierSubtab))
}

function switchNotifierSubtab(sub) {
    notifierSubtab = sub
    updateNotifierSubtabActive()
    loadNotifierSubtab()
}

function loadNotifierSubtab() {
    setNotifierPanel('<div class="loading-state">Loading…</div>')
    if (notifierSubtab === 'forwards') loadForwards()
    else loadBoardSettings()
}

function setNotifierPanel(html) {
    const el = document.getElementById('notifier-panel')
    if (el) el.innerHTML = html
}

// ─────────────────────────────────────────────────────────
// Board Settings (notifier rules: board → recipient routing)
// ─────────────────────────────────────────────────────────
async function loadBoardSettings() {
    try {
        const rules = await api('GET', '/notifiers/rules')
        renderBoardSettings(rules || [])
    } catch (e) {
        setNotifierPanel(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderBoardSettings(rules) {
    const header = `<div class="panel-header">
        <span class="panel-desc">Notify a Webex recipient when a ticket lands on a board.</span>
        <button class="btn btn-primary btn-sm" onclick="showNewRuleModal()">+ New Board Setting</button>
    </div>`

    const thead = '<th>Enabled</th><th>Board</th><th>Recipient</th><th></th>'
    const rows  = rules.map(r => `<tr>
        <td>${badge(r.enabled)}</td>
        <td>${esc(r.board_name)}</td>
        <td>${esc(r.recipient_name)} <span style="color:var(--muted);font-size:11px">${esc(r.recipient_type)}</span></td>
        <td class="actions"><button class="btn btn-danger" onclick="deleteRule(${r.id})">Delete</button></td>
    </tr>`)

    setNotifierPanel(header + tableWrap(thead, rows))
}

async function showNewRuleModal() {
    let boards, recipients
    try {
        [boards, recipients] = await Promise.all([
            api('GET', '/cw/boards'),
            api('GET', '/webex/rooms'),
        ])
    } catch (e) { toast(e.message, 'error'); return }

    if (!boards?.length)     { toast('No boards found — run a sync first', 'error'); return }
    if (!recipients?.length) { toast('No recipients found — run a sync first', 'error'); return }

    const boardOpts = boards.map(b =>
        `<option value="${b.id}">${esc(b.name)}</option>`).join('')
    const recipOpts = recipients.map(r =>
        `<option value="${r.id}">${esc(r.name)} (${esc(r.type)})</option>`).join('')

    openModal('New Board Setting', `
        <div class="form-group">
            <label>Connectwise Board</label>
            <select id="f-board">${boardOpts}</select>
        </div>
        <div class="form-group">
            <label>Webex Recipient</label>
            <select id="f-recipient">${recipOpts}</select>
        </div>`, async () => {
        const boardId = parseInt(document.getElementById('f-board').value)
        const recipId = parseInt(document.getElementById('f-recipient').value)
        try {
            await api('POST', '/notifiers/rules', {
                cw_board_id:    boardId,
                webex_room_id:  recipId,
                notify_enabled: true,
            })
            closeModal()
            toast('Board setting created', 'success')
            loadBoardSettings()
        } catch (e) { toast(e.message, 'error') }
    })
}

async function deleteRule(id) {
    if (!confirm('Delete this board setting?')) return
    try {
        await api('DELETE', `/notifiers/rules/${id}`)
        toast('Board setting deleted', 'success')
        loadBoardSettings()
    } catch (e) { toast(e.message, 'error') }
}

// ─────────────────────────────────────────────────────────
// Workflows
// ─────────────────────────────────────────────────────────
const WORKFLOW_ACTIONS = [
    { value: 'ticket_update',      label: 'Update Ticket' },
    { value: 'add_note',           label: 'Add Note' },
    { value: 'send_message',       label: 'Send Notification' },
    { value: 'skip_notifications', label: 'Skip Default Notifications' },
    { value: 'add_resource',       label: 'Add Resource' },
    { value: 'add_email_cc',       label: 'Add Email CC' },
]
const WORKFLOW_ON = [
    { value: 'new',     label: 'New tickets only' },
    { value: 'both',    label: 'New & Updated' },
    { value: 'updated', label: 'Updates only' },
]
const WORKFLOW_FIELDS = [
    { value: 'summary',            label: 'Summary' },
    { value: 'company_name',       label: 'Company Name' },
    { value: 'company_identifier', label: 'Company ID' },
    { value: 'contact_name',       label: 'Contact Name' },
    { value: 'status_name',        label: 'Status' },
    { value: 'board_name',         label: 'Board Name' },
    { value: 'type_name',          label: 'Type' },
    { value: 'subtype_name',       label: 'Subtype' },
    { value: 'priority_name',      label: 'Priority' },
    { value: 'source_name',        label: 'Source' },
    { value: 'last_note_text',     label: 'Last Note Text' },
    { value: 'last_note_sender',   label: 'Last Note Sender' },
    { value: 'last_note_type',     label: 'Last Note Type' },
]
const WORKFLOW_OPERATORS = [
    { value: 'contains',     label: 'contains' },
    { value: 'not_contains', label: 'does not contain' },
    { value: 'equals',       label: 'equals' },
    { value: 'not_equals',   label: 'does not equal' },
    { value: 'starts_with',  label: 'starts with' },
    { value: 'ends_with',    label: 'ends with' },
    { value: 'is_any_of',    label: 'is any of' },
    { value: 'is_none_of',   label: 'is none of' },
]
// Operators offered for the multi-select Last Note Type field (vs the standard
// text operators offered for every other field).
const NOTE_TYPE_OPERATORS = ['is_any_of', 'is_none_of']
const NOTE_TYPES = ['internal', 'discussion', 'resolution']
function wfActionLabel(v)   { return (WORKFLOW_ACTIONS.find(a => a.value === v)   || {}).label || v }
function wfOnLabel(v)       { return (WORKFLOW_ON.find(a => a.value === v)        || {}).label || v }
function wfFieldLabel(v)    { return (WORKFLOW_FIELDS.find(a => a.value === v)    || {}).label || v }
function wfOperatorLabel(v) { return (WORKFLOW_OPERATORS.find(a => a.value === v) || {}).label || v }

// wfConditionSummary renders the stored ConditionGroup tree as a compact string,
// e.g. (Summary contains "x" AND Status equals "y") OR Company ID equals "z".
function wfConditionSummary(root) {
    if (!root || !root.children || !root.children.length) return '<span style="color:var(--muted)">—</span>'
    return wfSummarizeGroup(root, true)
}
function wfSummarizeGroup(g, top) {
    const parts = (g.children || []).map(n => {
        if (n.group) return '(' + wfSummarizeGroup(n.group, false) + ')'
        if (n.condition) {
            const c = n.condition
            return `${esc(wfFieldLabel(c.field))} <em>${esc(wfOperatorLabel(c.operator))}</em> "${esc(c.value)}"`
        }
        return ''
    }).filter(Boolean)
    if (!parts.length) return ''
    const joiner = (g.operator === 'or') ? ' <b>OR</b> ' : ' <b>AND</b> '
    return parts.join(joiner)
}

let workflows = []
let wfBoardsById = {}

async function loadWorkflows() {
    try {
        const [wfs, boards] = await Promise.all([
            api('GET', '/workflows'),
            api('GET', '/cw/boards'),
        ])
        workflows = wfs || []
        wfBoardsById = {}
        ;(boards || []).forEach(b => { wfBoardsById[b.id] = b.name })
        renderWorkflows(workflows)
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderWorkflows(wfs) {
    const header = `<div class="tab-header">
        <h2>Workflows</h2>
        <button class="btn btn-primary btn-sm" onclick="showWorkflowModal()">+ New Workflow</button>
    </div>`

    const thead = '<th>Enabled</th><th>Priority</th><th>Name</th><th>Board</th><th>On</th><th>Actions</th><th>Conditions</th><th></th>'
    const rows  = wfs.map(w => `<tr>
        <td>${badge(w.enabled)}</td>
        <td>${esc(w.priority)}</td>
        <td>${esc(w.name)}</td>
        <td>${esc(wfBoardsById[w.cw_board_id] || `#${w.cw_board_id}`)}</td>
        <td>${esc(wfOnLabel(w.on_ticket_action))}</td>
        <td>${(w.actions || []).map(a => esc(wfActionLabel(a.type))).join(', ') || '<span style="color:var(--muted)">—</span>'}</td>
        <td style="font-size:12px">${wfConditionSummary(w.conditions)}</td>
        <td class="actions">
            <button class="btn btn-ghost btn-sm" onclick="editWorkflow(${w.id})">Edit</button>
            <button class="btn btn-danger" onclick="deleteWorkflow(${w.id})">Delete</button>
        </td>
    </tr>`)

    setContent(header + tableWrap(thead, rows))
}

function editWorkflow(id) {
    const wf = workflows.find(w => w.id === id)
    if (!wf) { toast('Workflow not found', 'error'); return }
    showWorkflowModal(wf)
}

// State for the workflow builder modal, populated when it opens. wfCatalog is the
// ticket_update field catalog; wfMembers/wfRooms back local pickers. wfBoardId is
// the required board (scopes status/contact pickers). wfTree is the authoritative
// nested condition tree and wfActions the authoritative ordered action list — the
// DOM is rendered from them, never the other way round.
let wfCatalog = []
let wfMembers = []
let wfRooms   = []
let wfBoardId = ''
let wfTree    = null
let wfActions = []
let wfUid     = 0
const wfNextId = () => 'n' + (++wfUid)

// showWorkflowModal opens the create form, or the edit form when `existing` is a
// workflow object.
async function showWorkflowModal(existing = null) {
    const isEdit = !!existing
    let boards = [], catalog = [], members = [], rooms = []
    try {
        [boards, catalog, members, rooms] = await Promise.all([
            api('GET', '/cw/boards'),
            api('GET', '/workflows/update-fields'),
            api('GET', '/cw/members'),
            api('GET', '/webex/rooms'),
        ])
    } catch (e) { toast(e.message, 'error'); return }

    wfCatalog = catalog || []
    wfMembers = members || []
    wfRooms   = rooms || []
    wfBoardId = (isEdit && existing.cw_board_id) ? String(existing.cw_board_id) : ''
    wfTree    = hydrateTree(isEdit ? existing.conditions : null)
    wfActions = (isEdit && Array.isArray(existing.actions) && existing.actions.length)
        ? existing.actions.map(a => ({ _id: wfNextId(), type: a.type, config: a.config || {} }))
        : [{ _id: wfNextId(), type: 'ticket_update', config: {} }]

    const sel = (cond) => cond ? ' selected' : ''
    const boardOpts = `<option value="">Select a board…</option>` + (boards || []).map(b =>
        `<option value="${b.id}"${sel(isEdit && existing.cw_board_id === b.id)}>${esc(b.name)}</option>`).join('')
    const onOpts = WORKFLOW_ON.map(a =>
        `<option value="${a.value}"${sel(isEdit && existing.on_ticket_action === a.value)}>${esc(a.label)}</option>`).join('')

    const nameVal        = isEdit ? esc(existing.name) : ''
    const priorityVal    = isEdit ? esc(existing.priority) : '100'
    const enabledChecked = (isEdit ? existing.enabled : true) ? 'checked' : ''

    openModal(isEdit ? 'Edit Workflow' : 'New Workflow', `
        <div class="form-group">
            <label>Name</label>
            <input id="wf-name" type="text" placeholder="e.g. Triage urgent printer tickets" value="${nameVal}">
        </div>
        <div class="form-group">
            <label>Board <span style="color:var(--danger)">*</span></label>
            <select id="wf-board">${boardOpts}</select>
        </div>
        <div class="form-group">
            <label>On Ticket Action</label>
            <select id="wf-on">${onOpts}</select>
        </div>
        <div class="form-group">
            <label>Conditions <span style="color:var(--muted);font-weight:400">(optional)</span></label>
            <div id="wf-conditions"></div>
        </div>
        <div class="form-group">
            <label>Actions <span style="color:var(--muted);font-weight:400">(run in order)</span></label>
            <div id="wf-actions"></div>
            <button type="button" class="btn btn-ghost btn-sm" onclick="wfAddAction()">+ Add action</button>
        </div>
        <div class="form-group">
            <label>Priority</label>
            <input id="wf-priority" type="number" value="${priorityVal}" min="0">
        </div>
        <div class="config-row" style="padding:0">
            <div>
                <div class="config-label">Enabled</div>
                <div class="config-desc">Disabled workflows are kept but never run</div>
            </div>
            <label class="toggle">
                <input type="checkbox" id="wf-enabled" ${enabledChecked}>
                <span class="toggle-track"></span>
            </label>
        </div>`, async () => {
        const name     = document.getElementById('wf-name').value.trim()
        const priority = parseInt(document.getElementById('wf-priority').value)
        if (!name)      { toast('Name is required', 'error'); return }
        if (!wfBoardId) { toast('A board is required', 'error'); return }

        harvestAllActions()
        if (!wfActions.length) { toast('Add at least one action', 'error'); return }
        const actions = []
        for (const a of wfActions) {
            const cfg = serializeAction(a)
            if (cfg === null) return // builder already toasted the problem
            actions.push({ type: a.type, config: cfg })
        }

        const payload = {
            name,
            cw_board_id: parseInt(wfBoardId),
            on_ticket_action: document.getElementById('wf-on').value,
            conditions: serializeTree(),
            actions,
            priority: Number.isFinite(priority) ? priority : 100,
            enabled: document.getElementById('wf-enabled').checked,
        }

        try {
            if (isEdit) await api('PUT', `/workflows/${existing.id}`, payload)
            else        await api('POST', '/workflows', payload)
            closeModal()
            toast(isEdit ? 'Workflow updated' : 'Workflow created', 'success')
            loadWorkflows()
        } catch (e) { toast(e.message, 'error') }
    }, isEdit ? 'Save' : 'Create')

    const boardSel = document.getElementById('wf-board')
    boardSel.value = wfBoardId
    boardSel.addEventListener('change', () => { wfBoardId = boardSel.value; onBoardChanged() })

    renderTree()
    renderActions()
}

// onBoardChanged re-renders the action cards so status/contact pickers re-scope to
// the new board. Harvest first so unsaved edits survive the re-render.
function onBoardChanged() {
    harvestAllActions()
    renderActions()
}

// ── Condition tree (authoritative state: wfTree) ──────────
function newLeafNode(field = 'summary', operator = 'contains', value = '') {
    return { _id: wfNextId(), kind: 'cond', field, operator, value }
}
function newGroupNode(op = 'and', children = []) {
    return { _id: wfNextId(), kind: 'group', op, children }
}
function hydrateTree(root) {
    if (!root || !root.children) return newGroupNode('and', [])
    return hydrateGroup(root)
}
function hydrateGroup(g) {
    return newGroupNode(g.operator || 'and', (g.children || []).map(hydrateChild))
}
function hydrateChild(n) {
    if (n.group) return hydrateGroup(n.group)
    const c = n.condition || {}
    return newLeafNode(c.field || 'summary', c.operator || 'contains', c.value || '')
}
function findNode(id, node = wfTree, parent = null) {
    if (!node) return null
    if (node._id === id) return { node, parent }
    if (node.kind === 'group') {
        for (const c of node.children) {
            const r = findNode(id, c, node)
            if (r) return r
        }
    }
    return null
}
function renderTree() {
    const host = document.getElementById('wf-conditions')
    if (host) host.innerHTML = renderGroup(wfTree, true)
}
function renderGroup(node, isRoot) {
    const kids = node.children.map(c => c.kind === 'group' ? renderGroup(c, false) : renderLeaf(c)).join('')
    return `<div class="cond-group" data-id="${node._id}">
        <div class="cond-group-head">
            ${andOrToggle(node)}
            <button type="button" class="btn btn-ghost btn-sm" onclick="addLeaf('${node._id}')">+ Condition</button>
            <button type="button" class="btn btn-ghost btn-sm" onclick="addGroup('${node._id}')">+ Group</button>
            ${isRoot ? '' : `<button type="button" class="btn btn-danger btn-sm" onclick="removeNode('${node._id}')">✕</button>`}
        </div>
        <div class="cond-group-body">${kids || '<div class="cond-empty">No conditions — this workflow always runs.</div>'}</div>
    </div>`
}
function andOrToggle(node) {
    const mk = (op, label) => `<button type="button" class="${node.op === op ? 'active' : ''}" onclick="setGroupOp('${node._id}','${op}')">${label}</button>`
    return `<div class="andor-toggle">${mk('and', 'AND')}${mk('or', 'OR')}</div>`
}
function renderLeaf(node) {
    const isNoteType = node.field === 'last_note_type'
    const fieldOpts  = WORKFLOW_FIELDS.map(f => `<option value="${f.value}"${f.value === node.field ? ' selected' : ''}>${esc(f.label)}</option>`).join('')
    const ops        = WORKFLOW_OPERATORS.filter(o => isNoteType ? NOTE_TYPE_OPERATORS.includes(o.value) : !NOTE_TYPE_OPERATORS.includes(o.value))
    const opOpts     = ops.map(o => `<option value="${o.value}"${o.value === node.operator ? ' selected' : ''}>${esc(o.label)}</option>`).join('')

    let valueControl
    if (isNoteType) {
        const sel = new Set((node.value || '').split(',').map(s => s.trim()))
        valueControl = `<div class="nt-checks">${NOTE_TYPES.map(t =>
            `<label><input type="checkbox" class="nt-check" value="${t}" ${sel.has(t) ? 'checked' : ''} onchange="setNoteTypes('${node._id}', this)"> ${t[0].toUpperCase() + t.slice(1)}</label>`
        ).join('')}</div>`
    } else {
        valueControl = `<input class="cond-value" type="text" placeholder="value" style="flex:1" value="${esc(node.value)}" oninput="setLeaf('${node._id}','value',this.value)">`
    }

    return `<div class="cond-leaf" data-id="${node._id}">
        <select class="cond-field" style="flex:0 0 32%" onchange="setLeafField('${node._id}', this.value)">${fieldOpts}</select>
        <select class="cond-op" style="flex:0 0 28%" onchange="setLeaf('${node._id}','operator',this.value)">${opOpts}</select>
        ${valueControl}
        <button type="button" class="btn btn-danger btn-sm" onclick="removeNode('${node._id}')">✕</button>
    </div>`
}
function addLeaf(groupId)  { const r = findNode(groupId); if (r) { r.node.children.push(newLeafNode()); renderTree() } }
function addGroup(groupId) { const r = findNode(groupId); if (r) { r.node.children.push(newGroupNode('and', [newLeafNode()])); renderTree() } }
function removeNode(id)    { const r = findNode(id); if (r && r.parent) { r.parent.children = r.parent.children.filter(c => c._id !== id); renderTree() } }
function setGroupOp(id, op){ const r = findNode(id); if (r) { r.node.op = op; renderTree() } }
// setLeaf updates the model in place WITHOUT re-rendering, so the focused input
// keeps focus while typing.
function setLeaf(id, key, val) { const r = findNode(id); if (r) r.node[key] = val }

// setLeafField changes a leaf's field and re-renders, since switching to/from the
// Last Note Type field swaps the operator set and value control. It resets the
// operator and value so the leaf stays valid for the new field.
function setLeafField(id, value) {
    const r = findNode(id)
    if (!r) return
    const wasNote = r.node.field === 'last_note_type'
    const isNote  = value === 'last_note_type'
    r.node.field = value
    if (isNote && !wasNote)      { r.node.operator = 'is_any_of'; r.node.value = '' }
    else if (!isNote && wasNote) { r.node.operator = 'contains';  r.node.value = '' }
    renderTree()
}

// setNoteTypes gathers the checked Last Note Type boxes for a leaf into a
// comma-separated value.
function setNoteTypes(id, el) {
    const r = findNode(id)
    if (!r) return
    const leaf = el.closest('.cond-leaf')
    r.node.value = [...leaf.querySelectorAll('.nt-check:checked')].map(c => c.value).join(',')
}

function serializeTree() {
    if (!wfTree) return null
    const g = serializeGroup(wfTree)
    return (g && g.children.length) ? g : null
}
function serializeGroup(node) {
    return { operator: node.op, children: node.children.map(serializeChild).filter(Boolean) }
}
function serializeChild(node) {
    if (node.kind === 'group') {
        const g = serializeGroup(node)
        return g.children.length ? { group: g } : null
    }
    if (!node.value.trim()) return null
    return { condition: { field: node.field, operator: node.operator, value: node.value } }
}

// ── Action list (authoritative state: wfActions) ──────────
// renderActions rebuilds the action cards from wfActions, then renders each card's
// type-specific body (and attaches its comboboxes).
function renderActions() {
    const host = document.getElementById('wf-actions')
    if (!host) return
    host.innerHTML = wfActions.map((a, i) => actionCardHTML(a, i)).join('')
    wfActions.forEach(a => {
        const card = host.querySelector(`.action-card[data-id="${a._id}"]`)
        if (card) renderActionBody(a, card.querySelector('.action-card-body'))
    })
}

function actionCardHTML(a, i) {
    const typeOpts = WORKFLOW_ACTIONS.map(o => `<option value="${o.value}"${o.value === a.type ? ' selected' : ''}>${esc(o.label)}</option>`).join('')
    const up = i > 0, down = i < wfActions.length - 1
    return `<div class="action-card" data-id="${a._id}">
        <div class="action-card-head">
            <select class="wf-action-type" onchange="wfChangeActionType('${a._id}', this.value)">${typeOpts}</select>
            <div class="action-card-tools">
                <button type="button" class="btn btn-ghost btn-sm" ${up ? '' : 'disabled'} onclick="wfMoveAction('${a._id}',-1)">↑</button>
                <button type="button" class="btn btn-ghost btn-sm" ${down ? '' : 'disabled'} onclick="wfMoveAction('${a._id}',1)">↓</button>
                <button type="button" class="btn btn-danger btn-sm" onclick="wfRemoveAction('${a._id}')">✕</button>
            </div>
        </div>
        <div class="action-card-body"></div>
    </div>`
}

// renderActionBody draws the type-specific config UI into a card body and attaches
// any comboboxes. Prefill comes from the action's config object.
function renderActionBody(a, bodyEl) {
    if (!bodyEl) return
    if (a.type === 'ticket_update') {
        bodyEl.innerHTML = `
            <label>Operations</label>
            <div class="wf-ops"></div>
            <button type="button" class="btn btn-ghost btn-sm" onclick="wfAddOp(this)">+ Add operation</button>
            <div class="config-desc">Each value supports Go templates, e.g. <code>[{{.Company.Identifier}}] {{.Summary}}</code></div>`
        const opsWrap = bodyEl.querySelector('.wf-ops')
        const ops = (a.config && Array.isArray(a.config.ops)) ? a.config.ops : []
        if (ops.length) ops.forEach(o => wfAddOpRow(opsWrap, o.path, o.op, o.value || '', o.label || ''))
        else wfAddOpRow(opsWrap)
        wfRefreshDependents(opsWrap)
    } else if (a.type === 'add_note') {
        const c = a.config || {}
        const flag = (on) => on ? 'checked' : ''
        bodyEl.innerHTML = `
            <label>Note text <span style="color:var(--muted);font-weight:400">(Go templates ok)</span></label>
            <textarea class="wf-note-text code-input" rows="3" placeholder="e.g. Auto-triaged for {{.Company.Name}}">${esc(c.text || '')}</textarea>
            <div style="display:flex;gap:16px;margin-top:8px;flex-wrap:wrap">
                <label style="display:inline-flex;gap:6px;align-items:center;font-weight:400"><input type="checkbox" class="wf-note-detail" ${flag(c.detail_description)}> Discussion</label>
                <label style="display:inline-flex;gap:6px;align-items:center;font-weight:400"><input type="checkbox" class="wf-note-internal" ${flag(c.internal)}> Internal</label>
                <label style="display:inline-flex;gap:6px;align-items:center;font-weight:400"><input type="checkbox" class="wf-note-resolution" ${flag(c.resolution)}> Resolution</label>
            </div>`
    } else if (a.type === 'send_message') {
        const c = a.config || {}
        const useCard = c.use_ticket_card !== false // default on
        bodyEl.innerHTML = `
            <label>Recipient</label>
            <div class="combobox wf-recip">
                <input type="hidden" class="po-value">
                <input type="hidden" class="po-label">
                <input type="text" class="cb-input" placeholder="Search rooms & people…" autocomplete="off" spellcheck="false">
                <div class="cb-menu hidden"></div>
            </div>
            <div class="wf-opt-row">
                <div>
                    <div class="config-label">Use ticket card</div>
                    <div class="config-desc">Send the standard ticket notification</div>
                </div>
                <label class="toggle">
                    <input type="checkbox" class="wf-send-card" ${useCard ? 'checked' : ''} onchange="wfToggleSendCard(this)">
                    <span class="toggle-track"></span>
                </label>
            </div>
            <div class="wf-send-custom form-group" style="${useCard ? 'display:none' : ''};margin-top:10px">
                <label>Message <span style="color:var(--muted);font-weight:400">(Go templates ok)</span></label>
                <textarea class="wf-send-text code-input" rows="3" placeholder="e.g. Heads up on {{.Summary}}">${esc(c.text || '')}</textarea>
            </div>
            <div class="wf-opt-row">
                <div>
                    <div class="config-label">Skip further notifications</div>
                    <div class="config-desc">Suppress the normal notifier for this ticket once sent</div>
                </div>
                <label class="toggle">
                    <input type="checkbox" class="wf-send-skip" ${c.skip_further_notifications ? 'checked' : ''}>
                    <span class="toggle-track"></span>
                </label>
            </div>`
        const recipOpts = wfRooms.map(r => ({ label: r.name, value: String(r.id), hint: r.type }))
        attachCombobox(bodyEl.querySelector('.wf-recip'), { options: recipOpts, initial: c.recipient_id ? String(c.recipient_id) : '' })
    } else if (a.type === 'skip_notifications') {
        bodyEl.innerHTML = `<div class="config-desc">Marks this ticket so the default notifier does not run. Send Notification actions in this or other workflows still fire.</div>`
    } else if (a.type === 'add_resource') {
        const c = a.config || {}
        bodyEl.innerHTML = `
            <label>Member</label>
            <div class="combobox wf-member">
                <input type="hidden" class="po-value">
                <input type="hidden" class="po-label">
                <input type="text" class="cb-input" placeholder="Search members…" autocomplete="off" spellcheck="false">
                <div class="cb-menu hidden"></div>
            </div>`
        attachCombobox(bodyEl.querySelector('.wf-member'), { options: wfPickerOptions('member'), initial: c.member_identifier || '' })
    } else if (a.type === 'add_email_cc') {
        const c = a.config || {}
        bodyEl.innerHTML = `
            <label>Email Address <span style="color:var(--muted);font-weight:400">(Go templates ok)</span></label>
            <input class="wf-cc-email" type="text" style="width:100%" placeholder="e.g. alerts@example.com" value="${esc(c.email || '')}">`
    }
}

function wfToggleSendCard(cb) {
    const custom = cb.closest('.action-card-body').querySelector('.wf-send-custom')
    if (custom) custom.style.display = cb.checked ? 'none' : ''
}

// harvestAllActions reads each card's live DOM back into its action config, so
// edits survive a re-render (reorder / type change / remove / board change).
function harvestAllActions() {
    const host = document.getElementById('wf-actions')
    if (!host) return
    wfActions.forEach(a => {
        const card = host.querySelector(`.action-card[data-id="${a._id}"]`)
        if (card) harvestActionBody(a, card.querySelector('.action-card-body'))
    })
}

function harvestActionBody(a, bodyEl) {
    if (!bodyEl) return
    if (a.type === 'ticket_update') {
        const ops = [...bodyEl.querySelectorAll('.wf-ops .patch-op')].map(row => {
            const path  = row.querySelector('.po-field').value
            const op    = row.querySelector('.po-op').value
            const vEl   = row.querySelector('.po-value')
            const lEl   = row.querySelector('.po-label')
            const value = op === 'remove' ? '' : (vEl ? vEl.value : '')
            const o = { op, path, value }
            if (op !== 'remove' && lEl && lEl.value) o.label = lEl.value
            return o
        })
        a.config = { ops }
    } else if (a.type === 'add_note') {
        a.config = {
            text:               bodyEl.querySelector('.wf-note-text')?.value || '',
            detail_description: !!bodyEl.querySelector('.wf-note-detail')?.checked,
            internal:           !!bodyEl.querySelector('.wf-note-internal')?.checked,
            resolution:         !!bodyEl.querySelector('.wf-note-resolution')?.checked,
        }
    } else if (a.type === 'send_message') {
        const recip = bodyEl.querySelector('.wf-recip .po-value')?.value || ''
        a.config = {
            recipient_id:               recip ? parseInt(recip) : 0,
            use_ticket_card:            !!bodyEl.querySelector('.wf-send-card')?.checked,
            text:                       bodyEl.querySelector('.wf-send-text')?.value || '',
            skip_further_notifications: !!bodyEl.querySelector('.wf-send-skip')?.checked,
        }
    } else if (a.type === 'skip_notifications') {
        a.config = {}
    } else if (a.type === 'add_resource') {
        a.config = { member_identifier: bodyEl.querySelector('.wf-member .po-value')?.value || '' }
    } else if (a.type === 'add_email_cc') {
        a.config = { email: bodyEl.querySelector('.wf-cc-email')?.value || '' }
    }
}

// serializeAction validates a harvested action and returns the config object the
// API expects, or null (after toasting) when it's incomplete.
function serializeAction(a) {
    if (a.type === 'ticket_update') {
        const ops = (a.config.ops || []).filter(o => o.op === 'remove' || (o.value || '').trim() !== '')
        if (!ops.length) { toast('A Ticket Update action needs at least one operation', 'error'); return null }
        const has = (p) => ops.some(o => o.path === p)
        if (has('contact') && !has('company')) { toast('Contact requires a Company operation in the same action', 'error'); return null }
        if (has('board') && !has('status'))    { toast('Changing the Board also requires a Status operation', 'error'); return null }
        return { ops }
    }
    if (a.type === 'add_note') {
        if (!(a.config.text || '').trim()) { toast('A Note action needs note text', 'error'); return null }
        return {
            text:               a.config.text,
            detail_description: !!a.config.detail_description,
            internal:           !!a.config.internal,
            resolution:         !!a.config.resolution,
        }
    }
    if (a.type === 'send_message') {
        if (!a.config.recipient_id) { toast('A Send Notification action needs a recipient', 'error'); return null }
        if (!a.config.use_ticket_card && !(a.config.text || '').trim()) { toast('Add message text or enable “Use ticket card”', 'error'); return null }
        return {
            recipient_id:               a.config.recipient_id,
            use_ticket_card:            !!a.config.use_ticket_card,
            text:                       a.config.text || '',
            skip_further_notifications: !!a.config.skip_further_notifications,
        }
    }
    if (a.type === 'skip_notifications') {
        return {}
    }
    if (a.type === 'add_resource') {
        if (!(a.config.member_identifier || '').trim()) { toast('An Add Resource action needs a member', 'error'); return null }
        return { member_identifier: a.config.member_identifier }
    }
    if (a.type === 'add_email_cc') {
        if (!(a.config.email || '').trim()) { toast('An Add Email CC action needs an email address', 'error'); return null }
        return { email: a.config.email.trim() }
    }
    return {}
}

function wfAddAction()        { harvestAllActions(); wfActions.push({ _id: wfNextId(), type: 'ticket_update', config: {} }); renderActions() }
function wfRemoveAction(id)   { harvestAllActions(); wfActions = wfActions.filter(a => a._id !== id); renderActions() }
function wfChangeActionType(id, type) {
    harvestAllActions()
    const a = wfActions.find(x => x._id === id)
    if (a) { a.type = type; a.config = {} }
    renderActions()
}
function wfMoveAction(id, dir) {
    harvestAllActions()
    const i = wfActions.findIndex(a => a._id === id)
    const j = i + dir
    if (i < 0 || j < 0 || j >= wfActions.length) return
    ;[wfActions[i], wfActions[j]] = [wfActions[j], wfActions[i]]
    renderActions()
}

// ── ticket_update op builder (card-scoped) ────────────────
function wfFieldByPath(path) { return wfCatalog.find(f => f.path === path) }

// wfPickerOptions maps local picker rows to {label, value, hint}. Board options
// come from the boards loaded for the table; member (owner) from the modal fetch.
// Company/contact/status are not here — they search ConnectWise live.
function wfPickerOptions(picker) {
    switch (picker) {
        case 'member': return wfMembers.map(m => ({ label: `${(m.first_name || '')} ${(m.last_name || '')}`.trim() || m.identifier, value: m.identifier, hint: m.identifier }))
        case 'board':  return Object.entries(wfBoardsById).map(([id, name]) => ({ label: name, value: String(id) }))
        default:       return []
    }
}

// wfOpValue returns the value held by an op row for a field path within one ops
// container (or '' if absent) — used to scope dependent pickers.
function wfOpValue(opsWrap, path) {
    for (const row of opsWrap.querySelectorAll('.patch-op')) {
        if (row.querySelector('.po-field').value !== path) continue
        const v = row.querySelector('.po-value')
        return v ? v.value : ''
    }
    return ''
}

function wfAddOp(btn) {
    const opsWrap = btn.closest('.action-card-body').querySelector('.wf-ops')
    if (opsWrap) wfAddOpRow(opsWrap)
}

// wfAddOpRow appends an op row (field select + op select + value control) to a card's ops container.
function wfAddOpRow(opsWrap, field = 'summary', op = 'replace', value = '', label = '') {
    const fieldOpts = wfCatalog.map(f => `<option value="${esc(f.path)}">${esc(f.label)}</option>`).join('')
    const row = document.createElement('div')
    row.className = 'patch-op'
    row.innerHTML = `
        <div style="display:flex;gap:6px;align-items:center;margin-bottom:6px">
            <select class="po-field" style="flex:1">${fieldOpts}</select>
            <select class="po-op" style="flex:0 0 100px"></select>
            <button type="button" class="btn btn-danger btn-sm" onclick="wfRemoveOp(this)">✕</button>
        </div>
        <div class="po-value-cell" data-initial="${esc(value)}" data-initial-label="${esc(label)}"></div>`
    opsWrap.appendChild(row)

    const fieldSel = row.querySelector('.po-field')
    if (wfFieldByPath(field)) fieldSel.value = field
    wfRenderOpOptions(row, op)
    wfRenderValueCell(row)
    fieldSel.addEventListener('change', () => {
        wfRenderOpOptions(row)
        wfRenderValueCell(row, { reset: true })
        if (fieldSel.value === 'board' && !wfHasOp(opsWrap, 'status')) wfAddOpRow(opsWrap, 'status')
        wfRefreshDependents(opsWrap)
    })
    row.querySelector('.po-op').addEventListener('change', () => { wfRenderValueCell(row); wfRefreshDependents(opsWrap) })
}

function wfRemoveOp(btn) {
    const opsWrap = btn.closest('.wf-ops')
    btn.closest('.patch-op').remove()
    if (opsWrap) wfRefreshDependents(opsWrap)
}

function wfHasOp(opsWrap, path) {
    return [...opsWrap.querySelectorAll('.po-field')].some(s => s.value === path)
}

function wfRefreshDependents(opsWrap) {
    opsWrap.querySelectorAll('.patch-op').forEach(row => {
        const f = wfFieldByPath(row.querySelector('.po-field').value)
        if (f && f.depends_on) wfRenderValueCell(row)
    })
}

function wfRenderOpOptions(row, selected) {
    const field = wfFieldByPath(row.querySelector('.po-field').value)
    const opSel = row.querySelector('.po-op')
    const cur = selected || opSel.value || 'replace'
    const opts = [{ value: 'replace', label: 'replace' }]
    if (field && field.allow_remove) opts.push({ value: 'remove', label: 'remove' })
    opSel.innerHTML = opts.map(o => `<option value="${o.value}">${o.label}</option>`).join('')
    opSel.value = opts.some(o => o.value === cur) ? cur : 'replace'
}

// wfOnParentChanged clears dependent rows whose parent (company/board) changed,
// then re-scopes their pickers.
function wfOnParentChanged(opsWrap, parentPath) {
    opsWrap.querySelectorAll('.patch-op').forEach(row => {
        const f = wfFieldByPath(row.querySelector('.po-field').value)
        if (f && f.depends_on === parentPath) {
            const cell = row.querySelector('.po-value-cell')
            cell.dataset.initial = ''
            cell.dataset.initialLabel = ''
            const v = row.querySelector('.po-value'); if (v) v.value = ''
            const l = row.querySelector('.po-label'); if (l) l.value = ''
        }
    })
    wfRefreshDependents(opsWrap)
}

// wfRenderValueCell (re)draws a row's value control: nothing for remove; a
// search-as-you-type combobox for picker-backed fields; otherwise a plain text
// input that accepts Go templates. Contact is scoped to the row's Company op and
// status to the row's Board op or the workflow's board.
function wfRenderValueCell(row, opts = {}) {
    const field = wfFieldByPath(row.querySelector('.po-field').value)
    const op    = row.querySelector('.po-op').value
    const cell  = row.querySelector('.po-value-cell')
    const existingV = row.querySelector('.po-value')
    const existingL = row.querySelector('.po-label')
    const prev      = opts.reset ? '' : (existingV ? existingV.value : (cell.dataset.initial || ''))
    const prevLabel = opts.reset ? '' : (existingL ? existingL.value : (cell.dataset.initialLabel || ''))

    if (op === 'remove' || !field) { cell.innerHTML = ''; return }

    const opsWrap = row.closest('.wf-ops')

    if (field.picker) {
        let fetcher = null, disabledMsg = ''
        if (field.picker === 'company') {
            fetcher = q => api('GET', `/cw/companies?q=${encodeURIComponent(q)}`)
        } else if (field.picker === 'contact') {
            const company = wfOpValue(opsWrap, 'company')
            if (!company) disabledMsg = 'Add & choose a Company operation first'
            else fetcher = q => api('GET', `/cw/contacts?company=${encodeURIComponent(company)}&q=${encodeURIComponent(q)}`)
        } else if (field.picker === 'status') {
            const board = wfOpValue(opsWrap, 'board') || wfBoardId
            if (!board) disabledMsg = 'Select the workflow board first'
            else fetcher = q => api('GET', `/cw/boards/${board}/statuses?q=${encodeURIComponent(q)}`)
        }

        if (disabledMsg) {
            cell.innerHTML = `<input type="hidden" class="po-value">
                <div class="cb-disabled-msg">${esc(disabledMsg)}</div>`
            return
        }

        cell.innerHTML = `<div class="combobox">
            <input type="hidden" class="po-value">
            <input type="hidden" class="po-label">
            <input type="text" class="cb-input" placeholder="Search ${esc(field.picker)}…" autocomplete="off" spellcheck="false">
            <div class="cb-menu hidden"></div>
        </div>`
        const isParent = wfCatalog.some(f => f.depends_on === field.path)
        const onChange = isParent ? (() => wfOnParentChanged(opsWrap, field.path)) : null
        const cfg = { initial: prev, initialLabel: prevLabel, onChange }
        if (fetcher) cfg.fetch = fetcher
        else cfg.options = wfPickerOptions(field.picker)
        attachCombobox(cell.querySelector('.combobox'), cfg)
        return
    }

    const ph = field.kind === 'text' ? 'value (Go template ok)' : 'name (Go template ok)'
    cell.innerHTML = `<input class="po-value" type="text" placeholder="${ph}" style="width:100%" value="${esc(prev)}">`
}

// attachCombobox wires a search-as-you-type picker onto a .combobox wrapper.
// cfg: { options | fetch, initial, initialLabel, onChange }.
//   - options: [{label,value,hint?}] filtered client-side (local pickers).
//   - fetch:   async q => [{label,value,hint?}], debounced (live CW pickers).
// The visible .cb-input shows only `label`; hidden .po-value holds the selected
// `value` (what the op stores) and hidden .po-label keeps the display label so
// edits prefill without a lookup. Typing filters; click / Enter selects; ↑/↓
// move; Esc/blur close. Text that isn't a selection is discarded on blur.
function attachCombobox(combo, cfg = {}) {
    const { options = null, fetch = null, onChange = null } = cfg
    const hidden = combo.querySelector('.po-value')
    const label  = combo.querySelector('.po-label')
    const input  = combo.querySelector('.cb-input')
    const menu   = combo.querySelector('.cb-menu')
    let filtered = [], active = -1, selectedLabel = '', seq = 0, debounce = null

    const staticLabelFor = (v) => { const o = (options || []).find(o => o.value === v); return o ? o.label : '' }

    hidden.value  = cfg.initial || ''
    selectedLabel = cfg.initialLabel || (options ? staticLabelFor(hidden.value) : '') || hidden.value || ''
    if (label) label.value = hidden.value ? selectedLabel : ''
    input.value = hidden.value ? selectedLabel : ''

    function render(list) {
        filtered = list; active = -1
        menu.innerHTML = list.length
            ? list.map((o, i) => `<div class="cb-item" data-i="${i}">
                    <span class="cb-item-label">${esc(o.label)}</span>
                    ${o.hint ? `<span class="cb-item-hint">${esc(o.hint)}</span>` : ''}
                </div>`).join('')
            : `<div class="cb-empty">No matches</div>`
        menu.classList.remove('hidden')
    }
    function close() { menu.classList.add('hidden'); active = -1 }
    function choose(o) {
        if (!o) return
        hidden.value = o.value; selectedLabel = o.label
        if (label) label.value = o.label
        input.value = o.label; close()
        if (onChange) onChange()
    }
    function setActive(i) {
        const items = [...menu.querySelectorAll('.cb-item')]
        items.forEach(el => el.classList.remove('active'))
        if (i >= 0 && i < items.length) { items[i].classList.add('active'); items[i].scrollIntoView({ block: 'nearest' }) }
        active = i
    }
    function load(q) {
        if (options) {
            const s = (q || '').trim().toLowerCase()
            render(!s ? options.slice(0, 100)
                      : options.filter(o => o.label.toLowerCase().includes(s) || (o.hint || '').toLowerCase().includes(s)).slice(0, 100))
            return
        }
        const my = ++seq
        menu.innerHTML = `<div class="cb-empty">Searching…</div>`; menu.classList.remove('hidden')
        Promise.resolve(fetch(q || ''))
            .then(res => { if (my === seq) render(res || []) })
            .catch(() => { if (my === seq) menu.innerHTML = `<div class="cb-empty">Search failed</div>` })
    }
    function loadDebounced(q) {
        if (options) { load(q); return }
        clearTimeout(debounce); debounce = setTimeout(() => load(q), 250)
    }

    input.addEventListener('focus', () => load(input.value === selectedLabel ? '' : input.value))
    input.addEventListener('input', () => { hidden.value = ''; if (label) label.value = ''; loadDebounced(input.value) })
    input.addEventListener('keydown', (e) => {
        if (menu.classList.contains('hidden')) { if (e.key === 'ArrowDown') load(input.value); return }
        if (e.key === 'ArrowDown')      { e.preventDefault(); setActive(Math.min(active + 1, filtered.length - 1)) }
        else if (e.key === 'ArrowUp')   { e.preventDefault(); setActive(Math.max(active - 1, 0)) }
        else if (e.key === 'Enter')     { e.preventDefault(); if (active >= 0) choose(filtered[active]) }
        else if (e.key === 'Escape')    { close() }
    })
    // mousedown (not click) so selection registers before the input's blur fires.
    menu.addEventListener('mousedown', (e) => {
        const item = e.target.closest('.cb-item'); if (!item) return
        e.preventDefault()
        choose(filtered[parseInt(item.dataset.i, 10)])
    })
    input.addEventListener('blur', () => setTimeout(() => {
        close()
        input.value = hidden.value ? selectedLabel : '' // discard stray typed text
    }, 150))
}

async function deleteWorkflow(id) {
    if (!confirm('Delete this workflow?')) return
    try {
        await api('DELETE', `/workflows/${id}`)
        toast('Workflow deleted', 'success')
        loadWorkflows()
    } catch (e) { toast(e.message, 'error') }
}

// ─────────────────────────────────────────────────────────
// Forwards
// ─────────────────────────────────────────────────────────
async function loadForwards() {
    try {
        const fwds = await api('GET', '/notifiers/forwards?filter=not-expired')
        renderForwards(fwds || [])
    } catch (e) {
        setNotifierPanel(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderForwards(fwds) {
    const header = `<div class="panel-header">
        <span class="panel-desc">Redirect a recipient's notifications to another for a time window.</span>
        <button class="btn btn-primary btn-sm" onclick="showNewForwardModal()">+ New Forward</button>
    </div>`

    const thead = '<th>Enabled</th><th>Keep Copy</th><th>Dates</th><th>Source</th><th>Destination</th><th></th>'
    const rows  = fwds.map(f => `<tr>
        <td>${badge(f.enabled)}</td>
        <td>${badge(f.user_keeps_copy)}</td>
        <td style="white-space:nowrap;color:var(--muted)">${fmtDateRange(f.start_date, f.end_date)}</td>
        <td>${esc(f.source_name)} <span style="color:var(--muted);font-size:11px">${esc(f.source_type)}</span></td>
        <td>${esc(f.destination_name)} <span style="color:var(--muted);font-size:11px">${esc(f.destination_type)}</span></td>
        <td class="actions"><button class="btn btn-danger" onclick="deleteForward(${f.id})">Delete</button></td>
    </tr>`)

    setNotifierPanel(header + tableWrap(thead, rows))
}

async function showNewForwardModal() {
    let recipients
    try {
        recipients = await api('GET', '/webex/rooms')
    } catch (e) { toast(e.message, 'error'); return }

    if (!recipients?.length) { toast('No recipients found — run a sync first', 'error'); return }

    const recipOpts = recipients.map(r =>
        `<option value="${r.id}">${esc(r.name)} (${esc(r.type)})</option>`).join('')

    openModal('New Forward', `
        <div class="form-group">
            <label>Source</label>
            <select id="f-source">${recipOpts}</select>
        </div>
        <div class="form-group">
            <label>Destination</label>
            <select id="f-dest">${recipOpts}</select>
        </div>
        <div class="form-group">
            <label>Start Date &amp; Time <span style="color:var(--muted)">(optional)</span></label>
            <div style="display:flex;gap:8px">
                <input type="date" id="f-start-date" style="flex:2">
                <input type="time" id="f-start-time" style="flex:1">
            </div>
        </div>
        <div class="form-group">
            <label>End Date &amp; Time <span style="color:var(--muted)">(optional)</span></label>
            <div style="display:flex;gap:8px">
                <input type="date" id="f-end-date" style="flex:2">
                <input type="time" id="f-end-time" style="flex:1">
            </div>
        </div>
        <div class="form-group">
            <label>Source Keeps Copy?</label>
            <select id="f-keep">
                <option value="true">Yes</option>
                <option value="false">No</option>
            </select>
        </div>`, async () => {
        const sourceId  = parseInt(document.getElementById('f-source').value)
        const destId    = parseInt(document.getElementById('f-dest').value)
        const startDate = document.getElementById('f-start-date').value
        const startTime = document.getElementById('f-start-time').value
        const endDate   = document.getElementById('f-end-date').value
        const endTime   = document.getElementById('f-end-time').value
        const keepCopy  = document.getElementById('f-keep').value === 'true'

        const startDT = startDate ? `${startDate}T${startTime || '00:00'}` : null
        const endDT   = endDate   ? `${endDate}T${endTime   || '23:59'}` : null

        if (sourceId === destId) { toast('Source and destination must be different', 'error'); return }
        if (startDT && endDT && endDT <= startDT) { toast('End must be after start', 'error'); return }

        const payload = {
            user_email:      sourceId,
            dest_email:      destId,
            enabled:         true,
            user_keeps_copy: keepCopy,
        }
        if (startDT) payload.start_date = new Date(startDT).toISOString()
        if (endDT)   payload.end_date   = new Date(endDT).toISOString()

        try {
            await api('POST', '/notifiers/forwards', payload)
            closeModal()
            toast('Forward created', 'success')
            loadForwards()
        } catch (e) { toast(e.message, 'error') }
    })
}

async function deleteForward(id) {
    if (!confirm('Delete this forward?')) return
    try {
        await api('DELETE', `/notifiers/forwards/${id}`)
        toast('Forward deleted', 'success')
        loadForwards()
    } catch (e) { toast(e.message, 'error') }
}

// ─────────────────────────────────────────────────────────
// Users
// ─────────────────────────────────────────────────────────
async function loadUsers() {
    try {
        const users = await api('GET', '/users')
        renderUsers(users || [])
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderUsers(users) {
    const header = `<div class="tab-header">
        <h2>Users</h2>
        <button class="btn btn-primary btn-sm" onclick="showNewUserModal()">+ New User</button>
    </div>`

    const thead = '<th>ID</th><th>Email</th><th>Created</th><th></th>'
    const rows  = users.map(u => `<tr>
        <td style="color:var(--muted)">${u.id}</td>
        <td>${esc(u.email_address)}</td>
        <td style="color:var(--muted)">${fmtDateTime(u.created_on)}</td>
        <td class="actions"><button class="btn btn-danger" onclick="deleteUser(${u.id})">Delete</button></td>
    </tr>`)

    setContent(header + tableWrap(thead, rows))
}

function showNewUserModal() {
    openModal('New User', `
        <div class="form-group">
            <label>Email Address</label>
            <input type="email" id="f-email" placeholder="user@example.com">
        </div>
        <div class="form-group">
            <label>Temporary Password</label>
            <input type="password" id="f-temp-password" placeholder="User must change on first login">
        </div>`, async () => {
        const email    = document.getElementById('f-email').value.trim()
        const password = document.getElementById('f-temp-password').value
        if (!email)     { toast('Email is required', 'error'); return }
        if (!password)  { toast('Temporary password is required', 'error'); return }
        try {
            await api('POST', '/users', { email_address: email, password })
            closeModal()
            toast('User created — they must change their password on first login', 'success')
            loadUsers()
        } catch (e) { toast(e.message, 'error') }
    })
}

async function deleteUser(id) {
    if (!confirm('Delete this user? Their API keys will also be removed.')) return
    try {
        await api('DELETE', `/users/${id}`)
        toast('User deleted', 'success')
        loadUsers()
    } catch (e) { toast(e.message, 'error') }
}

// ─────────────────────────────────────────────────────────
// API Keys
// ─────────────────────────────────────────────────────────
async function loadKeys() {
    try {
        const [keys, users] = await Promise.all([
            api('GET', '/users/keys'),
            api('GET', '/users'),
        ])
        renderKeys(keys || [], users || [])
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderKeys(keys, users) {
    const userMap = {}
    users.forEach(u => { userMap[u.id] = u.email_address })

    const header = `<div class="tab-header">
        <h2>API Keys</h2>
        <button class="btn btn-primary btn-sm" onclick="showNewKeyModal()">+ New Key</button>
    </div>`

    const thead = '<th>ID</th><th>User</th><th>Hint</th><th>Created</th><th></th>'
    const rows  = keys.map(k => `<tr>
        <td style="color:var(--muted)">${k.id}</td>
        <td>${esc(userMap[k.user_id] || `User #${k.user_id}`)}</td>
        <td style="font-family:monospace;color:var(--muted)">${k.key_hint ? `****${esc(k.key_hint)}` : '—'}</td>
        <td style="color:var(--muted)">${fmtDateTime(k.created_on)}</td>
        <td class="actions"><button class="btn btn-danger" onclick="deleteKey(${k.id})">Delete</button></td>
    </tr>`)

    setContent(header + tableWrap(thead, rows))
}

async function showNewKeyModal() {
    let users
    try {
        users = await api('GET', '/users')
    } catch (e) { toast(e.message, 'error'); return }

    if (!users?.length) { toast('No users found — create a user first', 'error'); return }

    const userOpts = users.map(u =>
        `<option value="${esc(u.email_address)}">${esc(u.email_address)}</option>`).join('')

    openModal('New API Key', `
        <div class="form-group">
            <label>User</label>
            <select id="f-user-email">${userOpts}</select>
        </div>`, async () => {
        const email = document.getElementById('f-user-email').value
        try {
            const res = await api('POST', '/users/keys', { email })
            // Replace modal with key display — key is only shown once
            document.getElementById('modal-body').innerHTML = `
                <p style="color:var(--warning);font-size:13px">
                    ⚠ Copy this key now — it will not be shown again.
                </p>
                <div class="key-display" id="created-key">${esc(res.key)}</div>`
            document.getElementById('modal-footer').innerHTML = `
                <button class="btn btn-ghost" onclick="copyCreatedKey()">Copy to Clipboard</button>
                <button class="btn btn-primary" onclick="closeModal(); loadKeys()">Done</button>`
            modalSubmitFn = null
        } catch (e) { toast(e.message, 'error') }
    })
}

function copyCreatedKey() {
    const key = document.getElementById('created-key')?.textContent
    if (key) navigator.clipboard.writeText(key).then(() => toast('Copied!', 'success'))
}

async function deleteKey(id) {
    if (!confirm('Delete this API key?')) return
    try {
        await api('DELETE', `/users/keys/${id}`)
        toast('Key deleted', 'success')
        loadKeys()
    } catch (e) { toast(e.message, 'error') }
}

// ─────────────────────────────────────────────────────────
// Sync
// ─────────────────────────────────────────────────────────
async function loadSync() {
    try {
        const status = await api('GET', '/sync/status')
        renderSync(status)
        if (status?.status) startSyncPoll()
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderSync(status) {
    const running   = status?.status === true
    const dotClass  = running ? 'running' : 'idle'
    const statusTxt = running ? 'Sync running…' : 'Idle'

    setContent(`<div class="tab-header">
        <h2>Sync</h2>
        <button class="btn btn-primary btn-sm" onclick="showNewSyncModal()" ${running ? 'disabled' : ''}>
            Run Sync
        </button>
    </div>
    <div class="sync-status">
        <div class="status-dot ${dotClass}"></div>
        <span style="color:var(--muted)">${statusTxt}</span>
    </div>
    <p style="color:var(--muted);font-size:13px;max-width:480px">
        Sync pulls the latest boards and Webex recipients from Connectwise and Webex.
        Run this after adding new boards or updating room memberships.
    </p>`)
}

async function showNewSyncModal() {
    let boards = []
    try { boards = await api('GET', '/cw/boards') ?? [] } catch { /* show modal without boards */ }

    const boardCheckboxes = boards.map(b =>
        `<label><input type="checkbox" name="board" value="${b.id}"> ${esc(b.name)}</label>`
    ).join('')

    openModal('Run Sync', `
        <div class="form-group">
            <label>What to sync</label>
        </div>
        <label style="display:flex;align-items:center;gap:8px">
            <input type="checkbox" id="f-sync-boards" checked> Sync Boards
        </label>
        <label style="display:flex;align-items:center;gap:8px">
            <input type="checkbox" id="f-sync-webex" checked> Sync Webex Recipients
        </label>
        <label style="display:flex;align-items:center;gap:8px">
            <input type="checkbox" id="f-sync-tickets"> Sync Tickets
        </label>
        ${boards.length ? `<div class="form-group" style="margin-top:4px">
            <label>Board filter <span style="color:var(--muted)">(empty = all boards)</span></label>
            <div class="check-list">${boardCheckboxes}</div>
        </div>` : ''}`, async () => {
        const boardIds = Array.from(
            document.querySelectorAll('input[name="board"]:checked')
        ).map(el => parseInt(el.value))

        try {
            await api('POST', '/sync', {
                cw_boards:        document.getElementById('f-sync-boards').checked,
                webex_recipients: document.getElementById('f-sync-webex').checked,
                cw_tickets:       document.getElementById('f-sync-tickets').checked,
                board_ids:        boardIds,
            })
            closeModal()
            toast('Sync started', 'success')
            loadSync()
            startSyncPoll()
        } catch (e) { toast(e.message, 'error') }
    }, 'Start')
}

function startSyncPoll() {
    stopSyncPoll()
    syncPollTimer = setInterval(async () => {
        if (currentTab !== 'sync') { stopSyncPoll(); return }
        try {
            const status = await api('GET', '/sync/status')
            renderSync(status)
            if (!status?.status) stopSyncPoll()
        } catch { stopSyncPoll() }
    }, 3000)
}

function stopSyncPoll() {
    clearInterval(syncPollTimer)
    syncPollTimer = null
}

// ─────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────
async function loadConfig() {
    try {
        const cfg = await api('GET', '/config')
        renderConfig(cfg)
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

let configEnvLocked = []

function renderConfig(cfg) {
    const L = new Set(cfg.env_locked || [])
    configEnvLocked = cfg.env_locked || []
    const dis = k => L.has(k) ? 'disabled' : ''
    const envNote = k => L.has(k) ? '<div class="config-desc" style="color:var(--warning)">Set via environment — change the env var to update</div>' : ''
    setContent(`<div class="tab-header">
        <h2>Configuration</h2>
    </div>
    <div class="config-form">
        <div class="config-row">
            <div>
                <div class="config-label">Enable Workflows</div>
                <div class="config-desc">Master switch for the ticket workflow pipeline</div>
                ${envNote('attempt_workflow')}
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-workflow" ${cfg.attempt_workflow ? 'checked' : ''} ${dis('attempt_workflow')}>
                <span class="toggle-track"></span>
            </label>
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Enable Notifications</div>
                <div class="config-desc">Master switch for sending ticket notifications</div>
                ${envNote('attempt_notify')}
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-notify" ${cfg.attempt_notify ? 'checked' : ''} ${dis('attempt_notify')}>
                <span class="toggle-track"></span>
            </label>
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Bot Member Identifier</div>
                <div class="config-desc">Connectwise member the bot writes as; used to skip its own webhooks (prevents workflow loops)</div>
                ${envNote('cw_bot_member_identifier')}
            </div>
            <input class="config-input" type="text" id="c-bot-member" value="${esc(cfg.cw_bot_member_identifier || '')}" ${dis('cw_bot_member_identifier')}>
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Max Message Length</div>
                <div class="config-desc">Truncation limit for ticket note content</div>
            </div>
            <input class="config-input" type="number" id="c-max-len" value="${cfg.max_message_length}" min="1">
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Max Concurrent Syncs</div>
                <div class="config-desc">Limits parallel requests to Connectwise</div>
            </div>
            <input class="config-input" type="number" id="c-max-syncs" value="${cfg.max_concurrent_syncs}" min="1">
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Require 2FA</div>
                <div class="config-desc">All users must set up two-factor authentication to access the app</div>
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-require-totp" ${cfg.require_totp ? 'checked' : ''}>
                <span class="toggle-track"></span>
            </label>
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Debug Logging</div>
                <div class="config-desc">Enable debug-level log output without a server restart</div>
                ${envNote('debug_logging')}
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-debug-logging" ${cfg.debug_logging ? 'checked' : ''} ${dis('debug_logging')}>
                <span class="toggle-track"></span>
            </label>
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Log Buffer Size</div>
                <div class="config-desc">Max log entries held in memory for the web panel</div>
            </div>
            <input class="config-input" type="number" id="c-log-buffer-size" value="${cfg.log_buffer_size}" min="100">
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Log Retention</div>
                <div class="config-desc">How many days of logs to keep in the database (0 = keep forever)</div>
            </div>
            <input class="config-input" type="number" id="c-log-retention" value="${cfg.log_retention_days}" min="0">
        </div>
        <div class="config-row">
            <div>
                <div class="config-label">Log Cleanup Interval</div>
                <div class="config-desc">How often old logs are deleted, in hours</div>
            </div>
            <input class="config-input" type="number" id="c-log-cleanup-interval" value="${cfg.log_cleanup_interval_hours}" min="1">
        </div>
        <div class="config-section-title">Connection &amp; Credentials</div>
        <div class="config-desc" style="margin:-4px 0 4px">Credential changes take effect after a server restart. Fields set via environment variables are locked here.</div>
        ${credRow(cfg, 'root_url',       'c-root-url',     'Root URL',       'Externally reachable base URL for Connectwise webhook callbacks')}
        ${credRow(cfg, 'cw_company_id',  'c-cw-company',   'CW Company ID',  'Connectwise company identifier')}
        ${credRow(cfg, 'cw_client_id',   'c-cw-client',    'CW Client ID',   'Connectwise API client ID')}
        ${credRow(cfg, 'cw_public_key',  'c-cw-pub',       'CW Public Key',  'Connectwise API public key')}
        ${credRow(cfg, 'cw_private_key', 'c-cw-priv',      'CW Private Key', 'Connectwise API private key', { secret: true })}
        ${credRow(cfg, 'webex_secret',   'c-webex-secret', 'Webex Token',    'Webex bot bearer token', { secret: true })}
        <div class="config-row">
            <div>
                <div class="config-label">Connection Test</div>
                <div class="config-desc">Checks the saved credentials against Connectwise &amp; Webex</div>
            </div>
            <div class="conn-test">
                <button class="btn btn-ghost btn-sm" onclick="testConnections(this)">Test Connection</button>
                <div id="c-conn-result"></div>
            </div>
        </div>
        <div class="config-row">
            <button class="btn btn-primary btn-sm" onclick="saveConfig()">Save Changes</button>
        </div>
    </div>`)
}

async function testConnections(btn) {
    const out = document.getElementById('c-conn-result')
    btn.disabled = true
    out.innerHTML = '<span class="conn-pending">Testing…</span>'
    try {
        const res = await api('GET', '/config/test')
        out.innerHTML = connLine('ConnectWise', res.cw) + connLine('Webex', res.webex)
    } catch (e) {
        out.innerHTML = `<div class="conn-fail">${esc(e.message)}</div>`
    } finally {
        btn.disabled = false
    }
}

function connLine(name, c) {
    return (c && c.ok)
        ? `<div class="conn-ok">✓ ${esc(name)} connected</div>`
        : `<div class="conn-fail">✗ ${esc(name)}: ${esc((c && c.error) || 'failed')}</div>`
}

// credRow renders one credential field. Secrets are write-only: the value is never
// pre-filled (the API blanks it); the placeholder shows whether it's configured and
// a blank value on save means "keep current". Fields locked by an env var are
// disabled and not submitted.
function credRow(cfg, key, id, label, desc, opts = {}) {
    const locked   = (cfg.env_locked || []).includes(key)
    const isSecret = !!opts.secret
    const type     = isSecret ? 'password' : 'text'
    const val      = isSecret ? '' : esc(cfg[key] || '')
    let ph = ''
    if (isSecret) ph = cfg[key + '_set'] ? '•••••••• (configured)' : 'Not set'
    if (locked)   ph = 'Set via environment'
    const note = locked
        ? '<div class="config-desc" style="color:var(--warning)">Set via environment — change the env var to update</div>'
        : (isSecret ? '<div class="config-desc">Leave blank to keep the current value</div>' : '')
    return `<div class="config-row">
        <div>
            <div class="config-label">${esc(label)}</div>
            <div class="config-desc">${esc(desc)}</div>
            ${note}
        </div>
        <input class="config-input" type="${type}" id="${id}" value="${val}" placeholder="${esc(ph)}" autocomplete="off" spellcheck="false" ${locked ? 'disabled' : ''}>
    </div>`
}

async function saveConfig() {
    const lockedSet = new Set(configEnvLocked)
    const payload = {
        max_message_length:         parseInt(document.getElementById('c-max-len').value)              || 300,
        max_concurrent_syncs:       parseInt(document.getElementById('c-max-syncs').value)            || 5,
        require_totp:               document.getElementById('c-require-totp').checked,
        log_buffer_size:            parseInt(document.getElementById('c-log-buffer-size').value)      || 500,
        log_retention_days:         parseInt(document.getElementById('c-log-retention').value)        ?? 7,
        log_cleanup_interval_hours: parseInt(document.getElementById('c-log-cleanup-interval').value) || 24,
    }

    // Env-lockable fields: only send when not pinned by an environment variable
    // (an env-locked field would just be overwritten by the env value on restart).
    if (!lockedSet.has('attempt_workflow'))         payload.attempt_workflow = document.getElementById('c-workflow').checked
    if (!lockedSet.has('attempt_notify'))           payload.attempt_notify = document.getElementById('c-notify').checked
    if (!lockedSet.has('debug_logging'))            payload.debug_logging = document.getElementById('c-debug-logging').checked
    if (!lockedSet.has('cw_bot_member_identifier')) payload.cw_bot_member_identifier = document.getElementById('c-bot-member').value.trim()

    // Credential fields: skip env-locked (disabled) ones; for secrets, a blank value
    // means "keep current" so we omit it.
    let credChanged = false
    const addCred = (key, id, secret) => {
        const el = document.getElementById(id)
        if (!el || el.disabled) return
        const v = el.value.trim()
        if (secret && v === '') return
        payload[key] = v
        credChanged = true
    }
    addCred('root_url',       'c-root-url')
    addCred('cw_company_id',  'c-cw-company')
    addCred('cw_client_id',   'c-cw-client')
    addCred('cw_public_key',  'c-cw-pub')
    addCred('cw_private_key', 'c-cw-priv',      true)
    addCred('webex_secret',   'c-webex-secret', true)

    try {
        await api('PUT', '/config', payload)
        toast(credChanged ? 'Config saved — restart the server to apply credential changes' : 'Config saved', 'success')
        loadConfig()
    } catch (e) { toast(e.message, 'error') }
}

// ─────────────────────────────────────────────────────────
// Logs
// ─────────────────────────────────────────────────────────
let logsPollTimer        = null
let logsFrozen           = false
let logsLastEntries      = []

function logsPrefs() {
    try { return JSON.parse(localStorage.getItem('logsPrefs') || '{}') } catch { return {} }
}
function saveLogsPrefs(patch) {
    localStorage.setItem('logsPrefs', JSON.stringify({ ...logsPrefs(), ...patch }))
}

const _lp              = logsPrefs()
let logsLevelFilter    = _lp.levelFilter    ?? 'ALL'
let logsContextFilter  = _lp.contextFilter  ?? 'ALL'
let logsHideGin        = _lp.hideGin         ?? false
let logsSearch         = ''

async function loadLogs() {
    try {
        const entries = await api('GET', '/logs')
        logsLastEntries = entries || []
        renderLogs(logsLastEntries)
        startLogsPoll()
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderLogs(entries) {
    const levelOpts = ['ALL', 'DEBUG', 'INFO', 'WARN', 'ERROR'].map(l =>
        `<option value="${l}" ${l === logsLevelFilter ? 'selected' : ''}>${l}</option>`
    ).join('')

    const contextDefs = [
        { value: 'ALL',      label: 'All' },
        { value: 'TICKET',   label: 'Ticket Processing' },
        { value: 'NOTIF',    label: 'Notification Processing' },
    ]
    const contextOpts = contextDefs.map(c =>
        `<option value="${c.value}" ${c.value === logsContextFilter ? 'selected' : ''}>${c.label}</option>`
    ).join('')

    let filtered = logsLevelFilter === 'ALL'
        ? entries
        : entries.filter(e => e.level.toUpperCase() === logsLevelFilter)

    if (logsContextFilter === 'TICKET') {
        filtered = filtered.filter(e => (e.message || '') === 'ticket processed')
    } else if (logsContextFilter === 'NOTIF') {
        filtered = filtered.filter(e => (e.message || '') === 'notification processed')
    }

    if (logsHideGin) {
        filtered = filtered.filter(e => !(e.message || '').startsWith('[GIN]'))
    }
    if (logsSearch) {
        const term = logsSearch.toLowerCase()
        filtered = filtered.filter(e => {
            if ((e.message || '').toLowerCase().includes(term)) return true
            if (e.attrs && JSON.stringify(e.attrs).toLowerCase().includes(term)) return true
            return false
        })
    }

    const rows = filtered.slice().reverse().map(e => {
        const lvl   = (e.level || '').toUpperCase()
        const cls   = lvl === 'ERROR' ? 'log-error' : lvl === 'WARN' ? 'log-warn' : lvl === 'DEBUG' ? 'log-debug' : ''
        const time  = e.time ? new Date(e.time).toLocaleTimeString() : '—'
        const attrs = e.attrs ? ' ' + Object.entries(e.attrs).map(([k,v]) => {
            const val = (v !== null && typeof v === 'object') ? JSON.stringify(v) : String(v)
            return `<span class="log-attr">${esc(k)}=<span class="log-attr-val">${esc(val)}</span></span>`
        }).join(' ') : ''
        return `<div class="log-row ${cls}">
            <span class="log-time">${time}</span>
            <span class="log-level">${esc(e.level)}</span>
            <span class="log-msg">${esc(e.message)}${attrs}</span>
        </div>`
    })

    const isEmpty        = rows.length === 0
    const searchFocused  = document.activeElement?.id === 'logs-search'
    const searchPos      = searchFocused ? document.getElementById('logs-search')?.selectionStart : null

    setContent(`<div class="tab-header">
        <h2>Logs</h2>
        <div style="display:flex;gap:8px;align-items:center">
            <select id="logs-level-filter" onchange="setLogsFilter(this.value)" class="btn btn-ghost btn-sm" style="cursor:pointer">${levelOpts}</select>
            <select id="logs-context-filter" onchange="setLogsContextFilter(this.value)" class="btn btn-ghost btn-sm" style="cursor:pointer;width:180px">${contextOpts}</select>
            <input id="logs-search" type="text" placeholder="Search…" value="${esc(logsSearch)}" oninput="setLogsSearch(this.value)" class="logs-search-input">
            <div class="logs-options-wrap">
                <button class="btn btn-ghost btn-sm" onclick="toggleLogsOptions(event)">Options</button>
                <div id="logs-options-popup" class="logs-options-popup hidden">
                    <label><input type="checkbox" ${logsHideGin ? 'checked' : ''} onchange="setLogsHideGin(this.checked)"> Hide request logs</label>
                </div>
            </div>
            <button class="btn btn-ghost btn-sm" onclick="toggleLogFreeze()">${logsFrozen ? 'Unfreeze' : 'Freeze'}</button>
            <button class="btn btn-ghost btn-sm" onclick="loadLogs()">Refresh</button>
        </div>
    </div>
    <div class="log-list">
        ${isEmpty ? '<div class="empty-state">No log entries</div>' : rows.join('')}
    </div>`)

    if (searchFocused) {
        const input = document.getElementById('logs-search')
        if (input) { input.focus(); if (searchPos != null) input.setSelectionRange(searchPos, searchPos) }
    }
}

function setLogsFilter(level) {
    logsLevelFilter = level
    saveLogsPrefs({ levelFilter: level })
    loadLogs()
}

function setLogsContextFilter(val) {
    logsContextFilter = val
    saveLogsPrefs({ contextFilter: val })
    loadLogs()
}

function setLogsHideGin(val) {
    logsHideGin = val
    saveLogsPrefs({ hideGin: val })
    loadLogs()
}

function setLogsSearch(val) {
    logsSearch = val
    renderLogs(logsLastEntries)
}

function toggleLogsOptions(e) {
    e.stopPropagation()
    document.getElementById('logs-options-popup')?.classList.toggle('hidden')
}

function toggleLogFreeze() {
    logsFrozen = !logsFrozen
    if (logsFrozen) {
        stopLogsPoll()
    } else {
        loadLogs()
    }
    // re-render toolbar state without re-fetching
    const btn = document.querySelector('.log-list')?.previousElementSibling?.querySelector('button[onclick="toggleLogFreeze()"]')
    if (btn) btn.textContent = logsFrozen ? 'Unfreeze' : 'Freeze'
}

function startLogsPoll() {
    if (logsFrozen) return
    stopLogsPoll()
    logsPollTimer = setInterval(async () => {
        if (currentTab !== 'logs') { stopLogsPoll(); return }
        if (logsFrozen) { stopLogsPoll(); return }
        try {
            const entries = await api('GET', '/logs')
            logsLastEntries = entries || []
            renderLogs(logsLastEntries)
        } catch { stopLogsPoll() }
    }, 5000)
}

function stopLogsPoll() {
    clearInterval(logsPollTimer)
    logsPollTimer = null
}

// ─────────────────────────────────────────────────────────
// Init
// ─────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('login-email').addEventListener('keydown', e => {
        if (e.key === 'Enter') document.getElementById('login-password').focus()
    })
    document.getElementById('login-password').addEventListener('keydown', e => {
        if (e.key === 'Enter') login()
    })
    document.getElementById('totp-code').addEventListener('keydown', e => {
        if (e.key === 'Enter') submitTOTPVerify()
    })
    document.getElementById('reset-current').addEventListener('keydown', e => {
        if (e.key === 'Enter') document.getElementById('reset-new').focus()
    })
    document.getElementById('reset-new').addEventListener('keydown', e => {
        if (e.key === 'Enter') document.getElementById('reset-confirm').focus()
    })
    document.getElementById('reset-confirm').addEventListener('keydown', e => {
        if (e.key === 'Enter') submitPasswordReset()
    })
    document.addEventListener('keydown', e => {
        if (e.key === 'Escape') closeModal()
    })
    document.addEventListener('click', () => {
        document.getElementById('account-dropdown').classList.add('hidden')
        document.getElementById('logs-options-popup')?.classList.add('hidden')
    })
    checkSavedKey()
})
