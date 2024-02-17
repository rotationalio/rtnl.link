package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/joho/godotenv"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/api/v1"
	"github.com/rotationalio/rtnl.link/pkg/client"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rotationalio/rtnl.link/pkg/keygen"
	"github.com/rotationalio/rtnl.link/pkg/passwd"
	"github.com/rotationalio/rtnl.link/pkg/rtnl"
	"github.com/rotationalio/rtnl.link/pkg/storage"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/urfave/cli/v2"
)

var (
	conf  config.Config
	svc   api.Service
	store *badger.DB

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
		&cli.StringFlag{
			Name:    "api-key",
			Aliases: []string{"a"},
			Usage:   "the api key to access the shortner service api",
			EnvVars: []string{"RTNL_API_KEY"},
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
			Name:     "register",
			Category: "admin",
			Usage:    "generate apikeys in maintenance mode",
			Action:   register,
			Before:   configure,
			Flags:    []cli.Flag{},
		},
		{
			Name:     "anonymize",
			Category: "admin",
			Usage:    "anonymize API keys (e.g. for creating a test fixture)",
			Action:   anonymize,
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
			Name:     "list",
			Category: "client",
			Usage:    "get list of short urls stored on the server",
			Action:   listLinks,
			Before:   makeClient,
			Flags:    []cli.Flag{},
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
			Name:      "updates",
			Category:  "client",
			Usage:     "subscribe to updates from the rtnl server",
			ArgsUsage: "[urlID]",
			Action:    updates,
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
		{
			Name:     "db:keys",
			Category: "debug",
			Usage:    "print out all of the keys in the local database",
			Action:   dbKeys,
			Before:   openStore,
			After:    closeStore,
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
// Admin Commands
//===========================================================================

func register(c *cli.Context) (err error) {
	if !conf.Maintenance {
		return cli.Exit("server must be in maintenance mode", 1)
	}

	// Open the database
	var store storage.Storage
	if store, err = storage.Open(conf.Storage); err != nil {
		return cli.Exit(err, 1)
	}
	defer store.Close()

	// Generate API key pair
	apikey := &models.APIKey{
		ClientID: keygen.KeyID(),
	}

	secret := keygen.Secret()
	if apikey.DerivedKey, err = passwd.CreateDerivedKey(secret); err != nil {
		return cli.Exit(err, 1)
	}

	if err = store.Register(apikey); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Println(apikey.ClientID + "-" + secret)
	return nil
}

func anonymize(c *cli.Context) (err error) {
	if !conf.Maintenance {
		return cli.Exit("server must be in maintenance mode", 1)
	}

	// Open the database
	var store storage.Storage
	if store, err = storage.Open(conf.Storage); err != nil {
		return cli.Exit(err, 1)
	}
	defer store.Close()

	var db *badger.DB
	if s, ok := store.(*storage.Store); ok {
		db = s.DB()
	} else {
		return cli.Exit("could not fetch db from storage", 1)
	}

	counts := make(map[string]int)
	err = db.Update(func(txn *badger.Txn) error {
		// TODO: provide more semantic method of listing API keys
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()

			counts[fmt.Sprintf("keysize %d", len(key))]++

			if len(key) == 16 {
				apikey := &models.APIKey{}
				if err = item.Value(apikey.UnmarshalValue); err != nil {
					fmt.Println(err)
					counts["errors"]++
					continue
				}

				apikey.ClientID = keygen.KeyID()
				if apikey.DerivedKey, err = passwd.CreateDerivedKey(keygen.Secret()); err != nil {
					fmt.Println(err)
					counts["errors"]++
					continue
				}

				var data []byte
				if data, err = apikey.MarshalValue(); err != nil {
					fmt.Println(err)
					counts["errors"]++
					continue
				}

				if err = txn.Set(key, data); err != nil {
					return err
				}
				counts["anonymized"]++
			}
		}

		return nil
	})

	if err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Println("Count\tMetric")
	for item, count := range counts {
		fmt.Printf("%d\t%s\n", count, item)
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

func listLinks(c *cli.Context) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var out *api.ShortURLList
	if out, err = svc.ShortURLList(ctx, nil); err != nil {
		return cli.Exit(err, 1)
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

func updates(c *cli.Context) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var linkID string
	if c.NArg() > 0 {
		if c.NArg() > 1 {
			return cli.Exit("only one link url can be specified", 1)
		}
		linkID = c.Args().First()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	updates, err := svc.Updates(ctx, linkID)
	if err != nil {
		return cli.Exit(err, 1)
	}

	for {
		select {
		case update := <-updates:
			data, err := json.Marshal(update)
			if err != nil {
				return cli.Exit(err, 1)
			}
			fmt.Println(string(data))
		case <-interrupt:
			return nil
		}
	}
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
// Debug Commands
//===========================================================================

func dbKeys(c *cli.Context) error {
	err := store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			fmt.Printf("key=%s\n", k)
		}
		return nil
	})

	if err != nil {
		return cli.Exit(err, 1)
	}
	return nil
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
	if svc, err = client.New(c.String("endpoint"), c.String("api-key")); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func openStore(c *cli.Context) (err error) {
	if err = configure(c); err != nil {
		return err
	}

	opts := badger.DefaultOptions(conf.Storage.DataPath)
	opts.ReadOnly = conf.Storage.ReadOnly
	opts.Logger = nil

	if store, err = badger.Open(opts); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}

func closeStore(c *cli.Context) (err error) {
	if err = store.Close(); err != nil {
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
