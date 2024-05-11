package models

import "github.com/rotationalio/rtnl.link/pkg/api/v1"

type Counts struct {
	Links     uint64 `msgpack:"links"`
	Clicks    uint64 `msgpack:"clicks"`
	Campaigns uint64 `msgpack:"campaigns"`
}

func (c *Counts) ToAPI() *api.ShortcrustInfo {
	return &api.ShortcrustInfo{
		Links:     c.Links,
		Clicks:    c.Clicks,
		Campaigns: c.Campaigns,
	}
}
