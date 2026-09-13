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

import { VLC_APP_STORE_URL, buildM3u, detectPlatform, externalPlayerLinks, m3uFileName, streamKind } from './streamLinks';

// mpegts.js logs every segment at info/debug level by default; keep only warnings and errors.
mpegts.LoggingControl.applyConfig({ enableDebug: false, enableVerbose: false, enableInfo: false });

// Low-latency live MPEG-TS settings. The first request carries no Range or custom headers
// (rangeLoadZeroStart false), so cross-origin requests to the proxy port need no CORS preflight.
const MPEGTS_CONFIG = {
  enableWorker: true,
  enableStashBuffer: false,
  lazyLoad: false,
  liveBufferLatencyChasing: true,
  autoCleanupSourceBuffer: true,
  rangeLoadZeroStart: false,
};

// If nothing plays after this long (channel offline, provider connection limit, unsupported codec),
// point the viewer to the external player options.
const STALL_TIMEOUT_MS = 15000;

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

function InBrowserPlayer({ url, kind }) {
  const videoRef = useRef(null);
  const supported = useMemo(() => canPlayInBrowser(kind), [kind]);
  const [status, setStatus] = useState('loading'); // loading | playing | stalled | error
  const [error, setError] = useState('');
  const [muted, setMuted] = useState(true);
  const [mediaInfo, setMediaInfo] = useState(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!supported || !video) return undefined;
    // Required for ManagedMediaSource (MSE) on iPhone Safari.
    video.disableRemotePlayback = true;
    setStatus('loading');
    setError('');
    setMediaInfo(null);

    const stallTimer = setTimeout(() => setStatus((s) => (s === 'loading' ? 'stalled' : s)), STALL_TIMEOUT_MS);
    const onPlaying = () => setStatus('playing');
    const onVolumeChange = () => setMuted(video.muted);
    const onVideoError = () => {
      setStatus('error');
      setError(describeMediaError(video.error));
    };
    video.addEventListener('playing', onPlaying);
    video.addEventListener('volumechange', onVolumeChange);
    video.addEventListener('error', onVideoError);

    let player = null;
    if (kind === 'mpegts') {
      player = mpegts.createPlayer({ type: 'mpegts', isLive: true, url }, MPEGTS_CONFIG);
      player.on(mpegts.Events.ERROR, (type, detail, info) => {
        setStatus('error');
        setError(describeMpegtsError(type, detail, info));
      });
      player.on(mpegts.Events.MEDIA_INFO, (info) => setMediaInfo(info));
      player.attachMediaElement(video);
      player.load();
    } else {
      video.src = url;
    }
    // Muted autoplay is allowed on mobile; if the browser still blocks it, the native controls remain.
    video.play()?.catch(() => {});

    return () => {
      clearTimeout(stallTimer);
      video.removeEventListener('playing', onPlaying);
      video.removeEventListener('volumechange', onVolumeChange);
      video.removeEventListener('error', onVideoError);
      // Tear down fully so the proxy (and the provider connection behind it) is released.
      if (player) {
        try {
          player.pause();
          player.unload();
          player.detachMediaElement();
          player.destroy();
        } catch {
          /* already torn down */
        }
      } else {
        video.pause();
        video.removeAttribute('src');
        video.load();
      }
    };
  }, [url, kind, supported]);

  const unmute = () => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = false;
    video.play()?.catch(() => {});
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
      <div style={{ position: 'relative', background: '#000', borderRadius: 6, overflow: 'hidden', aspectRatio: '16 / 9', maxWidth: '100%' }}>
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
        {muted && status !== 'error' && (
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
// schemes (about:blank#blocked), so deep links must be plain anchors.
function LinkButton({ href, primary, testId, children }) {
  return (
    <a
      href={href}
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
  const [copyLabel, setCopyLabel] = useState('');

  const downloadM3u = () => {
    const href = URL.createObjectURL(new Blob([buildM3u(name, url)], { type: 'audio/x-mpegurl' }));
    const a = document.createElement('a');
    a.href = href;
    a.download = m3uFileName(name);
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
  };

  const copyUrl = () => copyText(url).then(() => setCopyLabel('Copied'), () => setCopyLabel('Copy failed'));

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
        {kind === 'external' ? (
          <EuiCallOut size="s" color="warning" iconType="warning" title="This format can't play in the browser" data-testid="player-unsupported">
            <p>Use one of the options below to watch it in another app.</p>
          </EuiCallOut>
        ) : (
          <InBrowserPlayer key={url} url={url} kind={kind} />
        )}
        <EuiSpacer size="m" />
        <EuiTitle size="xxs">
          <h3>Watch in another app</h3>
        </EuiTitle>
        <EuiSpacer size="s" />
        <EuiFlexGroup direction="column" gutterSize="s" responsive={false}>
          {links.map((l, i) => (
            <EuiFlexItem key={l.id}>
              <LinkButton href={l.href} primary={i === 0} testId={`player-link-${l.id}`}>
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

// PlayStreamLink is a native <a href> to the stream (middle-click and "copy link" still work) that opens
// the player sheet on a plain click instead of navigating to the raw stream.
export function PlayStreamLink({ channel, iconSize = 's', style, testId, ...rest }) {
  const [open, setOpen] = useState(false);
  if (!channel?.stream_url) return null;
  const onClick = (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    setOpen(true);
  };
  return (
    <Fragment>
      <a {...rest} href={channel.stream_url} onClick={onClick} aria-label="Open stream" title={channel.stream_url} data-testid={testId} style={style}>
        <EuiIcon type="play" size={iconSize} />
      </a>
      {open && <PlayerSheet channel={channel} onClose={() => setOpen(false)} />}
    </Fragment>
  );
}
