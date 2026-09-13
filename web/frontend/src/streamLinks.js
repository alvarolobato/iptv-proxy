// Pure helpers for playing proxy stream URLs: in-browser playability, platform detection,
// external player deep links (VLC) and single-channel .m3u playlists.

export const VLC_PLAY_STORE_URL = 'https://play.google.com/store/apps/details?id=org.videolan.vlc';
export const VLC_APP_STORE_URL = 'https://apps.apple.com/app/vlc-media-player/id650377962';
const VLC_ANDROID_PACKAGE = 'org.videolan.vlc';

// streamKind classifies a stream URL by its extension:
//   'mpegts'      live MPEG-TS (.ts or no extension, e.g. Xtream /user/pass/id) -> mpegts.js
//   'hls'         .m3u8 -> native HLS only (Safari)
//   'progressive' .mp4/.m4v/.webm -> native <video src>
//   'external'    anything else (e.g. .mkv, .avi) -> external player only
export function streamKind(url) {
  let pathname;
  try {
    pathname = new URL(url).pathname.toLowerCase();
  } catch {
    return 'external';
  }
  const last = pathname.split('/').pop() || '';
  const dot = last.lastIndexOf('.');
  const ext = dot > 0 ? last.slice(dot + 1) : '';
  if (ext === '' || ext === 'ts') return 'mpegts';
  if (ext === 'm3u8') return 'hls';
  if (['mp4', 'm4v', 'webm'].includes(ext)) return 'progressive';
  return 'external';
}

// channelCategory maps the API channel type (first URL path segment, e.g. play/live/movie/series)
// to a TV view category.
export function channelCategory(type) {
  const t = String(type || '').toLowerCase();
  if (t === 'movie' || t === 'movies') return 'movies';
  if (t === 'series') return 'series';
  return 'live';
}

// detectPlatform picks which external player links to offer. iPadOS reports a Mac user agent,
// so a touch-capable "Macintosh" is treated as iOS.
export function detectPlatform(userAgent, maxTouchPoints = 0) {
  const ua = String(userAgent || '');
  if (/android/i.test(ua)) return 'android';
  if (/iphone|ipad|ipod/i.test(ua) || (/macintosh/i.test(ua) && maxTouchPoints > 1)) return 'ios';
  return 'desktop';
}

// androidIntentUrl opens url in a video app through an Android intent. With pkg it targets that app and
// falls back to its Play Store page when it is not installed; without pkg Android shows the app chooser.
export function androidIntentUrl(url, pkg) {
  const u = new URL(url);
  const scheme = u.protocol.replace(/:$/, '');
  let intent = `intent://${u.host}${u.pathname}${u.search}#Intent;scheme=${scheme};type=video/*;`;
  if (pkg) intent += `package=${pkg};S.browser_fallback_url=${encodeURIComponent(VLC_PLAY_STORE_URL)};`;
  return `${intent}end`;
}

// externalPlayerLinks returns deep links for the platform; the first entry is the primary action.
// Desktop VLC registers no URL scheme, so desktop gets none and uses the .m3u download instead.
export function externalPlayerLinks(url, platform) {
  if (!url) return [];
  try {
    if (platform === 'ios') {
      return [
        { id: 'vlc', label: 'Open in VLC', href: `vlc-x-callback://x-callback-url/stream?url=${encodeURIComponent(url)}` },
        { id: 'vlc-alt', label: 'Open in VLC (alternative link)', href: `vlc://${url}` },
      ];
    }
    if (platform === 'android') {
      return [
        { id: 'vlc', label: 'Open in VLC', href: androidIntentUrl(url, VLC_ANDROID_PACKAGE) },
        { id: 'other-app', label: 'Open with another app', href: androidIntentUrl(url) },
      ];
    }
  } catch {
    return [];
  }
  return [];
}

// buildM3u returns a one-channel playlist that desktop players (VLC, IINA, mpv) open directly.
export function buildM3u(name, url) {
  const title = String(name || 'Stream').replace(/[\r\n]+/g, ' ').trim();
  return `#EXTM3U\n#EXTINF:-1,${title}\n${url}\n`;
}

export function m3uFileName(name) {
  const base = String(name || '')
    .normalize('NFKD')
    .replace(/[^\w.-]+/g, '_')
    .replace(/^[_.]+|[_.]+$/g, '')
    .slice(0, 80);
  return `${base || 'stream'}.m3u`;
}
