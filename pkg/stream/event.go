package stream

import "github.com/rotationalio/go-ensign"

type Event interface {
	Event() *ensign.Event
}
