package models

type Model interface {
	Key() ([]byte, error)
	MarshalValue() ([]byte, error)
	UnmarshalValue(data []byte) error
}
