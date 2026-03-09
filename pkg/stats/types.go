package stats

import "time"

// EventKind represents the type of session lifecycle event.
type EventKind string

const (
	EventSessionStart EventKind = "session_start"
	EventSessionEnd   EventKind = "session_end"
	EventSessionError EventKind = "session_error"
)

// ChannelType represents the type of media stream.
type ChannelType string

const (
	ChannelTypeLive   ChannelType = "live"
	ChannelTypeMovie  ChannelType = "movie"
	ChannelTypeSeries ChannelType = "series"
	ChannelTypeM3U    ChannelType = "m3u"
)

// ProxyMode represents how the stream is proxied.
type ProxyMode string

const (
	ProxyModeXtream ProxyMode = "xtream"
	ProxyModeM3U    ProxyMode = "m3u"
)

// SessionEvent is a single session lifecycle event written to the iptv-sessions index.
type SessionEvent struct {
	// Timestamp is the event time.
	Timestamp time.Time `json:"@timestamp"`
	// EventKind is session_start, session_end, or session_error.
	EventKind EventKind `json:"event_kind"`
	// SessionID is a UUID identifying this streaming connection.
	SessionID string `json:"session_id"`
	// DurationSeconds is set on session_end.
	DurationSeconds int64 `json:"duration_seconds,omitempty"`
	// BytesTransferred is set on session_end.
	BytesTransferred int64 `json:"bytes_transferred,omitempty"`
	// ClientIP is the remote address of the viewer.
	ClientIP string `json:"client_ip,omitempty"`
	// UserAgent is the HTTP User-Agent of the viewer.
	UserAgent string `json:"user_agent,omitempty"`
	// ErrorMessage is set on session_error.
	ErrorMessage string `json:"error_message,omitempty"`

	// Channel fields.
	ChannelID       string      `json:"channel_id,omitempty"`
	ChannelName     string      `json:"channel_name,omitempty"`
	ChannelGroup    string      `json:"channel_group,omitempty"`
	ChannelType     ChannelType `json:"channel_type,omitempty"`
	ChannelStreamID string      `json:"channel_stream_id,omitempty"`

	// User fields (multi-user ready).
	UserName string `json:"user_name,omitempty"`

	// Proxy metadata.
	ProxyMode ProxyMode `json:"proxy_mode,omitempty"`
}

// ChannelMetric is a TSDB rollup document written every minute to iptv-channel-metrics.
type ChannelMetric struct {
	Timestamp          time.Time   `json:"@timestamp"`
	ChannelID          string      `json:"channel_id"`
	ChannelName        string      `json:"channel_name"`
	ChannelGroup       string      `json:"channel_group"`
	ChannelType        ChannelType `json:"channel_type"`
	SessionCount       int64       `json:"session_count"`
	ActiveSessions     int64       `json:"active_sessions"`
	TotalDurationSecs  int64       `json:"total_duration_seconds"`
	TotalBytes         int64       `json:"bytes_transferred"`
	ErrorCount         int64       `json:"error_count"`
	UniqueUsers        int64       `json:"unique_users"`
}

// UserSession is a completed session document written to iptv-user-history.
type UserSession struct {
	Timestamp        time.Time   `json:"@timestamp"`
	SessionID        string      `json:"session_id"`
	EndTime          time.Time   `json:"session_end_time"`
	DurationSeconds  int64       `json:"duration_seconds"`
	BytesTransferred int64       `json:"bytes_transferred"`
	ChannelID        string      `json:"channel_id,omitempty"`
	ChannelName      string      `json:"channel_name,omitempty"`
	ChannelGroup     string      `json:"channel_group,omitempty"`
	ChannelType      ChannelType `json:"channel_type,omitempty"`
	ChannelStreamID  string      `json:"channel_stream_id,omitempty"`
	UserName         string      `json:"user_name,omitempty"`
	ClientIP         string      `json:"client_ip,omitempty"`
	UserAgent        string      `json:"user_agent,omitempty"`
	ProxyMode        ProxyMode   `json:"proxy_mode,omitempty"`
}

// channelAccumulator holds in-memory aggregates per channel for TSDB rollup.
type channelAccumulator struct {
	channelID   string
	channelName string
	channelGroup string
	channelType  ChannelType
	sessionCount int64
	activeSessions int64
	totalDurationSecs int64
	totalBytes   int64
	errorCount   int64
	uniqueUsers  map[string]struct{}
}
