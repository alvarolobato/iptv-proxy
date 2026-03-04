// Package stats provides channel statistics tracking for iptv-proxy.
// It records session lifecycle events and channel usage metrics to Elasticsearch.
// When Elasticsearch is not configured, a no-op collector is used so there is
// no impact on proxy performance or correctness.
package stats

import "context"

// Collector is the interface for recording streaming session statistics.
// All methods are safe to call concurrently. Implementations must never
// block or panic; errors are logged internally.
type Collector interface {
	// RecordSessionStart records that a viewer started streaming a channel.
	// Returns a sessionID UUID to pass to subsequent calls.
	RecordSessionStart(ctx context.Context, event SessionEvent) string

	// RecordSessionEnd records that a stream ended normally.
	// sessionID must be the value returned by RecordSessionStart.
	RecordSessionEnd(ctx context.Context, sessionID string, event SessionEvent)

	// RecordSessionError records that a stream ended with an error.
	RecordSessionError(ctx context.Context, sessionID string, event SessionEvent)

	// Close flushes pending writes and shuts down background workers.
	Close() error
}

// NoopCollector silently discards all events. Used when ES is not configured.
type NoopCollector struct{}

func (n *NoopCollector) RecordSessionStart(_ context.Context, _ SessionEvent) string { return "" }
func (n *NoopCollector) RecordSessionEnd(_ context.Context, _ string, _ SessionEvent) {}
func (n *NoopCollector) RecordSessionError(_ context.Context, _ string, _ SessionEvent) {}
func (n *NoopCollector) Close() error                                                   { return nil }
