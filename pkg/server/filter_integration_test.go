package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
	"github.com/jamesnetherton/m3u"
)

// setupM3UTest parses testdata/mock.m3u, applies the given ProxyConfig options, runs
// playlistInitialization, and returns the path to the proxified M3U file.
func setupM3UTest(t *testing.T, opts ...func(*config.ProxyConfig)) (string, *Config) {
	t.Helper()
	playlist, err := m3u.Parse("testdata/mock.m3u")
	if err != nil {
		t.Fatalf("m3u.Parse testdata/mock.m3u: %v", err)
	}

	m3uPath := filepath.Join(t.TempDir(), "out.m3u")
	proxyConf := &config.ProxyConfig{
		HostConfig:           &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort:       8080,
		User:                 config.CredentialString("u"),
		Password:             config.CredentialString("p"),
	}
	for _, opt := range opts {
		opt(proxyConf)
	}

	fullTracks := make([]m3u.Track, len(playlist.Tracks))
	copy(fullTracks, playlist.Tracks)

	c := &Config{
		ProxyConfig:          proxyConf,
		playlist:             &playlist,
		fullPlaylistTracks:   fullTracks,
		proxyfiedM3UPath:     m3uPath,
		endpointAntiColision: "x",
	}

	if err := c.playlistInitialization(); err != nil {
		t.Fatalf("playlistInitialization: %v", err)
	}

	return m3uPath, c
}

// parseM3UChannelNames reads the proxified M3U file and returns the channel names in order.
func parseM3UChannelNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#EXTINF:") {
			if idx := strings.LastIndex(line, ","); idx >= 0 {
				names = append(names, strings.TrimSpace(line[idx+1:]))
			}
		}
	}
	return names, nil
}

// containsName returns true if needle is in names.
func containsName(names []string, needle string) bool {
	for _, n := range names {
		if n == needle {
			return true
		}
	}
	return false
}

// TestM3U_GroupExclusion verifies excluding the News group removes all 4 News channels.
func TestM3U_GroupExclusion(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		c.GroupExclusions = []string{`^News$`}
	})

	names, err := parseM3UChannelNames(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 11 {
		t.Errorf("want 11 channels (15-4 News), got %d: %v", len(names), names)
	}
	for _, newsChannel := range []string{"CNN International", "BBC News", "Al Jazeera", "Sky News"} {
		if containsName(names, newsChannel) {
			t.Errorf("News channel %q should be excluded but is present", newsChannel)
		}
	}
}

// TestM3U_GroupInclusion verifies including only Sports keeps exactly the 5 Sports channels.
func TestM3U_GroupInclusion(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		c.GroupInclusions = []string{`^Sports$`}
	})

	names, err := parseM3UChannelNames(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 5 {
		t.Errorf("want 5 Sports channels, got %d: %v", len(names), names)
	}
	for _, sportsChannel := range []string{"Sport Channel 1", "Sport Channel 2", "ESPN Live", "Sky Sports", "Eurosport"} {
		if !containsName(names, sportsChannel) {
			t.Errorf("Sports channel %q should be present but is absent", sportsChannel)
		}
	}
	// Non-Sports should be absent
	for _, other := range []string{"CNN International", "Comedy Central", "Cartoon Network"} {
		if containsName(names, other) {
			t.Errorf("Non-Sports channel %q should be absent but is present", other)
		}
	}
}

// TestM3U_ChannelExclusion verifies excluding CNN International by name removes it (14 remain).
func TestM3U_ChannelExclusion(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		c.ChannelExclusions = []string{`^CNN International$`}
	})

	names, err := parseM3UChannelNames(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 14 {
		t.Errorf("want 14 channels (15-1), got %d", len(names))
	}
	if containsName(names, "CNN International") {
		t.Error("CNN International should be excluded")
	}
}

// TestM3U_ChannelInclusion verifies including only ESPN Live yields exactly 1 channel.
func TestM3U_ChannelInclusion(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		c.ChannelInclusions = []string{`^ESPN Live$`}
	})

	names, err := parseM3UChannelNames(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 1 {
		t.Errorf("want 1 channel, got %d: %v", len(names), names)
	}
	if !containsName(names, "ESPN Live") {
		t.Error("ESPN Live should be present")
	}
}

// TestM3U_MultipleGroupExclusions verifies excluding News and Kids leaves 8 channels (Sports+Entertainment).
func TestM3U_MultipleGroupExclusions(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		c.GroupExclusions = []string{`^News$`, `^Kids$`}
	})

	names, err := parseM3UChannelNames(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 8 {
		t.Errorf("want 8 channels (5 Sports + 3 Entertainment), got %d: %v", len(names), names)
	}
	for _, excluded := range []string{"CNN International", "BBC News", "Al Jazeera", "Sky News", "Cartoon Network", "Disney Channel", "Nickelodeon"} {
		if containsName(names, excluded) {
			t.Errorf("channel %q should be excluded", excluded)
		}
	}
}

