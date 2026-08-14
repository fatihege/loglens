package source

import "errors"

var (
	ErrPathIsDir     = errors.New("provided path is a directory")
	ErrTerminalInput = errors.New("stdin is a terminal, not a pipe")
)
