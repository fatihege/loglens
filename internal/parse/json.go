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
}

func NewJSON() *JSON {
	return &JSON{
		fieldmap:  make(map[FieldMask]string),
		fieldErrs: make(map[FieldMask]int),
	}
}

func (j *JSON) Fieldmap() map[FieldMask]string {
	return maps.Clone(j.fieldmap)
}

func (j *JSON) FieldErrors() map[FieldMask]int {
	return maps.Clone(j.fieldErrs)
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
	valid := 0 // means the type is correct, even if the value is wrong

	timestampKey, ok := j.fieldmap[FieldTimestamp]
	if ok {
		r, rawOK := raw[timestampKey]
		if rawOK {
			timestamp, err := toTime(r)
			if errors.Is(err, ErrNull) {
				valid++
			} else if errors.Is(err, ErrIntOutOfRange) {
				valid++
				j.fieldErrs[FieldTimestamp]++
			} else if err != nil {
				j.fieldErrs[FieldTimestamp]++
			} else {
				entry.Timestamp = timestamp
				entry.Mark(FieldTimestamp)
				valid++
			}
		}
	}

	for _, f := range stringFields {
		key, ok := j.fieldmap[f.mask]
		if ok {
			r, rawOK := raw[key]
			if rawOK {
				v, err := toString(r)
				if errors.Is(err, ErrNull) {
					valid++
					continue
				} else if err != nil {
					j.fieldErrs[f.mask]++
					continue
				}

				*f.dest(&entry) = v
				entry.Mark(f.mask)
				valid++
			}
		}
	}

	statusKey, ok := j.fieldmap[FieldStatus]
	if ok {
		r, rawOK := raw[statusKey]
		if rawOK {
			status, err := toInt(r)
			if errors.Is(err, ErrNull) {
				valid++
			} else if errors.Is(err, ErrIntOutOfRange) || (err == nil && (status < 100 || status > 599)) {
				valid++
				j.fieldErrs[FieldStatus]++
			} else if err != nil {
				j.fieldErrs[FieldStatus]++
			} else {
				entry.Status = int(status)
				entry.Mark(FieldStatus)
				valid++
			}
		}
	}

	bytesKey, ok := j.fieldmap[FieldBytes]
	if ok {
		r, rawOK := raw[bytesKey]
		if rawOK {
			bytes, err := toInt(r)
			if errors.Is(err, ErrNull) {
				valid++
			} else if errors.Is(err, ErrIntOutOfRange) || (err == nil && bytes < 0) {
				valid++
				j.fieldErrs[FieldBytes]++
			} else if err != nil {
				j.fieldErrs[FieldBytes]++
			} else {
				entry.Bytes = bytes
				entry.Mark(FieldBytes)
				valid++
			}
		}
	}

	durationKey, ok := j.fieldmap[FieldDuration]
	if ok {
		r, rawOK := raw[durationKey]

		if rawOK {
			unit, ok := aliasUnits[durationKey]
			if !ok {
				unit = time.Millisecond
			}

			duration, err := toDuration(r, unit)
			if errors.Is(err, ErrNull) {
				valid++
			} else if errors.Is(err, ErrIntOutOfRange) || errors.Is(err, ErrNegativeDuration) {
				valid++
				j.fieldErrs[FieldDuration]++
			} else if err != nil {
				j.fieldErrs[FieldDuration]++
			} else {
				entry.Duration = duration
				entry.Mark(FieldDuration)
				valid++
			}
		}
	}

	if valid < 1 {
		return Entry{}, ErrMalformedLine
	}

	return entry, nil
}
