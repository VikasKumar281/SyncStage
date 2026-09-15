import { useState } from 'react';
import { humanDuration, timecode } from '../lib/timeline.js';


export default function ControlPanel({ snapshot, serverNow, actions, onError }) {
  const [syncMediaId, setSyncMediaId] = useState(snapshot.media[0]?.id ?? '');
  const [syncSeconds, setSyncSeconds] = useState(10);

  const [targetWindowId, setTargetWindowId] = useState(snapshot.windows[0]?.id ?? '');
  const [addMediaId, setAddMediaId] = useState(snapshot.media[0]?.id ?? '');
  const [addSeconds, setAddSeconds] = useState(8);

  const [newMedia, setNewMedia] = useState({ name: '', type: 'image', url: '', seconds: 8 });
  const [busy, setBusy] = useState(null);

  const run = async (key, fn) => {
    setBusy(key);
    try {
      await fn();
      onError(null);
    } catch (err) {
      onError(err.message);
    } finally {
      setBusy(null);
    }
  };

  const sync = snapshot.activeSync;
  const now = serverNow();
  const syncPending = sync && now < sync.startAtMs;
  const syncLive = sync && now >= sync.startAtMs && now < sync.startAtMs + sync.durationMs;
  const syncMediaName = sync
    ? (snapshot.media.find((m) => m.id === sync.mediaId)?.name ?? sync.mediaId)
    : null;

  return (
    <aside className="panel">
      <section className="panel__block">
        <h2 className="panel__title">Sync takeover</h2>
        <p className="panel__hint">
          Every window switches to the chosen media at the same instant, then returns to its own
          sequence without losing its place.
        </p>

        <label className="field">
          <span className="field__label">Media</span>
          <select
            className="field__control"
            value={syncMediaId}
            onChange={(e) => setSyncMediaId(e.target.value)}
          >
            {snapshot.media.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name} · {m.type}
              </option>
            ))}
          </select>
        </label>

        <label className="field">
          <span className="field__label">Hold for {syncSeconds}s</span>
          <input
            className="field__range"
            type="range"
            min="3"
            max="60"
            value={syncSeconds}
            onChange={(e) => setSyncSeconds(Number(e.target.value))}
          />
        </label>

        <div className="panel__actions">
          <button
            type="button"
            className="btn btn--signal"
            disabled={!syncMediaId || busy === 'sync'}
            onClick={() => run('sync', () => actions.triggerSync(syncMediaId, syncSeconds))}
          >
            {busy === 'sync' ? 'Scheduling…' : 'Sync all windows'}
          </button>
          {sync && (
            <button
              type="button"
              className="btn btn--ghost"
              disabled={busy === 'cancel'}
              onClick={() => run('cancel', actions.cancelSync)}
            >
              Cancel
            </button>
          )}
        </div>

        {(syncPending || syncLive) && (
          <p className="panel__state">
            {syncPending
              ? `${syncMediaName} starts in ${Math.max(0, Math.round((sync.startAtMs - now) / 100) / 10)}s`
              : `${syncMediaName} live on every window · ${timecode(sync.startAtMs + sync.durationMs - now)} left`}
          </p>
        )}
      </section>

      <section className="panel__block">
        <h2 className="panel__title">Add to a playlist</h2>
        <p className="panel__hint">
          Changes reach every open display immediately. The window keeps playing while its list
          changes.
        </p>

        <label className="field">
          <span className="field__label">Window</span>
          <select
            className="field__control"
            value={targetWindowId}
            onChange={(e) => setTargetWindowId(e.target.value)}
          >
            {snapshot.windows.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name}
              </option>
            ))}
          </select>
        </label>

        <label className="field">
          <span className="field__label">Media</span>
          <select
            className="field__control"
            value={addMediaId}
            onChange={(e) => setAddMediaId(e.target.value)}
          >
            {snapshot.media.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name} · {humanDuration(m.defaultDurationMs)}
              </option>
            ))}
          </select>
        </label>

        <label className="field">
          <span className="field__label">Duration in seconds</span>
          <input
            className="field__control"
            type="number"
            min="1"
            max="600"
            value={addSeconds}
            onChange={(e) => setAddSeconds(Number(e.target.value))}
          />
        </label>

        <button
          type="button"
          className="btn"
          disabled={!targetWindowId || !addMediaId || busy === 'add'}
          onClick={() =>
            run('add', () => actions.addPlaylistItem(targetWindowId, addMediaId, addSeconds))
          }
        >
          {busy === 'add' ? 'Adding…' : 'Add to window'}
        </button>
      </section>

      <section className="panel__block">
        <h2 className="panel__title">New media</h2>

        <label className="field">
          <span className="field__label">Name</span>
          <input
            className="field__control"
            type="text"
            placeholder="M7"
            value={newMedia.name}
            onChange={(e) => setNewMedia({ ...newMedia, name: e.target.value })}
          />
        </label>

        <label className="field">
          <span className="field__label">Kind</span>
          <select
            className="field__control"
            value={newMedia.type}
            onChange={(e) => setNewMedia({ ...newMedia, type: e.target.value })}
          >
            <option value="image">Image</option>
            <option value="video">Video</option>
            <option value="blank">Blank</option>
          </select>
        </label>

        {newMedia.type !== 'blank' && (
          <label className="field">
            <span className="field__label">Source URL</span>
            <input
              className="field__control"
              type="url"
              placeholder="https://…"
              value={newMedia.url}
              onChange={(e) => setNewMedia({ ...newMedia, url: e.target.value })}
            />
          </label>
        )}

        <label className="field">
          <span className="field__label">Default duration in seconds</span>
          <input
            className="field__control"
            type="number"
            min="1"
            max="600"
            value={newMedia.seconds}
            onChange={(e) => setNewMedia({ ...newMedia, seconds: Number(e.target.value) })}
          />
        </label>

        <button
          type="button"
          className="btn"
          disabled={busy === 'media'}
          onClick={() =>
            run('media', async () => {
              await actions.addMedia({
                name: newMedia.name,
                type: newMedia.type,
                url: newMedia.type === 'blank' ? '' : newMedia.url,
                defaultDurationSeconds: newMedia.seconds,
              });
              setNewMedia({ name: '', type: 'image', url: '', seconds: 8 });
            })
          }
        >
          {busy === 'media' ? 'Saving…' : 'Add to library'}
        </button>
      </section>

      <section className="panel__block">
        <h2 className="panel__title">Cycle</h2>
        <p className="panel__hint">
          Restart the {humanDuration(snapshot.cycleMs)} cycle now. Every window jumps back to the
          first item in its list together.
        </p>
        <button
          type="button"
          className="btn btn--ghost"
          disabled={busy === 'reset'}
          onClick={() => run('reset', actions.resetCycle)}
        >
          {busy === 'reset' ? 'Restarting…' : 'Restart cycle'}
        </button>
      </section>
    </aside>
  );
}
