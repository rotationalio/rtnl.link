package api

import (
	"github.com/rotationalio/rtnl.link/pkg"
)

type WebData struct {
	Version string
}

func NewWebData() WebData {
	return WebData{
		Version: pkg.Version(),
	}
}

type InfoDetail struct {
	WebData
	Info *ShortURL
}

func (s *ShortURL) WebData() InfoDetail {
	return InfoDetail{
		WebData: NewWebData(),
		Info:    s,
	}
}

type LinkList struct {
	WebData
	URLs []*ShortURL
	Page *PageQuery
}

func (s *ShortURLList) WebData() LinkList {
	return LinkList{
		WebData: NewWebData(),
		URLs:    s.URLs,
		Page:    s.Page,
	}
}
