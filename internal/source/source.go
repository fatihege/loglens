package source

import (
	"fmt"
	"io"
	"os"
)

func Open(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return file, fmt.Errorf("gather file stat: %w", err)
	}

	if fileInfo.IsDir() {
		return file, fmt.Errorf("%s: %w", path, ErrPathIsDir)
	}

	return file, nil
}
