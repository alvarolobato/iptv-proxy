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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/alvarolobato/iptv-proxy/pkg/stats"
)

// countingReader wraps an io.Reader and counts bytes read.
type countingReader struct {
	r     io.Reader
	count int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	atomic.AddInt64(&cr.count, int64(n))
	return n, err
}

func (cr *countingReader) Bytes() int64 {
	return atomic.LoadInt64(&cr.count)
}

func (c *Config) getM3U(ctx *gin.Context) {
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, c.M3UFileName))
	ctx.Header("Content-Type", "application/octet-stream")

	ctx.File(c.proxyfiedM3UPath)
}

func (c *Config) reverseProxy(ctx *gin.Context) {
	rpURL, err := url.Parse(c.track.URI)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	chanInfo := stats.SessionEvent{}
	if c.track != nil {
		chanInfo.ChannelName = c.track.Name
		chanInfo.ChannelGroup = getGroupTitle(*c.track)
		chanInfo.ChannelID = c.track.URI
		chanInfo.ChannelType = stats.ChannelTypeM3U
		chanInfo.ProxyMode = stats.ProxyModeM3U
	}
	c.streamWithStats(ctx, rpURL, chanInfo)
}

func (c *Config) m3u8ReverseProxy(ctx *gin.Context) {
	id := ctx.Param("id")

	rpURL, err := url.Parse(strings.ReplaceAll(c.track.URI, path.Base(c.track.URI), id))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	chanInfo := stats.SessionEvent{}
	if c.track != nil {
		chanInfo.ChannelName = c.track.Name
		chanInfo.ChannelGroup = getGroupTitle(*c.track)
		chanInfo.ChannelID = c.track.URI
		chanInfo.ChannelType = stats.ChannelTypeM3U
		chanInfo.ProxyMode = stats.ProxyModeM3U
	}
	c.streamWithStats(ctx, rpURL, chanInfo)
}

// stream proxies a single HTTP response body to the client without stats tracking.
func (c *Config) stream(ctx *gin.Context, oriURL *url.URL) {
	c.streamWithStats(ctx, oriURL, stats.SessionEvent{})
}

// streamWithStats proxies a single HTTP response body to the client, recording session stats.
func (c *Config) streamWithStats(ctx *gin.Context, oriURL *url.URL, chanInfo stats.SessionEvent) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", oriURL.String(), nil)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	mergeHttpHeader(req.Header, ctx.Request.Header)
	if rangeH := ctx.Request.Header.Get("Range"); rangeH != "" {
		req.Header.Set("Range", rangeH)
	}

	resp, err := client.Do(req)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}
	defer resp.Body.Close()

	// Record session start.
	startEvt := chanInfo
	startEvt.ClientIP = ctx.ClientIP()
	startEvt.UserAgent = ctx.Request.UserAgent()
	startEvt.UserName = c.ProxyConfig.User.String()
	sessionID := c.statsCollector.RecordSessionStart(context.Background(), startEvt)

	cr := &countingReader{r: resp.Body}
	startTime := time.Now()

	mergeHttpHeader(ctx.Writer.Header(), resp.Header)
	ctx.Status(resp.StatusCode)
	ctx.Stream(func(w io.Writer) bool {
		io.Copy(w, cr) // nolint: errcheck
		return false
	})

	// Record session end after streaming completes.
	endEvt := stats.SessionEvent{
		DurationSeconds:  int64(time.Since(startTime).Seconds()),
		BytesTransferred: cr.Bytes(),
	}
	c.statsCollector.RecordSessionEnd(context.Background(), sessionID, endEvt)
}

func (c *Config) xtreamStream(ctx *gin.Context, oriURL *url.URL) {
	id := ctx.Param("id")
	if strings.HasSuffix(id, ".m3u8") {
		c.hlsXtreamStream(ctx, oriURL)
		return
	}

	c.stream(ctx, oriURL)
}

// xtreamStreamWithChannelInfo proxies an Xtream stream with channel stats context.
func (c *Config) xtreamStreamWithChannelInfo(ctx *gin.Context, oriURL *url.URL, streamID string, chanType stats.ChannelType) {
	id := ctx.Param("id")
	if strings.HasSuffix(id, ".m3u8") {
		c.hlsXtreamStream(ctx, oriURL)
		return
	}
	chanInfo := stats.SessionEvent{
		ChannelID:       streamID,
		ChannelStreamID: streamID,
		ChannelType:     chanType,
		ProxyMode:       stats.ProxyModeXtream,
	}
	c.streamWithStats(ctx, oriURL, chanInfo)
}

type values []string

func (vs values) contains(s string) bool {
	for _, v := range vs {
		if v == s {
			return true
		}
	}

	return false
}

func mergeHttpHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			if values(dst.Values(k)).contains(v) {
				continue
			}
			dst.Add(k, v)
		}
	}
}

// authRequest handle auth credentials
type authRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func (c *Config) authenticate(ctx *gin.Context) {
	var authReq authRequest
	if err := ctx.Bind(&authReq); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err) // nolint: errcheck
		return
	}
	if c.ProxyConfig.User.String() != authReq.Username || c.ProxyConfig.Password.String() != authReq.Password {
		ctx.AbortWithStatus(http.StatusUnauthorized)
	}
}

func (c *Config) appAuthenticate(ctx *gin.Context) {
	contents, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	q, err := url.ParseQuery(string(contents))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}
	if len(q["username"]) == 0 || len(q["password"]) == 0 {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("bad body url query parameters")) // nolint: errcheck
		return
	}
	log.Printf("[iptv-proxy] %v | %s |App Auth\n", time.Now().Format("2006/01/02 - 15:04:05"), ctx.ClientIP())
	if c.ProxyConfig.User.String() != q["username"][0] || c.ProxyConfig.Password.String() != q["password"][0] {
		ctx.AbortWithStatus(http.StatusUnauthorized)
	}

	ctx.Request.Body = ioutil.NopCloser(bytes.NewReader(contents))
}
