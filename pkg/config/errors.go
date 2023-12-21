package config

import "errors"

var (
	ErrInvalidEnsignCredentials = errors.New("invalid configuration: must specify either path to ensign credentials or the client id and api key")
)
