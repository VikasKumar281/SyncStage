export const API_BASE = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '');

function url(path) {
  return `${API_BASE}${path}`;
}

async function request(path, options = {}) {
  const response = await fetch(url(path), {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  if (!response.ok) {
    let detail = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      if (body?.error) detail = body.error;
    } catch {
     
    }
    throw new Error(detail);
  }

  if (response.status === 204) return null;
  return response.json();
}

export const api = {
  getState: () => request('/api/state'),
  getServerTime: () => request('/api/time'),

  addMedia: (payload) =>
    request('/api/media', { method: 'POST', body: JSON.stringify(payload) }),

  createWindow: (name) =>
    request('/api/windows', { method: 'POST', body: JSON.stringify({ name }) }),

  addPlaylistItem: (windowId, payload) =>
    request(`/api/windows/${windowId}/playlist`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  removePlaylistItem: (windowId, itemId) =>
    request(`/api/windows/${windowId}/playlist/${itemId}`, { method: 'DELETE' }),

  triggerSync: (mediaId, durationSeconds) =>
    request('/api/sync', {
      method: 'POST',
      body: JSON.stringify({ mediaId, durationSeconds }),
    }),

  cancelSync: () => request('/api/sync', { method: 'DELETE' }),

  resetCycle: () => request('/api/cycle/reset', { method: 'POST' }),

  eventsUrl: () => url('/api/events'),
};

/**
 * @returns {Promise<{offsetMs: number, roundTripMs: number}>}
 *          offsetMs is added to Date.now() to get server time.
 */
export async function measureClockOffset(samples = 5) {
  let best = { offsetMs: 0, roundTripMs: Number.POSITIVE_INFINITY };

  for (let i = 0; i < samples; i += 1) {
    const sentAt = Date.now();
    let serverTimeMs;
    try {
      ({ serverTimeMs } = await api.getServerTime());
    } catch {
      continue; 
    }
    const receivedAt = Date.now();
    const roundTripMs = receivedAt - sentAt;

    const offsetMs = serverTimeMs - (sentAt + roundTripMs / 2);

    if (roundTripMs < best.roundTripMs) {
      best = { offsetMs, roundTripMs };
    }
  }

  if (!Number.isFinite(best.roundTripMs)) {
    return { offsetMs: 0, roundTripMs: 0 };
  }
  return best;
}
