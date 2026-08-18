package parse

import "time"

type Entry struct {
	Timestamp     time.Time
	HasTimestamp  bool
	Level         string
	HasLevel      bool
	Message       string
	HasMessage    bool
	Method        string
	HasMethod     bool
	Path          string
	HasPath       bool
	Status        int
	HasStatus     bool
	Bytes         int64
	HasBytes      bool
	Duration      time.Duration
	HasDuration   bool
	RequestID     string
	HasRequestID  bool
	RemoteAddr    string
	HasRemoteAddr bool
}

var entryKeys = map[string][]string{
	"Timestamp":  {"time", "timestamp", "ts", "@timestamp"},
	"Level":      {"level"},
	"Message":    {"msg", "message"},
	"Method":     {"method", "http_method", "verb"},
	"Path":       {"path", "url", "uri", "request_path"},
	"Status":     {"status", "status_code", "http_status", "code"},
	"Bytes":      {"bytes", "bytes_sent", "body_bytes_sent", "size", "resp_bytes", "content_length"},
	"Duration":   {"duration_ms", "latency_ms", "duration_us", "duration_ns", "request_time", "upstream_response_time", "duration", "elapsed", "latency"},
	"RequestID":  {"request_id"},
	"RemoteAddr": {"remote_addr"},
}

var aliasUnits = map[string]time.Duration{
	"duration_ms":            time.Millisecond,
	"latency_ms":             time.Millisecond,
	"duration_us":            time.Microsecond,
	"duration_ns":            time.Nanosecond,
	"duration":               time.Millisecond,
	"request_time":           time.Second,
	"upstream_response_time": time.Second,
	"elapsed":                time.Second,
	"latency":                time.Second,
}
