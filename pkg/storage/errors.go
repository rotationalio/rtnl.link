package storage

import "errors"

var (
	ErrNotFound      = errors.New("object not found in database")
	ErrAlreadyExists = errors.New("object already exists in the database")
)
