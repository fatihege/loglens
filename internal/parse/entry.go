package parse

import "time"

type FieldMask uint16

const (
	FieldTimestamp FieldMask = 1 << iota
	FieldLevel
	FieldMessage
	FieldMethod
	FieldPath
	FieldStatus
	FieldBytes
	FieldDuration
	FieldRequestID
	FieldRemoteAddr
	FieldUserAgent
	FieldReferer
	FieldUser
	FieldProtocol
	FieldQuery
)

type Entry struct {
	Timestamp  time.Time
	Level      string
	Message    string
	Method     string
	Path       string
	Status     int
	Bytes      int64
	Duration   time.Duration
	RequestID  string
	RemoteAddr string
	UserAgent  string
	Referer    string
	User       string
	Protocol   string
	Query      string
	Mask       FieldMask
}

func (e *Entry) Mark(field FieldMask) {
	e.Mask |= field
}

func (e *Entry) Has(field FieldMask) bool {
	return e.Mask&field != 0
}

var fieldAliases = map[FieldMask][]string{
	FieldTimestamp:  {"time", "timestamp", "ts", "@timestamp"},
	FieldLevel:      {"level", "severity"},
	FieldMessage:    {"msg", "message"},
	FieldMethod:     {"method", "http_method", "verb"},
	FieldPath:       {"path", "url", "uri", "request_path"},
	FieldStatus:     {"status", "status_code", "http_status", "code"},
	FieldBytes:      {"bytes", "bytes_sent", "body_bytes_sent", "size", "resp_bytes", "content_length"},
	FieldDuration:   {"duration_ms", "latency_ms", "duration_us", "duration_ns", "duration", "elapsed", "latency", "request_time", "upstream_response_time"},
	FieldRequestID:  {"request_id", "trace_id", "correlation_id"},
	FieldRemoteAddr: {"remote_addr", "x_forwarded_for", "client_ip"},
	FieldUserAgent:  {"user_agent", "http_user_agent", "ua"},
	FieldReferer:    {"referer", "http_referer", "referrer"},
	FieldUser:       {"user"},
	FieldProtocol:   {"protocol"},
	FieldQuery:      {"query"},
}

var aliasUnits = map[string]time.Duration{
	"duration_ms":            time.Millisecond,
	"latency_ms":             time.Millisecond,
	"duration_us":            time.Microsecond,
	"duration_ns":            time.Nanosecond,
	"duration":               time.Millisecond,
	"elapsed":                time.Millisecond,
	"latency":                time.Millisecond,
	"request_time":           time.Second,
	"upstream_response_time": time.Second,
}
