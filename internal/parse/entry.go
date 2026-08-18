package parse

import "time"

type Entry struct {
	Timestamp     *time.Time
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
	Bytes         int
	HasBytes      bool
	Duration      time.Duration
	HasDuration   bool
	RequestID     string
	HasRequestID  bool
	RemoteAddr    string
	HasRemoteAddr bool
}

var EntryKeys = map[string][]string{
	"Timestamp":  {"time", "timestamp", "ts", "@timestamp"},
	"Level":      {"level"},
	"Message":    {"msg", "message"},
	"Method":     {"method", "http_method", "verb"},
	"Path":       {"path", "url", "uri", "request_path"},
	"Status":     {"status", "status_code", "http_status", "code"},
	"Bytes":      {"bytes"},
	"Duration":   {"duration_ms", "duration", "elapsed", "latency_ms", "request_time"},
	"RequestID":  {"request_id"},
	"RemoteAddr": {"remote_addr"},
}
