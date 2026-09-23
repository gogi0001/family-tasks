const pad = n => String(n).padStart(2, '0');

// D.M.YY — например, 5.9.26
export function fmtDate(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d.getTime())) return '';
    const yy = String(d.getFullYear()).slice(-2);
    return `${d.getDate()}.${d.getMonth() + 1}.${yy}`;
}

// HH:mm — например, 09:05
export function fmtTime(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d.getTime())) return '';
    return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// D.M.YY HH:mm — например, 5.9.26 09:05
export function fmtDateTime(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d.getTime())) return '';
    return `${fmtDate(iso)} ${fmtTime(iso)}`;
}