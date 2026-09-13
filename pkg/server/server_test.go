/*
 * iptv-proxy2 is a fork of Iptv-Proxy by Pierre-Emmanuel Jacquier.
 * Original project: https://github.com/pierre-emmanuel-jacquier/iptv-proxy
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * New additions and modifications in this fork:
 * Copyright (C) 2026  Alvaro Lobato (github.com/alvarolobato/iptv-proxy)
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jamesnetherton/m3u"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
	"github.com/alvarolobato/iptv-proxy/pkg/stats"
)

func TestNewServer_EmptyRemoteURL(t *testing.T) {
	emptyURL, _ := url.Parse("")
	conf := &config.ProxyConfig{
		HostConfig:       &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		RemoteURL:       emptyURL,
		User:             config.CredentialString("u"),
		Password:         config.CredentialString("p"),
		AdvertisedPort:   8080,
	}
	srv, err := NewServer(conf, nil, nil)
	if err != nil {
		t.Fatalf("NewServer() err = %v", err)
	}
	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}
	if srv.ProxyConfig != conf {
		t.Error("ProxyConfig not set")
	}
}

func TestReplaceURL_M3U(t *testing.T) {
	conf := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:      &config.HostConfiguration{Hostname: "proxy.example.com", Port: 8080},
			AdvertisedPort: 8080,
			User:           config.CredentialString("u"),
			Password:       config.CredentialString("p"),
			HTTPS:          false,
		},
		endpointAntiColision: "abc",
	}
	got, err := conf.replaceURL("http://upstream.example.com/live/123.ts", 0, false)
	if err != nil {
		t.Fatalf("replaceURL() err = %v", err)
	}
	if got == "" {
		t.Fatal("replaceURL() returned empty")
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse(got) err = %v", err)
	}
	if parsed.Host != "proxy.example.com:8080" {
		t.Errorf("Host = %q, want proxy.example.com:8080", parsed.Host)
	}
}

func TestReplaceURL_EmptyHostnameFallback(t *testing.T) {
	// When Hostname is empty, replaceURL must use "localhost" so stream URLs are valid (no http://:9090/...).
	conf := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:      &config.HostConfiguration{Hostname: "", Port: 9090},
			AdvertisedPort:  9090,
			User:            config.CredentialString(""),
			Password:        config.CredentialString(""),
			HTTPS:           false,
		},
		endpointAntiColision: "fdb3eeb7",
	}
	got, err := conf.replaceURL("http://provider.com/live/17282/1198989.mkv", 0, false)
	if err != nil {
		t.Fatalf("replaceURL() err = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse(got) err = %v", err)
	}
	if parsed.Host != "localhost:9090" {
		t.Errorf("Host = %q, want localhost:9090 (empty hostname must fallback to localhost)", parsed.Host)
	}
}

// TestChannelsProcessed_StreamURL verifies that included channels get a non-empty stream_url.
func TestChannelsProcessed_StreamURL(t *testing.T) {
	uri := "http://example.com/live/stream"
	track := m3u.Track{Name: "Test Channel", URI: uri, Tags: []m3u.Tag{{Name: "group-title", Value: "Group1"}}}
	pl := &m3u.Playlist{Tracks: []m3u.Track{track}}
	fullTracks := []m3u.Track{track}

	c := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:      &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			AdvertisedPort:  8080,
			User:            config.CredentialString("u"),
			Password:        config.CredentialString("p"),
			XtreamBaseURL:   "", // M3U mode
		},
		playlist:             pl,
		fullPlaylistTracks:   fullTracks,
		trackIndexInPlaylist: map[string]int{uri: 0}, // index stored when building playlist
		endpointAntiColision: "x",
	}
	out := c.channelsProcessed()
	if len(out) != 1 {
		t.Fatalf("channelsProcessed() len = %d, want 1", len(out))
	}
	if out[0].StreamURL == "" {
		t.Error("channelsProcessed() stream_url empty for included channel")
	}
	if out[0].Excluded {
		t.Error("channelsProcessed() channel should not be excluded")
	}
}

// countEXTINFInFile returns the number of #EXTINF lines in an M3U file (one per track).
func countEXTINFInFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#EXTINF:") {
			n++
		}
	}
	return n, nil
}

// TestApplyLiveSettings_RestoresFullPlaylistBeforeFiltering verifies that when the user changes
// exclusions in the UI and saves, the served M3U is rebuilt from the full playlist, not from
// the previously filtered subset. Otherwise removing an exclusion would never bring back tracks.
func TestApplyLiveSettings_RestoresFullPlaylistBeforeFiltering(t *testing.T) {
	dir := t.TempDir()
	m3uPath := filepath.Join(dir, "out.m3u")

	track1 := m3u.Track{Name: "Ch1", URI: "http://example.com/1", Length: 0, Tags: []m3u.Tag{{Name: "group-title", Value: "Group1"}}}
	track2 := m3u.Track{Name: "Ch2", URI: "http://example.com/2", Length: 0, Tags: []m3u.Tag{{Name: "group-title", Value: "Group2"}}}
	track3 := m3u.Track{Name: "Ch3", URI: "http://example.com/3", Length: 0, Tags: []m3u.Tag{{Name: "group-title", Value: "Group3"}}}
	fullTracks := []m3u.Track{track1, track2, track3}
	playlist := &m3u.Playlist{Tracks: make([]m3u.Track, len(fullTracks))}
	copy(playlist.Tracks, fullTracks)

	c := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:    &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			AdvertisedPort: 8080,
			User:          config.CredentialString("u"),
			Password:      config.CredentialString("p"),
		},
		playlist:           playlist,
		fullPlaylistTracks: fullTracks,
		proxyfiedM3UPath:   m3uPath,
		endpointAntiColision: "x",
	}

	// First run: apply exclusion for Group2 -> served M3U should have 2 tracks.
	c.ProxyConfig.GroupExclusions = []string{`^Group2$`}
	if err := c.playlistInitialization(); err != nil {
		t.Fatalf("playlistInitialization(): %v", err)
	}
	if len(c.playlist.Tracks) != 2 {
		t.Errorf("after first init with exclusion: len(playlist.Tracks) = %d, want 2", len(c.playlist.Tracks))
	}
	count, err := countEXTINFInFile(m3uPath)
	if err != nil {
		t.Fatalf("countEXTINFInFile(): %v", err)
	}
	if count != 2 {
		t.Errorf("after first init: track count in file = %d, want 2", count)
	}

	// Apply new settings with no exclusions (user removed the exclusion in UI). Without restoring
	// fullPlaylistTracks, we would filter from the current 2-track list and never get the third back.
	settingsNoExcl := &config.SettingsJSON{GroupExclusions: []string{}}
	c.applyLiveSettings(settingsNoExcl)

	if len(c.playlist.Tracks) != 3 {
		t.Errorf("after applyLiveSettings(no exclusions): len(playlist.Tracks) = %d, want 3 (full list)", len(c.playlist.Tracks))
	}
	count, err = countEXTINFInFile(m3uPath)
	if err != nil {
		t.Fatalf("countEXTINFInFile(): %v", err)
	}
	if count != 3 {
		t.Errorf("after applyLiveSettings(no exclusions): track count in file = %d, want 3 (served M3U must include all)", count)
	}
}

// TestMarshallInto_GroupExclusionReducesTracks verifies that group exclusions are applied when
// writing the M3U so the file served to clients contains only non-excluded tracks.
func TestMarshallInto_GroupExclusionReducesTracks(t *testing.T) {
	dir := t.TempDir()
	m3uPath := filepath.Join(dir, "out.m3u")
	f, err := os.Create(m3uPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	track1 := m3u.Track{Name: "A", URI: "http://ex.com/1", Length: 0, Tags: []m3u.Tag{{Name: "group-title", Value: "Sports"}}}
	track2 := m3u.Track{Name: "B", URI: "http://ex.com/2", Length: 0, Tags: []m3u.Tag{{Name: "group-title", Value: "News"}}}
	track3 := m3u.Track{Name: "C", URI: "http://ex.com/3", Length: 0, Tags: []m3u.Tag{{Name: "group-title", Value: "Sports"}}}
	tracks := []m3u.Track{track1, track2, track3}
	playlist := &m3u.Playlist{Tracks: tracks}

	c := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:    &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			AdvertisedPort: 8080,
			User:          config.CredentialString("u"),
			Password:      config.CredentialString("p"),
			GroupExclusions: []string{`^News$`},
		},
		playlist:           playlist,
		endpointAntiColision: "x",
	}
	if err := c.marshallInto(f, false); err != nil {
		t.Fatalf("marshallInto(): %v", err)
	}
	// Sync so file is readable
	_ = f.Sync()
	f.Close()

	count, err := countEXTINFInFile(m3uPath)
	if err != nil {
		t.Fatalf("countEXTINFInFile(): %v", err)
	}
	if count != 2 {
		t.Errorf("track count in file = %d, want 2 (News excluded)", count)
	}
}

// TestXtreamRouteRegistration verifies that xtream routes can be registered without panicking.
// This catches Gin wildcard conflicts (e.g. /play/:token/:type vs /play/:user/:password/:id).
func TestXtreamRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			AdvertisedPort: 8080,
			User:           config.CredentialString("admin"),
			Password:       config.CredentialString("pass"),
			XtreamBaseURL:  "http://upstream:8080",
			XtreamUser:     config.CredentialString("xu"),
			XtreamPassword: config.CredentialString("xp"),
		},
		endpointAntiColision: "test",
		statsCollector:       &stats.NoopCollector{},
	}

	// This should not panic.
	r := gin.New()
	group := r.Group("")
	c.xtreamRoutes(group)
}

// newXtreamM3UConfig builds a Config whose M3U source is the Xtream account's own get.php
// (the mode used when only Xtream credentials are configured).
func newXtreamM3UConfig(t *testing.T, upstreamBase string, trackURI string) *Config {
	t.Helper()
	remote, err := url.Parse(upstreamBase + "/get.php?username=xu&password=xp&type=m3u_plus&output=mpegts")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	track := m3u.Track{Name: "Live Channel", URI: trackURI, Tags: []m3u.Tag{{Name: "group-title", Value: "Group1"}}}
	pc := &config.ProxyConfig{
		HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort: 8080,
		User:           config.CredentialString("u"),
		Password:       config.CredentialString("p"),
		RemoteURL:      remote,
		M3UFileName:    "iptv.m3u",
		XtreamBaseURL:  upstreamBase,
		XtreamUser:     config.CredentialString("xu"),
		XtreamPassword: config.CredentialString("xp"),
	}
	pc.MigrateDefaultUser()
	return &Config{
		ProxyConfig:          pc,
		playlist:             &m3u.Playlist{Tracks: []m3u.Track{track}},
		fullPlaylistTracks:   []m3u.Track{track},
		trackIndexInPlaylist: map[string]int{trackURI: 0},
		endpointAntiColision: "x",
		statsCollector:       &stats.NoopCollector{},
	}
}

// TestChannelsProcessed_StreamURL_XtreamM3U verifies that in Xtream M3U mode the UI stream_url uses the
// Xtream form (per-track anti-collision routes are not registered in that mode, so they would 404).
func TestChannelsProcessed_StreamURL_XtreamM3U(t *testing.T) {
	c := newXtreamM3UConfig(t, "http://upstream:80", "http://upstream:80/play/xu/xp/123.ts")
	out := c.channelsProcessed()
	if len(out) != 1 {
		t.Fatalf("channelsProcessed() len = %d, want 1", len(out))
	}
	if want := "http://localhost:8080/play/u/p/123.ts"; out[0].StreamURL != want {
		t.Errorf("stream_url = %q, want %q", out[0].StreamURL, want)
	}
}

// TestXtreamM3U_StreamURLReachesUpstream requests the UI stream_url through the real router and verifies
// it is routed (not a Gin 404) and proxied to the provider's /play path with the provider credentials.
func TestXtreamM3U_StreamURLReachesUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		if r.URL.Path != "/play/xu/xp/123.ts" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "video/mp2t")
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.ts")
	r := gin.New()
	c.routes(r.Group(""))

	out := c.channelsProcessed()
	if len(out) != 1 || out[0].StreamURL == "" {
		t.Fatalf("channelsProcessed() = %+v, want one row with stream_url", out)
	}
	streamURL, err := url.Parse(out[0].StreamURL)
	if err != nil {
		t.Fatalf("url.Parse(stream_url): %v", err)
	}

	// Real server: gin's ctx.Stream needs a CloseNotifier, which httptest.ResponseRecorder lacks.
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + streamURL.Path)
	if err != nil {
		t.Fatalf("GET %s: %v", streamURL.Path, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s status = %d, want 200 (upstream path %q)", streamURL.Path, resp.StatusCode, upstreamPath)
	}
	if upstreamPath != "/play/xu/xp/123.ts" {
		t.Errorf("upstream path = %q, want /play/xu/xp/123.ts", upstreamPath)
	}
}

// TestServesXtreamM3U verifies the Xtream M3U mode predicate only matches the Xtream account's own get.php.
func TestServesXtreamM3U(t *testing.T) {
	mk := func(base, remote, xu, xp string) *Config {
		var ru *url.URL
		if remote != "" {
			ru, _ = url.Parse(remote)
		}
		return &Config{ProxyConfig: &config.ProxyConfig{
			RemoteURL:      ru,
			XtreamBaseURL:  base,
			XtreamUser:     config.CredentialString(xu),
			XtreamPassword: config.CredentialString(xp),
		}}
	}
	tests := []struct {
		name string
		c    *Config
		want bool
	}{
		{"own get.php", mk("http://prov:8080", "http://prov:8080/get.php?username=xu&password=xp&type=m3u_plus", "xu", "xp"), true},
		{"default port on one side", mk("http://prov:80", "http://prov/get.php?username=xu&password=xp", "xu", "xp"), true},
		{"host only contained in base", mk("http://myprov.example:80", "http://prov.example/get.php?username=xu&password=xp", "xu", "xp"), false},
		{"different port", mk("http://prov:8080", "http://prov:9090/get.php?username=xu&password=xp", "xu", "xp"), false},
		{"not get.php", mk("http://prov:8080", "http://prov:8080/playlist.m3u?username=xu&password=xp", "xu", "xp"), false},
		{"empty credentials", mk("http://prov:8080", "http://prov:8080/get.php", "", ""), false},
		{"credential mismatch", mk("http://prov:8080", "http://prov:8080/get.php?username=xu&password=other", "xu", "xp"), false},
		{"no remote url", mk("http://prov:8080", "", "xu", "xp"), false},
		{"no xtream base", mk("", "http://prov:8080/get.php?username=xu&password=xp", "xu", "xp"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.servesXtreamM3U(); got != tt.want {
				t.Errorf("servesXtreamM3U() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestXtreamPlay_M3U8UsesHLSHandler verifies /play/:user/:password/:id.m3u8 (catch-all route, no :id param)
// is handled by hlsXtreamStream, which rewrites provider credentials inside the playlist.
func TestXtreamPlay_M3U8UsesHLSHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamURL string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/play/xu/xp/123.m3u8":
			http.Redirect(w, r, upstreamURL+"/hls/tok/123.m3u8", http.StatusFound)
		case "/hls/tok/123.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.Write([]byte("#EXTM3U\n/hlsr/tok/xu/xp/123/1/seg.ts\n")) // nolint: errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	upstreamURL = upstream.URL

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.m3u8")
	r := gin.New()
	c.routes(r.Group(""))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/play/u/p/123.m3u8")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "/hlsr/tok/u/p/123/1/seg.ts") {
		t.Errorf("playlist not rewritten by the HLS handler: %q", body)
	}
}

// TestXtreamStreams_EscapeProviderCredentials verifies provider credentials are path-escaped in upstream
// stream URLs, so passwords containing '#', '?' or '/' are forwarded intact.
func TestXtreamStreams_EscapeProviderCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.ts")
	c.XtreamPassword = config.CredentialString("p#ss/w?rd")
	r := gin.New()
	c.xtreamRoutes(r.Group(""))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	const esc = "p%23ss%2Fw%3Frd"
	tests := []struct{ path, want string }{
		{"/play/u/p/123.ts", "/play/xu/" + esc + "/123.ts"},
		{"/live/u/p/123.ts", "/live/xu/" + esc + "/123.ts"},
		{"/movie/u/p/123.mp4", "/movie/xu/" + esc + "/123.mp4"},
		{"/series/u/p/123.mkv", "/series/xu/" + esc + "/123.mkv"},
		{"/u/p/123", "/xu/" + esc + "/123"},
	}
	for _, tt := range tests {
		gotPath = ""
		resp, err := http.Get(proxy.URL + tt.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tt.path, err)
		}
		resp.Body.Close()
		if gotPath != tt.want {
			t.Errorf("GET %s: upstream path = %q, want %q", tt.path, gotPath, tt.want)
		}
	}
}

// TestReplaceURL_XtreamCredentialSegments verifies Xtream credentials are swapped as whole path segments,
// so numeric credentials don't corrupt stream ids or fixed path words that contain them.
func TestReplaceURL_XtreamCredentialSegments(t *testing.T) {
	tests := []struct{ user, pass, uri, wantPath string }{
		{"1234", "5678", "http://prov:80/play/1234/5678/12345678.ts", "/play/u/p/12345678.ts"},
		{"1234", "5678", "http://prov:80/1234/5678/91234.ts", "/u/p/91234.ts"},
		{"a", "5678", "http://prov:80/play/a/5678/1.ts", "/play/u/p/1.ts"},
		{"xu", "p#ss", "http://prov:80/live/xu/p%23ss/7.ts", "/live/u/p/7.ts"},
	}
	for _, tt := range tests {
		c := &Config{ProxyConfig: &config.ProxyConfig{
			HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			User:           config.CredentialString("u"),
			Password:       config.CredentialString("p"),
			XtreamUser:     config.CredentialString(tt.user),
			XtreamPassword: config.CredentialString(tt.pass),
		}}
		got, err := c.replaceURL(tt.uri, 0, true)
		if err != nil {
			t.Fatalf("replaceURL(%q): %v", tt.uri, err)
		}
		u, err := url.Parse(got)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", got, err)
		}
		if u.EscapedPath() != tt.wantPath {
			t.Errorf("replaceURL(%q) path = %q, want %q", tt.uri, u.EscapedPath(), tt.wantPath)
		}
	}
}

// TestXtreamStreams_EscapeCredentialsTimeshiftAndHLSR covers the remaining upstream URL builders.
func TestXtreamStreams_EscapeCredentialsTimeshiftAndHLSR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	redirect, err := url.Parse(upstream.URL + "/hls/tok/123.m3u8")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	hlsChannelsRedirectURLLock.Lock()
	hlsChannelsRedirectURL["123.m3u8"] = *redirect
	hlsChannelsRedirectURLLock.Unlock()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.ts")
	c.XtreamPassword = config.CredentialString("p#ss/w?rd")
	r := gin.New()
	c.xtreamRoutes(r.Group(""))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	const esc = "p%23ss%2Fw%3Frd"
	tests := []struct{ path, want string }{
		{"/timeshift/u/p/60/2026-01-01:10-00/123.ts", "/timeshift/xu/" + esc + "/60/2026-01-01:10-00/123.ts"},
		{"/hlsr/tok/u/p/123/hash/1.ts", "/hlsr/tok/xu/" + esc + "/123/hash/1.ts"},
	}
	for _, tt := range tests {
		gotPath = ""
		resp, err := http.Get(proxy.URL + tt.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tt.path, err)
		}
		resp.Body.Close()
		if gotPath != tt.want {
			t.Errorf("GET %s: upstream path = %q, want %q", tt.path, gotPath, tt.want)
		}
	}
}

// TestXtreamPlay_ProxyPasswordWithSlash verifies UI /play links work when the proxy password contains '/',
// which replaceURL writes as %2F and Gin decodes before routing.
func TestXtreamPlay_ProxyPasswordWithSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.EscapedPath()
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.ts")
	c.Password = config.CredentialString("a/b")
	c.ProxyConfig.Users = nil
	c.ProxyConfig.MigrateDefaultUser()
	r := gin.New()
	c.routes(r.Group(""))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	out := c.channelsProcessed()
	if len(out) != 1 || !strings.Contains(out[0].StreamURL, "/play/u/a%2Fb/123.ts") {
		t.Fatalf("channelsProcessed() = %+v, want stream_url with escaped password segment", out)
	}
	streamURL, err := url.Parse(out[0].StreamURL)
	if err != nil {
		t.Fatalf("url.Parse(stream_url): %v", err)
	}

	resp, err := http.Get(proxy.URL + streamURL.EscapedPath())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if upstreamPath != "/play/xu/xp/123.ts" {
		t.Errorf("upstream path = %q, want /play/xu/xp/123.ts", upstreamPath)
	}
}

// TestXtreamPlay_M3U8RewritesEscapedCredentials verifies the HLS playlist rewrite also replaces provider
// credentials that the provider wrote path-escaped, so they never reach the client.
func TestXtreamPlay_M3U8RewritesEscapedCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamURL string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/play/xu/p%23ss/123.m3u8":
			http.Redirect(w, r, upstreamURL+"/hls/tok/123.m3u8", http.StatusFound)
		case "/hls/tok/123.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.Write([]byte("#EXTM3U\n/hlsr/tok/xu/p%23ss/123/1/seg.ts\n")) // nolint: errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	upstreamURL = upstream.URL

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.m3u8")
	c.XtreamPassword = config.CredentialString("p#ss")
	r := gin.New()
	c.xtreamRoutes(r.Group(""))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/play/u/p/123.m3u8")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "/hlsr/tok/u/p/123/1/seg.ts") || strings.Contains(string(body), "p%23ss") {
		t.Errorf("playlist credentials not rewritten: %q", body)
	}
}

// TestRoutes_SameHostNonGetPHPKeepsM3URoutes verifies a plain M3U on the Xtream host (not get.php) still
// gets per-track anti-collision routes next to the Xtream routes, and the UI links to them.
func TestRoutes_SameHostNonGetPHPKeepsM3URoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/streams/stream1.ts")
	remote, err := url.Parse(upstream.URL + "/playlist.m3u?username=xu&password=xp")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	c.RemoteURL = remote
	if c.servesXtreamM3U() {
		t.Fatal("servesXtreamM3U() = true for a non-get.php playlist")
	}
	r := gin.New()
	c.routes(r.Group(""))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	out := c.channelsProcessed()
	if len(out) != 1 || !strings.Contains(out[0].StreamURL, "/x/u/p/0/stream1.ts") {
		t.Fatalf("channelsProcessed() = %+v, want anti-collision stream_url", out)
	}
	streamURL, err := url.Parse(out[0].StreamURL)
	if err != nil {
		t.Fatalf("url.Parse(stream_url): %v", err)
	}
	resp, err := http.Get(proxy.URL + streamURL.EscapedPath())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s status = %d, want 200", streamURL.EscapedPath(), resp.StatusCode)
	}
}

// TestXtreamPlay_CustomEndpointContainingPlay verifies the /play dispatcher strips the exact route prefix,
// so a custom endpoint that itself contains a "play" segment doesn't shift the credential segments.
func TestXtreamPlay_CustomEndpointContainingPlay(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.EscapedPath()
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.ts")
	c.CustomEndpoint = "/tv/play"
	r := gin.New()
	configureProxyEngine(r)
	c.routes(r.Group("/"))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	out := c.channelsProcessed()
	if len(out) != 1 || !strings.Contains(out[0].StreamURL, "/tv/play/play/u/p/123.ts") {
		t.Fatalf("channelsProcessed() = %+v, want stream_url under the custom endpoint", out)
	}
	streamURL, err := url.Parse(out[0].StreamURL)
	if err != nil {
		t.Fatalf("url.Parse(stream_url): %v", err)
	}
	resp, err := http.Get(proxy.URL + streamURL.EscapedPath())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s status = %d, want 200", streamURL.EscapedPath(), resp.StatusCode)
	}
	if upstreamPath != "/play/xu/xp/123.ts" {
		t.Errorf("upstream path = %q, want /play/xu/xp/123.ts", upstreamPath)
	}
}

// TestReplaceURL_ProviderEscapingDiffers verifies provider credentials are recognised however the provider
// escaped them, so they are never left in served URLs.
func TestReplaceURL_ProviderEscapingDiffers(t *testing.T) {
	tests := []struct{ user, pass, uri, wantPath string }{
		{"xu", "p@ss", "http://prov:80/live/xu/p%40ss/1.ts", "/live/u/p/1.ts"},
		{"xu", "xp", "http://prov:80/live/x%75/xp/1.ts", "/live/u/p/1.ts"},
		{"xu", "p@ss", "http://prov:80/live/xu/p@ss/1.ts", "/live/u/p/1.ts"},
	}
	for _, tt := range tests {
		c := &Config{ProxyConfig: &config.ProxyConfig{
			HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			User:           config.CredentialString("u"),
			Password:       config.CredentialString("p"),
			XtreamUser:     config.CredentialString(tt.user),
			XtreamPassword: config.CredentialString(tt.pass),
		}}
		got, err := c.replaceURL(tt.uri, 0, true)
		if err != nil {
			t.Fatalf("replaceURL(%q): %v", tt.uri, err)
		}
		u, err := url.Parse(got)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", got, err)
		}
		if u.EscapedPath() != tt.wantPath {
			t.Errorf("replaceURL(%q) path = %q, want %q", tt.uri, u.EscapedPath(), tt.wantPath)
		}
	}
}

// TestProxyPasswordWithSlash_ParamRoutes verifies :user/:password routes accept a proxy password containing '/'
// (sent as %2F) once the engine routes on the escaped path.
func TestProxyPasswordWithSlash_ParamRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.EscapedPath()
		w.Write([]byte("ts")) // nolint: errcheck
	}))
	defer upstream.Close()

	c := newXtreamM3UConfig(t, upstream.URL, upstream.URL+"/play/xu/xp/123.ts")
	c.Password = config.CredentialString("a/b")
	c.ProxyConfig.Users = nil
	c.ProxyConfig.MigrateDefaultUser()
	r := gin.New()
	configureProxyEngine(r)
	c.routes(r.Group("/"))
	proxy := httptest.NewServer(r)
	defer proxy.Close()

	tests := []struct{ path, want string }{
		{"/live/u/a%2Fb/123.ts", "/live/xu/xp/123.ts"},
		{"/movie/u/a%2Fb/123.mp4", "/movie/xu/xp/123.mp4"},
		{"/u/a%2Fb/123", "/xu/xp/123"},
		{"/play/u/a%2Fb/123.ts", "/play/xu/xp/123.ts"},
	}
	for _, tt := range tests {
		upstreamPath = ""
		resp, err := http.Get(proxy.URL + tt.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tt.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || upstreamPath != tt.want {
			t.Errorf("GET %s: status %d, upstream path %q; want 200 and %q", tt.path, resp.StatusCode, upstreamPath, tt.want)
		}
	}
}
