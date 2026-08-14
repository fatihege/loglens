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
	var file *os.File
	var err error
	var closeFn func() error
	stdin := false

	if path == "-" {
		file = os.Stdin
		stdin = true
		closeFn = func() error { return nil }
	} else {
		file, err = os.Open(path)
		if err != nil {
			return nil, err // os.PathError already contains enough information
		}

		closeFn = file.Close
	}

	success := false
	defer func() {
		if !success {
			closeFn()
		}
	}()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("gather stat for %s: %w", path, err)
	}

	if !stdin && fi.IsDir() {
		return nil, fmt.Errorf("%s: %w", path, ErrPathIsDir)
	}

	if stdin && fi.Mode()&os.ModeCharDevice != 0 {
		return nil, ErrTerminalInput
	}

	br := bufio.NewReader(file)
	header, err := br.Peek(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("peek %s: %w", path, err)
	}

	if bytes.HasPrefix(header, gzipMagic) {
		// the file is gz
		rc, err := openGzip(br, closeFn)
		if err != nil {
			return nil, err
		}

		success = true
		return rc, nil
	}

	success = true

	return &stream{Reader: br, close: closeFn}, nil
}

func openGzip(br *bufio.Reader, closeFn func() error) (io.ReadCloser, error) {
	gr, err := gzip.NewReader(br)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}

	return &stream{Reader: gr, close: func() error {
		return errors.Join(gr.Close(), closeFn())
	}}, nil
}
