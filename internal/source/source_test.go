package source

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name        string
		inputs      [][]byte
		want        string
		wantWrapErr bool
		wantReadErr bool
		compress    bool
		corrupt     func([]byte) []byte
	}{
		{name: "plain text", inputs: [][]byte{[]byte("love\ngo")}, want: "love\ngo"},
		{name: "valid gzip", inputs: [][]byte{[]byte("this is\na gzip")}, want: "this is\na gzip", compress: true},
		{name: "empty input", inputs: nil, want: ""},
		{name: "1-byte input", inputs: [][]byte{[]byte("f")}, want: "f"},
		{name: "corrupt gzip header", inputs: [][]byte{bytes.Join([][]byte{{0x1f, 0x8b}, []byte("corrupted")}, nil)}, wantWrapErr: true},
		{name: "truncated gzip", inputs: [][]byte{[]byte("always give up")}, wantReadErr: true, compress: true, corrupt: func(b []byte) []byte {
			return b[:len(b)-12]
		}},
		{name: "bad checksum", inputs: [][]byte{[]byte("john pork is calling")}, wantReadErr: true, compress: true, corrupt: func(b []byte) []byte {
			r := slices.Clone(b)
			r[len(r)-5] ^= 0xFF
			return r
		}},
		{name: "2 gzip members", inputs: [][]byte{[]byte("it rains"), []byte(" milk today")}, want: "it rains milk today", compress: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []byte = nil
			if len(tt.inputs) > 0 && tt.inputs[0] != nil {
				input = tt.inputs[0]
			}

			if tt.compress {
				input = compress(t, input)

				if len(tt.inputs) > 1 && tt.inputs[1] != nil {
					second := compress(t, tt.inputs[1])
					input = bytes.Join([][]byte{input, second}, nil)
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
				t.Errorf("first rc.Close() unexpected error: %v", err)
			}
			if err := rc.Close(); err != nil {
				t.Errorf("second rc.Close() unexpected error: %v", err)
			}
		})
	}
}

func TestOpen(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		want     string
		wantErr  error
		path     func(string) string
		compress bool
	}{
		{name: "plain file with .gz extension", input: []byte("hate flies"), want: "hate flies", path: func(dir string) string {
			return filepath.Join(dir, "file.gz")
		}},
		{name: "gzipped file with .txt extension", input: []byte("compressed text"), want: "compressed text", path: func(dir string) string {
			return filepath.Join(dir, "compressed.txt")
		}, compress: true},
		{name: "directory", wantErr: ErrPathIsDir, path: func(dir string) string {
			return dir
		}},
		{name: "missing file", wantErr: fs.ErrNotExist, path: func(dir string) string {
			return filepath.Join(dir, "nope")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.path(t.TempDir())

			if tt.wantErr == nil {
				input := tt.input

				if tt.compress {
					input = compress(t, input)
				}

				if err := os.WriteFile(p, input, 0o644); err != nil {
					t.Fatalf("os.WriteFile(%q) unexpected error: %v", p, err)
				}
			}

			rc, err := Open(p)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Open(%q) succeeded, want error", p)
				}
				if rc != nil {
					t.Errorf("Open(%q) returned non-nil reader (%T) alongside error", p, rc)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Open(%q) returned error %v, want %v", p, err, tt.wantErr)
				}
				return
			}

			t.Cleanup(func() { _ = rc.Close() })

			got, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("io.ReadAll(rc) unexpected error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("io.ReadAll(rc) = %q, want %q", got, tt.want)
			}

			if err := rc.Close(); err != nil {
				t.Errorf("first rc.Close() unexpected error: %v", err)
			}
			if err := rc.Close(); err != nil {
				t.Errorf("second rc.Close() unexpected error: %v", err)
			}
		})
	}
}

func compress(t *testing.T, b []byte) []byte {
	t.Helper()

	var compressedBuffer bytes.Buffer

	gw := gzip.NewWriter(&compressedBuffer)
	r := bytes.NewReader(b)

	_, err := io.Copy(gw, r)
	if err != nil {
		t.Fatalf("io.Copy(gw, r) unexpected error: %v", err)
	}

	if err := gw.Close(); err != nil {
		t.Fatalf("gw.Close() unexpected error: %v", err)
	}

	return compressedBuffer.Bytes()
}
