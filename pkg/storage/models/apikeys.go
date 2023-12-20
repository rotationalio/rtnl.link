package models

import (
	"encoding/base64"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type APIKey struct {
	ClientID   string    `msgpack:"client_id"`
	DerivedKey string    `msgpack:"derived_key"`
	Created    time.Time `msgpack:"created"`
	Modified   time.Time `msgpack:"modified"`
}

func (m *APIKey) Key() ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(m.ClientID)
}

func (m *APIKey) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *APIKey) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}
