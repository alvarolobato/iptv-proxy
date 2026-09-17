import {
  EuiButton,
  EuiButtonEmpty,
  EuiCallOut,
  EuiFlexGroup,
  EuiFlexItem,
  EuiIcon,
  EuiLink,
  EuiLoadingSpinner,
  EuiModal,
  EuiModalBody,
  EuiModalHeader,
  EuiModalHeaderTitle,
  EuiSpacer,
  EuiText,
  EuiTitle,
} from '@elastic/eui';
import mpegts from 'mpegts.js';
import { Fragment, useEffect, useMemo, useRef, useState } from 'react';

import { VLC_APP_STORE_URL, buildM3u, detectPlatform, externalPlayerLinks, isBlockedMixedContent, m3uFileName, streamKind } from './streamLinks';

// mpegts.js logs every segment at info/debug level by default; keep only warnings and errors.
mpegts.LoggingControl.applyConfig({ enableDebug: false, enableVerbose: false, enableInfo: false });

// Buffered live MPEG-TS settings (see ADR-016). Latency chasing stays on because nothing else paces live
// loading: with it off, a provider that bursts faster than realtime fills the SourceBuffer, mpegts.js suspends
// the transmuxer and never resumes for live, freezing playback. But the library defaults chase 1.5 s with 0.5 s
// remaining, which stalls constantly on IPTV streams (measured: 28 stalls in 45 s, vs 2 short ones with the
// limits below). Range loading keeps the mpegts.js defaults: the first request then carries no Range or custom
// headers, so cross-origin requests to the proxy port need no CORS preflight.
export const MPEGTS_CONFIG = {
  enableWorker: true,
  enableStashBuffer: true,
  stashInitialSize: 1024 * 1024,
  lazyLoad: false,
  liveBufferLatencyChasing: true,
  liveBufferLatencyMaxLatency: 10,
  liveBufferLatencyMinRemain: 4,
  autoCleanupSourceBuffer: true,
  autoCleanupMaxBackwardDuration: 60,
  autoCleanupMinBackwardDuration: 30,
};

// Exposed so an e2e test can assert these settings survive refactors (e2e blocks real streams, so playback
// itself can't guard them).
if (typeof window !== 'undefined') {
  window.__mpegtsConfig = MPEGTS_CONFIG;
}

// If nothing plays after this long (channel offline, provider connection limit, unsupported codec),
// point the viewer to the external player options.
const STALL_TIMEOUT_MS = 15000;

// While playing, a picture that stops advancing for this long counts as frozen.
const FREEZE_TIMEOUT_MS = 12000;
const FREEZE_CHECK_INTERVAL_MS = 2000;

function canPlayInBrowser(kind) {
  if (kind === 'mpegts') return mpegts.getFeatureList().mseLivePlayback;
  if (kind === 'progressive') return true;
  if (kind === 'hls') return document.createElement('video').canPlayType('application/vnd.apple.mpegurl') !== '';
  return false;
}

function describeMpegtsError(type, detail, info) {
  if (type === mpegts.ErrorTypes.NETWORK_ERROR) {
    return `The stream could not be loaded (${detail}${info?.code ? ` ${info.code}` : ''}). The channel may be offline or the provider connection limit reached.`;
  }
  if (type === mpegts.ErrorTypes.MEDIA_ERROR) {
    return `This browser can't play this channel's format (${info?.msg || detail}). Channels with HEVC video or AC-3/MP2 audio often need an external player.`;
  }
  return `Playback failed (${detail || type}).`;
}

function describeMediaError(err) {
  if (err?.code === 3 || err?.code === 4) return "This browser can't decode this stream's format.";
  if (err?.code === 2) return 'Network error while loading the stream.';
  return 'Playback failed.';
}

// copyText copies with the Clipboard API when available (secure contexts only) and falls back to
// execCommand, which also works when the UI is served over plain http on a LAN hostname.
export function copyText(text) {
  const str = String(text);
  const fallback = () =>
    new Promise((resolve, reject) => {
      const ta = document.createElement('textarea');
      ta.value = str;
      ta.setAttribute('readonly', '');
      ta.style.position = 'absolute';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      try {
        if (document.execCommand('copy')) resolve();
        else reject(new Error('copy failed'));
      } catch (e) {
        reject(e);
      } finally {
        document.body.removeChild(ta);
      }
    });
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(str).catch(fallback);
  }
  return fallback();
}

