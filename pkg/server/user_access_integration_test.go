package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/alvarolobato/iptv-proxy/pkg/config"
)

// --- Access rules API tests ---

func setupAccessTestConfig() *Config {
	gin.SetMode(gin.TestMode)
	conf := &config.ProxyConfig{
		User:     "admin",
		Password: "adminpass",
		Users: []config.User{
			{Username: "alice", Password: "alice123", Enabled: true},
			{Username: "bob", Password: "bob456", Enabled: true, GroupAllowList: []string{"^Sports$"}},
		},
	}
	conf.MigrateDefaultUser()
	return &Config{ProxyConfig: conf}
}

func setupAccessRouter(c *Config) *gin.Engine {
	r := gin.New()
	r.GET("/api/users", c.apiListUsers)
	r.GET("/api/users/:username/access", c.apiGetUserAccess)
	r.PUT("/api/users/:username/access", c.apiSetUserAccess)
	return r
}

func TestAPIGetUserAccess_NoRules(t *testing.T) {
	c := setupAccessTestConfig()
	r := setupAccessRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/alice/access", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string][]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp["group_allow_list"]) != 0 {
		t.Errorf("expected empty group_allow_list, got %v", resp["group_allow_list"])
	}
}

func TestAPIGetUserAccess_WithRules(t *testing.T) {
	c := setupAccessTestConfig()
	r := setupAccessRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/bob/access", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string][]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp["group_allow_list"]) != 1 || resp["group_allow_list"][0] != "^Sports$" {
		t.Errorf("expected [^Sports$], got %v", resp["group_allow_list"])
	}
}

