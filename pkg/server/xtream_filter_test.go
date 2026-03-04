package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
)

// newMockXtreamServer starts an httptest.Server implementing the minimal Xtream API.
// It responds to player_api.php with appropriate JSON based on the ?action= parameter.
func newMockXtreamServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/player_api.php", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		action := r.URL.Query().Get("action")
		switch action {
		case "": // authentication
			json.NewEncoder(w).Encode(map[string]interface{}{ //nolint: errcheck
				"user_info": map[string]interface{}{
					"username":               "u",
					"password":               "p",
					"status":                 "Active",
					"auth":                   1,
					"max_connections":        1,
					"active_cons":            0,
					"created_at":             0,
					"is_trial":               0,
					"exp_date":               nil,
					"allowed_output_formats": []string{"ts", "m3u8"},
				},
				"server_info": map[string]interface{}{
					"url":             "localhost",
					"port":            8000,
					"https_port":      8001,
					"server_protocol": "http",
					"rtmp_port":       8002,
					"timezone":        "UTC",
					"timestamp_now":   0,
					"time_now":        "2024-01-01 00:00:00",
				},
			})
		case "get_live_categories":
			json.NewEncoder(w).Encode([]map[string]interface{}{ //nolint: errcheck
				{"category_id": 1, "category_name": "Sports", "parent_id": 0},
				{"category_id": 2, "category_name": "News", "parent_id": 0},
				{"category_id": 3, "category_name": "Entertainment", "parent_id": 0},
			})
		case "get_live_streams":
			json.NewEncoder(w).Encode([]map[string]interface{}{ //nolint: errcheck
				{"stream_id": 1, "name": "Sport Channel 1", "category_id": 1, "category_name": "Sports", "num": 1, "rating": 0, "rating_5based": 0},
				{"stream_id": 2, "name": "Sport Channel 2", "category_id": 1, "category_name": "Sports", "num": 2, "rating": 0, "rating_5based": 0},
				{"stream_id": 3, "name": "CNN International", "category_id": 2, "category_name": "News", "num": 3, "rating": 0, "rating_5based": 0},
				{"stream_id": 4, "name": "BBC News", "category_id": 2, "category_name": "News", "num": 4, "rating": 0, "rating_5based": 0},
				{"stream_id": 5, "name": "Comedy Central", "category_id": 3, "category_name": "Entertainment", "num": 5, "rating": 0, "rating_5based": 0},
				{"stream_id": 6, "name": "Channel 4", "category_id": 3, "category_name": "Entertainment", "num": 6, "rating": 0, "rating_5based": 0},
			})
		case "get_vod_categories":
			json.NewEncoder(w).Encode([]map[string]interface{}{ //nolint: errcheck
				{"category_id": 10, "category_name": "Movies", "parent_id": 0},
				{"category_id": 11, "category_name": "Documentaries", "parent_id": 0},
			})
		case "get_vod_streams":
			json.NewEncoder(w).Encode([]map[string]interface{}{ //nolint: errcheck
				{"stream_id": 20, "name": "Action Movie 1", "category_id": 10, "category_name": "Movies", "num": 1, "rating": 0, "rating_5based": 0},
				{"stream_id": 21, "name": "Action Movie 2", "category_id": 10, "category_name": "Movies", "num": 2, "rating": 0, "rating_5based": 0},
				{"stream_id": 22, "name": "Nature Doc", "category_id": 11, "category_name": "Documentaries", "num": 3, "rating": 0, "rating_5based": 0},
				{"stream_id": 23, "name": "History Doc", "category_id": 11, "category_name": "Documentaries", "num": 4, "rating": 0, "rating_5based": 0},
			})
		case "get_series_categories":
			json.NewEncoder(w).Encode([]map[string]interface{}{ //nolint: errcheck
				{"category_id": 20, "category_name": "Drama", "parent_id": 0},
				{"category_id": 21, "category_name": "Comedy", "parent_id": 0},
			})
		case "get_series":
			json.NewEncoder(w).Encode([]map[string]interface{}{ //nolint: errcheck
				{"name": "Drama Series 1", "series_id": 1, "category_id": 20, "num": 1, "rating": 0, "rating_5based": 0},
				{"name": "Drama Series 2", "series_id": 2, "category_id": 20, "num": 2, "rating": 0, "rating_5based": 0},
				{"name": "Comedy Series 1", "series_id": 3, "category_id": 21, "num": 3, "rating": 0, "rating_5based": 0},
				{"name": "Comedy Series 2", "series_id": 4, "category_id": 21, "num": 4, "rating": 0, "rating_5based": 0},
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newXtreamTestConfig creates a Config pointing at the mock upstream server.
func newXtreamTestConfig(t *testing.T, upstreamURL string, opts ...func(*config.ProxyConfig)) *Config {
	t.Helper()
	cfg := &config.ProxyConfig{
		HostConfig:     &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		AdvertisedPort: 8080,
		User:           config.CredentialString("u"),
		Password:       config.CredentialString("p"),
		XtreamUser:     config.CredentialString("u"),
		XtreamPassword: config.CredentialString("p"),
		XtreamBaseURL:  upstreamURL,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Config{
		ProxyConfig:          cfg,
		endpointAntiColision: "x",
	}
}

// callXtreamAPIList calls xtreamPlayerAPI and decodes the JSON response as a list of objects.
func callXtreamAPIList(t *testing.T, c *Config, action string) []map[string]interface{} {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/player_api.php?action="+url.QueryEscape(action), nil)
	c.xtreamPlayerAPI(ctx, url.Values{"action": {action}})

	if w.Code != http.StatusOK {
		t.Fatalf("action=%q: HTTP %d: %s", action, w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("action=%q: decode response: %v\nbody=%s", action, err, w.Body.String())
	}
	return result
}

// categoryNames extracts the "category_name" field from a list response.
func categoryNames(items []map[string]interface{}) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if v, ok := item["category_name"].(string); ok {
			out = append(out, v)
		}
	}
	return out
}

// streamNames extracts the "name" field from a list response.
func streamNames(items []map[string]interface{}) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if v, ok := item["name"].(string); ok {
			out = append(out, v)
		}
	}
	return out
}

// streamCategoryNames extracts the "category_name" field from stream items.
func streamCategoryNames(items []map[string]interface{}) []string {
	return categoryNames(items) // same field
}

func withGroupExcl(groups ...string) func(*config.ProxyConfig) {
	return func(c *config.ProxyConfig) { c.GroupExclusions = groups }
}

func withGroupIncl(groups ...string) func(*config.ProxyConfig) {
	return func(c *config.ProxyConfig) { c.GroupInclusions = groups }
}

func withChannelExcl(channels ...string) func(*config.ProxyConfig) {
	return func(c *config.ProxyConfig) { c.ChannelExclusions = channels }
}

func withChannelIncl(channels ...string) func(*config.ProxyConfig) {
	return func(c *config.ProxyConfig) { c.ChannelInclusions = channels }
}

// --- Live category tests ---

func TestXtream_GetLiveCategories_GroupExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withGroupExcl(`^News$`))
	items := callXtreamAPIList(t, c, "get_live_categories")
	names := categoryNames(items)

	if len(names) != 2 {
		t.Errorf("want 2 categories (Sports, Entertainment), got %d: %v", len(names), names)
	}
	if containsName(names, "News") {
		t.Error("News category should be excluded")
	}
}

