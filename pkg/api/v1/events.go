package api

import (
	"net/url"
	"strings"
	"time"

	"github.com/rotationalio/go-ensign"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	mimetype "github.com/rotationalio/go-ensign/mimetype/v1beta1"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/vmihailenco/msgpack/v5"
)

const EventMime = mimetype.ApplicationMsgPack

var ClickType = &api.Type{
	Name:         "Click",
	MajorVersion: pkg.VersionMajor,
	MinorVersion: pkg.VersionMinor,
	PatchVersion: pkg.VersionPatch,
}

type Click struct {
	URL       string `json:"url" msgpack:"url"`
	Time      string `json:"time" msgpack:"time"`
	Views     int    `json:"views" msgpack:"views"`
	UserAgent string `json:"user_agent" msgpack:"user_agent"`
	IPAddr    string `json:"ip_address" msgpack:"ip_address"`
}

func (c *Click) Event() *ensign.Event {
	meta := make(ensign.Metadata)
	meta["id"] = c.LinkID()

	data, _ := c.MarshalValue()

	return &ensign.Event{
		Metadata: meta,
		Data:     data,
		Type:     ClickType,
		Mimetype: EventMime,
	}
}

func (m *Click) LinkID() string {
	u, _ := url.Parse(m.URL)
	return strings.Trim(u.Path, "/")
}

func (m *Click) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *Click) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}

var ShortenedType = &api.Type{
	Name:         "Shortened",
	MajorVersion: pkg.VersionMajor,
	MinorVersion: pkg.VersionMinor,
	PatchVersion: pkg.VersionPatch,
}

type Shortened struct {
	URL         string     `msgpack:"url"`
	AltURL      string     `msgpack:"alt_url"`
	Title       string     `msgpack:"title"`
	Description string     `msgpack:"description"`
	Expires     *time.Time `msgpack:"expires,omitempty"`
	CampaignID  uint64     `msgpack:"campaign_id,omitempty"`
	Campaigns   []uint64   `msgpack:"campaigns,omitempty"`
}

func (c *Shortened) Event() *ensign.Event {
	meta := make(ensign.Metadata)
	meta["id"] = c.LinkID()
	if c.CampaignID != 0 {
		meta["campaign_id"] = base62.Encode(c.CampaignID)
	}

	data, _ := c.MarshalValue()

	return &ensign.Event{
		Metadata: meta,
		Data:     data,
		Type:     ShortenedType,
		Mimetype: EventMime,
	}
}

func (m *Shortened) LinkID() string {
	u, _ := url.Parse(m.URL)
	return strings.Trim(u.Path, "/")
}

func (m *Shortened) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *Shortened) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}

var DeletedType = &api.Type{
	Name:         "Deleted",
	MajorVersion: pkg.VersionMajor,
	MinorVersion: pkg.VersionMinor,
	PatchVersion: pkg.VersionPatch,
}

type Deleted struct {
	LinkID  string `msgpack:"link_id"`
	Expired bool   `msgpack:"expired"`
}

func (c *Deleted) Event() *ensign.Event {
	meta := make(ensign.Metadata)
	meta["id"] = c.LinkID

	data, _ := c.MarshalValue()

	return &ensign.Event{
		Metadata: meta,
		Data:     data,
		Type:     DeletedType,
		Mimetype: EventMime,
	}
}

func (m *Deleted) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *Deleted) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}
