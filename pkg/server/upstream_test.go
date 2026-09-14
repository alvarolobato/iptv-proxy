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
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newUpstreamTestProxy serves c's Xtream routes and returns the proxy server.
func newUpstreamTestProxy(t *testing.T, c *Config) *httptest.Server {
	t.Helper()
	r := gin.New()
	c.xtreamRoutes(r.Group(""))
	proxy := httptest.NewServer(r)
	t.Cleanup(proxy.Close)
	return proxy
}

// TestStream_RetriesWhenRedirectTargetUnreachable verifies an unreachable stream server is retried by asking the
// provider again, which can redirect to a working server.
func TestStream_RetriesWhenRedirectTargetUnreachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dead, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	deadAddr := dead.Addr().String()
	dead.Close() // nothing listens here any more: connection refused

	edge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		w.Write([]byte("ts-data")) // nolint: errcheck
	}))
	defer edge.Close()

	var hits int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			http.Redirect(w, r, "http://"+deadAddr+r.URL.Path, http.StatusFound)
			return
		}
		http.Redirect(w, r, edge.URL+r.URL.Path, http.StatusFound)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/live/xu/xp/123.ts")
	c.UpstreamConnectTimeout = time.Second
	c.UpstreamRetries = 2
	proxy := newUpstreamTestProxy(t, c)

	resp, err := http.Get(proxy.URL + "/live/u/p/123.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK || string(body) != "ts-data" {
		t.Fatalf("status = %d body = %q, want 200 ts-data", resp.StatusCode, body)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("provider requests = %d, want 2 (retry asks the provider for a fresh redirect)", got)
	}
}

// TestStream_SilentUpstreamReturns504 verifies a stream server that accepts connections but never answers is
// retried and then reported as 504 quickly, instead of hanging.
func TestStream_SilentUpstreamReturns504(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The kernel completes TCP handshakes for the listen backlog, but nothing ever reads or answers.
	silent, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer silent.Close()

	var hits int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Redirect(w, r, "http://"+silent.Addr().String()+r.URL.Path, http.StatusFound)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/live/xu/xp/123.ts")
	c.UpstreamConnectTimeout = 100 * time.Millisecond
	c.UpstreamRetries = 1
	proxy := newUpstreamTestProxy(t, c)

	start := time.Now()
	resp, err := http.Get(proxy.URL + "/live/u/p/123.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("provider requests = %d, want 2 (1 retry)", got)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v, want a fast failure", elapsed)
	}
}

// TestStream_HTTPErrorIsNotRetried verifies provider HTTP error statuses are passed through without retrying.
func TestStream_HTTPErrorIsNotRetried(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var hits int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.NotFound(w, r)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/live/xu/xp/123.ts")
	c.UpstreamRetries = 2
	proxy := newUpstreamTestProxy(t, c)

	resp, err := http.Get(proxy.URL + "/live/u/p/123.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 passed through", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("provider requests = %d, want 1 (no retry on HTTP errors)", got)
	}
}

type fakeTimeoutErr struct{}

func (fakeTimeoutErr) Error() string   { return "i/o timeout" }
func (fakeTimeoutErr) Timeout() bool   { return true }
func (fakeTimeoutErr) Temporary() bool { return true }

func TestIsRetryableUpstreamError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"client canceled", &url.Error{Op: "Get", URL: "http://h/x", Err: context.Canceled}, false},
		{"timeout", &url.Error{Op: "Get", URL: "http://h/x", Err: fakeTimeoutErr{}}, true},
		{"dial error", &url.Error{Op: "Get", URL: "http://h/x", Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("no route to host")}}, true},
		{"connection refused", fmt.Errorf("wrapped: %w", syscall.ECONNREFUSED), true},
		{"connection reset", fmt.Errorf("wrapped: %w", syscall.ECONNRESET), true},
		{"other error", errors.New("malformed HTTP response"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableUpstreamError(tt.err); got != tt.want {
				t.Errorf("isRetryableUpstreamError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestUpstreamErrorReason_OmitsCredentials verifies logged upstream errors keep the host but drop the path and query.
func TestUpstreamErrorReason_OmitsCredentials(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "http://185.0.0.1:8000/play/user/s3cret/1.ts?token=abc",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: fakeTimeoutErr{}},
	}
	reason := upstreamErrorReason(err)
	if strings.Contains(reason, "s3cret") || strings.Contains(reason, "token") {
		t.Errorf("reason leaks credentials: %q", reason)
	}
	if !strings.Contains(reason, "185.0.0.1:8000") {
		t.Errorf("reason = %q, want the upstream host", reason)
	}
	if upstreamFailureStatus(err) != http.StatusGatewayTimeout {
		t.Errorf("upstreamFailureStatus(timeout) = %d, want 504", upstreamFailureStatus(err))
	}
	if upstreamFailureStatus(errors.New("refused")) != http.StatusBadGateway {
		t.Errorf("upstreamFailureStatus(other) want 502")
	}
}
