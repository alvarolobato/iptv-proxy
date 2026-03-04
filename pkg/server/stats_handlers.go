package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/alvarolobato/iptv-proxy/pkg/stats"
)

// registerStatsRoutes registers stats API endpoints on the given router.
// These endpoints are only meaningful when the ES collector is configured;
// with the no-op collector they return empty/zero results.
func (c *Config) registerStatsRoutes(router gin.IRouter) {
	router.GET("/api/stats/active", c.statsActive)
	router.GET("/api/stats/channels", c.statsTopChannels)
	router.GET("/api/stats/groups", c.statsTopGroups)
	router.GET("/api/stats/heatmap", c.statsHeatmap)
	router.GET("/api/stats/users", c.statsUsers)
	router.GET("/api/stats/channel/:id", c.statsChannel)
	router.GET("/api/stats/history", c.statsUserHistory)
}

// statsActive returns the count of currently active streaming sessions.
func (c *Config) statsActive(ctx *gin.Context) {
	type response struct {
		ActiveSessions int `json:"active_sessions"`
		StatsEnabled   bool `json:"stats_enabled"`
	}
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, response{ActiveSessions: 0, StatsEnabled: false})
		return
	}
	ctx.JSON(http.StatusOK, response{
		ActiveSessions: esColl.ActiveSessionCount(),
		StatsEnabled:   true,
	})
}

