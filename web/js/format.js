function formatDateShort(dateStr) {
  if (!dateStr) return '';
  try {
    const d = new Date(dateStr);
    return d.toLocaleDateString('id-ID', { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) + ' WIB';
  } catch (e) {
    return dateStr;
  }
}

function formatDateLong(dateStr) {
  if (!dateStr) return '';
  try {
    const d = new Date(dateStr);
    return d.toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' }) + ' WIB';
  } catch (e) {
    return dateStr;
  }
}

function isTaskUrgent(deadlineStr) {
  if (!deadlineStr) return false;
  try {
    const d = new Date(deadlineStr);
    const now = new Date('2026-09-24T12:00:00+07:00');
    const diffHours = (d - now) / (1000 * 60 * 60);
    return diffHours <= 48 && diffHours >= -24;
  } catch (e) {
    return false;
  }
}
