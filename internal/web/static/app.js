// ─────────────────────────────────────────────────────────
// Theme & palette
//
// index.html sets data-theme / data-palette before first paint; these keep
// them in sync with the controls in the topbar.
// ─────────────────────────────────────────────────────────
const PALETTES = [
    { id: 'harbor',   name: 'Harbor',   desc: 'Cool grey, deep teal accent',   seed: '#0E6F80' },
    { id: 'ember',    name: 'Ember',    desc: 'Warm paper, persimmon accent',  seed: '#C64A26' },
    { id: 'indigo',   name: 'Indigo',   desc: 'Slate neutrals, indigo accent', seed: '#3D45A8' },
    { id: 'moss',     name: 'Moss',     desc: 'Sage neutrals, forest accent',  seed: '#3E6B43' },
    { id: 'plum',     name: 'Plum',     desc: 'Mauve neutrals, plum accent',   seed: '#8E3A63' },
    { id: 'graphite', name: 'Graphite', desc: 'Monochrome, ink accent',        seed: '#26251F' },
]

function applyTheme(theme) {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('theme', theme)
    const btn = document.getElementById('theme-toggle')
    if (btn) btn.innerHTML = icon(theme === 'dark' ? 'sun' : 'moon')
}

function toggleTheme() {
    applyTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark')
}

function applyPalette(id) {
    document.documentElement.dataset.palette = id
    localStorage.setItem('palette', id)
}

function openPaletteMenu(e) {
    e.stopPropagation()
    const current = document.documentElement.dataset.palette
    toggleMenu(e.currentTarget, PALETTES.map(p => ({
        label: p.name,
        swatch: p.seed,
        current: p.id === current,
        run: () => { applyPalette(p.id); toast(`${p.name} palette — ${p.desc}`) },
    })))
}

// ─────────────────────────────────────────────────────────
// Shell: sidebar nav, popover menus, mobile drawer
// ─────────────────────────────────────────────────────────
const NAV = [
    { label: 'Automation', items: [
        { tab: 'workflows', name: 'Workflows', icon: 'bolt' },
        { tab: 'tickets',   name: 'Tickets',   icon: 'inbox' },
        { tab: 'forwards',  name: 'Forwards',  icon: 'mail' },
        { tab: 'lists',     name: 'Lists',     icon: 'blocks' },
    ]},
    { label: 'Administration', items: [
        { tab: 'users',  name: 'Users',  icon: 'users' },
        { tab: 'keys',   name: 'Keys',   icon: 'key' },
        { tab: 'sync',   name: 'Sync',   icon: 'globe' },
        { tab: 'intake', name: 'Intake', icon: 'inbox' },
        { tab: 'config', name: 'Config', icon: 'cog' },
        { tab: 'sso',    name: 'Single sign-on', icon: 'shield' },
        { tab: 'logs',   name: 'Logs',   icon: 'book' },
    ], admin: true },
]
const NAV_ITEMS = NAV.flatMap(g => g.items)
const ADMIN_TABS = new Set(NAV.filter(g => g.admin).flatMap(g => g.items.map(i => i.tab)))

// Roles. The server enforces these on every route; the UI only hides what would fail.
function isAdmin() { return currentUser?.role === 'admin' }
function canEdit() { return currentUser?.role === 'admin' || currentUser?.role === 'editor' }
// editOnly drops markup for controls a viewer cannot use.
function editOnly(html) { return canEdit() ? html : '' }

function buildNav() {
    document.getElementById('nav').innerHTML = NAV.filter(g => !g.admin || isAdmin()).map(g => `
        <div class="nav-group">
            <div class="nav-label eyebrow">${g.label}</div>
            ${g.items.map(i => `<button class="nav-item" data-tab="${i.tab}" onclick="switchTab('${i.tab}')">
                ${icon(i.icon)}<span class="label">${i.name}</span>
            </button>`).join('')}
        </div>`).join('')
}

function toggleNav(e) {
    e.stopPropagation()
    document.getElementById('app').classList.toggle('nav-open')
}

// openMenu renders a .popover anchored under a control. Items are
// { label, icon | swatch, danger, current, run }.
let menuEl     = null
let menuAnchor = null   // the control the open menu hangs off; a click on it closes the menu

function closeMenu() {
    menuEl?.remove()
    menuEl     = null
    menuAnchor = null
}

// toggleMenu closes the menu when its own anchor is clicked again, otherwise opens it.
function toggleMenu(anchor, items) {
    if (menuEl && menuAnchor === anchor) { closeMenu(); return }
    openMenu(anchor, items)
}

function openMenu(anchor, items) {
    closeMenu()
    menuAnchor = anchor
    buildMenu(items)

    const r = anchor.getBoundingClientRect()
    const below = window.scrollY + r.bottom + 6
    const above = window.scrollY + r.top - menuEl.offsetHeight - 6
    // anchored controls near the bottom of the frame (the sidebar foot) flip upwards
    menuEl.style.top  = `${r.bottom + menuEl.offsetHeight + 12 > window.innerHeight ? above : below}px`
    menuEl.style.left = `${Math.max(12, Math.min(r.left, window.innerWidth - menuEl.offsetWidth - 12))}px`
}

// openMenuAt opens the same popover at a viewport point: a context menu. It flips or slides to
// stay inside the frame.
function openMenuAt(x, y, items) {
    closeMenu()
    buildMenu(items)
    const w = menuEl.offsetWidth, h = menuEl.offsetHeight
    const left = x + w + 12 > window.innerWidth ? x - w : x
    const top  = y + h + 12 > window.innerHeight ? y - h : y
    menuEl.style.left = `${window.scrollX + Math.max(12, left)}px`
    menuEl.style.top  = `${window.scrollY + Math.max(12, top)}px`
}

// buildMenu renders the items into menuEl. An item is '-' for a divider or
// { label, icon?, swatch?, current?, danger?, disabled?, run }.
function buildMenu(items) {
    menuEl = document.createElement('div')
    menuEl.className = 'menu'
    menuEl.setAttribute('role', 'menu')
    menuEl.innerHTML = items.map((it, k) => it === '-' ? '<hr>' : `
        <button role="menuitem" data-mi="${k}" class="${it.danger ? 'danger' : ''}"${it.disabled ? ' disabled' : ''}>
            ${it.swatch ? `<span class="swatch" style="background:${it.swatch}"></span>` : (it.icon ? icon(it.icon) : '')}
            <span>${esc(it.label)}</span>
            ${it.current ? '<span class="check-mark">●</span>' : ''}
        </button>`).join('')
    document.body.appendChild(menuEl)
    menuEl.addEventListener('click', e => {
        const btn = e.target.closest('[data-mi]')
        if (!btn || btn.disabled) return
        const item = items[Number(btn.dataset.mi)]
        closeMenu()
        item.run?.()
    })
}

// ─────────────────────────────────────────────────────────
// State
// ─────────────────────────────────────────────────────────
let currentTab        = 'workflows'
let currentHash       = ''
let appConfig         = null   // last-loaded /config, shared across pages
let tabGuard          = null   // page-installed veto for navigation (unsaved changes)
let currentUser       = null
let authMethods       = { sso: false, password: true }  // last-loaded /auth/methods
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
            return `<span class="pwd-req${ok ? ' ok' : ''}">${icon(ok ? 'check' : 'dots')}${r.label}</span>`
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
    if (!res.ok) {
        const err = new Error(data.error || `Request failed: ${res.status}`)
        err.status = res.status
        err.data   = data
        if (res.status === 401 && !path.startsWith('/auth') && path !== '/authtest') sessionExpired()
        throw err
    }
    return data
}

// sessionExpired drops the user back to the login screen when the session cookie stops working
// mid-use, instead of every page showing a "Request failed: 401" toast.
let sessionExpiredShown = false
function sessionExpired() {
    if (sessionExpiredShown || !currentUser) return
    sessionExpiredShown = true
    stopSyncPoll()
    stopLogsPoll()
    stopIntakePoll()
    rsStopPoll()
    tabGuard = null
    currentUser = null
    totpSetupRequired = false
    closeModal()
    document.getElementById('app').classList.add('hidden')
    showLogin().then(() => showLoginErr('Your session expired. Sign in again to continue.'))
    setTimeout(() => { sessionExpiredShown = false }, 2000)
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
            // showApp prompts for 2FA setup itself when config requires it; only fall back to the
            // login response's flag if it could not (config fetch failed), or the modal opens twice
            const prompted = await showApp()
            if (res?.totp_setup_required && !prompted) showTOTPSetupModal(true)
        }
    } catch (e) {
        // 401 is a bad password; anything else (server down, 500, password sign-in off) deserves
        // its real message
        showLoginErr(e.status === 401 || e.status === 400 ? 'Invalid email or password' : (e.message || 'Login failed'))
    } finally {
        btn.disabled    = false
        btn.textContent = 'Continue'
    }
}

function showLoginErr(msg) {
    const el = document.getElementById('login-err')
    el.textContent = msg
    el.classList.remove('hidden')
}

