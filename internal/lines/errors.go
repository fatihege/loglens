package lines

import "errors"

var (
	ErrTooLong = errors.New("line exceeds maximum length")
)
