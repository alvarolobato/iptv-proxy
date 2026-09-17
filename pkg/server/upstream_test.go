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
	"compress/gzip"
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

// TestHLS_RetriesWholeRedirectFlow verifies an unreachable HLS stream server makes the proxy ask the provider
// again (which can redirect elsewhere) instead of retrying the same dead host.
func TestHLS_RetriesWholeRedirectFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dead, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	deadAddr := dead.Addr().String()
	dead.Close()

	manifest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Write([]byte("#EXTM3U\n/hlsr/tok/xu/xp/123/1/seg.ts\n")) // nolint: errcheck
	}))
	defer manifest.Close()

	var hits int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			http.Redirect(w, r, "http://"+deadAddr+"/hls/tok/123.m3u8", http.StatusFound)
			return
		}
		http.Redirect(w, r, manifest.URL+"/hls/tok/123.m3u8", http.StatusFound)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/play/xu/xp/123.m3u8")
	c.UpstreamConnectTimeout = time.Second
	c.UpstreamRetries = 2
	proxy := newUpstreamTestProxy(t, c)

	resp, err := http.Get(proxy.URL + "/play/u/p/123.m3u8")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", resp.StatusCode, body)
	}
	// Exact body and a clean read: rewriting credentials changes the length, so a copied Content-Length would
	// truncate the manifest and players would see an unexpected EOF.
	if readErr != nil {
		t.Errorf("reading manifest: %v (Content-Length %q)", readErr, resp.Header.Get("Content-Length"))
	}
	if want := "#EXTM3U\n/hlsr/tok/u/p/123/1/seg.ts\n"; string(body) != want {
		t.Errorf("manifest = %q, want %q", body, want)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("provider requests = %d, want 2 (retry re-asks the provider for a stream server)", got)
	}
}

// TestHLS_KeepsProviderErrorStatus verifies an error manifest is not reported to the player as 200.
func TestHLS_KeepsProviderErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manifest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer manifest.Close()

	var hits int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Redirect(w, r, manifest.URL+"/hls/tok/123.m3u8", http.StatusFound)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/play/xu/xp/123.m3u8")
	c.UpstreamRetries = 2
	proxy := newUpstreamTestProxy(t, c)

	resp, err := http.Get(proxy.URL + "/play/u/p/123.m3u8")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 passed through", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("provider requests = %d, want 1 (HTTP errors are not retried)", got)
	}
}

// TestUpstream_AttemptsBoundTotalWait verifies the configured attempt count and the per-attempt cap together
// bound the total wait: a silent server can't stretch it further.
func TestUpstream_AttemptsBoundTotalWait(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
	c.UpstreamConnectTimeout = 150 * time.Millisecond
	c.UpstreamRetries = 3
	proxy := newUpstreamTestProxy(t, c)

	start := time.Now()
	resp, err := http.Get(proxy.URL + "/live/u/p/123.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 4 {
		t.Errorf("provider requests = %d, want 4 (1 + 3 retries)", got)
	}
	if worst := c.upstreamWorstCase(); elapsed > worst+3*time.Second {
		t.Errorf("took %v, want within the worst case %v (+slack)", elapsed, worst)
	}
}

func TestRedactCredentials(t *testing.T) {
	in := `cannot reach server. Get "http://prov:80/player_api.php?username=xu&password=s3cret&action=get": dial tcp: i/o timeout`
	got := redactCredentials(in)
	if strings.Contains(got, "s3cret") || strings.Contains(got, "xu&") {
		t.Errorf("credentials not redacted: %q", got)
	}
	if !strings.Contains(got, "prov:80") || !strings.Contains(got, "i/o timeout") {
		t.Errorf("redaction removed useful context: %q", got)
	}
}

// TestUpstream_PerAttemptCapIsEnforced verifies a stream server that accepts the connection but delays its
// response headers past the per-attempt cap is cut off, retried and reported as 504 (not as a client disconnect).
func TestUpstream_PerAttemptCapIsEnforced(t *testing.T) {
	gin.SetMode(gin.TestMode)

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	defer slow.Close()

	var hits int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Redirect(w, r, slow.URL+r.URL.Path, http.StatusFound)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/live/xu/xp/123.ts")
	c.UpstreamConnectTimeout = 200 * time.Millisecond
	c.UpstreamRetries = 1
	proxy := newUpstreamTestProxy(t, c)

	start := time.Now()
	resp, err := http.Get(proxy.URL + "/live/u/p/123.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("provider requests = %d, want 2 (capped attempt is retried)", got)
	}
	if worst := c.upstreamWorstCase(); elapsed > worst+2*time.Second {
		t.Errorf("took %v, want within the worst case %v (+slack)", elapsed, worst)
	}
}