// loadAuthMethods asks the server which sign-in options to offer and shapes the login card:
// a Microsoft button, the password form, or both with a divider between them.
async function loadAuthMethods() {
    try { authMethods = await api('GET', '/auth/methods') } catch { authMethods = { sso: false, password: true } }
    const { sso, password } = authMethods
    document.getElementById('sso-btn').classList.toggle('hidden', !sso)
    document.getElementById('login-or').classList.toggle('hidden', !(sso && password))
    document.getElementById('login-password-form').classList.toggle('hidden', !password)
    // Only Microsoft on offer: keep a quiet way to the password form for the break-glass admin.
    // The server refuses everyone else, so revealing the form grants nothing.
    document.getElementById('login-alt').classList.toggle('hidden', !(sso && !password))
    document.getElementById('login-sub').textContent = sso && !password
        ? 'Use your Microsoft account to continue'
        : 'Enter your credentials to continue'
}

// showPasswordForm reveals the password form under the Microsoft button.
function showPasswordForm(e) {
    e.preventDefault()
    document.getElementById('login-alt').classList.add('hidden')
    document.getElementById('login-or').classList.remove('hidden')
    document.getElementById('login-password-form').classList.remove('hidden')
    document.getElementById('login-email').focus()
}

// showLogin reveals the login card with the right sign-in options.
async function showLogin() {
    await loadAuthMethods()
    document.getElementById('login').classList.remove('hidden')
}

// A failed Microsoft sign-in lands on /?err=<message>. Show it once, then drop it from
// the address bar so a refresh does not repeat it.
function consumeSSOError() {
    const params = new URLSearchParams(window.location.search)
    const err = params.get('err')
    if (!err) return
    showLoginErr(err)
    params.delete('err')
    const qs = params.toString()
    history.replaceState(null, '', window.location.pathname + (qs ? `?${qs}` : '') + window.location.hash)
}

async function logout() {
    try { await api('POST', '/auth/logout') } catch {}
    currentUser  = null
    pendingToken = null
    totpEnabled  = false
    stopSyncPoll()
    stopLogsPoll()
    stopIntakePoll()
    rsStopPoll()
    document.getElementById('login-email').value    = ''
    document.getElementById('login-password').value = ''
    document.getElementById('login-err').classList.add('hidden')
    closeMenu()
    document.getElementById('app').classList.add('hidden')
    document.getElementById('password-reset').classList.add('hidden')
    document.getElementById('totp-verify').classList.add('hidden')
    showLogin()
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

// showApp loads the signed-in shell. Returns true when it opened the required 2FA setup
// modal instead of routing, so callers do not open a second one.
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
        appConfig   = cfg
        requireTOTP = cfg.require_totp
        buildNav()
        document.getElementById('header-email').textContent    = currentUser.email_address
        document.getElementById('header-initials').textContent = emailInitials(currentUser.email_address)
        // Microsoft accounts have no local password or TOTP; Entra handles their MFA.
        if (requireTOTP && !totpEnabled && !currentUser.sso) {
            await showTOTPSetupModal(true)
            return true
        }
    } catch {}
    routeFromHash()
    return false
}

// ─────────────────────────────────────────────────────────
// Account menu
// ─────────────────────────────────────────────────────────
function emailInitials(email) {
    const name = String(email || '').split('@')[0]
    const parts = name.split(/[._-]+/).filter(Boolean)
    return ((parts[0]?.[0] || '') + (parts[1]?.[0] || '')).toUpperCase() || '—'
}

function totpMenuLabel() {
    if (!totpEnabled) return 'Set up 2FA'
    return requireTOTP ? 'Reset 2FA' : 'Disable 2FA'
}

function toggleAccountMenu(e) {
    e.stopPropagation()
    const items = []
    if (!currentUser?.sso) {
        items.push({ label: 'Change password', icon: 'key',    run: showChangePasswordModal })
        items.push({ label: totpMenuLabel(),   icon: 'shield', run: handleTOTPMenuClick })
        items.push('-')
    }
    if (isAdmin()) items.push({ label: 'Restart server', icon: 'bolt', danger: true, run: confirmRestart })
    items.push({ label: 'Log out', icon: 'logout', danger: true, run: logout })
    toggleMenu(e.currentTarget, items)
}

function showChangePasswordModal() {
    openModal('Change password', `
        <div class="stack gap4">
            <div class="field">
                <label for="f-cur-pwd">Current password</label>
                <input class="input" type="password" id="f-cur-pwd" autocomplete="current-password">
            </div>
            <div class="field">
                <label for="f-new-pwd">New password</label>
                <input class="input" type="password" id="f-new-pwd" autocomplete="new-password">
                <div id="f-pwd-reqs" class="pwd-reqs"></div>
            </div>
            <div class="field">
                <label for="f-confirm-pwd">Confirm new password</label>
                <input class="input" type="password" id="f-confirm-pwd" autocomplete="new-password">
            </div>
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
    }, 'Change password')
    setTimeout(() => attachPwdReqs('f-new-pwd', 'f-pwd-reqs'), 50)
}

function handleTOTPMenuClick() {
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
        ? `<div class="callout warn">${icon('shield')}<div class="body"><b>Two-factor authentication is required</b>Set it up to continue.</div></div>`
        : `<div class="callout info">${icon('info')}<div class="body">Scan this QR code with your authenticator app (Google Authenticator, Authy, 1Password).</div></div>`

    openModal('Set up two-factor auth', `
        <div class="stack gap4">
            ${desc}
            <div class="row gap4 wrap">
                <img class="qr" src="data:image/png;base64,${setupData.qr_png}" alt="QR code for the two-factor secret">
                <div class="field grow" style="min-width:200px">
                    <label for="f-totp-secret">Or enter this secret manually</label>
                    <div class="secret" id="f-totp-secret">${esc(setupData.secret)}</div>
                </div>
            </div>
            <div class="field">
                <label for="f-totp-pwd">Current password</label>
                <input class="input" type="password" id="f-totp-pwd" autocomplete="current-password">
            </div>
            <div class="field">
                <label for="f-totp-code">Confirmation code</label>
                <input class="input" type="text" id="f-totp-code" inputmode="numeric" autocomplete="one-time-code" placeholder="000000" maxlength="6">
                <span class="hint">The 6-digit code your authenticator app shows right now.</span>
            </div>
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
            // Replace modal body with recovery codes (shown once)
            document.getElementById('modal-title').textContent = 'Two-factor auth enabled'
            document.getElementById('modal-body').innerHTML = `
                <div class="stack gap4">
                    <div class="callout warn">${icon('alert')}<div class="body">
                        <b>Save these recovery codes somewhere safe</b>Each one works once, and they are not shown again.
                    </div></div>
                    <div class="recovery-codes">
                        ${res.recovery_codes.map(c => `<span>${esc(c)}</span>`).join('')}
                    </div>
                </div>`
            document.getElementById('modal-foot').innerHTML = `
                <button class="btn btn-ghost" onclick="copyRecoveryCodes()">${icon('copy')}Copy all</button>
                <button class="btn btn-primary" onclick="finishTOTPSetup()">Done</button>`
            modalSubmitFn = null
            window._recoveryCodes = res.recovery_codes
        } catch (e) { toast(e.message || 'Failed to enable 2FA', 'error') }
    }, 'Enable 2FA')

    if (required) {
        // Remove the cancel button — setup cannot be skipped when required
        const cancel = document.querySelector('#modal-foot .btn-ghost')
        if (cancel) cancel.remove()
    }
}

function finishTOTPSetup() {
    totpSetupRequired = false
    closeModal()
    routeFromHash()
}

function copyRecoveryCodes() {
    const codes = (window._recoveryCodes || []).join('\n')
    copyText(codes, 'Recovery codes copied')
}

function showTOTPDisableModal() {
    openModal('Disable two-factor auth', `
        <div class="stack gap4">
            <div class="callout warn">${icon('alert')}<div class="body">
                <b>This account loses its second factor</b>Your recovery codes are deleted too.
            </div></div>
            <div class="field">
                <label for="f-disable-pwd">Current password</label>
                <input class="input" type="password" id="f-disable-pwd" autocomplete="current-password">
            </div>
        </div>`, async () => {
        const pwd = document.getElementById('f-disable-pwd').value
        if (!pwd) { toast('Password is required', 'error'); return }
        try {
            await api('DELETE', '/auth/totp', { password: pwd })
            totpEnabled = false
            closeModal()
            toast('Two-factor authentication disabled', 'success')
        } catch (e) { toast(e.message || 'Failed to disable 2FA', 'error') }
    }, 'Disable 2FA', 'danger')
}

function confirmRestart() {
    openModal('Restart server', `
        <div class="callout warn">${icon('alert')}<div class="body">
            <b>Ticketbot goes offline for a few seconds</b>
            The server restarts and this page reconnects on its own. Webhooks that arrive during the restart are retried by ConnectWise.
        </div></div>`, async () => {
        try {
            await api('POST', '/admin/restart')
        } catch { /* server closes the connection during shutdown, that's fine */ }
        closeModal()
        showRestartBanner()
    }, 'Restart', 'danger')
}

