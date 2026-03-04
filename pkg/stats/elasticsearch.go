package stats

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

const (
	defaultIndexPrefix    = "iptv"
	rollupInterval        = 60 * time.Second
	writeTimeout          = 10 * time.Second
	maxPendingEvents      = 4096
)

// ESCollector writes session events and metrics to Elasticsearch.
type ESCollector struct {
	url        string
	apiKey     string
	username   string
	password   string
	prefix     string
	httpClient *http.Client

	// in-memory channel accumulators protected by mu
	mu      sync.Mutex
	accums  map[string]*channelAccumulator

	// active sessions: sessionID -> start time + event metadata
	activeSessions map[string]activeSession

	// async write channel
	events chan writeOp

	// pendingWrites tracks in-flight async writes for Flush() synchronization
	pendingWrites sync.WaitGroup

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type activeSession struct {
	startTime time.Time
	event     SessionEvent
}

type writeOp struct {
	index string
	body  interface{}
}

// ESConfig holds configuration for the Elasticsearch collector.
type ESConfig struct {
	URL         string
	APIKey      string
	Username    string
	Password    string
	IndexPrefix string
	// InsecureSkipVerify disables TLS verification (for testing only).
	InsecureSkipVerify bool
}

// NewESCollector creates an ESCollector, creates indices/templates if missing, and starts background workers.
func NewESCollector(cfg ESConfig) (*ESCollector, error) {
	if cfg.IndexPrefix == "" {
		cfg.IndexPrefix = defaultIndexPrefix
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, // nolint: gosec
		Proxy:           http.ProxyFromEnvironment,
	}
	c := &ESCollector{
		url:            strings.TrimRight(cfg.URL, "/"),
		apiKey:         cfg.APIKey,
		username:       cfg.Username,
		password:       cfg.Password,
		prefix:         cfg.IndexPrefix,
		httpClient:     &http.Client{Transport: transport, Timeout: writeTimeout},
		accums:         make(map[string]*channelAccumulator),
		activeSessions: make(map[string]activeSession),
		events:         make(chan writeOp, maxPendingEvents),
		stopCh:         make(chan struct{}),
	}

	if err := c.bootstrapIndices(); err != nil {
		return nil, fmt.Errorf("stats: bootstrap ES indices: %w", err)
	}

	// writer goroutine
	c.wg.Add(1)
	go c.writer()

	// TSDB rollup goroutine
	c.wg.Add(1)
	go c.rollupWorker()

	return c, nil
}

// Internal index names follow the metrics-* naming convention so Elasticsearch
// and Kibana recognise them as metrics data streams.
// Format: metrics-{prefix}.{dataset}
func (c *ESCollector) sessionsIndex() string      { return "metrics-" + c.prefix + ".sessions" }
func (c *ESCollector) channelMetricsIndex() string { return "metrics-" + c.prefix + ".channel_metrics" }
func (c *ESCollector) userHistoryIndex() string    { return "metrics-" + c.prefix + ".user_history" }

// Public index name accessors (used by stats_handlers.go).
func (c *ESCollector) SessionsIndexName() string      { return c.sessionsIndex() }
func (c *ESCollector) ChannelMetricsIndexName() string { return c.channelMetricsIndex() }
func (c *ESCollector) UserHistoryIndexName() string    { return c.userHistoryIndex() }

// ---- Collector interface ----

func (c *ESCollector) RecordSessionStart(ctx context.Context, event SessionEvent) string {
	sessionID := uuid.NewV4().String()
	event.Timestamp = time.Now().UTC()
	event.EventKind = EventSessionStart
	event.SessionID = sessionID

	c.mu.Lock()
	c.activeSessions[sessionID] = activeSession{startTime: event.Timestamp, event: event}
	// increment active count in accumulator
	acc := c.getOrCreateAccum(event)
	acc.sessionCount++
	acc.activeSessions++
	if event.UserName != "" {
		acc.uniqueUsers[event.UserName] = struct{}{}
	}
	c.mu.Unlock()

	c.enqueue(c.sessionsIndex(), event)
	return sessionID
}

func (c *ESCollector) RecordSessionEnd(ctx context.Context, sessionID string, event SessionEvent) {
	now := time.Now().UTC()
	event.Timestamp = now
	event.EventKind = EventSessionEnd
	event.SessionID = sessionID

	c.mu.Lock()
	active, ok := c.activeSessions[sessionID]
	if ok {
		delete(c.activeSessions, sessionID)
		if event.ChannelID == "" {
			event.ChannelID = active.event.ChannelID
		}
		if event.ChannelName == "" {
			event.ChannelName = active.event.ChannelName
		}
		if event.ChannelGroup == "" {
			event.ChannelGroup = active.event.ChannelGroup
		}
		if event.ChannelType == "" {
			event.ChannelType = active.event.ChannelType
		}
		if event.ChannelStreamID == "" {
			event.ChannelStreamID = active.event.ChannelStreamID
		}
		if event.UserName == "" {
			event.UserName = active.event.UserName
		}
		if event.ProxyMode == "" {
			event.ProxyMode = active.event.ProxyMode
		}
		if event.ClientIP == "" {
			event.ClientIP = active.event.ClientIP
		}
		if event.UserAgent == "" {
			event.UserAgent = active.event.UserAgent
		}
		dur := now.Sub(active.startTime)
		if event.DurationSeconds == 0 {
			event.DurationSeconds = int64(dur.Seconds())
		}
	}
	acc := c.getOrCreateAccum(event)
	if acc.activeSessions > 0 {
		acc.activeSessions--
	}
	acc.totalDurationSecs += event.DurationSeconds
	acc.totalBytes += event.BytesTransferred
	c.mu.Unlock()

	c.enqueue(c.sessionsIndex(), event)

	// also write to user history
	userSess := UserSession{
		Timestamp:        active.startTime,
		SessionID:        sessionID,
		EndTime:          now,
		DurationSeconds:  event.DurationSeconds,
		BytesTransferred: event.BytesTransferred,
		ChannelID:        event.ChannelID,
		ChannelName:      event.ChannelName,
		ChannelGroup:     event.ChannelGroup,
		ChannelType:      event.ChannelType,
		ChannelStreamID:  event.ChannelStreamID,
		UserName:         event.UserName,
		ClientIP:         event.ClientIP,
		UserAgent:        event.UserAgent,
		ProxyMode:        event.ProxyMode,
	}
	c.enqueue(c.userHistoryIndex(), userSess)
}

func (c *ESCollector) RecordSessionError(ctx context.Context, sessionID string, event SessionEvent) {
	event.Timestamp = time.Now().UTC()
	event.EventKind = EventSessionError
	event.SessionID = sessionID

	c.mu.Lock()
	if active, ok := c.activeSessions[sessionID]; ok {
		delete(c.activeSessions, sessionID)
		if event.ChannelID == "" {
			event.ChannelID = active.event.ChannelID
		}
		if event.ChannelName == "" {
			event.ChannelName = active.event.ChannelName
		}
		if event.ChannelGroup == "" {
			event.ChannelGroup = active.event.ChannelGroup
		}
		if event.ChannelType == "" {
			event.ChannelType = active.event.ChannelType
		}
		if event.UserName == "" {
			event.UserName = active.event.UserName
		}
		if event.ProxyMode == "" {
			event.ProxyMode = active.event.ProxyMode
		}
	}
	acc := c.getOrCreateAccum(event)
	acc.errorCount++
	if acc.activeSessions > 0 {
		acc.activeSessions--
	}
	c.mu.Unlock()

	c.enqueue(c.sessionsIndex(), event)
}

func (c *ESCollector) Close() error {
	close(c.stopCh)
	c.wg.Wait()
	return nil
}

// Flush waits for all pending write operations to complete (including in-flight HTTP writes).
// Useful in tests to ensure all events have been persisted before querying.
func (c *ESCollector) Flush() {
	c.pendingWrites.Wait()
}

// ---- Internal helpers ----

// getOrCreateAccum returns (or creates) the accumulator for this channel. Must be called with c.mu held.
func (c *ESCollector) getOrCreateAccum(event SessionEvent) *channelAccumulator {
	key := event.ChannelID
	if key == "" {
		key = event.ChannelName
	}
	if key == "" {
		key = "_unknown"
	}
	acc, ok := c.accums[key]
	if !ok {
		acc = &channelAccumulator{
			channelID:   event.ChannelID,
			channelName: event.ChannelName,
			channelGroup: event.ChannelGroup,
			channelType:  event.ChannelType,
			uniqueUsers:  make(map[string]struct{}),
		}
		c.accums[key] = acc
	}
	return acc
}

func (c *ESCollector) enqueue(index string, body interface{}) {
	c.pendingWrites.Add(1)
	select {
	case c.events <- writeOp{index: index, body: body}:
	default:
		c.pendingWrites.Done()
		log.Printf("[iptv-proxy] stats: event queue full, dropping event for index %s", index)
	}
}

// writer drains the events channel and writes docs to ES.
func (c *ESCollector) writer() {
	defer c.wg.Done()
	for {
		select {
		case op, ok := <-c.events:
			if !ok {
				return
			}
			c.writeOp(op)
		case <-c.stopCh:
			// drain remaining events
			for {
				select {
				case op := <-c.events:
					c.writeOp(op)
				default:
					return
				}
			}
		}
	}
}

func (c *ESCollector) writeOp(op writeOp) {
	defer c.pendingWrites.Done()
	if err := c.indexDoc(op.index, op.body); err != nil {
		log.Printf("[iptv-proxy] stats: index doc to %s: %v", op.index, err)
	}
}

// rollupWorker flushes accumulated per-channel metrics to the channel-metrics index every minute.
func (c *ESCollector) rollupWorker() {
	defer c.wg.Done()
	ticker := time.NewTicker(rollupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.flushRollup()
		case <-c.stopCh:
			c.flushRollup()
			return
		}
	}
}