// TestHLS_DoesNotCacheDeadStreamServer verifies a stream server that never served the manifest is not remembered
// for later /hls chunk requests.
func TestHLS_DoesNotCacheDeadStreamServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dead, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	deadAddr := dead.Addr().String()
	dead.Close()

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://"+deadAddr+"/hls/tok/777.m3u8", http.StatusFound)
	}))
	defer provider.Close()

	hlsChannelsRedirectURLLock.Lock()
	delete(hlsChannelsRedirectURL, "777.m3u8")
	hlsChannelsRedirectURLLock.Unlock()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/play/xu/xp/777.m3u8")
	c.UpstreamConnectTimeout = 300 * time.Millisecond
	c.UpstreamRetries = 1
	proxy := newUpstreamTestProxy(t, c)

	start := time.Now()
	resp, err := http.Get(proxy.URL + "/play/u/p/777.m3u8")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("status = 200, want a gateway error")
	}
	// Each of a flow attempt's two requests is capped, so the attempt count bounds a failing flow too.
	if worst := c.upstreamFlowWorstCase(); elapsed > worst+2*time.Second {
		t.Errorf("failing flow took %v, want within the worst case %v (+slack)", elapsed, worst)
	}
	hlsChannelsRedirectURLLock.RLock()
	cached, ok := hlsChannelsRedirectURL["777.m3u8"]
	hlsChannelsRedirectURLLock.RUnlock()
	if ok {
		t.Errorf("dead stream server cached for later chunks: %v", cached.Host)
	}
}

// TestUpstream_CumulativeCapFires verifies the per-attempt cap ends an attempt whose individual hops each answer
// within the header timeout but together exceed the cap.
func TestUpstream_CumulativeCapFires(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var hits int32
	var chain *httptest.Server
	chain = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hop := r.URL.Query().Get("hop")
		time.Sleep(200 * time.Millisecond) // under the 300ms header timeout, four of them exceed the 600ms cap
		switch hop {
		case "", "1":
			atomic.AddInt32(&hits, 1)
			http.Redirect(w, r, chain.URL+"/x?hop=2", http.StatusFound)
		case "2":
			http.Redirect(w, r, chain.URL+"/x?hop=3", http.StatusFound)
		case "3":
			http.Redirect(w, r, chain.URL+"/x?hop=4", http.StatusFound)
		default:
			w.Header().Set("Content-Type", "video/mp2t")
			w.Write([]byte("ts")) // nolint: errcheck
		}
	}))
	defer chain.Close()

	c := newXtreamM3UConfig(t, chain.URL, chain.URL+"/live/xu/xp/123.ts")
	c.UpstreamConnectTimeout = 300 * time.Millisecond // header timeout 300ms, attempt cap 600ms
	c.UpstreamRetries = 1
	proxy := newUpstreamTestProxy(t, c)

	resp, err := http.Get(proxy.URL + "/live/u/p/123.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504 (cumulative cap)", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("provider requests = %d, want 2 (capped attempt retried)", got)
	}
}

// TestHLS_RewritesCredentialsForGzipClient verifies a client asking for gzip still gets a rewritten manifest:
// forwarding its Accept-Encoding would hand the provider's compressed manifest through with credentials intact.
func TestHLS_RewritesCredentialsForGzipClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const manifestBody = "#EXTM3U\n/hlsr/tok/xu/xp/123/1/seg.ts\n"
	manifest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			gz.Write([]byte(manifestBody)) // nolint: errcheck
			gz.Close()
			return
		}
		w.Write([]byte(manifestBody)) // nolint: errcheck
	}))
	defer manifest.Close()

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, manifest.URL+"/hls/tok/123.m3u8", http.StatusFound)
	}))
	defer provider.Close()

	c := newXtreamM3UConfig(t, provider.URL, provider.URL+"/play/xu/xp/123.m3u8")
	proxy := newUpstreamTestProxy(t, c)

	req, err := http.NewRequest(http.MethodGet, proxy.URL+"/play/u/p/123.m3u8", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Accept-Encoding", "gzip") // disables Go's transparent decompression on the client side
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if strings.Contains(string(body), "/xu/xp/") {
		t.Errorf("provider credentials reached the client: %q", body)
	}
	if !strings.Contains(string(body), "/hlsr/tok/u/p/123/1/seg.ts") {
		t.Errorf("manifest not rewritten: %q", body)
	}
}
