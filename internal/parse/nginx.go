package parse

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"time"
)

var nginxFieldmap = map[FieldMask]string{
	FieldRemoteAddr: "$remote_addr",
	FieldUser:       "$remote_user",
	FieldTimestamp:  "$time_local",
	FieldMethod:     "$request",
	FieldPath:       "$request",
	FieldQuery:      "$request",
	FieldProtocol:   "$request",
	FieldStatus:     "$status",
	FieldBytes:      "$body_bytes_sent",
	FieldReferer:    "$http_referer",
	FieldUserAgent:  "$http_user_agent",
	FieldDuration:   "$request_time",
}

type Nginx struct {
	fieldErrs map[FieldMask]int
	fieldSeen map[FieldMask]int
}

func NewNginx() *Nginx {
	return &Nginx{
		fieldErrs: make(map[FieldMask]int),
		fieldSeen: make(map[FieldMask]int),
	}
}

func (n *Nginx) Fieldmap() map[FieldMask]string {
	return maps.Clone(nginxFieldmap)
}

func (n *Nginx) FieldErrors() map[FieldMask]int {
	return maps.Clone(n.fieldErrs)
}

func (n *Nginx) FieldSeen() map[FieldMask]int {
	return maps.Clone(n.fieldSeen)
}

func (n *Nginx) Parse(line []byte) (Entry, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return Entry{}, ErrMalformedLine
	}

	// used format: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $request_time
	var entry Entry
	i := 0

	remoteAddr, i, ok := parseBare(line, i)
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	if i, ok = consumeHSpace(line, i); !ok {
		return Entry{}, ErrMalformedLine
	}

	_, i, ok = parseBare(line, i)
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	if i, ok = consumeHSpace(line, i); !ok {
		return Entry{}, ErrMalformedLine
	}

	remoteUser, i, ok := parseBare(line, i)
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	if i, ok = consumeHSpace(line, i); !ok {
		return Entry{}, ErrMalformedLine
	}

	timeLocal, i, ok := parseDelimited(line, i, '[', ']')
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	if i, ok = consumeHSpace(line, i); !ok {
		return Entry{}, ErrMalformedLine
	}

	req, i, ok := parseDelimited(line, i, '"', '"')
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	if i, ok = consumeHSpace(line, i); !ok {
		return Entry{}, ErrMalformedLine
	}

	status, i, ok := parseBare(line, i)
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	if i, ok = consumeHSpace(line, i); !ok {
		return Entry{}, ErrMalformedLine
	}

	bytesSent, i, ok := parseBare(line, i)
	if !ok {
		return Entry{}, ErrMalformedLine
	}

	var referer, userAgent, requestTime []byte

	if i < len(line) {
		if i, ok = consumeHSpace(line, i); !ok {
			return Entry{}, ErrMalformedLine
		}

		referer, i, ok = parseDelimited(line, i, '"', '"')
		if !ok {
			return Entry{}, ErrMalformedLine
		}
	}

	if i < len(line) {
		if i, ok = consumeHSpace(line, i); !ok {
			return Entry{}, ErrMalformedLine
		}

		userAgent, i, ok = parseDelimited(line, i, '"', '"')
		if !ok {
			return Entry{}, ErrMalformedLine
		}
	}

	if i < len(line) {
		if i, ok = consumeHSpace(line, i); !ok {
			return Entry{}, ErrMalformedLine
		}

		requestTime, i, ok = parseBare(line, i)
		if !ok {
			return Entry{}, ErrMalformedLine
		}

		if i != len(line) {
			return Entry{}, ErrMalformedLine
		}
	}

	n.fieldSeen[FieldRemoteAddr]++
	if hasValue(remoteAddr) {
		entry.RemoteAddr = string(remoteAddr)
		entry.Mark(FieldRemoteAddr)
	}

	n.fieldSeen[FieldUser]++
	if hasValue(remoteUser) {
		entry.User = string(remoteUser)
		entry.Mark(FieldUser)
	}

	n.fieldSeen[FieldTimestamp]++
	if hasValue(timeLocal) {
		rawTime := json.RawMessage(slices.Concat([]byte("\""), timeLocal, []byte("\"")))
		if timestamp, err := toTime(rawTime); err != nil {
			n.fieldErrs[FieldTimestamp]++
		} else {
			entry.Timestamp = timestamp
			entry.Mark(FieldTimestamp)
		}
	}

	method, reqPath, query, protocol, ok := parseRequest(req)
	if !ok {
		for _, f := range []FieldMask{FieldMethod, FieldPath, FieldQuery, FieldProtocol} {
			n.fieldSeen[f]++
			n.fieldErrs[f]++
		}
	} else {
		if method != nil {
			n.fieldSeen[FieldMethod]++
			if hasValue(method) {
				entry.Method = string(method)
				entry.Mark(FieldMethod)
			}
		}

		if reqPath != nil {
			n.fieldSeen[FieldPath]++
			if hasValue(reqPath) {
				entry.Path = string(reqPath)
				entry.Mark(FieldPath)
			}
		}

		if query != nil {
			n.fieldSeen[FieldQuery]++
			if hasValue(query) {
				entry.Query = string(query)
				entry.Mark(FieldQuery)
			}
		}

		if protocol != nil {
			n.fieldSeen[FieldProtocol]++
			if hasValue(protocol) {
				entry.Protocol = string(protocol)
				entry.Mark(FieldProtocol)
			}
		}
	}

	n.fieldSeen[FieldStatus]++
	if hasValue(status) {
		rawStatus := json.RawMessage(status)

		if statusInt, err := toStatus(rawStatus); err != nil {
			n.fieldErrs[FieldStatus]++
		} else {
			entry.Status = statusInt
			entry.Mark(FieldStatus)
		}
	}

	n.fieldSeen[FieldBytes]++
	if hasValue(bytesSent) {
		rawBytes := json.RawMessage(bytesSent)

		if bytesInt, err := toBytes(rawBytes); err != nil {
			n.fieldErrs[FieldBytes]++
		} else {
			entry.Bytes = bytesInt
			entry.Mark(FieldBytes)
		}
	}

	if referer != nil {
		n.fieldSeen[FieldReferer]++
		if hasValue(referer) {
			entry.Referer = string(referer)
			entry.Mark(FieldReferer)
		}
	}

	if userAgent != nil {
		n.fieldSeen[FieldUserAgent]++
		if hasValue(userAgent) {
			entry.UserAgent = string(userAgent)
			entry.Mark(FieldUserAgent)
		}
	}

	if requestTime != nil {
		n.fieldSeen[FieldDuration]++
		if hasValue(requestTime) {
			rawDuration := json.RawMessage(requestTime)

			if duration, err := toDuration(rawDuration, time.Second); err != nil {
				n.fieldErrs[FieldDuration]++
			} else {
				entry.Duration = duration
				entry.Mark(FieldDuration)
			}
		}
	}

	return entry, nil
}

