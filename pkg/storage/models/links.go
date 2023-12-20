package models

import (
	"encoding/binary"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type ShortURL struct {
	ID          uint64    `msgpack:"id"`
	URL         string    `msgpack:"url"`
	Title       string    `msgpack:"title"`
	Description string    `msgpack:"description"`
	Expires     time.Time `msgpack:"expires"`
	Visits      uint64    `msgpack:"visits"`
	Created     time.Time `msgpack:"created"`
	Modified    time.Time `msgpack:"modified"`
	CreatedBy   string    `msgpack:"created_by"`
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