func (c *ESCollector) flushRollup() {
	c.mu.Lock()
	if len(c.accums) == 0 {
		c.mu.Unlock()
		return
	}
	snapshot := make(map[string]*channelAccumulator, len(c.accums))
	for k, v := range c.accums {
		// deep copy
		cp := *v
		cp.uniqueUsers = make(map[string]struct{}, len(v.uniqueUsers))
		for u := range v.uniqueUsers {
			cp.uniqueUsers[u] = struct{}{}
		}
		snapshot[k] = &cp
		// reset counters that should not be cumulative (session_count, error_count, bytes, duration)
		// keep activeSessions as it's a live gauge
		v.sessionCount = 0
		v.totalDurationSecs = 0
		v.totalBytes = 0
		v.errorCount = 0
		v.uniqueUsers = make(map[string]struct{})
	}
	c.mu.Unlock()

	now := time.Now().UTC().Truncate(rollupInterval)
	for _, acc := range snapshot {
		// TSDB requires all dimension fields to be present and non-empty.
		channelID := acc.channelID
		if channelID == "" {
			channelID = acc.channelName
		}
		if channelID == "" {
			channelID = "_unknown"
		}
		channelName := acc.channelName
		if channelName == "" {
			channelName = channelID
		}
		channelGroup := acc.channelGroup
		if channelGroup == "" {
			channelGroup = "_unknown"
		}
		channelType := acc.channelType
		if channelType == "" {
			channelType = ChannelTypeLive
		}

		metric := ChannelMetric{
			Timestamp:         now,
			ChannelID:         channelID,
			ChannelName:       channelName,
			ChannelGroup:      channelGroup,
			ChannelType:       channelType,
			SessionCount:      acc.sessionCount,
			ActiveSessions:    acc.activeSessions,
			TotalDurationSecs: acc.totalDurationSecs,
			TotalBytes:        acc.totalBytes,
			ErrorCount:        acc.errorCount,
			UniqueUsers:       int64(len(acc.uniqueUsers)),
		}
		if err := c.indexDoc(c.channelMetricsIndex(), metric); err != nil {
			log.Printf("[iptv-proxy] stats: rollup to %s: %v", c.channelMetricsIndex(), err)
		}
	}
}

