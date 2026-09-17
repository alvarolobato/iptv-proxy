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
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultUpstreamConnectTimeout = 8 * time.Second
	upstreamRetryBackoff          = 300 * time.Millisecond
)

// upstreamTransports caches one transport per connect timeout so connections are pooled across requests.
var upstreamTransports sync.Map // time.Duration -> *http.Transport

// upstreamTransport returns a shared transport whose dial and response-header timeouts derive from connectTimeout.
func upstreamTransport(connectTimeout time.Duration) *http.Transport {
	if connectTimeout <= 0 {
		connectTimeout = defaultUpstreamConnectTimeout
	}
	if t, ok := upstreamTransports.Load(connectTimeout); ok {
		return t.(*http.Transport)
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: connectTimeout, KeepAlive: 30 * time.Second}).DialContext
	t.TLSHandshakeTimeout = connectTimeout
	// Once connected, stream servers answer within a second or two; a silent server is treated like an unreachable one.
	t.ResponseHeaderTimeout = 2 * connectTimeout
	actual, _ := upstreamTransports.LoadOrStore(connectTimeout, t)
	return actual.(*http.Transport)
}

// upstreamClient returns an HTTP client for provider requests. With followRedirects false, 3xx responses are returned as-is.
func (c *Config) upstreamClient(followRedirects bool) *http.Client {
	client := &http.Client{Transport: upstreamTransport(c.UpstreamConnectTimeout)}
	if !followRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return client
}

// upstreamAttempts is how many times a provider request may be tried (1 + configured retries).
func (c *Config) upstreamAttempts() int {
	if attempts := 1 + c.UpstreamRetries; attempts > 1 {
		return attempts
	}
	return 1
}

// upstreamBudget bounds the total time spent on a request and its retries, so a sequence of slow attempts can't
// take longer than a single un-bounded attempt used to. One attempt costs at most a dial plus a header wait on
// each of the provider and stream-server hops.
func (c *Config) upstreamBudget() time.Duration {
	connect := c.UpstreamConnectTimeout
	if connect <= 0 {
		connect = defaultUpstreamConnectTimeout
	}
	return time.Duration(c.upstreamAttempts()) * 2 * connect
}

// doUpstreamOnce sends a single provider request bound to the client request context, with no retry. Callers that
// retry a multi-step flow (HLS: provider redirect, then manifest) use this so a retry restarts the whole flow.
func (c *Config) doUpstreamOnce(ctx *gin.Context, client *http.Client, newReq func(context.Context) (*http.Request, error)) (*http.Response, error) {
	req, err := newReq(ctx.Request.Context())
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// doUpstream sends a provider request, retrying when the upstream (typically the stream server the provider
// redirects to) can't be reached or doesn't answer. Each attempt builds a fresh request for the original URL, so
// the provider can hand out a different stream server. Nothing has been written to the client yet, so retrying is
// safe. Requests are bound to the client request's context and stop when the client goes away.
func (c *Config) doUpstream(ctx *gin.Context, client *http.Client, newReq func(context.Context) (*http.Request, error)) (*http.Response, error) {
	attempts := c.upstreamAttempts()
	budget := c.upstreamBudget()
	start := time.Now()
	reqCtx := ctx.Request.Context()
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := newReq(reqCtx)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if reqCtx.Err() != nil || !isRetryableUpstreamError(err) || attempt == attempts || time.Since(start) >= budget {
			break
		}
		log.Printf("[iptv-proxy] upstream attempt %d/%d failed (%s); retrying", attempt, attempts, upstreamErrorReason(err))
		select {
		case <-reqCtx.Done():
			return nil, reqCtx.Err()
		case <-time.After(time.Duration(attempt) * upstreamRetryBackoff):
		}
	}
	return nil, lastErr
}

// retryUpstreamFlow runs a multi-step provider flow until an attempt writes the client response, fails with a
// non-retryable error, or the attempt/time budget runs out; then it aborts with the gateway status. attempt
// reports whether it wrote the response, and otherwise the error that decides whether the flow is retried.
func (c *Config) retryUpstreamFlow(ctx *gin.Context, attempt func() (bool, error)) {
	attempts := c.upstreamAttempts()
	budget := c.upstreamBudget()
	start := time.Now()
	reqCtx := ctx.Request.Context()
	var lastErr error
	for i := 1; i <= attempts; i++ {
		handled, err := attempt()
		if handled {
			return
		}
		lastErr = err
		if reqCtx.Err() != nil || !isRetryableUpstreamError(err) || i == attempts || time.Since(start) >= budget {
			break
		}
		log.Printf("[iptv-proxy] upstream flow attempt %d/%d failed (%s); retrying", i, attempts, upstreamErrorReason(err))
		select {
		case <-reqCtx.Done():
			abortUpstream(ctx, reqCtx.Err())
			return
		case <-time.After(time.Duration(i) * upstreamRetryBackoff):
		}
	}
	if lastErr == nil {
		lastErr = errors.New("upstream request failed")
	}
	abortUpstream(ctx, lastErr)
}

// credentialParamRE matches credential query parameters in free-form messages.
var credentialParamRE = regexp.MustCompile(`(?i)\b(username|password|token)=[^&\s"']+`)

// redactCredentials masks credentials in a message. Errors from the vendored Xtream client embed the request URL
// with username/password and are formatted with %v, so they can't be unwrapped and sanitised structurally.
func redactCredentials(s string) string {
	return credentialParamRE.ReplaceAllString(s, "$1=<redacted>")
}

// isRetryableUpstreamError reports whether a failed provider request is worth retrying: timeouts, failed dials,
// refused or reset connections. HTTP error statuses are responses, not errors, and are never retried.
func isRetryableUpstreamError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET)
}

// abortUpstream ends a request whose provider request failed: silently when the client went away, otherwise with
// 504/502 and a log reason that doesn't include the provider URL.
func abortUpstream(ctx *gin.Context, err error) {
	if ctx.Request.Context().Err() != nil {
		ctx.Abort()
		return
	}
	ctx.AbortWithError(upstreamFailureStatus(err), errors.New(upstreamErrorReason(err))) // nolint: errcheck
}

// upstreamFailureStatus maps a failed provider request to the gateway status returned to the client.
func upstreamFailureStatus(err error) int {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

// upstreamErrorReason describes a provider request error without the URL path or query, which carry credentials.
func upstreamErrorReason(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		host := ""
		if u, perr := url.Parse(ue.URL); perr == nil {
			host = u.Host
		}
		return fmt.Sprintf("%s %s: %v", ue.Op, host, ue.Err)
	}
	return err.Error()
}