function showRestartBanner() {
    const banner = document.createElement('div')
    banner.id = 'restart-banner'
    banner.className = 'banner banner-sticky'
    banner.setAttribute('role', 'status')
    banner.innerHTML = `${icon('clock')}<div><b>Restarting</b> — reconnecting to the server…</div>`
    document.getElementById('content').prepend(banner)

    // the old process answers for a moment after the restart call; only a failure followed by a
    // success means the new process is up (or a 10s ceiling, in case shutdown was too fast to see)
    let sawDown = false
    const started = Date.now()
    const poll = setInterval(async () => {
        try {
            const res = await fetch('/healthcheck', { cache: 'no-store' })
            if (!res.ok) throw new Error(String(res.status))
            if (!sawDown && Date.now() - started < 10000) return
            clearInterval(poll)
            banner.remove()
            toast('Server restarted successfully', 'success')
        } catch { sawDown = true }
    }, 1500)
}

function checkSavedKey() {
    api('GET', '/authtest')
        .then(showApp)
        .catch(async () => {
            await showLogin()
            consumeSSOError()
        })
}

// ─────────────────────────────────────────────────────────
// Tabs
// ─────────────────────────────────────────────────────────
const tabLoaders = {
    forwards: loadForwards,
    users:    loadUsers,
    keys:     loadKeys,
    sync:     loadSync,
    config:   loadConfig,
    sso:      loadSSO,
    logs:     loadLogs,
}

// Responses faster than this render straight into the view with no loading skeleton first.
const SKELETON_DELAY_MS = 150
let switchSeq = 0   // increments per tab switch
let renderSeq = 0   // increments per setContent, so a switch can tell whether its loader drew yet

