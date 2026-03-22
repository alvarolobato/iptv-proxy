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
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/alvarolobato/iptv-proxy/pkg/config"
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

	// If multi-user: rewrite default credentials in M3U to the requesting user's credentials.
	authUser, exists := ctx.Get("authenticated_user")
	if !exists {
		ctx.File(c.proxyfiedM3UPath)
		return
	}
	userName := authUser.(string)
	defaultUser := url.PathEscape(c.pathAuthUser())
	defaultPass := url.PathEscape(c.pathAuthPassword())

	c.mu.RLock()
	user := c.ProxyConfig.FindUser(userName)
	c.mu.RUnlock()
	if user == nil {
		ctx.File(c.proxyfiedM3UPath)
		return
	}

	// Per-user access filter (Phase 2): if the user has access rules, generate M3U on-the-fly.
	filter := NewUserAccessFilter(user)
	if filter != nil {
		c.serveFilteredM3U(ctx, user, filter)
		return
	}

	// Fast path: no per-user rules. If requesting user is the default, no rewriting needed.
	if userName == c.pathAuthUser() {
		ctx.File(c.proxyfiedM3UPath)
		return
	}
	data, err := os.ReadFile(c.proxyfiedM3UPath)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}
	content := strings.ReplaceAll(string(data),
		"/"+defaultUser+"/"+defaultPass+"/",
		"/"+url.PathEscape(userName)+"/"+url.PathEscape(user.Password)+"/")
	ctx.Data(http.StatusOK, "application/octet-stream", []byte(content))
}

// serveFilteredM3U generates an M3U on-the-fly with per-user access rules applied.
func (c *Config) serveFilteredM3U(ctx *gin.Context, user *config.User, filter *UserAccessFilter) {
	data, err := os.ReadFile(c.proxyfiedM3UPath)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	defaultUser := url.PathEscape(c.pathAuthUser())
	defaultPass := url.PathEscape(c.pathAuthPassword())
	userEsc := url.PathEscape(user.Username)
	passEsc := url.PathEscape(user.Password)

	lines := strings.Split(string(data), "\n")
	var buf strings.Builder
	buf.Grow(len(data))

	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(line, "#EXTINF:") {
			// Parse group-title and channel name from the EXTINF line.
			group := extractM3UTagValue(line, "group-title")
			// Channel name is after the last comma.
			channelName := ""
			if idx := strings.LastIndex(line, ","); idx >= 0 {
				channelName = strings.TrimSpace(line[idx+1:])
			}
			if filter.MatchTrack(group, channelName) {
				// Rewrite credentials and include.
				rewritten := strings.ReplaceAll(line, "/"+defaultUser+"/"+defaultPass+"/", "/"+userEsc+"/"+passEsc+"/")
				buf.WriteString(rewritten)
				buf.WriteByte('\n')
				i++
				if i < len(lines) {
					rewritten = strings.ReplaceAll(lines[i], "/"+defaultUser+"/"+defaultPass+"/", "/"+userEsc+"/"+passEsc+"/")
					buf.WriteString(rewritten)
					buf.WriteByte('\n')
				}
			} else {
				// Skip this track: skip EXTINF line and the URL line.
				i++
			}
		} else {
			// Header or other lines: rewrite credentials.
			rewritten := strings.ReplaceAll(line, "/"+defaultUser+"/"+defaultPass+"/", "/"+userEsc+"/"+passEsc+"/")
			buf.WriteString(rewritten)
			buf.WriteByte('\n')
		}
		i++
	}
	ctx.Data(http.StatusOK, "application/octet-stream", []byte(strings.TrimRight(buf.String(), "\n")))
}

// extractM3UTagValue extracts a tag value like group-title="value" from an EXTINF line.
func extractM3UTagValue(line, tagName string) string {
	key := tagName + `="`
	idx := strings.Index(line, key)
	if idx < 0 {
		return ""
	}
	start := idx + len(key)
	end := strings.Index(line[start:], `"`)
	if end < 0 {
		return ""
	}
	return line[start : start+end]
}

