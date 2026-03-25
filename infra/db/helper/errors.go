package helper

import "errors"

var (
	ErrTooManyIDs       = errors.New("too many ids")
	ErrTooManyAuthors   = errors.New("too many authors")
	ErrTooManyKinds     = errors.New("too many kinds")
	ErrTooManyTagValues = errors.New("too many tag values")
	ErrEmptyTagSet      = errors.New("empty tag set")
)