// hash is "tab" or "tab/sub" (e.g. tickets/123). Admin-only tabs fall back to workflows for
// everyone else, so a stale bookmark does not open a page that can only 403.
function parseHash() {
    const [tab, ...rest] = window.location.hash.replace(/^#/, '').split('/')
    const known = tabLoaders[tab] && (!ADMIN_TABS.has(tab) || isAdmin())
    return { tab: known ? tab : 'workflows', sub: rest.join('/') || null }
}

function switchTab(tab, sub = null) {
    // A guard that needs to ask the user vetoes the move and re-runs it itself
    // once they answer, because the question is a modal and cannot block here.
    if (tabGuard && !tabGuard(() => switchTab(tab, sub))) {
        window.location.hash = currentHash
        return
    }
    tabGuard = null
    stopSyncPoll()
    stopLogsPoll()
    stopIntakePoll()
    rsStopPoll()
    currentTab  = tab
    currentHash = sub ? `${tab}/${sub}` : tab
    window.location.hash = currentHash
    document.querySelectorAll('.nav-item').forEach(el => {
        const on = el.dataset.tab === tab
        el.classList.toggle('active', on)
        if (on) el.setAttribute('aria-current', 'page'); else el.removeAttribute('aria-current')
    })
    document.getElementById('app').classList.remove('nav-open')
    setCrumbs(tab, sub)
    // The skeleton only appears when the data is slow. A fast response renders the page once,
    // so the view's entry fade runs once and the height never jumps from skeleton to content.
    // It is painted only if nothing has been rendered since this switch: a loader that draws its
    // shell early and keeps fetching (tickets, list detail) must not have that shell replaced.
    const token   = ++switchSeq
    const painted = renderSeq
    const skeletonTimer = setTimeout(() => {
        if (token === switchSeq && renderSeq === painted) setContent(skeletonPage())
    }, SKELETON_DELAY_MS)
    Promise.resolve(tabLoaders[tab](sub)).finally(() => clearTimeout(skeletonTimer))
}

function routeFromHash() {
    const { tab, sub } = parseHash()
    switchTab(tab, sub)
}

window.addEventListener('hashchange', () => {
    if (window.location.hash.replace(/^#/, '') === currentHash) return
    routeFromHash()
})

window.addEventListener('beforeunload', e => {
    if (tabGuard?.isDirty?.()) {
        e.preventDefault()
        e.returnValue = ''
    }
})

function setContent(html) {
    renderSeq++
    document.getElementById('content').innerHTML = html
}

// setCrumbs keeps the topbar trail and the document title in step with the route.
function setCrumbs(tab, sub) {
    const item   = NAV_ITEMS.find(i => i.tab === tab)
    const name   = item ? item.name : tab
    const here   = document.getElementById('crumb-here')
    const parent = document.getElementById('crumb-parent')
    if (sub) {
        parent.innerHTML = `<a href="#${tab}">${esc(name)}</a><span class="sep">/</span>`
        here.textContent = sub
    } else {
        parent.innerHTML = ''
        here.textContent = name
    }
    document.title = sub ? `${name} · ${sub} · Ticketbot` : `${name} · Ticketbot`
}

// setCrumbHere replaces the trailing crumb (the raw id from the hash) with the
// record's name once a detail page has loaded it.
function setCrumbHere(text) {
    if (!text) return
    document.getElementById('crumb-here').textContent = text
    const item = NAV_ITEMS.find(i => i.tab === currentTab)
    document.title = `${item ? item.name : currentTab} · ${text} · Ticketbot`
}

// skeletonPage is the loading state between routes: shaped like a page, so the
// layout does not jump when the real content lands.
function skeletonPage() {
    const bar = (w, h = 13) => `<div class="skeleton" style="height:${h}px;width:${w}"></div>`
    return `<div class="stack gap6" aria-busy="true" aria-label="Loading">
        <div class="stack gap3">${bar('220px', 30)}${bar('320px')}</div>
        <div class="card"><div class="stack gap4" style="padding:var(--s5)">
            ${bar('100%')}${bar('92%')}${bar('84%')}${bar('88%')}${bar('70%')}
        </div></div>
    </div>`
}

// ─────────────────────────────────────────────────────────
// Toast
// ─────────────────────────────────────────────────────────
// type is the app's own vocabulary ('success' | 'error' | 'info'); ui.css's
// variants are ok / bad / (default).
const TOAST_VARIANT = { success: 'ok', error: 'bad', info: '' }
const TOAST_TITLE   = { success: 'Done', error: 'Something went wrong', info: 'Heads up' }

function toast(msg, type = 'info') {
    const variant = TOAST_VARIANT[type] ?? ''
    const el = document.createElement('div')
    el.className = `toast ${variant}`.trim()
    el.setAttribute('role', type === 'error' ? 'alert' : 'status')
    el.innerHTML = `${icon(variant === 'bad' ? 'alert' : variant === 'ok' ? 'check' : 'info')}
        <div><b>${esc(TOAST_TITLE[type] ?? TOAST_TITLE.info)}</b><p>${esc(msg)}</p></div>`
    document.getElementById('toasts').appendChild(el)
    setTimeout(() => {
        el.classList.add('out')
        setTimeout(() => el.remove(), 220)
    }, 4000)
}

// ─────────────────────────────────────────────────────────
// Modal
// ─────────────────────────────────────────────────────────
function openModal(title, bodyHTML, submitFn, submitLabel = 'Create', variant = 'primary') {
    document.getElementById('modal-title').textContent = title
    document.getElementById('modal-body').innerHTML    = bodyHTML
    document.getElementById('modal-foot').innerHTML    = `
        <button class="btn btn-ghost" onclick="closeModal()">Cancel</button>
        <button id="modal-submit" class="btn btn-${variant}">${esc(submitLabel)}</button>`
    document.getElementById('modal-submit').addEventListener('click', handleModalSubmit)
    modalSubmitFn = submitFn
    document.getElementById('modal').classList.add('on')
    document.getElementById('scrim').classList.add('on')
    setTimeout(() => {
        const first = document.querySelector('#modal-body input, #modal-body select, #modal-body textarea')
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
    document.getElementById('modal').classList.remove('on')
    document.getElementById('scrim').classList.remove('on')
    modalSubmitFn = null
}

// confirmModal asks before something destructive or lossy.
//
// It replaces window.confirm(), which cannot be trusted here: embedded web views
// — the desktop app's browser pane among them — return false immediately without
// showing anything, so every guarded action silently did nothing.
function confirmModal({ title, body, confirmLabel = 'Delete', tone = 'bad', onConfirm }) {
    openModal(title,
        `<div class="callout ${tone}">${icon('alert')}<div class="body">${body}</div></div>`,
        async () => { closeModal(); await onConfirm() },
        confirmLabel,
        tone === 'bad' ? 'danger' : 'primary')
}

// ─────────────────────────────────────────────────────────
// Typeahead popup
//
// Search inputs offer their matches in a .typeahead-pop. The native <datalist>
// popup is drawn by the browser rather than the page, and embedded web views put it
// nowhere near the input. The popup hangs off <body> at the input's page coordinates,
// so an ancestor that clips (the condition box has overflow: hidden) cannot cut it off.
// ─────────────────────────────────────────────────────────
let taOpen = null   // { input, items, onPick, pop, cursor }

// typeaheadShow lists items ({ label, sub?, ...data }) under input; onPick gets the chosen item.
function typeaheadShow(input, items, onPick) {
    typeaheadHide()
    if (!items.length || document.activeElement !== input) return   // the user has moved on
    typeaheadBind(input)

    const pop = document.createElement('div')
    pop.className = 'typeahead-pop'
    pop.setAttribute('role', 'listbox')
    pop.innerHTML = items.map((it, k) =>
        `<button type="button" role="option" data-k="${k}">${esc(it.label)}${it.sub ? `<span class="cell-sub">${esc(it.sub)}</span>` : ''}</button>`).join('')
    // mousedown rather than click: a click would blur the input first, and blur closes the popup
    pop.addEventListener('mousedown', e => {
        const b = e.target.closest('[data-k]')
        if (!b) return
        e.preventDefault()
        typeaheadPick(Number(b.dataset.k))
    })
    document.body.appendChild(pop)

    const r = input.getBoundingClientRect()
    pop.style.minWidth = `${r.width}px`
    pop.style.top      = `${window.scrollY + r.bottom + 4}px`
    pop.style.left     = `${window.scrollX + Math.max(12, Math.min(r.left, window.innerWidth - pop.offsetWidth - 12))}px`
    taOpen = { input, items, onPick, pop, cursor: -1 }
}

function typeaheadHide() {
    taOpen?.pop.remove()
    taOpen = null
}

function typeaheadPick(k) {
    const { items, onPick } = taOpen
    typeaheadHide()
    onPick(items[k])
}

function typeaheadMove(dir) {
    const n = taOpen.items.length
    taOpen.cursor = (taOpen.cursor + dir + n) % n
    taOpen.pop.querySelectorAll('button').forEach((b, k) => b.classList.toggle('cursor', k === taOpen.cursor))
}

// typeaheadBind wires keyboard navigation and dismissal to an input, once.
function typeaheadBind(input) {
    if (input.dataset.ta) return
    input.dataset.ta = '1'
    input.addEventListener('keydown', e => {
        if (!taOpen || taOpen.input !== input) return
        switch (e.key) {
        case 'ArrowDown': e.preventDefault(); typeaheadMove(1); break
        case 'ArrowUp':   e.preventDefault(); typeaheadMove(-1); break
        case 'Enter':     if (taOpen.cursor >= 0) { e.preventDefault(); typeaheadPick(taOpen.cursor) } break
        case 'Escape':    e.stopPropagation(); typeaheadHide(); break   // closes the list, not the modal
        }
    })
    input.addEventListener('blur', typeaheadHide)
}
// the popup is pinned to page coordinates, so any scroll would leave it behind
window.addEventListener('scroll', typeaheadHide, true)

// ─────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────
// copyText writes to the clipboard and says so either way: the clipboard API rejects in
// some embedded web views, and a copy button that does nothing is worse than a message.
function copyText(text, okMsg) {
    navigator.clipboard.writeText(text)
        .then(() => toast(okMsg, 'success'))
        .catch(() => toast('Could not copy — select the text and copy it by hand', 'error'))
}

function esc(str) {
    return String(str ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')
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

// localDT turns a UTC ISO string into the local 'YYYY-MM-DDTHH:MM' a datetime-local input takes.
function localDT(iso) {
    if (!iso) return ''
    const d = new Date(iso)
    const p = n => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

// ─────────────────────────────────────────────────────────
// Render helpers (ui.css components)
// ─────────────────────────────────────────────────────────
// badgeTag renders a labelled badge. variant is a ui.css tone —
// ok | bad | warn | info | accent | outline | '' (neutral).
function badgeTag(label, variant = '') {
    const dot = ['ok', 'bad', 'warn', 'info'].includes(variant) ? '<i class="dot"></i>' : ''
    return `<span class="badge ${variant}">${dot}${esc(label)}</span>`
}

function badge(val) {
    return badgeTag(val ? 'Yes' : 'No', val ? 'ok' : 'bad')
}

// pageActions is what is left of a page head: the topbar trail names the page and
// the page itself says what it does, so all that is left is the buttons. Markup.
function pageActions(actions = '') {
    if (!actions) return ''
    return `<header class="page-head row spread wrap gap4">
        <span></span>
        <div class="row gap2 wrap">${actions}</div>
    </header>`
}

// backRow is the detail-page return path: "← Parent / this record".
function backRow(href, parent, here = '') {
    return `<div class="back-row">
        <a class="back-link" href="#${href}">${icon('arrowL')}${esc(parent)}</a>
        ${here ? `<span class="muted">/</span><span class="cell-sub num">${esc(here)}</span>` : ''}
    </div>`
}

// emptyState says what would be here and offers the action that creates it.
function emptyState(title, body = '', action = '', art = 'inbox') {
    return `<div class="empty">
        <div class="empty-art">${icon(art)}</div>
        <div class="stack gap2">
            <h3>${esc(title)}</h3>
            ${body ? `<p>${esc(body)}</p>` : ''}
        </div>
        ${action}
    </div>`
}

// errorState is what a view shows when its data could not be loaded. The default
// action re-runs the current route.
function errorState(msg, retry = `<button class="btn btn-default btn-sm" onclick="routeFromHash()">Try again</button>`) {
    return `<div class="card"><div class="empty">
        <div class="empty-art">${icon('alert')}</div>
        <div class="stack gap2">
            <h3>Could not load this page</h3>
            <p>${esc(msg)}</p>
        </div>
        ${retry}
    </div></div>`
}

// tableCard wraps a table in a card, with an empty state when there are no rows
// and an optional footer (row count, pagination).
function tableCard(thead, rows, opts = {}) {
    const { empty = emptyState('Nothing here yet', 'Items you create will show up in this table.'), foot = '', toolbar = '' } = opts
    if (!rows.length) return `<div class="card">${toolbar ? `<div class="toolbar">${toolbar}</div>` : ''}${empty}</div>`
    return `<div class="card">
        ${toolbar ? `<div class="toolbar">${toolbar}</div>` : ''}
        <div class="table-wrap"><table class="tbl">
            <thead><tr>${thead}</tr></thead>
            <tbody>${rows.join('')}</tbody>
        </table></div>
        ${foot ? `<div class="card-foot">${foot}</div>` : ''}
    </div>`
}

// deleteButton is the destructive action in a table row: a word, not a colour.
function deleteButton(onclick, label = 'Delete') {
    return editOnly(`<button class="btn btn-ghost btn-sm" onclick="${onclick}">${icon('trash')}${esc(label)}</button>`)
}

// checkbox is the styled .checkbox; the box element carries the tick.
function checkbox(label, attrs = '', checked = false) {
    return `<label class="check">
        <input type="checkbox" ${checked ? 'checked' : ''} ${attrs}>
        <span class="box">${icon('check')}</span>
        <span>${label}</span>
    </label>`
}

// toggle is the .toggle switch; .track must stay a flex box or the thumb collapses.
// A switch with no visible label takes its name from `tip`; it gets no tooltip,
// because the setting it belongs to is already named beside it.
function toggle(attrs = '', checked = false, opts = {}) {
    const { label = '', small = false, tip = '' } = opts
    return `<label class="switch${small ? ' sm' : ''}">
        <input type="checkbox" ${checked ? 'checked' : ''} ${attrs}${label ? '' : ` aria-label="${esc(tip || 'Enabled')}"`}>
        <span class="track"><span class="thumb"></span></span>
        ${label ? `<span>${label}</span>` : ''}
    </label>`
}

// ─────────────────────────────────────────────────────────
// Forwards
// ─────────────────────────────────────────────────────────
let forwardsCache = []

async function loadForwards() {
    try {
        const fwds = await api('GET', '/notifiers/forwards?filter=not-expired')
        renderForwards(fwds || [])
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function renderForwards(fwds) {
    forwardsCache = fwds

    const head = pageActions(
        editOnly(`<button class="btn btn-primary" onclick="showForwardModal()">${icon('plus')}New forward</button>`))

    const thead = `<th>Source</th><th>Destination</th><th>Dates</th><th class="c">Enabled</th>
        <th class="c">Keeps copy</th><th class="c">Sole only</th><th class="c">Public only</th><th class="r">Actions</th>`
    const rows  = fwds.map(f => `<tr>
        <td><div class="cell-primary">${esc(f.source_name)}</div><div class="cell-sub">${esc(f.source_type)}</div></td>
        <td><div class="cell-primary">${esc(f.destination_name)}</div><div class="cell-sub">${esc(f.destination_type)}</div></td>
        <td class="nowrap muted">${fmtDateRange(f.start_date, f.end_date)}</td>
        <td class="c">${badge(f.enabled)}</td>
        <td class="c">${badge(f.user_keeps_copy)}</td>
        <td class="c">${badge(f.only_if_sole_resource)}</td>
        <td class="c">${badge(f.public_only)}</td>
        <td class="r nowrap">
            <button class="btn btn-ghost btn-sm" onclick="editForward(${f.id})">${icon('edit')}Edit</button>
            ${deleteButton(`deleteForward(${f.id})`)}
        </td>
    </tr>`)

    setContent(head + tableCard(thead, rows, {
        empty: emptyState('No forwards yet',
            'A forward re-routes one person\u2019s ticket notifications to someone else while they are away.',
            editOnly(`<button class="btn btn-primary btn-sm" onclick="showForwardModal()">${icon('plus')}New forward</button>`), 'mail'),
        foot: `<span>${fwds.length} forward${fwds.length === 1 ? '' : 's'}</span>`,
    }))
}

function editForward(id) {
    const f = forwardsCache.find(x => x.id === id)
    if (!f) { toast('Forward not found — reload the tab', 'error'); return }
    showForwardModal(f)
}

async function showForwardModal(existing = null) {
    let recipients
    try {
        recipients = await api('GET', '/webex/rooms')
    } catch (e) { toast(e.message, 'error'); return }

    if (!recipients?.length) { toast('No recipients found — run a sync first', 'error'); return }

    const recipOpts = sel => recipients.map(r =>
        `<option value="${r.id}"${r.id === sel ? ' selected' : ''}>${esc(r.name)} (${esc(r.type)})</option>`).join('')

    // yes/no select where the first option listed is the one shown when adding a new forward
    const yesNo = (val, yesFirst) => {
        const opts = yesFirst ? [true, false] : [false, true]
        return opts.map(o =>
            `<option value="${o}"${o === val ? ' selected' : ''}>${o ? 'Yes' : 'No'}</option>`).join('')
    }

    // One datetime-local per bound. A separate date + time pair does not fit the modal's
    // two-column grid: native date and time inputs have a fixed minimum width, so the time
    // box was pushed past the edge of the dialog.
    const startLocal = localDT(existing?.start_date)
    const endLocal   = localDT(existing?.end_date)

    openModal(existing ? 'Edit forward' : 'New forward', `
        <div class="stack gap4">
            <div class="field">
                <label for="f-source">Source</label>
                <select class="select" id="f-source">${recipOpts(existing?.source_id)}</select>
                <span class="hint">Whose notifications are forwarded.</span>
            </div>
            <div class="field">
                <label for="f-dest">Destination</label>
                <select class="select" id="f-dest">${recipOpts(existing?.destination_id)}</select>
            </div>
            <div class="grid g2" style="gap:var(--s3)">
                <div class="field">
                    <label for="f-start">Start <span class="muted">(optional)</span></label>
                    <input class="input" type="datetime-local" id="f-start" value="${startLocal}">
                </div>
                <div class="field">
                    <label for="f-end">End <span class="muted">(optional)</span></label>
                    <input class="input" type="datetime-local" id="f-end" value="${endLocal}">
                </div>
            </div>
            <div class="grid g2" style="gap:var(--s3)">
                <div class="field">
                    <label for="f-enabled">Enabled</label>
                    <select class="select" id="f-enabled">${yesNo(existing ? existing.enabled : true, true)}</select>
                </div>
                <div class="field">
                    <label for="f-keep">Source keeps a copy</label>
                    <select class="select" id="f-keep">${yesNo(existing ? existing.user_keeps_copy : true, true)}</select>
                </div>
                <div class="field">
                    <label for="f-sole">Only if sole resource</label>
                    <select class="select" id="f-sole">${yesNo(existing ? existing.only_if_sole_resource : false, false)}</select>
                </div>
                <div class="field">
                    <label for="f-public">Public notes only</label>
                    <select class="select" id="f-public">${yesNo(existing ? existing.public_only : false, false)}</select>
                </div>
            </div>
        </div>`, async () => {
        const sourceId  = parseInt(document.getElementById('f-source').value)
        const destId    = parseInt(document.getElementById('f-dest').value)
        const startDT   = document.getElementById('f-start').value   // 'YYYY-MM-DDTHH:MM' or ''
        const endDT     = document.getElementById('f-end').value
        const enabled   = document.getElementById('f-enabled').value === 'true'
        const keepCopy  = document.getElementById('f-keep').value === 'true'
        const soleResource = document.getElementById('f-sole').value === 'true'
        const publicOnly   = document.getElementById('f-public').value === 'true'

        if (sourceId === destId) { toast('Source and destination must be different', 'error'); return }
        if (startDT && endDT && endDT <= startDT) { toast('End must be after start', 'error'); return }

        const payload = {
            user_email:      sourceId,
            dest_email:      destId,
            enabled:         enabled,
            user_keeps_copy: keepCopy,
            only_if_sole_resource: soleResource,
            public_only:     publicOnly,
        }
        if (startDT) payload.start_date = new Date(startDT).toISOString()
        if (endDT)   payload.end_date   = new Date(endDT).toISOString()

        try {
            if (existing) {
                await api('PUT', `/notifiers/forwards/${existing.id}`, payload)
            } else {
                await api('POST', '/notifiers/forwards', payload)
            }
            closeModal()
            toast(existing ? 'Forward updated' : 'Forward created', 'success')
            loadForwards()
        } catch (e) { toast(e.message, 'error') }
    }, existing ? 'Save' : 'Create')
}

function deleteForward(id) {
    const f = forwardsCache.find(x => x.id === id)
    confirmModal({
        title: 'Delete this forward?',
        body: f
            ? `<b>${esc(f.source_name)} → ${esc(f.destination_name)}</b>Notifications stop being forwarded as soon as this is deleted.`
            : 'Notifications stop being forwarded as soon as this is deleted.',
        confirmLabel: 'Delete forward',
        onConfirm: async () => {
            try {
                await api('DELETE', `/notifiers/forwards/${id}`)
                toast('Forward deleted', 'success')
                loadForwards()
            } catch (e) { toast(e.message, 'error') }
        },
    })
}

// ─────────────────────────────────────────────────────────
// Users
// ─────────────────────────────────────────────────────────
async function loadUsers() {
    try {
        const users = await api('GET', '/users')
        renderUsers(users || [])
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function renderUsers(users) {
    const head = pageActions(`<button class="btn btn-primary" onclick="showNewUserModal()">${icon('plus')}New user</button>`)

    const thead = '<th class="r">ID</th><th>Email</th><th>Role</th><th>Created</th><th class="r">Actions</th>'
    const rows  = users.map(u => `<tr>
        <td class="r num muted">${u.id}</td>
        <td>
            <div class="row gap3">
                <span class="avatar sm">${esc(emailInitials(u.email_address))}</span>
                <div>
                    <div class="cell-primary">${esc(u.email_address)}</div>
                    ${u.id === currentUser?.id ? '<div class="cell-sub">Signed in as this account</div>' : ''}
                </div>
            </div>
        </td>
        <td>${roleCell(u)}</td>
        <td class="muted nowrap">${fmtDateTime(u.created_on)}</td>
        <td class="r nowrap">${u.id === currentUser?.id
            ? '<span class="badge outline">You</span>'
            : u.break_glass
                ? `<span class="badge outline" data-tip="Set by INITIAL_ADMIN_EMAIL. Always able to sign in with a password, so it cannot be deleted.">Break-glass</span>`
                : deleteButton(`deleteUser(${u.id})`)}</td>
    </tr>`)

    setContent(head + tableCard(thead, rows, {
        empty: emptyState('No users yet', 'Create a user so someone can sign in.',
            `<button class="btn btn-primary btn-sm" onclick="showNewUserModal()">${icon('plus')}New user</button>`, 'users'),
        foot: `<span>${users.length} user${users.length === 1 ? '' : 's'}</span>`,
    }))
}

const ROLES = [
    { value: 'viewer', label: 'Viewer', desc: 'Read workflows, tickets, forwards and lists' },
    { value: 'editor', label: 'Editor', desc: 'Also create and change workflows, forwards and lists' },
    { value: 'admin',  label: 'Admin',  desc: 'Also manage users, keys, config, sync and SSO' },
]

function roleOptions(selected) {
    return ROLES.map(r => `<option value="${r.value}" ${r.value === selected ? 'selected' : ''}>${r.label}</option>`).join('')
}

// roleCell is the role control in a user row. Your own role and Entra-managed roles are shown,
// not edited: the server refuses both.
function roleCell(u) {
    const name = ROLES.find(r => r.value === u.role)?.label || esc(u.role || '—')
    if (u.sso) return `<div class="row gap2 wrap"><span class="badge">${name}</span><span class="badge info" data-tip="Role comes from an Entra app role on every sign-in"><i class="dot"></i>Entra</span></div>`
    if (u.id === currentUser?.id || u.break_glass) return `<span class="badge">${name}</span>`
    return `<select class="select" aria-label="Role for ${esc(u.email_address)}" onchange="setUserRole(${u.id}, this.value, this)">${roleOptions(u.role)}</select>`
}

async function setUserRole(id, role, el) {
    el.disabled = true
    try {
        await api('PUT', `/users/${id}/role`, { role })
        toast('Role updated', 'success')
    } catch (e) {
        toast(e.message, 'error')
        loadUsers()
    } finally { el.disabled = false }
}

function showNewUserModal() {
    openModal('New user', `
        <div class="stack gap4">
            <div class="field">
                <label for="f-email">Email address</label>
                <input class="input" type="email" id="f-email" placeholder="user@example.com">
            </div>
            <div class="field">
                <label for="f-role">Role</label>
                <select class="select" id="f-role">${roleOptions('viewer')}</select>
                <span class="hint" id="f-role-hint">${ROLES[0].desc}</span>
            </div>
            <div class="field">
                <label for="f-temp-password">Temporary password</label>
                <input class="input" type="password" id="f-temp-password" placeholder="••••••••">
                <span class="hint">The user must change this on first sign-in.</span>
            </div>
        </div>`, async () => {
        const email    = document.getElementById('f-email').value.trim()
        const password = document.getElementById('f-temp-password').value
        const role     = document.getElementById('f-role').value
        if (!email)     { toast('Email is required', 'error'); return }
        if (!password)  { toast('Temporary password is required', 'error'); return }
        try {
            await api('POST', '/users', { email_address: email, password, role })
            closeModal()
            toast('User created — they must change their password on first login', 'success')
            loadUsers()
        } catch (e) { toast(e.message, 'error') }
    })
    document.getElementById('f-role').addEventListener('change', e => {
        document.getElementById('f-role-hint').textContent = ROLES.find(r => r.value === e.target.value)?.desc || ''
    })
}

function deleteUser(id) {
    confirmModal({
        title: 'Delete this user?',
        body: '<b>They lose access immediately</b>Their API keys are deleted with them, so anything using one stops working.',
        confirmLabel: 'Delete user',
        onConfirm: async () => {
            try {
                await api('DELETE', `/users/${id}`)
                toast('User deleted', 'success')
                loadUsers()
            } catch (e) { toast(e.message, 'error') }
        },
    })
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
        setContent(errorState(e.message))
    }
}

function renderKeys(keys, users) {
    const userMap = {}
    users.forEach(u => { userMap[u.id] = u.email_address })

    const head = pageActions(`<button class="btn btn-primary" onclick="showNewKeyModal()">${icon('plus')}New key</button>`)

    const thead = '<th class="r">ID</th><th>User</th><th>Key</th><th>Created</th><th class="r">Actions</th>'
    const rows  = keys.map(k => `<tr>
        <td class="r num muted">${k.id}</td>
        <td class="cell-primary">${esc(userMap[k.user_id] || `User #${k.user_id}`)}</td>
        <td class="num muted">${k.key_hint ? `••••${esc(k.key_hint)}` : '—'}</td>
        <td class="muted nowrap">${fmtDateTime(k.created_on)}</td>
        <td class="r nowrap">${deleteButton(`deleteKey(${k.id})`, 'Revoke')}</td>
    </tr>`)

    setContent(head + tableCard(thead, rows, {
        empty: emptyState('No API keys', 'Create a key to let a script or integration call the ticketbot API.',
            `<button class="btn btn-primary btn-sm" onclick="showNewKeyModal()">${icon('plus')}New key</button>`, 'key'),
        foot: `<span>${keys.length} key${keys.length === 1 ? '' : 's'}</span>`,
    }))
}

async function showNewKeyModal() {
    let users
    try {
        users = await api('GET', '/users')
    } catch (e) { toast(e.message, 'error'); return }

    if (!users?.length) { toast('No users found — create a user first', 'error'); return }

    const userOpts = users.map(u =>
        `<option value="${esc(u.email_address)}">${esc(u.email_address)}</option>`).join('')

    openModal('New API key', `
        <div class="field">
            <label for="f-user-email">User</label>
            <select class="select" id="f-user-email">${userOpts}</select>
            <span class="hint">The key acts as this user.</span>
        </div>`, async () => {
        const email = document.getElementById('f-user-email').value
        try {
            const res = await api('POST', '/users/keys', { email })
            // Replace modal with key display — key is only shown once
            document.getElementById('modal-body').innerHTML = `
                <div class="stack gap4">
                    <div class="callout warn">${icon('alert')}<div class="body">
                        <b>Copy this key now</b>It is not stored in full and will not be shown again.
                    </div></div>
                    <div class="secret reveal" id="created-key">${esc(res.key)}</div>
                </div>`
            document.getElementById('modal-foot').innerHTML = `
                <button class="btn btn-ghost" onclick="copyCreatedKey()">${icon('copy')}Copy key</button>
                <button class="btn btn-primary" onclick="closeModal(); loadKeys()">Done</button>`
            modalSubmitFn = null
        } catch (e) { toast(e.message, 'error') }
    })
}

function copyCreatedKey() {
    const key = document.getElementById('created-key')?.textContent
    if (key) copyText(key, 'Copied!')
}

function deleteKey(id) {
    confirmModal({
        title: 'Revoke this API key?',
        body: '<b>This cannot be undone</b>Any script or integration still sending this key starts getting 401s.',
        confirmLabel: 'Revoke key',
        onConfirm: async () => {
            try {
                await api('DELETE', `/users/keys/${id}`)
                toast('Key revoked', 'success')
                loadKeys()
            } catch (e) { toast(e.message, 'error') }
        },
    })
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
        setContent(errorState(e.message))
    }
}

function renderSync(status) {
    const running = status?.status === true

    setContent(pageActions(
        // while a sync runs the button is disabled, so it drops the accent: a dimmed
        // accent fill does not hold its contrast
        `<button class="btn ${running ? 'btn-default' : 'btn-primary'}" onclick="showNewSyncModal()" ${running ? 'disabled' : ''}>${icon('globe')}Run sync</button>`) +
    `<div class="card card-pad">
        <div class="row gap3 wrap">
            ${running
                ? '<span class="badge ok"><i class="dot pulse"></i>Running</span>'
                : '<span class="badge outline"><i class="dot"></i>Idle</span>'}
            <span class="muted">${running
                ? 'A sync is in progress. This page updates every few seconds.'
                : 'No sync is running right now.'}</span>
        </div>
    </div>`)
}

async function showNewSyncModal() {
    let boards = []
    try { boards = await api('GET', '/cw/boards') ?? [] } catch { /* show modal without boards */ }

    const boardCheckboxes = boards.map(b =>
        checkbox(esc(b.name), `name="board" value="${b.id}"`)
    ).join('')

    openModal('Run sync', `
        <div class="stack gap4">
            <div class="field">
                <label>What to sync</label>
                <div class="stack gap2" style="margin-top:2px">
                    ${checkbox('Boards', 'id="f-sync-boards"', true)}
                    ${checkbox('Webex recipients', 'id="f-sync-webex"', true)}
                    ${checkbox('Tickets', 'id="f-sync-tickets"')}
                </div>
            </div>
            ${boards.length ? `<div class="field">
                <label>Board filter <span class="muted">(none selected = all boards)</span></label>
                <div class="stack gap2" style="max-height:220px;overflow-y:auto;margin-top:2px">${boardCheckboxes}</div>
            </div>` : ''}
        </div>`, async () => {
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
// ─────────────────────────────────────────────────────────
// Single sign-on (Microsoft Entra)
// ─────────────────────────────────────────────────────────
async function loadSSO() {
    try {
        const st = await api('GET', '/sso')
        renderSSO(st)
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function renderSSO(st) {
    const row = (title, desc, control) => `<div class="form-row">
        <div><h4>${title}</h4><p class="desc">${desc}</p></div>
        <div class="row gap3 wrap">${control}</div>
    </div>`

    const configured = st.configured
        ? badgeTag('Configured', 'ok')
        : badgeTag('Not configured', 'warn')

    const mappingRows = (st.mappings || []).map(m => `<tr>
        <td><code class="code inline">${esc(m.entra_role)}</code></td>
        <td><select class="select" aria-label="Ticketbot role for ${esc(m.entra_role)}" onchange="ssoSaveMapping(${JSON.stringify(m.entra_role)}, this.value, this)">${roleOptions(m.role)}</select></td>
        <td class="r nowrap">${deleteButton(`ssoDeleteMapping(${m.id})`, 'Remove')}</td>
    </tr>`)

    setContent(
    `<div class="stack gap5">
        <div class="card card-pad">
            ${row('Credentials',
                'Tenant, client ID and client secret come from the ENTRA_* variables in .env and need a restart to change.',
                `<div class="stack gap2">
                    ${configured}
                    ${st.configured ? `<span class="muted" style="font-size:var(--text-xs)">Tenant <code class="code inline">${esc(st.tenant_id)}</code> · Client <code class="code inline">${esc(st.client_id)}</code></span>` : ''}
                </div>`)}
            ${row('Redirect URI',
                'Register this exact value under Authentication → Web on the app registration.',
                `<div class="row gap2 wrap"><code class="code inline" id="sso-redirect">${esc(st.redirect_uri)}</code>
                    <button class="btn btn-ghost btn-sm" onclick="copyText(document.getElementById('sso-redirect').textContent, 'Copied')">${icon('copy')}Copy</button></div>`)}
            ${row('Connection',
                'Runs discovery and checks Microsoft accepts the client secret. It cannot check the redirect URI or role assignments; only a real sign-in does.',
                `<div class="stack gap2">
                    <div><button class="btn btn-default" onclick="ssoTest()" ${st.configured ? '' : 'disabled'}>${icon('play')}Test connection</button></div>
                    <div id="sso-test-result"></div>
                </div>`)}
            ${row('Enabled',
                'Show “Sign in with Microsoft” on the login card. Requires credentials.',
                toggle(`id="sso-enabled" onchange="ssoSaveToggles()" ${st.configured ? '' : 'disabled'}`, st.enabled, { tip: 'SSO enabled' }))}
            ${row('Password sign-in',
                'Allow email and password sign-in alongside Microsoft. Can only be turned off while SSO is on. The INITIAL_ADMIN_EMAIL account can always use a password through the “Sign in with a password instead” link on the login card.',
                toggle(`id="sso-password-login" onchange="ssoSaveToggles()"`, st.password_login_enabled, { tip: 'Password sign-in' }))}
        </div>

        <div class="section-head"><div><h3>Role mappings</h3><p class="muted">Entra app role value → ticketbot role. A person gets the highest role any of their app roles maps to; with no match, sign-in is refused.</p></div></div>
        ${tableCard('<th>Entra app role</th><th>Ticketbot role</th><th class="r">Actions</th>', mappingRows, {
            toolbar: `<div class="row gap2 wrap grow">
                <input class="input" id="sso-new-role" placeholder="e.g. TicketBot.Admin" aria-label="Entra app role value" style="max-width:260px">
                <select class="select" id="sso-new-map" aria-label="Ticketbot role" style="max-width:160px">${roleOptions('viewer')}</select>
                <button class="btn btn-default" onclick="ssoAddMapping()">${icon('plus')}Add mapping</button>
            </div>`,
            empty: emptyState('No role mappings', 'Nobody can sign in with Microsoft until at least one Entra app role maps to a ticketbot role.', '', 'shield'),
        })}

        <div class="card card-pad stack gap4">
            <div><h3>Setup</h3><p class="muted">Five steps in the Entra admin centre.</p></div>
            <ol class="steps stack gap2">
                <li><b>Register the app.</b> App registrations → New registration, single tenant. Authentication → Add a platform → <b>Web</b>, redirect URI as shown above.</li>
                <li><b>Copy the IDs and a secret.</b> Overview gives the Directory (tenant) ID and Application (client) ID; Certificates &amp; secrets → New client secret. Put all three in <code class="code inline">.env</code> as <code class="code inline">ENTRA_TENANT_ID</code>, <code class="code inline">ENTRA_CLIENT_ID</code>, <code class="code inline">ENTRA_CLIENT_SECRET</code> and restart.</li>
                <li><b>Create app roles.</b> App roles → Create app role, one per ticketbot role you need, allowed member type Users/Groups. The <em>value</em> is what you map below.</li>
                <li><b>Assign people.</b> Enterprise applications → this app → Users and groups → add users or groups to those roles.</li>
                <li><b>Map and enable.</b> Add the mappings above, run Test connection, then turn Enabled on.</li>
            </ol>
            <div class="callout warn">${icon('alert')}<div class="body"><b>Client secrets expire</b>When the secret lapses every Microsoft sign-in fails until a new one is set in .env.</div></div>
        </div>
    </div>`)
}

async function ssoTest() {
    const out = document.getElementById('sso-test-result')
    out.innerHTML = '<span class="muted">Testing…</span>'
    try {
        const res = await api('POST', '/sso/test')
        out.innerHTML = res.ok
            ? `<div class="callout ok">${icon('check')}<div class="body"><b>Connected</b>Microsoft accepted the client secret.</div></div>`
            : `<div class="callout bad">${icon('alert')}<div class="body"><b>Connection failed</b>${esc(res.error)}</div></div>`
    } catch (e) {
        out.innerHTML = `<div class="callout bad">${icon('alert')}<div class="body"><b>Connection failed</b>${esc(e.message)}</div></div>`
    }
}

async function ssoSaveToggles() {
    const enabled  = document.getElementById('sso-enabled').checked
    const password = document.getElementById('sso-password-login').checked
    try {
        const res = await api('PUT', '/config', { sso_enabled: enabled, password_login_enabled: password })
        if (res) appConfig = res
        toast('Sign-in settings saved', 'success')
    } catch (e) {
        toast(e.message, 'error')
        loadSSO()
    }
}

async function ssoSaveMapping(entraRole, role, el) {
    if (el) el.disabled = true
    try {
        await api('PUT', '/sso/mappings', { entra_role: entraRole, role })
        toast('Mapping saved', 'success')
        if (!el) loadSSO()
    } catch (e) {
        toast(e.message, 'error')
        loadSSO()
    } finally { if (el) el.disabled = false }
}

function ssoAddMapping() {
    const entraRole = document.getElementById('sso-new-role').value.trim()
    const role      = document.getElementById('sso-new-map').value
    if (!entraRole) { toast('Enter the Entra app role value', 'error'); return }
    ssoSaveMapping(entraRole, role, null)
}

function ssoDeleteMapping(id) {
    confirmModal({
        title: 'Remove this mapping?',
        body: '<b>People holding only this app role lose access</b>Their next Microsoft sign-in is refused. Existing sessions run until they expire.',
        confirmLabel: 'Remove mapping',
        onConfirm: async () => {
            try {
                await api('DELETE', `/sso/mappings/${id}`)
                toast('Mapping removed', 'success')
                loadSSO()
            } catch (e) { toast(e.message, 'error') }
        },
    })
}

async function loadConfig() {
    try {
        const [cfg, recipients] = await Promise.all([
            api('GET', '/config'),
            api('GET', '/webex/rooms').catch(() => []),
        ])
        renderConfig(cfg, recipients || [])
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function renderConfig(cfg, recipients = []) {
    // room pickers: "None" plus every synced room or person; the current value stays selectable
    // even if the recipient list failed to load
    const roomSelect = (id, value, label) => {
        const opts = recipients.map(r => `<option value="${r.id}"${r.id === value ? ' selected' : ''}>${esc(r.name)} (${esc(r.type)})</option>`)
        if (value && !recipients.some(r => r.id === value)) opts.unshift(`<option value="${value}" selected>Recipient #${value}</option>`)
        return `<select class="select" style="max-width:320px" id="${id}" aria-label="${esc(label)}"><option value="0"${value ? '' : ' selected'}>None</option>${opts.join('')}</select>`
    }

    // one settings row: description on the left, control on the right
    const row = (title, desc, control) => `<div class="form-row">
        <div><h4>${title}</h4><p class="desc">${desc}</p></div>
        <div class="row gap3 wrap">${control}</div>
    </div>`

    const numberInput = (id, value, min) =>
        `<input class="input" style="max-width:160px" type="number" id="${id}" value="${esc(value)}" min="${min}" aria-label="${esc(id)}">`

    setContent(pageActions(`<button class="btn btn-primary" onclick="saveConfig()">Save changes</button>`) +
    `<div class="card card-pad">
        ${row('Master dry run',
            'Run every workflow as a dry run: no ConnectWise writes, Webex messages are mocked.',
            toggle(`id="c-master-dry-run"`, cfg.master_dry_run, { tip: 'Master dry run' }))}
        ${row('CW API member identifier',
            'ConnectWise member the API key belongs to; its own updates never trigger workflows. Auto-filled after the first note ticketbot posts.',
            `<input class="input" style="max-width:260px" type="text" id="c-api-member" value="${esc(cfg.cw_api_member_identifier || '')}" placeholder="e.g. ticketbot" aria-label="CW API member identifier">`)}
        ${row('Max message length',
            'Truncation limit for note content in Webex notifications, in characters.',
            numberInput('c-max-len', cfg.max_message_length, 1))}
        ${row('Note preview length',
            'How many characters of a new note the ticket history keeps. Applies to events recorded from now on.',
            numberInput('c-note-preview', cfg.note_preview_length, 1))}
        ${row('Write cap per ticket',
            'How many ConnectWise writes workflows may make to one ticket in 15 minutes before that ticket is blocked for an hour and the ops room is alerted. A normal run is one write plus one per note. 0 disables the cap.',
            numberInput('c-write-cap', cfg.write_cap_per_ticket, 0))}
        ${row('Ops room',
            'Webex room or person that receives operator alerts: failed webhooks, write-cap blocks and webhook staleness. With none set, alerts only go to the log.',
            roomSelect('c-ops-room', cfg.ops_room_id, 'Ops room'))}
        ${row('Redirect notifications to',
            'While set, every notification goes to this room instead of its intended recipient, prefixed with who it was for, and is sent even under dry run. Use it for the parallel run; clear it at cutover.',
            roomSelect('c-redirect-room', cfg.redirect_room_id, 'Redirect room'))}
        ${row('Stale webhook alert',
            'Minutes without a ticket webhook, during business hours (7:30am to 7pm Central, weekdays), before the ops room is alerted. 0 disables the check.',
            numberInput('c-stale-minutes', cfg.stale_alert_minutes, 0))}
        ${row('Max concurrent syncs',
            'Limits parallel requests to ConnectWise.',
            numberInput('c-max-syncs', cfg.max_concurrent_syncs, 1))}
        ${row('Require 2FA',
            'All users must set up two-factor authentication to access the app.',
            toggle(`id="c-require-totp"`, cfg.require_totp, { tip: 'Require 2FA' }))}
        ${row('Debug logging',
            'Enable debug-level log output without a server restart.',
            toggle(`id="c-debug-logging"`, cfg.debug_logging, { tip: 'Debug logging' }))}
        ${row('Log buffer size',
            'Maximum log entries held in memory for the web panel.',
            numberInput('c-log-buffer-size', cfg.log_buffer_size, 100))}
        ${row('History retention',
            'How many days of workflow run results and ticket history to keep. They are the same records seen two ways, so they age out together. 0 keeps them forever.',
            numberInput('c-history-retention', cfg.history_retention_days, 0))}
        ${row('Log retention',
            'How many days of logs to keep in the database. 0 keeps them forever.',
            numberInput('c-log-retention', cfg.log_retention_days, 0))}
        ${row('Log cleanup interval',
            'How often old logs are deleted, in hours.',
            numberInput('c-log-cleanup-interval', cfg.log_cleanup_interval_hours, 1))}
    </div>`)
}

async function saveConfig() {
    // every numeric field must be a whole number within its input's min; say which one is wrong
    const num = (id, label) => {
        const el = document.getElementById(id)
        const n  = Number(el.value)
        const min = el.min === '' ? -Infinity : Number(el.min)
        if (el.value.trim() === '' || !Number.isInteger(n) || n < min) {
            throw new Error(`${label} must be a whole number${min > -Infinity ? ` of at least ${min}` : ''}`)
        }
        return n
    }
    try {
        const res = await api('PUT', '/config', {
            master_dry_run:             document.getElementById('c-master-dry-run').checked,
            cw_api_member_identifier:   document.getElementById('c-api-member').value.trim(),
            max_message_length:         num('c-max-len', 'Max message length'),
            note_preview_length:        num('c-note-preview', 'Note preview length'),
            max_concurrent_syncs:       num('c-max-syncs', 'Max concurrent syncs'),
            write_cap_per_ticket:       num('c-write-cap', 'Write cap per ticket'),
            ops_room_id:                Number(document.getElementById('c-ops-room').value),
            redirect_room_id:           Number(document.getElementById('c-redirect-room').value),
            stale_alert_minutes:        num('c-stale-minutes', 'Stale webhook alert'),
            history_retention_days:     num('c-history-retention', 'History retention'),
            require_totp:               document.getElementById('c-require-totp').checked,
            debug_logging:              document.getElementById('c-debug-logging').checked,
            log_buffer_size:            num('c-log-buffer-size', 'Log buffer size'),
            log_retention_days:         num('c-log-retention', 'Log retention'),
            log_cleanup_interval_hours: num('c-log-cleanup-interval', 'Log cleanup interval'),
        })
        if (res) appConfig = res
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
let logsHideGin        = _lp.hideGin         ?? true   // request lines swamp the buffer; opt in to see them
let logsSearch         = ''

async function loadLogs() {
    try {
        const entries = await api('GET', '/logs')
        logsLastEntries = entries || []
        renderLogs(logsLastEntries)
        startLogsPoll()
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function renderLogs(entries, full = false) {
    const levelOpts = ['ALL', 'DEBUG', 'INFO', 'WARN', 'ERROR'].map(l =>
        `<option value="${l}" ${l === logsLevelFilter ? 'selected' : ''}>${l}</option>`
    ).join('')

    const contextDefs = [
        { value: 'ALL',      label: 'All' },
        { value: 'TICKET',   label: 'Ticket processing' },
        { value: 'NOTIF',    label: 'Notifications' },
        { value: 'SYNC',     label: 'Sync' },
        { value: 'AUTH',     label: 'Auth & users' },
    ]
    const contextOpts = contextDefs.map(c =>
        `<option value="${c.value}" ${c.value === logsContextFilter ? 'selected' : ''}>${c.label}</option>`
    ).join('')

    let filtered = logsLevelFilter === 'ALL'
        ? entries
        : entries.filter(e => e.level.toUpperCase() === logsLevelFilter)

    const contextMatch = {
        TICKET: e => /^(ticketbot|workflow):/.test(e.message || '') || e.attrs?.ticket_id !== undefined,
        NOTIF:  e => /^notifier:/.test(e.message || ''),
        SYNC:   e => /sync/i.test(e.message || ''),
        AUTH:   e => /^(totp|login|user|auth)/i.test(e.message || ''),
    }[logsContextFilter]
    if (contextMatch) filtered = filtered.filter(contextMatch)

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
        const cls   = lvl === 'ERROR' ? 'error' : lvl === 'WARN' ? 'warn' : lvl === 'DEBUG' ? 'debug' : ''
        const time  = e.time ? new Date(e.time).toLocaleTimeString() : '—'
        const attrs = e.attrs ? Object.entries(e.attrs).map(([k, v]) => {
            const val = (v !== null && typeof v === 'object') ? JSON.stringify(v) : String(v)
            return `<span class="log-attr"><b>${esc(k)}</b>=${esc(val)}</span>`
        }).join('') : ''
        return `<div class="log-row ${cls}">
            <span class="log-time">${time}</span>
            <span class="log-level">${esc(lvl)}</span>
            <span class="log-msg">${esc(e.message)}${attrs}</span>
        </div>`
    })

    const isEmpty = rows.length === 0
    const body    = isEmpty
        ? emptyState('No matching log entries',
            'Nothing in the buffer matches these filters. Widen the level or clear the search.',
            `<button class="btn btn-default btn-sm" onclick="resetLogsFilters()">Clear filters</button>`, 'book')
        : `<div class="log-list">${rows.join('')}</div>
           <div class="card-foot"><span>${rows.length} of ${entries.length} entries</span></div>`
    const status  = logsFrozen
        ? `<span class="badge outline"><i class="dot"></i>Frozen</span>
           <button class="btn btn-default btn-sm" onclick="toggleLogFreeze()">Resume</button>`
        : `<span class="badge ok"><i class="dot pulse"></i>Streaming</span>
           <button class="btn btn-default btn-sm" onclick="toggleLogFreeze()">Freeze</button>`

    // The poll re-renders every few seconds. Redrawing the filter bar with it would close an
    // open dropdown or popover under the user's cursor, so once the frame exists only the
    // list and the streaming status are replaced; `full` forces a redraw after a filter reset.
    if (!full && document.getElementById('logs-frame')) {
        document.getElementById('logs-status').innerHTML = status
        document.getElementById('logs-body').innerHTML   = body
        return
    }

    setContent(
    `<div class="card" id="logs-frame">
        <div class="filter-bar">
            <select class="select" id="logs-level-filter" onchange="setLogsFilter(this.value)" aria-label="Filter by level">${levelOpts}</select>
            <select class="select" id="logs-context-filter" onchange="setLogsContextFilter(this.value)" aria-label="Filter by area">${contextOpts}</select>
            <div class="input-group">
                ${icon('search')}
                <input class="input" id="logs-search" type="text" style="width:220px" placeholder="Search…" value="${esc(logsSearch)}" oninput="setLogsSearch(this.value)" aria-label="Search log messages">
            </div>
            <button class="btn btn-default btn-sm" onclick="openLogsOptions(event)">${icon('filter')}Options</button>
            <div class="grow"></div>
            <span id="logs-status" class="row gap2">${status}</span>
            <button class="btn btn-default btn-sm" onclick="loadLogs()">Refresh</button>
        </div>
        <div id="logs-body">${body}</div>
    </div>`)
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

function openLogsOptions(e) {
    e.stopPropagation()
    toggleMenu(e.currentTarget, [
        {
            label: logsHideGin ? 'Show request logs' : 'Hide request logs',
            icon: logsHideGin ? 'globe' : 'filter',
            run: () => setLogsHideGin(!logsHideGin),
        },
    ])
}

function resetLogsFilters() {
    logsSearch = ''
    logsLevelFilter = 'ALL'
    logsContextFilter = 'ALL'
    saveLogsPrefs({ levelFilter: 'ALL', contextFilter: 'ALL' })
    renderLogs(logsLastEntries, true)   // the selects and search box must show the reset values
}

function toggleLogFreeze() {
    logsFrozen = !logsFrozen
    if (logsFrozen) stopLogsPoll()
    else loadLogs()
    renderLogs(logsLastEntries)
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
        if (e.key !== 'Escape') return
        if (menuEl) { closeMenu(); return }
        closeModal()
        document.getElementById('app').classList.remove('nav-open')
    })
    // a click anywhere else dismisses an open popover or the mobile nav drawer
    document.addEventListener('click', e => {
        if (menuEl && !menuEl.contains(e.target) && !menuAnchor?.contains(e.target)) closeMenu()
        const app = document.getElementById('app')
        if (app.classList.contains('nav-open') && !e.target.closest('.sidebar')) app.classList.remove('nav-open')
    }, true)

    buildNav()
    applyTheme(document.documentElement.dataset.theme || 'light')
    checkSavedKey()
})
