package server

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// StartupContext holds information for the server-ready log (data folder, files, overrides).
type StartupContext struct {
	HidePasswords        bool
	ConfigFilePath       string // path to config file, or empty if not used
	DataFolder           string // --data-folder path
	SettingsPath         string // full path to settings.json
	SettingsPresent      bool
	ReplacementsInFile  string // "settings.json" or "replacements.json" or ""
	ReplacementCounts   struct{ Global, Names, Groups int }
	OverriddenBySettings []string
}

// maskCred hides the value when HidePasswords is true.
func (s *StartupContext) maskCred(hide bool, v string) string {
	if !hide || v == "" {
		return v
	}
	if len(v) <= 4 {
		return "****"
	}
	return "****" + v[len(v)-4:]
}

// ServeWithContext runs the server and logs a comprehensive startup summary before listening.
func (c *Config) ServeWithContext(ctx *StartupContext) error {
	if err := c.playlistInitialization(); err != nil {
		return err
	}

	if c.UIPort > 0 {
		go c.runUIServer()
	}

	router := gin.Default()
	router.Use(cors.Default())
	group := router.Group("/")
	c.routes(group)

	if ctx != nil {
		c.logServerReady(ctx)
	} else {
		log.Printf("[iptv-proxy] Server starting, binding to :%d", c.HostConfig.Port)
	}

	return router.Run(fmt.Sprintf(":%d", c.HostConfig.Port))
}

