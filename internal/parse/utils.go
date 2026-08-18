package parse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

func toInt(raw json.RawMessage) (int, error) {
	raw = bytes.TrimSpace(raw)

	if len(raw) == 0 {
		return 0, ErrEmptyRawMessage
	}

	if bytes.Equal(raw, []byte("null")) {
		return 0, fmt.Errorf("%q: %w", raw, ErrNull)
	}

	if raw[0] == '"' {
		var d string
		err := json.Unmarshal(raw, &d)
		if err != nil {
			return 0, fmt.Errorf("%q: %w", raw, ErrNotInt)
		}

		i, err := strconv.Atoi(d)
		if err != nil {
			return 0, fmt.Errorf("%q: %w", d, ErrNotInt)
		}

		return i, nil
	}

	var d float64
	err := json.Unmarshal(raw, &d)
	if err != nil {
		return 0, fmt.Errorf("%q: %w", raw, ErrNotInt)
	}

	trunc, frac := math.Modf(d)

	if frac != 0 {
		return 0, fmt.Errorf("%f: %w", d, ErrNotInt)
	}

	return int(trunc), nil
}

func toString(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)

	if len(raw) == 0 {
		return "", ErrEmptyRawMessage
	}

	if bytes.Equal(raw, []byte("null")) {
		return "", fmt.Errorf("%q: %w", raw, ErrNull)
	}

	if raw[0] != '"' {
		return "", fmt.Errorf("%q: %w", raw, ErrNotString)
	}

	var d string
	err := json.Unmarshal(raw, &d)
	if err != nil {
		return "", fmt.Errorf("%q: %w", raw, err)
	}

	return d, nil
}

func toTime(raw json.RawMessage) (time.Time, error) {
	raw = bytes.TrimSpace(raw)

	if len(raw) == 0 {
		return time.Time{}, ErrEmptyRawMessage
	}

	if raw[0] != '"' {
		var timeFloat float64
		var t time.Time

		err := json.Unmarshal(raw, &timeFloat)
		if err != nil {
			return time.Time{}, fmt.Errorf("%q: %w", raw, err)
		}

		trunc, frac := math.Modf(timeFloat)
		whole := int64(trunc)
		absWhole := whole

		if absWhole < 0 {
			absWhole = -absWhole
		}

		switch {
		case absWhole > 1e17:
			t = time.Unix(0, whole)
		case absWhole > 1e14:
			extraNano := int64(frac * 1e3)
			t = time.UnixMicro(whole).Add(time.Duration(extraNano) * time.Nanosecond)
		case absWhole > 1e11:
			extraNano := int64(frac * 1e6)
			t = time.UnixMilli(whole).Add(time.Duration(extraNano) * time.Nanosecond)
		default:
			extraNano := int64(frac * 1e9)
			t = time.Unix(whole, extraNano)
		}

		return t, nil
	}

	var timeStr string
	err := json.Unmarshal(raw, &timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q: %w", raw, err)
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("%q: %w", timeStr, ErrParseTime)
}
