package server

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesnetherton/m3u"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
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