// TestM3U_GroupAndChannelExclusion verifies excluding the Sports group AND CNN International leaves 9 channels.
func TestM3U_GroupAndChannelExclusion(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		c.GroupExclusions = []string{`^Sports$`}
		c.ChannelExclusions = []string{`^CNN International$`}
	})

	names, err := parseM3UChannelNames(path)
	if err != nil {
		t.Fatal(err)
	}

	// 15 - 5 Sports - 1 CNN International = 9
	if len(names) != 9 {
		t.Errorf("want 9 channels, got %d: %v", len(names), names)
	}
	for _, excluded := range []string{"Sport Channel 1", "Sport Channel 2", "ESPN Live", "Sky Sports", "Eurosport", "CNN International"} {
		if containsName(names, excluded) {
			t.Errorf("channel %q should be excluded", excluded)
		}
	}
}

// TestM3U_GroupReplacement verifies that group-title "News" is renamed to "Breaking News" in the output.
func TestM3U_GroupReplacement(t *testing.T) {
	path, _ := setupM3UTest(t, func(c *config.ProxyConfig) {
		// no filter; replacements come from settings
	})

	// Re-run with settings containing a group replacement.
	playlist, err := m3u.Parse("testdata/mock.m3u")
	if err != nil {
		t.Fatalf("m3u.Parse: %v", err)
	}
	m3uPath := filepath.Join(t.TempDir(), "out.m3u")
	settings := &config.SettingsJSON{
		Replacements: &config.ReplacementsInSettings{
			Groups: []config.ReplacementRule{{Replace: `^News$`, With: "Breaking News"}},
		},
	}
	fullTracks := make([]m3u.Track, len(playlist.Tracks))
	copy(fullTracks, playlist.Tracks)
	c := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			AdvertisedPort: 8080,
			User:           config.CredentialString("u"),
			Password:       config.CredentialString("p"),
		},
		settings:             settings,
		playlist:             &playlist,
		fullPlaylistTracks:   fullTracks,
		proxyfiedM3UPath:     m3uPath,
		endpointAntiColision: "x",
	}
	if err := c.playlistInitialization(); err != nil {
		t.Fatalf("playlistInitialization: %v", err)
	}

	data, err := os.ReadFile(m3uPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, `group-title="Breaking News"`) {
		t.Error(`expected group-title="Breaking News" in output but not found`)
	}
	if strings.Contains(content, `group-title="News"`) {
		t.Error(`original group-title="News" should have been replaced but still present`)
	}
	// All 15 channels should still be present
	count, _ := countEXTINFInFile(m3uPath)
	if count != 15 {
		t.Errorf("want 15 channels, got %d", count)
	}
	// Suppress unused warning for path
	_ = path
}

// TestM3U_ChannelReplacement verifies that "CNN International" is renamed to "CNN" in the output.
func TestM3U_ChannelReplacement(t *testing.T) {
	playlist, err := m3u.Parse("testdata/mock.m3u")
	if err != nil {
		t.Fatalf("m3u.Parse: %v", err)
	}
	m3uPath := filepath.Join(t.TempDir(), "out.m3u")
	settings := &config.SettingsJSON{
		Replacements: &config.ReplacementsInSettings{
			Names: []config.ReplacementRule{{Replace: `^CNN International$`, With: "CNN"}},
		},
	}
	fullTracks := make([]m3u.Track, len(playlist.Tracks))
	copy(fullTracks, playlist.Tracks)
	c := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
			AdvertisedPort: 8080,
			User:           config.CredentialString("u"),
			Password:       config.CredentialString("p"),
		},
		settings:             settings,
		playlist:             &playlist,
		fullPlaylistTracks:   fullTracks,
		proxyfiedM3UPath:     m3uPath,
		endpointAntiColision: "x",
	}
	if err := c.playlistInitialization(); err != nil {
		t.Fatalf("playlistInitialization: %v", err)
	}

	names, err := parseM3UChannelNames(m3uPath)
	if err != nil {
		t.Fatal(err)
	}

	if !containsName(names, "CNN") {
		t.Error(`expected channel "CNN" after replacement but not found`)
	}
	if containsName(names, "CNN International") {
		t.Error(`"CNN International" should have been renamed to "CNN"`)
	}
	if len(names) != 15 {
		t.Errorf("want 15 channels, got %d", len(names))
	}
}
