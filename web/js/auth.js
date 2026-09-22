import { els, log } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { emit } from './events.js';

let activeTab = 'login';           // 'login' | 'register'
let inviteFromURL = '';

// --- Модалка входа / регистрации ---

export function openAuthModal(tab) {
    setTab(tab || activeTab);
    els.authModal.classList.add('open');
    setTimeout(() => {
        const input = activeTab === 'login' ? els.loginEmail : els.registerEmail;
        input.focus();
    }, 0);
}

export function closeAuthModal() {
    els.authModal.classList.remove('open');
}

function isAuthOpen() {
    return els.authModal.classList.contains('open');
}

function setTab(tab) {
    activeTab = tab;
    els.authTabLogin.classList.toggle('active', tab === 'login');
    els.authTabRegister.classList.toggle('active', tab === 'register');
    els.authLoginForm.hidden = tab !== 'login';
    els.authRegForm.hidden = tab !== 'register';
}

// --- URL: ?invite=CODE ---

function readInviteFromURL() {
    const params = new URLSearchParams(location.search);
    const code = (params.get('invite') || '').trim();
    if (!code) return '';
    inviteFromURL = code;
    const url = new URL(location.href);
    url.searchParams.delete('invite');
    history.replaceState(null, '', url.pathname + url.search + url.hash);
    return code;
}

export function getInviteFromURL() {
    return inviteFromURL;
}

async function checkInvite(code) {
    if (!code) {
        els.inviteHint.hidden = true;
        return;
    }
    try {
        const info = await api(`/invites/${encodeURIComponent(code)}`);
        els.inviteHint.hidden = false;
        if (info.valid) {
            els.inviteHint.textContent = `Семья: ${info.familyName}`;
            els.inviteHint.classList.remove('error');
        } else {
            els.inviteHint.textContent = 'Код недействителен или уже использован';
            els.inviteHint.classList.add('error');
        }
    } catch {
        els.inviteHint.hidden = true;
    }
}

// --- Логин / регистрация ---

async function onLogin(e) {
    e.preventDefault();
    const email = els.loginEmail.value.trim();
    const password = els.loginPassword.value;
    if (!email || !password) return;

    try {
        state.user = await api('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        log('auth: logged in as', state.user.name);
        els.loginPassword.value = '';
        closeAuthModal();
        emit('identified');
    } catch (err) {
        if (err.status === 0) return;
        toast(err.message || 'Ошибка входа', 'error');
    }
}

async function onRegister(e) {
    e.preventDefault();
    const email = els.registerEmail.value.trim();
    const name = els.registerName.value.trim();
    const password = els.registerPassword.value;
    const invite = (els.registerInvite.value.trim() || inviteFromURL);

    if (!email || !name || !password) return;

    try {
        state.user = await api('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password, name, invite }),
        });
        log('auth: registered as', state.user.name, 'invite:', invite || '—');
        els.registerPassword.value = '';
        inviteFromURL = '';
        closeAuthModal();
        emit('identified');
    } catch (err) {
        if (err.status === 0) return;
        toast(err.message || 'Ошибка регистрации', 'error');
    }
}

async function onLogout() {
    try {
        await api('/auth/logout', { method: 'POST' });
    } catch { }
    state.user = null;
    emit('logged-out');
}

// --- Смена своего пароля ---

function openPasswordModal() {
    if (!state.user) return;
    els.pwdCurrent.value = '';
    els.pwdNew.value = '';
    els.pwdConfirm.value = '';
    els.passwordModal.classList.add('open');
    setTimeout(() => els.pwdCurrent.focus(), 0);
}

export function closePasswordModal() {
    els.passwordModal.classList.remove('open');
    els.pwdCurrent.value = '';
    els.pwdNew.value = '';
    els.pwdConfirm.value = '';
}

function isPasswordModalOpen() {
    return els.passwordModal.classList.contains('open');
}

async function onPasswordSubmit(e) {
    e.preventDefault();

    const current = els.pwdCurrent.value;
    const next = els.pwdNew.value;
    const confirm = els.pwdConfirm.value;

    if (!current || !next || !confirm) return;
    if (next.length < 8) {
        toast('Новый пароль должен быть не короче 8 символов', 'error');
        return;
    }
    if (next !== confirm) {
        toast('Новый пароль и подтверждение не совпадают', 'error');
        return;
    }
    if (next === current) {
        toast('Новый пароль совпадает с текущим', 'error');
        return;
    }

    try {
        await api('/auth/change-password', {
            method: 'POST',
            body: JSON.stringify({ currentPassword: current, newPassword: next }),
        });
        closePasswordModal();
        toast('Пароль изменён', 'info');
    } catch (err) {
        if (err.status === 0) return;
        toast(err.message || 'Не удалось сменить пароль', 'error');
    }
}

// --- Init ---

export function init() {
    els.authTabLogin.addEventListener('click', () => setTab('login'));
    els.authTabRegister.addEventListener('click', () => setTab('register'));

    els.authLoginForm.addEventListener('submit', onLogin);
    els.authRegForm.addEventListener('submit', onRegister);

    els.whoLogout.addEventListener('click', onLogout);

    // Смена пароля
    els.whoPassword.addEventListener('click', openPasswordModal);
    els.pwdCancel.addEventListener('click', closePasswordModal);
    els.passwordModal.addEventListener('click', (e) => {
        if (e.target === els.passwordModal) closePasswordModal();
    });
    els.passwordForm.addEventListener('submit', onPasswordSubmit);

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && isPasswordModalOpen()) closePasswordModal();
    });

    // Код из URL
    const urlInvite = readInviteFromURL();
    if (urlInvite) {
        setTab('register');
        els.registerInvite.value = urlInvite;
        checkInvite(urlInvite);
    }

    let timer = null;
    els.registerInvite.addEventListener('input', () => {
        clearTimeout(timer);
        const code = els.registerInvite.value.trim();
        timer = setTimeout(() => checkInvite(code), 400);
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && isAuthOpen() && !urlInvite) closeAuthModal();
    });
}

export function requireAuth() {
    openAuthModal('login');
}