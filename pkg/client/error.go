package client

import (
	"fmt"

	"github.com/rotationalio/rtnl.link/pkg/api/v1"
)

// StatusError decodes an error response from the Service
type StatusError struct {
	StatusCode int
	Reply      api.Reply
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("[%d] %s", e.StatusCode, e.Reply.Error)
}
