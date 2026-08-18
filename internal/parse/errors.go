package parse

import "errors"

var (
	ErrEmptyRawMessage = errors.New("empty raw message")
	ErrNull            = errors.New("data is null")
	ErrNotInt          = errors.New("received data is not integer")
	ErrNotString       = errors.New("received data is not string")
	ErrParseTime       = errors.New("could not parse time string with any known layout")
)
