package api

import (
	"net/url"
	"sync"

	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/config"
)

var (
	prepare   sync.Once
	webData   WebData
	loginData LoginData
)

func Prepare(conf config.Config) {
	prepare.Do(func() {
		loginURI, _ := url.Parse(conf.Origin)
		loginURI.Path = "/login"

		webData = WebData{
			Version: pkg.Version(),
		}

		loginData = LoginData{
			WebData:        webData,
			GoogleClientID: conf.Auth.GoogleClientID,
			LoginURI:       loginURI.String(),
		}
	})
}

type WebData struct {
	Version string
}

func GetWebData() WebData {
	return webData
}

type LoginData struct {
	WebData
	GoogleClientID string
	LoginURI       string
}

func GetLoginData() LoginData {
	return loginData
}

type InfoDetail struct {
	WebData
	Info *ShortURL
}

func (s *ShortURL) WebData() InfoDetail {
	return InfoDetail{
		WebData: GetWebData(),
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
		WebData: GetWebData(),
		URLs:    s.URLs,
		Page:    s.Page,
	}
}

type Stats struct {
	WebData
	Info *ShortcrustInfo
}

func (s *ShortcrustInfo) WebData() Stats {
	return Stats{
		WebData: GetWebData(),
		Info:    s,
	}
}
