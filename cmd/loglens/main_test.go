package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatihege/loglens/internal/parse"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		input      string
		nilStdin   bool
		want       int
		wantStderr string
		wantStdout string
	}{
		{
			name:       "incorrect flag type",
			args:       []string{"-top", "five"},
			nilStdin:   true,
			want:       2,
			wantStderr: "invalid value",
		},
		{
			name:       "nil stdin",
			args:       []string{"-"},
			nilStdin:   true,
			want:       2,
			wantStderr: ErrNoInput.Error(),
		},
		{
			name:       "piped stdin",
			args:       []string{"-"},
			input:      "go is the\nperfect\nlanguage",
			want:       0,
			wantStdout: "read lines 3",
			wantStderr: parse.ErrMalformedLine.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input io.Reader
			var outBuffer bytes.Buffer
			var errBuffer bytes.Buffer

			if !tt.nilStdin {
				input = strings.NewReader(tt.input)
			}

			if got := run(tt.args, input, &outBuffer, &errBuffer); got != tt.want {
				t.Errorf("run(%q) = %d, want %d", tt.args, got, tt.want)
			}

			if tt.wantStderr != "" {
				if !strings.Contains(errBuffer.String(), tt.wantStderr) {
					t.Errorf("run(%q) stderr = %q, want substring %q", tt.args, errBuffer.String(), tt.wantStderr)
				}
			} else if errBuffer.String() != "" {
				t.Errorf("run(%q) unexpected stderr: %v", tt.args, errBuffer.String())
			}

			if tt.wantStdout != "" {
				if !strings.Contains(outBuffer.String(), tt.wantStdout) {
					t.Errorf("run(%q) stdout = %q, want substring %q", tt.args, outBuffer.String(), tt.wantStdout)
				}
			} else if outBuffer.String() != "" {
				t.Errorf("run(%q) stdout = %q, want empty", tt.args, outBuffer.String())
			}
		})
	}
}

func TestRunFile(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		file       string
		real       bool
		want       int
		wantStdout string
		wantStderr bool // done this way because fs.ErrNotExist is os-spesific
	}{
		{
			name:       "real file",
			input:      []byte("hell yeah"),
			file:       "real_asl.txt",
			real:       true,
			want:       0,
			wantStdout: "read lines 1",
			wantStderr: true,
		},
		{
			name:       "non-existing path",
			file:       "there_aint_nothing",
			want:       1,
			wantStderr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), tt.file)

			if tt.real {
				if err := os.WriteFile(p, tt.input, 0o644); err != nil {
					t.Fatalf("os.WriteFile(%q) unexpected error: %v", p, err)
				}
			}

			args := []string{p}
			var outBuffer bytes.Buffer
			var errBuffer bytes.Buffer

			if got := run(args, nil, &outBuffer, &errBuffer); got != tt.want {
				t.Errorf("run(%q) = %d, want %d", args, got, tt.want)
			}

			if tt.wantStderr {
				if errBuffer.String() == "" {
					t.Errorf("run(%q) stderr empty, want non-empty", args)
				}
			} else if errBuffer.String() != "" {
				t.Errorf("run(%q) unexpected stderr: %v", args, errBuffer.String())
			}

			if tt.wantStdout != "" {
				if !strings.Contains(outBuffer.String(), tt.wantStdout) {
					t.Errorf("run(%q) stdout = %q, want substring %q", args, outBuffer.String(), tt.wantStdout)
				}
			} else if outBuffer.String() != "" {
				t.Errorf("run(%q) stdout = %q, want empty", args, outBuffer.String())
			}
		})
	}
}
