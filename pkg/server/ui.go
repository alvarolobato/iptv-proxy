/*
 * Iptv-Proxy configuration UI and API.
 */

package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// runUIServer starts the configuration UI HTTP server on c.ProxyConfig.UIPort. Call from Serve() in a goroutine.
func (c *Config) runUIServer() {
	port := c.ProxyConfig.UIPort
	if port <= 0 {
		return
	}
	router := gin.Default()
	router.Use(gin.Recovery())

	// API: list unique group titles from playlist
	router.GET("/api/groups", func(ctx *gin.Context) {
		groups := c.groupsFromPlaylist()
		ctx.JSON(http.StatusOK, groups)
	})

	// API: list channels (name, group, tvg-id, tvg-name, tvg-logo) from playlist
	router.GET("/api/channels", func(ctx *gin.Context) {
		channels := c.channelsFromPlaylist()
		ctx.JSON(http.StatusOK, channels)
	})

	// API: get replacements.json
	router.GET("/api/replacements", func(ctx *gin.Context) {
		data, err := c.readReplacementsFile()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.Data(http.StatusOK, "application/json", data)
	})

	// API: save replacements.json
	router.PUT("/api/replacements", func(ctx *gin.Context) {
		var raw replacementsJSON
		if err := ctx.ShouldBindJSON(&raw); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := c.writeReplacementsFile(&raw); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(http.StatusOK)
	})

	// Serve static UI
	router.GET("/", func(ctx *gin.Context) { ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(uiHTML)) })
	router.GET("/index.html", func(ctx *gin.Context) { ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(uiHTML)) })

	log.Printf("[iptv-proxy] Configuration UI listening on :%d", port)
	if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Printf("[iptv-proxy] UI server error: %v", err)
	}
}

func (c *Config) groupsFromPlaylist() []string {
	seen := make(map[string]struct{})
	var out []string
	for _, track := range c.playlist.Tracks {
		for _, tag := range track.Tags {
			if tag.Name == "group-title" && tag.Value != "" {
				if _, ok := seen[tag.Value]; !ok {
					seen[tag.Value] = struct{}{}
					out = append(out, tag.Value)
				}
				break
			}
		}
	}
	return out
}

type channelRow struct {
	Name     string `json:"name"`
	Group    string `json:"group"`
	TvgID    string `json:"tvg_id"`
	TvgName  string `json:"tvg_name"`
	TvgLogo  string `json:"tvg_logo"`
}

func (c *Config) channelsFromPlaylist() []channelRow {
	out := make([]channelRow, 0, len(c.playlist.Tracks))
	for _, track := range c.playlist.Tracks {
		row := channelRow{Name: track.Name}
		for _, tag := range track.Tags {
			switch tag.Name {
			case "group-title":
				row.Group = tag.Value
			case "tvg-id":
				row.TvgID = tag.Value
			case "tvg-name":
				row.TvgName = tag.Value
			case "tvg-logo":
				row.TvgLogo = tag.Value
			}
		}
		out = append(out, row)
	}
	return out
}

func (c *Config) replacementsPath() string {
	if c.JSONFolder == "" {
		return ""
	}
	return filepath.Join(c.JSONFolder, "replacements.json")
}

func (c *Config) readReplacementsFile() ([]byte, error) {
	stub := replacementsJSON{Global: []Replacement{}, Names: []Replacement{}, Groups: []Replacement{}}
	path := c.replacementsPath()
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

func (c *Config) writeReplacementsFile(raw *replacementsJSON) error {
	path := c.replacementsPath()
	if path == "" {
		return fmt.Errorf("json-folder not set")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

