package api

import "context"

//===========================================================================
// Service Interface
//===========================================================================

type QuarterdeckClient interface {
	// Unauthenticated endpoints
	Status(context.Context) (*StatusReply, error)
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Unverified bool   `json:"unverified,omitempty"`
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
	NextPageToken string `json:"next_page_token" url:"next_page_token,omitempty" form:"next_page_token"`
}
