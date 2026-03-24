package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/alvarolobato/iptv-proxy/pkg/config"
	"github.com/alvarolobato/iptv-proxy/pkg/stats"
)

func setupStatsTestConfig() *Config {
	gin.SetMode(gin.TestMode)
	conf := &config.ProxyConfig{
		User:     "admin",
		Password: "adminpass",
		Users: []config.User{
			{Username: "admin", Password: "adminpass", Enabled: true},
			{Username: "alice", Password: "alice123", Enabled: true},
		},
	}
	return &Config{
		ProxyConfig:    conf,
		statsCollector: &stats.NoopCollector{},
	}
}

func setupStatsRouter(c *Config) *gin.Engine {
	r := gin.New()
	c.registerStatsRoutes(r)
	return r
}

// TestStatsEndpoints_UserParam_NoopCollector verifies all stats endpoints accept ?user= gracefully with noop collector.
func TestStatsEndpoints_UserParam_NoopCollector(t *testing.T) {
	c := setupStatsTestConfig()
	r := setupStatsRouter(c)

	endpoints := []struct {
		path string
		key  string // expected key in response
	}{
		{"/api/stats/channels?user=alice", "stats_enabled"},
		{"/api/stats/groups?user=alice", "stats_enabled"},
		{"/api/stats/heatmap?user=alice", "stats_enabled"},
		{"/api/stats/channel/123?user=alice", "stats_enabled"},
		{"/api/stats/history?user=alice", "stats_enabled"},
		{"/api/stats/users", "stats_enabled"},
		{"/api/stats/active", "stats_enabled"},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ep.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}
			if _, ok := resp[ep.key]; !ok {
				t.Errorf("expected %q key in response, got %v", ep.key, resp)
			}
			// With noop collector, stats_enabled should be false.
			if enabled, ok := resp["stats_enabled"].(bool); ok && enabled {
				t.Error("expected stats_enabled=false with noop collector")
			}
		})
	}
}

// TestStatsEndpoints_WithoutUserParam verifies endpoints work without ?user= (backward compatible).
func TestStatsEndpoints_WithoutUserParam(t *testing.T) {
	c := setupStatsTestConfig()
	r := setupStatsRouter(c)

	endpoints := []string{
		"/api/stats/channels",
		"/api/stats/groups",
		"/api/stats/heatmap",
		"/api/stats/channel/123",
		"/api/stats/active",
	}

	for _, path := range endpoints {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
