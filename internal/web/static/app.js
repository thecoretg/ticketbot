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
    switchTab(tabLoaders[hash] ? hash : 'rules')
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
    switchTab(tabLoaders[hash] ? hash : 'rules')
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
    rules:    loadRules,
    forwards: loadForwards,
    users:    loadUsers,
    keys:     loadKeys,
    sync:     loadSync,
    config:   loadConfig,
    logs:     loadLogs,
}

function switchTab(tab) {
    stopSyncPoll()
    stopLogsPoll()
    currentTab = tab
    window.location.hash = tab
    document.querySelectorAll('.tab').forEach(el => {
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
// Rules
// ─────────────────────────────────────────────────────────
async function loadRules() {
    try {
        const rules = await api('GET', '/notifiers/rules')
        renderRules(rules || [])
    } catch (e) {
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderRules(rules) {
    const header = `<div class="tab-header">
        <h2>Notifier Rules</h2>
        <button class="btn btn-primary btn-sm" onclick="showNewRuleModal()">+ New Rule</button>
    </div>`

    const thead = '<th>Enabled</th><th>Board</th><th>Recipient</th><th></th>'
    const rows  = rules.map(r => `<tr>
        <td>${badge(r.enabled)}</td>
        <td>${esc(r.board_name)}</td>
        <td>${esc(r.recipient_name)} <span style="color:var(--muted);font-size:11px">${esc(r.recipient_type)}</span></td>
        <td class="actions"><button class="btn btn-danger" onclick="deleteRule(${r.id})">Delete</button></td>
    </tr>`)

    setContent(header + tableWrap(thead, rows))
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

    openModal('New Notifier Rule', `
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
            toast('Rule created', 'success')
            loadRules()
        } catch (e) { toast(e.message, 'error') }
    })
}

async function deleteRule(id) {
    if (!confirm('Delete this rule?')) return
    try {
        await api('DELETE', `/notifiers/rules/${id}`)
        toast('Rule deleted', 'success')
        loadRules()
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
        setContent(`<div class="empty-state">${esc(e.message)}</div>`)
    }
}

function renderForwards(fwds) {
    const header = `<div class="tab-header">
        <h2>Notification Forwards</h2>
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

    setContent(header + tableWrap(thead, rows))
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

function renderConfig(cfg) {
    setContent(`<div class="tab-header">
        <h2>Configuration</h2>
    </div>
    <div class="config-form">
        <div class="config-row">
            <div>
                <div class="config-label">Attempt Notify</div>
                <div class="config-desc">Master switch for sending ticket notifications</div>
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-notify" ${cfg.attempt_notify ? 'checked' : ''}>
                <span class="toggle-track"></span>
            </label>
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
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-debug-logging" ${cfg.debug_logging ? 'checked' : ''}>
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
        <div class="config-row">
            <button class="btn btn-primary btn-sm" onclick="saveConfig()">Save Changes</button>
        </div>
    </div>`)
}

async function saveConfig() {
    try {
        await api('PUT', '/config', {
            attempt_notify:             document.getElementById('c-notify').checked,
            max_message_length:         parseInt(document.getElementById('c-max-len').value)              || 300,
            max_concurrent_syncs:       parseInt(document.getElementById('c-max-syncs').value)            || 5,
            require_totp:               document.getElementById('c-require-totp').checked,
            debug_logging:              document.getElementById('c-debug-logging').checked,
            log_buffer_size:            parseInt(document.getElementById('c-log-buffer-size').value)      || 500,
            log_retention_days:         parseInt(document.getElementById('c-log-retention').value)        ?? 7,
            log_cleanup_interval_hours: parseInt(document.getElementById('c-log-cleanup-interval').value) || 24,
        })
        toast('Config saved', 'success')
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
