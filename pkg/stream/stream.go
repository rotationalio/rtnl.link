package stream

import (
	"errors"
	"io"
	"sync"

	"github.com/rotationalio/go-ensign"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	defaultBufferSize = 1024
)

type Stream interface {
	io.Closer
	Click(*api.Click)
	Shortened(*api.ShortURL)
	Deleted(linkID string)
	Subscribe() (*ensign.Subscription, error)
}

type AnalyticsStream struct {
	sync.RWMutex
	conf   config.EnsignConfig
	ensign *ensign.Client
	clicks chan<- Event
}

func New(conf config.EnsignConfig) (_ Stream, err error) {
	if err = conf.Validate(); err != nil {
		return nil, err
	}

	if conf.Maintenance {
		return &noop{}, nil
	}

	clicks := make(chan Event, defaultBufferSize)
	stream := &AnalyticsStream{
		conf:   conf,
		clicks: clicks,
	}

	if stream.ensign, err = ensign.New(conf.Options()); err != nil {
		return nil, err
	}

	go stream.Run(clicks)
	return stream, nil
}

func (s *AnalyticsStream) Click(click *api.Click) {
	s.RLock()
	defer s.RUnlock()
	if s.clicks != nil {
		s.clicks <- click
	}
}

func (s *AnalyticsStream) Shortened(in *api.ShortURL) {
	s.RLock()
	defer s.RUnlock()
	if s.clicks != nil {
		s.clicks <- &api.Shortened{
			URL:         in.URL,
			AltURL:      in.AltURL,
			Title:       in.Title,
			Description: in.Description,
			Expires:     in.Expires,
			CampaignID:  in.CampaignID,
			Campaigns:   in.Campaigns,
		}
	}
}

func (s *AnalyticsStream) Deleted(linkID string) {
	s.RLock()
	defer s.RUnlock()
	if s.clicks != nil {
		s.clicks <- &api.Deleted{
			LinkID:  linkID,
			Expired: false,
		}
	}
}

func (s *AnalyticsStream) Subscribe() (*ensign.Subscription, error) {
	s.RLock()
	defer s.RUnlock()
	if s.ensign != nil {
		return s.ensign.Subscribe(s.conf.Topic)
	}
	return nil, errors.New("analytics stream is not connected")
}

func (s *AnalyticsStream) Close() error {
	s.Lock()
	defer s.Unlock()

	if s.ensign != nil {
		return s.ensign.Close()
	}

	if s.clicks != nil {
		close(s.clicks)
		s.clicks = nil
	}
	return nil
}

func (s *AnalyticsStream) Run(events <-chan Event) {
	for event := range events {
		s.RLock()
		if s.ensign != nil {
			if err := s.ensign.Publish(s.conf.Topic, event.Event()); err != nil {
				log.Warn().Err(err).Msg("could not publish ensign event")
			}
		}
		s.RUnlock()
	}
}
