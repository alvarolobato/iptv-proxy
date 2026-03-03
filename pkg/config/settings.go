/*
 * Settings file (settings.json) structure. When present in the data folder,
 * values take precedence over flags and environment variables; a warning is logged per overridden key.
 */

package config

// SettingsJSON is the shape of settings.json in the data folder (--data-folder).
// Any non-zero value overrides the corresponding flag/env; see ApplyTo and LogOverrides.
type SettingsJSON struct {
	// M3U and output
	M3UURL        string `json:"m3u_url,omitempty"`
	M3UFileName   string `json:"m3u_file_name,omitempty"`
	CustomEndpoint string `json:"custom_endpoint,omitempty"`
	CustomID      string `json:"custom_id,omitempty"`

	// Network and auth
	Port           int    `json:"port,omitempty"`
	AdvertisedPort int    `json:"advertised_port,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	HTTPS          bool   `json:"https,omitempty"`
	User           string `json:"user,omitempty"`
	Password       string `json:"password,omitempty"`

	// Xtream
	XtreamUser        string `json:"xtream_user,omitempty"`
	XtreamPassword    string `json:"xtream_password,omitempty"`
	XtreamBaseURL     string `json:"xtream_base_url,omitempty"`
	XtreamAPIGet      bool   `json:"xtream_api_get,omitempty"`
	M3UCacheExpiration int   `json:"m3u_cache_expiration,omitempty"`

	// Filter (inclusions/exclusions) and replacement — all from settings only
	GroupInclusions   []string `json:"group_inclusions,omitempty"`   // regex list: keep if group matches any (empty = all)
	GroupExclusions   []string `json:"group_exclusions,omitempty"`   // regex list: drop if group matches any
	ChannelInclusions []string `json:"channel_inclusions,omitempty"` // regex list: keep if channel name matches any (empty = all)
	ChannelExclusions []string `json:"channel_exclusions,omitempty"` // regex list: drop if channel name matches any
	DivideByRes       bool     `json:"divide_by_res,omitempty"`
	Replacements      *ReplacementsInSettings `json:"replacements,omitempty"`

	// XMLTV
	XMLTVCacheTTL        string `json:"xmltv_cache_ttl,omitempty"` // e.g. "1h"
	XMLTVCacheMaxEntries int    `json:"xmltv_cache_max_entries,omitempty"`

	// Debug and advanced
	DebugLoggingEnabled       bool   `json:"debug_logging,omitempty"`
	CacheFolder               string `json:"cache_folder,omitempty"`
	UseXtreamAdvancedParsing bool   `json:"use_xtream_advanced_parsing,omitempty"`

	// UI
	UIPort int `json:"ui_port,omitempty"`
}

// ReplacementsInSettings is the replacements section inside settings.json (replaces standalone replacements.json).
// Keys match the former replacements.json for compatibility.
type ReplacementsInSettings struct {
	Global []ReplacementRule `json:"global-replacements,omitempty"`
	Names  []ReplacementRule `json:"names-replacements,omitempty"`
	Groups []ReplacementRule `json:"groups-replacements,omitempty"`
}

// ReplacementRule is a single regex replace rule in settings.
type ReplacementRule struct {
	Replace string `json:"replace"`
	With    string `json:"with"`
}

// CurrentFromProxyConfig returns a SettingsJSON populated from the running ProxyConfig (CLI/env).
// Used to prepopulate the UI when a key is not set in settings.json.
func CurrentFromProxyConfig(p *ProxyConfig) SettingsJSON {
	if p == nil {
		return SettingsJSON{}
	}
	s := SettingsJSON{}
	if p.RemoteURL != nil {
		s.M3UURL = p.RemoteURL.String()
	}
	s.M3UFileName = p.M3UFileName
	s.CustomEndpoint = p.CustomEndpoint
	s.CustomID = p.CustomId
	if p.HostConfig != nil {
		s.Port = p.HostConfig.Port
		s.Hostname = p.HostConfig.Hostname
	}
	s.AdvertisedPort = p.AdvertisedPort
	s.HTTPS = p.HTTPS
	s.User = p.User.String()
	s.Password = p.Password.String()
	s.XtreamUser = p.XtreamUser.String()
	s.XtreamPassword = p.XtreamPassword.String()
	s.XtreamBaseURL = p.XtreamBaseURL
	s.XtreamAPIGet = p.XtreamGenerateApiGet
	s.M3UCacheExpiration = p.M3UCacheExpiration
	s.DivideByRes = p.DivideByRes
	s.DebugLoggingEnabled = p.DebugLoggingEnabled
	s.CacheFolder = p.CacheFolder
	s.UseXtreamAdvancedParsing = p.UseXtreamAdvancedParsing
	s.UIPort = p.UIPort
	if p.XMLTVCacheTTL > 0 {
		s.XMLTVCacheTTL = p.XMLTVCacheTTL.String()
	}
	s.XMLTVCacheMaxEntries = p.XMLTVCacheMaxEntries
	return s
}

// MergeWithCurrent returns a copy of file with empty/zero fields filled from current (flag/env).
// Used so the UI can show effective values while still knowing what is actually in the file.
func MergeWithCurrent(file, current SettingsJSON) SettingsJSON {
	out := file
	if out.M3UURL == "" {
		out.M3UURL = current.M3UURL
	}
	if out.M3UFileName == "" {
		out.M3UFileName = current.M3UFileName
	}
	if out.CustomEndpoint == "" {
		out.CustomEndpoint = current.CustomEndpoint
	}
	if out.CustomID == "" {
		out.CustomID = current.CustomID
	}
	if out.Port == 0 {
		out.Port = current.Port
	}
	if out.AdvertisedPort == 0 {
		out.AdvertisedPort = current.AdvertisedPort
	}
	if out.Hostname == "" {
		out.Hostname = current.Hostname
	}
	if !out.HTTPS {
		out.HTTPS = current.HTTPS
	}
	if out.User == "" {
		out.User = current.User
	}
	if out.Password == "" {
		out.Password = current.Password
	}
	if out.XtreamUser == "" {
		out.XtreamUser = current.XtreamUser
	}
	if out.XtreamPassword == "" {
		out.XtreamPassword = current.XtreamPassword
	}
	if out.XtreamBaseURL == "" {
		out.XtreamBaseURL = current.XtreamBaseURL
	}
	if !out.XtreamAPIGet {
		out.XtreamAPIGet = current.XtreamAPIGet
	}
	if out.M3UCacheExpiration == 0 {
		out.M3UCacheExpiration = current.M3UCacheExpiration
	}
	if len(out.GroupInclusions) == 0 {
		out.GroupInclusions = current.GroupInclusions
	}
	if len(out.GroupExclusions) == 0 {
		out.GroupExclusions = current.GroupExclusions
	}
	if len(out.ChannelInclusions) == 0 {
		out.ChannelInclusions = current.ChannelInclusions
	}
	if len(out.ChannelExclusions) == 0 {
		out.ChannelExclusions = current.ChannelExclusions
	}
	if !out.DivideByRes {
		out.DivideByRes = current.DivideByRes
	}
	if out.Replacements == nil && current.Replacements != nil {
		out.Replacements = current.Replacements
	}
	if out.XMLTVCacheTTL == "" {
		out.XMLTVCacheTTL = current.XMLTVCacheTTL
	}
	if out.XMLTVCacheMaxEntries == 0 {
		out.XMLTVCacheMaxEntries = current.XMLTVCacheMaxEntries
	}
	if !out.DebugLoggingEnabled {
		out.DebugLoggingEnabled = current.DebugLoggingEnabled
	}
	if out.CacheFolder == "" {
		out.CacheFolder = current.CacheFolder
	}
	if !out.UseXtreamAdvancedParsing {
		out.UseXtreamAdvancedParsing = current.UseXtreamAdvancedParsing
	}
	if out.UIPort == 0 {
		out.UIPort = current.UIPort
	}
	return out
}

// slicesEqual returns true if a and b have the same length and elements.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// replacementsEqual compares two ReplacementsInSettings (nil or same content).
func replacementsEqual(a, b *ReplacementsInSettings) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !slicesReplacementsEqual(a.Global, b.Global) || !slicesReplacementsEqual(a.Names, b.Names) || !slicesReplacementsEqual(a.Groups, b.Groups) {
		return false
	}
	return true
}

func slicesReplacementsEqual(a, b []ReplacementRule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Replace != b[i].Replace || a[i].With != b[i].With {
			return false
		}
	}
	return true
}

// SettingsOverridesOnly returns a SettingsJSON containing only fields where current differs from default.
// Used when writing settings.json so only user overrides are persisted, not flag/env defaults.
func SettingsOverridesOnly(current, defaultVal *SettingsJSON) SettingsJSON {
	if defaultVal == nil {
		return *current
	}
	out := SettingsJSON{}
	if current.M3UURL != defaultVal.M3UURL {
		out.M3UURL = current.M3UURL
	}
	if current.M3UFileName != defaultVal.M3UFileName {
		out.M3UFileName = current.M3UFileName
	}
	if current.CustomEndpoint != defaultVal.CustomEndpoint {
		out.CustomEndpoint = current.CustomEndpoint
	}
	if current.CustomID != defaultVal.CustomID {
		out.CustomID = current.CustomID
	}
	if current.Port != defaultVal.Port {
		out.Port = current.Port
	}
	if current.AdvertisedPort != defaultVal.AdvertisedPort {
		out.AdvertisedPort = current.AdvertisedPort
	}
	if current.Hostname != defaultVal.Hostname {
		out.Hostname = current.Hostname
	}
	if current.HTTPS != defaultVal.HTTPS {
		out.HTTPS = current.HTTPS
	}
	if current.User != defaultVal.User {
		out.User = current.User
	}
	if current.Password != defaultVal.Password {
		out.Password = current.Password
	}
	if current.XtreamUser != defaultVal.XtreamUser {
		out.XtreamUser = current.XtreamUser
	}
	if current.XtreamPassword != defaultVal.XtreamPassword {
		out.XtreamPassword = current.XtreamPassword
	}
	if current.XtreamBaseURL != defaultVal.XtreamBaseURL {
		out.XtreamBaseURL = current.XtreamBaseURL
	}
	if current.XtreamAPIGet != defaultVal.XtreamAPIGet {
		out.XtreamAPIGet = current.XtreamAPIGet
	}
	if current.M3UCacheExpiration != defaultVal.M3UCacheExpiration {
		out.M3UCacheExpiration = current.M3UCacheExpiration
	}
	if !slicesEqual(current.GroupInclusions, defaultVal.GroupInclusions) {
		out.GroupInclusions = current.GroupInclusions
	}
	if !slicesEqual(current.GroupExclusions, defaultVal.GroupExclusions) {
		out.GroupExclusions = current.GroupExclusions
	}
	if !slicesEqual(current.ChannelInclusions, defaultVal.ChannelInclusions) {
		out.ChannelInclusions = current.ChannelInclusions
	}
	if !slicesEqual(current.ChannelExclusions, defaultVal.ChannelExclusions) {
		out.ChannelExclusions = current.ChannelExclusions
	}
	if current.DivideByRes != defaultVal.DivideByRes {
		out.DivideByRes = current.DivideByRes
	}
	if !replacementsEqual(current.Replacements, defaultVal.Replacements) {
		out.Replacements = current.Replacements
	}
	if current.XMLTVCacheTTL != defaultVal.XMLTVCacheTTL {
		out.XMLTVCacheTTL = current.XMLTVCacheTTL
	}
	if current.XMLTVCacheMaxEntries != defaultVal.XMLTVCacheMaxEntries {
		out.XMLTVCacheMaxEntries = current.XMLTVCacheMaxEntries
	}
	if current.DebugLoggingEnabled != defaultVal.DebugLoggingEnabled {
		out.DebugLoggingEnabled = current.DebugLoggingEnabled
	}
	if current.CacheFolder != defaultVal.CacheFolder {
		out.CacheFolder = current.CacheFolder
	}
	if current.UseXtreamAdvancedParsing != defaultVal.UseXtreamAdvancedParsing {
		out.UseXtreamAdvancedParsing = current.UseXtreamAdvancedParsing
	}
	if current.UIPort != defaultVal.UIPort {
		out.UIPort = current.UIPort
	}
	return out
}