func TestXtream_GetLiveCategories_GroupInclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withGroupIncl(`^Sports$`))
	items := callXtreamAPIList(t, c, "get_live_categories")
	names := categoryNames(items)

	if len(names) != 1 {
		t.Errorf("want 1 category (Sports), got %d: %v", len(names), names)
	}
	if !containsName(names, "Sports") {
		t.Error("Sports should be present")
	}
}

// --- Live stream tests ---

func TestXtream_GetLiveStreams_GroupExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withGroupExcl(`^News$`))
	items := callXtreamAPIList(t, c, "get_live_streams")
	names := streamNames(items)

	if len(names) != 4 {
		t.Errorf("want 4 streams (Sports×2 + Entertainment×2), got %d: %v", len(names), names)
	}
	for _, excluded := range []string{"CNN International", "BBC News"} {
		if containsName(names, excluded) {
			t.Errorf("stream %q (News group) should be excluded", excluded)
		}
	}
}

func TestXtream_GetLiveStreams_ChannelExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withChannelExcl(`^CNN International$`))
	items := callXtreamAPIList(t, c, "get_live_streams")
	names := streamNames(items)

	if len(names) != 5 {
		t.Errorf("want 5 streams (6-1), got %d: %v", len(names), names)
	}
	if containsName(names, "CNN International") {
		t.Error("CNN International should be excluded")
	}
}

