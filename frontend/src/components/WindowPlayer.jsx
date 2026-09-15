import { useEffect, useState } from 'react';
import MediaSurface from './MediaSurface.jsx';
import {
  SOURCE_SYNC,
  humanDuration,
  resolve,
  resolveSequence,
  timecode,
  upNext,
} from '../lib/timeline.js';

export default function WindowPlayer({
  snapshot,
  window: win,
  serverNow,
  solo = false,
  onRemoveItem,
  onOpenSolo,
}) {
  // Safely handle missing/null playlist while state is loading.
  const playlist = Array.isArray(win?.playlist) ? win.playlist : [];

  const [playback, setPlayback] = useState(() =>
    resolve(snapshot, win, serverNow())
  );

  const [tick, setTick] = useState(() => serverNow());

  useEffect(() => {
    let timer;

    const step = () => {
      const now = serverNow();
      const next = resolve(snapshot, win, now);

      setPlayback(next);
      setTick(now);

      const delay = Math.max(
        30,
        next.endsAtMs - now + 20
      );

      timer = setTimeout(step, delay);
    };

    step();

    return () => clearTimeout(timer);
  }, [snapshot, win, serverNow]);

  useEffect(() => {
    const id = setInterval(
      () => setTick(serverNow()),
      250
    );

    return () => clearInterval(id);
  }, [serverNow]);

  const isSync = playback.source === SOURCE_SYNC;

  const slotMs = Math.max(
    1,
    playback.endsAtMs - playback.startedAtMs
  );

  const elapsedMs = Math.min(
    slotMs,
    Math.max(
      0,
      tick - playback.startedAtMs
    )
  );

  const progress = elapsedMs / slotMs;

  const remainingMs = Math.max(
    0,
    playback.endsAtMs - tick
  );

  const ownSequence = resolveSequence(
    snapshot,
    win,
    tick
  );

  const next = upNext(
    snapshot,
    win,
    tick
  );

  return (
    <article
      className={`player${
        isSync ? ' player--sync' : ''
      }${
        solo ? ' player--solo' : ''
      }`}
    >
      <header className="player__head">
        <h2 className="player__name">
          {win.name}
        </h2>

        <div className="player__headMeta">
          {isSync && (
            <span className="tally">
              Sync
            </span>
          )}

          <span className="player__count">
            {playlist.length} items
          </span>

          {!solo && onOpenSolo && (
            <button
              type="button"
              className="btn btn--ghost btn--small"
              onClick={() => onOpenSolo(win.id)}
              title="Open this window on its own — use it to verify sync across separate browser windows"
            >
              Detach
            </button>
          )}
        </div>
      </header>

      <div className="player__stage">
        <MediaSurface
          playback={playback}
          nowMs={tick}
        />

        <div className="player__overlay">
          <span className="player__nowLabel">
            {playback.media
              ? playback.media.name
              : 'Nothing scheduled'}
          </span>

          <span className="player__clock">
            {timecode(remainingMs)}
          </span>
        </div>

        <div
          className="player__progress"
          aria-hidden="true"
        >
          <span
            className="player__progressFill"
            style={{
              transform: `scaleX(${progress})`,
            }}
          />
        </div>
      </div>

      <div className="player__status">
        {isSync ? (
          <p className="player__statusLine player__statusLine--sync">
            Taken over by sync. Own sequence is still running on{' '}
            <strong>
              {ownSequence.media?.name ?? 'nothing'}
            </strong>{' '}
            and resumes in{' '}
            {timecode(remainingMs)}.
          </p>
        ) : (
          <p className="player__statusLine">
            Next up{' '}
            <strong>
              {next?.media?.name ?? '—'}
            </strong>

            {next &&
              ` for ${humanDuration(
                next.endsAtMs - next.startedAtMs
              )}`}
          </p>
        )}
      </div>

      <ol className="strip">
        {playlist.map((item, index) => {
          const media = Array.isArray(snapshot?.media)
            ? snapshot.media.find(
                (m) => m.id === item.mediaId
              )
            : null;

          const active =
            !isSync &&
            index === playback.itemIndex;

          return (
            <li
              key={item.id}
              className={`strip__item${
                active
                  ? ' strip__item--active'
                  : ''
              }`}
            >
              <span className="strip__name">
                {media?.name ?? 'Missing media'}
              </span>

              <span className="strip__time">
                {humanDuration(item.durationMs)}
              </span>

              {onRemoveItem && (
                <button
                  type="button"
                  className="strip__remove"
                  onClick={() =>
                    onRemoveItem(
                      win.id,
                      item.id
                    )
                  }
                  aria-label={`Remove ${
                    media?.name ?? 'item'
                  } from ${win.name}`}
                >
                  ×
                </button>
              )}
            </li>
          );
        })}

        {playlist.length === 0 && (
          <li className="strip__empty">
            Add media to start this window playing.
          </li>
        )}
      </ol>
    </article>
  );
}