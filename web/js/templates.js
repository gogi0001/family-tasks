import { els } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';

const WD_NAMES = ['вс', 'пн', 'вт', 'ср', 'чт', 'пт', 'сб'];

let loaded = false;

// --- Модалка списка ---

export function openModal() {
    els.templatesModal.classList.add('open');
    load();
}

export function closeModal() {
    els.templatesModal.classList.remove('open');
}

function isOpen() {
    return els.templatesModal.classList.contains('open');
}

async function load() {
    if (!state.family) return;
    try {
        const list = await api('/templates');
        renderList(list || []);
        loaded = true;
    } catch (e) {
        if (e.status === 0) return;
        toast(e.message, 'error');
    }
}

// --- Рендер ---

function describeRule(rule) {
    if (!rule) return '';
    switch (rule.type) {
        case 'daily': {
            const i = rule.interval && rule.interval > 1
                ? `каждые ${rule.interval} дн.`
                : 'каждый день';
            return `${i} в ${rule.time}`;
        }
        case 'weekly': {
            const names = (rule.weekdays || [])
                .map(d => WD_NAMES[d] ?? '?')
                .join(', ');
            return `по ${names} в ${rule.time}`;
        }
        case 'monthly': {
            return `${rule.dayOfMonth}-го числа в ${rule.time}`;
        }
        default:
            return '';
    }
}

function fmtDate(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d.getTime())) return '';
    return d.toLocaleString('ru-RU', {
        day: '2-digit', month: '2-digit',
        hour: '2-digit', minute: '2-digit',
    });
}

function renderList(list) {
    els.templatesList.replaceChildren();
    els.templatesEmpty.hidden = list.length > 0;

    for (const t of list) {
        const row = document.createElement('div');
        row.className = 'template-row' + (t.active ? '' : ' inactive');

        const head = document.createElement('div');
        head.className = 'template-head';

        const title = document.createElement('h3');
        title.className = 'template-title';
        title.textContent = t.title;
        head.appendChild(title);

        const actions = document.createElement('div');
        actions.className = 'template-actions';

        // Toggle
        const toggle = document.createElement('button');
        toggle.type = 'button';
        toggle.title = t.active ? 'Приостановить' : 'Возобновить';
        toggle.textContent = t.active ? '⏸' : '▶';
        toggle.addEventListener('click', () => toggleActive(t.id, !t.active));
        actions.appendChild(toggle);

        // Delete
        const del = document.createElement('button');
        del.type = 'button';
        del.title = 'Удалить правило';
        del.textContent = '×';
        del.addEventListener('click', () => removeTemplate(t.id, t.title));
        actions.appendChild(del);

        head.appendChild(actions);
        row.appendChild(head);

        const meta = document.createElement('div');
        meta.className = 'template-meta';
        meta.textContent = '→ ' + t.assignee + ' · ' + describeRule(t.rule);
        row.appendChild(meta);

        const when = document.createElement('div');
        when.className = 'template-when muted small';
        if (t.active) {
            when.textContent = 'Следующая: ' + fmtDate(t.nextRunAt);
        } else {
            when.textContent = 'Приостановлено';
        }
        row.appendChild(when);

        els.templatesList.appendChild(row);
    }
}

// --- Действия ---

async function toggleActive(id, active) {
    try {
        await api(`/templates/${id}`, {
            method: 'PATCH',
            body: JSON.stringify({ active }),
        });
        await load();
    } catch (e) {
        if (e.status === 0) return;
        toast(e.message, 'error');
    }
}

async function removeTemplate(id, title) {
    if (!confirm(`Удалить правило «${title}»? Уже созданные задачи останутся.`)) return;
    try {
        await api(`/templates/${id}`, { method: 'DELETE' });
        await load();
    } catch (e) {
        if (e.status === 0) return;
        toast(e.message, 'error');
    }
}

// --- Init ---

export function init() {
    els.templatesOpen.addEventListener('click', openModal);
    els.templatesClose.addEventListener('click', closeModal);
    els.templatesCloseBtn.addEventListener('click', closeModal);
    els.templatesModal.addEventListener('click', (e) => {
        if (e.target === els.templatesModal) closeModal();
    });
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && isOpen()) closeModal();
    });
}

// Перечитать список, если модалка открыта (вызывается по SSE).
export function reloadIfOpen() {
    if (isOpen()) load();
}