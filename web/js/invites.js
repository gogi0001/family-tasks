import { els } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';

export async function loadInvites() {
    if (!isOwner()) {
        els.invitesSection.hidden = true;
        return;
    }
    els.invitesSection.hidden = false;

    try {
        const list = await api('/families/invites');
        renderInvites(list || []);
    } catch (e) {
        if (e.status === 0) return;
        toast(e.message, 'error');
    }
}

function isOwner() {
    const me = state.family?.members.find(m => m.id === state.user?.id);
    return me?.role === 'owner';
}

function renderInvites(list) {
    els.invitesList.replaceChildren();
    if (list.length === 0) {
        const empty = document.createElement('p');
        empty.className = 'muted small';
        empty.textContent = 'Нет активных ссылок';
        els.invitesList.appendChild(empty);
        return;
    }

    const now = Date.now();
    for (const inv of list) {
        const row = document.createElement('div');
        row.className = 'invite-row';

        const used = !!inv.usedBy;
        const expired = new Date(inv.expiresAt).getTime() < now;
        const status = used ? 'использован' : (expired ? 'просрочен' : 'активен');

        const link = document.createElement('div');
        link.className = 'invite-link';
        const codeEl = document.createElement('b');
        codeEl.textContent = inv.code.slice(0, 12) + '…';
        codeEl.title = inv.code;
        link.appendChild(codeEl);
        const statusEl = document.createElement('span');
        statusEl.className = 'invite-status ' + (used || expired ? 'muted' : 'ok');
        statusEl.textContent = status;
        link.appendChild(statusEl);
        row.appendChild(link);

        const actions = document.createElement('div');
        actions.className = 'invite-actions';

        if (!used && !expired) {
            const copy = document.createElement('button');
            copy.type = 'button';
            copy.textContent = 'Ссылка';
            copy.title = 'Скопировать ссылку-приглашение';
            copy.addEventListener('click', () => copyInviteURL(inv.code));
            actions.appendChild(copy);
        }

        const del = document.createElement('button');
        del.type = 'button';
        del.textContent = '×';
        del.title = 'Отозвать';
        del.addEventListener('click', () => revokeInvite(inv.id));
        actions.appendChild(del);

        row.appendChild(actions);
        els.invitesList.appendChild(row);
    }
}

function inviteURL(code) {
    return `${location.origin}/?invite=${encodeURIComponent(code)}`;
}

async function copyInviteURL(code) {
    const url = inviteURL(code);
    try {
        await navigator.clipboard.writeText(url);
        toast('Ссылка скопирована', 'info');
    } catch {
        // Fallback: prompt со ссылкой
        window.prompt('Скопируйте ссылку:', url);
    }
}

async function revokeInvite(id) {
    if (!confirm('Отозвать эту ссылку? Она станет недействительной.')) return;
    try {
        await api(`/families/invites/${id}`, { method: 'DELETE' });
        await loadInvites();
    } catch (e) {
        if (e.status === 0) return;
        toast(e.message, 'error');
    }
}

async function createInvite() {
    try {
        const inv = await api('/families/invites', {
            method: 'POST',
            body: JSON.stringify({}),
        });
        await loadInvites();
        // Сразу копируем ссылку — удобно
        await copyInviteURL(inv.code);
    } catch (e) {
        if (e.status === 0) return;
        toast(e.message, 'error');
    }
}

export function init() {
    els.inviteCreate.addEventListener('click', createInvite);
}