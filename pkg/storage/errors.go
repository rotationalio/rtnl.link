package storage

import "errors"

var (
	ErrNotFound = errors.New("object not found in database")
)
