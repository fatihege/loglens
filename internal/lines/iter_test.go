package lines

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestIter(t *testing.T) {
	tests := []struct {
		name          string
		input         []string
		bufferSize    int
		expectedTotal int
		expectedValid int
		expectedSkip  int
		noNewLine     bool
	}{
		{
			name:          "ordinary content",
			input:         []string{"if you want to know more about", "anything in this world", "you can", "go fishing"},
			bufferSize:    16,
			expectedTotal: 4,
			expectedValid: 2,
			expectedSkip:  2,
		},
		{
			name:          "15, 16, 17",
			input:         []string{"123456789012345", "1234567890123456", "12345678901234567"},
			bufferSize:    16,
			expectedTotal: 3,
			expectedValid: 1,
			expectedSkip:  2,
		},
		{
			name:       "empty content",
			input:      nil,
			bufferSize: 16,
			noNewLine:  true,
		},
		{
			name:          "escape characters",
			input:         []string{"a\r", "b\r"},
			bufferSize:    16,
			expectedTotal: 2,
			expectedValid: 2,
		},
		{
			name:          "no trailing new line",
			input:         []string{"a", "b"},
			bufferSize:    16,
			expectedTotal: 2,
			expectedValid: 2,
			noNewLine:     true,
		},
		{
			name:          "over-long line at eof",
			input:         []string{"a", "b", "this will exceed the limit"},
			bufferSize:    16,
			expectedTotal: 3,
			expectedValid: 2,
			expectedSkip:  1,
			noNewLine:     true,
		},
		{
			name:          "empty line in the middle",
			input:         []string{"a", "", "b"},
			bufferSize:    16,
			expectedTotal: 3,
			expectedValid: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.Join(tt.input, "\n")

			if !tt.noNewLine {
				input += "\n"
			}

			r := strings.NewReader(input)
			name := "stdin"
			iter := New(r, name, tt.bufferSize) // bufio.Reader enforces 16-byte minimum
			count, skip := 0, 0

			if gotName := iter.Name(); gotName != name {
				t.Errorf("iter.Name() = %q, want %q", gotName, name)
			}

			for i := 0; ; i++ {
				if i > 1000 {
					t.Fatal("runaway")
				}

				line, err := iter.Next()
				if errors.Is(err, io.EOF) {
					break
				} else if errors.Is(err, ErrTooLong) {
					skip++
					continue
				} else if err != nil {
					t.Fatalf("iter.Next() unexpected error: %v", err)
				}

				if len(tt.input) > i && string(line) != strings.TrimRight(tt.input[i], "\n\r") {
					t.Errorf("iter.Next() = %q, want %q", line, strings.TrimRight(tt.input[i], "\n\r"))
				}

				if i >= len(tt.input) {
					t.Errorf("iter.Next() returned more lines than expected (line %d)", i+1)
				}

				if got := iter.Num(); got != i+1 {
					t.Errorf("iter.Num() = %d, want %d", got, i+1)
				}

				count++
			}

			total := count + skip

			if total != tt.expectedTotal {
				t.Errorf("got total lines of %d, want %d", total, tt.expectedTotal)
			}

			if count != tt.expectedValid {
				t.Errorf("got valid lines of %d, want %d", count, tt.expectedValid)
			}

			if skip != tt.expectedSkip {
				t.Errorf("got malformed lines of %d, want %d", skip, tt.expectedSkip)
			}
		})
	}
}

func TestIterFatalError(t *testing.T) {
	want := [2]string{"lorem", "ipsum"}
	r := &fatalReader{text: want[0] + "\n" + want[1] + "\n"}
	iter := New(r, "stdin", 16)

	for i := range 4 {
		line, err := iter.Next()
		if i == 0 || i == 1 {
			if err != nil {
				t.Fatalf("iter.Next() unexpected error: %v", err)
			}

			if string(line) != want[i] {
				t.Errorf("iter.Next() = %q, want %q", line, want[i])
			}
		} else if !errors.Is(err, errFatalReader) {
			t.Errorf("iter.Next() call %d returned error %v, want %v", i+1, err, errFatalReader)
		}
	}
}

var errFatalReader = errors.New("damn")

type fatalReader struct {
	text      string
	index     int
	triggered bool
}

func (r *fatalReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.text) {
		if !r.triggered {
			r.triggered = true
			return 0, errFatalReader
		}
		return 0, io.EOF
	}

	n = copy(p, r.text[r.index:])
	r.index += n

	return n, nil
}
