
import { resolve } from '../src/lib/timeline.js';

const base = (process.argv[2] ?? 'http://localhost:8080').replace(/\/$/, '');

const state = await fetch(`${base}/api/state`).then((r) => r.json());

let failures = 0;

for (const win of state.windows) {
  const remote = await fetch(`${base}/api/windows/${win.id}/now`).then((r) => r.json());

  const local = resolve(state, win, remote.serverTimeMs);
  const expected = remote.playback;

  const same =
    local.source === expected.source &&
    local.mediaId === (expected.mediaId || null) &&
    local.itemIndex === expected.itemIndex &&
    local.startedAtMs === expected.startedAtMs &&
    local.endsAtMs === expected.endsAtMs;

  if (same) {
    console.log(`  ok   ${win.id}  ${local.source}/${local.mediaId} idx=${local.itemIndex}`);
  } else {
    failures += 1;
    console.error(`  FAIL ${win.id}`);
    console.error('    go:', JSON.stringify(expected));
    console.error('    js:', JSON.stringify(local));
  }
}

if (failures > 0) {
  console.error(`\n${failures} window(s) disagree between the Go and JS timelines.`);
  process.exit(1);
}
console.log('\nGo and JS timelines agree on every window.');