// logServerReady prints a full reference block: listening addresses, URLs, data folder, files, effective config.
func (c *Config) logServerReady(ctx *StartupContext) {
	hide := ctx.HidePasswords
	mask := func(v string) string { return ctx.maskCred(hide, v) }

	scheme := "http"
	if c.HTTPS {
		scheme = "https"
	}
	host := c.HostConfig.Hostname
	if host == "" {
		host = "localhost"
	}
	port := c.AdvertisedPort
	if port == 0 {
		port = c.HostConfig.Port
	}
	customEnd := strings.Trim(c.CustomEndpoint, "/")
	if customEnd != "" {
		customEnd = "/" + customEnd
	}

	// Listening
	log.Printf("[iptv-proxy] ========== Server ready ==========")
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			log.Printf("[iptv-proxy] Listening: %s:%d (proxy)", ipnet.IP, c.HostConfig.Port)
		}
	}
	log.Printf("[iptv-proxy] Listening: 0.0.0.0:%d (proxy)", c.HostConfig.Port)
	if c.UIPort > 0 {
		log.Printf("[iptv-proxy] Listening: 0.0.0.0:%d (configuration UI)", c.UIPort)
	}

	// URLs
	base := fmt.Sprintf("%s://%s:%d%s", scheme, host, port, customEnd)
	log.Printf("[iptv-proxy] --- URLs (use these in your apps) ---")
	log.Printf("[iptv-proxy] M3U playlist: %s/%s?username=%s&password=%s", base, c.M3UFileName, c.User, mask(string(c.Password)))
	if c.XtreamBaseURL != "" {
		log.Printf("[iptv-proxy] Xtream API base: %s", base)
		log.Printf("[iptv-proxy]   get.php: %s/get.php?username=%s&password=%s", base, c.User, mask(string(c.Password)))
		log.Printf("[iptv-proxy]   player_api.php: %s/player_api.php", base)
		log.Printf("[iptv-proxy]   xmltv.php (EPG): %s/xmltv.php?username=%s&password=%s", base, c.User, mask(string(c.Password)))
	}
	if c.UIPort > 0 {
		uiHost := host
		if host == "localhost" || host == "" {
			uiHost = "localhost"
		}
		log.Printf("[iptv-proxy] Configuration UI: http://%s:%d/", uiHost, c.UIPort)
	}

	// Data folder and files
	log.Printf("[iptv-proxy] --- Data folder and files ---")
	if ctx.DataFolder != "" {
		log.Printf("[iptv-proxy] Data folder: %s", ctx.DataFolder)
		log.Printf("[iptv-proxy]   settings.json: %s (present: %v)", ctx.SettingsPath, ctx.SettingsPresent)
		if ctx.ReplacementsInFile != "" {
			log.Printf("[iptv-proxy]   Replacements from: %s (global: %d, names: %d, groups: %d)",
				ctx.ReplacementsInFile, ctx.ReplacementCounts.Global, ctx.ReplacementCounts.Names, ctx.ReplacementCounts.Groups)
		}
	} else {
		log.Printf("[iptv-proxy] Data folder: not set (no settings.json or replacements)")
	}
	if ctx.ConfigFilePath != "" {
		log.Printf("[iptv-proxy] Config file: %s", ctx.ConfigFilePath)
	} else {
		log.Printf("[iptv-proxy] Config file: none (using flags/env)")
	}

	// Effective configuration and source
	log.Printf("[iptv-proxy] --- Effective configuration (source: flag/env or settings.json) ---")
	overriddenSet := make(map[string]bool)
	for _, k := range ctx.OverriddenBySettings {
		overriddenSet[k] = true
	}
	source := func(key string) string {
		if overriddenSet[key] {
			return "settings.json"
		}
		return "flag/env"
	}

	m3uURLStr := ""
	if c.RemoteURL != nil {
		m3uURLStr = c.RemoteURL.String()
	}
	rows := []struct{ key, value, src string }{
		{"m3u_url", m3uURLStr, source("m3u_url")},
		{"m3u_file_name", c.M3UFileName, source("m3u_file_name")},
		{"custom_endpoint", c.CustomEndpoint, source("custom_endpoint")},
		{"custom_id", c.CustomId, source("custom_id")},
		{"port", fmt.Sprintf("%d", c.HostConfig.Port), source("port")},
		{"advertised_port", fmt.Sprintf("%d", c.AdvertisedPort), source("advertised_port")},
		{"hostname", c.HostConfig.Hostname, source("hostname")},
		{"https", fmt.Sprintf("%v", c.HTTPS), source("https")},
		{"user", string(c.User), source("user")},
		{"password", mask(string(c.Password)), source("password")},
		{"xtream_user", string(c.XtreamUser), source("xtream_user")},
		{"xtream_password", mask(string(c.XtreamPassword)), source("xtream_password")},
		{"xtream_base_url", c.XtreamBaseURL, source("xtream_base_url")},
		{"xtream_api_get", fmt.Sprintf("%v", c.XtreamGenerateApiGet), source("xtream_api_get")},
		{"m3u_cache_expiration", fmt.Sprintf("%d", c.M3UCacheExpiration), source("m3u_cache_expiration")},
		{"group_inclusions", strings.Join(c.GroupInclusions, ", "), source("group_inclusions")},
		{"group_exclusions", strings.Join(c.GroupExclusions, ", "), source("group_exclusions")},
		{"channel_inclusions", strings.Join(c.ChannelInclusions, ", "), source("channel_inclusions")},
		{"channel_exclusions", strings.Join(c.ChannelExclusions, ", "), source("channel_exclusions")},
		{"data_folder", c.DataFolder, "flag/env"},
		{"divide_by_res", fmt.Sprintf("%v", c.DivideByRes), source("divide_by_res")},
		{"xmltv_cache_ttl", fmt.Sprintf("%v", c.XMLTVCacheTTL), source("xmltv_cache_ttl")},
		{"xmltv_cache_max_entries", fmt.Sprintf("%d", c.XMLTVCacheMaxEntries), source("xmltv_cache_max_entries")},
		{"debug_logging", fmt.Sprintf("%v", c.DebugLoggingEnabled), source("debug_logging")},
		{"cache_folder", c.CacheFolder, source("cache_folder")},
		{"use_xtream_advanced_parsing", fmt.Sprintf("%v", c.UseXtreamAdvancedParsing), source("use_xtream_advanced_parsing")},
		{"ui_port", fmt.Sprintf("%d", c.UIPort), source("ui_port")},
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].key < rows[j].key })
	for _, r := range rows {
		log.Printf("[iptv-proxy]   %s = %q [%s]", r.key, r.value, r.src)
	}

	log.Printf("[iptv-proxy] ========================================")
}
