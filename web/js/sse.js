import { log, warn } from './dom.js';
import { emit } from './events.js';
import { api } from './api.js';

let es = null;
let started = false;

export function start() {
    if (started) return;
    started = true;

    log('sse: start');

    es = new EventSource('/api/v1/events');

    es.onopen = () => {
        log('sse: connected');
        emit('sse-open');
    };

    es.onmessage = (e) => {
        let data;
        try {
            data = JSON.parse(e.data);
        } catch {
            warn('sse: bad json', e.data);
            return;
        }
        log('sse: event', data.type);
        emit('sse-event', data);
    };

    es.onerror = async () => {
        if (!es) return;

        // CONNECTING — EventSource сам переподключится, ничего не делаем.
        if (es.readyState !== EventSource.CLOSED) {
            warn('sse: reconnecting…');
            return;
        }

        // CLOSED — финальное отключение. Возможно, протухла cookie.
        warn('sse: closed, checking auth');
        es = null;
        started = false;

        try {
            await api('/me');
            setTimeout(start, 5000);
        } catch (e) {
            if (e.status === 401) emit('unauthorized');
            // Сетевая ошибка — не перезапускаем, connection-changed сам всё поднимет.
        }
    };
}

export function stop() {
    if (!es) {
        started = false;
        return;
    }
    log('sse: stop');
    es.close();
    es = null;
    started = false;
}