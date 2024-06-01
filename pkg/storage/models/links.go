package models

import (
	"encoding/binary"
	"time"

	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/vmihailenco/msgpack/v5"
)

// ShortURL represents a shortened link and the location to redirect the user to. The
// ShortURL also contains a basic visit counter and other metadata like title and
// description to make things easier to read on the front-end.
//
// A Campaign is a relationship between shortened URLs that have different marketing
// purposes. For example, we might shorten a webinar link then create campaign links
// for sendgrid, twitter, linkedin, etc. The purpose of the campaign is to identify what
// channels are performing best. In terms of the data structure, a short URL can either
// have a campaign id -- meaning it is a campaign link for another URL or it can have
// a list of campaigns, it's sublinks. Technically a tree-structure is possible, but in
// practice, short urls should have either campaign id or campaigns.
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
	CampaignID  uint64    `msgpack:"campaign_id"`
	Campaigns   []uint64  `msgpack:"campaigns"`
}

var _ Model = &ShortURL{}

func (m *ShortURL) Key() []byte {
	key := make([]byte, 12)
	copy(key[0:4], LinksBucket[:])
	binary.LittleEndian.PutUint64(key[4:], m.ID)
	return key
}

func (m *ShortURL) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *ShortURL) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}

// Creates an api.ShortURL object and populates it with the fields from the model that
// can be populated directly. Note that URL and AltURL cannot be directly populated
// without a configuration object.
func (m *ShortURL) ToAPI() *api.ShortURL {
	out := &api.ShortURL{
		Target:      m.URL,
		Title:       m.Title,
		Description: m.Description,
		Visits:      m.Visits,
		CampaignID:  m.CampaignID,
		Campaigns:   m.Campaigns,
	}

	if !m.Expires.IsZero() {
		out.Expires = &m.Expires
	}

	if !m.Created.IsZero() {
		out.Created = &m.Created
	}

	if !m.Modified.IsZero() {
		out.Modified = &m.Modified
	}

	return out
}