func TestAPIGetUserAccess_NotFound(t *testing.T) {
	c := setupAccessTestConfig()
	r := setupAccessRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/unknown/access", nil)
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPISetUserAccess_Success(t *testing.T) {
	c := setupAccessTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupAccessRouter(c)

	body, _ := json.Marshal(map[string]interface{}{
		"group_allow_list":   []string{"^News$", "^Sports$"},
		"group_block_list":   []string{},
		"channel_allow_list": []string{},
		"channel_block_list": []string{"^Premium.*"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/alice/access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify in-memory state.
	alice := c.ProxyConfig.FindUser("alice")
	if len(alice.GroupAllowList) != 2 {
		t.Errorf("expected 2 group allow patterns, got %d", len(alice.GroupAllowList))
	}
	if len(alice.ChannelBlockList) != 1 || alice.ChannelBlockList[0] != "^Premium.*" {
		t.Errorf("expected channel block [^Premium.*], got %v", alice.ChannelBlockList)
	}
}

func TestAPISetUserAccess_InvalidRegex(t *testing.T) {
	c := setupAccessTestConfig()
	r := setupAccessRouter(c)

	body, _ := json.Marshal(map[string]interface{}{
		"group_allow_list":   []string{"[invalid"},
		"group_block_list":   []string{},
		"channel_allow_list": []string{},
		"channel_block_list": []string{},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/alice/access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPISetUserAccess_NotFound(t *testing.T) {
	c := setupAccessTestConfig()
	r := setupAccessRouter(c)

	body, _ := json.Marshal(map[string]interface{}{
		"group_allow_list":   []string{},
		"group_block_list":   []string{},
		"channel_allow_list": []string{},
		"channel_block_list": []string{},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/unknown/access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Users list shows has_access_rule ---

func TestAPIListUsers_HasAccessRule(t *testing.T) {
	c := setupAccessTestConfig()
	r := setupAccessRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Users []map[string]interface{} `json:"users"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	for _, u := range resp.Users {
		name := u["username"].(string)
		hasRule := u["has_access_rule"].(bool)
		switch name {
		case "bob":
			if !hasRule {
				t.Error("bob should have has_access_rule=true")
			}
		case "alice", "admin":
			if hasRule {
				t.Errorf("%s should have has_access_rule=false", name)
			}
		}
	}
}

// --- Per-user Xtream filtering integration test ---

func TestXtream_PerUserFilter_GroupAllow(t *testing.T) {
	srv := newMockXtreamServer(t)
	cfg := &config.ProxyConfig{
		HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort: 8080,
		User:           config.CredentialString("u"),
		Password:       config.CredentialString("p"),
		XtreamUser:     config.CredentialString("u"),
		XtreamPassword: config.CredentialString("p"),
		XtreamBaseURL:  srv.URL,
		Users: []config.User{
			{Username: "u", Password: "p", Enabled: true},
			{Username: "restricted", Password: "rpass", Enabled: true, GroupAllowList: []string{"^Sports$"}},
		},
	}
	c := &Config{
		ProxyConfig:          cfg,
		endpointAntiColision: "x",
	}

	gin.SetMode(gin.TestMode)

	// Call as "restricted" user — should only see Sports categories.
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/player_api.php?action=get_live_categories", nil)
	ctx.Set("authenticated_user", "restricted")
	c.xtreamPlayerAPI(ctx, url.Values{"action": {"get_live_categories"}})

	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var cats []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cats)
	names := categoryNames(cats)
	if len(names) != 1 || names[0] != "Sports" {
		t.Errorf("expected only Sports, got %v", names)
	}

	// Call as "restricted" user — should only see Sports streams.
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/player_api.php?action=get_live_streams", nil)
	ctx.Set("authenticated_user", "restricted")
	c.xtreamPlayerAPI(ctx, url.Values{"action": {"get_live_streams"}})

	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var streams []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &streams)
	streamN := streamNames(streams)
	if len(streamN) != 2 {
		t.Errorf("expected 2 Sports streams, got %d: %v", len(streamN), streamN)
	}
}

func TestXtream_PerUserFilter_ChannelBlock(t *testing.T) {
	srv := newMockXtreamServer(t)
	cfg := &config.ProxyConfig{
		HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort: 8080,
		User:           config.CredentialString("u"),
		Password:       config.CredentialString("p"),
		XtreamUser:     config.CredentialString("u"),
		XtreamPassword: config.CredentialString("p"),
		XtreamBaseURL:  srv.URL,
		Users: []config.User{
			{Username: "u", Password: "p", Enabled: true},
			{Username: "limited", Password: "lpass", Enabled: true, ChannelBlockList: []string{"^CNN International$", "^Comedy Central$"}},
		},
	}
	c := &Config{
		ProxyConfig:          cfg,
		endpointAntiColision: "x",
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/player_api.php?action=get_live_streams", nil)
	ctx.Set("authenticated_user", "limited")
	c.xtreamPlayerAPI(ctx, url.Values{"action": {"get_live_streams"}})

	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var streams []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &streams)
	names := streamNames(streams)
	if len(names) != 4 {
		t.Errorf("expected 4 streams (6-2), got %d: %v", len(names), names)
	}
	if containsName(names, "CNN International") || containsName(names, "Comedy Central") {
		t.Errorf("blocked channels should not appear: %v", names)
	}
}

func TestXtream_PerUserFilter_NoRulesFullAccess(t *testing.T) {
	srv := newMockXtreamServer(t)
	cfg := &config.ProxyConfig{
		HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort: 8080,
		User:           config.CredentialString("u"),
		Password:       config.CredentialString("p"),
		XtreamUser:     config.CredentialString("u"),
		XtreamPassword: config.CredentialString("p"),
		XtreamBaseURL:  srv.URL,
		Users: []config.User{
			{Username: "u", Password: "p", Enabled: true},
		},
	}
	c := &Config{
		ProxyConfig:          cfg,
		endpointAntiColision: "x",
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/player_api.php?action=get_live_streams", nil)
	ctx.Set("authenticated_user", "u")
	c.xtreamPlayerAPI(ctx, url.Values{"action": {"get_live_streams"}})

	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var streams []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &streams)
	if len(streams) != 6 {
		t.Errorf("expected 6 streams (no filter), got %d", len(streams))
	}
}

func TestXtream_PerUserFilter_CombinedWithGlobal(t *testing.T) {
	srv := newMockXtreamServer(t)
	cfg := &config.ProxyConfig{
		HostConfig:      &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort:  8080,
		User:            config.CredentialString("u"),
		Password:        config.CredentialString("p"),
		XtreamUser:      config.CredentialString("u"),
		XtreamPassword:  config.CredentialString("p"),
		XtreamBaseURL:   srv.URL,
		GroupExclusions: []string{"^Entertainment$"}, // Global filter removes Entertainment
		Users: []config.User{
			{Username: "u", Password: "p", Enabled: true},
			{Username: "restricted", Password: "rpass", Enabled: true, GroupAllowList: []string{"^Sports$", "^Entertainment$"}},
		},
	}
	c := &Config{
		ProxyConfig:          cfg,
		endpointAntiColision: "x",
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/player_api.php?action=get_live_streams", nil)
	ctx.Set("authenticated_user", "restricted")
	c.xtreamPlayerAPI(ctx, url.Values{"action": {"get_live_streams"}})

	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var streams []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &streams)
	names := streamNames(streams)
	// Global filter removes Entertainment, user allow list has Sports+Entertainment.
	// Result: only Sports (2 streams).
	if len(names) != 2 {
		t.Errorf("expected 2 Sports streams (Entertainment removed by global), got %d: %v", len(names), names)
	}
}
