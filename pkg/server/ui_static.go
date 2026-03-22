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
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed uistatic/*
var uistaticFS embed.FS

func serveStaticUI(ctx *gin.Context) {
	if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead {
		ctx.Status(http.StatusNotFound)
		return
	}
	path := strings.TrimPrefix(ctx.Request.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	// Serve from embedded uistatic; path is relative to pkg/server, so use uistatic/ prefix
	embedPath := "uistatic/" + path
	data, err := fs.ReadFile(uistaticFS, embedPath)
	if err != nil {
		// SPA fallback: any unknown path serves index.html
		data, err = fs.ReadFile(uistaticFS, "uistatic/index.html")
		if err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", data)
		return
	}
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css"
	}
	ctx.Data(http.StatusOK, contentType, data)
}
