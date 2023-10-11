package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/client"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rotationalio/rtnl.link/pkg/rtnl"
	"github.com/urfave/cli/v2"
)

var (
	conf config.Config
	svc  api.Service

	timeout = 30 * time.Second
)

func main() {
	// If a dotenv file exists load it for configuration
	godotenv.Load()

	// Create a multi-command CLI application
	app := cli.NewApp()
	app.Name = "rtnl"
	app.Version = pkg.Version()
	app.Usage = "utilities and administrative commands rtnl.link"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "endpoint",
			Aliases: []string{"e"},
			Usage:   "specify the endpoint to the shortener service",
			Value:   "https://rtnl.link",
			EnvVars: []string{"RTNL_ENDPOINT"},
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:     "serve",
			Category: "server",
			Usage:    "run the rtnl server",
			Action:   serve,
			Before:   configure,
			Flags:    []cli.Flag{},
		},
		{
			Name:      "shorten",
			Category:  "client",
			Usage:     "create a short url from a long url",
			ArgsUsage: "url [url ...]",
			Action:    shorten,
			Before:    makeClient,
			Flags: []cli.Flag{
				&cli.TimestampFlag{
					Name:    "expires",
					Aliases: []string{"E"},
					Usage:   "specify a timestamp for the URL to expire",
					Layout:  time.RFC3339,
				},
				&cli.DurationFlag{
					Name:    "ttl",
					Aliases: []string{"t"},
					Usage:   "specify a time to live for the shortened url",
				},
			},
		},
		{
			Name:      "info",
			Category:  "client",
			Usage:     "get info about short url usage",
			ArgsUsage: "urlID [urlID ...]",
			Action:    info,
			Before:    makeClient,
			Flags:     []cli.Flag{},
		},
		{
			Name:      "delete",
			Category:  "client",
			Usage:     "delete a short url so that it is no longer used",
			ArgsUsage: "urlID [urlID ...]",
			Action:    delete,
			Before:    makeClient,
			Flags:     []cli.Flag{},
		},
		{
			Name:     "status",
			Category: "client",
			Usage:    "check on th status of the shortener service",
			Action:   status,
			Before:   makeClient,
			Flags:    []cli.Flag{},
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(2)
	}
}

//===========================================================================
// Server Commands
//===========================================================================

func serve(c *cli.Context) (err error) {
	var srv *rtnl.Server
	if srv, err = rtnl.New(conf); err != nil {
		return cli.Exit(err, 1)
	}

	if err = srv.Serve(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

//===========================================================================
// Client Commands
//===========================================================================

func shorten(c *cli.Context) (err error) {
	if c.NArg() == 0 {
		return cli.Exit("specify at least one url to shorten", 1)
	}

	expiresAt := c.Timestamp("expires")
	ttl := c.Duration("ttl")

	if expiresAt != nil && !expiresAt.IsZero() && ttl > 0 {
		return cli.Exit("specify either expires or ttl not both", 1)
	}

	req := &api.LongURL{}
	if expiresAt != nil && !expiresAt.IsZero() {
		req.Expires = expiresAt.Format(time.RFC3339)
	}

	if ttl > 0 {
		req.Expires = time.Now().Add(ttl).Format(time.RFC3339)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out := make([]*api.ShortURL, 0, c.NArg())
	for i := 0; i < c.NArg(); i++ {
		req.URL = c.Args().Get(i)

		var rep *api.ShortURL
		if rep, err = svc.ShortenURL(ctx, req); err != nil {
			return cli.Exit(err, 1)
		}
		out = append(out, rep)
	}

	if len(out) == 1 {
		return display(out[0])
	}
	return display(out)
}

func info(c *cli.Context) (err error) {
	if c.NArg() == 0 {
		return cli.Exit("specify at least one short url ID to get info for", 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out := make([]*api.ShortURL, 0, c.NArg())
	for i := 0; i < c.NArg(); i++ {
		sid := c.Args().Get(i)
		if strings.HasPrefix(sid, "http") {
			if u, err := url.Parse(sid); err == nil {
				sid = strings.TrimPrefix(u.Path, "/")
			}
		}

		var rep *api.ShortURL
		if rep, err = svc.ShortURLInfo(ctx, sid); err != nil {
			return cli.Exit(err, 1)
		}

		out = append(out, rep)
	}

	if len(out) == 1 {
		return display(out[0])
	}
	return display(out)
}

func delete(c *cli.Context) (err error) {
	if c.NArg() == 0 {
		return cli.Exit("specify at least one short url ID to get info for", 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for i := 0; i < c.NArg(); i++ {
		sid := c.Args().Get(i)
		if strings.HasPrefix(sid, "http") {
			if u, err := url.Parse(sid); err == nil {
				sid = strings.TrimPrefix(u.Path, "/")
			}
		}

		if err = svc.DeleteShortURL(ctx, sid); err != nil {
			return cli.Exit(err, 1)
		}
		fmt.Printf("short url %s has been deleted\n", sid)
	}
	return nil
}

func status(c *cli.Context) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var status *api.StatusReply
	if status, err = svc.Status(ctx); err != nil {
		return cli.Exit(err, 1)
	}

	return display(status)
}

//===========================================================================
// Helper Commands
//===========================================================================

func configure(c *cli.Context) (err error) {
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func makeClient(c *cli.Context) (err error) {
	if svc, err = client.New(c.String("endpoint")); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func display(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}
