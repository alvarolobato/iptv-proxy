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
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
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
	t.ResponseHeaderTimeout = connectTimeout
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

// upstreamConnectTimeout is the configured connect timeout, or the default when unset.
func (c *Config) upstreamConnectTimeout() time.Duration {
	if c.UpstreamConnectTimeout > 0 {
		return c.UpstreamConnectTimeout
	}
	return defaultUpstreamConnectTimeout
}

// upstreamAttemptTimeout hard-caps one attempt (dial plus response headers, across any provider redirect).
// It is enforced with a cancellable context that lives until the response body is closed, so a streaming body is
// never cut off mid-playback.
func (c *Config) upstreamAttemptTimeout() time.Duration {
	return 2 * c.upstreamConnectTimeout()
}

// upstreamBudget bounds the total time spent on a request and its retries: attempts x the per-attempt cap.
// Retries only start while the remaining budget still covers a full attempt, so the total can't drift past it.
func (c *Config) upstreamBudget() time.Duration {
	return time.Duration(c.upstreamAttempts()) * c.upstreamAttemptTimeout()
}

// cancelOnCloseBody keeps an attempt's context alive while the response body is streaming and releases it on Close.
type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// doAttempt sends one provider request with a hard cap on dial + response headers. The cap is released once the
// headers arrive, so the body can stream for as long as the client keeps reading.
func (c *Config) doAttempt(ctx *gin.Context, client *http.Client, newReq func(context.Context) (*http.Request, error)) (*http.Response, error) {
	attemptCtx, cancel := context.WithCancel(ctx.Request.Context())
	var capped atomic.Bool
	timer := time.AfterFunc(c.upstreamAttemptTimeout(), func() {
		capped.Store(true)
		cancel()
	})
	req, err := newReq(attemptCtx)
	if err != nil {
		timer.Stop()
		cancel()
		return nil, err
	}
	resp, err := client.Do(req)
	stopped := timer.Stop()
	if err != nil {
		cancel()
		// Cancelling for the cap looks like a client disconnect; report it as the timeout it is, so it is retried
		// and surfaces as 504 rather than 502.
		if capped.Load() {
			return nil, &url.Error{Op: "Get", URL: req.URL.String(), Err: attemptTimeoutError{}}
		}
		return nil, err
	}
	if !stopped && capped.Load() {
		// The cap fired just as the headers arrived.
		resp.Body.Close()
		cancel()
		return nil, &url.Error{Op: "Get", URL: req.URL.String(), Err: attemptTimeoutError{}}
	}
	resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

// attemptTimeoutError marks the per-attempt cap so it is classified as a retryable timeout.
type attemptTimeoutError struct{}

func (attemptTimeoutError) Error() string   { return "upstream attempt timeout" }
func (attemptTimeoutError) Timeout() bool   { return true }
func (attemptTimeoutError) Temporary() bool { return true }

// doUpstream sends a provider request, retrying when the upstream (typically the stream server the provider
// redirects to) can't be reached or doesn't answer. Each attempt builds a fresh request for the original URL, so
// the provider can hand out a different stream server. Nothing has been written to the client yet, so retrying is
// safe. Requests are bound to the client request's context and stop when the client goes away.
func (c *Config) doUpstream(ctx *gin.Context, client *http.Client, newReq func(context.Context) (*http.Request, error)) (*http.Response, error) {
	attempts := c.upstreamAttempts()
	budget := c.upstreamBudget()
	perAttempt := c.upstreamAttemptTimeout()
	start := time.Now()
	reqCtx := ctx.Request.Context()
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := c.doAttempt(ctx, client, newReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// Only start another attempt while the remaining budget still covers a whole one.
		if reqCtx.Err() != nil || !isRetryableUpstreamError(err) || attempt == attempts || budget-time.Since(start) < perAttempt {
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
	perAttempt := c.upstreamAttemptTimeout()
	start := time.Now()
	reqCtx := ctx.Request.Context()
	var lastErr error
	for i := 1; i <= attempts; i++ {
		handled, err := attempt()
		if handled {
			return
		}
		lastErr = err
		// Only start another attempt while the remaining budget still covers a whole one.
		if reqCtx.Err() != nil || !isRetryableUpstreamError(err) || i == attempts || budget-time.Since(start) < perAttempt {
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

// redactCredentials masks credential query parameters (username/password/token) in a message. It is enough for
// its call site (errors from the vendored Xtream client, which are query-style); path-style credentials are
// handled by upstreamErrorReason, which drops the path entirely. Errors from the vendored Xtream client embed the request URL
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
