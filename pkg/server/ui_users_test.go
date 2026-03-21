package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/alvarolobato/iptv-proxy/pkg/config"
)

func setupUserTestConfig() *Config {
	gin.SetMode(gin.TestMode)
	return &Config{
		ProxyConfig: &config.ProxyConfig{
			User:     "admin",
			Password: "adminpass",
			Users: []config.User{
				{Username: "alice", Password: "alice123", Enabled: true, CreatedAt: "2026-03-21T10:00:00Z"},
			},
		},
	}
}

func setupUserRouter(c *Config) *gin.Engine {
	r := gin.New()
	r.GET("/api/users", c.apiListUsers)
	r.POST("/api/users", c.apiCreateUser)
	r.PUT("/api/users/:username", c.apiUpdateUser)
	r.DELETE("/api/users/:username", c.apiDeleteUser)
	r.GET("/api/users/:username/watch", c.apiUserWatch)
	return r
}

func TestAPIListUsers(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct{ Users []config.UserInfo }
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(resp.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Users))
	}
	if !resp.Users[0].IsDefault {
		t.Error("first user should be default")
	}
}

func TestAPICreateUser_Success(t *testing.T) {
	c := setupUserTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"username": "charlie", "password": "charlie789", "enabled": true})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(c.ProxyConfig.Users) != 2 {
		t.Errorf("expected 2 users in config, got %d", len(c.ProxyConfig.Users))
	}
}

func TestAPICreateUser_DuplicateUsername(t *testing.T) {
	c := setupUserTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"username": "alice", "password": "pass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPICreateUser_DuplicateDefault(t *testing.T) {
	c := setupUserTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"username": "admin", "password": "pass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPICreateUser_InvalidUsername(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"username": "user name", "password": "pass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPICreateUser_ReservedUsername(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"username": "live", "password": "pass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPICreateUser_EmptyPassword(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"username": "newuser", "password": ""})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPIUpdateUser_Password(t *testing.T) {
	c := setupUserTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupUserRouter(c)

	newPass := "newpass123"
	body, _ := json.Marshal(map[string]interface{}{"password": newPass})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/alice", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if c.ProxyConfig.Users[0].Password != newPass {
		t.Errorf("password not updated, got %q", c.ProxyConfig.Users[0].Password)
	}
}

func TestAPIUpdateUser_Disable(t *testing.T) {
	c := setupUserTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupUserRouter(c)

	enabled := false
	body, _ := json.Marshal(map[string]interface{}{"enabled": enabled})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/alice", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if c.ProxyConfig.Users[0].Enabled {
		t.Error("user should be disabled")
	}
}

func TestAPIUpdateUser_NotFound(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	body, _ := json.Marshal(map[string]interface{}{"password": "x"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/unknown", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPIDeleteUser_Success(t *testing.T) {
	c := setupUserTestConfig()
	c.ProxyConfig.DataFolder = t.TempDir()
	config.EnsureStubSettings(c.ProxyConfig.DataFolder)
	r := setupUserRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/users/alice", nil)
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if len(c.ProxyConfig.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(c.ProxyConfig.Users))
	}
}

func TestAPIDeleteUser_DefaultUser(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/users/admin", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPIDeleteUser_NotFound(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/users/unknown", nil)
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPIUserWatch(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/alice/watch", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp["username"] != "alice" || resp["password"] != "alice123" {
		t.Errorf("unexpected response: %v", resp)
	}
}

func TestAPIUserWatch_DefaultUser(t *testing.T) {
	c := setupUserTestConfig()
	r := setupUserRouter(c)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/admin/watch", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp["password"] != "adminpass" {
		t.Errorf("expected adminpass, got %q", resp["password"])
	}
}

func TestAuthenticate_MultiUser(t *testing.T) {
	c := setupUserTestConfig()
	r := gin.New()
	r.GET("/test", c.authenticate, func(ctx *gin.Context) {
		user, _ := ctx.Get("authenticated_user")
		ctx.JSON(200, gin.H{"user": user})
	})

	// Test with additional user.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?username=alice&password=alice123", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if resp["user"] != "alice" {
		t.Errorf("expected alice, got %q", resp["user"])
	}

	// Test with wrong password.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?username=alice&password=wrong", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// Test disabled user.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?username=bob&password=bob456", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401 for disabled user, got %d", w.Code)
	}
}

func TestAuthenticatePath(t *testing.T) {
	c := setupUserTestConfig()
	r := gin.New()
	r.GET("/stream/:user/:password/:id", c.authenticatePath, func(ctx *gin.Context) {
		user, _ := ctx.Get("authenticated_user")
		ctx.JSON(200, gin.H{"user": user})
	})

	// Success.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stream/alice/alice123/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Failure.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/stream/alice/wrong/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
