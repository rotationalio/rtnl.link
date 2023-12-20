package rtnl

import "github.com/rotationalio/rtnl.link/pkg"

func NewWebData() WebData {
	return WebData{
		Version: pkg.Version(),
	}
}

type WebData struct {
	Version string
}
