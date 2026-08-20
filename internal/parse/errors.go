package parse

import "errors"

var (
	ErrEmpty               = errors.New("empty data")
	ErrNull                = errors.New("null data")
	ErrNaN                 = errors.New("NaN")
	ErrInf                 = errors.New("infinite number")
	ErrParseInt            = errors.New("could not parse integer")
	ErrIntOutOfRange       = errors.New("integer out of range")
	ErrParseString         = errors.New("could not parse string")
	ErrParseTime           = errors.New("could not parse time")
	ErrParseDuration       = errors.New("could not parse duration")
	ErrNegativeDuration    = errors.New("negative duration")
	ErrDurationTooLarge    = errors.New("duration too large")
	ErrUnknownDurationUnit = errors.New("unknown duration unit")
	ErrMalformedLine       = errors.New("malformed line")
)
