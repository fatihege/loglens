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
	raw, err := normalize(raw)
	if err != nil {
		return 0, err // returned error already has context
	}

	var f float64

	if raw[0] == '"' {
		var s string
		err := json.Unmarshal(raw, &s)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", truncate(raw), ErrParseInt)
		}

		f, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("%q: %w", s, ErrParseInt)
		}
	} else {
		err = json.Unmarshal(raw, &f)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", truncate(raw), ErrParseInt)
		}
	}

	trunc, frac := math.Modf(f)

	if frac != 0 {
		return 0, fmt.Errorf("%v: %w", f, ErrParseInt)
	}

	if err := checkIntRange(trunc); err != nil {
		return 0, err // returned error already has context
	}

	return int(trunc), nil
}

func toString(raw json.RawMessage) (string, error) {
	raw, err := normalize(raw)
	if err != nil {
		return "", err
	}

	if raw[0] != '"' {
		return "", fmt.Errorf("%s: %w", truncate(raw), ErrParseString)
	}

	var s string
	err = json.Unmarshal(raw, &s)
	if err != nil {
		return "", fmt.Errorf("%s: %w", truncate(raw), ErrParseString)
	}

	return s, nil
}

func toTime(raw json.RawMessage) (time.Time, error) {
	raw, err := normalize(raw)
	if err != nil {
		return time.Time{}, err
	}

	if raw[0] != '"' {
		var timeFloat float64
		var t time.Time
		err := json.Unmarshal(raw, &timeFloat)
		if err != nil {
			return time.Time{}, fmt.Errorf("%s: %w", truncate(raw), ErrParseTime)
		}

		trunc, frac := math.Modf(timeFloat)

		if err := checkFloatRange(trunc); err != nil {
			return time.Time{}, err
		}

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
	err = json.Unmarshal(raw, &timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", truncate(raw), ErrParseTime)
	}

	layouts := []string{
		time.RFC3339, // this type handles fractional seconds, so there is no need to the nano variant
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}

	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, timeStr, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("%q: %w", timeStr, ErrParseTime)
}

func toDuration(raw json.RawMessage, unit time.Duration) (time.Duration, error) {
	raw, err := normalize(raw)
	if err != nil {
		return 0, err
	}

	var duration time.Duration

	if raw[0] == '"' {
		var s string
		err := json.Unmarshal(raw, &s)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", truncate(raw), ErrParseDuration)
		}

		duration, err = time.ParseDuration(s)
		if err != nil {
			f, e := strconv.ParseFloat(s, 64)
			if e != nil {
				return 0, fmt.Errorf("%s: %w", truncate(raw), ErrParseDuration)
			}

			durationFloat := f * float64(unit)

			if err := checkFloatRange(durationFloat); err != nil {
				return 0, err
			}

			duration = time.Duration(durationFloat)
		}
	} else {
		var f float64
		err := json.Unmarshal(raw, &f)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", truncate(raw), ErrParseDuration)
		}

		durationFloat := f * float64(unit)

		if err := checkFloatRange(durationFloat); err != nil {
			return 0, err
		}

		duration = time.Duration(durationFloat)
	}

	if duration < 0 {
		return 0, fmt.Errorf("%s: %w", truncate(raw), ErrNegativeDuration)
	}

	return duration, nil
}

func normalize(raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)

	if len(raw) == 0 {
		return nil, ErrEmpty
	}

	if bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("%s: %w", truncate(raw), ErrNull)
	}

	return raw, nil
}

func truncate(raw json.RawMessage) json.RawMessage {
	if len(raw) > 64 {
		return raw[:64]
	}

	return raw
}

func checkIntRange(f float64) error {
	// asymmetric on purpose: float64(MaxInt) rounds up to 2^63 (out of range, so >=),
	// while float64(MinInt) is exactly -2^63 (in range, so <).
	if f >= float64(math.MaxInt) || f < float64(math.MinInt) {
		return fmt.Errorf("%v: %w", f, ErrIntOutOfRange)
	}

	return nil
}

func checkFloatRange(f float64) error {
	// asymmetric on purpose: float64(MaxInt64) rounds up to 2^63 (out of range, so >=),
	// while float64(MinInt64) is exactly -2^63 (in range, so <).
	if f >= float64(math.MaxInt64) || f < float64(math.MinInt64) {
		return fmt.Errorf("%v: %w", f, ErrIntOutOfRange)
	}

	return nil
}
