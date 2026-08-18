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
	"strings"
	"testing"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name        string
		inputs      [][]byte
		want        string
		wantWrapErr error
		wantReadErr error
		compress    bool
		corrupt     func([]byte) []byte
	}{
		{name: "plain text", inputs: [][]byte{[]byte("love\ngo")}, want: "love\ngo"},
		{name: "valid gzip", inputs: [][]byte{[]byte("this is\na gzip")}, want: "this is\na gzip", compress: true},
		{name: "empty input", inputs: nil, want: ""},
		{name: "1-byte input", inputs: [][]byte{[]byte("f")}, want: "f"},
		{name: "corrupt gzip header", inputs: [][]byte{bytes.Join([][]byte{{0x1f, 0x8b}, []byte("corrupted")}, nil)}, wantWrapErr: gzip.ErrHeader},
		{name: "truncated gzip", inputs: [][]byte{[]byte("always give up")}, wantReadErr: io.ErrUnexpectedEOF, compress: true, corrupt: func(b []byte) []byte {
			return b[:len(b)-12] // drop the 8-footer plus part of the deflate stream
		}},
		{name: "bad checksum", inputs: [][]byte{[]byte("john pork is calling")}, wantReadErr: gzip.ErrChecksum, compress: true, corrupt: func(b []byte) []byte {
			r := slices.Clone(b)
			r[len(r)-5] ^= 0xFF // flip the last byte of the CRC32
			return r
		}},
		{name: "2 gzip members", inputs: [][]byte{[]byte("it rains"), []byte(" milk today")}, want: "it rains milk today", compress: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []byte

			if len(tt.inputs) > 0 && tt.inputs[0] != nil {
				for i := 0; i < len(tt.inputs); i++ {
					current := tt.inputs[i]

					if tt.compress {
						current = gzipped(t, current)
					}

					input = bytes.Join([][]byte{input, current}, nil)
				}
			}

			if tt.corrupt != nil {
				input = tt.corrupt(input)
			}

			br := bytes.NewReader(input)
			rc, err := Wrap(br)
			if tt.wantWrapErr != nil {
				if err == nil {
					t.Fatal("Wrap(br) succeeded, want error")
				}
				if !errors.Is(err, tt.wantWrapErr) {
					t.Fatalf("Wrap(br) returned error %v, want %v", err, tt.wantWrapErr)
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
			if tt.wantReadErr != nil {
				if err == nil {
					t.Fatalf("io.ReadAll(rc) = %q, want error", got)
				}
				if !errors.Is(err, tt.wantReadErr) {
					t.Fatalf("io.ReadAll(rc) returned error %v, want %v", err, tt.wantReadErr)
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
		{name: "corrupt gzip file", input: bytes.Join([][]byte{{0x1f, 0x8b}, []byte("corrupted")}, nil), wantErr: gzip.ErrHeader, path: func(dir string) string { // input had to be longer than 10 bytes because gzip header is 10 bytes
			return filepath.Join(dir, "garbage.gz")
		}},
		{name: "empty gzip file", input: []byte{}, want: "", path: func(dir string) string {
			return filepath.Join(dir, "empty.gz")
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

			if tt.input != nil {
				input := tt.input

				if tt.compress {
					input = gzipped(t, input)
				}

				if err := os.WriteFile(p, input, 0o644); err != nil {
					t.Fatalf("os.WriteFile(%q) unexpected error: %v", p, err)
				}
			}

			rc, filename, err := Open(p)
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

			if rc == nil {
				t.Fatalf("Open(%q) returned nil reader without error", p)
			}

			t.Cleanup(func() { _ = rc.Close() })

			if filename == "" {
				t.Errorf("Open(%q) returned empty filename", p)
			} else if filename != p {
				if strings.Contains(p, filename) {
					t.Errorf("Open(%q) returned substring filename %q", p, filename)
				} else {
					t.Errorf("Open(%q) returned filename %q", p, filename)
				}
			}

			got, err := io.ReadAll(rc)
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

func TestReaderClose(t *testing.T) {
	tests := []struct {
		name     string
		compress bool
		path     func(string) string
	}{
		{name: "plain file", path: func(dir string) string {
			return filepath.Join(dir, "close.txt")
		}},
		{name: "gzipped file", compress: true, path: func(dir string) string {
			return filepath.Join(dir, "close.gzip")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.path(t.TempDir())
			d := []byte("whatever ts is")

			if tt.compress {
				d = gzipped(t, d)
			}

			if err := os.WriteFile(p, d, 0o644); err != nil {
				t.Fatalf("os.WriteFile(%q) unexpected error: %v", p, err)
			}

			rc, filename, err := Open(p)
			if err != nil {
				t.Fatalf("Open(%q) unexpected error: %v", p, err)
			}
			if rc == nil {
				t.Fatalf("Open(%q) returned nil reader without error", p)
			}

			t.Cleanup(func() { _ = rc.Close() })

			if filename == "" {
				t.Errorf("Open(%q) returned empty filename", p)
			} else if filename != p {
				if strings.Contains(p, filename) {
					t.Errorf("Open(%q) returned substring filename %q", p, filename)
				} else {
					t.Errorf("Open(%q) returned filename %q", p, filename)
				}
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

func gzipped(t *testing.T, b []byte) []byte {
	t.Helper()

	var compressedBuffer bytes.Buffer

	gw := gzip.NewWriter(&compressedBuffer)

	_, err := gw.Write(b)
	if err != nil {
		t.Fatalf("gw.Write(b) unexpected error: %v", err)
	}

	if err := gw.Close(); err != nil {
		t.Fatalf("gw.Close() unexpected error: %v", err)
	}

	return compressedBuffer.Bytes()
}
