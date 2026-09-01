package lines

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type Iter struct {
	br     *bufio.Reader
	name   string
	num    int
	sticky error
}

func New(r io.Reader, name string, max int) *Iter {
	br := bufio.NewReaderSize(r, max)
	return &Iter{
		br:   br,
		name: name,
	}
}

func (i *Iter) Next() ([]byte, error) {
	if i.sticky != nil {
		return nil, i.sticky
	}

	line, err := i.br.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		for errors.Is(err, bufio.ErrBufferFull) {
			_, err = i.br.ReadSlice('\n')
		}

		if err != nil {
			i.sticky = err
		}

		i.num++

		return nil, ErrTooLong
	} else if errors.Is(err, io.EOF) {
		i.sticky = err
		if len(line) < 1 {
			return nil, err
		}
	} else if err != nil {
		i.sticky = err
		return nil, err
	}

	line = bytes.TrimRight(line, "\n\r")
	i.num++

	return line, nil
}

func (i *Iter) Name() string {
	return i.name
}

func (i *Iter) Num() int {
	return i.num
}
