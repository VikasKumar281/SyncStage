import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, measureClockOffset } from './api.js';


export function useSequencer() {
  const [snapshot, setSnapshot] = useState(null);
  const [connection, setConnection] = useState('connecting');
  const [error, setError] = useState(null);
  const [clock, setClock] = useState({ offsetMs: 0, roundTripMs: 0 });


  const clockRef = useRef(clock);
  clockRef.current = clock;

  const serverNow = useCallback(() => Date.now() + clockRef.current.offsetMs, []);

  const refresh = useCallback(async () => {
    try {
      const next = await api.getState();
      setSnapshot(next);
      setError(null);
      return next;
    } catch (err) {
      setError(err.message);
      throw err;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const calibrate = async () => {
      const measured = await measureClockOffset();
      if (!cancelled) setClock(measured);
    };

    calibrate();
    const id = setInterval(calibrate, 120_000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, []);


  useEffect(() => {
    const source = new EventSource(api.eventsUrl());

    const applySnapshot = (event) => {
      try {
        setSnapshot(JSON.parse(event.data));
        setConnection('live');
        setError(null);
      } catch (err) {
        setError(`Could not read update: ${err.message}`);
      }
    };

    for (const name of [
      'snapshot',
      'playlist.updated',
      'sync.scheduled',
      'sync.cancelled',
      'media.created',
      'window.created',
      'cycle.reset',
    ]) {
      source.addEventListener(name, applySnapshot);
    }

    source.onopen = () => setConnection('live');
    source.onerror = () => {
      setConnection('reconnecting');
    };

    refresh().catch(() => {});

    return () => source.close();
  }, [refresh]);


  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        refresh().catch(() => {});
        measureClockOffset(3).then(setClock);
      }
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [refresh]);

  const actions = useMemo(
    () => ({
      addPlaylistItem: (windowId, mediaId, durationSeconds) =>
        api.addPlaylistItem(windowId, { mediaId, durationSeconds }),
      removePlaylistItem: (windowId, itemId) => api.removePlaylistItem(windowId, itemId),
      addMedia: (payload) => api.addMedia(payload),
      createWindow: (name) => api.createWindow(name),
      triggerSync: (mediaId, durationSeconds) => api.triggerSync(mediaId, durationSeconds),
      cancelSync: () => api.cancelSync(),
      resetCycle: () => api.resetCycle(),
      refresh,
    }),
    [refresh],
  );

  return { snapshot, connection, error, clock, serverNow, actions, setError };
}
