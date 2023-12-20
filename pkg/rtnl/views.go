package rtnl

import (
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
)

func NewWebData() WebData {
	return WebData{
		Version: pkg.Version(),
	}
}

type WebData struct {
	Version string
}

type InfoDetail struct {
	WebData
	Info *api.ShortURL
}
