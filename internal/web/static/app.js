// ─────────────────────────────────────────────────────────
// State
// ─────────────────────────────────────────────────────────
let currentTab    = 'rules'
let currentUser   = null
let pendingToken  = null   // pending TOTP token after password login
let totpEnabled   = false  // cached TOTP status for account menu
let syncPollTimer = null
let modalSubmitFn = null

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
            showApp()
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
        const [me, totp] = await Promise.all([
            api('GET', '/users/me'),
            api('GET', '/auth/totp'),
        ])
        currentUser = me
        totpEnabled = totp.enabled
        document.getElementById('header-email').textContent   = currentUser.email_address
        document.getElementById('dropdown-email').textContent = currentUser.email_address
        updateTOTPMenuItem()
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
    btn.textContent = totpEnabled ? 'Disable 2FA' : 'Set Up 2FA'
}

function handleTOTPMenuClick() {
    document.getElementById('account-dropdown').classList.add('hidden')
    if (totpEnabled) {
        showTOTPDisableModal()
    } else {
        showTOTPSetupModal()
    }
}

async function showTOTPSetupModal() {
    // Phase 1: fetch QR code and secret
    let setupData
    try {
        setupData = await api('POST', '/auth/totp/setup')
    } catch (e) { toast(e.message, 'error'); return }

    openModal('Set Up Two-Factor Auth', `
        <p style="color:var(--muted);font-size:13px">Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.).</p>
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
                <button class="btn btn-primary" onclick="closeModal()">Done</button>`
            modalSubmitFn = null
            window._recoveryCodes = res.recovery_codes
        } catch (e) { toast(e.message || 'Failed to enable 2FA', 'error') }
    }, 'Enable 2FA')
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
}

function switchTab(tab) {
    stopSyncPoll()
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
    const fmt = d => new Date(d).toLocaleDateString('en-US', { month: '2-digit', day: '2-digit' })
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
            <label>Start Date <span style="color:var(--muted)">(optional)</span></label>
            <input type="date" id="f-start">
        </div>
        <div class="form-group">
            <label>End Date <span style="color:var(--muted)">(optional)</span></label>
            <input type="date" id="f-end">
        </div>
        <div class="form-group">
            <label>Source Keeps Copy?</label>
            <select id="f-keep">
                <option value="true">Yes</option>
                <option value="false">No</option>
            </select>
        </div>`, async () => {
        const sourceId = parseInt(document.getElementById('f-source').value)
        const destId   = parseInt(document.getElementById('f-dest').value)
        const startRaw = document.getElementById('f-start').value
        const endRaw   = document.getElementById('f-end').value
        const keepCopy = document.getElementById('f-keep').value === 'true'

        if (sourceId === destId) { toast('Source and destination must be different', 'error'); return }
        if (endRaw && startRaw && endRaw < startRaw) { toast('End date cannot be before start date', 'error'); return }

        const payload = {
            user_email:     sourceId,
            dest_email:     destId,
            enabled:        true,
            user_keeps_copy: keepCopy,
        }
        if (startRaw) payload.start_date = new Date(startRaw).toISOString()
        if (endRaw)   payload.end_date   = new Date(endRaw).toISOString()

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
                cw_boards:           document.getElementById('f-sync-boards').checked,
                webex_recipients:    document.getElementById('f-sync-webex').checked,
                cw_tickets:          document.getElementById('f-sync-tickets').checked,
                board_ids:           boardIds,
                max_concurrent_syncs: 5,
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
                <div class="config-label">Skip Launch Syncs</div>
                <div class="config-desc">Skip syncing boards and recipients on server startup</div>
            </div>
            <label class="toggle">
                <input type="checkbox" id="c-skip-launch" ${cfg.skip_launch_syncs ? 'checked' : ''}>
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
            <button class="btn btn-primary btn-sm" onclick="saveConfig()">Save Changes</button>
        </div>
    </div>`)
}

async function saveConfig() {
    try {
        await api('PUT', '/config', {
            id:                   1,
            attempt_notify:       document.getElementById('c-notify').checked,
            skip_launch_syncs:    document.getElementById('c-skip-launch').checked,
            max_message_length:   parseInt(document.getElementById('c-max-len').value)   || 300,
            max_concurrent_syncs: parseInt(document.getElementById('c-max-syncs').value) || 5,
        })
        toast('Config saved', 'success')
    } catch (e) { toast(e.message, 'error') }
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
    })
    checkSavedKey()
})
