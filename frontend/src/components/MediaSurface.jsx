import { useEffect, useRef } from 'react';

export default function MediaSurface({ playback, nowMs }) {
  const videoRef = useRef(null);
  const media = playback?.media;
  const mediaId = media?.id ?? null;
  const itemKey = `${playback?.itemId ?? 'none'}:${playback?.startedAtMs ?? 0}`;

  useEffect(() => {
    const video = videoRef.current;
    if (!video || media?.type !== 'video') return;

    const elapsedSeconds = Math.max(0, (nowMs - playback.startedAtMs) / 1000);

    const seekAndPlay = () => {
      // Clips shorter than their slot loop, so the seek target wraps.
      const target = video.duration
        ? elapsedSeconds % video.duration
        : elapsedSeconds;
      if (Math.abs(video.currentTime - target) > 0.35) {
        video.currentTime = target;
      }
      const attempt = video.play();
      if (attempt?.catch) attempt.catch(() => {});
    };

    if (video.readyState >= 1) {
      seekAndPlay();
    } else {
      video.addEventListener('loadedmetadata', seekAndPlay, { once: true });
    }

  }, [itemKey, mediaId]);

  if (!media || media.type === 'blank') {
    return (
      <div className="surface surface--blank" role="img" aria-label="Blank slot">
        <span className="surface__blankMark" aria-hidden="true" />
        <span className="surface__blankText">
          {playback?.source === 'idle' ? 'No playlist configured' : 'Blank slot'}
        </span>
      </div>
    );
  }

  if (media.type === 'video') {
    return (
      <video
        key={itemKey}
        ref={videoRef}
        className="surface surface--video"
        src={media.url}
        muted
        playsInline
        loop
        preload="auto"
        aria-label={media.name}
      />
    );
  }

  return (
    <img
      className="surface surface--image"
      src={media.url}
      alt={media.name}
      loading="eager"
      decoding="async"
    />
  );
}
