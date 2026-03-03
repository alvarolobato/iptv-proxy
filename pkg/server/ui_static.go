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