func (c *Config) reverseProxy(ctx *gin.Context) {
	// Per-user access check for M3U track streaming.
	if c.track != nil {
		if userName, ok := ctx.Get("authenticated_user"); ok {
			c.mu.RLock()
			user := c.ProxyConfig.FindUser(userName.(string))
			c.mu.RUnlock()
			filter := NewUserAccessFilter(user)
			if filter != nil && !filter.MatchTrack(getGroupTitle(*c.track), c.track.Name) {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
	}

	rpURL, err := url.Parse(c.track.URI)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	chanInfo := stats.SessionEvent{}
	if c.track != nil {
		chanInfo.ChannelID = getTvgName(*c.track)
		chanInfo.ChannelName = c.track.Name
		chanInfo.ChannelGroup = getGroupTitle(*c.track)
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
		chanInfo.ChannelID = getTvgName(*c.track)
		chanInfo.ChannelName = c.track.Name
		chanInfo.ChannelGroup = getGroupTitle(*c.track)
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
	if u, ok := ctx.Get("authenticated_user"); ok {
		startEvt.UserName = u.(string)
	} else {
		startEvt.UserName = c.ProxyConfig.User.String()
	}
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
	// Use the stream numeric ID as fallback; try to resolve a human-readable name from the playlist.
	chanInfo := stats.SessionEvent{
		ChannelID:       streamID,
		ChannelStreamID: streamID,
		ChannelType:     chanType,
		ProxyMode:       stats.ProxyModeXtream,
	}
	if t := c.lookupTrackByStreamID(streamID); t != nil {
		chanInfo.ChannelID = getTvgName(*t)
		chanInfo.ChannelName = t.Name
		chanInfo.ChannelGroup = getGroupTitle(*t)
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

// checkUserStreamAccess verifies the authenticated user has access to the given stream.
// Returns true if access is allowed. Sends 403 and returns false if blocked.
func (c *Config) checkUserStreamAccess(ctx *gin.Context, streamID string) bool {
	userName, ok := ctx.Get("authenticated_user")
	if !ok {
		return true
	}
	c.mu.RLock()
	user := c.ProxyConfig.FindUser(userName.(string))
	c.mu.RUnlock()
	filter := NewUserAccessFilter(user)
	if filter == nil {
		return true
	}
	// Look up the track to get group/name for filtering.
	t := c.lookupTrackByStreamID(streamID)
	if t == nil {
		return true // unknown stream, allow (will fail upstream if truly invalid)
	}
	group := getGroupTitle(*t)
	name := t.Name
	if !filter.MatchTrack(group, name) {
		ctx.AbortWithStatus(http.StatusForbidden)
		return false
	}
	return true
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
	c.mu.RLock()
	matchedUser := c.ProxyConfig.ValidateCredentials(authReq.Username, authReq.Password)
	c.mu.RUnlock()
	if matchedUser == "" {
		log.Printf("[iptv-proxy] AUTH: Failed login attempt for user %q from %s", authReq.Username, ctx.ClientIP())
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.Set("authenticated_user", matchedUser)
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
	c.mu.RLock()
	matchedUser := c.ProxyConfig.ValidateCredentials(q["username"][0], q["password"][0])
	c.mu.RUnlock()
	if matchedUser == "" {
		log.Printf("[iptv-proxy] AUTH: Failed login attempt for user %q from %s", q["username"][0], ctx.ClientIP())
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.Set("authenticated_user", matchedUser)

	ctx.Request.Body = ioutil.NopCloser(bytes.NewReader(contents))
}

// authenticatePath validates user/password from URL path params.
func (c *Config) authenticatePath(ctx *gin.Context) {
	user := ctx.Param("user")
	pass := ctx.Param("password")
	c.mu.RLock()
	matched := c.ProxyConfig.ValidateCredentials(user, pass)
	c.mu.RUnlock()
	if matched == "" {
		log.Printf("[iptv-proxy] AUTH: Failed login attempt for user %q from %s", user, ctx.ClientIP())
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.Set("authenticated_user", matched)
}
