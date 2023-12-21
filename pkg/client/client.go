package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/websocket"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
)

// New creates a new API v1 client that implements the Service interface.
func New(endpoint, apiKey string) (_ api.Service, err error) {
	c := &APIv1{
		client: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Timeout:       30 * time.Second,
		},
		apiKey: apiKey,
	}

	if c.endpoint, err = url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("could not parse endpoint: %s", err)
	}

	if c.client.Jar, err = cookiejar.New(nil); err != nil {
		return nil, fmt.Errorf("could not create cookiejar: %w", err)
	}

	return c, nil
}

// APIv1 implements the Service interface
type APIv1 struct {
	endpoint *url.URL     // the base url for all requests
	apiKey   string       // the API key for authorized requests
	client   *http.Client // used to make http requests to the server
}

// Ensure the APIv1 implements the Service interface
var _ api.Service = &APIv1{}

//===========================================================================
// Client Methods
//===========================================================================

func (c *APIv1) Status(ctx context.Context) (out *api.StatusReply, err error) {
	// Make the HTTP request
	var req *http.Request
	if req, err = c.NewRequest(ctx, http.MethodGet, "/v1/status", nil, nil); err != nil {
		return nil, err
	}

	// NOTE: we cannot use s.Do because we want to parse 503 Unavailable errors
	var rep *http.Response
	if rep, err = c.client.Do(req); err != nil {
		return nil, err
	}
	defer rep.Body.Close()

	// Detect other errors
	if rep.StatusCode != http.StatusOK && rep.StatusCode != http.StatusServiceUnavailable {
		return nil, fmt.Errorf("%s", rep.Status)
	}

	// Deserialize the JSON data from the response
	out = &api.StatusReply{}
	if err = json.NewDecoder(rep.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("could not deserialize status reply: %s", err)
	}
	return out, nil
}

func (c *APIv1) ShortenURL(ctx context.Context, in *api.LongURL) (out *api.ShortURL, err error) {
	var req *http.Request
	if req, err = c.NewRequest(ctx, http.MethodPost, "/v1/shorten", in, nil); err != nil {
		return nil, err
	}

	if _, err = c.Do(req, &out, true); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *APIv1) ShortURLInfo(ctx context.Context, id string) (out *api.ShortURL, err error) {
	endpoint := fmt.Sprintf("/v1/links/%s", id)

	var req *http.Request
	if req, err = c.NewRequest(ctx, http.MethodGet, endpoint, nil, nil); err != nil {
		return nil, err
	}

	if _, err = c.Do(req, &out, true); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *APIv1) DeleteShortURL(ctx context.Context, id string) (err error) {
	endpoint := fmt.Sprintf("/v1/links/%s", id)

	var req *http.Request
	if req, err = c.NewRequest(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return err
	}

	if _, err = c.Do(req, nil, true); err != nil {
		return err
	}

	return nil
}

func (c *APIv1) ShortURLList(ctx context.Context, page *api.PageQuery) (out *api.ShortURLList, err error) {
	var params *url.Values
	if page != nil {
		var values url.Values
		if values, err = query.Values(page); err != nil {
			return nil, fmt.Errorf("could not encode query params: %w", err)
		}
		params = &values
	}

	var req *http.Request
	if req, err = c.NewRequest(ctx, http.MethodGet, "/v1/links", nil, params); err != nil {
		return nil, err
	}

	if _, err = c.Do(req, &out, true); err != nil {
		return nil, err
	}

	return out, nil
}

