package parse

import (
	"errors"
	"testing"
	"time"
)

func TestNginx(t *testing.T) {
	tests := []struct {
		name    string
		input   [][]byte
		check   []func(*Entry, *Nginx) (bool, string)
		wantErr []error
	}{
		{
			name:    "empty",
			input:   [][]byte{nil},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "spaces",
			input:   [][]byte{[]byte("          ")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "common format",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /search\\\" HTTP/1.0\" 200 624")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					loc := time.FixedZone("UTC-4", -4*60*60)
					return e.RemoteAddr == "google.com" && e.User == "A" &&
						e.Timestamp.Equal(time.Date(1995, time.August, 1, 0, 0, 1, 0, loc)) && e.Method == "GET" &&
						e.Path == "/search\\\"" && e.Protocol == "HTTP/1.0" && e.Status == 200 && e.Bytes == 624 &&
						!e.Has(FieldReferer) && !e.Has(FieldUserAgent) && !e.Has(FieldDuration), "<each field same>"
				},
			},
		},
		{
			name:  "combined format",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /search HTTP/1.0\" 200 624 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\"")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					loc := time.FixedZone("UTC-4", -4*60*60)
					return e.RemoteAddr == "google.com" && e.User == "A" &&
						e.Timestamp.Equal(time.Date(1995, time.August, 1, 0, 0, 1, 0, loc)) && e.Method == "GET" &&
						e.Path == "/search" && e.Protocol == "HTTP/1.0" && e.Status == 200 && e.Bytes == 624 &&
						e.Referer == "https://shop.example/" && e.UserAgent == "Mozilla/5.0 (Macintosh)" &&
						!e.Has(FieldDuration), "<each field same>"
				},
			},
		},
		{
			name:  "combined+timing format",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /search HTTP/1.0\" 200 624 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					loc := time.FixedZone("UTC-4", -4*60*60)
					return e.RemoteAddr == "google.com" && e.User == "A" &&
						e.Timestamp.Equal(time.Date(1995, time.August, 1, 0, 0, 1, 0, loc)) && e.Method == "GET" &&
						e.Path == "/search" && e.Protocol == "HTTP/1.0" && e.Status == 200 && e.Bytes == 624 &&
						e.Referer == "https://shop.example/" && e.UserAgent == "Mozilla/5.0 (Macintosh)" &&
						e.Duration == 240*time.Millisecond, "<each field same>"
				},
			},
		},
		{
			name:  "tabs instead of spaces",
			input: [][]byte{[]byte("google.com\t-\tA\t[01/Aug/1995:00:00:01 -0400]\t\"GET\t/search HTTP/1.0\"\t200\t624 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					loc := time.FixedZone("UTC-4", -4*60*60)
					return e.RemoteAddr == "google.com" && e.User == "A" &&
						e.Timestamp.Equal(time.Date(1995, time.August, 1, 0, 0, 1, 0, loc)) && e.Method == "GET" &&
						e.Path == "/search" && e.Protocol == "HTTP/1.0" && e.Status == 200 && e.Bytes == 624 &&
						e.Referer == "https://shop.example/" && e.UserAgent == "Mozilla/5.0 (Macintosh)" &&
						e.Duration == 240*time.Millisecond, "<each field same>"
				},
			},
		},
		{
			name:  "timestamp not set",
			input: [][]byte{[]byte("github.com - fatihege [-] \"GET /search HTTP/1.0\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldTimestamp) && n.FieldErrors()[FieldTimestamp] == 0,
						"!e.Has(Timestamp) && n.FieldErrors[Timestamp] == 0"
				},
			},
		},
		{
			name:  "invalid timestamp format",
			input: [][]byte{[]byte("github.com - fatihege [01/08/1995:00:00:01 -04:00] \"GET /search HTTP/1.0\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldTimestamp) && n.FieldErrors()[FieldTimestamp] == 1,
						"!e.Has(Timestamp) && n.FieldErrors[Timestamp] == 1"
				},
			},
		},
		{
			name:  "no fields set",
			input: [][]byte{[]byte("- - - [-] \"-\" - - \"-\" \"-\" -")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return e.Mask == 0 && len(n.FieldErrors()) == 0,
						"e.Mask == 0 && len(n.FieldErrors) == 0"
				},
			},
		},
		{
			name:  "only remote address set",
			input: [][]byte{[]byte("github.com - - [-] \"-\" - - \"-\" \"-\" -")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return e.RemoteAddr == "github.com" && e.Mask == FieldRemoteAddr && len(n.FieldErrors()) == 0,
						"e.RemoteAddr == \"github.com\" && e.Mask == RemoteAddr && len(n.FieldErrors) == 0"
				},
			},
		},
		{
			name:  "path with query string",
			input: [][]byte{[]byte("- - - [-] \"GET /search?q=abc\" - - \"-\" \"-\" -")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					return e.Method == "GET" && e.Path == "/search" && e.Query == "q=abc",
						"e.Method == \"GET\" && e.Path == \"/search\" && e.Query == \"q=abc\""
				},
			},
		},
		{
			name:  "empty query string",
			input: [][]byte{[]byte("- - - [-] \"GET /search?\" - - \"-\" \"-\" -")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return e.Method == "GET" && e.Path == "/search" && !e.Has(FieldQuery) && n.FieldErrors()[FieldQuery] == 0,
						"e.Method == \"GET\" && e.Path == \"/search\" && !e.Has(Query) && n.FieldErrors[Query] == 0"
				},
			},
		},
		{
			name:  "request with only method",
			input: [][]byte{[]byte("- - - [-] \"GET\" - - \"-\" \"-\" -")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					return e.Mask == 0,
						"e.Mask == 0"
				},
			},
		},
		{
			name:  "unusual method",
			input: [][]byte{[]byte("- - - [-] \"GRBG /health\" - - \"-\" \"-\" -")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					return e.Method == "GRBG" && e.Has(FieldMethod),
						"e.Method == \"GRBG\" && e.Has(Method)"
				},
			},
		},
		{
			name:  "valid line with surrounding spaces",
			input: [][]byte{[]byte("  github.com - - [-] \"-\" - 64 \"-\" \"-\" -  ")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					return e.RemoteAddr == "github.com" && e.Bytes == 64 && e.Has(FieldRemoteAddr) && e.Has(FieldBytes),
						"e.RemoteAddr == \"github.com\" && e.Bytes == 64 && e.Has(RemoteAddr) && e.Has(Bytes)"
				},
			},
		},
		{
			name:    "backslash as last byte in quote",
			input:   [][]byte{[]byte("- - - [-] \"abc\\\" - - \"-\" \"-\" -")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "random spaces in between",
			input: [][]byte{[]byte("  google.com  -    A  [01/Aug/1995:00:00:01 -0400] \"GET  /health \" 200  64  \"https://shop.example/\"   \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, _ *Nginx) (bool, string) {
					loc := time.FixedZone("UTC-4", -4*60*60)
					return e.RemoteAddr == "google.com" && e.User == "A" &&
						e.Timestamp.Equal(time.Date(1995, time.August, 1, 0, 0, 1, 0, loc)) && e.Method == "GET" &&
						e.Path == "/health" && !e.Has(FieldProtocol) && e.Status == 200 &&
						e.Bytes == 64, "<each field present and same, except protocol>"
				},
			},
		},
		{
			name:    "no space between brackets and quotes",
			input:   [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400]\"GET /health\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "unterminated bracket",
			input:   [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400 \"GET /health\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "unterminated quote",
			input:   [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "missing opening bracket",
			input:   [][]byte{[]byte("google.com - A 01/Aug/1995:00:00:01 -0400] \"GET /health\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "missing opening quote",
			input:   [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] GET /health\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "repeated brackets",
			input:   [][]byte{[]byte("google.com - A [[01/Aug/1995:00:00:01 -0400]] \"GET /health\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "repeated quotes",
			input:   [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"\"GET /health\"\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:    "missing after request",
			input:   [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health\"")},
			wantErr: []error{ErrMalformedLine},
		},
		{
			name:  "status out of range",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health\" 50 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldStatus) && n.FieldErrors()[FieldStatus] == 1,
						"!e.Has(Status) && n.FieldErrors[Status] == 1"
				},
			},
		},
		{
			name:  "negative bytes",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health\" 200 -64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldBytes) && n.FieldErrors()[FieldBytes] == 1,
						"!e.Has(Bytes) && n.FieldErrors[Bytes] == 1"
				},
			},
		},
		{
			name:  "numeric field overflow",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health\" 200 1e500 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldBytes) && n.FieldErrors()[FieldBytes] == 1,
						"!e.Has(Bytes) && n.FieldErrors[Bytes] == 1"
				},
			},
		},
		{
			name:  "non-numeric int field",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health\" 200 abc \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" 0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldBytes) && n.FieldErrors()[FieldBytes] == 1,
						"!e.Has(Bytes) && n.FieldErrors[Bytes] == 1"
				},
			},
		},
		{
			name:  "negative duration",
			input: [][]byte{[]byte("google.com - A [01/Aug/1995:00:00:01 -0400] \"GET /health\" 200 64 \"https://shop.example/\" \"Mozilla/5.0 (Macintosh)\" -0.24")},
			check: []func(*Entry, *Nginx) (bool, string){
				func(e *Entry, n *Nginx) (bool, string) {
					return !e.Has(FieldDuration) && n.FieldErrors()[FieldDuration] == 1,
						"!e.Has(Duration) && n.FieldErrors[Duration] == 1"
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNginx()

			for i := range tt.input {
				line := tt.input[i]
				var check func(*Entry, *Nginx) (bool, string)
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

				e, err := n.Parse(line)

				if check != nil {
					if result, str := check(&e, n); !result {
						t.Errorf("check(%q) failed for %#v", str, e)
					}
				}

				if wantErr != nil {
					if err == nil {
						t.Fatalf("n.Parse(%q) succeeded, want error", line)
					}
					if e.Mask != 0 {
						t.Errorf("n.Parse(%q) returned non-empty entry alongside error", line)
					}
					if !errors.Is(err, wantErr) {
						t.Errorf("n.Parse(%q) returned error %v, want %v", line, err, wantErr)
					}
				} else if err != nil {
					t.Fatalf("n.Parse(%q) unexpected error: %v", line, err)
				}
			}
		})
	}
}