func consumeHSpace(line []byte, i int) (int, bool) {
	start := i
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}

	return i, start < i
}

func parseBare(line []byte, i int) ([]byte, int, bool) {
	if i >= len(line) {
		return nil, i, false
	}

	start := i
	for i < len(line) && line[i] != ' ' && line[i] != '\t' {
		i++
	}

	if i == start {
		return nil, i, false
	}

	return line[start:i], i, true
}

func parseDelimited(line []byte, i int, left, right byte) ([]byte, int, bool) {
	if i >= len(line) || line[i] != left {
		return nil, i, false
	}

	end := i + 1
	if right == '"' {
		for end < len(line) && line[end] != right {
			if line[end] == '\\' {
				end += 2
				continue
			}

			end++
		}

		if end >= len(line) {
			return nil, i, false
		}
	} else {
		end = bytes.IndexByte(line[i+1:], right)
		if end == -1 {
			return nil, i, false
		}
		end += i + 1
	}

	return line[i+1 : end], end + 1, true
}

func hasValue(tok []byte) bool {
	return len(tok) > 0 && string(tok) != "-"
}

func splitPath(b []byte) ([]byte, []byte) {
	if len(b) == 0 {
		return nil, nil
	}

	i := bytes.IndexByte(b, '?')

	if i == -1 {
		return b, nil
	} else {
		var p, q []byte
		if i > 0 {
			p = b[:i]
		}

		if i+1 < len(b) {
			q = b[i+1:]
		}

		return p, q
	}
}

func parseRequest(b []byte) (method, path, query, protocol []byte, ok bool) {
	if !hasValue(b) {
		return nil, nil, nil, nil, true
	}

	parts := bytes.Fields(b)

	switch {
	case len(parts) == 2:
		method = parts[0]
		path, query = splitPath(parts[1])
	case len(parts) == 3:
		method = parts[0]
		path, query = splitPath(parts[1])
		protocol = parts[2]
	case len(parts) > 3:
		method = parts[0]
		path, query = splitPath(bytes.Join(parts[1:len(parts)-1], []byte(" ")))
		protocol = parts[len(parts)-1]
	default:
		return nil, nil, nil, nil, false
	}

	return method, path, query, protocol, true
}
