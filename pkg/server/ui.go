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
	"fmt"
	"log"
	"net/http"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
	"github.com/gin-gonic/gin"
)

// UIServer is responsible for exposing the configuration/settings UI.
// It runs on a separate port (uiPort) and only uses settings-related APIs.
type UIServer struct {
	uiPort int
	cfg    *Config
}

// NewUIServer creates a new UIServer bound to the given Config.
func NewUIServer(c *Config) *UIServer {
	return &UIServer{
		uiPort: c.UIPort,
		cfg:    c,
	}
}

// runUIServer starts the configuration UI HTTP server on the configured UI port.
func (c *Config) runUIServer() {
	if c.UIPort <= 0 {
		log.Printf("[iptv-proxy] UI server disabled (ui-port=%d)", c.UIPort)
		return
	}
	port := c.UIPort

	log.Printf("[iptv-proxy] Starting configuration UI on :%d", port)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// API: simple healthcheck for the UI frontend.
	router.GET("/api/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API: get groups/channels summary from the last-fetched playlist (for filtering UI, etc.).
	router.GET("/api/groups", c.handleGroups)
	router.GET("/api/channels", c.handleChannels)

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
		// If Xtream passthrough is enabled, the proxy does not need its own auth
		// credentials for clients. Hide them from the UI to avoid confusion.
		if c.ProxyConfig.XtreamPassthrough {
			file.User = ""
			file.Password = ""
			current.User = ""
			current.Password = ""
		}
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
		// Do not persist proxy user/password in passthrough mode.
		if c.ProxyConfig.XtreamPassthrough {
			s.User = ""
			s.Password = ""
		}
		if err := c.writeSettingsFile(&s); err != nil {
			log.Printf("[iptv-proxy] PUT /api/settings: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.applyLiveSettings(&s)
		ctx.Status(http.StatusOK)
	})

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
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Excluded   bool   `json:"excluded"`
	Replaced   bool   `json:"replaced"`
	Resolution string `json:"resolution"`
}
