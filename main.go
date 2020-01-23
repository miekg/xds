package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const padding = 3

func main() {
	app := &cli.App{
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "s", Usage: "server `ADDRESS` to connect to", Required: true},
			&cli.StringFlag{Name: "n", Usage: "node `ID` to use", Value: "test-id"},
			&cli.BoolFlag{Name: "k", Usage: "disable TLS"},
			&cli.BoolFlag{Name: "H", Usage: "print header in ouput", Value: true},
			&cli.BoolFlag{Name: "N", Usage: "dry run", Value: false},
		},
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "[CLUSTERS [ENDPOINT]]",
				Action:  listClusters,
				Subcommands: []*cli.Command{
					{
						Name:    "clusters",
						Aliases: []string{"cluster"},
						Usage:   "list cluster [CLUSTER]",
						Action:  listClusters,
					},
					{
						Name:    "endpoints",
						Aliases: []string{"endpoint"},
						Usage:   "list endpoints [CLUSTER]",
						Action:  listEndpoints,
					},
				},
			},
			{
				Name:     "drain",
				Category: "drain",
				Usage:    "REGION ZONE SUBZONE or a CLUSTER [ENDPOINT]",
				Action:   drain,
				Subcommands: []*cli.Command{
					{
						Name:   "cluster",
						Usage:  "drain cluster CLUSTER [ENDPOINT]",
						Action: drainCluster,
					},
				},
			},
			{
				Name:     "undrain",
				Category: "drain",
				Usage:    "REGION ZONE SUBZONE or a CLUSTER [ENDPOINT]",
				Action:   undrain,
				Subcommands: []*cli.Command{
					{
						Name:   "cluster",
						Usage:  "undrain cluster CLUSTER [ENDPOINT]",
						Action: undrainCluster,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		errorf(err)
	}
}

func errorf(err error) {
	fmt.Printf("%s\n", err)
	os.Exit(1)
}

var (
	ErrArg      = func(s []string) error { return fmt.Errorf("parse error with arguments: %v", s) }
	ErrNotFound = func(s []string, typ string) error { return fmt.Errorf("no such %s: %q", typ, s) }
)
