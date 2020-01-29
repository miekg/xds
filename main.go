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
						Name:      "clusters",
						Aliases:   []string{"cluster"},
						Usage:     "list cluster [CLUSTER]",
						ArgsUsage: "[CLUSTER]",
						Action:    listClusters,
					},
					{
						Name:      "endpoints",
						Aliases:   []string{"endpoint"},
						Usage:     "list endpoints [CLUSTER]",
						ArgsUsage: "[CLUSTER]",
						Action:    listEndpoints,
					},
				},
			},
			{
				Name:        "drain",
				Description: "bla",
				Category:    "drain",
				Usage:       "CLUSTER [ENDPOINT]",
				ArgsUsage:   "CLUSTER [ENDPOINT]",
				Action:      drain,
				Subcommands: []*cli.Command{
					{
						Name:      "cluster",
						Usage:     "drain cluster CLUSTER [ENDPOINT]",
						ArgsUsage: "CLUSTER [ENDPOINT]",
						Action:    drainCluster,
					},
				},
			},
			{
				Name:        "undrain",
				Description: "bla",
				Category:    "drain",
				Usage:       "CLUSTER [ENDPOINT]",
				ArgsUsage:   "CLUSTER [ENDPOINT]",
				Action:      undrain,
				Subcommands: []*cli.Command{
					{
						Name:      "cluster",
						Usage:     "undrain cluster CLUSTER [ENDPOINT]",
						ArgsUsage: "CLUSTER [ENDPOINT]",
						Action:    undrainCluster,
					},
				},
			},
			{
				Name: "health",
				Description: "Health sets the health for endpoints in a cluster. If no endpoint is given all endpoints will be set.\n" +
					"   The mandatory argument HEALT_STATUS can be: 'UNKNOWN', 'HEALTHY', 'UNHEALTHY', 'DRAINING', 'TIMEOUT' or 'DEGRADED'.",
				ArgsUsage: "CLUSTER [ENDPOINT] HEALTH_STATUS",
				Usage:     "set health status for endpoints or entire clusters",
				Action:    health,
			},
			{
				Name:        "add",
				Description: "Add adds clusters and endpoints. A new endpoint will have its health set to UNKNOWN.",
				Category:    "admin",
				ArgsUsage:   "CLUSTER [ENDPOINT]",
				Usage:       "add a cluster or add a cluster and endpoint",
				Action:      add,
			},
			{
				Name:        "remove",
				Description: "Remove removes clusters and endpoints. If no endpoint is given the entire cluster is removed.",
				Category:    "admin",
				Aliases:     []string{"rm"},
				Usage:       "remove  a cluster or remove a cluster and endpoint",
				ArgsUsage:   "CLUSTER [ENDPOINT]",
				Action:      remove,
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