// indexDoc POSTs a document to ES using the index API.
func (c *ESCollector) indexDoc(index string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc", c.url, index)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ES returned %d: %s", resp.StatusCode, string(b))
	}
	io.Copy(io.Discard, resp.Body) // nolint: errcheck
	return nil
}

func (c *ESCollector) setAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	} else if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// esRequest executes a generic ES request and returns the response body.
func (c *ESCollector) esRequest(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewReader(data)
	}
	url := c.url + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

// ---- ES data stream + index template bootstrap ----
//
// All three indices use the metrics-* naming convention which ES reserves
// for data streams. Bootstrap therefore:
//   1. Creates a composable index template (with mappings + TSDB settings where applicable)
//   2. Creates the data stream itself via the data-stream API

func (c *ESCollector) bootstrapIndices() error {
	if err := c.ensureSessionsDataStream(); err != nil {
		return err
	}
	if err := c.ensureChannelMetricsDataStream(); err != nil {
		return err
	}
	if err := c.ensureUserHistoryDataStream(); err != nil {
		return err
	}
	return nil
}

// dataStreamExists returns true when the data stream already exists.
func (c *ESCollector) dataStreamExists(name string) bool {
	_, status, err := c.esRequest(http.MethodGet, "/_data_stream/"+name, nil)
	return err == nil && status == http.StatusOK
}

