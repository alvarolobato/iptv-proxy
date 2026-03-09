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

package config

import (
	"net/url"
	"time"
)

// CredentialString represents an iptv-proxy credential.
type CredentialString string

// PathEscape escapes the credential for an url path.
func (c CredentialString) PathEscape() string {
	return url.PathEscape(string(c))
}

// String returns the credential string.
func (c CredentialString) String() string {
	return string(c)
}

// HostConfiguration containt host infos
type HostConfiguration struct {
	Hostname string
	Port     int
}

// ProxyConfig Contain original m3u playlist and HostConfiguration
type ProxyConfig struct {
	HostConfig           *HostConfiguration
	XtreamUser           CredentialString
	XtreamPassword       CredentialString
	XtreamBaseURL        string
	XtreamGenerateApiGet bool
	M3UCacheExpiration   int
	XMLTVCacheTTL        time.Duration // 0 = no cache
	XMLTVCacheMaxEntries int           // max cached responses (0 = use default)
	M3UFileName          string
	CustomEndpoint       string
	CustomId             string
	RemoteURL            *url.URL
	AdvertisedPort       int
	HTTPS                bool
	User, Password       CredentialString
	// M3U filter (from settings.json only) and replacement
	GroupInclusions    []string // keep only if group-title matches any (empty = all)
	GroupExclusions    []string // drop if group-title matches any
	ChannelInclusions  []string // keep only if channel name matches any (empty = all)
	ChannelExclusions  []string // drop if channel name matches any
	DataFolder         string   // folder for settings.json and replacement rules (--data-folder)
	DivideByRes  bool   // divide groups by resolution (FHD/HD/SD)
	// UseXtreamAdvancedParsing uses alternate parsing for some Xtream requests to preserve raw provider response (default false).
	UseXtreamAdvancedParsing bool
	// DebugLoggingEnabled enables verbose debug logging when true.
	DebugLoggingEnabled bool
	// CacheFolder is the directory for saving provider/client responses (when non-empty). Use filepath.Join with this; no trailing separator.
	CacheFolder string
	// UIPort is the port for the configuration UI (default 8081; 0 = disabled).
	UIPort int

	// Elasticsearch stats configuration. All fields are optional; when ESURL is empty, stats are disabled.
	ESUrl      string // e.g. https://mycluster.es.io
	ESApiKey   string // base64-encoded id:key
	ESUsername string // alternative to API key
	ESPassword string
	// ESIndexPrefix is the prefix for all stats index names (default: iptv).
	ESIndexPrefix string
	// StatsEnabled explicitly enables/disables stats (default: true when ESUrl is set).
	StatsEnabled bool
}
