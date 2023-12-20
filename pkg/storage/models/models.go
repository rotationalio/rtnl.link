package models

type Model interface {
	Key() []byte
	MarshalValue() ([]byte, error)
	UnmarshalValue(data []byte) error
}

// A bucket is a 4 byte prefix that allows us to determine the object type by the key.
type Bucket [4]byte

// Buckets in use by the models in rtnl.link (appropriate emojis in unicode)
var (
	LinksBucket    = Bucket{240, 159, 148, 151}
	APIKeysBucket  = Bucket{240, 159, 148, 145}
	CampaignBucket = Bucket{240, 159, 142, 186}
)
