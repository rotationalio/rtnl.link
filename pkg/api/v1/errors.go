package api

import (
	"encoding/json"
	"fmt"
)

var (
	ErrUnsuccessfulReply = Reply{Success: false}
	ErrNotFoundReply     = Reply{Success: false, Error: "resource not found"}
	ErrNotAllowedReply   = Reply{Success: false, Error: "method not allowed"}
)

// Construct a new response for an error or simply return unsuccessful.
func ErrorResponse(err interface{}) Reply {
	if err == nil {
		return ErrUnsuccessfulReply
	}

	rep := Reply{Success: false}
	switch err := err.(type) {
	case error:
		rep.Error = err.Error()
	case string:
		rep.Error = err
	case fmt.Stringer:
		rep.Error = err.String()
	case json.Marshaler:
		data, e := err.MarshalJSON()
		if e != nil {
			panic(err)
		}
		rep.Error = string(data)
	default:
		rep.Error = "unhandled error response"
	}

	return rep
}