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
	"net/url"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func (c *Config) routes(r *gin.RouterGroup) {
	r = r.Group(c.CustomEndpoint)

	//Xtream service endopoints
	if c.ProxyConfig.XtreamBaseURL != "" {
		c.xtreamRoutes(r)
		if c.servesXtreamM3U() {
			r.GET("/"+c.M3UFileName, c.authenticate, c.xtreamGetAuto)
			// XXX Private need: for external Android app
			r.POST("/"+c.M3UFileName, c.authenticate, c.xtreamGetAuto)

			return
		}
	}

	c.m3uRoutes(r)
}

// servesXtreamM3U reports whether the M3U source is the Xtream account's own get.php.
// In that mode per-track M3U routes are not registered, so stream URLs must use the
// Xtream form (credentials swapped in the upstream path) instead of the anti-collision form.
func (c *Config) servesXtreamM3U() bool {
	if c.ProxyConfig.XtreamBaseURL == "" || c.RemoteURL == nil ||
		c.XtreamUser.String() == "" || c.XtreamPassword.String() == "" {
		return false
	}
	base, err := url.Parse(c.XtreamBaseURL)
	if err != nil || !sameHost(base, c.RemoteURL) {
		return false
	}
	return path.Base(c.RemoteURL.Path) == "get.php" &&
		c.XtreamUser.String() == c.RemoteURL.Query().Get("username") &&
		c.XtreamPassword.String() == c.RemoteURL.Query().Get("password")
}

// sameHost compares hostnames and ports, treating a missing port as the scheme default.
func sameHost(a, b *url.URL) bool {
	return a.Hostname() != "" && strings.EqualFold(a.Hostname(), b.Hostname()) && urlPort(a) == urlPort(b)
}

func urlPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

func (c *Config) xtreamRoutes(r *gin.RouterGroup) {
	getphp := gin.HandlerFunc(c.xtreamGet)
	if c.XtreamGenerateApiGet {
		getphp = c.xtreamApiGet
	}
	r.GET("/get.php", c.authenticate, getphp)
	r.POST("/get.php", c.authenticate, getphp)
	r.GET("/apiget", c.authenticate, c.xtreamApiGet)
	r.GET("/player_api.php", c.authenticate, c.xtreamPlayerAPIGET)
	r.POST("/player_api.php", c.appAuthenticate, c.xtreamPlayerAPIPOST)
	r.GET("/xmltv.php", c.authenticate, c.xtreamXMLTV)

	// Stream routes with user/password in path.
	// Routes under fixed prefixes (/live, /movie, /series, /timeshift) can use :user/:password params safely.
	r.GET("/live/:user/:password/:id", c.authenticatePath, c.xtreamStreamLive)
	r.GET("/timeshift/:user/:password/:duration/:start/:id", c.authenticatePath, c.xtreamStreamTimeshift)
	r.GET("/movie/:user/:password/:id", c.authenticatePath, c.xtreamStreamMovie)
	r.GET("/series/:user/:password/:id", c.authenticatePath, c.xtreamStreamSeries)
	r.GET("/hlsr/:token/:user/:password/:channel/:hash/:chunk", c.authenticatePath, c.xtreamHlsrStream)
	// Single catch-all: Gin cannot have both /hls/:chunk and /hls/:token/:chunk (conflicting wildcards)
	r.GET("/hls/*path", c.xtreamHlsDispatch)

	// /play has two patterns: /play/:token/:type and /play/:user/:password/:id
	// These conflict in Gin's router, so we use a catch-all dispatcher.
	r.GET("/play/*path", c.xtreamPlayDispatch)

	// Root-level /:user/:password/:id conflicts with other root routes (e.g. /get.php).
	// Gin doesn't allow wildcards at the same level as literal segments in all cases,
	// but this particular pattern works because /get.php, /apiget, etc. are registered first
	// and Gin tries literal matches before wildcards.
	r.GET("/:user/:password/:id", c.authenticatePath, c.xtreamStreamHandler)
}

func (c *Config) m3uRoutes(r *gin.RouterGroup) {
	r.GET("/"+c.M3UFileName, c.authenticate, c.getM3U)
	// XXX Private need: for external Android app
	r.POST("/"+c.M3UFileName, c.authenticate, c.getM3U)

	for i, track := range c.playlist.Tracks {
		trackConfig := &Config{
			ProxyConfig:    c.ProxyConfig,
			track:          &c.playlist.Tracks[i],
			statsCollector: c.statsCollector,
		}

		if strings.HasSuffix(track.URI, ".m3u8") {
			r.GET(fmt.Sprintf("/%s/:user/:password/%d/:id", c.endpointAntiColision, i), c.authenticatePath, trackConfig.m3u8ReverseProxy)
		} else {
			r.GET(fmt.Sprintf("/%s/:user/:password/%d/%s", c.endpointAntiColision, i, path.Base(track.URI)), c.authenticatePath, trackConfig.reverseProxy)
		}
	}
}
