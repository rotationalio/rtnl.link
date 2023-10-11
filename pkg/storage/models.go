package storage

import (
	"encoding/base64"
	"encoding/binary"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type ShortURL struct {
	ID        uint64    `msgpack:"id"`
	URL       string    `msgpack:"url"`
	Expires   time.Time `msgpack:"expires"`
	Visits    uint64    `msgpack:"visits"`
	Created   time.Time `msgpack:"created"`
	Modified  time.Time `msgpack:"modified"`
	CreatedBy string    `msgpack:"created_by"`
}

func (m *ShortURL) Key() []byte {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, m.ID)
	return key
}

func (m *ShortURL) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *ShortURL) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}

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
