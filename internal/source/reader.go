package source

import (
	"fmt"
	"os"
)

type Reader struct {
	filepath string
}

func NewReader(filepath string) *Reader {
	return &Reader{
		filepath: filepath,
	}
}

func (r *Reader) Read() (string, error) {
	file, err := os.Open(r.filepath)
	if err != nil {
		return "", fmt.Errorf("open file: %v", err)
	}

	b := make([]byte, 5)
	n, err := file.Read(b)
	if err != nil {
		return "", fmt.Errorf("read file: %v", err)
	}

	return fmt.Sprintf("b: %d, n: %d\n", b, n), nil
}
