package source

import (
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
	close  func() error
	closed bool
}

func (s *stream) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	return s.close()
}

func Open(path string) (io.ReadCloser, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err // os.PathError already contains enough information
	}

	success := false
	defer func() {
		if !success {
			file.Close()
		}
	}()

	fi, err := file.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("gather stat for %s: %w", path, err)
	}

	if fi.IsDir() {
		return nil, path, fmt.Errorf("%s: %w", path, ErrPathIsDir)
	}

	wrapped, err := Wrap(file)
	if err != nil {
		return nil, path, fmt.Errorf("%s: %w", path, err)
	}

	success = true
	return &stream{Reader: wrapped, close: func() error {
		return errors.Join(wrapped.Close(), file.Close())
	}}, path, nil
}

func Wrap(r io.Reader) (io.ReadCloser, error) {
	header := make([]byte, 2)
	n, err := io.ReadFull(r, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("peek: %w", err)
	}

	mr := io.MultiReader(bytes.NewReader(header[:n]), r)

	if bytes.HasPrefix(header, gzipMagic) {
		// the file is gz
		gr, err := gzip.NewReader(mr)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}

		return &stream{Reader: gr, close: gr.Close}, nil
	}

	return &stream{Reader: mr, close: func() error { return nil }}, nil
}
