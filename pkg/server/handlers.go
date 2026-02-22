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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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

	c.stream(ctx, rpURL)
}

func (c *Config) m3u8ReverseProxy(ctx *gin.Context) {
	id := ctx.Param("id")

	rpURL, err := url.Parse(strings.ReplaceAll(c.track.URI, path.Base(c.track.URI), id))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}

	c.stream(ctx, rpURL)
}

func (c *Config) stream(ctx *gin.Context, oriURL *url.URL) {
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	requestRangeHeader := ctx.Request.Header.Get("Range")
	forwardRange := requestRangeHeader != ""
	resp, err := c.forwardStreamRequest(ctx, client, oriURL, forwardRange)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
		return
	}
	logUpstreamStatus(oriURL, resp, forwardRange && requestRangeHeader != "", "initial")

	if forwardRange && shouldRetryWithoutRange(resp) {
		log.Printf("[iptv-proxy] Upstream %s returned 206 with zero/invalid payload (Range=%q); retrying without Range header", oriURL.String(), ctx.Request.Header.Get("Range"))
		resp.Body.Close()
		resp, err = c.forwardStreamRequest(ctx, client, oriURL, false)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err) // nolint: errcheck
			return
		}
		logUpstreamStatus(oriURL, resp, false, "retry-no-range")
	}
	defer resp.Body.Close()

	// If upstream returned an error status, log headers and a small portion
	// of the response body to aid debugging (don't consume large bodies).
	if resp.StatusCode >= 400 {
		// copy headers
		hdrs := make(map[string][]string)
		for k, v := range resp.Header {
			hdrs[k] = v
		}
		log.Printf("[iptv-proxy] Upstream %s returned status %d; headers=%v", oriURL.String(), resp.StatusCode, hdrs)

		// read up to 8KB of the body for logging
		lr := io.LimitReader(resp.Body, 8*1024)
		b, _ := ioutil.ReadAll(lr)
		if len(b) > 0 {
			log.Printf("[iptv-proxy] Upstream error body (truncated): %s", string(b))
		}
		// Reset body reader so we can still stream the full response to the client.
		// Note: we can't rewind the original resp.Body. Instead, for error statuses
		// we'll return the truncated body we read above and set the status.
		mergeHttpHeader(ctx.Writer.Header(), resp.Header)
		ctx.Status(resp.StatusCode)
		if len(b) > 0 {
			ctx.Writer.Write(b) // nolint: errcheck
		}
		return
	}

	mergeHttpHeader(ctx.Writer.Header(), resp.Header)
	ctx.Status(resp.StatusCode)
	ctx.Stream(func(w io.Writer) bool {
		io.Copy(w, resp.Body) // nolint: errcheck
		return false
	})
}

func (c *Config) forwardStreamRequest(ctx *gin.Context, client *http.Client, oriURL *url.URL, forwardRange bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx.Request.Context(), "GET", oriURL.String(), nil)
	if err != nil {
		return nil, err
	}

	mergeHttpHeader(req.Header, ctx.Request.Header)

	if ua := ctx.Request.UserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}

	// Do not leak client auth headers to the upstream provider.
	req.Header.Del("Authorization")
	req.Header.Del("Proxy-Authorization")

	if forwardRange && ctx.Request.Header.Get("Range") != "" {
		req.Header.Set("Range", ctx.Request.Header.Get("Range"))
	} else {
		req.Header.Del("Range")
		req.Header.Del("If-Range")
	}

	resp, err := client.Do(req)
	if err == nil {
		return resp, nil
	}

	if retryReq, rerr := c.retryRequestOnDNSError(ctx, req, err); rerr == nil && retryReq != nil {
		log.Printf("[iptv-proxy] DNS lookup failed for %s; retrying against base host %s", req.URL.String(), retryReq.URL.Host)
		return client.Do(retryReq)
	}

	return nil, err
}

func shouldRetryWithoutRange(resp *http.Response) bool {
	if resp == nil || resp.StatusCode != http.StatusPartialContent {
		return false
	}

	if resp.ContentLength == 0 {
		return true
	}

	if resp.ContentLength < 0 {
		if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
			if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
				return v == 0
			}
		}
		return false
	}

	return false
}

func logUpstreamStatus(oriURL *url.URL, resp *http.Response, usedRange bool, attempt string) {
	if resp == nil {
		return
	}

	contentRange := resp.Header.Get("Content-Range")
	contentLength := resp.ContentLength
	if contentLength < 0 {
		if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
			if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
				contentLength = v
			}
		}
	}

	log.Printf(
		"[iptv-proxy] Upstream %s (%s) status=%d usedRange=%t contentLength=%d contentRange=%q",
		oriURL.String(),
		attempt,
		resp.StatusCode,
		usedRange,
		contentLength,
		contentRange,
	)
}

func (c *Config) xtreamStream(ctx *gin.Context, oriURL *url.URL) {
	id := ctx.Param("id")
	if strings.HasSuffix(id, ".m3u8") {
		c.hlsXtreamStream(ctx, oriURL)
		return
	}

	c.stream(ctx, oriURL)
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

func (c *Config) retryRequestOnDNSError(ctx *gin.Context, originalReq *http.Request, err error) (*http.Request, error) {
	if c == nil || c.XtreamBaseURL == "" {
		return nil, nil
	}

	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return nil, nil
	}

	var dnsErr *net.DNSError
	if !errors.As(urlErr.Err, &dnsErr) || !dnsErr.IsNotFound {
		return nil, nil
	}

	baseURL, err := url.Parse(c.XtreamBaseURL)
	if err != nil {
		return nil, err
	}

	failingTarget, err := url.Parse(urlErr.URL)
	if err != nil {
		return nil, err
	}

	fallbackURL := *failingTarget
	fallbackReq := originalReq.Clone(ctx.Request.Context())
	fallbackReq.URL = &fallbackURL
	c.routeThroughBaseHost(fallbackReq, failingTarget.Host)

	return fallbackReq, nil
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
