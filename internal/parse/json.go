package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"
)

type JSON struct {
	fieldmap  map[FieldMask]string
	fieldErrs map[FieldMask]int
	fieldSeen map[FieldMask]int
}

func NewJSON() *JSON {
	return &JSON{
		fieldmap:  make(map[FieldMask]string),
		fieldErrs: make(map[FieldMask]int),
		fieldSeen: make(map[FieldMask]int),
	}
}

func (j *JSON) Fieldmap() map[FieldMask]string {
	return maps.Clone(j.fieldmap)
}

func (j *JSON) FieldErrors() map[FieldMask]int {
	return maps.Clone(j.fieldErrs)
}

func (j *JSON) FieldSeen() map[FieldMask]int {
	return maps.Clone(j.fieldSeen)
}

func (j *JSON) Parse(line []byte) (Entry, error) {
	var rawEntry map[string]json.RawMessage
	err := json.Unmarshal(line, &rawEntry)
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %v", ErrMalformedLine, err)
	}

	j.resolveKeys(rawEntry)

	return j.buildEntry(rawEntry)
}

func (j *JSON) resolveKeys(raw map[string]json.RawMessage) {
	for f, a := range fieldAliases {
		for _, k := range a {
			_, rawOK := raw[k]
			_, fmOK := j.fieldmap[f]
			if rawOK && !fmOK {
				j.fieldmap[f] = k
				break
			}
		}
	}
}

var stringFields = []struct {
	mask FieldMask
	dest func(*Entry) *string
}{
	{mask: FieldLevel, dest: func(e *Entry) *string { return &e.Level }},
	{mask: FieldMessage, dest: func(e *Entry) *string { return &e.Message }},
	{mask: FieldMethod, dest: func(e *Entry) *string { return &e.Method }},
	{mask: FieldPath, dest: func(e *Entry) *string { return &e.Path }},
	{mask: FieldRequestID, dest: func(e *Entry) *string { return &e.RequestID }},
	{mask: FieldRemoteAddr, dest: func(e *Entry) *string { return &e.RemoteAddr }},
	{mask: FieldUserAgent, dest: func(e *Entry) *string { return &e.UserAgent }},
	{mask: FieldReferer, dest: func(e *Entry) *string { return &e.Referer }},
	{mask: FieldUser, dest: func(e *Entry) *string { return &e.User }},
	{mask: FieldProtocol, dest: func(e *Entry) *string { return &e.Protocol }},
	{mask: FieldQuery, dest: func(e *Entry) *string { return &e.Query }},
}

func (j *JSON) buildEntry(raw map[string]json.RawMessage) (Entry, error) {
	var entry Entry
	sawAnyKey := false

	if assign(
		j,
		&entry,
		raw,
		FieldTimestamp,
		toTime,
		func(_ time.Time, err error) bool { return errors.Is(err, ErrIntOutOfRange) },
		&entry.Timestamp,
	) {
		sawAnyKey = true
	}

	for _, f := range stringFields {
		if assign(
			j,
			&entry,
			raw,
			f.mask,
			toString,
			func(v string, err error) bool { return false },
			f.dest(&entry),
		) {
			sawAnyKey = true
		}
	}

	if assign(
		j,
		&entry,
		raw,
		FieldStatus,
		func(r json.RawMessage) (int, error) { i, err := toInt(r); return int(i), err },
		func(v int, err error) bool {
			return errors.Is(err, ErrIntOutOfRange) || (err == nil && (v < 100 || v > 599))
		},
		&entry.Status,
	) {
		sawAnyKey = true
	}

	if assign(
		j,
		&entry,
		raw,
		FieldBytes,
		toInt,
		func(v int64, err error) bool {
			return errors.Is(err, ErrIntOutOfRange) || (err == nil && v < 0)
		},
		&entry.Bytes,
	) {
		sawAnyKey = true
	}

	durationKey, ok := j.fieldmap[FieldDuration]
	if ok {
		unit, ok := aliasUnits[durationKey]
		if !ok {
			unit = time.Millisecond
		}

		if assign(
			j,
			&entry,
			raw,
			FieldDuration,
			func(r json.RawMessage) (time.Duration, error) { return toDuration(r, unit) },
			func(_ time.Duration, err error) bool {
				return errors.Is(err, ErrIntOutOfRange) || errors.Is(err, ErrNegativeDuration)
			},
			&entry.Duration,
		) {
			sawAnyKey = true
		}
	}

	if !sawAnyKey {
		return Entry{}, ErrUnrecognized
	}

	return entry, nil
}

func assign[T any](
	j *JSON,
	e *Entry,
	raw map[string]json.RawMessage,
	mask FieldMask,
	conv func(json.RawMessage) (T, error),
	rejected func(T, error) bool,
	dest *T,
) (sawKey bool) {
	key, ok := j.fieldmap[mask]
	if !ok {
		return false
	}

	r, ok := raw[key]
	if !ok {
		return false
	}

	j.fieldSeen[mask]++

	v, err := conv(r)
	if errors.Is(err, ErrNull) {
		return true
	} else if err != nil || rejected(v, err) {
		j.fieldErrs[mask]++
		return true
	}

	*dest = v
	e.Mark(mask)
	return true
}
