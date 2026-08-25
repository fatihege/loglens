package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

func toInt(raw json.RawMessage) (int64, error) {
	raw, err := checkRaw(raw)
	if err != nil {
		return 0, err // returned error already has context
	}

	var n json.Number

	err = json.Unmarshal(raw, &n)
	if err != nil {
		return 0, fmt.Errorf("%q: %w", truncate(raw), ErrParseInt)
	}

	if i, err := n.Int64(); err == nil {
		return i, nil
	}

	f, err := n.Float64()
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, fmt.Errorf("%q: %w", truncate(raw), ErrIntOutOfRange)
		}
		return 0, fmt.Errorf("%q: %w", truncate(raw), ErrParseInt)
	}

	trunc, frac := math.Modf(f)

	if frac != 0 {
		return 0, fmt.Errorf("%v: %w", f, ErrParseInt)
	}

	if err := checkInt64Range(trunc); err != nil {
		return 0, err // returned error already has context
	}

	return int64(trunc), nil
}

func toString(raw json.RawMessage) (string, error) {
	raw, err := checkRaw(raw)
	if err != nil {
		return "", err
	}

	if raw[0] != '"' {
		return "", fmt.Errorf("%q: %w", truncate(raw), ErrParseString)
	}

	var s string
	err = json.Unmarshal(raw, &s)
	if err != nil {
		return "", fmt.Errorf("%q: %w", truncate(raw), ErrParseString)
	}

	return s, nil
}

func toTime(raw json.RawMessage) (time.Time, error) {
	raw, err := checkRaw(raw)
	if err != nil {
		return time.Time{}, err
	}

	var n json.Number
	if err = json.Unmarshal(raw, &n); err == nil {
		var t time.Time
		var whole int64
		var frac float64

		i, err := n.Int64()
		if err != nil {
			f, err := n.Float64()
			if err != nil {
				return time.Time{}, fmt.Errorf("%q: %w", truncate(raw), ErrParseTime)
			}

			var trunc float64
			trunc, frac = math.Modf(f)

			if err := checkInt64Range(trunc); err != nil {
				return time.Time{}, err
			}

			whole = int64(trunc)
		} else {
			whole = i
		}

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

	var s string
	err = json.Unmarshal(raw, &s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q: %w", truncate(raw), ErrParseTime)
	}

	layouts := []string{
		time.RFC3339, // this type handles fractional seconds, so there is no need to the nano variant
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"02/Jan/2006:15:04:05 -0700",
	}

	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("%q: %w", s, ErrParseTime)
}

func toDuration(raw json.RawMessage, unit time.Duration) (time.Duration, error) {
	raw, err := checkRaw(raw)
	if err != nil {
		return 0, err
	}

	var duration time.Duration

	var n json.Number
	if err = json.Unmarshal(raw, &n); err != nil {
		var s string
		err := json.Unmarshal(raw, &s)
		if err != nil {
			return 0, fmt.Errorf("%q: %w", truncate(raw), ErrParseDuration)
		}

		duration, err = time.ParseDuration(s)
		if err != nil {
			f, e := strconv.ParseFloat(s, 64)
			if e != nil {
				return 0, fmt.Errorf("%q: %w", truncate(raw), ErrParseDuration)
			}

			durationFloat := f * float64(unit)

			if e := checkInt64Range(durationFloat); e != nil {
				return 0, e
			}

			duration = time.Duration(durationFloat)
		}
	} else {
		f, err := n.Float64()
		if err != nil {
			return 0, fmt.Errorf("%q: %w", truncate(raw), ErrParseDuration)
		}

		durationFloat := f * float64(unit)

		if err := checkInt64Range(durationFloat); err != nil {
			return 0, err
		}

		duration = time.Duration(durationFloat)
	}

	if duration < 0 {
		return 0, fmt.Errorf("%q: %w", truncate(raw), ErrNegativeDuration)
	}

	return duration, nil
}

func checkRaw(raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)

	if len(raw) == 0 {
		return nil, ErrEmpty
	}

	if bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("%q: %w", truncate(raw), ErrNull)
	}

	return raw, nil
}

func truncate(raw json.RawMessage) json.RawMessage {
	if len(raw) <= 64 {
		return raw
	}

	limit := 64

	for limit > 0 && !utf8.RuneStart(raw[limit]) {
		limit--
	}

	truncated := make([]byte, limit, limit+3)
	copy(truncated, raw[:limit])
	truncated = append(truncated, "..."...)

	return truncated
}

func checkInt64Range(f float64) error {
	if math.IsNaN(f) {
		return fmt.Errorf("%v: %w", f, ErrNaN)
	}

	if math.IsInf(f, 0) {
		return fmt.Errorf("%v: %w", f, ErrInf)
	}

	// asymmetric on purpose: float64(MaxInt64) rounds up to 2^63 (out of range, so >=),
	// while float64(MinInt64) is exactly -2^63 (in range, so <).
	if f >= float64(math.MaxInt64) || f < float64(math.MinInt64) {
		return fmt.Errorf("%v: %w", f, ErrIntOutOfRange)
	}

	return nil
}
