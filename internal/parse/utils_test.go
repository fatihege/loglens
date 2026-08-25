package parse

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrEmpty},
		{name: "quoted empty", input: "\"\"", wantErr: ErrParseInt},
		{name: "null input", input: "null", wantErr: ErrNull},
		{name: "unquoted garbage", input: "abc", wantErr: ErrParseInt},
		{name: "quoted non-numeric", input: "\"abc\"", wantErr: ErrParseInt},
		{name: "boolean", input: "true", wantErr: ErrParseInt},
		{name: "array", input: "[]", wantErr: ErrParseInt},
		{name: "object", input: "{}", wantErr: ErrParseInt},
		{name: "positive int", input: "24", want: 24},
		{name: "negative int", input: "-24", want: -24},
		{name: "int with surrounding spaces", input: " 24 ", want: 24},
		{name: "invalid number", input: "007", wantErr: ErrParseInt},
		{name: "float with fraction", input: "24.4", wantErr: ErrParseInt},
		{name: "float without fraction", input: "24.0", want: 24},
		{name: "string int", input: "\"24\"", want: 24},
		{name: "string float with fraction", input: "\"24.4\"", wantErr: ErrParseInt},
		{name: "string float without fraction", input: "\"24.0\"", want: 24},
		{name: "int at max boundary", input: "9223372036854775807", want: 9223372036854775807},
		{name: "int out of max range", input: "9223372036854775808", wantErr: ErrIntOutOfRange},
		{name: "int out of min range", input: "-1e19", wantErr: ErrIntOutOfRange},
		{name: "float out of range", input: "1e500", wantErr: ErrIntOutOfRange},
	}

	t.Parallel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := json.RawMessage(tt.input)
			got, err := toInt(raw)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("toInt(%q) succeeded, want error", raw)
				}
				if got != 0 {
					t.Errorf("toInt(%q) returned non-zero int (%d) alongside error", raw, got)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("toInt(%q) returned error %v, want %v", raw, err, tt.wantErr)
				}
				return
			} else if err != nil {
				t.Fatalf("toInt(%q) unexpected error: %v", raw, err)
			}

			if got != tt.want {
				t.Errorf("toInt(%q) = %d, want %d", raw, got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrEmpty},
		{name: "quoted empty", input: "\"\"", want: ""},
		{name: "null input", input: "null", wantErr: ErrNull},
		{name: "unquoted garbage", input: "abc", wantErr: ErrParseString},
		{name: "boolean", input: "true", wantErr: ErrParseString},
		{name: "array", input: "[]", wantErr: ErrParseString},
		{name: "object", input: "{}", wantErr: ErrParseString},
		{name: "int", input: "24", wantErr: ErrParseString},
		{name: "float", input: "24.4", wantErr: ErrParseString},
		{name: "valid string", input: "\"abc\"", want: "abc"},
		{name: "string with surrounding spaces", input: " \"abc\" ", want: "abc"},
		{name: "unfinished quotes", input: "\"abc", wantErr: ErrParseString},
		{name: "escaped characters", input: "\"abc \\\"def\\\"\\nghi\"", want: "abc \"def\"\nghi"},
	}

	t.Parallel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := json.RawMessage(tt.input)
			got, err := toString(raw)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("toString(%q) succeeded, want error", raw)
				}
				if got != "" {
					t.Errorf("toString(%q) returned non-empty string (%q) alongside error", raw, got)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("toString(%q) returned error %v, want %v", raw, err, tt.wantErr)
				}
				return
			} else if err != nil {
				t.Fatalf("toString(%q) unexpected error: %v", raw, err)
			}

			if got != tt.want {
				t.Errorf("toString(%q) = %q, want %q", raw, got, tt.want)
			}
		})
	}
}

func TestToTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    func() time.Time
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrEmpty},
		{name: "quoted empty", input: "\"\"", wantErr: ErrParseTime},
		{name: "null input", input: "null", wantErr: ErrNull},
		{name: "unquoted garbage", input: "abc", wantErr: ErrParseTime},
		{name: "quoted garbage", input: "\"abc\"", wantErr: ErrParseTime},
		{name: "boolean", input: "true", wantErr: ErrParseTime},
		{name: "array", input: "[]", wantErr: ErrParseTime},
		{name: "object", input: "{}", wantErr: ErrParseTime},
		{name: "int nanosecond", input: "133532235323343554", want: func() time.Time { return time.Unix(0, 133532235323343554) }},
		{name: "int microsecond", input: "133532235323343", want: func() time.Time { return time.UnixMicro(133532235323343) }},
		{name: "int millisecond", input: "133532235323", want: func() time.Time { return time.UnixMilli(133532235323) }},
		{name: "int second", input: "133532235", want: func() time.Time { return time.Unix(133532235, 0) }},
		{name: "negative int millisecond", input: "-133532235323", want: func() time.Time { return time.UnixMilli(-133532235323) }},
		{name: "float microsecond", input: "133532235323343.25", want: func() time.Time { return time.UnixMicro(133532235323343).Add(250 * time.Nanosecond) }}, // used .25 because it is exactly reprasantable in binary (2^-2)
		{name: "float millisecond", input: "133532235323.25", want: func() time.Time { return time.UnixMilli(133532235323).Add(250 * time.Microsecond) }},
		{name: "float second", input: "133532235.25", want: func() time.Time { return time.Unix(133532235, int64(250*time.Millisecond)) }},
		{name: "negative float millisecond", input: "-133532235323.25", want: func() time.Time { return time.UnixMilli(-133532235323).Add(-250 * time.Microsecond) }},
		{name: "float out of max range", input: "1e500", wantErr: ErrParseTime},
		{name: "float out of min range", input: "-1e500", wantErr: ErrParseTime},
		{name: "string with timezone", input: "\"2026-08-25 22:42:17+03:00\"", want: func() time.Time {
			t, _ := time.Parse("2006-01-02 15:04:05Z07:00", "2026-08-25 22:42:17+03:00")
			return t
		}},
		{name: "rfc3339 string with timezone", input: "\"2024-10-10T14:03:14.240+03:00\"", want: func() time.Time {
			t, _ := time.Parse(time.RFC3339, "2024-10-10T14:03:14.240+03:00")
			return t
		}},
		{name: "string with timezone with surrounding spaces", input: " \"2026-08-25 22:42:17+03:00\" ", want: func() time.Time {
			t, _ := time.Parse("2006-01-02 15:04:05Z07:00", "2026-08-25 22:42:17+03:00")
			return t
		}},
		{name: "string without timezone", input: "\"2026-08-25 22:42:17\"", want: func() time.Time {
			t, _ := time.ParseInLocation("2006-01-02 15:04:05", "2026-08-25 22:42:17", time.Local)
			return t
		}},
		{name: "invalid time string", input: "\"2026-16-25 44:42:17+03:00\"", wantErr: ErrParseTime},
		{name: "string with invalid format", input: "\"2026-8-25 22:42:17 +03:00\"", wantErr: ErrParseTime},
	}

	t.Parallel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := json.RawMessage(tt.input)
			got, err := toTime(raw)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("toTime(%q) succeeded, want error", raw)
				}
				if !got.IsZero() {
					t.Errorf("toTime(%q) returned non-zero time (%v) alongside error", raw, got)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("toTime(%q) returned error %v, want %v", raw, err, tt.wantErr)
				}
				return
			} else if err != nil {
				t.Fatalf("toTime(%q) unexpected error: %v", raw, err)
			}

			if want := tt.want(); !got.Equal(want) {
				t.Errorf("toTime(%q) = %v, want %v", raw, got, want)
			}
		})
	}
}

func TestToDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		unit    time.Duration
		want    time.Duration
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrEmpty},
		{name: "quoted empty", input: "\"\"", wantErr: ErrParseDuration},
		{name: "null input", input: "null", wantErr: ErrNull},
		{name: "unquoted garbage", input: "abc", wantErr: ErrParseDuration},
		{name: "quoted garbage", input: "\"abc\"", wantErr: ErrParseDuration},
		{name: "boolean", input: "true", wantErr: ErrParseDuration},
		{name: "array", input: "[]", wantErr: ErrParseDuration},
		{name: "object", input: "{}", wantErr: ErrParseDuration},
		{name: "int ms", input: "24", unit: time.Millisecond, want: 24 * time.Millisecond},
		{name: "int us", input: "24", unit: time.Microsecond, want: 24 * time.Microsecond},
		{name: "negative", input: "-24", unit: time.Millisecond, wantErr: ErrNegativeDuration},
		{name: "float", input: "24.4", unit: time.Millisecond, want: 24400 * time.Microsecond},
		{name: "int out of range", input: "1e500", unit: time.Microsecond, wantErr: ErrParseDuration},
		{name: "valid string with unit", input: "\"24ms\"", unit: time.Second, want: 24 * time.Millisecond},
		{name: "string with surrounding spaces", input: " \"2m5s\" ", unit: time.Second, want: 2*time.Minute + 5*time.Second},
		{name: "string number without fraction", input: "\"24\"", unit: time.Second, want: 24 * time.Second},
		{name: "string number with fraction", input: "\"24.4\"", unit: time.Second, want: 24400 * time.Millisecond},
		{name: "string number out of range", input: "\"1e500\"", unit: time.Second, wantErr: ErrParseDuration},
		{name: "negative string number", input: "\"-24\"", unit: time.Second, wantErr: ErrNegativeDuration},
		{name: "us from ms", input: "0.024", unit: time.Millisecond, want: 24 * time.Microsecond},
		{name: "zero", input: "0", unit: time.Millisecond, want: 0},
	}

	t.Parallel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := json.RawMessage(tt.input)
			got, err := toDuration(raw, tt.unit)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("toDuration(%q) succeeded, want error", raw)
				}
				if got != 0 {
					t.Errorf("toDuration(%q) returned non-zero duration (%v) alongside error", raw, got)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("toDuration(%q) returned error %v, want %v", raw, err, tt.wantErr)
				}
				return
			} else if err != nil {
				t.Fatalf("toDuration(%q) unexpected error: %v", raw, err)
			}

			if got != tt.want {
				t.Errorf("toDuration(%q) = %v, want %v", raw, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input func() string
		want  func() string
	}{
		{name: "valid", input: func() string { return "abc" }, want: func() string { return "abc" }},
		{name: "empty", input: func() string { return "" }, want: func() string { return "" }},
		{name: "out of range", input: func() string { return strings.Repeat("a", 72) }, want: func() string { return strings.Repeat("a", 64) + "..." }},
		{name: "multi-byte rune", input: func() string { return strings.Repeat("a", 63) + "🎉" }, want: func() string { return strings.Repeat("a", 63) + "..." }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input()
			raw := json.RawMessage(input)
			want := tt.want()
			if got := truncate(raw); string(got) != want {
				t.Errorf("truncate(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestCheckInt64Range(t *testing.T) {
	tests := []struct {
		name    string
		input   float64
		wantErr error
	}{
		{name: "valid", input: 10, wantErr: nil},
		{name: "nan", input: math.NaN(), wantErr: ErrNaN},
		{name: "inf", input: math.Inf(1), wantErr: ErrInf},
		{name: "out of range", input: 9223372036854775808, wantErr: ErrIntOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkInt64Range(tt.input); !errors.Is(err, tt.wantErr) {
				t.Errorf("checkInt64Range(%v) returned error %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
