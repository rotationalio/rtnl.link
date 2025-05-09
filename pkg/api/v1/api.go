package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

//===========================================================================
// Service Interface
//===========================================================================

type Service interface {
	// Unauthenticated endpoints
	Status(context.Context) (*StatusReply, error)

	// URL Management
	// TODO: edit short url with details
	ShortURLList(context.Context, *PageQuery) (*ShortURLList, error)
	ShortenURL(context.Context, *LongURL) (*ShortURL, error)
	ShortURLInfo(context.Context, string) (*ShortURL, error)
	DeleteShortURL(context.Context, string) error

	// Stats/Info

	// Campaigns
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Returned on status requests.
type StatusReply struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// PageQuery manages paginated list requests.
type PageQuery struct {
	PageSize      int    `json:"page_size" url:"page_size,omitempty" form:"page_size"`
	PrevPageToken string `json:"prev_page_token" url:"prev_page_token,omitempty" form:"prev_page_token"`
	NextPageToken string `json:"next_page_token" url:"next_page_token,omitempty" form:"next_page_token"`
}

// LoginForm is used for Google to submit an id token back to the server.
type LoginForm struct {
	Credential string `json:"credential" url:"credential" form:"credential"`
	Next       string `json:"next" url:"next" form:"next"`
}

//===========================================================================
// URL Shortening Endpoints
//===========================================================================

type LongURL struct {
	URL     string `json:"url" form:"url"`
	Expires string `json:"expires,omitempty" form:"expires"`
}

// TODO: change campaign uint64s to links
type ShortURL struct {
	URL         string     `json:"url"`
	AltURL      string     `json:"alt_url,omitempty"`
	Target      string     `json:"target,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Visits      uint64     `json:"visits"`
	Expires     *time.Time `json:"expires,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	Modified    *time.Time `json:"modified,omitempty"`
	CampaignID  uint64     `json:"campaign_id,omitempty"`
	Campaigns   []uint64   `json:"campaigns,omitempty"`
}

type ShortURLList struct {
	URLs []*ShortURL `json:"urls"`
	Page *PageQuery  `json:"page"`
}

//===========================================================================
// API Input Validation
//===========================================================================

func (p *PageQuery) Validate() error {
	if p.PrevPageToken != "" && p.NextPageToken != "" {
		return ErrForwardsBackwards
	}
	return nil
}

func (u *LongURL) Validate() error {
	u.URL = strings.TrimSpace(u.URL)
	u.Expires = strings.TrimSpace(u.Expires)

	if u.URL == "" {
		return ErrMissingURL
	}

	if u.Expires != "" {
		ts, err := u.ExpiresAt()
		if err != nil {
			return err
		}

		if !ts.After(time.Now()) {
			return ErrInvalidExpires
		}
	}

	return nil
}

var dateFormats = []string{
	time.RFC3339,
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05Z",
}

func (u *LongURL) ExpiresAt() (time.Time, error) {
	if u.Expires == "" {
		return time.Time{}, nil
	}

	for _, layout := range dateFormats {
		if ts, err := time.Parse(layout, u.Expires); err == nil {
			return ts, nil
		}
	}

	return time.Time{}, ErrCannotParseExpires
}

func (u *ShortURL) InfoURL() string {
	if strings.HasPrefix(u.URL, "http") {
		result, _ := url.JoinPath(u.URL, "info")
		return result
	}
	return fmt.Sprintf("/%s/info", u.URL)
}

func (u *ShortURL) DeleteURL() string {
	if strings.HasPrefix(u.URL, "http") {
		return u.URL
	}
	return fmt.Sprintf("/%s", u.URL)
}

//===========================================================================
// Info Endpoints
//===========================================================================

type ShortcrustInfo struct {
	Links     uint64 `json:"links"`
	Clicks    uint64 `json:"clicks"`
	Campaigns uint64 `json:"campaigns"`
}

func (s *ShortcrustInfo) CampaignsPerLink() float64 {
	if s.Campaigns == 0 {
		return 0.0
	}
	return float64(s.Campaigns) / float64(s.Links)
}
