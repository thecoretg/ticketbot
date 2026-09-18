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
    openMenu(e.currentTarget, PALETTES.map(p => ({
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
        { tab: 'config', name: 'Config', icon: 'cog' },
        { tab: 'logs',   name: 'Logs',   icon: 'book' },
    ]},
]
const NAV_ITEMS = NAV.flatMap(g => g.items)

function buildNav() {
    document.getElementById('nav').innerHTML = NAV.map(g => `
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

// openMenu renders a navi popover anchored under a control. Items are
// { label, icon | swatch, danger, current, run }.
let menuEl = null

function closeMenu() {
    menuEl?.remove()
    menuEl = null
}

function openMenu(anchor, items) {
    closeMenu()
    menuEl = document.createElement('div')
    menuEl.className = 'menu'
    menuEl.setAttribute('role', 'menu')
    menuEl.innerHTML = items.map((it, k) => it === '-' ? '<hr>' : `
        <button role="menuitem" data-mi="${k}" class="${it.danger ? 'danger' : ''}">
            ${it.swatch ? `<span class="swatch" style="background:${it.swatch}"></span>` : (it.icon ? icon(it.icon) : '')}
            <span>${esc(it.label)}</span>
            ${it.current ? '<span class="check-mark">●</span>' : ''}
        </button>`).join('')
    document.body.appendChild(menuEl)

    const r = anchor.getBoundingClientRect()
    const below = window.scrollY + r.bottom + 6
    const above = window.scrollY + r.top - menuEl.offsetHeight - 6
    // anchored controls near the bottom of the frame (the sidebar foot) flip upwards
    menuEl.style.top  = `${r.bottom + menuEl.offsetHeight + 12 > window.innerHeight ? above : below}px`
    menuEl.style.left = `${Math.max(12, Math.min(r.left, window.innerWidth - menuEl.offsetWidth - 12))}px`

    menuEl.addEventListener('click', e => {
        const btn = e.target.closest('[data-mi]')
        if (!btn) return
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
    tabGuard = null
    currentUser = null
    totpSetupRequired = false
    closeModal()
    document.getElementById('app').classList.add('hidden')
    document.getElementById('login').classList.remove('hidden')
    showLoginErr('Your session expired. Sign in again to continue.')
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
            await showApp()
            if (res?.totp_setup_required) showTOTPSetupModal(true)
        }
    } catch (e) {
        // 401 is a bad password; anything else (server down, 500) deserves its real message
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

async function logout() {
    try { await api('POST', '/auth/logout') } catch {}
    currentUser  = null
    pendingToken = null
    totpEnabled  = false
    stopSyncPoll()
    document.getElementById('login-email').value    = ''
    document.getElementById('login-password').value = ''
    document.getElementById('login-err').classList.add('hidden')
    closeMenu()
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
        appConfig   = cfg
        requireTOTP = cfg.require_totp
        document.getElementById('header-email').textContent    = currentUser.email_address
        document.getElementById('header-initials').textContent = emailInitials(currentUser.email_address)
        if (requireTOTP && !totpEnabled) {
            showTOTPSetupModal(true)
            return
        }
    } catch {}
    routeFromHash()
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
    if (menuEl) { closeMenu(); return }
    openMenu(e.currentTarget, [
        { label: 'Change password', icon: 'key',    run: showChangePasswordModal },
        { label: totpMenuLabel(),   icon: 'shield', run: handleTOTPMenuClick },
        '-',
        { label: 'Restart server', icon: 'bolt',   danger: true, run: confirmRestart },
        { label: 'Log out',        icon: 'logout', danger: true, run: logout },
    ])
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
    navigator.clipboard.writeText(codes).then(() => toast('Recovery codes copied', 'success'))
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
        .catch(() => {
            document.getElementById('login').classList.remove('hidden')
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
    logs:     loadLogs,
}

// hash is "tab" or "tab/sub" (e.g. tickets/123)
function parseHash() {
    const [tab, ...rest] = window.location.hash.replace(/^#/, '').split('/')
    return { tab: tabLoaders[tab] ? tab : 'workflows', sub: rest.join('/') || null }
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
    setContent(skeletonPage())
    tabLoaders[tab](sub)
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
// type is the app's own vocabulary ('success' | 'error' | 'info'); navi's
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
// Utilities
// ─────────────────────────────────────────────────────────
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

// splits a UTC ISO string into local ['YYYY-MM-DD', 'HH:MM'] for date/time inputs
function splitLocalDT(iso) {
    if (!iso) return ['', '']
    const d = new Date(iso)
    const p = n => String(n).padStart(2, '0')
    return [
        `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`,
        `${p(d.getHours())}:${p(d.getMinutes())}`,
    ]
}

// ─────────────────────────────────────────────────────────
// Render helpers (navi components)
// ─────────────────────────────────────────────────────────
// badgeTag renders a labelled badge. variant is a navi tone —
// ok | bad | warn | info | accent | outline | '' (neutral).
function badgeTag(label, variant = '') {
    const dot = ['ok', 'bad', 'warn', 'info'].includes(variant) ? '<i class="dot"></i>' : ''
    return `<span class="badge ${variant}">${dot}${esc(label)}</span>`
}

function badge(val) {
    return badgeTag(val ? 'Yes' : 'No', val ? 'ok' : 'bad')
}

// pageHead is the title block every view starts with. actions is button markup.
function pageHead(title, sub = '', actions = '') {
    return `<header class="page-head row spread wrap gap4">
        <div>
            <h1 class="page-title">${esc(title)}</h1>
            ${sub ? `<p class="page-sub">${sub}</p>` : ''}
        </div>
        ${actions ? `<div class="row gap2 wrap">${actions}</div>` : ''}
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
    return `<button class="btn btn-ghost btn-sm" onclick="${onclick}">${icon('trash')}${esc(label)}</button>`
}

// checkbox is navi's styled checkbox; the box element carries the tick.
function checkbox(label, attrs = '', checked = false) {
    return `<label class="check">
        <input type="checkbox" ${checked ? 'checked' : ''} ${attrs}>
        <span class="box">${icon('check')}</span>
        <span>${label}</span>
    </label>`
}

// toggle is navi's switch; .track must stay a flex box or the thumb collapses.
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

    const head = pageHead(
        'Notification forwards',
        'Send another person\u2019s ticket notifications to a room or teammate for a period of time.',
        `<button class="btn btn-primary" onclick="showForwardModal()">${icon('plus')}New forward</button>`)

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
            `<button class="btn btn-primary btn-sm" onclick="showForwardModal()">${icon('plus')}New forward</button>`, 'mail'),
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

    const [startDate, startTime] = splitLocalDT(existing?.start_date)
    const [endDate, endTime]     = splitLocalDT(existing?.end_date)

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
                    <label for="f-start-date">Start <span class="muted">(optional)</span></label>
                    <div class="row gap2">
                        <input class="input" type="date" id="f-start-date" style="flex:2" value="${startDate}" aria-label="Start date">
                        <input class="input" type="time" id="f-start-time" style="flex:1" value="${startTime}" aria-label="Start time">
                    </div>
                </div>
                <div class="field">
                    <label for="f-end-date">End <span class="muted">(optional)</span></label>
                    <div class="row gap2">
                        <input class="input" type="date" id="f-end-date" style="flex:2" value="${endDate}" aria-label="End date">
                        <input class="input" type="time" id="f-end-time" style="flex:1" value="${endTime}" aria-label="End time">
                    </div>
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
        const startDate = document.getElementById('f-start-date').value
        const startTime = document.getElementById('f-start-time').value
        const endDate   = document.getElementById('f-end-date').value
        const endTime   = document.getElementById('f-end-time').value
        const enabled   = document.getElementById('f-enabled').value === 'true'
        const keepCopy  = document.getElementById('f-keep').value === 'true'
        const soleResource = document.getElementById('f-sole').value === 'true'
        const publicOnly   = document.getElementById('f-public').value === 'true'

        const startDT = startDate ? `${startDate}T${startTime || '00:00'}` : null
        const endDT   = endDate   ? `${endDate}T${endTime   || '23:59'}` : null

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
    const head = pageHead('Users', 'People who can sign in to this console.',
        `<button class="btn btn-primary" onclick="showNewUserModal()">${icon('plus')}New user</button>`)

    const thead = '<th class="r">ID</th><th>Email</th><th>Created</th><th class="r">Actions</th>'
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
        <td class="muted nowrap">${fmtDateTime(u.created_on)}</td>
        <td class="r nowrap">${u.id === currentUser?.id
            ? '<span class="badge outline">You</span>'
            : deleteButton(`deleteUser(${u.id})`)}</td>
    </tr>`)

    setContent(head + tableCard(thead, rows, {
        empty: emptyState('No users yet', 'Create a user so someone can sign in.',
            `<button class="btn btn-primary btn-sm" onclick="showNewUserModal()">${icon('plus')}New user</button>`, 'users'),
        foot: `<span>${users.length} user${users.length === 1 ? '' : 's'}</span>`,
    }))
}

function showNewUserModal() {
    openModal('New user', `
        <div class="stack gap4">
            <div class="field">
                <label for="f-email">Email address</label>
                <input class="input" type="email" id="f-email" placeholder="user@example.com">
            </div>
            <div class="field">
                <label for="f-temp-password">Temporary password</label>
                <input class="input" type="password" id="f-temp-password" placeholder="••••••••">
                <span class="hint">The user must change this on first sign-in.</span>
            </div>
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

    const head = pageHead('API keys', 'Keys authenticate machine callers against the ticketbot API.',
        `<button class="btn btn-primary" onclick="showNewKeyModal()">${icon('plus')}New key</button>`)

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
    if (key) navigator.clipboard.writeText(key).then(() => toast('Copied!', 'success'))
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

    setContent(pageHead('Sync',
        'Pulls the latest boards, Webex recipients and tickets from ConnectWise and Webex. Run it after adding a board or changing room membership.',
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
async function loadConfig() {
    try {
        const cfg = await api('GET', '/config')
        renderConfig(cfg)
    } catch (e) {
        setContent(errorState(e.message))
    }
}

function renderConfig(cfg) {
    // one settings row: description on the left, control on the right
    const row = (title, desc, control) => `<div class="form-row">
        <div><h4>${title}</h4><p class="desc">${desc}</p></div>
        <div class="row gap3 wrap">${control}</div>
    </div>`

    const numberInput = (id, value, min) =>
        `<input class="input" style="max-width:160px" type="number" id="${id}" value="${esc(value)}" min="${min}" aria-label="${esc(id)}">`

    setContent(pageHead('Configuration', 'Runtime settings. Changes take effect without a restart.',
        `<button class="btn btn-primary" onclick="saveConfig()">Save changes</button>`) +
    `<div class="card card-pad">
        ${row('Master dry run',
            'Run every workflow as a dry run: no ConnectWise writes, Webex messages are mocked.',
            toggle(`id="c-master-dry-run"`, cfg.master_dry_run, { tip: 'Master dry run' }))}
        ${row('CW API member identifier',
            'ConnectWise member the API key belongs to; its own updates never trigger workflows. Auto-filled after the first note ticketbot posts.',
            `<input class="input" style="max-width:260px" type="text" id="c-api-member" value="${esc(cfg.cw_api_member_identifier || '')}" placeholder="e.g. ticketbot" aria-label="CW API member identifier">`)}
        ${row('Max message length',
            'Truncation limit for ticket note content, in characters.',
            numberInput('c-max-len', cfg.max_message_length, 1))}
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
            max_concurrent_syncs:       num('c-max-syncs', 'Max concurrent syncs'),
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

function renderLogs(entries) {
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

    const isEmpty        = rows.length === 0
    const searchFocused  = document.activeElement?.id === 'logs-search'
    const searchPos      = searchFocused ? document.getElementById('logs-search')?.selectionStart : null

    setContent(pageHead('Logs', 'Everything the server has logged since it started, newest first.') +
    `<div class="card">
        <div class="filter-bar">
            <select class="select" id="logs-level-filter" onchange="setLogsFilter(this.value)" aria-label="Filter by level">${levelOpts}</select>
            <select class="select" id="logs-context-filter" onchange="setLogsContextFilter(this.value)" aria-label="Filter by area">${contextOpts}</select>
            <div class="input-group">
                ${icon('search')}
                <input class="input" id="logs-search" type="text" style="width:220px" placeholder="Search…" value="${esc(logsSearch)}" oninput="setLogsSearch(this.value)" aria-label="Search log messages">
            </div>
            <button class="btn btn-default btn-sm" onclick="openLogsOptions(event)">${icon('filter')}Options</button>
            <div class="grow"></div>
            ${logsFrozen
                ? '<span class="badge outline"><i class="dot"></i>Frozen</span>'
                : '<span class="badge ok"><i class="dot pulse"></i>Streaming</span>'}
            <button class="btn btn-default btn-sm" onclick="toggleLogFreeze()">${logsFrozen ? 'Resume' : 'Freeze'}</button>
            <button class="btn btn-default btn-sm" onclick="loadLogs()">Refresh</button>
        </div>
        ${isEmpty
            ? emptyState('No matching log entries',
                'Nothing in the buffer matches these filters. Widen the level or clear the search.',
                `<button class="btn btn-default btn-sm" onclick="resetLogsFilters()">Clear filters</button>`, 'book')
            : `<div class="log-list">${rows.join('')}</div>`}
        ${isEmpty ? '' : `<div class="card-foot"><span>${rows.length} of ${entries.length} entries</span></div>`}
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

function openLogsOptions(e) {
    e.stopPropagation()
    if (menuEl) { closeMenu(); return }
    openMenu(e.currentTarget, [
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
    renderLogs(logsLastEntries)
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
        if (menuEl && !menuEl.contains(e.target)) closeMenu()
        const app = document.getElementById('app')
        if (app.classList.contains('nav-open') && !e.target.closest('.sidebar')) app.classList.remove('nav-open')
    }, true)

    buildNav()
    applyTheme(document.documentElement.dataset.theme || 'light')
    checkSavedKey()
})
