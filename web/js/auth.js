import { els, log } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { emit } from './events.js';

let activeTab = 'login';           // 'login' | 'register'
let inviteFromURL = '';

// --- Модалка ---

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
    // Очищаем URL, чтобы не тащить код в истории и закладках.
    const url = new URL(location.href);
    url.searchParams.delete('invite');
    history.replaceState(null, '', url.pathname + url.search + url.hash);
    return code;
}

export function getInviteFromURL() {
    return inviteFromURL;
}

// Проверка кода и подсказка в форме.
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

// --- Handlers ---

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

// --- Init ---

export function init() {
    els.authTabLogin.addEventListener('click', () => setTab('login'));
    els.authTabRegister.addEventListener('click', () => setTab('register'));

    els.authLoginForm.addEventListener('submit', onLogin);
    els.authRegForm.addEventListener('submit', onRegister);

    els.whoLogout.addEventListener('click', onLogout);

    // Код из URL
    const urlInvite = readInviteFromURL();
    if (urlInvite) {
        setTab('register');
        els.registerInvite.value = urlInvite;
        checkInvite(urlInvite);
    }

    // Проверка при вводе кода вручную (с дебаунсом)
    let timer = null;
    els.registerInvite.addEventListener('input', () => {
        clearTimeout(timer);
        const code = els.registerInvite.value.trim();
        timer = setTimeout(() => checkInvite(code), 400);
    });

    // Escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && isAuthOpen() && !urlInvite) closeAuthModal();
    });
}

// Открыть модалку, когда что-то запрашивает авторизацию.
export function requireAuth() {
    openAuthModal('login');
}