func (c *ESCollector) ensureSessionsDataStream() error {
	name := c.sessionsIndex()
	if c.dataStreamExists(name) {
		return nil
	}
	// 1. Index template
	tpl := map[string]interface{}{
		"index_patterns": []string{name},
		"data_stream":    map[string]interface{}{},
		"template": map[string]interface{}{
			"mappings": map[string]interface{}{
				"dynamic": "strict",
				"properties": map[string]interface{}{
					"@timestamp":        map[string]interface{}{"type": "date"},
					"event_kind":        map[string]interface{}{"type": "keyword"},
					"session_id":        map[string]interface{}{"type": "keyword"},
					"duration_seconds":  map[string]interface{}{"type": "long"},
					"bytes_transferred": map[string]interface{}{"type": "long"},
					"client_ip":         map[string]interface{}{"type": "keyword"},
					"user_agent":        map[string]interface{}{"type": "keyword"},
					"error_message":     map[string]interface{}{"type": "text"},
					"channel_id":        map[string]interface{}{"type": "keyword"},
					"channel_name":      map[string]interface{}{"type": "keyword"},
					"channel_group":     map[string]interface{}{"type": "keyword"},
					"channel_type":      map[string]interface{}{"type": "keyword"},
					"channel_stream_id": map[string]interface{}{"type": "keyword"},
					"user_name":         map[string]interface{}{"type": "keyword"},
					"proxy_mode":        map[string]interface{}{"type": "keyword"},
				},
			},
		},
		"priority": 200,
	}
	if body, status, err := c.esRequest(http.MethodPut, "/_index_template/"+name, tpl); err != nil || status >= 300 {
		return fmt.Errorf("create sessions index template: %d: %s", status, string(body))
	}
	// 2. Data stream
	body, status, err := c.esRequest(http.MethodPut, "/_data_stream/"+name, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("create sessions data stream: %d: %s", status, string(body))
	}
	log.Printf("[iptv-proxy] stats: created data stream %s", name)
	return nil
}

