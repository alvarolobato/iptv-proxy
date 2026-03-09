package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestESCollector_Integration tests the ES collector against a real Elasticsearch cluster.
// Set ELASTICSEARCH_URL and ELASTICSEARCH_API_KEY env vars to run this test.
func TestESCollector_Integration(t *testing.T) {
	esURL := os.Getenv("ELASTICSEARCH_URL")
	esKey := os.Getenv("ELASTICSEARCH_API_KEY")
	if esURL == "" {
		t.Skip("ELASTICSEARCH_URL not set; skipping integration test")
	}
	// Strip surrounding quotes if present (env var is stored with quotes)
	if len(esURL) >= 2 && esURL[0] == '"' && esURL[len(esURL)-1] == '"' {
		esURL = esURL[1 : len(esURL)-1]
	}
	if len(esKey) >= 2 && esKey[0] == '"' && esKey[len(esKey)-1] == '"' {
		esKey = esKey[1 : len(esKey)-1]
	}

	cfg := ESConfig{
		URL:         esURL,
		APIKey:      esKey,
		IndexPrefix: "iptv-test",
	}

	t.Logf("Connecting to Elasticsearch at %s", esURL)
	collector, err := NewESCollector(cfg)
	if err != nil {
		t.Fatalf("NewESCollector: %v", err)
	}
	defer collector.Close()

	t.Log("ES collector initialized; indices bootstrapped")

	// Test session lifecycle
	startEvt := SessionEvent{
		ChannelID:    "test-channel-1",
		ChannelName:  "Test Channel 1",
		ChannelGroup: "Test Group",
		ChannelType:  ChannelTypeLive,
		ProxyMode:    ProxyModeM3U,
		ClientIP:     "127.0.0.1",
		UserAgent:    "test-agent/1.0",
		UserName:     "testuser",
	}

	sessionID := collector.RecordSessionStart(context.Background(), startEvt)
	if sessionID == "" {
		t.Error("RecordSessionStart: expected non-empty sessionID")
	}
	t.Logf("Session started: %s", sessionID)

	// Simulate some streaming time
	time.Sleep(100 * time.Millisecond)

	endEvt := SessionEvent{
		BytesTransferred: 1024 * 1024, // 1MB
	}
	collector.RecordSessionEnd(context.Background(), sessionID, endEvt)
	t.Log("Session ended")

	// Test error recording
	errSessID := collector.RecordSessionStart(context.Background(), startEvt)
	collector.RecordSessionError(context.Background(), errSessID, SessionEvent{
		ErrorMessage: "connection reset by peer",
	})
	t.Log("Error session recorded")

	// Synchronously flush all pending events to ES.
	collector.Flush()
	t.Log("Events flushed")

	// Verify active session count is 0
	if n := collector.ActiveSessionCount(); n != 0 {
		t.Errorf("expected 0 active sessions, got %d", n)
	}

	// Trigger rollup flush (synchronous direct call; channel_metrics TSDB index).
	collector.flushRollup()
	t.Log("Rollup flushed")

	// Query sessions index to verify docs were written
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"session_id": sessionID,
			},
		},
	}

	// Give ES time to index (refresh interval)
	time.Sleep(2 * time.Second)
	result, err := collector.SearchDocs(collector.sessionsIndex(), query)
	if err != nil {
		t.Fatalf("SearchDocs sessions: %v", err)
	}

	t.Logf("Sessions docs for session %s: %d", sessionID, result.Hits.Total.Value)
	if result.Hits.Total.Value < 2 {
		t.Errorf("expected at least 2 session docs (start + end), got %d", result.Hits.Total.Value)
	}

	t.Log("Integration test PASSED")
}

// mockESServer records documents indexed to the channel_metrics index for assertion.
type mockESServer struct {
	*httptest.Server
	channelMetricsDocs []ChannelMetric
	mu                 sync.Mutex
}

