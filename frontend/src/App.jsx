import { useCallback, useEffect, useState } from 'react';
import ControlPanel from './components/ControlPanel.jsx';
import StatusBar from './components/StatusBar.jsx';
import WindowPlayer from './components/WindowPlayer.jsx';
import { useSequencer } from './lib/useSequencer.js';


function useSoloWindowId() {
  const read = () => new URLSearchParams(window.location.search).get('window');
  const [id, setId] = useState(read);

  useEffect(() => {
    const onPop = () => setId(read());
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);

  return id;
}

export default function App() {
  const { snapshot, connection, error, clock, serverNow, actions, setError } = useSequencer();
  const soloWindowId = useSoloWindowId();

  const openSolo = useCallback((windowId) => {
    window.open(`${window.location.pathname}?window=${windowId}`, '_blank', 'noopener');
  }, []);

  const removeItem = useCallback(
    async (windowId, itemId) => {
      try {
        await actions.removePlaylistItem(windowId, itemId);
        setError(null);
      } catch (err) {
        setError(err.message);
      }
    },
    [actions, setError],
  );

  if (!snapshot) {
    return (
      <main className="boot">
        <p className="boot__text">
          {error ? `Cannot reach the backend: ${error}` : 'Loading playout state…'}
        </p>
        {error && (
          <p className="boot__hint">
            Start the Go server, then reload. It listens on port 8080 by default.
          </p>
        )}
      </main>
    );
  }

  if (soloWindowId) {
    const win = snapshot.windows.find((w) => w.id === soloWindowId);
    if (!win) {
      return (
        <main className="boot">
          <p className="boot__text">No window with id “{soloWindowId}”.</p>
        </main>
      );
    }
    return (
      <main className="solo">
        <WindowPlayer snapshot={snapshot} window={win} serverNow={serverNow} solo />
      </main>
    );
  }

  return (
    <div className="app">
      <StatusBar
        snapshot={snapshot}
        serverNow={serverNow}
        connection={connection}
        clock={clock}
      />

      {error && (
        <p className="alert" role="alert">
          {error}
        </p>
      )}

      <div className="app__body">
        <main className="grid">
          {snapshot.windows.map((win) => (
            <WindowPlayer
              key={win.id}
              snapshot={snapshot}
              window={win}
              serverNow={serverNow}
              onRemoveItem={removeItem}
              onOpenSolo={openSolo}
            />
          ))}
        </main>

        <ControlPanel
          snapshot={snapshot}
          serverNow={serverNow}
          actions={actions}
          onError={setError}
        />
      </div>
    </div>
  );
}
