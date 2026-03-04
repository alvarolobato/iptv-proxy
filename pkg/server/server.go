/*
 * Iptv-Proxy is a project to proxyfie an m3u file and to proxyfie an Xtream iptv service (client API).
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
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
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jamesnetherton/m3u"
	"github.com/alvarolobato/iptv-proxy/pkg/config"
	"github.com/alvarolobato/iptv-proxy/pkg/stats"
	uuid "github.com/satori/go.uuid"
)

var defaultProxyfiedM3UPath = filepath.Join(os.TempDir(), uuid.NewV4().String()+".iptv-proxy.m3u")
var endpointAntiColision = strings.Split(uuid.NewV4().String(), "-")[0]

// pathAuthUser returns the path segment for proxy user (use "-" when empty to avoid route conflicts).
func (c *Config) pathAuthUser() string {
	if s := c.User.String(); s != "" {
		return s
	}
	return "-"
}

// pathAuthPassword returns the path segment for proxy password (use "-" when empty to avoid route conflicts).
func (c *Config) pathAuthPassword() string {
	if s := c.Password.String(); s != "" {
		return s
	}
	return "-"
}

// Config represent the server configuration
type Config struct {
	*config.ProxyConfig

	// settings is the loaded settings.json (nil if not used). Used for replacements and GET/PUT /api/settings.
	settings *config.SettingsJSON
	// defaultSettings is the config from flag/env before settings were applied; used to persist only overrides to settings.json.
	defaultSettings *config.SettingsJSON

	// M3U service part
	playlist *m3u.Playlist
	// fullPlaylistTracks is a copy of the initial fetch; used by UI API so /api/groups and /api/channels can return all items with excluded flag.
	fullPlaylistTracks []m3u.Track
	// trackIndexInPlaylist maps upstream track URI -> index in filtered playlist; used to build proxified URL when needed.
	trackIndexInPlaylist map[string]int
	// this variable is set only for m3u proxy endpoints
	track *m3u.Track
	// path to the proxyfied m3u file
	proxyfiedM3UPath string

	endpointAntiColision string

	xmltvCache *responseCache

	// statsCollector records session events to Elasticsearch (or no-ops when ES is not configured).
	statsCollector stats.Collector
}

// NewServer initialize a new server configuration. settings is optional (from settings.json); when set, replacements come from it.
// defaultSettings is the config from flag/env before ApplyTo (used to write only overrides to settings.json); may be nil.
func NewServer(proxyConfig *config.ProxyConfig, settings *config.SettingsJSON, defaultSettings *config.SettingsJSON) (*Config, error) {
	var p m3u.Playlist
	if proxyConfig.RemoteURL != nil && proxyConfig.RemoteURL.String() != "" {
		var err error
		p, err = m3u.Parse(proxyConfig.RemoteURL.String())
		if err != nil {
			return nil, err
		}
	}

	if trimmedCustomId := strings.Trim(proxyConfig.CustomId, "/"); trimmedCustomId != "" {
		endpointAntiColision = trimmedCustomId
	}

	fullTracks := make([]m3u.Track, len(p.Tracks))
	copy(fullTracks, p.Tracks)
	cfg := &Config{
		ProxyConfig:          proxyConfig,
		settings:             settings,
		defaultSettings:      defaultSettings,
		playlist:             &p,
		fullPlaylistTracks:   fullTracks,
		track:                nil,
		proxyfiedM3UPath:     defaultProxyfiedM3UPath,
		endpointAntiColision: endpointAntiColision,
		statsCollector:       &stats.NoopCollector{},
	}
	cfg.xmltvCache = newResponseCache(proxyConfig.XMLTVCacheTTL, proxyConfig.XMLTVCacheMaxEntries)

	// Initialize Elasticsearch stats collector when URL is configured.
	if proxyConfig.StatsEnabled && proxyConfig.ESUrl != "" {
		esCfg := stats.ESConfig{
			URL:         proxyConfig.ESUrl,
			APIKey:      proxyConfig.ESApiKey,
			Username:    proxyConfig.ESUsername,
			Password:    proxyConfig.ESPassword,
			IndexPrefix: proxyConfig.ESIndexPrefix,
		}
		esCollector, err := stats.NewESCollector(esCfg)
		if err != nil {
			log.Printf("[iptv-proxy] WARN: could not initialize stats collector: %v; stats will be disabled", err)
		} else {
			cfg.statsCollector = esCollector
			log.Printf("[iptv-proxy] Stats: Elasticsearch collector initialized (prefix: %s)", proxyConfig.ESIndexPrefix)
		}
	}

	return cfg, nil
}

// Serve runs the server (with minimal startup log). For full startup summary use ServeWithContext.
func (c *Config) Serve() error {
	return c.ServeWithContext(nil)
}

func (c *Config) playlistInitialization() error {
	if len(c.playlist.Tracks) == 0 {
		return nil
	}

	f, err := os.Create(c.proxyfiedM3UPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return c.marshallInto(f, false)
}

// marshallInto writes the playlist to a file, applying optional filter, replacement, and resolution grouping.
func (c *Config) marshallInto(into *os.File, xtream bool) error {
	if !xtream {
		c.trackIndexInPlaylist = make(map[string]int)
	}
	filteredTrack := make([]m3u.Track, 0, len(c.playlist.Tracks))

	// Compile inclusion/exclusion regex lists (from settings only)
	groupInclRE := compileRegexList(c.GroupInclusions, "group_inclusions")
	groupExclRE := compileRegexList(c.GroupExclusions, "group_exclusions")
	channelInclRE := compileRegexList(c.ChannelInclusions, "channel_inclusions")
	channelExclRE := compileRegexList(c.ChannelExclusions, "channel_exclusions")

	tracks := make([]m3u.Track, 0, len(c.playlist.Tracks))
	for _, track := range c.playlist.Tracks {
		group := getGroupTitle(track)
		name := track.Name
		if !matchInclusionExclusion(group, name, groupInclRE, groupExclRE, channelInclRE, channelExclRE) {
			continue
		}
		tracks = append(tracks, track)
	}

	reFHD := regexp.MustCompile(`\sFHD$`)
	reHD := regexp.MustCompile(`\sHD$`)
	reSD := regexp.MustCompile(`\sSD$`)

	var replacements Replacements
	if c.settings != nil && c.settings.Replacements != nil {
		replacements = ReplacementsFromSettings(c.settings.Replacements)
	} else if c.DataFolder != "" {
		replacements = loadReplacements(filepath.Join(c.DataFolder, "replacements.json"))
	}

	for i := range tracks {
		track := &tracks[i]
		// Ensure tvg-id tag exists
		hasTvgID := false
		for t := range track.Tags {
			if track.Tags[t].Name == "tvg-id" {
				hasTvgID = true
				break
			}
		}
		if !hasTvgID {
			track.Tags = append(track.Tags, m3u.Tag{Name: "tvg-id", Value: ""})
		}

		track.Name = applyReplacements(replacements.Global, track.Name)
		track.Name = applyReplacements(replacements.Names, track.Name)

		// Derive resolution from track.Name and then from tvg-name tag
		isFHD := reFHD.MatchString(track.Name)
		isHD := reHD.MatchString(track.Name)
		isSD := reSD.MatchString(track.Name)
		for j := range track.Tags {
			tag := &track.Tags[j]
			tag.Value = applyReplacements(replacements.Global, tag.Value)
			switch tag.Name {
			case "tvg-name":
				if c.DivideByRes {
					isFHD = isFHD || reFHD.MatchString(tag.Value)
					isHD = isHD || reHD.MatchString(tag.Value)
					isSD = isSD || reSD.MatchString(tag.Value)
					if !isFHD && !isHD && !isSD {
						isHD = true
					}
				}
			case "group-title":
				tag.Value = applyReplacements(replacements.Groups, tag.Value)
				if c.DivideByRes {
					switch {
					case isFHD:
						tag.Value = tag.Value + " FHD"
					case isHD:
						tag.Value = tag.Value + " HD"
					case isSD:
						tag.Value = tag.Value + " SD"
					}
				}
			}
		}

		if c.DivideByRes {
			switch {
			case isFHD:
				track.Name = reFHD.ReplaceAllString(track.Name, "")
			case isHD:
				track.Name = reHD.ReplaceAllString(track.Name, "")
			case isSD:
				track.Name = reSD.ReplaceAllString(track.Name, "")
			}
		}
	}

	ret := 0
	into.WriteString("#EXTM3U\n") // nolint: errcheck
	for i, track := range tracks {
		tvgName := track.Name
		var buffer bytes.Buffer
		buffer.WriteString("#EXTINF:")                       // nolint: errcheck
		buffer.WriteString(fmt.Sprintf("%d ", track.Length)) // nolint: errcheck
		for ti := range track.Tags {
			if track.Tags[ti].Name == "tvg-name" {
				tvgName = track.Tags[ti].Value
			}
			if track.Tags[ti].Name == "tvg-logo" && strings.Contains(track.Tags[ti].Value, ",") {
				log.Printf("[iptv-proxy] tvg-logo contained comma, clearing (track: %q)", track.Name)
				track.Tags[ti].Value = ""
			}
			if ti == len(track.Tags)-1 {
				buffer.WriteString(fmt.Sprintf("%s=%q", track.Tags[ti].Name, track.Tags[ti].Value)) // nolint: errcheck
				continue
			}
			buffer.WriteString(fmt.Sprintf("%s=%q ", track.Tags[ti].Name, track.Tags[ti].Value)) // nolint: errcheck
		}

		uri, err := c.replaceURL(track.URI, i-ret, xtream)
		if err != nil {
			ret++
			log.Printf("ERROR: track: %s: %s", track.Name, err)
			continue
		}
		if !xtream && c.trackIndexInPlaylist != nil {
			c.trackIndexInPlaylist[track.URI] = i - ret
		}

		displayName := track.Name
		if track.Name == "dpr_auto" || track.Name == "h_256" || strings.Contains(track.Name, "320\"") {
			displayName = tvgName
		}
		into.WriteString(fmt.Sprintf("%s, %s\n%s\n", buffer.String(), displayName, uri)) // nolint: errcheck

		filteredTrack = append(filteredTrack, track)
	}
	// Avoid clearing the playlist when filters would remove everything (e.g. bad/misapplied settings in e2e)
	if len(filteredTrack) == 0 && len(c.playlist.Tracks) > 0 {
		log.Printf("[iptv-proxy] WARN: filters would remove all tracks; keeping current playlist")
		return into.Sync()
	}
	c.playlist.Tracks = filteredTrack

	return into.Sync()
}

// ReplaceURL replace original playlist url by proxy url
func (c *Config) replaceURL(uri string, trackIndex int, xtream bool) (string, error) {
	oriURL, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	protocol := "http"
	if c.HTTPS {
		protocol = "https"
	}

	customEnd := strings.Trim(c.CustomEndpoint, "/")
	if customEnd != "" {
		customEnd = fmt.Sprintf("/%s", customEnd)
	}

	uriPath := oriURL.EscapedPath()
	if xtream {
		uriPath = strings.ReplaceAll(uriPath, c.XtreamUser.PathEscape(), url.PathEscape(c.pathAuthUser()))
		uriPath = strings.ReplaceAll(uriPath, c.XtreamPassword.PathEscape(), url.PathEscape(c.pathAuthPassword()))
	} else {
		uriPath = path.Join("/", c.endpointAntiColision, c.pathAuthUser(), c.pathAuthPassword(), fmt.Sprintf("%d", trackIndex), path.Base(uriPath))
	}

	basicAuth := oriURL.User.String()
	if basicAuth != "" {
		basicAuth += "@"
	}

	hostname := "localhost"
	if c.HostConfig != nil && c.HostConfig.Hostname != "" {
		hostname = c.HostConfig.Hostname
	}
	port := c.AdvertisedPort
	if port == 0 && c.HostConfig != nil {
		port = c.HostConfig.Port
	}

	newURI := fmt.Sprintf(
		"%s://%s%s:%d%s%s",
		protocol,
		basicAuth,
		hostname,
		port,
		customEnd,
		uriPath,
	)

	newURL, err := url.Parse(newURI)
	if err != nil {
		return "", err
	}

	return newURL.String(), nil
}

// compileRegexList compiles each pattern; invalid patterns are logged and skipped.
func compileRegexList(patterns []string, label string) []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, p := range patterns {
		if p == "" {
			continue
		}
		re, err := regexp.Compile(p)
		if err != nil {
			log.Printf("[iptv-proxy] Invalid %s pattern %q: %v", label, p, err)
			continue
		}
		out = append(out, re)
	}
	return out
}

func getGroupTitle(track m3u.Track) string {
	for _, t := range track.Tags {
		if t.Name == "group-title" {
			return t.Value
		}
	}
	return ""
}

// matchInclusionExclusion returns true if the track should be kept.
// Empty inclusion list = allow all; empty exclusion list = exclude none.
func matchInclusionExclusion(group, channelName string, groupIncl, groupExcl, channelIncl, channelExcl []*regexp.Regexp) bool {
	matchAny := func(s string, list []*regexp.Regexp) bool {
		for _, re := range list {
			if re.MatchString(s) {
				return true
			}
		}
		return false
	}
	if len(groupIncl) > 0 && !matchAny(group, groupIncl) {
		return false
	}
	if len(groupExcl) > 0 && matchAny(group, groupExcl) {
		return false
	}
	if len(channelIncl) > 0 && !matchAny(channelName, channelIncl) {
		return false
	}
	if len(channelExcl) > 0 && matchAny(channelName, channelExcl) {
		return false
	}
	return true
}
