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

var _ Model = &APIKey{}

func (m *APIKey) Key() []byte {
	data, _ := base64.RawStdEncoding.DecodeString(m.ClientID)
	key := make([]byte, len(data)+4)
	copy(key[0:4], APIKeysBucket[:])
	copy(key[4:], data)
	return key
}

func (m *APIKey) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *APIKey) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}
