package parse

import (
	"errors"
	"testing"
	"time"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   [][]byte
		check   []func(*Entry, *JSON) (bool, string)
		wantErr []error
	}{
		{
			name:    "empty",
			input:   [][]byte{nil},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "spaces",
			input:   [][]byte{[]byte("   ")},
			wantErr: []error{ErrMalformedLine},
		},
		// pink floyd reference
		{
			name:    "non-json",
			input:   [][]byte{[]byte("not json")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "array",
			input:   [][]byte{[]byte("[1,2,3]")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "null",
			input:   [][]byte{[]byte("null")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "brackets only",
			input:   [][]byte{[]byte("{}")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "no alias match",
			input:   [][]byte{[]byte("{\"a\":1}")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "value override",
			input: [][]byte{[]byte("{\"status\":200, \"status\":404}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Status == 404, "e.Status == 404" },
			},
		},
		{
			name:    "capitalized alias",
			input:   [][]byte{[]byte("{\"Status\":200}")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "valid with surrounding spaces",
			input: [][]byte{[]byte(" {\"status\":200} ")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Status == 200, "e.Status == 200" },
			},
		},
		{
			name:  "status out of range",
			input: [][]byte{[]byte("{\"status\":700}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return !e.Has(FieldStatus) && j.FieldErrors()[FieldStatus] == 1,
						"!e.Has(Status) && j.FieldErrors[Status] == 1"
				},
			},
		},
		{
			name:  "valid and null string fields",
			input: [][]byte{[]byte("{\"level\":\"INFO\", \"msg\":null}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return e.Level == "INFO" && !e.Has(FieldMessage) && j.FieldErrors()[FieldMessage] == 0,
						"e.Level == \"INFO\" && !e.Has(Message) && j.FieldErrors[Message] == 0"
				},
			},
		},
		{
			name:    "non-string value into string field",
			input:   [][]byte{[]byte("{\"request_id\":3}")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "valid bytes",
			input: [][]byte{[]byte("{\"bytes\":24}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Bytes == 24, "e.Bytes == 24" },
			},
		},
		{
			name:  "negative bytes",
			input: [][]byte{[]byte("{\"bytes\":-24}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return !e.Has(FieldBytes) && j.FieldErrors()[FieldBytes] == 1,
						"!e.Has(Bytes) && j.FieldErrors[Bytes] == 1"
				},
			},
		},
		{
			name:  "large bytes",
			input: [][]byte{[]byte("{\"bytes\":1e500}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return !e.Has(FieldBytes) && j.FieldErrors()[FieldBytes] == 1,
						"!e.Has(Bytes) && j.FieldErrors[Bytes] == 1"
				},
			},
		},
		{
			name:    "invalid bytes",
			input:   [][]byte{[]byte("{\"bytes\":\"abc\"}")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "2 time aliases",
			input: [][]byte{[]byte("{\"time\":\"2026-08-28T09:57:15+03:00\", \"timestamp\":\"2026-07-28T09:57:15+03:00\"}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return j.Fieldmap()[FieldTimestamp] == "time", "j.Fieldmap[Timestamp] == \"time\""
				},
			},
		},
		{
			name:  "2 lines with different status aliases",
			input: [][]byte{[]byte("{\"code\":500}"), []byte("{\"status\":200}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return e.Status == 500 && j.Fieldmap()[FieldStatus] == "code",
						"e.Status == 500 && j.Fieldmap[Status] == \"code\""
				},
			},
			wantErr: []error{nil, ErrMalformedLine},
		},
		{
			name:  "1 null 1 valid timestamp with different aliases",
			input: [][]byte{[]byte("{\"time\":null}"), []byte("{\"timestamp\":\"2026-08-28T09:57:15+03:00\"}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return !e.Has(FieldTimestamp) && j.Fieldmap()[FieldTimestamp] == "time" && j.FieldErrors()[FieldTimestamp] == 0,
						"!e.Has(Timestamp) && j.Fieldmap[Timestamp] == \"time\" && j.FieldErrors[Timestamp] == 0"
				},
			},
			wantErr: []error{nil, ErrMalformedLine},
		},
		{
			name:  "duration ms",
			input: [][]byte{[]byte("{\"duration_ms\":24}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Duration == 24*time.Millisecond, "e.Duration == 24ms" },
			},
		},
		{
			name:  "duration us",
			input: [][]byte{[]byte("{\"duration_us\":24}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Duration == 24*time.Microsecond, "e.Duration == 24us" },
			},
		},
		{
			name:  "request time (s) as ms float",
			input: [][]byte{[]byte("{\"request_time\":0.024}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Duration == 24*time.Millisecond, "e.Duration == 24ms" },
			},
		},
		{
			name:  "duration (ms) as us float",
			input: [][]byte{[]byte("{\"duration\":0.024}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Duration == 24*time.Microsecond, "e.Duration == 24us" },
			},
		},
		{
			name:  "duration us as ms unit string",
			input: [][]byte{[]byte("{\"duration_us\":\"24ms\"}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Duration == 24*time.Millisecond, "e.Duration == 24ms" },
			},
		},
		{
			name:  "duration us as string",
			input: [][]byte{[]byte("{\"duration_us\":\"24\"}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, _ *JSON) (bool, string) { return e.Duration == 24*time.Microsecond, "e.Duration == 24us" },
			},
		},
		{
			name:  "negative duration ms",
			input: [][]byte{[]byte("{\"duration_ms\":-4}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return !e.Has(FieldDuration) && j.FieldErrors()[FieldDuration] == 1,
						"!e.Has(Duration) && j.FieldErrors[Duration] == 1"
				},
			},
		},
		{
			name:  "large duration ms",
			input: [][]byte{[]byte("{\"duration_ms\":1e300}")},
			check: []func(*Entry, *JSON) (bool, string){
				func(e *Entry, j *JSON) (bool, string) {
					return !e.Has(FieldDuration) && j.FieldErrors()[FieldDuration] == 1,
						"!e.Has(Duration) && j.FieldErrors[Duration] == 1"
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := NewJSON()

			for i := range tt.input {
				line := tt.input[i]
				var check func(*Entry, *JSON) (bool, string)
				var wantErr error

				if i < len(tt.check) {
					check = tt.check[i]
				} else {
					check = nil
				}

				if i < len(tt.wantErr) {
					wantErr = tt.wantErr[i]
				} else {
					wantErr = nil
				}

				e, err := j.Parse(line)

				if wantErr != nil {
					if err == nil {
						t.Fatalf("j.Parse(%q) succeeded, want error", line)
					}
					if !e.IsEmpty() {
						t.Errorf("j.Parse(%q) returned non-empty entry alongside error", line)
					}
					if !errors.Is(err, wantErr) {
						t.Errorf("j.Parse(%q) returned error %v, want %v", line, err, wantErr)
					}
					continue
				} else if err != nil {
					t.Fatalf("j.Parse(%q) unexpected error: %v", line, err)
				}

				if check != nil {
					if result, str := check(&e, j); !result {
						t.Errorf("check(%q) failed for %#v", str, e)
					}
				}
			}
		})
	}
}
