export const SOURCE_SEQUENCE = 'sequence';
export const SOURCE_SYNC = 'sync';
export const SOURCE_IDLE = 'idle';

function floorMod(a, m) {
  if (m <= 0) return 0;
  const r = a % m;
  return r < 0 ? r + m : r;
}

function floorDiv(a, m) {
  if (m <= 0) return 0;
  return Math.floor(a / m);
}

function getPlaylist(window) {
  return Array.isArray(window?.playlist) ? window.playlist : [];
}

function playlistDuration(window) {
  const playlist = getPlaylist(window);

  return playlist.reduce(
    (total, item) => total + (Number(item.durationMs) || 0),
    0
  );
}

function findMedia(media, id) {
  if (!Array.isArray(media)) return null;
  return media.find((m) => m.id === id) ?? null;
}

function syncIsActiveAt(sync, nowMs) {
  if (!sync) return false;

  return (
    nowMs >= sync.startAtMs &&
    nowMs < sync.startAtMs + sync.durationMs
  );
}

/**
 * @param {object} snapshot  the payload from /api/state
 * @param {object} window    one entry from snapshot.windows
 * @param {number} nowMs     server-corrected epoch milliseconds
 */
export function resolve(snapshot, window, nowMs) {
  const { activeSync, media } = snapshot;

  if (syncIsActiveAt(activeSync, nowMs)) {
    const endsAtMs =
      activeSync.startAtMs + activeSync.durationMs;

    return {
      windowId: window.id,
      source: SOURCE_SYNC,
      mediaId: activeSync.mediaId,
      itemId: activeSync.id,
      itemIndex: -1,
      startedAtMs: activeSync.startAtMs,
      endsAtMs,
      remainingMs: endsAtMs - nowMs,
      ...cyclePosition(snapshot, nowMs),
      media: findMedia(media, activeSync.mediaId),
    };
  }

  return resolveSequence(snapshot, window, nowMs);
}

export function resolveSequence(snapshot, window, nowMs) {
  const { cycleMs, media } = snapshot;

  const playlist = getPlaylist(window);

  const { cycleIndex, offsetInCycleMs } =
    cyclePosition(snapshot, nowMs);

  const base = {
    windowId: window.id,
    itemIndex: -1,
    cycleIndex,
    offsetInCycleMs,
    media: null,
  };

  const totalMs = playlistDuration(window);

  // No playlist or invalid/empty playlist
  if (playlist.length === 0 || totalMs <= 0) {
    const endsAtMs =
      nowMs + (cycleMs - offsetInCycleMs);

    return {
      ...base,
      source: SOURCE_IDLE,
      mediaId: null,
      itemId: null,
      startedAtMs: nowMs,
      endsAtMs,
      remainingMs: endsAtMs - nowMs,
    };
  }

  const posInPass =
    floorMod(offsetInCycleMs, totalMs);

  let accumulated = 0;
  let index = 0;

  for (let i = 0; i < playlist.length; i += 1) {
    const durationMs =
      Number(playlist[i].durationMs) || 0;

    if (posInPass < accumulated + durationMs) {
      index = i;
      break;
    }

    accumulated += durationMs;
    index = i;
  }

  const item = playlist[index];

  const itemDurationMs =
    Number(item.durationMs) || 0;

  const startedAtMs =
    nowMs - (posInPass - accumulated);

  const cycleEndsAtMs =
    nowMs + (cycleMs - offsetInCycleMs);

  const endsAtMs = Math.min(
    startedAtMs + itemDurationMs,
    cycleEndsAtMs
  );

  return {
    ...base,
    source: SOURCE_SEQUENCE,
    mediaId: item.mediaId,
    itemId: item.id,
    itemIndex: index,
    startedAtMs,
    endsAtMs,
    remainingMs: endsAtMs - nowMs,
    media: findMedia(media, item.mediaId),
  };
}

export function cyclePosition(snapshot, nowMs) {
  const delta =
    nowMs - snapshot.cycleAnchorMs;

  return {
    cycleIndex: floorDiv(
      delta,
      snapshot.cycleMs
    ),

    offsetInCycleMs: floorMod(
      delta,
      snapshot.cycleMs
    ),
  };
}

export function upNext(snapshot, window, nowMs) {
  const current =
    resolveSequence(snapshot, window, nowMs);

  if (current.source !== SOURCE_SEQUENCE) {
    return null;
  }

  return resolveSequence(
    snapshot,
    window,
    current.endsAtMs + 1
  );
}

export function timecode(ms) {
  const total =
    Math.max(0, Math.floor(ms / 1000));

  const h =
    Math.floor(total / 3600);

  const m =
    Math.floor((total % 3600) / 60);

  const s =
    total % 60;

  const pad = (n) =>
    String(n).padStart(2, '0');

  return h > 0
    ? `${h}:${pad(m)}:${pad(s)}`
    : `${pad(m)}:${pad(s)}`;
}

export function humanDuration(ms) {
  const total =
    Math.round(ms / 1000);

  if (total < 60) {
    return `${total}s`;
  }

  const m =
    Math.floor(total / 60);

  const s =
    total % 60;

  return s === 0
    ? `${m}m`
    : `${m}m ${s}s`;
}