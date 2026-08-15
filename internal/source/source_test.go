package source

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		input2      []byte
		want        string
		wantWrapErr bool
		wantReadErr bool
		compress    bool
		corrupt     func([]byte) []byte
	}{
		{name: "plain text", input: []byte("love\ngo"), want: "love\ngo"},
		{name: "valid gzip", input: []byte("this is\na gzip"), want: "this is\na gzip", compress: true},
		{name: "empty input", input: nil, want: ""},
		{name: "1-byte input", input: []byte("f"), want: "f"},
		{name: "corrupt gzip header", input: bytes.Join([][]byte{{0x1f, 0x8b}, []byte("corrupted")}, nil), wantWrapErr: true},
		{name: "truncated gzip", input: []byte("give up"), wantReadErr: true, compress: true, corrupt: func(b []byte) []byte {
			return b[:len(b)-20]
		}},
		{name: "bad checksum", input: []byte("john pork is calling"), wantReadErr: true, compress: true, corrupt: func(b []byte) []byte {
			r := b
			r[len(r)-5] ^= 0xFF
			return r
		}},
		{name: "2 gzip members", input: []byte("it rains"), input2: []byte(" milk today"), want: "it rains milk today", compress: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			input2 := tt.input2

			if tt.compress {
				input = compress(t, bytes.NewReader(input))

				if input2 != nil {
					input2 = compress(t, bytes.NewReader(input2))
					input = bytes.Join([][]byte{input, input2}, nil)
				}
			}

			if tt.corrupt != nil {
				input = tt.corrupt(input)
			}

			br := bytes.NewReader(input)
			rc, err := Wrap(br)
			if tt.wantWrapErr {
				if err == nil {
					t.Fatal("Wrap(br) succeeded, want error")
				}
				if rc != nil {
					t.Errorf("Wrap(br) returned non-nil reader (%T) alongside error", rc)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrap(br) unexpected error: %v", err)
			}

			t.Cleanup(func() { _ = rc.Close() })

			got, err := io.ReadAll(rc)
			if tt.wantReadErr {
				if err == nil {
					t.Fatalf("io.ReadAll(rc) = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("io.ReadAll(rc) unexpected error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("io.ReadAll(rc) = %q, want %q", got, tt.want)
			}

			if err := rc.Close(); err != nil {
				t.Errorf("rc.Close() unexpected error: %v", err)
			}
		})
	}
}

func compress(t *testing.T, r io.Reader) []byte {
	t.Helper()

	var compressedBuffer bytes.Buffer

	gw := gzip.NewWriter(&compressedBuffer)

	_, err := io.Copy(gw, r)
	if err != nil {
		t.Fatalf("io.Copy(gw, br) unexpected error: %v", err)
	}

	if err := gw.Close(); err != nil {
		t.Fatalf("gw.Close() unexpected error: %v", err)
	}

	return compressedBuffer.Bytes()
}