// TODO: figure out how to gracefully close the connection!
func (c *APIv1) Updates(ctx context.Context, id string) (_ <-chan *api.Click, err error) {
	path := "/v1/updates"
	if id != "" {
		path = fmt.Sprintf("/v1/%s/updates", id)
	}

	// Set the headers on the request
	header := http.Header{}
	header.Add("User-Agent", userAgent)
	header.Add("Accept", accept)
	header.Add("Accept-Language", acceptLang)
	header.Add("Accept-Encoding", acceptEncode)
	header.Add("Content-Type", contentType)

	// Add API Key if available
	if c.apiKey != "" {
		header.Add("Authorization", "Bearer "+c.apiKey)
	}

	// Create the websockets endpoint
	endpoint := c.endpoint.ResolveReference(&url.URL{Path: path})
	endpoint.Scheme = "ws"

	conn, _, err := websocket.DefaultDialer.Dial(endpoint.String(), header)
	if err != nil {
		return nil, err
	}

	updates := make(chan *api.Click, 10)

	// Start the reader go routine
	go func(updates chan<- *api.Click) {
		defer conn.Close()
		defer close(updates)

		for {
			click := &api.Click{}
			if err = conn.ReadJSON(click); err != nil {
				return
			}
			updates <- click
		}

	}(updates)

	return updates, nil
}

//===========================================================================
// Helper Methods
//===========================================================================

const (
	userAgent    = "Rotational Golang API Client/v1"
	accept       = "application/json"
	acceptLang   = "en-US,en"
	acceptEncode = "gzip, deflate, br"
	contentType  = "application/json; charset=utf-8"
)

func (s *APIv1) NewRequest(ctx context.Context, method, path string, data interface{}, params *url.Values) (req *http.Request, err error) {
	// Resolve the URL reference from the path
	url := s.endpoint.ResolveReference(&url.URL{Path: path})
	if params != nil && len(*params) > 0 {
		url.RawQuery = params.Encode()
	}

	var body io.ReadWriter
	switch {
	case data == nil:
		body = nil
	default:
		body = &bytes.Buffer{}
		if err = json.NewEncoder(body).Encode(data); err != nil {
			return nil, fmt.Errorf("could not serialize request data as json: %s", err)
		}
	}

	// Create the http request
	if req, err = http.NewRequestWithContext(ctx, method, url.String(), body); err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	// Set the headers on the request
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Accept", accept)
	req.Header.Add("Accept-Language", acceptLang)
	req.Header.Add("Accept-Encoding", acceptEncode)
	req.Header.Add("Content-Type", contentType)

	// Add API Key if available
	if s.apiKey != "" {
		req.Header.Add("Authorization", "Bearer "+s.apiKey)
	}

	// Add CSRF protection if its available
	if s.client.Jar != nil {
		cookies := s.client.Jar.Cookies(url)
		for _, cookie := range cookies {
			if cookie.Name == "csrf_token" {
				req.Header.Add("X-CSRF-TOKEN", cookie.Value)
			}
		}
	}
	return req, nil
}

// Do executes an http request against the server, performs error checking, and
// deserializes the response data into the specified struct.
func (s *APIv1) Do(req *http.Request, data interface{}, checkStatus bool) (rep *http.Response, err error) {
	if rep, err = s.client.Do(req); err != nil {
		return rep, fmt.Errorf("could not execute request: %s", err)
	}
	defer rep.Body.Close()

	// Detect http status errors if they've occurred
	if checkStatus {
		if rep.StatusCode < 200 || rep.StatusCode >= 300 {
			// Attempt to read the error response from JSON, if available
			serr := &StatusError{
				StatusCode: rep.StatusCode,
				Reply:      api.Reply{},
			}

			if err = json.NewDecoder(rep.Body).Decode(&serr.Reply); err == nil {
				return rep, serr
			}

			serr.Reply = api.Reply{Error: "something went wrong"}
			return rep, serr
		}
	}

	// Deserialize the JSON data from the body
	if data != nil && rep.StatusCode >= 200 && rep.StatusCode < 300 && rep.StatusCode != http.StatusNoContent {
		ct := rep.Header.Get("Content-Type")
		if ct != "" {
			mt, _, err := mime.ParseMediaType(ct)
			if err != nil {
				return nil, fmt.Errorf("malformed content-type header: %w", err)
			}

			if mt != accept {
				return nil, fmt.Errorf("unexpected content type: %q", mt)
			}
		}

		if err = json.NewDecoder(rep.Body).Decode(data); err != nil {
			return nil, fmt.Errorf("could not deserialize response data: %s", err)
		}
	}

	return rep, nil
}
