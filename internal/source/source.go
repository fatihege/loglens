package source

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

var gzipMagic = []byte{0x1f, 0x8b}

type stream struct {
	io.Reader
	close func() error
}

func (s *stream) Close() error {
	return s.close()
}

func Open(path string) (io.ReadCloser, error) {
	if path == "-" {
		return &stream{Reader: os.Stdin, close: func() error { return nil }}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err // os.PathError already contains enough information
	}

	success := false
	defer func() {
		if !success {
			_ = file.Close()
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("gather file stat: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%s: %w", path, ErrPathIsDir)
	}

	br := bufio.NewReader(file)
	header, err := br.Peek(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("peek file: %w", err)
	}

	if bytes.HasPrefix(header, gzipMagic) {
		// the file is gz
		rc, err := openGzip(file, br)
		if err != nil {
			return nil, err
		}

		success = true
		return rc, nil
	}

	success = true
	return &stream{Reader: br, close: file.Close}, nil
}

func openGzip(file *os.File, br *bufio.Reader) (io.ReadCloser, error) {
	gr, err := gzip.NewReader(br)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}

	return &stream{Reader: gr, close: func() error {
		return errors.Join(gr.Close(), file.Close())
	}}, nil
}