func newMockESServer(t *testing.T) *mockESServer {
	m := &mockESServer{}
	m.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// _data_stream/... exists check: return 404 so bootstrap creates templates/streams
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			// index template, data stream
			w.WriteHeader(http.StatusOK)
		case http.MethodPost:
			// _doc index
			if r.URL.Path != "" && len(r.URL.Path) > 1 {
				// path like /metrics-iptv-stats-test.channel_metrics/_doc
				if strings.Contains(r.URL.Path, "channel_metrics") {
					var doc ChannelMetric
					if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
						t.Logf("decode channel_metrics doc: %v", err)
					} else {
						m.mu.Lock()
						m.channelMetricsDocs = append(m.channelMetricsDocs, doc)
						m.mu.Unlock()
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"_index":"x","_id":"1"}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	return m
}

func TestChannelAndSessionStatsConsistency(t *testing.T) {
	mock := newMockESServer(t)
	defer mock.Close()

	cfg := ESConfig{
		URL:                mock.URL,
		IndexPrefix:        "iptv-stats-test",
		InsecureSkipVerify: true,
	}
	collector, err := NewESCollector(cfg)
	if err != nil {
		t.Fatalf("NewESCollector: %v", err)
	}
	defer collector.Close()

	ctx := context.Background()
	ch1 := SessionEvent{
		ChannelID:    "ch1",
		ChannelName:  "Channel 1",
		ChannelGroup: "Group1",
		ChannelType:  ChannelTypeLive,
		ProxyMode:    ProxyModeM3U,
	}

	// Simulate 3 sessions on ch1 that start and end with known durations.
	sessionIDs := make([]string, 3)
	durations := []int64{10, 20, 30}
	for i := 0; i < 3; i++ {
		sessionIDs[i] = collector.RecordSessionStart(ctx, ch1)
		if sessionIDs[i] == "" {
			t.Fatalf("RecordSessionStart %d: expected non-empty sessionID", i)
		}
	}
	for i := 0; i < 3; i++ {
		collector.RecordSessionEnd(ctx, sessionIDs[i], SessionEvent{
			DurationSeconds:  durations[i],
			BytesTransferred: int64(100 * (i + 1)),
		})
	}

	// Ghost end: session that never started. Must not be counted in channel metrics.
	collector.RecordSessionEnd(ctx, "unknown-session-id", SessionEvent{
		DurationSeconds:  100,
		BytesTransferred: 9999,
	})

	collector.Flush()
	collector.flushRollup()
	collector.Flush() // wait for rollup index writes

	mock.mu.Lock()
	docs := mock.channelMetricsDocs
	mock.mu.Unlock()

	var totalSessionCount, totalDurationSecs, totalBytes int64
	for _, d := range docs {
		totalSessionCount += d.SessionCount
		totalDurationSecs += d.TotalDurationSecs
		totalBytes += d.TotalBytes
	}

	// Channel stats must match session activity: 3 sessions started, total duration 10+20+30.
	if totalSessionCount != 3 {
		t.Errorf("channel_metrics session_count sum: want 3, got %d", totalSessionCount)
	}
	if totalDurationSecs != 60 {
		t.Errorf("channel_metrics total_duration_seconds sum: want 60, got %d (ghost end must not be counted)", totalDurationSecs)
	}
	// 100+200+300 = 600 bytes from the 3 sessions only
	if totalBytes != 600 {
		t.Errorf("channel_metrics bytes_transferred sum: want 600, got %d", totalBytes)
	}
}

func TestChannelAndSessionStatsConsistency_ErrorSession(t *testing.T) {
	mock := newMockESServer(t)
	defer mock.Close()

	cfg := ESConfig{
		URL:                mock.URL,
		IndexPrefix:        "iptv-stats-err-test",
		InsecureSkipVerify: true,
	}
	collector, err := NewESCollector(cfg)
	if err != nil {
		t.Fatalf("NewESCollector: %v", err)
	}
	defer collector.Close()

	ctx := context.Background()
	ch1 := SessionEvent{
		ChannelID:    "ch-err",
		ChannelName:  "Channel Err",
		ChannelGroup: "Group1",
		ChannelType:  ChannelTypeLive,
	}

	sid := collector.RecordSessionStart(ctx, ch1)
	collector.RecordSessionError(ctx, sid, SessionEvent{ErrorMessage: "fail"})

	// Ghost error: unknown session. Must not be counted in channel metrics.
	collector.RecordSessionError(ctx, "unknown-session", SessionEvent{ErrorMessage: "ghost"})

	collector.Flush()
	collector.flushRollup()
	collector.Flush()

	mock.mu.Lock()
	docs := mock.channelMetricsDocs
	mock.mu.Unlock()

	var totalErrorCount int64
	for _, d := range docs {
		totalErrorCount += d.ErrorCount
	}
	if totalErrorCount != 1 {
		t.Errorf("channel_metrics error_count sum: want 1 (only the tracked session), got %d", totalErrorCount)
	}
}
