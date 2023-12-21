package stream

import (
	"errors"

	"github.com/rotationalio/go-ensign"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
)

// In maintenance mode the analytics stream simply does nothing.
type noop struct{}

func (s *noop) Click(*api.Click)        {}
func (s *noop) Shortened(*api.ShortURL) {}
func (s *noop) Deleted(linkID string)   {}
func (s *noop) Close() error            { return nil }

func (s *noop) Subscribe() (*ensign.Subscription, error) {
	return nil, errors.New("no ensign connection configured")
}
