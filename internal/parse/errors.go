package parse

import "errors"

var (
	ErrEmpty               = errors.New("empty")
	ErrNull                = errors.New("null")
	ErrNaN                 = errors.New("NaN")
	ErrInf                 = errors.New("infinite number")
	ErrParseInt            = errors.New("could not parse integer")
	ErrIntOutOfRange       = errors.New("integer out of range")
	ErrNotString           = errors.New("not a string")
	ErrParseString         = errors.New("could not parse string")
	ErrParseTime           = errors.New("could not parse time")
	ErrParseDuration       = errors.New("could not parse duration")
	ErrNegativeDuration    = errors.New("negative duration")
	ErrDurationTooLarge    = errors.New("duration too large")
	ErrUnknownDurationUnit = errors.New("unknown duration unit")
	ErrMalformedLine       = errors.New("malformed line")
	ErrUnrecognized        = errors.New("unrecognized JSON")
)
