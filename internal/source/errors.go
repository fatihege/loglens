package source

import "errors"

var (
	ErrPathIsDir = errors.New("provided path is a directory")
)
