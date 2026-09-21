import { els } from './dom.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { emit } from './events.js';

let pendingTaskId = null;
let uploadInFlight = false;

export function init() {
    els.uploadInput.addEventListener('change', onFilesChosen);

    els.lightbox.addEventListener('click', (e) => {
        if (e.target === els.lightbox) closeLightbox();
    });
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && !els.lightbox.hidden) closeLightbox();
    });
}

// Запустить выбор файлов для конкретной задачи.
export function pickFilesFor(taskId) {
    if (uploadInFlight) {
        toast('Дождитесь завершения загрузки', 'error');
        return;
    }
    pendingTaskId = taskId;
    els.uploadInput.value = '';
    els.uploadInput.click();
}

async function onFilesChosen() {
    const files = Array.from(els.uploadInput.files || []);
    const taskId = pendingTaskId;
    pendingTaskId = null;

    if (!taskId || files.length === 0) return;

    uploadInFlight = true;
    let uploaded = 0;
    try {
        for (const file of files) {
            await uploadOne(taskId, file);
            uploaded++;
        }
        if (uploaded > 0) emit('attachments-changed');
    } catch (e) {
        if (uploaded > 0) emit('attachments-changed');
        if (e.status === 0) return;
        toast(e.message || 'Не удалось загрузить', 'error');
    } finally {
        uploadInFlight = false;
    }
}

function uploadOne(taskId, file) {
    const form = new FormData();
    form.append('file', file);
    return api(`/tasks/${taskId}/attachments`, {
        method: 'POST',
        body: form,
    });
}

export function openLightbox(att) {
    els.lightboxImg.src = att.url;
    els.lightboxImg.alt = att.filename || '';
    els.lightbox.hidden = false;
}

export function closeLightbox() {
    els.lightbox.hidden = true;
    els.lightboxImg.src = '';
}