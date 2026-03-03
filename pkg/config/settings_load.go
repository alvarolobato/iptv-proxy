package config

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// EnsureStubSettings creates settings.json in dir with empty structure (including replacements) if the file does not exist.
func EnsureStubSettings(dir string) {
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "settings.json")
	_, err := os.Stat(path)
	if err == nil {
		return
	}
	if !os.IsNotExist(err) {
		log.Printf("[iptv-proxy] Cannot create stub settings %s: %v", path, err)
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[iptv-proxy] Could not create data folder %s: %v", dir, err)
		return
	}
	stub := SettingsJSON{
		Replacements: &ReplacementsInSettings{
			Global: []ReplacementRule{},
			Names:  []ReplacementRule{},
			Groups: []ReplacementRule{},
		},
		GroupInclusions:   []string{},
		GroupExclusions:   []string{},
		ChannelInclusions: []string{},
		ChannelExclusions: []string{},
	}
	data, err := json.MarshalIndent(stub, "", "  ")
	if err != nil {
		log.Printf("[iptv-proxy] Could not marshal stub settings: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[iptv-proxy] Could not write stub %s: %v", path, err)
		return
	}
	log.Printf("[iptv-proxy] Created stub %s", path)
}

// LoadSettings reads settings.json from dir (e.g. data folder). Returns nil if file does not exist or dir is empty.
func LoadSettings(dir string) (*SettingsJSON, error) {
	if dir == "" {
		return nil, nil
	}
	path := filepath.Join(dir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s SettingsJSON
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ApplyTo applies non-zero settings to conf and logs a warning for each key that overrides.
// It returns the list of keys that were overridden by settings.json (for startup summary).
// parseDuration is used for xmltv_cache_ttl (e.g. "1h"). RemoteURL is not set from settings (m3u_url is a URL).
func ApplyTo(s *SettingsJSON, conf *ProxyConfig, parseDuration func(string) time.Duration) (overridden []string) {
	if s == nil {
		return nil
	}
	warn := func(key string) {
		log.Printf("[iptv-proxy] WARN: %s overridden by settings.json", key)
		overridden = append(overridden, key)
	}

	if s.M3UURL != "" {
		u, err := url.Parse(s.M3UURL)
		if err == nil {
			conf.RemoteURL = u
			warn("m3u_url")
		}
	}
	if s.M3UFileName != "" {
		conf.M3UFileName = s.M3UFileName
		warn("m3u_file_name")
	}
	if s.CustomEndpoint != "" {
		conf.CustomEndpoint = s.CustomEndpoint
		warn("custom_endpoint")
	}
	if s.CustomID != "" {
		conf.CustomId = s.CustomID
		warn("custom_id")
	}
	if s.Port != 0 {
		conf.HostConfig.Port = s.Port
		warn("port")
	}
	if s.AdvertisedPort != 0 {
		conf.AdvertisedPort = s.AdvertisedPort
		warn("advertised_port")
	}
	if s.Hostname != "" {
		conf.HostConfig.Hostname = s.Hostname
		warn("hostname")
	}
	if s.HTTPS {
		conf.HTTPS = true
		warn("https")
	}
	if s.User != "" {
		conf.User = CredentialString(s.User)
		warn("user")
	}
	if s.Password != "" {
		conf.Password = CredentialString(s.Password)
		warn("password")
	}
	if s.XtreamUser != "" {
		conf.XtreamUser = CredentialString(s.XtreamUser)
		warn("xtream_user")
	}
	if s.XtreamPassword != "" {
		conf.XtreamPassword = CredentialString(s.XtreamPassword)
		warn("xtream_password")
	}
	if s.XtreamBaseURL != "" {
		conf.XtreamBaseURL = s.XtreamBaseURL
		warn("xtream_base_url")
	}
	if s.XtreamAPIGet {
		conf.XtreamGenerateApiGet = true
		warn("xtream_api_get")
	}
	if s.M3UCacheExpiration != 0 {
		conf.M3UCacheExpiration = s.M3UCacheExpiration
		warn("m3u_cache_expiration")
	}
	if len(s.GroupInclusions) > 0 {
		conf.GroupInclusions = append([]string(nil), s.GroupInclusions...)
		warn("group_inclusions")
	}
	if len(s.GroupExclusions) > 0 {
		conf.GroupExclusions = append([]string(nil), s.GroupExclusions...)
		warn("group_exclusions")
	}
	if len(s.ChannelInclusions) > 0 {
		conf.ChannelInclusions = append([]string(nil), s.ChannelInclusions...)
		warn("channel_inclusions")
	}
	if len(s.ChannelExclusions) > 0 {
		conf.ChannelExclusions = append([]string(nil), s.ChannelExclusions...)
		warn("channel_exclusions")
	}
	if s.DivideByRes {
		conf.DivideByRes = true
		warn("divide_by_res")
	}
	if s.XMLTVCacheTTL != "" {
		conf.XMLTVCacheTTL = parseDuration(s.XMLTVCacheTTL)
		warn("xmltv_cache_ttl")
	}
	if s.XMLTVCacheMaxEntries != 0 {
		conf.XMLTVCacheMaxEntries = s.XMLTVCacheMaxEntries
		warn("xmltv_cache_max_entries")
	}
	if s.DebugLoggingEnabled {
		conf.DebugLoggingEnabled = true
		warn("debug_logging")
	}
	if s.CacheFolder != "" {
		conf.CacheFolder = s.CacheFolder
		warn("cache_folder")
	}
	if s.UseXtreamAdvancedParsing {
		conf.UseXtreamAdvancedParsing = true
		warn("use_xtream_advanced_parsing")
	}
	if s.UIPort != 0 {
		conf.UIPort = s.UIPort
		warn("ui_port")
	}
	return overridden
}