func TestXtream_GetLiveStreams_ChannelInclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withChannelIncl(`^Sport Channel 1$`, `^BBC News$`))
	items := callXtreamAPIList(t, c, "get_live_streams")
	names := streamNames(items)

	if len(names) != 2 {
		t.Errorf("want 2 streams, got %d: %v", len(names), names)
	}
	if !containsName(names, "Sport Channel 1") || !containsName(names, "BBC News") {
		t.Errorf("expected Sport Channel 1 and BBC News, got %v", names)
	}
}

// --- VOD category tests ---

func TestXtream_GetVodCategories_GroupExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withGroupExcl(`^Documentaries$`))
	items := callXtreamAPIList(t, c, "get_vod_categories")
	names := categoryNames(items)

	if len(names) != 1 {
		t.Errorf("want 1 category (Movies), got %d: %v", len(names), names)
	}
	if containsName(names, "Documentaries") {
		t.Error("Documentaries should be excluded")
	}
}

// --- VOD stream tests ---

func TestXtream_GetVodStreams_GroupExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withGroupExcl(`^Documentaries$`))
	items := callXtreamAPIList(t, c, "get_vod_streams")
	names := streamNames(items)

	if len(names) != 2 {
		t.Errorf("want 2 VOD streams (Movies only), got %d: %v", len(names), names)
	}
	for _, excluded := range []string{"Nature Doc", "History Doc"} {
		if containsName(names, excluded) {
			t.Errorf("VOD stream %q (Documentaries) should be excluded", excluded)
		}
	}
}

// --- Series category tests ---

func TestXtream_GetSeriesCategories_GroupExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withGroupExcl(`^Comedy$`))
	items := callXtreamAPIList(t, c, "get_series_categories")
	names := categoryNames(items)

	if len(names) != 1 {
		t.Errorf("want 1 series category (Drama), got %d: %v", len(names), names)
	}
	if containsName(names, "Comedy") {
		t.Error("Comedy category should be excluded")
	}
}

// --- Series tests ---

func TestXtream_GetSeries_ChannelExclusion(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL, withChannelExcl(`^Comedy Series 1$`, `^Comedy Series 2$`))
	items := callXtreamAPIList(t, c, "get_series")
	names := streamNames(items)

	if len(names) != 2 {
		t.Errorf("want 2 series (Drama only), got %d: %v", len(names), names)
	}
	for _, excluded := range []string{"Comedy Series 1", "Comedy Series 2"} {
		if containsName(names, excluded) {
			t.Errorf("series %q should be excluded", excluded)
		}
	}
}

// --- Replacement tests ---

func TestXtream_LiveCategories_GroupReplacement(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL)
	c.settings = &config.SettingsJSON{
		Replacements: &config.ReplacementsInSettings{
			Groups: []config.ReplacementRule{{Replace: `^News$`, With: "Breaking News"}},
		},
	}

	items := callXtreamAPIList(t, c, "get_live_categories")
	names := categoryNames(items)

	if containsName(names, "News") {
		t.Error(`"News" should have been renamed`)
	}
	if !containsName(names, "Breaking News") {
		t.Errorf(`expected "Breaking News" in %v`, names)
	}
	if len(names) != 3 {
		t.Errorf("want 3 categories, got %d: %v", len(names), names)
	}
}

func TestXtream_LiveStreams_ChannelReplacement(t *testing.T) {
	srv := newMockXtreamServer(t)
	c := newXtreamTestConfig(t, srv.URL)
	c.settings = &config.SettingsJSON{
		Replacements: &config.ReplacementsInSettings{
			Names: []config.ReplacementRule{{Replace: `^CNN International$`, With: "CNN"}},
		},
	}

	items := callXtreamAPIList(t, c, "get_live_streams")
	names := streamNames(items)

	if containsName(names, "CNN International") {
		t.Error(`"CNN International" should have been renamed to "CNN"`)
	}
	if !containsName(names, "CNN") {
		t.Errorf(`expected "CNN" in %v`, names)
	}
	if len(names) != 6 {
		t.Errorf("want 6 streams (no filter), got %d", len(names))
	}
}