// statsTopChannels returns the top channels by total watch time (last 7 days by default).
func (c *Config) statsTopChannels(ctx *gin.Context) {
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"channels": []interface{}{}, "stats_enabled": false})
		return
	}

	days := 7
	if d := ctx.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	size := 20
	if s := ctx.Query("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			size = n
		}
	}

	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": from.Format(time.RFC3339),
				},
			},
		},
		"aggs": map[string]interface{}{
			"top_channels": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "channel_id",
					"size":  size,
					"order": map[string]interface{}{"total_duration": "desc"},
				},
				"aggs": map[string]interface{}{
					"total_duration": map[string]interface{}{
						"sum": map[string]interface{}{"field": "total_duration_seconds"},
					},
					"total_sessions": map[string]interface{}{
						"sum": map[string]interface{}{"field": "session_count"},
					},
					"total_bytes": map[string]interface{}{
						"sum": map[string]interface{}{"field": "bytes_transferred"},
					},
					"channel_name": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "channel_name",
							"size":  1,
						},
					},
					"channel_group": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "channel_group",
							"size":  1,
						},
					},
				},
			},
		},
	}

	result, err := esColl.SearchDocs(esColl.ChannelMetricsIndexName(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Cache-Control", "private, max-age=60")
	ctx.JSON(http.StatusOK, gin.H{
		"stats_enabled": true,
		"days":          days,
		"raw":           result,
	})
}

// statsTopGroups returns the top groups by total watch time.
func (c *Config) statsTopGroups(ctx *gin.Context) {
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"groups": []interface{}{}, "stats_enabled": false})
		return
	}

	days := 7
	if d := ctx.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	size := 50
	if s := ctx.Query("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			size = n
		}
	}

	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": from.Format(time.RFC3339),
				},
			},
		},
		"aggs": map[string]interface{}{
			"top_groups": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "channel_group",
					"size":  size,
					"order": map[string]interface{}{"total_duration": "desc"},
				},
				"aggs": map[string]interface{}{
					"total_duration": map[string]interface{}{
						"sum": map[string]interface{}{"field": "total_duration_seconds"},
					},
					"total_sessions": map[string]interface{}{
						"sum": map[string]interface{}{"field": "session_count"},
					},
					"unique_channels": map[string]interface{}{
						"cardinality": map[string]interface{}{"field": "channel_id"},
					},
				},
			},
		},
	}

	result, err := esColl.SearchDocs(esColl.ChannelMetricsIndexName(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Cache-Control", "private, max-age=60")
	ctx.JSON(http.StatusOK, gin.H{
		"stats_enabled": true,
		"days":          days,
		"raw":           result,
	})
}

// statsHeatmap returns hourly session event counts for the last N days (default 7).
// The result is a 24×7 matrix (hour × day_of_week) suitable for a heatmap visualization.
func (c *Config) statsHeatmap(ctx *gin.Context) {
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"buckets": []interface{}{}, "stats_enabled": false})
		return
	}

	days := 30
	if d := ctx.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{"event_kind": "session_start"},
					},
					map[string]interface{}{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{"gte": from.Format(time.RFC3339)},
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"by_hour": map[string]interface{}{
				"date_histogram": map[string]interface{}{
					"field":             "@timestamp",
					"calendar_interval": "hour",
					"min_doc_count":     0,
				},
			},
		},
	}

	result, err := esColl.SearchDocs(esColl.SessionsIndexName(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Cache-Control", "private, max-age=300")
	ctx.JSON(http.StatusOK, gin.H{
		"stats_enabled": true,
		"days":          days,
		"raw":           result,
	})
}

// statsUsers returns per-user session statistics.
func (c *Config) statsUsers(ctx *gin.Context) {
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"users": []interface{}{}, "stats_enabled": false})
		return
	}

	days := 30
	if d := ctx.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{"gte": from.Format(time.RFC3339)},
			},
		},
		"aggs": map[string]interface{}{
			"by_user": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "user_name",
					"size":  100,
				},
				"aggs": map[string]interface{}{
					"total_sessions": map[string]interface{}{
						"value_count": map[string]interface{}{"field": "session_id"},
					},
					"total_duration": map[string]interface{}{
						"sum": map[string]interface{}{"field": "duration_seconds"},
					},
					"total_bytes": map[string]interface{}{
						"sum": map[string]interface{}{"field": "bytes_transferred"},
					},
					"unique_channels": map[string]interface{}{
						"cardinality": map[string]interface{}{"field": "channel_id"},
					},
					"last_seen": map[string]interface{}{
						"max": map[string]interface{}{"field": "@timestamp"},
					},
				},
			},
		},
	}

	result, err := esColl.SearchDocs(esColl.UserHistoryIndexName(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Cache-Control", "private, max-age=60")
	ctx.JSON(http.StatusOK, gin.H{
		"stats_enabled": true,
		"days":          days,
		"raw":           result,
	})
}

// statsChannel returns per-channel historical metrics and recent sessions.
func (c *Config) statsChannel(ctx *gin.Context) {
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"channel": nil, "stats_enabled": false})
		return
	}

	channelID := ctx.Param("id")
	days := 30
	if d := ctx.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	// Get aggregate metrics
	metricsQuery := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{"term": map[string]interface{}{"channel_id": channelID}},
					map[string]interface{}{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{"gte": from.Format(time.RFC3339)},
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"total_duration": map[string]interface{}{
				"sum": map[string]interface{}{"field": "total_duration_seconds"},
			},
			"total_sessions": map[string]interface{}{
				"sum": map[string]interface{}{"field": "session_count"},
			},
			"total_bytes": map[string]interface{}{
				"sum": map[string]interface{}{"field": "bytes_transferred"},
			},
			"by_hour": map[string]interface{}{
				"date_histogram": map[string]interface{}{
					"field":             "@timestamp",
					"calendar_interval": "hour",
					"min_doc_count":     1,
				},
				"aggs": map[string]interface{}{
					"sessions": map[string]interface{}{
						"sum": map[string]interface{}{"field": "session_count"},
					},
				},
			},
		},
	}

	metricsResult, err := esColl.SearchDocs(esColl.ChannelMetricsIndexName(), metricsQuery)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get recent sessions
	sessionsQuery := map[string]interface{}{
		"size": 20,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{"term": map[string]interface{}{"channel_id": channelID}},
					map[string]interface{}{"term": map[string]interface{}{"event_kind": "session_end"}},
					map[string]interface{}{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{"gte": from.Format(time.RFC3339)},
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{"@timestamp": "desc"},
		},
	}

	sessionsResult, err := esColl.SearchDocs(esColl.SessionsIndexName(), sessionsQuery)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Cache-Control", "private, max-age=60")
	ctx.JSON(http.StatusOK, gin.H{
		"stats_enabled":   true,
		"channel_id":      channelID,
		"days":            days,
		"metrics":         metricsResult,
		"recent_sessions": sessionsResult,
	})
}

// statsUserHistory returns recent sessions for the authenticated user.
func (c *Config) statsUserHistory(ctx *gin.Context) {
	esColl, ok := c.statsCollector.(*stats.ESCollector)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"sessions": []interface{}{}, "stats_enabled": false})
		return
	}

	userName := ctx.Query("user")
	if userName == "" {
		userName = c.ProxyConfig.User.String()
	}
	days := 30
	if d := ctx.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	size := 50
	if s := ctx.Query("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			size = n
		}
	}

	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	query := map[string]interface{}{
		"size": size,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{"term": map[string]interface{}{"user_name": userName}},
					map[string]interface{}{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{"gte": from.Format(time.RFC3339)},
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{"@timestamp": "desc"},
		},
	}

	result, err := esColl.SearchDocs(esColl.UserHistoryIndexName(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Cache-Control", "private, max-age=30")
	ctx.JSON(http.StatusOK, gin.H{
		"stats_enabled": true,
		"user":          userName,
		"days":          days,
		"raw":           result,
	})
}
