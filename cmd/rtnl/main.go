package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/rotationalio/rtnl.link/pkg"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rotationalio/rtnl.link/pkg/rtnl"
	"github.com/urfave/cli/v2"
)

var conf config.Config

func main() {
	// If a dotenv file exists load it for configuration
	godotenv.Load()

	// Create a multi-command CLI application
	app := cli.NewApp()
	app.Name = "rtnl"
	app.Version = pkg.Version()
	app.Usage = "utilities and administrative commands rtnl.link"
	app.Flags = []cli.Flag{}
	app.Before = configure
	app.Commands = []*cli.Command{
		{
			Name:     "serve",
			Usage:    "run the rtnl server",
			Action:   serve,
			Category: "server",
			Flags:    []cli.Flag{},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
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

//===========================================================================
// Helper Commands
//===========================================================================

func configure(c *cli.Context) (err error) {
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}