func (c *ESCollector) ensureChannelMetricsDataStream() error {
	name := c.channelMetricsIndex()
	if c.dataStreamExists(name) {
		return nil
	}
	// TSDB data stream: index.mode=time_series with typed dimensions and metrics.
	// Dimensions (time_series_dimension: true) uniquely identify a time series.
	// Metrics: gauge for point-in-time values, counter for monotonically increasing totals.
	tpl := map[string]interface{}{
		"index_patterns": []string{name},
		"data_stream":    map[string]interface{}{},
		"template": map[string]interface{}{
			"settings": map[string]interface{}{
				"index.mode":         "time_series",
				"index.routing_path": []string{"channel_id", "channel_type"},
			},
			"mappings": map[string]interface{}{
				"dynamic": "strict",
				"properties": map[string]interface{}{
					"@timestamp":    map[string]interface{}{"type": "date"},
					"channel_id":    map[string]interface{}{"type": "keyword", "time_series_dimension": true},
					"channel_name":  map[string]interface{}{"type": "keyword", "time_series_dimension": true},
					"channel_group": map[string]interface{}{"type": "keyword", "time_series_dimension": true},
					"channel_type":  map[string]interface{}{"type": "keyword", "time_series_dimension": true},
					// Gauges: point-in-time values
					"session_count":   map[string]interface{}{"type": "long", "time_series_metric": "gauge"},
					"active_sessions": map[string]interface{}{"type": "long", "time_series_metric": "gauge"},
					"unique_users":    map[string]interface{}{"type": "long", "time_series_metric": "gauge"},
					// Counters: monotonically increasing totals
					"total_duration_seconds": map[string]interface{}{"type": "long", "time_series_metric": "counter"},
					"bytes_transferred":      map[string]interface{}{"type": "long", "time_series_metric": "counter"},
					"error_count":            map[string]interface{}{"type": "long", "time_series_metric": "counter"},
				},
			},
		},
		"priority": 200,
	}
	if body, status, err := c.esRequest(http.MethodPut, "/_index_template/"+name, tpl); err != nil || status >= 300 {
		return fmt.Errorf("create channel_metrics index template: %d: %s", status, string(body))
	}
	body, status, err := c.esRequest(http.MethodPut, "/_data_stream/"+name, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("create channel_metrics data stream: %d: %s", status, string(body))
	}
	log.Printf("[iptv-proxy] stats: created TSDB data stream %s (dimensions: channel_id, channel_name, channel_group, channel_type)", name)
	return nil
}

func (c *ESCollector) ensureUserHistoryDataStream() error {
	name := c.userHistoryIndex()
	if c.dataStreamExists(name) {
		return nil
	}
	tpl := map[string]interface{}{
		"index_patterns": []string{name},
		"data_stream":    map[string]interface{}{},
		"template": map[string]interface{}{
			"mappings": map[string]interface{}{
				"dynamic": "strict",
				"properties": map[string]interface{}{
					"@timestamp":        map[string]interface{}{"type": "date"},
					"session_id":        map[string]interface{}{"type": "keyword"},
					"session_end_time":  map[string]interface{}{"type": "date"},
					"duration_seconds":  map[string]interface{}{"type": "long"},
					"bytes_transferred": map[string]interface{}{"type": "long"},
					"channel_id":        map[string]interface{}{"type": "keyword"},
					"channel_name":      map[string]interface{}{"type": "keyword"},
					"channel_group":     map[string]interface{}{"type": "keyword"},
					"channel_type":      map[string]interface{}{"type": "keyword"},
					"channel_stream_id": map[string]interface{}{"type": "keyword"},
					"user_name":         map[string]interface{}{"type": "keyword"},
					"client_ip":         map[string]interface{}{"type": "keyword"},
					"user_agent":        map[string]interface{}{"type": "keyword"},
					"proxy_mode":        map[string]interface{}{"type": "keyword"},
				},
			},
		},
		"priority": 200,
	}
	if body, status, err := c.esRequest(http.MethodPut, "/_index_template/"+name, tpl); err != nil || status >= 300 {
		return fmt.Errorf("create user_history index template: %d: %s", status, string(body))
	}
	body, status, err := c.esRequest(http.MethodPut, "/_data_stream/"+name, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("create user_history data stream: %d: %s", status, string(body))
	}
	log.Printf("[iptv-proxy] stats: created data stream %s", name)
	return nil
}

// ---- Stats query helpers (for API endpoints) ----

// QueryResult is a generic ES query result.
type QueryResult struct {
	Took int  `json:"took"`
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
	Aggregations json.RawMessage `json:"aggregations,omitempty"`
}

// SearchDocs runs an ES search and returns the raw result.
func (c *ESCollector) SearchDocs(index string, query map[string]interface{}) (*QueryResult, error) {
	body, status, err := c.esRequest(http.MethodGet, "/"+index+"/_search", query)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("ES search %s: %d: %s", index, status, string(body))
	}
	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ActiveSessionCount returns the current number of active streaming sessions.
func (c *ESCollector) ActiveSessionCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.activeSessions)
}
