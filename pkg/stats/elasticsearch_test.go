package stats

import (
	"context"
	"os"
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
