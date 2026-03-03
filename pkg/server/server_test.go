package server

import (
	"net/url"
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
