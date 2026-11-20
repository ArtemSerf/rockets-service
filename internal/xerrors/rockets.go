package xerrors

import "errors"

var (
	ErrInvalidEvent = errors.New("invalid event")
	ErrInvalidQuery = errors.New("invalid query")
	ErrNotFound     = errors.New("rocket not found")
)
