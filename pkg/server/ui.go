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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamesnetherton/m3u"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
)

// getReplacements returns compiled replacements from settings or legacy file (same as marshallInto).
func (c *Config) getReplacements() Replacements {
	if c.settings != nil && c.settings.Replacements != nil {
		return ReplacementsFromSettings(c.settings.Replacements)
	}
	if c.DataFolder != "" {
		return loadReplacements(filepath.Join(c.DataFolder, "replacements.json"))
	}
	return Replacements{}
}

// runUIServer starts the configuration UI HTTP server on c.ProxyConfig.UIPort. Call from Serve() in a goroutine.
func (c *Config) runUIServer() {
	port := c.ProxyConfig.UIPort
	if port <= 0 {
		return
	}
	router := gin.Default()

	// Readiness for e2e: 200 only when playlist has been loaded (so Playwright waits for data before running tests)
	router.GET("/api/ready", func(ctx *gin.Context) {
		n := 0
		if c.playlist != nil {
			n = len(c.playlist.Tracks)
		}
		if n == 0 {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "tracks": 0})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"ready": true, "tracks": n})
	})

	// API: list unique group titles with channel count, replacements applied, excluded/replaced flags (cache 2 min)
	router.GET("/api/groups", func(ctx *gin.Context) {
		groups := c.groupsProcessed()
		ctx.Header("Cache-Control", "private, max-age=120")
		ctx.JSON(http.StatusOK, groups)
	})

	// API: list channels with replacements applied, excluded/replaced flags (cache 2 min)
	router.GET("/api/channels", func(ctx *gin.Context) {
		channels := c.channelsProcessed()
		ctx.Header("Cache-Control", "private, max-age=120")
		ctx.JSON(http.StatusOK, channels)
	})

	// API: get replacements (from settings.json replacements section or legacy replacements.json)
	router.GET("/api/replacements", func(ctx *gin.Context) {
		data, err := c.readReplacementsFile()
		if err != nil {
			log.Printf("[iptv-proxy] GET /api/replacements: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.Data(http.StatusOK, "application/json", data)
	})

	// API: save replacements (writes into settings.json replacements section, or legacy file)
	router.PUT("/api/replacements", func(ctx *gin.Context) {
		var raw replacementsJSON
		if err := ctx.ShouldBindJSON(&raw); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := c.writeReplacementsFile(&raw); err != nil {
			log.Printf("[iptv-proxy] PUT /api/replacements: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(http.StatusOK)
	})

	// API: get settings as { "file": actual settings.json content, "effective": file merged with flag/env }.
	// The UI uses "effective" for the form and "file" for the Raw JSON tab and to know which keys are overrides.
	router.GET("/api/settings", func(ctx *gin.Context) {
		file, err := c.readSettingsFileStruct()
		if err != nil {
			log.Printf("[iptv-proxy] GET /api/settings: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		current := config.CurrentFromProxyConfig(c.ProxyConfig)
		effective := config.MergeWithCurrent(file, current)
		ctx.Header("Cache-Control", "no-store")
		ctx.JSON(http.StatusOK, gin.H{"file": file, "effective": effective})
	})

	// API: save full settings.json (applies filters and replacements immediately; no restart needed)
	router.PUT("/api/settings", func(ctx *gin.Context) {
		var s config.SettingsJSON
		if err := ctx.ShouldBindJSON(&s); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := c.writeSettingsFile(&s); err != nil {
			log.Printf("[iptv-proxy] PUT /api/settings: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.applyLiveSettings(&s)
		ctx.Status(http.StatusOK)
	})

	// User management API
	router.GET("/api/users", c.apiListUsers)
	router.POST("/api/users", c.apiCreateUser)
	router.PUT("/api/users/:username", c.apiUpdateUser)
	router.DELETE("/api/users/:username", c.apiDeleteUser)
	router.GET("/api/users/:username/watch", c.apiUserWatch)
	router.GET("/api/users/:username/access", c.apiGetUserAccess)
	router.PUT("/api/users/:username/access", c.apiSetUserAccess)

	// Stats API endpoints (Elasticsearch-backed; no-ops when ES not configured)
	c.registerStatsRoutes(router)

	// Serve embedded React UI (SPA fallback for /settings etc.)
	router.NoRoute(serveStaticUI)

	log.Printf("[iptv-proxy] Configuration UI listening on :%d", port)
	if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Printf("[iptv-proxy] UI server error: %v", err)
	}
}

// groupWithCountProcessed is returned by groupsProcessed: display name after replacements, excluded flag, replaced flag.
type groupWithCountProcessed struct {
	Name         string `json:"name"`
	ChannelCount int    `json:"channel_count"`
	Excluded     bool   `json:"excluded"`
	Replaced     bool   `json:"replaced"`
}

func (c *Config) groupsProcessed() []groupWithCountProcessed {
	groupInclRE := compileRegexList(c.GroupInclusions, "group_inclusions")
	groupExclRE := compileRegexList(c.GroupExclusions, "group_exclusions")
	channelInclRE := compileRegexList(c.ChannelInclusions, "channel_inclusions")
	channelExclRE := compileRegexList(c.ChannelExclusions, "channel_exclusions")
	replacements := c.getReplacements()

	type agg struct {
		count   int
		excluded bool
		replaced bool
	}
	byDisplay := make(map[string]*agg)

	tracksForAPI := c.fullPlaylistTracks
	if len(tracksForAPI) == 0 {
		tracksForAPI = c.playlist.Tracks
	}
	for _, track := range tracksForAPI {
		rawGroup := getGroupTitle(track)
		rawName := track.Name
		excluded := !matchInclusionExclusion(rawGroup, rawName, groupInclRE, groupExclRE, channelInclRE, channelExclRE)
		displayGroup := applyReplacements(replacements.Global, rawGroup)
		displayGroup = applyReplacements(replacements.Groups, displayGroup)
		replaced := displayGroup != rawGroup

		if byDisplay[displayGroup] == nil {
			byDisplay[displayGroup] = &agg{}
		}
		a := byDisplay[displayGroup]
		a.count++
		if excluded {
			a.excluded = true
		}
		if replaced {
			a.replaced = true
		}
	}

	out := make([]groupWithCountProcessed, 0, len(byDisplay))
	for name, a := range byDisplay {
		out = append(out, groupWithCountProcessed{
			Name:         name,
			ChannelCount: a.count,
			Excluded:     a.excluded,
			Replaced:     a.replaced,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// channelTypeFromURI returns the first path segment of the track URI (e.g. "live", "series", "movies").
// Used for type filter in the UI. Defaults to "live" if unparseable.
func channelTypeFromURI(uri string) string {
	if uri == "" {
		return "live"
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "live"
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "live"
	}
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "live"
	}
	t := strings.ToLower(parts[0])
	if t == "" {
		return "live"
	}
	return t
}

// channelRowProcessed is channel with replacements applied and excluded/replaced flags.
type channelRowProcessed struct {
	Name          string `json:"name"`
	Group         string `json:"group"`
	TvgID         string `json:"tvg_id"`
	TvgName       string `json:"tvg_name"`
	TvgLogo       string `json:"tvg_logo"`
	Type          string `json:"type"` // from URL path segment, e.g. "live", "series", "movies"
	Excluded      bool   `json:"excluded"`
	NameReplaced  bool   `json:"name_replaced"`
	GroupReplaced bool   `json:"group_replaced"`
	StreamURL     string `json:"stream_url,omitempty"` // proxified stream URL (only for included tracks in M3U mode)
}

func (c *Config) channelsProcessed() []channelRowProcessed {
	groupInclRE := compileRegexList(c.GroupInclusions, "group_inclusions")
	groupExclRE := compileRegexList(c.GroupExclusions, "group_exclusions")
	channelInclRE := compileRegexList(c.ChannelInclusions, "channel_inclusions")
	channelExclRE := compileRegexList(c.ChannelExclusions, "channel_exclusions")
	replacements := c.getReplacements()

	tracksForAPI := c.fullPlaylistTracks
	if len(tracksForAPI) == 0 {
		tracksForAPI = c.playlist.Tracks
	}

	// URI->index for stream URLs (set in marshallInto; fallback from current playlist if nil so API always has data).
	uriToIndex := c.trackIndexInPlaylist
	if uriToIndex == nil && c.playlist != nil && len(c.playlist.Tracks) > 0 {
		uriToIndex = make(map[string]int)
		for idx, t := range c.playlist.Tracks {
			uriToIndex[t.URI] = idx
		}
	}

	out := make([]channelRowProcessed, 0, len(tracksForAPI))
	for _, track := range tracksForAPI {
		rawGroup := getGroupTitle(track)
		rawName := track.Name
		excluded := !matchInclusionExclusion(rawGroup, rawName, groupInclRE, groupExclRE, channelInclRE, channelExclRE)

		displayName := applyReplacements(replacements.Global, rawName)
		displayName = applyReplacements(replacements.Names, displayName)
		displayGroup := applyReplacements(replacements.Global, rawGroup)
		displayGroup = applyReplacements(replacements.Groups, displayGroup)

		channelType := channelTypeFromURI(track.URI)

		streamURL := ""
		if uriToIndex != nil {
			if idx, ok := uriToIndex[track.URI]; ok {
				if u, err := c.replaceURL(track.URI, idx, false); err == nil {
					streamURL = u
				}
			}
		}

		row := channelRowProcessed{
			Name:          displayName,
			Group:         displayGroup,
			Type:          channelType,
			Excluded:      excluded,
			NameReplaced:  displayName != rawName,
			GroupReplaced: displayGroup != rawGroup,
			StreamURL:     streamURL,
		}
		for _, tag := range track.Tags {
			switch tag.Name {
			case "tvg-id":
				row.TvgID = tag.Value
			case "tvg-name":
				row.TvgName = tag.Value
			case "tvg-logo":
				row.TvgLogo = tag.Value
			}
			// type could be set from tag in future (e.g. tvg-type=vod)
		}
		out = append(out, row)
	}
	return out
}

func (c *Config) settingsPath() string {
	if c.DataFolder == "" {
		return ""
	}
	return filepath.Join(c.DataFolder, "settings.json")
}

func (c *Config) replacementsPath() string {
	if c.DataFolder == "" {
		return ""
	}
	return filepath.Join(c.DataFolder, "replacements.json")
}

func (c *Config) readSettingsFile() ([]byte, error) {
	stub := config.SettingsJSON{Replacements: &config.ReplacementsInSettings{}}
	path := c.settingsPath()
	if path == "" {
		return json.MarshalIndent(stub, "", "  ")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return json.MarshalIndent(stub, "", "  ")
		}
		return nil, err
	}
	return data, nil
}

// readSettingsFileStruct returns the parsed settings.json (empty struct if file missing or invalid).
func (c *Config) readSettingsFileStruct() (config.SettingsJSON, error) {
	data, err := c.readSettingsFile()
	if err != nil {
		return config.SettingsJSON{}, err
	}
	var s config.SettingsJSON
	if err := json.Unmarshal(data, &s); err != nil {
		return config.SettingsJSON{Replacements: &config.ReplacementsInSettings{}}, nil
	}
	if s.Replacements == nil {
		s.Replacements = &config.ReplacementsInSettings{}
	}
	return s, nil
}

func (c *Config) writeSettingsFile(s *config.SettingsJSON) error {
	path := c.settingsPath()
	if path == "" {
		return fmt.Errorf("data-folder not set")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	// Preserve users from in-memory state (users are managed via /api/users, not /api/settings).
	if len(s.Users) == 0 && len(c.ProxyConfig.Users) > 0 {
		s.Users = c.ProxyConfig.Users
	}
	toWrite := s
	if c.defaultSettings != nil {
		overrides := config.SettingsOverridesOnly(s, c.defaultSettings)
		toWrite = &overrides
	}
	data, err := json.MarshalIndent(toWrite, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// applyLiveSettings applies filter lists and replacements from s to the running config and regenerates the proxyfied M3U.
// Only safe-to-reload-at-runtime fields are applied (inclusions, exclusions, replacements); port/hostname etc. are unchanged.
// We restore the full playlist from fullPlaylistTracks before re-filtering so that the served M3U always reflects the
// current inclusion/exclusion rules applied to the full source list (not to a previously filtered subset).
func (c *Config) applyLiveSettings(s *config.SettingsJSON) {
	if s == nil {
		return
	}
	c.ProxyConfig.GroupInclusions = append([]string(nil), s.GroupInclusions...)
	c.ProxyConfig.GroupExclusions = append([]string(nil), s.GroupExclusions...)
	c.ProxyConfig.ChannelInclusions = append([]string(nil), s.ChannelInclusions...)
	c.ProxyConfig.ChannelExclusions = append([]string(nil), s.ChannelExclusions...)
	// Deep-copy settings so getReplacements() and groupsProcessed/channelsProcessed use the new data
	sCopy := *s
	if s.Replacements != nil {
		sCopy.Replacements = &config.ReplacementsInSettings{
			Global: append([]config.ReplacementRule(nil), s.Replacements.Global...),
			Names:  append([]config.ReplacementRule(nil), s.Replacements.Names...),
			Groups: append([]config.ReplacementRule(nil), s.Replacements.Groups...),
		}
	}
	c.settings = &sCopy
	// Restore full playlist so marshallInto applies current filters to the full list (not to already-filtered tracks).
	if len(c.fullPlaylistTracks) > 0 {
		c.playlist.Tracks = make([]m3u.Track, len(c.fullPlaylistTracks))
		copy(c.playlist.Tracks, c.fullPlaylistTracks)
	}
	if err := c.playlistInitialization(); err != nil {
		log.Printf("[iptv-proxy] applyLiveSettings: playlistInitialization: %v", err)
	}
}

// readReplacementsFile returns replacements as JSON. Prefers settings.json replacements section; falls back to replacements.json.
func (c *Config) readReplacementsFile() ([]byte, error) {
	stub := replacementsJSON{Global: []Replacement{}, Names: []Replacement{}, Groups: []Replacement{}}
	path := c.settingsPath()
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			var s config.SettingsJSON
			if json.Unmarshal(data, &s) == nil && s.Replacements != nil {
				out := replacementsJSON{
					Global: toReplacementSlice(s.Replacements.Global),
					Names:  toReplacementSlice(s.Replacements.Names),
					Groups: toReplacementSlice(s.Replacements.Groups),
				}
				return json.MarshalIndent(out, "", "  ")
			}
		}
	}
	legacyPath := c.replacementsPath()
	if legacyPath == "" {
		return json.MarshalIndent(stub, "", "  ")
	}
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return json.MarshalIndent(stub, "", "  ")
		}
		return nil, err
	}
	return data, nil
}

func toReplacementSlice(rules []config.ReplacementRule) []Replacement {
	out := make([]Replacement, 0, len(rules))
	for _, r := range rules {
		out = append(out, Replacement{Replace: r.Replace, With: r.With})
	}
	return out
}

// writeReplacementsFile writes replacements into settings.json (replacements section) or legacy replacements.json if no settings.
func (c *Config) writeReplacementsFile(raw *replacementsJSON) error {
	path := c.settingsPath()
	if path != "" {
		var s config.SettingsJSON
		data, err := os.ReadFile(path)
		if err == nil {
			_ = json.Unmarshal(data, &s)
		}
		if s.Replacements == nil {
			s.Replacements = &config.ReplacementsInSettings{}
		}
		s.Replacements.Global = toReplacementRuleSlice(raw.Global)
		s.Replacements.Names = toReplacementRuleSlice(raw.Names)
		s.Replacements.Groups = toReplacementRuleSlice(raw.Groups)
		return c.writeSettingsFile(&s)
	}
	legacyPath := c.replacementsPath()
	if legacyPath == "" {
		return fmt.Errorf("data-folder not set")
	}
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(legacyPath, data, 0644)
}

func toReplacementRuleSlice(r []Replacement) []config.ReplacementRule {
	out := make([]config.ReplacementRule, 0, len(r))
	for _, x := range r {
		out = append(out, config.ReplacementRule{Replace: x.Replace, With: x.With})
	}
	return out
}

// --- User management API ---

func (c *Config) apiListUsers(ctx *gin.Context) {
	c.mu.RLock()
	users := c.ProxyConfig.AllUsers()
	c.mu.RUnlock()
	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

type createUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"` // pointer to distinguish absent from false
}

func (c *Config) apiCreateUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !config.ValidUsername(req.Username) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid username: must be 1-64 alphanumeric, hyphen, or underscore characters and not a reserved name"})
		return
	}
	if req.Password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check uniqueness.
	for _, u := range c.ProxyConfig.Users {
		if u.Username == req.Username {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
			return
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC().Format(time.RFC3339)
	newUser := config.User{
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
		Enabled:     enabled,
		CreatedAt:   now,
	}
	c.ProxyConfig.Users = append(c.ProxyConfig.Users, newUser)

	if err := c.persistUsers(); err != nil {
		// Roll back in-memory change.
		c.ProxyConfig.Users = c.ProxyConfig.Users[:len(c.ProxyConfig.Users)-1]
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("[iptv-proxy] AUDIT: User created: %s", req.Username)
	ctx.JSON(http.StatusCreated, config.UserInfo{
		Username:  newUser.Username,
		Enabled:   newUser.Enabled,
		CreatedAt: newUser.CreatedAt,
	})
}

type updateUserRequest struct {
	Password    *string `json:"password"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
}

func (c *Config) apiUpdateUser(ctx *gin.Context) {
	username := ctx.Param("username")
	var req updateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.ProxyConfig.Users {
		if c.ProxyConfig.Users[i].Username == username {
			// Guard: don't allow disabling if it would leave zero enabled users.
			if req.Enabled != nil && !*req.Enabled && c.ProxyConfig.Users[i].Enabled {
				if c.ProxyConfig.EnabledUserCount() <= 1 {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable the last enabled user"})
					return
				}
			}
			if req.Password != nil && *req.Password != "" {
				c.ProxyConfig.Users[i].Password = *req.Password
			}
			if req.Description != nil {
				c.ProxyConfig.Users[i].Description = *req.Description
			}
			if req.Enabled != nil {
				c.ProxyConfig.Users[i].Enabled = *req.Enabled
			}
			if err := c.persistUsers(); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			log.Printf("[iptv-proxy] AUDIT: User updated: %s", username)
			ctx.JSON(http.StatusOK, config.UserInfo{
				Username:    c.ProxyConfig.Users[i].Username,
				Description: c.ProxyConfig.Users[i].Description,
				Enabled:     c.ProxyConfig.Users[i].Enabled,
				CreatedAt:   c.ProxyConfig.Users[i].CreatedAt,
			})
			return
		}
	}
	ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
}

func (c *Config) apiDeleteUser(ctx *gin.Context) {
	username := ctx.Param("username")

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.ProxyConfig.Users {
		if c.ProxyConfig.Users[i].Username == username {
			// Guard: don't allow deleting the last enabled user.
			if c.ProxyConfig.Users[i].Enabled && c.ProxyConfig.EnabledUserCount() <= 1 {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last enabled user"})
				return
			}
			c.ProxyConfig.Users = append(c.ProxyConfig.Users[:i], c.ProxyConfig.Users[i+1:]...)
			if err := c.persistUsers(); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			log.Printf("[iptv-proxy] AUDIT: User deleted: %s", username)
			ctx.Status(http.StatusNoContent)
			return
		}
	}
	ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
}

// apiUserWatch returns username + password for the Watch tab display.
// Security note: this endpoint is on the UI server (internal port) which has no auth,
// consistent with GET /api/settings which also returns the default user's password.
// The UI port should not be exposed publicly (documented in security considerations).
func (c *Config) apiUserWatch(ctx *gin.Context) {
	username := ctx.Param("username")

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, u := range c.ProxyConfig.Users {
		if u.Username == username {
			ctx.JSON(http.StatusOK, gin.H{"username": u.Username, "password": u.Password})
			return
		}
	}
	ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
}

// --- User access rules API ---

type accessRulesResponse struct {
	GroupAllowList   []string `json:"group_allow_list"`
	GroupBlockList   []string `json:"group_block_list"`
	ChannelAllowList []string `json:"channel_allow_list"`
	ChannelBlockList []string `json:"channel_block_list"`
}

func (c *Config) apiGetUserAccess(ctx *gin.Context) {
	username := ctx.Param("username")

	c.mu.RLock()
	defer c.mu.RUnlock()

	user := c.ProxyConfig.FindUser(username)
	if user == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	ctx.JSON(http.StatusOK, accessRulesResponse{
		GroupAllowList:   nonNilSlice(user.GroupAllowList),
		GroupBlockList:   nonNilSlice(user.GroupBlockList),
		ChannelAllowList: nonNilSlice(user.ChannelAllowList),
		ChannelBlockList: nonNilSlice(user.ChannelBlockList),
	})
}

func (c *Config) apiSetUserAccess(ctx *gin.Context) {
	username := ctx.Param("username")

	var req accessRulesResponse
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate all regex patterns before applying.
	if errs := ValidatePatterns(req.GroupAllowList, req.GroupBlockList, req.ChannelAllowList, req.ChannelBlockList); errs != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid regex patterns", "details": errs})
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.ProxyConfig.Users {
		if c.ProxyConfig.Users[i].Username == username {
			c.ProxyConfig.Users[i].GroupAllowList = req.GroupAllowList
			c.ProxyConfig.Users[i].GroupBlockList = req.GroupBlockList
			c.ProxyConfig.Users[i].ChannelAllowList = req.ChannelAllowList
			c.ProxyConfig.Users[i].ChannelBlockList = req.ChannelBlockList

			if err := c.persistUsers(); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			log.Printf("[iptv-proxy] AUDIT: Access rules updated for user: %s", username)
			ctx.JSON(http.StatusOK, accessRulesResponse{
				GroupAllowList:   nonNilSlice(c.ProxyConfig.Users[i].GroupAllowList),
				GroupBlockList:   nonNilSlice(c.ProxyConfig.Users[i].GroupBlockList),
				ChannelAllowList: nonNilSlice(c.ProxyConfig.Users[i].ChannelAllowList),
				ChannelBlockList: nonNilSlice(c.ProxyConfig.Users[i].ChannelBlockList),
			})
			return
		}
	}
	ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
}

// nonNilSlice ensures a nil slice is returned as empty array in JSON.
func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// persistUsers writes the current users list to settings.json.
// Caller must hold c.mu write lock.
func (c *Config) persistUsers() error {
	s, err := c.readSettingsFileStruct()
	if err != nil {
		return err
	}
	s.Users = c.ProxyConfig.Users
	return c.writeSettingsFile(&s)
}

