import { useEffect, useState } from 'react';
import { cyclePosition, timecode } from '../lib/timeline.js';


export default function StatusBar({ snapshot, serverNow, connection, clock }) {
  const [now, setNow] = useState(() => serverNow());

  useEffect(() => {
    const id = setInterval(() => setNow(serverNow()), 500);
    return () => clearInterval(id);
  }, [serverNow]);

  const { cycleIndex, offsetInCycleMs } = cyclePosition(snapshot, now);
  const cycleProgress = offsetInCycleMs / snapshot.cycleMs;
  const sync = snapshot.activeSync;
  const syncLive = sync && now >= sync.startAtMs && now < sync.startAtMs + sync.durationMs;

  const connectionCopy = {
    live: 'Live',
    connecting: 'Connecting',
    reconnecting: 'Reconnecting',
  }[connection];

  return (
    <div className={`status${syncLive ? ' status--sync' : ''}`}>
      <div className="status__brand">
        <span className={`status__lamp status__lamp--${connection}`} aria-hidden="true" />
        <div>
          <h1 className="status__title">Sequencer</h1>
          <p className="status__subtitle">
            {snapshot.windows.length} windows · {connectionCopy}
          </p>
        </div>
      </div>

      <div className="status__cycle">
        <div className="status__cycleHead">
          <span>Cycle {cycleIndex + 1}</span>
          <span className="status__timecode">
            {timecode(offsetInCycleMs)} / {timecode(snapshot.cycleMs)}
          </span>
        </div>
        <div className="status__track">
          <span className="status__trackFill" style={{ transform: `scaleX(${cycleProgress})` }} />
        </div>
        <p className="status__cycleFoot">
          Restarts in {timecode(snapshot.cycleMs - offsetInCycleMs)}
        </p>
      </div>

      <div className="status__clock">
        <span className="status__timecode status__timecode--large">
          {new Date(now).toLocaleTimeString([], { hour12: false })}
        </span>
        <p className="status__clockFoot">
          Clock offset {Math.round(clock.offsetMs)} ms · round trip {Math.round(clock.roundTripMs)} ms
        </p>
      </div>
    </div>
  );
}