function InBrowserPlayer({ url, kind, focusOnMount = false }) {
  const videoRef = useRef(null);

  // After "Resume in browser" the resume button is gone; move focus into the player so keyboard
  // users (and Escape to close the sheet) keep working.
  useEffect(() => {
    if (focusOnMount) videoRef.current?.focus({ preventScroll: true });
  }, [focusOnMount]);
  const supported = useMemo(() => canPlayInBrowser(kind), [kind]);
  const [status, setStatus] = useState('loading'); // loading | playing | paused | stalled | error
  const [error, setError] = useState('');
  const [muted, setMuted] = useState(true);
  const [mediaInfo, setMediaInfo] = useState(null);

  // Any loading period (initial or after tapping Play) that doesn't start in time becomes a stall notice.
  useEffect(() => {
    if (!supported || status !== 'loading') return undefined;
    const timer = setTimeout(() => setStatus((s) => (s === 'loading' ? 'stalled' : s)), STALL_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [status, supported]);

  // A picture that freezes while "playing" (provider stopped sending, or mpegts.js suspended loading) raises no
  // error and not always a 'waiting' event, so watch playback progress and offer the external players instead.
  useEffect(() => {
    if (!supported || status !== 'playing') return undefined;
    const video = videoRef.current;
    if (!video) return undefined;
    let lastTime = video.currentTime;
    let lastProgressAt = Date.now();
    const timer = setInterval(() => {
      if (video.paused) {
        lastProgressAt = Date.now();
        return;
      }
      if (video.currentTime > lastTime + 0.05) {
        lastTime = video.currentTime;
        lastProgressAt = Date.now();
        return;
      }
      if (Date.now() - lastProgressAt >= FREEZE_TIMEOUT_MS) setStatus('stalled');
    }, FREEZE_CHECK_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [status, supported]);

  useEffect(() => {
    const video = videoRef.current;
    if (!supported || !video) return undefined;
    // Required for ManagedMediaSource (MSE) on iPhone Safari.
    video.disableRemotePlayback = true;
    setStatus('loading');
    setError('');
    setMediaInfo(null);

    let player = null;
    let disposed = false;
    let failed = false;

    // teardown stops the stream request so the proxy releases the provider connection. Idempotent.
    const teardown = () => {
      if (disposed) return;
      disposed = true;
      if (player) {
        try {
          player.pause();
          player.unload();
          player.detachMediaElement();
          player.destroy();
        } catch {
          /* already torn down */
        }
        player = null;
      }
      video.pause();
      video.removeAttribute('src');
      video.load();
    };

    // Errors end in-browser playback: mpegts.js keeps downloading after media errors, which would hold a
    // provider connection that the external player options need. Tear down outside the emitter callback.
    const fail = (message) => {
      if (disposed || failed) return;
      failed = true;
      setStatus('error');
      setError(message);
      setTimeout(teardown, 0);
    };

    const onPlaying = () => {
      if (!disposed && !failed) setStatus('playing');
    };
    const onVolumeChange = () => setMuted(video.muted);
    const onVideoError = () => {
      if (!disposed) fail(describeMediaError(video.error));
    };
    video.addEventListener('playing', onPlaying);
    video.addEventListener('volumechange', onVolumeChange);
    video.addEventListener('error', onVideoError);

    if (kind === 'mpegts') {
      player = mpegts.createPlayer({ type: 'mpegts', isLive: true, url }, MPEGTS_CONFIG);
      player.on(mpegts.Events.ERROR, (type, detail, info) => fail(describeMpegtsError(type, detail, info)));
      player.on(mpegts.Events.MEDIA_INFO, (info) => setMediaInfo(info));
      player.attachMediaElement(video);
      player.load();
    } else {
      video.src = url;
    }
    // Muted autoplay is normally allowed. When the browser still blocks it (e.g. iOS Low Power Mode),
    // show a Play button instead of reporting a stall.
    video.play()?.catch((err) => {
      if (!disposed && !failed && err?.name === 'NotAllowedError') setStatus('paused');
    });

    return () => {
      video.removeEventListener('playing', onPlaying);
      video.removeEventListener('volumechange', onVolumeChange);
      video.removeEventListener('error', onVideoError);
      teardown();
    };
  }, [url, kind, supported]);

  const unmute = () => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = false;
    video.play()?.catch(() => {});
  };

  const startPlayback = () => {
    const video = videoRef.current;
    if (!video) return;
    // The Play button unmounts on click; keep focus inside the sheet.
    video.focus({ preventScroll: true });
    setStatus('loading');
    video.play()?.catch((err) => {
      if (err?.name === 'NotAllowedError') setStatus('paused');
    });
  };

  if (!supported) {
    return (
      <EuiCallOut size="s" color="warning" iconType="warning" title="This browser can't play this stream" data-testid="player-unsupported">
        <p>Use one of the options below to watch it in another app.</p>
      </EuiCallOut>
    );
  }

  const codecs = mediaInfo ? [mediaInfo.videoCodec, mediaInfo.audioCodec].filter(Boolean).join(' · ') : '';

  return (
    <Fragment>
      <div data-testid="player-frame" data-status={status} style={{ position: 'relative', background: '#000', borderRadius: 6, overflow: 'hidden', aspectRatio: '16 / 9', maxWidth: '100%' }}>
        <video
          ref={videoRef}
          data-testid="player-video"
          controls
          playsInline
          autoPlay
          muted
          style={{ width: '100%', height: '100%', display: 'block' }}
        />
        {status === 'loading' && (
          <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', pointerEvents: 'none' }}>
            <EuiLoadingSpinner size="xl" />
          </div>
        )}
        {status === 'paused' && (
          <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <EuiButton fill iconType="play" onClick={startPlayback} data-testid="player-start">
              Play
            </EuiButton>
          </div>
        )}
        {muted && (status === 'loading' || status === 'playing' || status === 'stalled') && (
          <EuiButton size="s" fill color="text" onClick={unmute} style={{ position: 'absolute', top: 8, left: 8 }} data-testid="player-unmute">
            Tap to unmute
          </EuiButton>
        )}
      </div>
      {codecs && (
        <EuiText size="xs" color="subdued" style={{ marginTop: 4 }}>
          {codecs}
          {mediaInfo?.width ? ` · ${mediaInfo.width}×${mediaInfo.height}` : ''}
        </EuiText>
      )}
      {(status === 'error' || status === 'stalled') && (
        <Fragment>
          <EuiSpacer size="s" />
          <EuiCallOut
            size="s"
            color={status === 'error' ? 'danger' : 'warning'}
            iconType="warning"
            title={status === 'error' ? 'Playback failed' : 'The stream is taking too long to start'}
            data-testid="player-error"
          >
            <p>{status === 'error' ? error : 'The channel may be offline, busy or use a format this browser cannot play.'} Try &quot;Open in VLC&quot; or the .m3u download below.</p>
          </EuiCallOut>
        </Fragment>
      )}
    </Fragment>
  );
}

// LinkButton is a button-styled native <a href>: EuiButton/EuiButtonEmpty with href block custom URL
// schemes (about:blank#blocked), so deep links must be plain anchors. title shows the target on hover.
function LinkButton({ href, primary, testId, onClick, children }) {
  return (
    <a
      href={href}
      title={href}
      onClick={onClick}
      data-testid={testId}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 8,
        minHeight: 40,
        padding: '0 16px',
        borderRadius: 6,
        fontWeight: 500,
        textDecoration: 'none',
        background: primary ? '#07C' : '#e6f1fa',
        color: primary ? '#fff' : '#0061a6',
      }}
    >
      <EuiIcon type="popout" size="m" />
      {children}
    </a>
  );
}

const PLATFORM_HINTS = {
  ios: (
    <Fragment>
      Needs <EuiLink href={VLC_APP_STORE_URL} target="_blank" external>VLC for iOS</EuiLink>. If the first link does nothing, try the alternative link.
    </Fragment>
  ),
  android: 'If VLC is not installed, "Open in VLC" opens its Play Store page.',
  desktop: 'Open the downloaded .m3u with VLC, IINA or mpv, or paste the copied URL in VLC → Open Network Stream.',
};

// PlayerSheet plays a channel in the browser and offers external players for formats or devices
// the browser can't handle. EuiModal is full-screen on small viewports.
export function PlayerSheet({ channel, onClose }) {
  const url = channel?.stream_url || '';
  const name = channel?.name || 'Stream';
  const kind = streamKind(url);
  const platform = useMemo(() => detectPlatform(navigator.userAgent, navigator.maxTouchPoints), []);
  const links = externalPlayerLinks(url, platform);
  const mixedContent = isBlockedMixedContent(window.location.protocol, url);
  const [copyLabel, setCopyLabel] = useState('');
  // Opening an external player stops in-browser playback first: providers allow few simultaneous
  // connections and the external app needs one.
  const [stopped, setStopped] = useState(false);
  const [resumed, setResumed] = useState(false);
  const stopInBrowser = () => setStopped(true);
  const resumeInBrowser = () => {
    setStopped(false);
    setResumed(true);
  };

  const downloadM3u = () => {
    stopInBrowser();
    const href = URL.createObjectURL(new Blob([buildM3u(name, url)], { type: 'audio/x-mpegurl' }));
    const a = document.createElement('a');
    a.href = href;
    a.download = m3uFileName(name);
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
  };

  const copyUrl = () => {
    // The copied URL is meant for another player, which needs the connection.
    stopInBrowser();
    return copyText(url).then(() => setCopyLabel('Copied'), () => setCopyLabel('Copy failed'));
  };

  let playerArea;
  if (kind === 'external') {
    playerArea = (
      <EuiCallOut size="s" color="warning" iconType="warning" title="This format can't play in the browser" data-testid="player-unsupported">
        <p>Use one of the options below to watch it in another app.</p>
      </EuiCallOut>
    );
  } else if (mixedContent) {
    playerArea = (
      <EuiCallOut size="s" color="warning" iconType="warning" title="The browser blocks this stream here" data-testid="player-mixed-content">
        <p>This page is served over HTTPS but the stream uses plain HTTP, so the browser won&apos;t load it. Use one of the options below, or serve the proxy streams over HTTPS.</p>
      </EuiCallOut>
    );
  } else if (stopped) {
    playerArea = (
      <EuiCallOut size="s" title="Browser playback stopped" data-testid="player-stopped">
        <p>Stopped here so the other app can use the connection (providers allow only a few at a time).</p>
        <EuiButton size="s" iconType="play" onClick={resumeInBrowser} data-testid="player-resume">
          Resume in browser
        </EuiButton>
      </EuiCallOut>
    );
  } else {
    playerArea = <InBrowserPlayer key={url} url={url} kind={kind} focusOnMount={resumed} />;
  }

  return (
    <EuiModal onClose={onClose} style={{ width: 'min(960px, 100vw)' }} aria-labelledby="player-sheet-title" data-testid="player-sheet">
      <EuiModalHeader>
        <EuiModalHeaderTitle size="s" id="player-sheet-title" style={{ overflowWrap: 'anywhere' }}>
          {name}
        </EuiModalHeaderTitle>
      </EuiModalHeader>
      <EuiModalBody>
        {channel?.group && (
          <Fragment>
            <EuiText size="s" color="subdued">{channel.group}</EuiText>
            <EuiSpacer size="s" />
          </Fragment>
        )}
        {playerArea}
        <EuiSpacer size="m" />
        <EuiTitle size="xxs">
          <h3>Watch in another app</h3>
        </EuiTitle>
        <EuiSpacer size="s" />
        <EuiFlexGroup direction="column" gutterSize="s" responsive={false}>
          {links.map((l, i) => (
            <EuiFlexItem key={l.id}>
              <LinkButton href={l.href} primary={i === 0} testId={`player-link-${l.id}`} onClick={stopInBrowser}>
                {l.label}
              </LinkButton>
            </EuiFlexItem>
          ))}
          <EuiFlexItem>
            <EuiButton iconType="download" fill={platform === 'desktop'} onClick={downloadM3u} fullWidth data-testid="player-download-m3u">
              Download .m3u
            </EuiButton>
          </EuiFlexItem>
          <EuiFlexItem>
            <EuiButtonEmpty iconType="copyClipboard" onClick={copyUrl} data-testid="player-copy-url">
              {copyLabel || 'Copy stream URL'}
            </EuiButtonEmpty>
          </EuiFlexItem>
        </EuiFlexGroup>
        <EuiSpacer size="s" />
        <EuiText size="xs" color="subdued">
          <p>{PLATFORM_HINTS[platform]}</p>
        </EuiText>
      </EuiModalBody>
    </EuiModal>
  );
}

// PlayStreamLink is a native <a href> to the stream (middle-click and "copy link" still work) that calls
// onPlay(channel) on a plain click instead of navigating to the raw stream. The parent renders a single
// PlayerSheet outside any table, so layout changes (e.g. rotating a phone) don't unmount the player.
export function PlayStreamLink({ channel, onPlay, iconSize = 's', style, testId, ariaLabel = 'Open stream', ...rest }) {
  if (!channel?.stream_url) return null;
  const onClick = (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    onPlay(channel);
  };
  return (
    <a {...rest} href={channel.stream_url} onClick={onClick} aria-label={ariaLabel} title={channel.stream_url} data-testid={testId} style={style}>
      <EuiIcon type="play" size={iconSize} />
    </a>
  );
}
